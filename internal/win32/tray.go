//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"syscall"
	"unsafe"
)

// TrayItem 托盘右键菜单项。
type TrayItem struct {
	ID        int
	Label     string
	Separator bool
	Checked   bool
	Disabled  bool
	// Submenu 非空时生成二级菜单。
	Submenu []TrayItem
}

// Tray 任务栏通知区图标。
type Tray struct {
	Hwnd uintptr
	ID   uint32

	Tooltip string
	OnClick func()

	items  []TrayItem
	onMenu func(id int)

	added bool
}

// NewTray 在指定窗口的消息循环上创建托盘图标。ok 回调可选。
func NewTray(hwnd uintptr, id uint32, tooltip string, onClick func(), items []TrayItem, onMenu func(id int)) (*Tray, error) {
	t := &Tray{Hwnd: hwnd, ID: id, Tooltip: tooltip, OnClick: onClick, items: items, onMenu: onMenu}
	t.attach(hwnd)

	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.Hwnd = hwnd
	nid.UID = id
	nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP | NIF_SHOWTIP
	nid.UCallbackMessage = WM_TRAYICON
	nid.HIcon = LoadIconW(0, IDI_APPLICATION)
	copy(nid.SzTip[:], syscall.StringToUTF16(tooltip))
	if Shell_NotifyIconW(NIM_ADD, &nid) == 0 {
		return nil, syscall.EINVAL
	}
	// 使用 V4 行为（鼠标消息落在 lParam 低位）
	nid.UVersion = NOTIFYICON_VERSION_4
	Shell_NotifyIconW(NIM_SETVERSION, &nid)
	t.added = true
	return t, nil
}

// attach 把托盘挂到窗口，供 wndProc 派发消息。
func (t *Tray) attach(hwnd uintptr) {
	mu.Lock()
	traysByHwnd[hwnd] = t
	mu.Unlock()
}

// SetTooltip 更新提示文字。
func (t *Tray) SetTooltip(s string) {
	t.Tooltip = s
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.Hwnd = t.Hwnd
	nid.UID = t.ID
	nid.UFlags = NIF_TIP | NIF_SHOWTIP
	copy(nid.SzTip[:], syscall.StringToUTF16(s))
	Shell_NotifyIconW(NIM_MODIFY, &nid)
}

// SetItems 更新右键菜单项。
func (t *Tray) SetItems(items []TrayItem) { t.items = items }

// Notify 在托盘图标上弹出气泡提示（title 可空，msg 为正文）。
func (t *Tray) Notify(title, msg string) {
	if t == nil || !t.added {
		return
	}
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.Hwnd = t.Hwnd
	nid.UID = t.ID
	nid.UFlags = NIF_INFO
	nid.DwInfoFlags = NIIF_INFO
	copy(nid.SzInfo[:], syscall.StringToUTF16(msg))
	if title != "" {
		copy(nid.SzInfoTitle[:], syscall.StringToUTF16(title))
	}
	Shell_NotifyIconW(NIM_MODIFY, &nid)
}

// Remove 移除托盘图标。
func (t *Tray) Remove() {
	if !t.added {
		return
	}
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.Hwnd = t.Hwnd
	nid.UID = t.ID
	Shell_NotifyIconW(NIM_DELETE, &nid)
	t.added = false
	mu.Lock()
	delete(traysByHwnd, t.Hwnd)
	mu.Unlock()
}

// handleTray 处理 WM_TRAYICON（在 wndProc 中调用），返回是否已消费。
func handleTray(hwnd uintptr, lp uintptr) bool {
	mu.Lock()
	t := traysByHwnd[hwnd]
	mu.Unlock()
	if t == nil {
		return false
	}
	switch uint32(lp & 0xFFFF) {
	case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
		if t.OnClick != nil {
			t.OnClick()
		}
	case WM_RBUTTONUP:
		t.popup()
	}
	return true
}

// popup 弹出右键菜单并回调。
func (t *Tray) popup() {
	if len(t.items) == 0 {
		return
	}
	menu := CreatePopupMenu()
	if menu == 0 {
		return
	}
	defer DestroyMenu(menu)
	buildMenu(menu, t.items)

	var pt POINT
	GetCursorPos(&pt)
	SetForegroundWindow(t.Hwnd)
	const cmd = TPM_RETURNCMD | TPM_RIGHTBUTTON | TPM_NONOTIFY
	id := TrackPopupMenu(menu, cmd, pt.X, pt.Y, 0, t.Hwnd, 0)
	PostMessageW(t.Hwnd, WM_NULL, 0, 0)
	if id != 0 && t.onMenu != nil {
		t.onMenu(int(id))
	}
}

func buildMenu(menu uintptr, items []TrayItem) {
	for _, it := range items {
		if it.Separator {
			AppendMenuW(menu, MF_SEPARATOR, 0, nil)
			continue
		}
		flags := uint32(MF_STRING)
		if it.Checked {
			flags |= MF_CHECKED
		}
		if it.Disabled {
			flags |= MF_GRAYED
		}
		label, _ := syscall.UTF16PtrFromString(it.Label)
		if len(it.Submenu) > 0 {
			sub := CreatePopupMenu()
			buildMenu(sub, it.Submenu)
			AppendMenuW(menu, flags|MF_POPUP, sub, label)
			continue
		}
		AppendMenuW(menu, flags, uintptr(it.ID), label)
	}
}
