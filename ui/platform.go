//go:build windows

package ui

import "gogui/internal/win32"

// ---- 文件对话框 ----

// OpenFileDialog 打开文件对话框。filter 形如 "图片|*.png;*.jpg|所有文件|*.*"。
func OpenFileDialog(title, filter string) (string, bool) {
	var hwnd uintptr
	if a := currentApp.Load(); a != nil {
		hwnd = a.HWND()
	}
	return win32.OpenFile(hwnd, title, filter)
}

// SaveFileDialog 保存文件对话框。
func SaveFileDialog(title, filter, defExt string) (string, bool) {
	var hwnd uintptr
	if a := currentApp.Load(); a != nil {
		hwnd = a.HWND()
	}
	return win32.SaveFile(hwnd, title, filter, defExt)
}

// PickFolderDialog 选择文件夹。
func PickFolderDialog(title string) (string, bool) {
	var hwnd uintptr
	if a := currentApp.Load(); a != nil {
		hwnd = a.HWND()
	}
	return win32.PickFolder(hwnd, title)
}

// ShellOpen 用系统默认程序打开路径/网址。
func ShellOpen(path string) { win32.ShellOpen(path) }

// ---- 托盘 ----

// TrayItem 托盘菜单项（Separator 为分隔线）。
type TrayItem = win32.TrayItem

// Tray 任务栏托盘图标（依附当前窗口的消息循环）。
type Tray struct {
	t *win32.Tray
}

// NewTray 创建托盘图标：onClick 为左键点击，items/onMenu 为右键菜单。
func NewTray(tooltip string, onClick func(), items []TrayItem, onMenu func(id int)) (*Tray, error) {
	var hwnd uintptr
	if a := currentApp.Load(); a != nil {
		hwnd = a.HWND()
	}
	t, err := win32.NewTray(hwnd, 1, tooltip, onClick, items, onMenu)
	if err != nil {
		return nil, err
	}
	return &Tray{t: t}, nil
}

// SetTooltip 更新托盘提示。
func (t *Tray) SetTooltip(s string) {
	if t != nil && t.t != nil {
		t.t.SetTooltip(s)
	}
}

// SetItems 更新右键菜单。
func (t *Tray) SetItems(items []TrayItem) {
	if t != nil && t.t != nil {
		t.t.SetItems(items)
	}
}

// Remove 移除托盘图标。
func (t *Tray) Remove() {
	if t != nil && t.t != nil {
		t.t.Remove()
	}
}

// ---- 窗口 ----

// Current 返回当前活跃窗口（无则 nil）。
func Current() *App { return currentApp.Load() }

// NewWindow 打开新窗口（须在 UI 线程/回调中调用）。
func NewWindow(opts Options, view func() *Node) (*App, error) {
	return OpenWindow(opts, view)
}
