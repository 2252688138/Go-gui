//go:build windows

package ui

import "gogui/internal/win32"

// platformWin 桌面端窗口：包住 win32.Window，把它的方法/事件翻译成平台无关形式，
// 让 app.go 里的布局、绘制、命中与分发逻辑不需要认识任何 Win32 细节。
type platformWin struct {
	*win32.Window
	comp    *win32.Compositor
	handler func(Event)
}

// newPlatformWindow 创建 Win32 窗口（含图标），返回平台窗口。
func newPlatformWindow(opts Options) (*platformWin, error) {
	w, err := win32.NewWindow(win32.Config{
		Title:        opts.Title,
		Width:        opts.Width,
		Height:       opts.Height,
		Resizable:    true,
		Decorated:    !opts.NoTitleBar,
		Transparent:  false,
		CornerRadius: opts.CornerRadius,
	})
	if err != nil {
		return nil, err
	}
	ico := opts.Icon
	if len(ico) == 0 {
		ico = defaultIcon
	}
	_ = w.SetIcon(ico)

	p := &platformWin{Window: w}
	w.SetEventHandler(p.onWin32Event)
	return p, nil
}

// pollMessages 处理一轮窗口消息；返回 false 表示所有窗口都已关闭。
func pollMessages() bool { return win32.PollMessages() }

// SetEventHandler 注册事件回调（App.handleEvent）。
func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }

func (p *platformWin) onWin32Event(ev win32.Event) {
	if p.handler == nil {
		return
	}
	p.handler(Event{
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
	})
}

// AttachCompose 尝试启用 DirectComposition 合成（失败则回退子窗口叠加）。
func (p *platformWin) AttachCompose(pw, ph int32) {
	if !win32.ComposeSupported() {
		return
	}
	comp, err := win32.NewCompositor(p.Hwnd, pw, ph)
	if err != nil {
		win32.Debugf("DComp 合成不可用，回退 A 阶段: %v", err)
		return
	}
	p.comp = comp
}

// ComposeBegin 开始一帧合成，返回离屏目标 FBO（0 = 不用合成）。
func (p *platformWin) ComposeBegin() uint32 {
	if p.comp == nil {
		return 0
	}
	f, _, _, err := p.comp.BeginFrame()
	if err != nil {
		return 0
	}
	return f
}

// HasCompose 是否启用了 DirectComposition 合成。
func (p *platformWin) HasCompose() bool { return p.comp != nil }

// ResizeCompose 尺寸变化时重建合成表面（无合成器时为空操作）。
func (p *platformWin) ResizeCompose(pw, ph int32) {
	if p.comp != nil {
		_ = p.comp.Resize(pw, ph)
	}
}

// ComposeEnd 提交本帧合成。
func (p *platformWin) ComposeEnd() {
	if p.comp != nil {
		p.comp.EndFrame()
	}
}

// Destroy 释放窗口（先关合成器）。
func (p *platformWin) Destroy() {
	if p.comp != nil {
		p.comp.Close()
		p.comp = nil
	}
	if p.Window != nil {
		p.Window.Destroy()
	}
}
