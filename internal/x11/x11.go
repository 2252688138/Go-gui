//go:build (linux || freebsd) && !android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package x11 极简 Xlib 绑定（purego 动态加载 libX11，零 cgo）。
//
// 只绑定框架真正用到的函数：建窗口、收事件、改尺寸、置顶/全屏、光标键盘输入。
// Xlib 的结构体在 LP64 上按 C 的 `long` 对齐（8 字节），因此事件一律放在
// 固定长度的 [196]byte 缓冲里，用**偏移量**读取字段，不做 Go 结构体映射——
// 这样既不用 cgo，也不会因为结构体对齐规则差异读错字节。
package x11

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Display X 服务器连接。
type Display struct {
	p      uintptr
	root   Window
	screen int
	atoms  map[string]uintptr
}

// Window X 窗口 id。
type Window uintptr

// 已解析的 Xlib 函数。
var (
	xOpenDisplay        func(string) uintptr
	xCloseDisplay       func(uintptr) int
	xDefaultScreen      func(uintptr) int
	xRootWindow         func(uintptr, int) uintptr
	xCreateSimpleWindow func(uintptr, uintptr, int, int, uint, uint, uint, uintptr, uintptr) uintptr
	xSelectInput        func(uintptr, uintptr, uintptr)
	xMapWindow          func(uintptr, uintptr) int
	xUnmapWindow        func(uintptr, uintptr) int
	xDestroyWindow      func(uintptr, uintptr) int
	xStoreName          func(uintptr, uintptr, string) int
	xInternAtom         func(uintptr, string, int) uintptr
	xSetWMProtocols     func(uintptr, uintptr, *uintptr, int) int
	xNextEvent          func(uintptr, unsafe.Pointer)
	xPending            func(uintptr) int
	xFlush              func(uintptr) int
	xSync               func(uintptr, int) int
	xResizeWindow       func(uintptr, uintptr, uint, uint) int
	xMoveResizeWindow   func(uintptr, uintptr, int, int, uint, uint) int
	xMoveWindow         func(uintptr, uintptr, int, int) int
	xIconifyWindow      func(uintptr, uintptr, int) int
	xSetInputFocus      func(uintptr, uintptr, int, uintptr) int
	xLookupKeysym       func(unsafe.Pointer, int) uintptr
	xkbKeycodeToKeysym  func(uintptr, byte, int, int) uintptr
	xLookupString       func(unsafe.Pointer, *byte, int, *uintptr, unsafe.Pointer) int
	xChangeProperty     func(uintptr, uintptr, uintptr, uintptr, int, int, *byte, int) int
	xSendEvent          func(uintptr, uintptr, int, uintptr, unsafe.Pointer) int
	xResourceManagerStr func(uintptr) *byte
	xGetGeometry        func(uintptr, uintptr, *uintptr, *int, *int, *uint, *uint, *uint, *uint) int
)

var loadErr error
var loadOnce sync.Once

// Load 动态加载 libX11（首次调用时执行，失败后重复返回同一错误）。
func Load() error {
	loadOnce.Do(func() {
		var h uintptr
		for _, name := range []string{"libX11.so.6", "libX11.so", "/usr/lib/libX11.so.6", "/usr/local/lib/libX11.so"} {
			if lib, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
				h = lib
				break
			}
		}
		if h == 0 {
			loadErr = errors.New("x11: 找不到 libX11（请安装 libx11-6 / X11 客户端库）")
			return
		}
		bind := func(dst any, name string) {
			if addr, err := purego.Dlsym(h, name); err == nil && addr != 0 {
				purego.RegisterFunc(dst, addr)
			}
		}
		bind(&xOpenDisplay, "XOpenDisplay")
		bind(&xCloseDisplay, "XCloseDisplay")
		bind(&xDefaultScreen, "XDefaultScreen")
		bind(&xRootWindow, "XRootWindow")
		bind(&xCreateSimpleWindow, "XCreateSimpleWindow")
		bind(&xSelectInput, "XSelectInput")
		bind(&xMapWindow, "XMapWindow")
		bind(&xUnmapWindow, "XUnmapWindow")
		bind(&xDestroyWindow, "XDestroyWindow")
		bind(&xStoreName, "XStoreName")
		bind(&xInternAtom, "XInternAtom")
		bind(&xSetWMProtocols, "XSetWMProtocols")
		bind(&xNextEvent, "XNextEvent")
		bind(&xPending, "XPending")
		bind(&xFlush, "XFlush")
		bind(&xSync, "XSync")
		bind(&xResizeWindow, "XResizeWindow")
		bind(&xMoveResizeWindow, "XMoveResizeWindow")
		bind(&xMoveWindow, "XMoveWindow")
		bind(&xIconifyWindow, "XIconifyWindow")
		bind(&xSetInputFocus, "XSetInputFocus")
		bind(&xLookupKeysym, "XLookupKeysym")
		bind(&xkbKeycodeToKeysym, "XkbKeycodeToKeysym")
		bind(&xLookupString, "XLookupString")
		bind(&xChangeProperty, "XChangeProperty")
		bind(&xSendEvent, "XSendEvent")
		bind(&xResourceManagerStr, "XResourceManagerString")
		bind(&xGetGeometry, "XGetGeometry")
		if xOpenDisplay == nil {
			loadErr = errors.New("x11: libX11 缺少 XOpenDisplay 符号")
		}
	})
	return loadErr
}

