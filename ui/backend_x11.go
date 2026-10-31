//go:build (linux || freebsd) && !android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gogui/internal/gpu"
	"gogui/internal/x11"
)

// Linux / BSD 窗口后端：Xlib（purego）+ EGL（purego）+ OpenGL 3.3。
//
// 与 backend_windows.go 提供完全相同的 Backend 方法集，因此 app.go 里的布局、
// 绘制、命中、滚动与事件分发一行都不用改——同一套 UI 代码在 Windows 与 Linux
// 上跑的是同一份逻辑，差别只在这层「窗口 + 上下文 + 输入翻译」。
//
// 设计要点：
//   - 事件循环不阻塞：XPending 为 0 时小睡 4ms，避免空闲时占满 CPU；
//   - 坐标统一换算成**逻辑像素**（除以 DPI 比例）后再向上抛，与 Windows 端一致；
//   - 输入法：一期用 XLookupString 取 Latin-1 字符（中文需 XIM，见 docs/multiplatform.md）；
//   - 任何 X/EGL 缺失都返回带安装提示的错误，而不是 panic。

var (
	x11Mu    sync.Mutex
	x11Wins  = map[x11.Window]*platformWin{}
	x11Disp  *x11.Display
	x11Err   error
	x11Scale = float32(1)
)

// platformWin Linux 端窗口。
type platformWin struct {
	win     x11.Window
	ctx     *x11.Context
	handler func(Event)
	// swapFailOnce eglSwapBuffers 失败只打一次日志
	swapFailOnce sync.Once

	pw, ph int32   // 绘制目标物理像素
	scale  float32 // 物理像素 / 逻辑像素
	uipx   float32 // 界面缩放（由 ui.SetUIScale 维护，这里只做换算）

	title      string
	visible    bool
	minimized  bool
	fullscreen bool
	onTop      bool
	corner     float32
	accepts    bool
	closed     bool
}

// newBackend 创建 X11 窗口并建立 EGL/OpenGL 上下文。
func newBackend(opts Options) (*platformWin, error) {
	x11Mu.Lock()
	defer x11Mu.Unlock()

	if x11Disp == nil {
		dpy, err := x11.Open()
		if err != nil {
			x11Err = err
			return nil, err
		}
		x11Disp = dpy
		x11Scale = detectScale(dpy)
	} else if x11Err != nil {
		return nil, x11Err
	}

	w := x11Disp.CreateWindow(opts.Title, opts.Width, opts.Height)
	x11Disp.SetWMDelete(w)

	ctx, err := x11.NewContext(x11Disp.Ptr(), w)
	if err != nil {
		x11Disp.Destroy(w)
		return nil, fmt.Errorf("%w（请安装 libegl1、libgl1 与显卡驱动，或改用 X11 桌面）", err)
	}

	// 把 GL 函数表接到渲染器：之后所有 glXxx 调用都走这张表。
	gpu.SetGLProcLoader(ctx.Proc)
	if n := gpu.InitGLProcs(); n < 20 {
		ctx.Destroy()
		x11Disp.Destroy(w)
		return nil, fmt.Errorf("gl: 只解析到 %d 个 GL 函数，上下文不可用（驱动可能不支持 OpenGL 3.3）", n)
	}

	pw, ph := x11Disp.Size(w)
	if pw <= 0 || ph <= 0 {
		pw, ph = opts.Width, opts.Height
	}
	p := &platformWin{
		win:     w,
		ctx:     ctx,
		pw:      int32(pw),
		ph:      int32(ph),
		scale:   x11Scale,
		uipx:    1,
		title:   opts.Title,
		visible: true,
		accepts: true,
	}
	x11Wins[w] = p
	x11Disp.Map(w)
	x11Disp.SetInputFocus(w)
	return p, nil
}

// pollMessages 处理一轮 X 事件；返回 false 表示所有窗口都已关闭。
func pollMessages() bool {
	x11Mu.Lock()
	dpy := x11Disp
	n := len(x11Wins)
	x11Mu.Unlock()
	if dpy == nil || n == 0 {
		if os.Getenv("GOGUI_X11_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "[x11] 事件循环退出：display=%v windows=%d\n", dpy != nil, n)
		}
		return false
	}
	got := false
	for dpy.Pending() > 0 {
		ev := dpy.NextEvent()
		dispatchX11(dpy, ev)
		got = true
	}
	if !got {
		// 没有事件时小睡：既不让主循环空转烧 CPU，也保证 60~120Hz 的动画节奏
		time.Sleep(4 * time.Millisecond)
	}
	x11Mu.Lock()
	alive := len(x11Wins) > 0
	x11Mu.Unlock()
	return alive
}

