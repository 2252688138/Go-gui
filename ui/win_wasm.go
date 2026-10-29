//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"syscall/js"

	"gogui/internal/gpu"
)

// 浏览器端窗口层：<canvas> + WebGL 2 + DOM 事件 + requestAnimationFrame。
//
// 与桌面端 win_windows.go 提供完全相同的 platformWin 方法集，因此 app.go 里的
// 布局、绘制、命中、滚动与事件分发一行都不用改——同一套 UI 代码两端通用。

var (
	frameCh = make(chan int, 1)
	winOnce *platformWin
	// frames 已渲染帧数（写到 window.__goguiFrames，浏览器冒烟测试用）
	frames int64
)

// signalFrame 非阻塞放行一帧（rAF 回调里调用）。
func signalFrame() {
	select {
	case frameCh <- 1:
	default:
	}
}

type platformWin struct {
	canvas  js.Value
	handler func(Event)

	pw, ph int32   // 画布像素尺寸
	dpr    float32 // 设备像素比
	uiPx   float32 // 界面缩放（浏览器端固定 1）

	visible    bool
	minimized  bool
	fullscreen bool
	title      string
	closed     bool

	// Hwnd 浏览器端无窗口句柄，占位以保持接口一致
	Hwnd uintptr
	// 指针位置：wheel / mouseup 等事件不带坐标时复用
	lastX, lastY float32

	raf js.Func
}

// newPlatformWindow 创建浏览器端窗口（canvas 已存在则复用，否则自动插入）。
func newPlatformWindow(opts Options) (*platformWin, error) {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "screen")
	if canvas.IsNull() || canvas.IsUndefined() {
		canvas = doc.Call("createElement", "canvas")
		canvas.Set("id", "screen")
		doc.Get("body").Call("appendChild", canvas)
	}
	if err := gpu.InitWasm(canvas); err != nil {
		return nil, err
	}

	p := &platformWin{canvas: canvas, visible: true, title: opts.Title, uiPx: 1}
	p.dpr = float32(js.Global().Get("devicePixelRatio").Float())
	if p.dpr <= 0 {
		p.dpr = 1
	}
	p.resize()
	p.bindEvents()
	winOnce = p
	return p, nil
}

// pollMessages 等待下一帧（rAF 节拍），让共享的 runLoop 在浏览器里同样按帧推进。
func pollMessages() bool {
	if winOnce == nil || winOnce.closed {
		return false
	}
	<-frameCh
	return true
}

func (p *platformWin) resize() {
	w := float32(js.Global().Get("innerWidth").Float())
	h := float32(js.Global().Get("innerHeight").Float())
	if w < 16 {
		w = 1024
	}
	if h < 16 {
		h = 700
	}
	p.pw = int32(w * p.dpr)
	p.ph = int32(h * p.dpr)
	p.canvas.Set("width", int(p.pw))
	p.canvas.Set("height", int(p.ph))
	style := p.canvas.Get("style")
	style.Set("width", fmt.Sprintf("%.0fpx", w))
	style.Set("height", fmt.Sprintf("%.0fpx", h))
	if p.handler != nil {
		p.handler(Event{Type: EvResize, Width: w, Height: h, Scale: p.dpr})
	}
}

func (p *platformWin) bindEvents() {
	move := js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		p.emit(Event{Type: EvMouseMove, X: float32(e.Get("offsetX").Float()), Y: float32(e.Get("offsetY").Float())})
		return nil
	})
	down := js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		p.canvas.Call("focus")
		p.emit(Event{Type: EvMouseDown, X: float32(e.Get("offsetX").Float()), Y: float32(e.Get("offsetY").Float())})
		return nil
	})
	up := js.FuncOf(func(this js.Value, args []js.Value) any {
		p.emit(Event{Type: EvMouseUp, X: p.lastX, Y: p.lastY})
		return nil
	})
	leave := js.FuncOf(func(this js.Value, args []js.Value) any {
		p.emit(Event{Type: EvMouseLeave})
		return nil
	})
	dropOver := js.FuncOf(func(this js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		return nil
	})
	// 拖入文件：浏览器只暴露文件名，这里把文件名当路径交给同一套 OnDrop 逻辑
	drop := js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		var paths []string
		files := e.Get("dataTransfer").Get("files")
		for i := 0; i < files.Length(); i++ {
			paths = append(paths, files.Index(i).Get("name").String())
		}
		if len(paths) == 0 {
			return nil
		}
		p.emit(Event{Type: EvDropFiles, Paths: paths, X: p.lastX, Y: p.lastY})
		return nil
	})
	wheel := js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		p.emit(Event{Type: EvMouseWheel, X: p.lastX, Y: p.lastY, DY: float32(e.Get("deltaY").Float())})
		return nil
	})
	key := js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		k := e.Get("key").String()
		vk := keyCode(k)
		if vk != 0 {
			e.Call("preventDefault")
			p.emit(Event{Type: EvKeyDown, Key: vk})
			return nil
		}
		if len([]rune(k)) == 1 {
			e.Call("preventDefault")
			for _, r := range k {
				p.emit(Event{Type: EvChar, Char: r})
			}
		}
		return nil
	})
	compStart := js.FuncOf(func(this js.Value, args []js.Value) any {
		p.emit(Event{Type: EvCompositionStart})
		return nil
	})
	compUpdate := js.FuncOf(func(this js.Value, args []js.Value) any {
		p.emit(Event{Type: EvComposition, Text: args[0].Get("data").String()})
		return nil
	})
	compEnd := js.FuncOf(func(this js.Value, args []js.Value) any {
		txt := args[0].Get("data").String()
		p.emit(Event{Type: EvCompositionEnd})
		for _, r := range txt {
			p.emit(Event{Type: EvChar, Char: r})
		}
		return nil
	})
	resize := js.FuncOf(func(this js.Value, args []js.Value) any { p.resize(); return nil })

	p.canvas.Call("addEventListener", "mousemove", move)
	p.canvas.Call("addEventListener", "mousedown", down)
	js.Global().Call("addEventListener", "mouseup", up)
	p.canvas.Call("addEventListener", "mouseleave", leave)
	p.canvas.Call("addEventListener", "wheel", wheel, map[string]any{"passive": false})
	p.canvas.Call("addEventListener", "keydown", key)
	p.canvas.Call("addEventListener", "dragover", dropOver)
	p.canvas.Call("addEventListener", "drop", drop)
	p.canvas.Call("addEventListener", "compositionstart", compStart)
	p.canvas.Call("addEventListener", "compositionupdate", compUpdate)
	p.canvas.Call("addEventListener", "compositionend", compEnd)
	js.Global().Call("addEventListener", "resize", resize)
	p.canvas.Set("tabIndex", 0)

	// 帧节拍：rAF 只做「放行一帧」的信号，真正绘制仍在共享 runLoop 里
	p.raf = js.FuncOf(func(this js.Value, args []js.Value) any {
		signalFrame()
		frames++
		js.Global().Set("__goguiFrames", frames)
		js.Global().Call("requestAnimationFrame", p.raf)
		return nil
	})
	js.Global().Call("requestAnimationFrame", p.raf)
}

