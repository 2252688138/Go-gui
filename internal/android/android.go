//go:build android && cgo

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package android Android 原生窗口层：NativeActivity + EGL + GLES 3 + AInputQueue。
//
// 这一端有意使用 cgo（移动端唯一可行路径，见 docs/multiplatform.md 的「零 cgo 边界」）：
//   - 窗口与生命周期由 ANativeActivity 回调驱动（等价于 android_native_app_glue）；
//   - 图形走 EGL + OpenGL ES 3（着色器与桌面 OpenGL 3.3 同源，仅版本头不同）；
//   - 输入直接读 AInputQueue（触摸/按键），不需要 Java 介入。
//
// 句柄约定：EGL/ANativeWindow 的指针统一用 uintptr 保存，调用时再转成 C 类型，
// 避免 cgo 对不同 typedef 的指针类型做严格区分（EGLDisplay 不是 void*）。
package android

/*
#cgo LDFLAGS: -landroid -llog -lEGL -lGLESv3
#include <android/native_activity.h>
#include <android/native_window.h>
#include <android/input.h>
#include <android/keycodes.h>
#include <android/looper.h>
#include <EGL/egl.h>
#include <EGL/eglext.h>
#include <GLES3/gl3.h>
#include <stdlib.h>

static int gg_has_events(AInputQueue *q) { return q ? AInputQueue_hasEvents(q) : 0; }
static int gg_get_event(AInputQueue *q, AInputEvent **ev) { return q ? AInputQueue_getEvent(q, ev) : -1; }
static int32_t gg_predispatch(AInputQueue *q, AInputEvent *ev) { return AInputQueue_preDispatchEvent(q, ev); }
static void gg_finish(AInputQueue *q, AInputEvent *ev, int32_t handled) { AInputQueue_finishEvent(q, ev, handled); }
static int gg_window_w(ANativeWindow *w) { return w ? ANativeWindow_getWidth(w) : 0; }
static int gg_window_h(ANativeWindow *w) { return w ? ANativeWindow_getHeight(w) : 0; }
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// 输入事件种类（与 ui 的事件语义对齐）。
const (
	EvNone = iota
	EvTouchDown
	EvTouchMove
	EvTouchUp
	EvKey
	EvChar
	EvResize
	EvBack
	EvFocusGained
	EvFocusLost
)

// EGL 常量（EGL/eglext.h 的 EGL_OPENGL_ES3_BIT）。
const eglOpenGLES3Bit = 0x0040

// Event 原生事件。
type Event struct {
	Kind    int
	X, Y    float32
	Pointer int // 多点触控的手指序号
	Key     uint32
	Char    rune
	Width   float32
	Height  float32
}

var (
	mu       sync.Mutex
	activity uintptr
	window   uintptr
	queue    uintptr
	lastW    int32
	lastH    int32

	eglDisplay uintptr
	eglSurface uintptr
	eglContext uintptr
	eglConfig  uintptr
)

// SetActivity 由 ANativeActivity_onCreate 回调注入。
func SetActivity(a uintptr) {
	mu.Lock()
	activity = a
	mu.Unlock()
}

// CurrentActivity 返回 ANativeActivity 指针（JNI 场景取 JavaVM / clazz）。
func CurrentActivity() uintptr {
	mu.Lock()
	defer mu.Unlock()
	return activity
}

// SetInputQueue 挂上输入队列（由 ANativeActivity 回调注入；0 = 分离）。
func SetInputQueue(q uintptr) {
	mu.Lock()
	queue = q
	mu.Unlock()
}

// SetWindow 由 onNativeWindowCreated / Destroyed 注入窗口并重建 EGL 表面。
func SetWindow(w uintptr) error {
	mu.Lock()
	window = w
	mu.Unlock()
	if w == 0 {
		destroySurface()
		return nil
	}
	return createSurface(w)
}

// WindowSize 读取原生窗口尺寸（旋转后变化）。
func WindowSize() (int32, int32) {
	mu.Lock()
	w := window
	mu.Unlock()
	if w == 0 {
		return 0, 0
	}
	nw := C.gg_window_w((*C.ANativeWindow)(unsafe.Pointer(w)))
	nh := C.gg_window_h((*C.ANativeWindow)(unsafe.Pointer(w)))
	return int32(nw), int32(nh)
}

// createSurface 为原生窗口创建 EGL 表面与 GLES 3 上下文。
func createSurface(w uintptr) error {
	destroySurface()

	dpy := C.eglGetDisplay(C.EGLNativeDisplayType(unsafe.Pointer(C.EGL_DEFAULT_DISPLAY)))
	if uintptr(unsafe.Pointer(dpy)) == 0 {
		return errors.New("android: eglGetDisplay 失败")
	}
	var major, minor C.EGLint
	if C.eglInitialize(dpy, &major, &minor) == 0 {
		return errors.New("android: eglInitialize 失败")
	}
	C.eglBindAPI(C.EGL_OPENGL_ES_API)

	cfgAttribs := []C.EGLint{
		C.EGL_SURFACE_TYPE, C.EGL_WINDOW_BIT,
		C.EGL_RENDERABLE_TYPE, C.EGLint(eglOpenGLES3Bit),
		C.EGL_RED_SIZE, 8, C.EGL_GREEN_SIZE, 8,
		C.EGL_BLUE_SIZE, 8, C.EGL_ALPHA_SIZE, 8,
		C.EGL_DEPTH_SIZE, 0,
		C.EGL_NONE,
	}
	var cfg C.EGLConfig
	var num C.EGLint
	if C.eglChooseConfig(dpy, &cfgAttribs[0], &cfg, 1, &num) == 0 || num < 1 {
		C.eglTerminate(dpy)
		return errors.New("android: eglChooseConfig 找不到可用配置")
	}
	surf := C.eglCreateWindowSurface(dpy, cfg, C.EGLNativeWindowType(unsafe.Pointer(w)), nil)
	if uintptr(unsafe.Pointer(surf)) == 0 {
		C.eglTerminate(dpy)
		return errors.New("android: eglCreateWindowSurface 失败")
	}
	ctxAttribs := []C.EGLint{C.EGL_CONTEXT_CLIENT_VERSION, 3, C.EGL_NONE}
	ctx := C.eglCreateContext(dpy, cfg, nil, &ctxAttribs[0])
	if uintptr(unsafe.Pointer(ctx)) == 0 {
		C.eglDestroySurface(dpy, surf)
		C.eglTerminate(dpy)
		return errors.New("android: eglCreateContext 失败（设备不支持 GLES 3）")
	}
	if C.eglMakeCurrent(dpy, surf, surf, ctx) == 0 {
		C.eglDestroySurface(dpy, surf)
		C.eglTerminate(dpy)
		return errors.New("android: eglMakeCurrent 失败")
	}
	C.eglSwapInterval(dpy, 1)

	nw, nh := WindowSize()
	mu.Lock()
	eglDisplay = uintptr(unsafe.Pointer(dpy))
	eglSurface = uintptr(unsafe.Pointer(surf))
	eglContext = uintptr(unsafe.Pointer(ctx))
	eglConfig = uintptr(unsafe.Pointer(cfg))
	lastW, lastH = nw, nh
	mu.Unlock()
	return nil
}

// destroySurface 释放 EGL 资源。
func destroySurface() {
	mu.Lock()
	dpy := C.EGLDisplay(unsafe.Pointer(eglDisplay))
	surf := C.EGLSurface(unsafe.Pointer(eglSurface))
	ctx := C.EGLContext(unsafe.Pointer(eglContext))
	eglDisplay, eglSurface, eglContext, eglConfig = 0, 0, 0, 0
	mu.Unlock()

	if uintptr(unsafe.Pointer(dpy)) == 0 {
		return
	}
	C.eglMakeCurrent(dpy, nil, nil, nil)
	if uintptr(unsafe.Pointer(ctx)) != 0 {
		C.eglDestroyContext(dpy, ctx)
	}
	if uintptr(unsafe.Pointer(surf)) != 0 {
		C.eglDestroySurface(dpy, surf)
	}
	C.eglTerminate(dpy)
}

// Ready 上下文是否可用。
func Ready() bool {
	mu.Lock()
	defer mu.Unlock()
	return eglDisplay != 0
}

// MakeCurrent 绑定上下文（切后台回来时需重新绑定）。
func MakeCurrent() {
	mu.Lock()
	dpy := C.EGLDisplay(unsafe.Pointer(eglDisplay))
	surf := C.EGLSurface(unsafe.Pointer(eglSurface))
	ctx := C.EGLContext(unsafe.Pointer(eglContext))
	mu.Unlock()
	if uintptr(unsafe.Pointer(dpy)) != 0 && uintptr(unsafe.Pointer(surf)) != 0 && uintptr(unsafe.Pointer(ctx)) != 0 {
		C.eglMakeCurrent(dpy, surf, surf, ctx)
	}
}

// Swap 提交一帧。
func Swap() {
	mu.Lock()
	dpy := C.EGLDisplay(unsafe.Pointer(eglDisplay))
	surf := C.EGLSurface(unsafe.Pointer(eglSurface))
	mu.Unlock()
	if uintptr(unsafe.Pointer(dpy)) != 0 && uintptr(unsafe.Pointer(surf)) != 0 {
		C.eglSwapBuffers(dpy, surf)
	}
}

// Destroy 释放全部图形资源。
func Destroy() { destroySurface() }

// GetProcAddress 按名字取 GL 函数（交给 internal/gpu 的函数表）。
func GetProcAddress(name string) uintptr {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return uintptr(unsafe.Pointer(C.eglGetProcAddress(cs)))
}

// PollEvents 取一轮输入事件（非阻塞，最多 64 个）。
func PollEvents(emit func(Event)) {
	mu.Lock()
	q := (*C.AInputQueue)(unsafe.Pointer(queue))
	mu.Unlock()
	if q == nil {
		pollResize(emit)
		return
	}
	for i := 0; i < 64; i++ {
		if C.gg_has_events(q) == 0 {
			break
		}
		var ev *C.AInputEvent
		if C.gg_get_event(q, &ev) != 0 || ev == nil {
			break
		}
		handled := C.gg_predispatch(q, ev)
		if handled == 0 {
			translateEvent(ev, emit)
		}
		C.gg_finish(q, ev, handled)
	}
	pollResize(emit)
}

// pollResize 尺寸变化（旋转 / 分屏 / 键盘弹出）通过轮询检测。
func pollResize(emit func(Event)) {
	nw, nh := WindowSize()
	mu.Lock()
	changed := nw > 0 && nh > 0 && (nw != lastW || nh != lastH)
	lastW, lastH = nw, nh
	mu.Unlock()
	if changed {
		emit(Event{Kind: EvResize, Width: float32(nw), Height: float32(nh)})
	}
}

// translateEvent 把 AInputEvent 翻译成框架事件。
func translateEvent(ev *C.AInputEvent, emit func(Event)) {
	switch C.AInputEvent_getType(ev) {
	case C.AINPUT_EVENT_TYPE_MOTION:
		action := C.AMotionEvent_getAction(ev)
		masked := action & C.AMOTION_EVENT_ACTION_MASK
		idx := action >> C.AMOTION_EVENT_ACTION_POINTER_INDEX_SHIFT
		x := float32(C.AMotionEvent_getX(ev, C.size_t(idx)))
		y := float32(C.AMotionEvent_getY(ev, C.size_t(idx)))
		switch masked {
		case C.AMOTION_EVENT_ACTION_DOWN, C.AMOTION_EVENT_ACTION_POINTER_DOWN:
			emit(Event{Kind: EvTouchDown, X: x, Y: y, Pointer: int(idx)})
		case C.AMOTION_EVENT_ACTION_MOVE:
			emit(Event{Kind: EvTouchMove, X: x, Y: y, Pointer: int(idx)})
		case C.AMOTION_EVENT_ACTION_UP, C.AMOTION_EVENT_ACTION_POINTER_UP, C.AMOTION_EVENT_ACTION_CANCEL:
			emit(Event{Kind: EvTouchUp, X: x, Y: y, Pointer: int(idx)})
		}
	case C.AINPUT_EVENT_TYPE_KEY:
		code := C.AKeyEvent_getKeyCode(ev)
		if C.AKeyEvent_getAction(ev) == C.AKEY_EVENT_ACTION_DOWN {
			if code == C.AKEYCODE_BACK {
				emit(Event{Kind: EvBack})
				return
			}
			emit(Event{Kind: EvKey, Key: vkFromAndroid(code)})
		}
	}
}

// vkFromAndroid Android 键码 → 框架 VK（数值与 Windows 一致）。
func vkFromAndroid(code C.int32_t) uint32 {
	switch code {
	case C.AKEYCODE_BACK:
		return 0x1B
	case C.AKEYCODE_ENTER, C.AKEYCODE_NUMPAD_ENTER:
		return 0x0D
	case C.AKEYCODE_TAB:
		return 0x09
	case C.AKEYCODE_DEL:
		return 0x08
	case C.AKEYCODE_FORWARD_DEL:
		return 0x2E
	case C.AKEYCODE_DPAD_LEFT:
		return 0x25
	case C.AKEYCODE_DPAD_UP:
		return 0x26
	case C.AKEYCODE_DPAD_RIGHT:
		return 0x27
	case C.AKEYCODE_DPAD_DOWN:
		return 0x28
	case C.AKEYCODE_SPACE:
		return 0x20
	}
	return 0
}

// ShowSoftInput 弹出软键盘（需 JNI 调 InputMethodManager，一期空实现）。
func ShowSoftInput() {}

// HideSoftInput 收起软键盘。
func HideSoftInput() {}

// String 调试描述。
func String() string {
	w, h := WindowSize()
	return fmt.Sprintf("android.Window{%dx%d px}", w, h)
}