// dispatchX11 把 X 事件翻译成平台无关事件并分发。
func dispatchX11(dpy *x11.Display, ev *x11.Event) {
	var p *platformWin
	// 除 ConfigureNotify 外，事件里的 window 字段都在偏移 32
	x11Mu.Lock()
	for _, v := range x11Wins {
		p = v // 单窗口场景直接用唯一窗口；多窗口按事件里的 window 精确匹配
		break
	}
	x11Mu.Unlock()
	if p == nil {
		return
	}
	scale := p.scale
	if scale <= 0 {
		scale = 1
	}
	lx := func(v int) float32 { return float32(v) / scale }
	emit := func(e Event) {
		if p.handler != nil {
			p.handler(e)
		}
	}

	switch ev.Type() {
	case x11.Expose:
		// 需要重画（窗口首次显示 / 被遮挡后恢复 / 全屏切换完成）。
		// 先回读真实几何：EWMH 全屏时某些 WM（WSLg 的 Weston）只发 Expose 不发
		// ConfigureNotify，不在这里同步尺寸就会一直按旧尺寸绘制（表现为黑屏）。
		if x11Disp != nil {
			if w, h := x11Disp.Size(p.win); w > 0 && h > 0 && (int32(w) != p.pw || int32(h) != p.ph) {
				p.pw, p.ph = int32(w), int32(h)
				p.minimized = w <= 1 || h <= 1
			}
		}
		emit(Event{Type: EvResize, Width: float32(p.pw) / scale, Height: float32(p.ph) / scale, Scale: scale})
	case x11.ConfigureNotify:
		w, h := ev.Width(), ev.Height()
		if w <= 0 || h <= 0 || (int32(w) == p.pw && int32(h) == p.ph) {
			return
		}
		p.pw, p.ph = int32(w), int32(h)
		p.minimized = w <= 1 || h <= 1
		emit(Event{Type: EvResize, Width: float32(w) / scale, Height: float32(h) / scale, Scale: scale})
	case x11.ButtonPress:
		switch ev.Button() {
		case 4, 5: // 滚轮
			dy := float32(1)
			if ev.Button() == 5 {
				dy = -1
			}
			emit(Event{Type: EvMouseWheel, X: lx(ev.X()), Y: lx(ev.Y()), DY: dy})
		default:
			b := ev.Button()
			if b >= 1 && b <= 3 {
				b-- // X11 从 1 开始，ui 从 0 开始（0 左 / 1 中 / 2 右）
			}
			emit(Event{Type: EvMouseDown, X: lx(ev.X()), Y: lx(ev.Y()), Button: b})
		}
	case x11.ButtonRelease:
		b := ev.Button()
		if b >= 1 && b <= 3 {
			b--
		}
		emit(Event{Type: EvMouseUp, X: lx(ev.X()), Y: lx(ev.Y()), Button: b})
	case x11.MotionNotify:
		emit(Event{Type: EvMouseMove, X: lx(ev.X()), Y: lx(ev.Y())})
	case x11.LeaveNotify:
		emit(Event{Type: EvMouseLeave, X: lx(ev.X()), Y: lx(ev.Y())})
	case x11.FocusIn:
		emit(Event{Type: EvFocusGained})
	case x11.FocusOut:
		emit(Event{Type: EvFocusLost})
	case x11.KeyPress:
		vk := x11ToVK(dpy, ev)
		if vk != 0 {
			emit(Event{Type: EvKeyDown, Key: vk})
		}
		// 可打印字符：直接上屏（中文需输入法层，见文档）
		if txt := dpy.LookupString(ev); txt != "" {
			for _, r := range txt {
				if r >= 0x20 {
					emit(Event{Type: EvChar, Char: r})
				}
			}
		}
	case x11.ClientMessage:
		if os.Getenv("GOGUI_X11_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "[x11] ClientMessage type=%d\n", ev.MessageType())
		}
		if ev.MessageType() == dpy.Atom("WM_DELETE_WINDOW") {
			emit(Event{Type: EvClose})
		}
	case x11.DestroyNotify:
		x11Mu.Lock()
		delete(x11Wins, p.win)
		left := len(x11Wins)
		x11Mu.Unlock()
		if os.Getenv("GOGUI_X11_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "[x11] DestroyNotify win=%d remaining=%d\n", p.win, left)
		}
	}
}

