//go:build !windows && !js && !linux && !freebsd && !darwin

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "errors"

// 非 Windows / 非 Linux / 非 Web 平台的窗口后端骨架（macOS / BSD / Android / iOS）。
//
// 这些平台的窗口层分别需要 Cocoa（NSWindow + NSOpenGLView）、Android
// （NativeActivity + EGL）、UIKit（UIWindow + CAEAGLLayer）实现，属于后续阶段；
// 这里先提供**完整的 Backend 实现骨架**：创建时返回明确错误，接口方法全部
// 安全空操作。收益是 `GOOS=darwin go build ./...` 等目标现在就能通过，
// 平台后端可以按文件增量替换，不必再改 app.go。

// errNoWindowBackend 当前平台尚无窗口后端。
var errNoWindowBackend = errors.New("ui: 当前平台尚未实现窗口后端（已支持 Windows / Linux X11 / Web）")

// platformWin 占位窗口后端。
type platformWin struct {
	handler func(Event)
	closed  bool
}

// newBackend 创建窗口（本平台未实现，返回明确错误）。
func newBackend(opts Options) (*platformWin, error) {
	return &platformWin{}, errNoWindowBackend
}

// pollMessages 处理一轮消息（无窗口 → false，主循环立即结束）。
func pollMessages() bool { return false }

func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }
func (p *platformWin) MakeCurrent()                   {}
func (p *platformWin) Swap()                          {}
func (p *platformWin) PixelSize() (int32, int32)      { return 0, 0 }
func (p *platformWin) Scale() float32                 { return 1 }

func (p *platformWin) IsMinimized() bool         { return false }
func (p *platformWin) SetTitle(s string)         {}
func (p *platformWin) SetIcon(ico []byte) bool   { return false }
func (p *platformWin) SetVisible(v bool)         {}
func (p *platformWin) Minimize()                 {}
func (p *platformWin) Restore()                  {}
func (p *platformWin) Center()                   {}
func (p *platformWin) BeginDragMove()            {}
func (p *platformWin) ToggleMaximize()           {}
func (p *platformWin) IsMaximized() bool         { return false }
func (p *platformWin) SetFullscreen(v bool)      {}
func (p *platformWin) IsFullscreen() bool        { return false }
func (p *platformWin) SetAlwaysOnTop(on bool)    {}
func (p *platformWin) AlwaysOnTop() bool         { return false }
func (p *platformWin) SetCornerRadius(v float32) {}
func (p *platformWin) CornerRadius() float32     { return 0 }

func (p *platformWin) SetAcceptDrops(on bool)    {}
func (p *platformWin) AcceptDrops() bool         { return false }
func (p *platformWin) SetImePosition(x, y int32) {}

func (p *platformWin) RegisterHotKey(mods, vk uint32, fn func()) int32 { return 0 }
func (p *platformWin) UnregisterHotkey(id int32)                       {}

func (p *platformWin) AttachCompose(pw, ph int32) {}
func (p *platformWin) HasCompose() bool           { return false }
func (p *platformWin) ResizeCompose(pw, ph int32) {}
func (p *platformWin) ComposeBegin() uint32       { return 0 }
func (p *platformWin) ComposeEnd()                {}

func (p *platformWin) NativeHandle() uintptr { return 0 }

func (p *platformWin) Close()   { p.closed = true }
func (p *platformWin) Destroy() { p.closed = true }