func (p *platformWin) emit(ev Event) {
	if ev.Type == EvMouseMove || ev.Type == EvMouseDown || ev.Type == EvMouseUp {
		p.lastX, p.lastY = ev.X, ev.Y
	}
	if p.handler != nil {
		p.handler(ev)
	}
}

// keyCode 把 DOM key 映射成 Windows VK 码（框架里只用到少数几个功能键）。
func keyCode(k string) uint32 {
	switch k {
	case "Escape":
		return VK_ESCAPE
	case "Enter":
		return VK_RETURN
	case "Tab":
		return VK_TAB
	case "Backspace":
		return VK_BACK
	case "Delete":
		return VK_DELETE
	case "ArrowLeft":
		return VK_LEFT
	case "ArrowRight":
		return VK_RIGHT
	case "ArrowUp":
		return VK_UP
	case "ArrowDown":
		return VK_DOWN
	case " ":
		return VK_SPACE
	}
	return 0
}

// ---- platformWin 方法集（与桌面端一一对应）----

func (p *platformWin) SetEventHandler(fn func(Event)) { p.handler = fn }
func (p *platformWin) MakeCurrent()                  {}
func (p *platformWin) Swap()                         {}
func (p *platformWin) PixelSize() (int32, int32)     { return p.pw, p.ph }
func (p *platformWin) Scale() float32                { return p.uiPx * p.dpr }
func (p *platformWin) IsMinimized() bool             { return p.minimized }
func (p *platformWin) SetTitle(s string)             { p.title = s }
func (p *platformWin) SetIcon([]byte) bool           { return true }
func (p *platformWin) SetVisible(v bool)             { p.visible = v }
func (p *platformWin) Minimize()                     { p.minimized = true }
func (p *platformWin) Restore()                      { p.minimized = false }
func (p *platformWin) Center()                       {}
func (p *platformWin) BeginDragMove()                {}

func (p *platformWin) SetAlwaysOnTop(bool) {}
func (p *platformWin) AlwaysOnTop() bool   { return false }
func (p *platformWin) SetAcceptDrops(bool) {}
func (p *platformWin) AcceptDrops() bool   { return true }
func (p *platformWin) RegisterHotKey(mods, vk uint32, fn func()) int32 { return 0 }
func (p *platformWin) UnregisterHotkey(id int32)                       {}
func (p *platformWin) SetCornerRadius(float32)                         {}
func (p *platformWin) CornerRadius() float32                           { return 0 }

func (p *platformWin) ToggleMaximize() { p.SetFullscreen(!p.fullscreen) }
func (p *platformWin) IsMaximized() bool { return p.fullscreen }

func (p *platformWin) SetFullscreen(v bool) {
	p.fullscreen = v
	doc := js.Global().Get("document")
	if v {
		doc.Get("documentElement").Call("requestFullscreen")
	} else {
		doc.Call("exitFullscreen")
	}
	p.resize()
}

func (p *platformWin) IsFullscreen() bool       { return p.fullscreen }
func (p *platformWin) SetImePosition(x, y int32) {}

func (p *platformWin) Close() {
	if p.closed {
		return
	}
	p.closed = true
	if p.handler != nil {
		p.handler(Event{Type: EvClose})
	}
}

func (p *platformWin) Destroy() { p.closed = true }

// 合成与硬件句柄：浏览器端不存在，统一空实现。
func (p *platformWin) AttachCompose(pw, ph int32) {}
func (p *platformWin) HasCompose() bool           { return false }
func (p *platformWin) ResizeCompose(pw, ph int32) {}
func (p *platformWin) ComposeBegin() uint32       { return 0 }
func (p *platformWin) ComposeEnd()                {}
