//go:build darwin && !ios

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"

	"gogui/internal/cocoa"
	"gogui/internal/gpu"
)

// macOS 窗口后端：AppKit（purego objc_msgSend）+ NSOpenGLContext + OpenGL 3.2 Core。
//
// 与 backend_windows.go / backend_x11.go 提供完全相同的 Backend 方法集，
// app.go 一行不用改。零 cgo。
//
// 生命周期：AppKit 的事件循环由 cocoa.PumpEvents 驱动（非阻塞拉取），
// 因此 pollMessages 每次调用 = 「泵一轮事件 + 小睡」，与 X11 后端同构。

// platformWin macOS 端窗口。
type platformWin struct {
	w       *cocoa.Window
	handler func(Event)

	corner  float32
	accepts bool
	visible bool
}

// newBackend 创建 NSWindow 并建立 OpenGL 上下文。
func newBackend(opts Options) (*platformWin, error) {
	w, err := cocoa.NewWindow(opts.Title, opts.Width, opts.Height)
	if err != nil {
		return nil, err
	}
	// 把 GL 函数表接到渲染器（CGLGetProcAddress）
	gpu.SetGLProcLoader(w.Proc)
	if n := gpu.InitGLProcs(); n < 20 {
		w.Destroy()
		return nil, fmt.Errorf("gl: 只解析到 %d 个 GL 函数，上下文不可用（该机器可能不支持 OpenGL 3.2 Core）", n)
	}
	p := &platformWin{w: w, accepts: true, visible: true}
	w.SetHandler(p.onCocoaEvent)
	return p, nil
}

// pollMessages 泵一轮 AppKit 事件；返回 false 表示所有窗口都已关闭。
func pollMessages() bool {
	if cocoa.WindowCount() == 0 {
		return false
	}
	cocoa.PumpEvents()
	return cocoa.WindowCount() > 0
}

// onCocoaEvent 把原生事件翻译成 ui.Event。
func (p *platformWin) onCocoaEvent(ev cocoa.Event) {
	if p.handler == nil {
		return
	}
	e := Event{X: ev.X, Y: ev.Y, DY: ev.DY, Button: ev.Button, Scale: p.w.Scale()}
	switch ev.Kind {
	case cocoa.EvMouseMove:
		e.Type = EvMouseMove
	case cocoa.EvMouseDown:
		e.Type = EvMouseDown
	case cocoa.EvMouseUp:
		e.Type = EvMouseUp
	case cocoa.EvMouseWheel:
		e.Type = EvMouseWheel
	case cocoa.EvMouseLeave:
		e.Type = EvMouseLeave
	case cocoa.EvKeyDown:
		e.Type = EvKeyDown
		e.Key = ev.Key
	case cocoa.EvKeyUp:
		e.Type = EvKeyUp
		e.Key = ev.Key
	case cocoa.EvChar:
		e.Type = EvChar
		e.Char = ev.Char
	case cocoa.EvResize:
		e.Type = EvResize
		e.Width, e.Height = ev.Width, ev.Height
	case cocoa.EvClose:
		e.Type = EvClose
	case cocoa.EvFocusGained:
		e.Type = EvFocusGained
	case cocoa.EvFocusLost:
		e.Type = EvFocusLost
	default:
		return
	}
	p.handler(e)
}

// ---- Backend 方法集 ----

func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }
func (p *platformWin) MakeCurrent()                   { p.w.MakeCurrent() }
func (p *platformWin) Swap()                          { p.w.Swap() }
func (p *platformWin) PixelSize() (int32, int32)      { return p.w.PixelSize() }
func (p *platformWin) Scale() float32                 { return p.w.Scale() }

func (p *platformWin) IsMinimized() bool       { return p.w.IsMinimized() }
func (p *platformWin) SetTitle(s string)       { p.w.SetTitle(s) }
func (p *platformWin) SetIcon(ico []byte) bool { return false } // macOS 应用图标在 .app 包内，运行期不可改
func (p *platformWin) SetVisible(v bool) {
	p.visible = v
	p.w.Show(v)
}
func (p *platformWin) Minimize() { p.w.Minimize() }
func (p *platformWin) Restore()  { p.w.Show(true) }
func (p *platformWin) Center()   { p.w.Center() }

// BeginDragMove 自绘标题栏拖动：macOS 由系统标题栏负责，这里空实现。
func (p *platformWin) BeginDragMove() {}

func (p *platformWin) ToggleMaximize()           { p.w.SetFullscreen(!p.w.IsFullscreen()) }
func (p *platformWin) IsMaximized() bool         { return p.w.IsFullscreen() }
func (p *platformWin) SetFullscreen(v bool)      { p.w.SetFullscreen(v) }
func (p *platformWin) IsFullscreen() bool        { return p.w.IsFullscreen() }
func (p *platformWin) SetAlwaysOnTop(on bool)    { p.w.SetAlwaysOnTop(on) }
func (p *platformWin) AlwaysOnTop() bool         { return p.w.AlwaysOnTop() }
func (p *platformWin) SetCornerRadius(v float32) { p.corner = v }
func (p *platformWin) CornerRadius() float32     { return p.corner }

func (p *platformWin) SetAcceptDrops(on bool) { p.accepts = on }
func (p *platformWin) AcceptDrops() bool      { return p.accepts }

// SetImePosition 输入法候选框（AppKit 自动跟随插入点，空实现）。
func (p *platformWin) SetImePosition(x, y int32) { p.w.SetImePosition(x, y) }

// RegisterHotKey 全局热键需要 CGEventTap，一期返回 0（不支持）。
func (p *platformWin) RegisterHotKey(mods, vk uint32, fn func()) int32 { return 0 }

func (p *platformWin) UnregisterHotkey(id int32) {}

func (p *platformWin) AttachCompose(pw, ph int32) {}
func (p *platformWin) HasCompose() bool           { return false }
func (p *platformWin) ResizeCompose(pw, ph int32) {}
func (p *platformWin) ComposeBegin() uint32       { return 0 }
func (p *platformWin) ComposeEnd()                {}

// NativeHandle 原生句柄（NSView*）。
func (p *platformWin) NativeHandle() uintptr { return p.w.NativeHandle() }

func (p *platformWin) Close() { p.w.Close() }

func (p *platformWin) Destroy() { p.w.Destroy() }
