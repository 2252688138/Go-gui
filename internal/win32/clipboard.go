//go:build windows

package win32

import (
	"syscall"
	"unsafe"
)

// 剪贴板（CF_UNICODETEXT）。绑定在 api.go 的 init 中完成。
var (
	OpenClipboard              func(hwnd uintptr) int32
	CloseClipboard             func() int32
	EmptyClipboard             func() int32
	SetClipboardData           func(format uint32, mem uintptr) uintptr
	GetClipboardData           func(format uint32) uintptr
	IsClipboardFormatAvailable func(format uint32) int32

	GlobalAlloc  func(flags uint32, bytes uintptr) uintptr
	GlobalLock   func(mem uintptr) uintptr
	GlobalUnlock func(mem uintptr) int32
	GlobalFree   func(mem uintptr) uintptr
)

const (
	// CFUnicodeText 剪贴板 Unicode 文本格式。
	CFUnicodeText = 13
	// GMEMoveable 可移动全局内存（剪贴板要求）。
	GMEMoveable = 0x0002
)

// SetClipboardText 把文本写入系统剪贴板（UTF-16）。
func SetClipboardText(s string) bool {
	if OpenClipboard == nil || SetClipboardData == nil {
		return false
	}
	if OpenClipboard(0) == 0 {
		return false
	}
	defer CloseClipboard()
	EmptyClipboard()

	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return false
	}
	h := GlobalAlloc(GMEMoveable, uintptr(len(u)*2))
	if h == 0 {
		return false
	}
	p := GlobalLock(h)
	if p == 0 {
		GlobalFree(h)
		return false
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u))
	copy(dst, u)
	GlobalUnlock(h)
	if SetClipboardData(CFUnicodeText, h) == 0 {
		GlobalFree(h)
		return false
	}
	return true
}

// GetClipboardText 读取剪贴板中的 Unicode 文本（失败返回空串）。
func GetClipboardText() string {
	if OpenClipboard == nil || GetClipboardData == nil {
		return ""
	}
	if OpenClipboard(0) == 0 {
		return ""
	}
	defer CloseClipboard()
	if IsClipboardFormatAvailable != nil && IsClipboardFormatAvailable(CFUnicodeText) == 0 {
		return ""
	}
	h := GetClipboardData(CFUnicodeText)
	if h == 0 {
		return ""
	}
	p := GlobalLock(h)
	if p == 0 {
		return ""
	}
	defer GlobalUnlock(h)
	// 逐字扫描到 NUL，避免依赖 GlobalSize
	ptr := (*uint16)(unsafe.Pointer(p))
	n := 0
	for n < 1<<20 {
		if *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(n)*2)) == 0 {
			break
		}
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(ptr, n))
}
