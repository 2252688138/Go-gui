//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

var (
	advapi32        = syscall.NewLazyDLL("advapi32.dll")
	procRegGetValue = advapi32.NewProc("RegGetValueW")
)

// platformPrefersDark 读取注册表 AppsUseLightTheme 判断系统深色模式。
func platformPrefersDark() bool {
	const (
		hkeyCurrentUser = 0x80000001
		rrfRtRegDword   = 0x00000010
	)
	key, err := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)
	if err != nil {
		return true
	}
	name, err := syscall.UTF16PtrFromString("AppsUseLightTheme")
	if err != nil {
		return true
	}
	var val uint32
	size := uint32(4)
	r, _, _ := procRegGetValue.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(name)),
		rrfRtRegDword,
		0,
		uintptr(unsafe.Pointer(&val)),
		uintptr(unsafe.Pointer(&size)),
	)
	if r != 0 {
		return true
	}
	return val == 0
}
