//go:build !windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"image"
)

// 桌面增强能力的非 Windows 降级实现：符号齐全、返回零值 / 不支持。
// 业务代码应先用 HasCap 判断（CapScreenCapture / CapPrint / CapMultiMonitor …）。

// Monitor 显示器信息（非 Windows 端字段保留，值由各端自行填充或为空）。
type Monitor struct {
	Index   int
	Name    string
	X, Y    int
	W, H    int
	WorkX   int
	WorkY   int
	WorkW   int
	WorkH   int
	Scale   float32
	Primary bool
}

// Monitors 非 Windows 端返回空（X11 可后续接 XRandR）。
func Monitors() []Monitor { return nil }

// MonitorAt 非 Windows 端恒不支持。
func MonitorAt(x, y int) (Monitor, bool) { return Monitor{}, false }

// VirtualScreen 非 Windows 端返回零值。
func VirtualScreen() (int, int, int, int) { return 0, 0, 0, 0 }

// CaptureScreen 非 Windows 端暂不支持。
func CaptureScreen(x, y, w, h int, cursor bool) (*image.NRGBA, error) {
	return nil, errors.New("ui: 当前平台暂不支持屏幕捕获")
}

// CaptureFullScreen 非 Windows 端暂不支持。
func CaptureFullScreen() (*image.NRGBA, error) {
	return nil, errors.New("ui: 当前平台暂不支持屏幕捕获")
}

// ClipboardInfo 剪贴板快照。
type ClipboardInfo struct {
	Seq      uint32
	Text     string
	HasText  bool
	HasImage bool
	HasFiles bool
}

// Clipboard 读取剪贴板摘要（非 Windows 仅文本）。
func Clipboard() ClipboardInfo {
	t := ClipboardText()
	return ClipboardInfo{Text: t, HasText: t != ""}
}

// ClipboardImage 非 Windows 端不支持。
func ClipboardImage() (*image.NRGBA, error) {
	return nil, errors.New("ui: 当前平台暂不支持读取剪贴板图片")
}

// WatchClipboard 非 Windows 端用轮询文本变化近似实现。
func WatchClipboard(fn func(ClipboardInfo)) (stop func()) {
	return watchClipboardPolling(fn)
}

// SingleInstance 非 Windows 端不做锁，恒视为首个实例。
func SingleInstance(key string) bool { return true }

// NotifyFirstInstance 非 Windows 端空实现。
func NotifyFirstInstance(key string, args []string) {}

// WatchSecondInstance 非 Windows 端空实现。
func WatchSecondInstance(key string, fn func(args []string)) (stop func()) {
	return func() {}
}

// ---- 打印 ----

func printDocument(hwnd uintptr, title string, doc Printable) (bool, error) {
	return false, errors.New("ui: 当前平台暂不支持打印")
}

func printText(dc uintptr, x, y, size float32, s, color string)         {}
func printRect(dc uintptr, x, y, w, h, stroke float32, sc, fill string) {}
func printLine(dc uintptr, x1, y1, x2, y2, width float32, color string) {}

// postToUI 非 Windows 端直接执行。
func postToUI(fn func()) { fn() }
