//go:build windows && cgo && gogui_cgo

package win32

// float 参数 COM 调用（cgo 版）：把「取虚表 + 把 float 放进 XMM」交给 C 编译器生成。
//
// 只 include <windows.h>，不链接 d3d11/dcomp 导入库——本文件只做「通过函数指针调用」，
// 对象与槽位都由 Go 侧（purego）创建，因此 cgo 版本对构建环境要求极低（MinGW 即可）。

/*
#include <windows.h>
#include <stdint.h>

typedef HRESULT (STDMETHODCALLTYPE *gogui_fn1f)(void *self, float a);
typedef HRESULT (STDMETHODCALLTYPE *gogui_fn2f)(void *self, float a, float b);
typedef HRESULT (STDMETHODCALLTYPE *gogui_fnf1p)(void *self, float a, void *p);
typedef HRESULT (STDMETHODCALLTYPE *gogui_fnp1f)(void *self, void *p, float a);
typedef HRESULT (STDMETHODCALLTYPE *gogui_fnp)(void *self, void *p);

static void **gogui_vtbl(void *obj) { return *(void ***)obj; }

HRESULT gogui_call_1f(void *obj, int slot, float a) {
    gogui_fn1f fn = (gogui_fn1f)gogui_vtbl(obj)[slot];
    if (!fn) return E_POINTER;
    return fn(obj, a);
}

HRESULT gogui_call_2f(void *obj, int slot, float a, float b) {
    gogui_fn2f fn = (gogui_fn2f)gogui_vtbl(obj)[slot];
    if (!fn) return E_POINTER;
    return fn(obj, a, b);
}

HRESULT gogui_call_f1p(void *obj, int slot, float a, void *p) {
    gogui_fnf1p fn = (gogui_fnf1p)gogui_vtbl(obj)[slot];
    if (!fn) return E_POINTER;
    return fn(obj, a, p);
}

HRESULT gogui_call_p1f(void *obj, int slot, void *p, float a) {
    gogui_fnp1f fn = (gogui_fnp1f)gogui_vtbl(obj)[slot];
    if (!fn) return E_POINTER;
    return fn(obj, p, a);
}

HRESULT gogui_call_p(void *obj, int slot, void *p) {
    gogui_fnp fn = (gogui_fnp)gogui_vtbl(obj)[slot];
    if (!fn) return E_POINTER;
    return fn(obj, p);
}
*/
import "C"

import "unsafe"

// CgoEnabled 是否以 cgo 构建（决定 WebView 走 A 还是 B 阶段）。
func CgoEnabled() bool { return true }

// ComCallF 调用 obj 的第 slot 个方法，参数为单个 float。
func ComCallF(obj uintptr, slot int, v float32) uint32 {
	if obj == 0 {
		return 0x80004003
	}
	return uint32(C.gogui_call_1f(unsafe.Pointer(obj), C.int(slot), C.float(v)))
}

// ComCallFF 调用 obj 的第 slot 个方法，参数为两个 float。
func ComCallFF(obj uintptr, slot int, a, b float32) uint32 {
	if obj == 0 {
		return 0x80004003
	}
	return uint32(C.gogui_call_2f(unsafe.Pointer(obj), C.int(slot), C.float(a), C.float(b)))
}

// ComCallF1P 调用 fn(self, float, void*)。
func ComCallF1P(obj uintptr, slot int, a float32, p uintptr) uint32 {
	if obj == 0 {
		return 0x80004003
	}
	return uint32(C.gogui_call_f1p(unsafe.Pointer(obj), C.int(slot), C.float(a), unsafe.Pointer(p)))
}

// ComCallP1F 调用 fn(self, void*, float)。
func ComCallP1F(obj uintptr, slot int, p uintptr, a float32) uint32 {
	if obj == 0 {
		return 0x80004003
	}
	return uint32(C.gogui_call_p1f(unsafe.Pointer(obj), C.int(slot), unsafe.Pointer(p), C.float(a)))
}
