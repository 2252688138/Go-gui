//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package display

import "gogui/internal/win32"

// winWindow 把 Win32 窗口适配成 display.Window。
//
// 这一层刻意很薄：Win32 侧已有 DPI / 圆角 / IME / 拖入等完整实现，
// 这里只做类型与事件结构体的翻译，保证上层拿到的是平台无关契约。
type winWindow struct {
	w *win32.Window
}

// New 创建 Win32 窗口并初始化 WGL 上下文。
func New(cfg Config) (Window, error) {
	w, err := win32.NewWindow(win32.Config{
		Title:        cfg.Title,
		Width:        cfg.Width,
		Height:       cfg.Height,
		MinWidth:     cfg.MinWidth,
		MinHeight:    cfg.MinHeight,
		Resizable:    cfg.Resizable,
		Decorated:    cfg.Decorated,
		Transparent:  cfg.Transparent,
		CornerRadius: cfg.CornerRadius,
		NoDrops:      cfg.NoDrops,
	})
	if err != nil {
		return nil, err
	}
	return &winWindow{w: w}, nil
}

// Poll 处理一轮窗口消息；返回 false 表示所有窗口都已关闭。
func Poll() bool { return win32.PollMessages() }

// ---- Window 接口 ----

func (v *winWindow) NativeHandle() uintptr { return v.w.Hwnd }

func (v *winWindow) MakeCurrent()              { v.w.MakeCurrent() }
func (v *winWindow) Swap()                     { v.w.Swap() }
func (v *winWindow) PixelSize() (int32, int32) { return v.w.PixelSize() }
func (v *winWindow) Scale() float32            { return v.w.Scale() }

func (v *winWindow) IsMinimized() bool         { return v.w.IsMinimized() }
func (v *winWindow) SetTitle(s string)         { v.w.SetTitle(s) }
func (v *winWindow) SetIcon(ico []byte) bool   { return v.w.SetIcon(ico) }
func (v *winWindow) SetVisible(on bool)        { v.w.SetVisible(on) }
func (v *winWindow) Minimize()                 { v.w.Minimize() }
func (v *winWindow) Restore()                  { v.w.Restore() }
func (v *winWindow) Center()                   { v.w.Center() }
func (v *winWindow) BeginDragMove()            { v.w.BeginDragMove() }
func (v *winWindow) ToggleMaximize()           { v.w.ToggleMaximize() }
func (v *winWindow) IsMaximized() bool         { return v.w.IsMaximized() }
func (v *winWindow) SetFullscreen(on bool)     { v.w.SetFullscreen(on) }
func (v *winWindow) IsFullscreen() bool        { return v.w.IsFullscreen() }
func (v *winWindow) SetAlwaysOnTop(on bool)    { v.w.SetAlwaysOnTop(on) }
func (v *winWindow) AlwaysOnTop() bool         { return v.w.AlwaysOnTop() }
func (v *winWindow) SetCornerRadius(r float32) { v.w.SetCornerRadius(r) }
func (v *winWindow) CornerRadius() float32     { return v.w.CornerRadius() }

func (v *winWindow) SetEventHandler(fn func(Event)) {
	v.w.SetEventHandler(func(ev win32.Event) { fn(fromWin32(ev)) })
}
func (v *winWindow) SetAcceptDrops(on bool)    { v.w.SetAcceptDrops(on) }
func (v *winWindow) AcceptDrops() bool         { return v.w.AcceptDrops() }
func (v *winWindow) SetImePosition(x, y int32) { v.w.SetImePosition(x, y) }

func (v *winWindow) RegisterHotKey(mods, vk uint32, fn func()) int32 {
	return v.w.RegisterHotKey(mods, vk, fn)
}
func (v *winWindow) UnregisterHotkey(id int32) { v.w.UnregisterHotkey(id) }

func (v *winWindow) Close()   { v.w.Close() }
func (v *winWindow) Destroy() { v.w.Destroy() }

// fromWin32 把 Win32 事件翻译成平台无关事件。
func fromWin32(ev win32.Event) Event {
	return Event{
		Type:     EventType(ev.Type),
		X:        ev.X,
		Y:        ev.Y,
		DX:       ev.DX,
		DY:       ev.DY,
		Button:   ev.Button,
		Key:      ev.Key,
		Char:     ev.Char,
		Runes:    ev.Runes,
		Width:    ev.Width,
		Height:   ev.Height,
		Scale:    ev.Scale,
		Text:     ev.Text,
		Caret:    ev.Caret,
		Paths:    ev.Paths,
		HotkeyID: ev.HotkeyID,
	}
}
