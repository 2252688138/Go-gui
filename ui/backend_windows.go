//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"gogui/internal/display"
	"gogui/internal/win32"
)

// platformWin 桌面端（Windows）窗口后端。
//
// 它把 display.Window（原生窗口 + GL 上下文）适配成 ui.Backend，并额外承担
// Windows 独有的 DirectComposition 合成（网页与自绘内容同层）。
// 其余平台（backend_wasm.go / backend_x11.go / backend_cocoa.go / backend_android.go）
// 提供完全相同的方法集，因此 app.go 一行都不用改。
type platformWin struct {
	display.Window
	comp    *win32.Compositor
	handler func(Event)
}

// newBackend 创建 Win32 窗口（含图标与 WGL 上下文），返回平台窗口。
func newBackend(opts Options) (*platformWin, error) {
	w, err := display.New(display.Config{
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
	w.SetEventHandler(p.onDisplayEvent)
	return p, nil
}

// pollMessages 处理一轮窗口消息；返回 false 表示所有窗口都已关闭。
func pollMessages() bool { return display.Poll() }

// SetEventHandler 注册事件回调（App.handleEvent）。
func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }

// onDisplayEvent 把平台无关事件转成 ui.Event 后交给 App。
func (p *platformWin) onDisplayEvent(ev display.Event) {
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
		Pointer:  ev.Pointer,
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
	comp, err := win32.NewCompositor(p.NativeHandle(), pw, ph)
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

// Compositor 返回 DirectComposition 合成器（未启用时为 nil）。
// 供 webview.go 把网页做成合成图层（B 阶段）。
func (p *platformWin) Compositor() any {
	if p.comp == nil {
		return nil
	}
	return p.comp
}

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
