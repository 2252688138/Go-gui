//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"syscall"
	"unsafe"
)

// SPI_GETCLIENTAREAANIMATION：系统「在 Windows 中显示动画」开关。
const spiGetClientAreaAnimation = 0x1042

var procSystemParametersInfoW = syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW")

// platformReducedMotionImpl 读取系统「动画效果」设置（关闭动画 = 减弱动态效果）。
func platformReducedMotionImpl() bool {
	if err := procSystemParametersInfoW.Find(); err != nil {
		return false
	}
	var enabled int32 = 1
	r, _, _ := procSystemParametersInfoW.Call(
		spiGetClientAreaAnimation,
		0,
		uintptr(unsafe.Pointer(&enabled)),
		0,
	)
	if r == 0 {
		return false
	}
	return enabled == 0
}
