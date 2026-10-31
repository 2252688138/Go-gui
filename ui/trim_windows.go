//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "gogui/internal/win32"

// trimProcessMemory 把本进程可回收的物理页还给系统（空闲 / 最小化时调用）。
// Windows 端让系统自行决定工作集大小；其它平台是空实现（见 trim_other.go）。
func trimProcessMemory() { win32.TrimWorkingSet() }

// windowInForeground 窗口是否处于前台。
// 二级（工作集级）回收只在切到其它程序或最小化后做：
// 用户正看着窗口时保持页面常驻，避免重新调页带来的顿挫。
func windowInForeground(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	fg := win32.GetForegroundWindow
	if fg == nil {
		return true
	}
	return fg() == hwnd
}
