//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "gogui/internal/win32"

// Windows 端的系统能力实现（system.go 里是与平台无关的组件部分）。

// PickFiles 打开多选文件对话框（filter 形如 "图片|*.png;*.jpg|所有文件|*.*"）。
func PickFiles(title, filter string) ([]string, bool) {
	var hwnd uintptr
	if a := currentApp.Load(); a != nil {
		hwnd = a.HWND()
	}
	return win32.OpenFiles(hwnd, title, filter)
}

// ---- 剪贴板 ----

// ClipboardText 读取剪贴板文本（无文本返回空串）。
func ClipboardText() string { return win32.GetClipboardText() }

// ClipboardSetText 写入剪贴板文本。
func ClipboardSetText(s string) bool { return win32.SetClipboardText(s) }

// ClipboardSetFiles 把文件路径列表写入剪贴板（CF_HDROP，可直接粘贴到资源管理器）。
func ClipboardSetFiles(paths []string) bool { return win32.SetClipboardFiles(paths) }

// ---- 电源 / 资源管理器 ----

// SetKeepAwake 阻止系统休眠与关屏（false = 恢复默认）。返回调用前的标志位。
func SetKeepAwake(on bool) uint32 { return win32.KeepAwake(on) }

// RevealInExplorer 在资源管理器中定位并选中给定路径。
func RevealInExplorer(path string) bool { return win32.RevealInExplorer(path) }

// MoveToRecycleBin 把文件/文件夹移入回收站（不弹确认框，可从回收站还原）。
func MoveToRecycleBin(path string) bool { return win32.MoveToRecycleBin(path) }

// SystemDarkMode 系统是否处于深色模式（读注册表 AppsUseLightTheme）。
func SystemDarkMode() bool { return win32.SystemDarkMode() }

// Notify 在托盘图标上弹出气泡提示（title 可空）。
func (t *Tray) Notify(title, msg string) {
	if t == nil || t.t == nil {
		return
	}
	t.t.Notify(title, msg)
}
