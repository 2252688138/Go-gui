//go:build ios && cgo

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package uikit iOS 窗口层：UIApplication + UIWindow + EAGL(CAEAGLLayer) + GLES3。
//
// ⚠️ 验证状态：本包需要 macOS + Xcode（iOS 强制 cgo + 外部链接），
// 无法在 Windows 上编译验证。逻辑与 Android 的 EGL 实现同构
// （framebuffer 0 直接指向屏幕、触摸合成鼠标事件），代码路径已按 UIKit 文档编写。
package uikit

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework UIKit -framework OpenGLES -framework QuartzCore -framework CoreGraphics
#include <stdlib.h>
#include "bridge_ios.h"
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// 事件种类（与 ui/display 语义对齐）。
const (
	EvNone = iota
	EvTouchDown
	EvTouchMove
	EvTouchUp
	EvKey
	EvChar
	EvResize
	EvFocusGained
	EvFocusLost
)

// Event 原生事件。
type Event struct {
	Kind   int
	X, Y   float32
	Key    uint32
	Char   rune
	Width  float32
	Height float32
	Scale  float32
}

var (
	mu      sync.Mutex
	handler func(Event)
	window  uintptr
	width   int32
	height  int32
	scale   float32 = 1
	ready   bool
)

// SetHandler 注册事件回调。
func SetHandler(fn func(Event)) {
	mu.Lock()
	handler = fn
	mu.Unlock()
}

func emit(e Event) {
	mu.Lock()
	h := handler
	mu.Unlock()
	if h != nil {
		h(e)
	}
}

// Init 初始化 UIWindow 与 EAGL 视图。
func Init() error {
	if ready {
		return nil
	}
	w := C.gg_ios_init()
	if w == nil {
		return errors.New("uikit: UIWindow 创建失败（不在 iOS 应用进程中？）")
	}
	window = uintptr(w)
	if C.gg_ios_setup_gl() == 0 {
		return errors.New("uikit: EAGL renderbuffer 创建失败（设备不支持 OpenGL ES 3）")
	}
	refreshSize()
	ready = true
	return nil
}

// refreshSize 同步物理像素尺寸与缩放。
func refreshSize() {
	var w, h C.int
	var sc C.double
	C.gg_ios_size(&w, &h, &sc)
	mu.Lock()
	width, height = int32(w), int32(h)
	if sc > 0 {
		scale = float32(sc)
	}
	mu.Unlock()
}

// Size 绘制缓冲的物理像素尺寸。
func Size() (int32, int32) {
	mu.Lock()
	defer mu.Unlock()
	return width, height
}

// Scale 物理像素 / 逻辑像素（Retina 为 2 或 3）。
func Scale() float32 {
	mu.Lock()
	defer mu.Unlock()
	return scale
}

// MakeCurrent 绑定 EAGL 上下文与 framebuffer。
func MakeCurrent() bool { return C.gg_ios_make_current() != 0 }

// Present 提交一帧。
func Present() { C.gg_ios_present() }

// Proc 取 GL 函数地址。
func Proc(name string) uintptr {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return uintptr(C.gg_ios_gl_proc(cs))
}

// SetTitle iOS 无窗口标题（显示在 App 名/导航条，一期忽略）。
func SetTitle(s string) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.gg_ios_set_title(unsafe.Pointer(window), cs)
}

// SetTouchEnabled 是否接收触摸。
func SetTouchEnabled(on bool) {
	var v C.int
	if on {
		v = 1
	}
	C.gg_ios_set_touch_events(v)
}

// Poll 泵一轮（UIKit 事件由 runloop 派发，这里只同步尺寸）。
func Poll() { C.gg_ios_poll() }

// Ready 是否已初始化。
func Ready() bool {
	mu.Lock()
	defer mu.Unlock()
	return ready
}

// NativeHandle 原生窗口句柄（UIWindow*）。
func NativeHandle() uintptr {
	mu.Lock()
	defer mu.Unlock()
	return window
}

// ---- 以下为 Objective-C 回调（bridge_ios.m 调用）----

//export ggIOSTouch
func ggIOSTouch(phase C.int, x, y C.double) {
	switch int(phase) {
	case 0:
		emit(Event{Kind: EvTouchDown, X: float32(x), Y: float32(y)})
	case 1:
		emit(Event{Kind: EvTouchMove, X: float32(x), Y: float32(y)})
	default:
		emit(Event{Kind: EvTouchUp, X: float32(x), Y: float32(y)})
	}
}

//export ggIOSResize
func ggIOSResize(w, h C.int, sc C.double) {
	mu.Lock()
	changed := int32(w) != width || int32(h) != height
	width, height = int32(w), int32(h)
	if sc > 0 {
		scale = float32(sc)
	}
	s := scale
	mu.Unlock()
	if changed {
		emit(Event{Kind: EvResize, Width: float32(w), Height: float32(h), Scale: s})
	}
}

//export ggIOSKey
func ggIOSKey(keyCode C.int, chars *C.char, down C.int) {
	if int(down) == 0 {
		return
	}
	if keyCode != 0 {
		emit(Event{Kind: EvKey, Key: uint32(keyCode)})
		return
	}
	if chars == nil {
		return
	}
	for _, r := range C.GoString(chars) {
		if r >= 0x20 && r != 0x7f {
			emit(Event{Kind: EvChar, Char: r})
		}
	}
}
