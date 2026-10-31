//go:build darwin && !ios

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package cocoa 在 Go 侧用 purego 驱动 AppKit：NSApplication + NSWindow +
// 自定义 NSView 子类（事件回调）+ NSOpenGLContext。全程零 cgo。
//
// 事件注入只使用「参数全是对象指针」的委托方法（mouseDown: / keyDown: …），
// 因为 purego 的 NewCallback 不支持结构体参数；矩形/坐标在回调内部另行查询。
// 尺寸变化不走回调（会带 NSSize 参数），改由 PumpEvents 轮询 bounds 检测。
package cocoa

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/ebitengine/purego"

	"gogui/internal/objc"
)

// AppKit 常量（NSOpenGL.h / NSWindow.h）。
const (
	attrAllRenderers    = 1
	attrDoubleBuffer    = 5
	attrColorSize       = 8
	attrAlphaSize       = 11
	attrDepthSize       = 12
	attrAccelerated     = 73
	attrProfileVersion  = 99
	attrVersion3_2Core  = 0x3200
	ctxSwapInterval     = 222
	styleTitled         = 1 << 0
	styleClosable       = 1 << 1
	styleMiniaturizable = 1 << 2
	styleResizable      = 1 << 3
	backingBuffered     = 2
	activationRegular   = 0
	levelFloating       = 3
	levelNormal         = 0
)

// 事件种类（与 ui/display 的事件语义对齐）。
const (
	EvNone = iota
	EvMouseMove
	EvMouseDown
	EvMouseUp
	EvMouseWheel
	EvMouseLeave
	EvKeyDown
	EvKeyUp
	EvChar
	EvResize
	EvClose
	EvFocusGained
	EvFocusLost
)

// Event 原生事件。
type Event struct {
	Kind   int
	X, Y   float32
	DY     float32
	Button int
	Key    uint32
	Char   rune
	Width  float32
	Height float32
}

var (
	mu      sync.Mutex
	windows = map[uintptr]*Window{} // NSView* -> Window

	appInited bool
	initErr   error
	// callbacks 必须常驻，否则 C 侧函数指针会失效
	callbacks []uintptr
	viewClass uintptr

	glGetProc func(string) uintptr
)

// InitApp 初始化 NSApplication（幂等，必须在主线程首次调用）。
func InitApp() error {
	if appInited || initErr != nil {
		return initErr
	}
	runtime.LockOSThread() // AppKit 要求 UI 操作在主线程
	if err := objc.Load(); err != nil {
		initErr = err
		return err
	}
	cls := objc.Class("NSApplication")
	if cls == 0 {
		initErr = errors.New("cocoa: 找不到 NSApplication（不在 macOS 图形会话中？）")
		return initErr
	}
	app := objc.Send(cls, "sharedApplication")
	if app == 0 {
		initErr = errors.New("cocoa: NSApplication 初始化失败")
		return initErr
	}
	objc.SendI(app, "setActivationPolicy:", activationRegular)
	installAppMenu(app)
	objc.Send(app, "finishLaunching")
	loadGL()
	appInited = true
	return nil
}