// Open 连接 X 服务器（DISPLAY 为空时由 Xlib 使用环境变量）。
func Open() (*Display, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	p := xOpenDisplay("")
	if p == 0 {
		return nil, errors.New("x11: 无法连接 X 服务器（DISPLAY 未设置，或未安装 X11 桌面）")
	}
	sc := xDefaultScreen(p)
	return &Display{
		p:      p,
		root:   Window(xRootWindow(p, sc)),
		screen: sc,
		atoms:  map[string]uintptr{},
	}, nil
}

// Close 断开连接。
func (d *Display) Close() {
	if d != nil && d.p != 0 && xCloseDisplay != nil {
		xCloseDisplay(d.p)
		d.p = 0
	}
}

// Ptr 原始 Display* 指针（EGL 需要）。
func (d *Display) Ptr() uintptr { return d.p }

// Root 根窗口。
func (d *Display) Root() Window { return d.root }

// Screen 默认屏幕号。
func (d *Display) Screen() int { return d.screen }

// CreateWindow 创建顶层窗口（无边框参数由 WM 决定，背景先给黑色）。
func (d *Display) CreateWindow(title string, w, h int) Window {
	win := Window(xCreateSimpleWindow(d.p, uintptr(d.root), 0, 0, uint(w), uint(h), 0, 0, 0))
	if title != "" {
		xStoreName(d.p, uintptr(win), title)
	}
	mask := uintptr(KeyPressMask | KeyReleaseMask | ButtonPressMask | ButtonReleaseMask |
		EnterWindowMask | LeaveWindowMask | PointerMotionMask | ExposureMask |
		StructureNotifyMask | FocusChangeMask | PropertyChangeMask)
	xSelectInput(d.p, uintptr(win), mask)
	return win
}

// Atom 取（或创建）原子。
func (d *Display) Atom(name string) uintptr {
	if a, ok := d.atoms[name]; ok {
		return a
	}
	a := xInternAtom(d.p, name, 0)
	d.atoms[name] = a
	return a
}

// SetWMDelete 声明窗口关闭走 WM_DELETE_WINDOW 协议（否则点 × 会被直接杀掉）。
func (d *Display) SetWMDelete(w Window) {
	a := d.Atom("WM_DELETE_WINDOW")
	xSetWMProtocols(d.p, uintptr(w), &a, 1)
}

// Map 显示窗口。
func (d *Display) Map(w Window) {
	xMapWindow(d.p, uintptr(w))
	d.Flush()
}

// Unmap 隐藏窗口。
func (d *Display) Unmap(w Window) { xUnmapWindow(d.p, uintptr(w)) }

// Destroy 销毁窗口。
func (d *Display) Destroy(w Window) { xDestroyWindow(d.p, uintptr(w)) }

// SetTitle 修改标题（同时写 _NET_WM_NAME 以支持 UTF-8）。
func (d *Display) SetTitle(w Window, title string) {
	xStoreName(d.p, uintptr(w), title)
	// _NET_WM_NAME 用 UTF8_STRING 承载中文标题
	utf8 := d.Atom("UTF8_STRING")
	name := d.Atom("_NET_WM_NAME")
	if utf8 != 0 && name != 0 && len(title) > 0 {
		b := []byte(title)
		xChangeProperty(d.p, uintptr(w), name, utf8, 8, 0, &b[0], len(b))
	}
}

// Resize 改窗口尺寸。
func (d *Display) Resize(w Window, width, height int) {
	xResizeWindow(d.p, uintptr(w), uint(width), uint(height))
}

