//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"image"
	"time"

	"gogui/internal/win32"
)

// 桌面增强能力（Windows 真实现）：多显示器、屏幕捕获与吸管、剪贴板监听、
// 单实例锁 + 二次启动参数、GDI 打印。其他平台见 system_extra_other.go。

// Monitor 显示器信息。
type Monitor = win32.MonitorInfo

// Monitors 枚举显示器（主屏在前）。
func Monitors() []Monitor { return win32.Monitors() }

// MonitorAt 返回包含指定屏幕坐标的显示器。
func MonitorAt(x, y int) (Monitor, bool) {
	for _, m := range Monitors() {
		if x >= m.X && x < m.X+m.W && y >= m.Y && y < m.Y+m.H {
			return m, true
		}
	}
	return Monitor{}, false
}

// VirtualScreen 所有显示器并集（x, y, w, h）。
func VirtualScreen() (int, int, int, int) { return win32.VirtualScreen() }

// CaptureScreen 截取屏幕区域（物理像素，支持负坐标 / 多屏）。cursor 预留，
// 当前实现不合成鼠标指针。
func CaptureScreen(x, y, w, h int, cursor bool) (*image.NRGBA, error) {
	return win32.CaptureScreen(x, y, w, h, cursor)
}

// CaptureFullScreen 截取主屏。
func CaptureFullScreen() (*image.NRGBA, error) {
	ms := Monitors()
	if len(ms) == 0 {
		return nil, fmt.Errorf("没有可用显示器")
	}
	m := ms[0]
	return win32.CaptureScreen(m.X, m.Y, m.W, m.H, false)
}

// ClipboardInfo 剪贴板快照。
type ClipboardInfo = win32.ClipboardInfo

// Clipboard 读取剪贴板摘要。
func Clipboard() ClipboardInfo { return win32.ClipboardSnapshot() }

// ClipboardImage 读取剪贴板位图。
func ClipboardImage() (*image.NRGBA, error) { return win32.ClipboardImage() }

// WatchClipboard 监听剪贴板变化（500ms 轮询序号，变化时回调）。
// 返回停止函数；回调在 UI 线程执行。
func WatchClipboard(fn func(ClipboardInfo)) (stop func()) {
	if fn == nil {
		return func() {}
	}
	quit := make(chan struct{})
	go func() {
		defer recoverReport("clipboard-watch")
		last := win32.ClipboardSequence()
		tk := time.NewTicker(500 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-quit:
				return
			case <-tk.C:
				seq := win32.ClipboardSequence()
				if seq == last {
					continue
				}
				last = seq
				info := win32.ClipboardSnapshot()
				postToUI(func() {
					fn(info)
					RequestRedraw()
				})
			}
		}
	}()
	return func() { close(quit) }
}

// ---- 单实例 ----

var secondInstanceKey string

// SingleInstance 申请单实例锁。返回 false 表示已有实例在运行，此时调用
// NotifyFirstInstance 把参数传给首个实例并退出即可。
func SingleInstance(key string) bool {
	secondInstanceKey = key
	return win32.SingleInstance(key)
}

// NotifyFirstInstance 把本次启动参数发给已运行实例。
func NotifyFirstInstance(key string, args []string) {
	win32.SecondInstancePingArgs(key, args)
}

// WatchSecondInstance 轮询二次启动参数（首个实例调用），args 非空时回调。
func WatchSecondInstance(key string, fn func(args []string)) (stop func()) {
	if fn == nil {
		return func() {}
	}
	quit := make(chan struct{})
	go func() {
		defer recoverReport("second-instance-watch")
		tk := time.NewTicker(700 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-quit:
				return
			case <-tk.C:
				if args := win32.PollSecondInstanceArgs(key); len(args) > 0 {
					a := args
					postToUI(func() {
						fn(a)
						RequestRedraw()
					})
				}
			}
		}
	}()
	return func() { close(quit) }
}

// ---- 打印 ----

func printDocument(hwnd uintptr, title string, doc Printable) (bool, error) {
	pages := doc.Pages()
	if pages <= 0 {
		pages = 1
	}
	return win32.PrintDialog(hwnd, title, pages, func(dc uintptr, page int, info win32.PrintInfo) bool {
		pw, ph := win32.PagePixels(info)
		scale := float32(pw) / PageW96
		c := &PrintCanvas{W: PageW96, H: PageH96, scale: scale, dc: dc, live: true}
		_ = ph
		doc.DrawPage(c, page)
		return true
	})
}

func printText(dc uintptr, x, y, size float32, s, color string) {
	win32.PrintText(dc, x, y, size, s, color)
}

func printRect(dc uintptr, x, y, w, h, stroke float32, strokeColor, fill string) {
	win32.PrintRect(dc, x, y, w, h, stroke, strokeColor, fill)
}

func printLine(dc uintptr, x1, y1, x2, y2, width float32, color string) {
	win32.PrintLine(dc, x1, y1, x2, y2, width, color)
}

// postToUI 把回调切到 UI 线程（无窗口时直接执行）。
func postToUI(fn func()) {
	if a := currentApp.Load(); a != nil {
		a.Post(fn)
		return
	}
	fn()
}

func init() {
	// 吸管取色一律用屏幕坐标（物理像素）：以前拿窗口内坐标去取，窗口不在 (0,0)
	// 或指针移出窗口后取到的是别处颜色，看起来就像「吸管没反应」。
	ScreenPickColor = func() (string, bool) {
		x, y, ok := win32.CursorScreenPos()
		if !ok {
			return "", false
		}
		return ScreenPixelAt(x, y)
	}
	ScreenPixelAt = func(x, y int) (string, bool) {
		c, ok := win32.ScreenPixelAt(x, y)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B), true
	}
	CursorAt = win32.CursorScreenPos
	LeftButtonDown = win32.LeftButtonDown
	RightButtonDown = win32.RightButtonDown
	KeyDown = win32.KeyDown
}
