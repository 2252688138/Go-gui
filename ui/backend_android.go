//go:build android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"time"

	"gogui/internal/android"
	"gogui/internal/gpu"
)

// Android 窗口后端：NativeActivity + EGL + GLES 3，触摸事件合成为鼠标事件。
//
// 与其它端提供同一份 Backend 接口，因此 app.go 的布局/绘制/命中/滚动逻辑完全复用：
// 触摸按下 → EvMouseDown、移动 → EvMouseMove、抬起 → EvMouseUp，同时额外发
// EvTouch*（业务需要区分手指/鼠标时看 Pointer 字段）。
//
// 线程模型：Go 的 main 在独立线程跑 ui.Run；窗口由 Java 侧（NativeActivity）
// 在自己的线程创建后通过 android.SetWindow 交过来，因此这里要「等窗口就绪」。

// platformWin Android 端窗口。
type platformWin struct {
	handler func(Event)

	pw, ph int32
	scale  float32
	title  string
	closed bool
	corner float32

	// 触摸合成的鼠标状态
	prevX, prevY float32
	touching     bool
}

// androidWin 当前窗口（只有一个；pollMessages 靠它分发事件）。
var androidWin *platformWin

// newBackend 等待原生窗口就绪并建立 GLES3 上下文。
func newBackend(opts Options) (*platformWin, error) {
	// 等到 Java 侧把 ANativeWindow 交过来（最多 5 秒）
	deadline := time.Now().Add(5 * time.Second)
	for {
		w, h := android.WindowSize()
		if w > 0 && h > 0 && android.Ready() {
			break
		}
		if time.Now().After(deadline) {
			return nil, errors.New("android: 等待原生窗口超时（ANativeWindow 未就绪）")
		}
		time.Sleep(16 * time.Millisecond)
	}

	gpu.SetGLProcLoader(android.GetProcAddress)
	if n := gpu.InitGLProcs(); n < 20 {
		return nil, errors.New("android: GLES3 函数解析失败（设备不支持 OpenGL ES 3？）")
	}
	w, h := android.WindowSize()
	p := &platformWin{
		title: opts.Title,
		pw:    w,
		ph:    h,
		scale: 1, // Android 用物理像素布局；如需按 dp 缩放可用 ui.SetUIScale
	}
	androidWin = p
	return p, nil
}

// pollMessages 取一轮输入事件并分发（runLoop 每帧调用一次）；
// 返回 false 表示 GL 上下文已不可用（应用退出 / 窗口销毁）。
func pollMessages() bool {
	if !android.Ready() {
		return false
	}
	if androidWin != nil {
		androidPump(androidWin)
	}
	return true
}

// pumpEvents 由 runLoop 之外的帧驱动调用：把 Android 事件翻译成 ui 事件。
func androidPump(p *platformWin) {
	android.PollEvents(func(ev android.Event) {
		if p.handler == nil {
			return
		}
		switch ev.Kind {
		case android.EvTouchDown:
			p.touching = true
			p.prevX, p.prevY = ev.X, ev.Y
			p.handler(Event{Type: EvTouchDown, X: ev.X, Y: ev.Y, Pointer: 1})
			p.handler(Event{Type: EvMouseDown, X: ev.X, Y: ev.Y, Pointer: 1})
		case android.EvTouchMove:
			p.prevX, p.prevY = ev.X, ev.Y
			p.handler(Event{Type: EvTouchMove, X: ev.X, Y: ev.Y, Pointer: 1})
			p.handler(Event{Type: EvMouseMove, X: ev.X, Y: ev.Y, Pointer: 1})
		case android.EvTouchUp:
			p.touching = false
			p.handler(Event{Type: EvTouchUp, X: ev.X, Y: ev.Y, Pointer: 1})
			p.handler(Event{Type: EvMouseUp, X: ev.X, Y: ev.Y, Pointer: 1})
		case android.EvKey:
			p.handler(Event{Type: EvKeyDown, Key: ev.Key})
		case android.EvBack:
			p.handler(Event{Type: EvBack})
		case android.EvResize:
			if int32(ev.Width) != p.pw || int32(ev.Height) != p.ph {
				p.pw, p.ph = int32(ev.Width), int32(ev.Height)
				p.handler(Event{Type: EvResize, Width: ev.Width, Height: ev.Height, Scale: p.scale})
			}
		}
	})
}

// ---- Backend 方法集 ----

func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }
func (p *platformWin) MakeCurrent()                   { android.MakeCurrent() }
func (p *platformWin) Swap()                          { android.Swap() }
func (p *platformWin) PixelSize() (int32, int32)      { return p.pw, p.ph }
func (p *platformWin) Scale() float32                 { return p.scale }

func (p *platformWin) IsMinimized() bool       { return !android.Ready() }
func (p *platformWin) SetTitle(s string)       { p.title = s } // 标题栏由系统管理
func (p *platformWin) SetIcon(ico []byte) bool { return false }
func (p *platformWin) SetVisible(v bool)       {}
func (p *platformWin) Minimize()               {}
func (p *platformWin) Restore()                {}
func (p *platformWin) Center()                 {}
func (p *platformWin) BeginDragMove()          {}
func (p *platformWin) ToggleMaximize()         {}

// IsMaximized 移动端没有窗口最大化概念（恒 true）。
func (p *platformWin) IsMaximized() bool         { return true }
func (p *platformWin) SetFullscreen(v bool)      {} // 由 manifest 的 theme 决定
func (p *platformWin) IsFullscreen() bool        { return true }
func (p *platformWin) SetAlwaysOnTop(on bool)    {}
func (p *platformWin) AlwaysOnTop() bool         { return false }
func (p *platformWin) SetCornerRadius(v float32) { p.corner = v }
func (p *platformWin) CornerRadius() float32     { return p.corner }

func (p *platformWin) SetAcceptDrops(on bool) {}
func (p *platformWin) AcceptDrops() bool      { return false }

// SetImePosition 软键盘候选框位置（由系统输入法决定）。
func (p *platformWin) SetImePosition(x, y int32) {}

func (p *platformWin) RegisterHotKey(mods, vk uint32, fn func()) int32 { return 0 }
func (p *platformWin) UnregisterHotkey(id int32)                       {}

func (p *platformWin) AttachCompose(pw, ph int32) {}
func (p *platformWin) HasCompose() bool           { return false }
func (p *platformWin) ResizeCompose(pw, ph int32) {}
func (p *platformWin) ComposeBegin() uint32       { return 0 }
func (p *platformWin) ComposeEnd()                {}

// NativeHandle 原生窗口指针（ANativeWindow*）。
func (p *platformWin) NativeHandle() uintptr { return android.CurrentActivity() }

func (p *platformWin) Close() { p.closed = true }

func (p *platformWin) Destroy() {
	p.closed = true
	android.Destroy()
}