// MoveResize 移动并改尺寸。
func (d *Display) MoveResize(w Window, x, y, width, height int) {
	xMoveResizeWindow(d.p, uintptr(w), x, y, uint(width), uint(height))
}

// Size 读取窗口当前客户区尺寸。
func (d *Display) Size(w Window) (int, int) {
	if xGetGeometry == nil {
		return 0, 0
	}
	var root uintptr
	var x, y int
	var width, height, border, depth uint
	if xGetGeometry(d.p, uintptr(w), &root, &x, &y, &width, &height, &border, &depth) == 0 {
		return 0, 0
	}
	return int(width), int(height)
}

// Iconify 最小化。
func (d *Display) Iconify(w Window) { xIconifyWindow(d.p, uintptr(w), d.screen) }

// SetInputFocus 让窗口获得键盘焦点。
func (d *Display) SetInputFocus(w Window) {
	const revertToParent = 2
	xSetInputFocus(d.p, uintptr(w), revertToParent, 0)
}

// Flush 把请求刷到服务器。
func (d *Display) Flush() { xFlush(d.p) }

// Sync 等待服务器处理完。
func (d *Display) Sync() { xSync(d.p, 0) }

// Pending 是否有待处理事件。
func (d *Display) Pending() int { return xPending(d.p) }

// NextEvent 阻塞读取一个事件。
func (d *Display) NextEvent() *Event {
	ev := &Event{}
	if xNextEvent != nil {
		xNextEvent(d.p, unsafe.Pointer(&ev.Raw[0]))
	}
	return ev
}

// Keysym 取按键对应的 KeySym（layout group 0，level 0）。
func (d *Display) Keysym(ev *Event) uintptr {
	if xkbKeycodeToKeysym != nil {
		return xkbKeycodeToKeysym(d.p, ev.Keycode(), 0, 0)
	}
	if xLookupKeysym != nil {
		return xLookupKeysym(unsafe.Pointer(&ev.Raw[0]), 0)
	}
	return 0
}

// LookupString 取按键对应的文本（ASCII / Latin-1；中文输入法需要 XIM，见文档）。
func (d *Display) LookupString(ev *Event) string {
	if xLookupString == nil {
		return ""
	}
	var buf [16]byte
	var ks uintptr
	n := xLookupString(unsafe.Pointer(&ev.Raw[0]), &buf[0], len(buf), &ks, nil)
	if n <= 0 {
		return ""
	}
	return string(buf[:n])
}

// SetWMState 设置 _NET_WM_STATE（above / fullscreen 等），需要窗口管理器支持。
func (d *Display) SetWMState(w Window, action int, a1, a2 uintptr) {
	const (
		clientMessage        = 33
		substructureNotify   = 1 << 19
		substructureRedirect = 1 << 20
	)
	var ev [196]byte
	setLong(ev[:], 0, clientMessage)
	setLong(ev[:], 32, uint64(uintptr(w)))
	setLong(ev[:], 40, uint64(d.Atom("_NET_WM_STATE")))
	setLong(ev[:], 48, 32) // format
	setLong(ev[:], 56, uint64(action))
	setLong(ev[:], 64, uint64(a1))
	setLong(ev[:], 72, uint64(a2))
	setLong(ev[:], 80, 1) // source indication: normal application
	mask := uintptr(substructureNotify | substructureRedirect)
	xSendEvent(d.p, uintptr(d.root), 0, mask, unsafe.Pointer(&ev[0]))
	d.Flush()
}

// setLong 写入 little-endian 机器字。

// XftDPI 读取 X 资源里的 Xft.dpi（未设置返回 0）。
func (d *Display) XftDPI() float32 {
	if xResourceManagerStr == nil {
		return 0
	}
	p := xResourceManagerStr(d.p)
	if p == nil {
		return 0
	}
	s := goString(p)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Xft.dpi:") {
			continue
		}
		v := strings.TrimSpace(strings.TrimPrefix(line, "Xft.dpi:"))
		var f float64
		if _, err := fmt.Sscanf(v, "%g", &f); err == nil && f > 0 {
			return float32(f)
		}
	}
	return 0
}

// goString 读取 C 字符串。
func goString(p *byte) string {
	if p == nil {
		return ""
	}
	var n int
	for *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n))) != 0 {
		n++
		if n > 1<<20 {
			break
		}
	}
	return string(unsafe.Slice(p, n))
}
