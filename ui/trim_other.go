//go:build !windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// trimProcessMemory 非 Windows 平台暂无等价的「归还工作集」调用：
// Linux/Android 由内核按内存压力自动回收，macOS 由系统内存压缩处理，
// Web 端内存归浏览器管理，因此这里是空实现（调用点无需平台分支）。
func trimProcessMemory() {}

// windowInForeground 非 Windows 平台返回 true：不影响逻辑，
// 因为 trimProcessMemory 在本平台本身是空实现。
func windowInForeground(hwnd uintptr) bool { return true }
