//go:build windows && amd64

package win32

import "unsafe"

// Win64 调用约定：float/double 走 XMM0-3，整数/指针走 RCX/RDX/R8/R9。
// 带 float 参数的 COM 方法（DirectComposition 的 SetOffsetX/SetLeft/圆角半径…）
// 由 com_float_amd64.go 里的机器码 trampoline 调用；
// 本文件只保留虚表寻址与整数参数调用。

// ComSlot 取 COM 对象指定虚表槽位的函数地址（0=QueryInterface）。
func ComSlot(obj uintptr, slot int) uintptr {
	if obj == 0 {
		return 0
	}
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	return *(*uintptr)(unsafe.Pointer(vtbl + uintptr(slot)*unsafe.Sizeof(uintptr(0))))
}
