//go:build windows && amd64 && !(cgo && gogui_cgo)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"sync"
	"syscall"
	"unsafe"
)

// float 参数 COM 调用：用运行时生成的机器码 trampoline 把 float 送进 XMM0/XMM1。
//
// 为什么不用 Go 汇编：手写 Go 汇编函数要走 ABI0 ↔ ABIInternal 转换，且外部
// Win64 约定下「CALL 前 RSP 需 16 字节对齐 + 参数读取必须在改 SP 之前」，
// 这两个坑实测都会导致崩溃。这里直接生成一段机器码，用 syscall.SyscallN 调用：
//
//	tramp1(RCX=fn, RDX=self, R8=floatBits)：
//	    mov r11, rcx        ; 49 89 CB
//	    mov rcx, rdx        ; 48 89 D1
//	    movd xmm0, r8d      ; 66 41 0F 6E C0
//	    sub rsp, 0x28       ; 48 83 EC 28
//	    call r11            ; 41 FF D3
//	    add rsp, 0x28       ; 48 83 C4 28
//	    ret                 ; C3
var (
	trampOnce sync.Once
	tramp1    uintptr // (fn, self, bitsA) -> HRESULT
	tramp2    uintptr // (fn, self, bitsA, bitsB) -> HRESULT
	trampErr  error
)

func ensureTramps() error {
	trampOnce.Do(func() {
		k32 := syscall.NewLazyDLL("kernel32.dll")
		alloc := k32.NewProc("VirtualAlloc")
		const memCommitReserve = 0x3000
		const pageExecuteReadWrite = 0x40

		code1 := []byte{
			0x49, 0x89, 0xCB, // mov r11, rcx
			0x48, 0x89, 0xD1, // mov rcx, rdx
			0x66, 0x41, 0x0F, 0x6E, 0xC0, // movd xmm0, r8d
			0x48, 0x83, 0xEC, 0x28, // sub rsp, 0x28
			0x41, 0xFF, 0xD3, // call r11
			0x48, 0x83, 0xC4, 0x28, // add rsp, 0x28
			0xC3, // ret
		}
		code2 := []byte{
			0x49, 0x89, 0xCB, // mov r11, rcx
			0x48, 0x89, 0xD1, // mov rcx, rdx
			0x66, 0x41, 0x0F, 0x6E, 0xC0, // movd xmm0, r8d
			0x66, 0x41, 0x0F, 0x6E, 0xC9, // movd xmm1, r9d
			0x48, 0x83, 0xEC, 0x28, // sub rsp, 0x28
			0x41, 0xFF, 0xD3, // call r11
			0x48, 0x83, 0xC4, 0x28, // add rsp, 0x28
			0xC3,
		}
		p1, _, _ := alloc.Call(0, 64, memCommitReserve, pageExecuteReadWrite)
		p2, _, _ := alloc.Call(0, 96, memCommitReserve, pageExecuteReadWrite)
		if p1 == 0 || p2 == 0 {
			trampErr = syscall.EINVAL
			return
		}
		copy(unsafe.Slice((*byte)(unsafe.Pointer(p1)), len(code1)), code1)
		copy(unsafe.Slice((*byte)(unsafe.Pointer(p2)), len(code2)), code2)
		tramp1, tramp2 = p1, p2
	})
	return trampErr
}

// ComCallF 调用 fn(self, float)，返回 HRESULT。
func ComCallF(obj uintptr, slot int, v float32) uint32 {
	if ensureTramps() != nil {
		return 0x80004005
	}
	fn := ComSlot(obj, slot)
	if fn == 0 {
		return 0x80004003
	}
	r, _, _ := syscall.SyscallN(tramp1, fn, obj, uintptr(float32Bits(v)))
	return uint32(r)
}

// ComCallFF 调用 fn(self, float, float)，返回 HRESULT。
func ComCallFF(obj uintptr, slot int, a, b float32) uint32 {
	if ensureTramps() != nil {
		return 0x80004005
	}
	fn := ComSlot(obj, slot)
	if fn == 0 {
		return 0x80004003
	}
	r, _, _ := syscall.SyscallN(tramp2, fn, obj, uintptr(float32Bits(a)), uintptr(float32Bits(b)))
	return uint32(r)
}

func float32Bits(f float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&f))
}

// ComCallF1P / ComCallP1F 纯 Go 下不实现（B 阶段要求 cgo）。
func ComCallF1P(obj uintptr, slot int, a float32, p uintptr) uint32 { return 0x80004005 }

func ComCallP1F(obj uintptr, slot int, p uintptr, a float32) uint32 { return 0x80004005 }
