//go:build ios && cgo

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"

	"gogui/internal/gpu"
	"gogui/internal/uikit"
)

// iOS 窗口后端：UIKit + EAGL + GLES 3（cgo）。
//
// 与 Android 后端同构：framebuffer 0 直接指向屏幕，触摸合成鼠标事件。
//
// ⚠️ 验证状态：需要 macOS + Xcode（iOS 强制 cgo + 外部链接），
// 未在 Windows 开发机上编译验证；结构与 backend_android.go 一一对应。
//
// 构建（macOS）：
//
//	GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
//	  go build -buildmode=c-archive -o libgogui.a ./mobile/ios

// platformWin iOS 端窗口。
type platformWin struct {
	handler func(Event)

	pw, ph int32
	scale  float32
	title  string
	closed bool
	corner float32
}

// iosWin 当前窗口（只有一个）。
var iosWin *platformWin

// newBackend 初始化 UIWindow 与 EAGL 上下文。
func newBackend(opts Options) (*platformWin, error) {
	if err := uikit.Init(); err != nil {
		return nil, err
	}
	gpu.SetGLProcLoader(uikit.Proc)
	if n := gpu.InitGLProcs(); n < 20 {
		return nil, errors.New("ios: GLES3 函数解析失败（设备不支持 OpenGL ES 3？）")
	}
	w, h := uikit.Size()
	p := &platformWin{
		title: opts.Title,
		pw:    w,
		ph:    h,
		scale: uikit.Scale(),
	}
	iosWin = p
	uikit.SetHandler(p.onUIKitEvent)
	uikit.SetTitle(opts.Title)
	return p, nil
}

// pollMessages 泵一轮 UIKit 事件。
func pollMessages() bool {
	if !uikit.Ready() {
		return false
	}
	uikit.Poll()
	w, h := uikit.Size()
	if iosWin != nil {
		if w != iosWin.pw || h != iosWin.ph {
			iosWin.pw, iosWin.ph = w, h
			iosWin.scale = uikit.Scale()
			if iosWin.handler != nil {
				iosWin.handler(Event{Type: EvResize, Width: float32(w), Height: float32(h), Scale: iosWin.scale})
			}
		}
	}
	return true
}

// onUIKitEvent 把原生事件翻译成 ui.Event（触摸同时合成鼠标事件）。
func (p *platformWin) onUIKitEvent(ev uikit.Event) {
	if p.handler == nil {
		return
	}
	switch ev.Kind {
	case uikit.EvTouchDown:
		p.handler(Event{Type: EvTouchDown, X: ev.X, Y: ev.Y, Pointer: 1})
		p.handler(Event{Type: EvMouseDown, X: ev.X, Y: ev.Y, Pointer: 1})
	case uikit.EvTouchMove:
		p.handler(Event{Type: EvTouchMove, X: ev.X, Y: ev.Y, Pointer: 1})
		p.handler(Event{Type: EvMouseMove, X: ev.X, Y: ev.Y, Pointer: 1})
	case uikit.EvTouchUp:
		p.handler(Event{Type: EvTouchUp, X: ev.X, Y: ev.Y, Pointer: 1})
		p.handler(Event{Type: EvMouseUp, X: ev.X, Y: ev.Y, Pointer: 1})
	case uikit.EvKey:
		p.handler(Event{Type: EvKeyDown, Key: ev.Key})
	case uikit.EvChar:
		p.handler(Event{Type: EvChar, Char: ev.Char})
	case uikit.EvResize:
		p.handler(Event{Type: EvResize, Width: ev.Width, Height: ev.Height, Scale: ev.Scale})
	}
}

// ---- Backend 方法集 ----

func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }
func (p *platformWin) MakeCurrent()                   { uikit.MakeCurrent() }
func (p *platformWin) Swap()                          { uikit.Present() }
func (p *platformWin) PixelSize() (int32, int32)      { return p.pw, p.ph }
func (p *platformWin) Scale() float32                 { return p.scale }

func (p *platformWin) IsMinimized() bool       { return false }
func (p *platformWin) SetTitle(s string)       { p.title = s; uikit.SetTitle(s) }
func (p *platformWin) SetIcon(ico []byte) bool { return false }
func (p *platformWin) SetVisible(v bool)       {}
func (p *platformWin) Minimize()               {}
func (p *platformWin) Restore()                {}
func (p *platformWin) Center()                 {}
func (p *platformWin) BeginDragMove()          {}
func (p *platformWin) ToggleMaximize()         {}

func (p *platformWin) IsMaximized() bool         { return true }
func (p *platformWin) SetFullscreen(v bool)      {} // 移动端恒全屏
func (p *platformWin) IsFullscreen() bool        { return true }
func (p *platformWin) SetAlwaysOnTop(on bool)    {}
func (p *platformWin) AlwaysOnTop() bool         { return false }
func (p *platformWin) SetCornerRadius(v float32) { p.corner = v }
func (p *platformWin) CornerRadius() float32     { return p.corner }

func (p *platformWin) SetAcceptDrops(on bool) {}
func (p *platformWin) AcceptDrops() bool      { return false }

// SetImePosition 软键盘由系统摆放，空实现。
func (p *platformWin) SetImePosition(x, y int32) {}

func (p *platformWin) RegisterHotKey(mods, vk uint32, fn func()) int32 { return 0 }
func (p *platformWin) UnregisterHotkey(id int32)                       {}

func (p *platformWin) AttachCompose(pw, ph int32) {}
func (p *platformWin) HasCompose() bool           { return false }
func (p *platformWin) ResizeCompose(pw, ph int32) {}
func (p *platformWin) ComposeBegin() uint32       { return 0 }
func (p *platformWin) ComposeEnd()                {}

// NativeHandle 原生窗口句柄（UIWindow*）。
func (p *platformWin) NativeHandle() uintptr { return uikit.NativeHandle() }

func (p *platformWin) Close() { p.closed = true }

func (p *platformWin) Destroy() { p.closed = true }