// loadGL 绑定 OpenGL.framework 的 CGLGetProcAddress（取 GL 函数地址用）。
func loadGL() {
	h, err := purego.Dlopen("/System/Library/Frameworks/OpenGL.framework/OpenGL", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	if addr, e := purego.Dlsym(h, "CGLGetProcAddress"); e == nil && addr != 0 {
		purego.RegisterFunc(&glGetProc, addr)
	}
}

// installAppMenu 最小菜单栏（App 菜单 + Quit），否则 ⌘Q 无效。
func installAppMenu(app uintptr) {
	menubar := objc.Send(objc.Send(objc.Class("NSMenu"), "alloc"), "init")
	if menubar == 0 {
		return
	}
	appItem := objc.Send(objc.Send(objc.Class("NSMenuItem"), "alloc"), "init")
	objc.Send1(menubar, "addItem:", appItem)

	appMenu := objc.Send(objc.Send(objc.Class("NSMenu"), "alloc"), "init")
	quit := objc.Send(objc.Class("NSMenuItem"), "alloc")
	quit = objc.Send3(quit, "initWithTitle:action:keyEquivalent:",
		objc.NSString("Quit"), objc.Sel("terminate:"), objc.NSString("q"))
	objc.Send1(appMenu, "addItem:", quit)
	objc.Send1(appItem, "setSubmenu:", appMenu)
	objc.Send1(app, "setMainMenu:", menubar)
}

// ensureViewClass 创建带事件回调的 NSView 子类（只创建一次）。
func ensureViewClass() uintptr {
	if viewClass != 0 {
		return viewClass
	}
	if c := objc.Class("GoguiView"); c != 0 {
		viewClass = c
		return c
	}
	cls := objc.Subclass(objc.Class("NSView"), "GoguiView")
	if cls == 0 {
		return 0
	}

	// 鼠标按下/松开（带 buttonNumber）
	mouseBtn := func(kind int) uintptr {
		return purego.NewCallback(func(self, cmd, ev uintptr) uintptr {
			if w := windowOf(self); w != nil {
				p := objc.PointOf(ev, "locationInWindow")
				w.emit(Event{
					Kind: kind, X: float32(p.X), Y: w.height - float32(p.Y),
					Button: int(objc.IntOf(ev, "buttonNumber")),
				})
			}
			return 0
		})
	}
	// 拖动 / 移动（同一处理）
	mouseMove := purego.NewCallback(func(self, cmd, ev uintptr) uintptr {
		if w := windowOf(self); w != nil {
			p := objc.PointOf(ev, "locationInWindow")
			w.emit(Event{Kind: EvMouseMove, X: float32(p.X), Y: w.height - float32(p.Y)})
		}
		return 0
	})
	// 滚轮（scrollingDeltaY 为正表示向上滚动）
	scroll := purego.NewCallback(func(self, cmd, ev uintptr) uintptr {
		if w := windowOf(self); w != nil {
			p := objc.PointOf(ev, "locationInWindow")
			dy := objc.FloatOf(ev, "scrollingDeltaY")
			w.emit(Event{
				Kind: EvMouseWheel, X: float32(p.X), Y: w.height - float32(p.Y),
				DY: float32(dy / 10),
			})
		}
		return 0
	})
	keyDown := purego.NewCallback(func(self, cmd, ev uintptr) uintptr {
		if w := windowOf(self); w != nil {
			w.handleKey(ev, true)
		}
		return 0
	})
	keyUp := purego.NewCallback(func(self, cmd, ev uintptr) uintptr {
		if w := windowOf(self); w != nil {
			w.handleKey(ev, false)
		}
		return 0
	})
	moved := purego.NewCallback(func(self, cmd, ev uintptr) uintptr {
		if w := windowOf(self); w != nil {
			p := objc.PointOf(ev, "locationInWindow")
			w.emit(Event{Kind: EvMouseMove, X: float32(p.X), Y: w.height - float32(p.Y)})
		}
		return 0
	})

	callbacks = append(callbacks,
		mouseBtn(EvMouseDown), // 0
		mouseBtn(EvMouseUp),   // 1
		mouseMove,             // 2
		moved,                 // 3
		scroll,                // 4
		keyDown,               // 5
		keyUp,                 // 6
	)
	objc.AddMethod(cls, "mouseDown:", callbacks[0], "v@:@")
	objc.AddMethod(cls, "mouseUp:", callbacks[1], "v@:@")
	objc.AddMethod(cls, "mouseDragged:", callbacks[2], "v@:@")
	objc.AddMethod(cls, "mouseMoved:", callbacks[3], "v@:@")
	objc.AddMethod(cls, "scrollWheel:", callbacks[4], "v@:@")
	objc.AddMethod(cls, "keyDown:", callbacks[5], "v@:@")
	objc.AddMethod(cls, "keyUp:", callbacks[6], "v@:@")
	// 右键同左键处理（Button=1 由 buttonNumber 给出）
	objc.AddMethod(cls, "rightMouseDown:", callbacks[0], "v@:@")
	objc.AddMethod(cls, "rightMouseUp:", callbacks[1], "v@:@")

	viewClass = cls
	return cls
}

// windowOf 由 NSView* 找回窗口。
func windowOf(view uintptr) *Window {
	mu.Lock()
	defer mu.Unlock()
	return windows[view]
}

// Window 一个 macOS 窗口 + OpenGL 上下文。
type Window struct {
	win  uintptr // NSWindow*
	view uintptr // GoguiView*
	ctx  uintptr // NSOpenGLContext*

	title  string
	width  float32 // 逻辑尺寸（点）
	height float32
	pw, ph int32 // 物理像素
	scale  float32

	closed     bool
	fullscreen bool
	onTop      bool
	live       bool // 上次轮询时是否可见/最小化

	handler func(Event)
}

// NewWindow 创建窗口并建立 OpenGL 3.2 Core 上下文。
func NewWindow(title string, w, h int) (*Window, error) {
	if err := InitApp(); err != nil {
		return nil, err
	}
	if w <= 0 {
		w = 1024
	}
	if h <= 0 {
		h = 700
	}
	cls := ensureViewClass()
	if cls == 0 {
		return nil, errors.New("cocoa: 无法创建视图子类（NSView 不可用）")
	}

	winObj := objc.Send(objc.Class("NSWindow"), "alloc")
	style := uintptr(styleTitled | styleClosable | styleMiniaturizable | styleResizable)
	win := objc.SendRect3(winObj, "initWithContentRect:styleMask:backing:defer:",
		objc.NewRect(120, 120, float64(w), float64(h)), style, backingBuffered, 0)
	if win == 0 {
		return nil, errors.New("cocoa: NSWindow 创建失败")
	}
	objc.Send1(win, "setTitle:", objc.NSString(title))
	objc.SendB(win, "setAcceptsMouseMovedEvents:", true)
	objc.Send1(win, "setReleasedWhenClosed:", 0)

	view := objc.Send(cls, "alloc")
	view = objc.SendRect(view, "initWithFrame:", objc.NewRect(0, 0, float64(w), float64(h)))
	objc.SendB(view, "setWantsBestResolutionOpenGLSurface:", true)
	objc.Send1(win, "setContentView:", view)

	// 像素格式：硬件加速 + 双缓冲 + 24 位色 + 8 位 alpha + OpenGL 3.2 Core
	// （macOS 上的 OpenGL 最高 4.1，3.2 Core 与桌面端 3.3 的着色器语法兼容）
	attrs := []uint32{
		attrAccelerated, attrDoubleBuffer, attrColorSize, 24, attrAlphaSize, 8,
		attrDepthSize, 0, attrProfileVersion, attrVersion3_2Core, 0,
	}
	pf := objc.Send(objc.Class("NSOpenGLPixelFormat"), "alloc")
	pf = objc.Send1(pf, "initWithAttributes:", objc.U32Ptr(attrs))
	if pf == 0 {
		return nil, errors.New("cocoa: NSOpenGLPixelFormat 创建失败（该机器无 OpenGL 支持）")
	}
	ctx := objc.Send(objc.Class("NSOpenGLContext"), "alloc")
	ctx = objc.Send2(ctx, "initWithFormat:shareContext:", pf, 0)
	if ctx == 0 {
		return nil, errors.New("cocoa: NSOpenGLContext 创建失败")
	}
	objc.Send1(ctx, "setView:", view)
	swap := int32(1) // 垂直同步
	objc.Send2(ctx, "setValues:forParameter:", objc.I32Ptr(&swap), ctxSwapInterval)

	b := objc.RectOf(view, "bounds")
	bs := objc.RectConvert(view, "convertRectToBacking:", b)

	wd := &Window{
		win:    win,
		view:   view,
		ctx:    ctx,
		title:  title,
		width:  float32(b.Size.W),
		height: float32(b.Size.H),
		pw:     int32(bs.Size.W),
		ph:     int32(bs.Size.H),
		scale:  scaleOf(win),
		live:   true,
	}
	mu.Lock()
	windows[view] = wd
	mu.Unlock()

	objc.Send1(win, "makeKeyAndOrderFront:", 0)
	objc.Send1(win, "makeFirstResponder:", view)
	return wd, nil
}

// scaleOf Retina 比例（backingScaleFactor）。
func scaleOf(win uintptr) float32 {
	if f := objc.FloatOf(win, "backingScaleFactor"); f > 0 {
		return float32(f)
	}
	return 1
}

// SetHandler 注册事件回调。
func (w *Window) SetHandler(fn func(Event)) { w.handler = fn }

func (w *Window) emit(e Event) {
	if w != nil && w.handler != nil {
		w.handler(e)
	}
}

// handleKey 翻译按键：功能键走键码表，可打印字符经 characters 上屏。
func (w *Window) handleKey(ev uintptr, down bool) {
	code := objc.IntOf(ev, "keyCode")
	if vk := vkFromKeyCode(code); vk != 0 {
		if down {
			w.emit(Event{Kind: EvKeyDown, Key: vk})
		} else {
			w.emit(Event{Kind: EvKeyUp, Key: vk})
		}
	}
	if !down {
		return
	}
	// 组合键（⌘/⌃）不当作文本输入
	flags := objc.IntOf(ev, "modifierFlags")
	const cmdFlag, ctrlFlag = 1 << 20, 1 << 18
	if flags&(cmdFlag|ctrlFlag) != 0 {
		return
	}
	if chars := objc.Send(ev, "characters"); chars != 0 {
		for _, r := range objc.UTF8(chars) {
			if r >= 0x20 && r != 0x7f {
				w.emit(Event{Kind: EvChar, Char: r})
			}
		}
	}
}

// MakeCurrent 绑定上下文。
func (w *Window) MakeCurrent() {
	if w != nil && w.ctx != 0 {
		objc.Send(w.ctx, "makeCurrentContext")
	}
}

// Swap 提交一帧。
func (w *Window) Swap() {
	if w != nil && w.ctx != 0 {
		objc.Send(w.ctx, "flushBuffer")
	}
}

// Proc 取 GL 函数地址（交给 gpu 的函数表）。
func (w *Window) Proc(name string) uintptr {
	if glGetProc == nil {
		return 0
	}
	return glGetProc(name)
}

// PixelSize 绘制目标物理像素尺寸。
func (w *Window) PixelSize() (int32, int32) { return w.pw, w.ph }

// Scale 物理像素 / 逻辑像素。
func (w *Window) Scale() float32 { return w.scale }

// SetTitle 修改标题。
func (w *Window) SetTitle(s string) {
	w.title = s
	if w.win != 0 {
		objc.Send1(w.win, "setTitle:", objc.NSString(s))
	}
}

// Center 窗口居中。
func (w *Window) Center() {
	if w.win != 0 {
		objc.Send(w.win, "center")
	}
}

// SetAlwaysOnTop 置顶。
func (w *Window) SetAlwaysOnTop(on bool) {
	w.onTop = on
	if w.win == 0 {
		return
	}
	lvl := int64(levelNormal)
	if on {
		lvl = levelFloating
	}
	objc.SendI(w.win, "setLevel:", lvl)
}

// AlwaysOnTop 是否置顶。
func (w *Window) AlwaysOnTop() bool { return w.onTop }

// SetFullscreen 切换全屏。
func (w *Window) SetFullscreen(v bool) {
	if w.win == 0 || w.fullscreen == v {
		return
	}
	w.fullscreen = v
	objc.Send1(w.win, "toggleFullScreen:", 0)
}

// IsFullscreen 是否全屏。
func (w *Window) IsFullscreen() bool { return w.fullscreen }

// Minimize 最小化。
func (w *Window) Minimize() {
	if w.win != 0 {
		objc.Send1(w.win, "miniaturize:", 0)
	}
}

// Show 显示 / 隐藏。
func (w *Window) Show(v bool) {
	if w.win == 0 {
		return
	}
	if v {
		objc.Send1(w.win, "makeKeyAndOrderFront:", 0)
	} else {
		objc.Send1(w.win, "orderOut:", 0)
	}
}

// IsMinimized 是否最小化。
func (w *Window) IsMinimized() bool {
	if w.win == 0 {
		return false
	}
	return objc.BoolOf(w.win, "isMiniaturized")
}

// Close 关闭窗口（触发 EvClose）。
func (w *Window) Close() {
	if w == nil || w.closed {
		return
	}
	w.closed = true
	w.emit(Event{Kind: EvClose})
}

// SetImePosition 输入法候选框位置（AppKit 自动跟随插入点，这里空实现）。
func (w *Window) SetImePosition(x, y int32) {}

// NativeHandle 原生视图指针（NSView*）。
func (w *Window) NativeHandle() uintptr { return w.view }

// Destroy 释放窗口与上下文。
func (w *Window) Destroy() {
	if w == nil {
		return
	}
	if w.ctx != 0 {
		objc.Send(w.ctx, "clearDrawable")
	}
	if w.win != 0 {
		objc.Send(w.win, "close")
	}
	mu.Lock()
	delete(windows, w.view)
	mu.Unlock()
	w.win, w.view, w.ctx = 0, 0, 0
}

// poll 轮询窗口尺寸变化（回调带 NSSize 参数，因此不用回调而用轮询）。
func (w *Window) poll() {
	if w == nil || w.view == 0 {
		return
	}
	b := objc.RectOf(w.view, "bounds")
	lw, lh := float32(b.Size.W), float32(b.Size.H)
	if lw == w.width && lh == w.height {
		return
	}
	w.width, w.height = lw, lh
	bs := objc.RectConvert(w.view, "convertRectToBacking:", b)
	w.pw, w.ph = int32(bs.Size.W), int32(bs.Size.H)
	w.scale = scaleOf(w.win)
	w.emit(Event{Kind: EvResize, Width: lw, Height: lh})
}

// PumpEvents 处理一轮 AppKit 事件（不阻塞），并检测尺寸变化。
func PumpEvents() {
	if !appInited {
		return
	}
	app := objc.Send(objc.Class("NSApplication"), "sharedApplication")
	if app != 0 {
		date := objc.Send(objc.Class("NSDate"), "distantPast")
		const maskAny = ^uintptr(0)
		mode := objc.NSString("kCFRunLoopDefaultMode")
		for i := 0; i < 256; i++ { // 单轮上限，避免长帧
			ev := objc.Send4(app, "nextEventMatchingMask:untilDate:inMode:dequeue:", maskAny, date, mode, 1)
			if ev == 0 {
				break
			}
			objc.Send1(app, "sendEvent:", ev)
		}
	}
	mu.Lock()
	list := make([]*Window, 0, len(windows))
	for _, w := range windows {
		list = append(list, w)
	}
	mu.Unlock()
	for _, w := range list {
		w.poll()
	}
}

// WindowCount 当前存活窗口数。
func WindowCount() int {
	mu.Lock()
	defer mu.Unlock()
	return len(windows)
}

// String 调试用描述。
func (w *Window) String() string {
	return fmt.Sprintf("cocoa.Window{%dx%d px, scale=%.2f}", w.pw, w.ph, w.scale)
}

// vkFromKeyCode macOS 虚拟键码 → 框架 VK（数值与 Windows 一致）。
func vkFromKeyCode(code int64) uint32 {
	switch code {
	case 53:
		return 0x1B // Escape
	case 36, 76:
		return 0x0D // Return / Enter
	case 48:
		return 0x09 // Tab
	case 51:
		return 0x08 // Delete(Backspace)
	case 117:
		return 0x2E // Forward Delete
	case 123:
		return 0x25 // Left
	case 124:
		return 0x27 // Right
	case 125:
		return 0x28 // Down
	case 126:
		return 0x26 // Up
	case 49:
		return 0x20 // Space
	}
	return 0
}