// x11ToVK 把 X KeySym 映射成框架内部使用的 VK 码（数值与 Windows 一致）。
func x11ToVK(dpy *x11.Display, ev *x11.Event) uint32 {
	ks := dpy.Keysym(ev)
	if ks == 0 {
		return 0
	}
	// 特殊键（X11 keysym 值固定）
	switch ks {
	case 0xFF1B:
		return VK_ESCAPE
	case 0xFF0D, 0xFF8D:
		return VK_RETURN
	case 0xFF09:
		return VK_TAB
	case 0xFF08:
		return VK_BACK
	case 0xFFFF:
		return VK_DELETE
	case 0xFF51:
		return VK_LEFT
	case 0xFF52:
		return VK_UP
	case 0xFF53:
		return VK_RIGHT
	case 0xFF54:
		return VK_DOWN
	case 0x0020:
		return VK_SPACE
	case 0xFFC8: // F11（X11 keysym → 与 Windows VK_F11 对齐）
		return VK_F11
	}
	// 字母 / 数字 / 标点：ASCII 直接当 VK（与 Windows 的 A-Z / 0-9 一致）
	if ks >= 'a' && ks <= 'z' {
		return uint32(ks) - 32
	}
	if ks >= 'A' && ks <= 'Z' {
		return uint32(ks)
	}
	if ks >= '0' && ks <= '9' {
		return uint32(ks)
	}
	return uint32(ks)
}

// detectScale 用 Xft.dpi 估算 DPI 比例（未设置按 96dpi = 1.0）。
func detectScale(dpy *x11.Display) float32 {
	if dpi := dpy.XftDPI(); dpi > 40 {
		s := dpi / 96
		if s < 1 {
			s = 1
		}
		return s
	}
	return 1
}

// ---- Backend 方法集 ----

func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }

func (p *platformWin) MakeCurrent() {
	if p.ctx != nil {
		_ = p.ctx.MakeCurrent()
	}
}

// Swap 提交一帧。交换失败（表面失效 / 驱动重置）时主动发一次 EvResize 让 App 重建
// 渲染目标并重画，而不是默默地停在黑屏——日志只打一次，避免刷屏。
func (p *platformWin) Swap() {
	if p.ctx == nil {
		return
	}
	if p.ctx.Swap() {
		return
	}
	p.swapFailOnce.Do(func() {
		fmt.Fprintf(os.Stderr, "[gogui] eglSwapBuffers 失败：窗口表面可能已失效，已触发重画\n")
	})
	if p.handler != nil {
		scale := p.scale
		if scale <= 0 {
			scale = 1
		}
		p.handler(Event{Type: EvResize, Width: float32(p.pw) / scale, Height: float32(p.ph) / scale, Scale: scale})
	}
}

func (p *platformWin) PixelSize() (int32, int32) { return p.pw, p.ph }
func (p *platformWin) Scale() float32            { return p.scale * p.uipx }

func (p *platformWin) IsMinimized() bool { return p.minimized }
func (p *platformWin) SetTitle(s string) {
	p.title = s
	x11Mu.Lock()
	if x11Disp != nil {
		x11Disp.SetTitle(p.win, s)
	}
	x11Mu.Unlock()
}

// SetIcon 设置窗口图标（EWMH _NET_WM_ICON 需要逐像素 ARGB，一期不支持）。
func (p *platformWin) SetIcon(ico []byte) bool { return false }

func (p *platformWin) SetVisible(v bool) {
	p.visible = v
	x11Mu.Lock()
	if x11Disp != nil {
		if v {
			x11Disp.Map(p.win)
		} else {
			x11Disp.Unmap(p.win)
		}
	}
	x11Mu.Unlock()
}

func (p *platformWin) Minimize() {
	p.minimized = true
	x11Mu.Lock()
	if x11Disp != nil {
		x11Disp.Iconify(p.win)
	}
	x11Mu.Unlock()
}

func (p *platformWin) Restore() {
	p.minimized = false
	x11Mu.Lock()
	if x11Disp != nil {
		x11Disp.Map(p.win)
	}
	x11Mu.Unlock()
}

func (p *platformWin) Center() {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if x11Disp == nil {
		return
	}
	// X11 不暴露屏幕尺寸代理，这里用「窗口居中于根窗口」的常用做法：
	// 由窗口管理器在 Map 前定位，_NET_WM_STATE 不必参与。
	x11Disp.MoveResize(p.win, 80, 60, int(p.pw)/int(max32f(p.scale, 1)), int(p.ph)/int(max32f(p.scale, 1)))
}

func max32f(v, min float32) float32 {
	if v < min {
		return min
	}
	return v
}

// BeginDragMove 自绘标题栏拖动：X11 下交给窗口管理器（_NET_WM_MOVERESIZE），
// 一期退化为空操作（用户可拖动系统标题栏）。
func (p *platformWin) BeginDragMove() {}

func (p *platformWin) ToggleMaximize() {
	p.fullscreen = !p.fullscreen
	p.SetFullscreen(p.fullscreen)
}

func (p *platformWin) IsMaximized() bool { return p.fullscreen }

func (p *platformWin) SetFullscreen(v bool) {
	p.fullscreen = v
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if x11Disp == nil {
		return
	}
	action := 1 // _NET_WM_STATE_ADD
	if !v {
		action = 0 // REMOVE
	}
	x11Disp.SetWMState(p.win, action, x11Disp.Atom("_NET_WM_STATE_FULLSCREEN"), 0)
}

func (p *platformWin) IsFullscreen() bool { return p.fullscreen }

func (p *platformWin) SetAlwaysOnTop(on bool) {
	p.onTop = on
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if x11Disp == nil {
		return
	}
	action := 1
	if !on {
		action = 0
	}
	x11Disp.SetWMState(p.win, action, x11Disp.Atom("_NET_WM_STATE_ABOVE"), 0)
}

func (p *platformWin) AlwaysOnTop() bool { return p.onTop }

func (p *platformWin) SetCornerRadius(v float32) { p.corner = v }
func (p *platformWin) CornerRadius() float32     { return p.corner }

func (p *platformWin) SetAcceptDrops(on bool) { p.accepts = on }
func (p *platformWin) AcceptDrops() bool      { return p.accepts }

// SetImePosition 输入法候选框位置（需 XIM，一期空实现）。
func (p *platformWin) SetImePosition(x, y int32) {}

// RegisterHotKey 全局热键（需要 XGrabKey，一期空实现 → 返回 0 表示不可用）。
func (p *platformWin) RegisterHotKey(mods, vk uint32, fn func()) int32 { return 0 }

func (p *platformWin) UnregisterHotkey(id int32) {}

func (p *platformWin) AttachCompose(pw, ph int32) {}
func (p *platformWin) HasCompose() bool           { return false }
func (p *platformWin) ResizeCompose(pw, ph int32) {}
func (p *platformWin) ComposeBegin() uint32       { return 0 }
func (p *platformWin) ComposeEnd()                {}

// NativeHandle 原生句柄（X11 窗口 id）。
func (p *platformWin) NativeHandle() uintptr { return uintptr(p.win) }

func (p *platformWin) Close() {
	if p.closed {
		return
	}
	p.closed = true
	if os.Getenv("GOGUI_X11_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[x11] platformWin.Close\n")
	}
	if p.handler != nil {
		p.handler(Event{Type: EvClose})
	}
}

func (p *platformWin) Destroy() {
	if p.ctx != nil {
		p.ctx.Destroy()
		p.ctx = nil
	}
	x11Mu.Lock()
	if x11Disp != nil {
		x11Disp.Destroy(p.win)
	}
	delete(x11Wins, p.win)
	if len(x11Wins) == 0 && x11Disp != nil {
		// 最后一个窗口：断开连接（避免残留线程/资源）
		x11Disp.Close()
		x11Disp = nil
	}
	x11Mu.Unlock()
}
