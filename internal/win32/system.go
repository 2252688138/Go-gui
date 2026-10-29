//go:build windows

package win32

import (
	"syscall"
	"unsafe"
)

// 系统操作能力（窗口置顶 / 全局热键 / 阻止休眠 / 回收站 / 定位文件 / 深浅色）。
// 全部是对 user32 / kernel32 / shell32 / advapi32 的薄封装，不含 UI 依赖。

// 全局热键（RegisterHotKey）
var (
	RegisterHotKey   func(hwnd uintptr, id int32, mods uint32, vk uint32) int32
	UnregisterHotKey func(hwnd uintptr, id int32) int32
)

// 回收站删除（SHFileOperationW）
var SHFileOperationW func(op *SHFILEOPSTRUCTW) int32

// 阻止休眠 / 屏幕常亮（SetThreadExecutionState）
var SetThreadExecutionState func(flags uint32) uint32

// 注册表读取（系统深浅色）
var (
	RegOpenKeyExW    func(hkey uintptr, sub *uint16, opt, access uint32, phk *uintptr) int32
	RegQueryValueExW func(hkey uintptr, name *uint16, reserved *uint32, typ *uint32, data *byte, cb *uint32) int32
	RegCloseKey      func(hkey uintptr) int32
)

// ---- 窗口置顶 ----

// SetAlwaysOnTop 设置窗口是否置顶（HWND_TOPMOST / HWND_NOTOPMOST）。
func (w *Window) SetAlwaysOnTop(on bool) {
	if w == nil || w.Hwnd == 0 || SetWindowPos == nil {
		return
	}
	after := HWND_NOTOPMOST
	if on {
		after = HWND_TOPMOST
	}
	SetWindowPos(w.Hwnd, after, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
	w.topmost = on
}

// AlwaysOnTop 当前窗口是否置顶。
func (w *Window) AlwaysOnTop() bool { return w != nil && w.topmost }

// ---- 全局热键 ----

var (
	hotkeyMu     = map[uintptr]map[int32]func(){}
	hotkeyNextID int32 = 100
)

// RegisterHotKey 注册系统级热键（窗口失焦、最小化时同样触发）。
// mods 见 MOD_ALT / MOD_CONTROL / MOD_SHIFT / MOD_WIN，vk 为虚拟键码（如 'S'）。
// 返回 id，用于 UnregisterHotkey；成功后在消息循环中回调 fn。
func (w *Window) RegisterHotKey(mods uint32, vk uint32, fn func()) int32 {
	if w == nil || w.Hwnd == 0 || RegisterHotKey == nil {
		return 0
	}
	mu.Lock()
	m := hotkeyMu[w.Hwnd]
	if m == nil {
		m = map[int32]func(){}
		hotkeyMu[w.Hwnd] = m
	}
	mu.Unlock()
	for {
		id := hotkeyNextID
		hotkeyNextID++
		if hotkeyNextID > 0xBFFF {
			hotkeyNextID = 100
		}
		if _, used := m[id]; used {
			continue
		}
		if RegisterHotKey(w.Hwnd, id, mods|MOD_NOREPEAT, vk) == 0 {
			dbg("hotkey: 注册失败 mods=%#x vk=%#x err=%d", mods, vk, GetLastError())
			return 0
		}
		m[id] = fn
		dbg("hotkey: 注册成功 id=%d mods=%#x vk=%#x", id, mods, vk)
		return id
	}
}

// UnregisterHotkey 注销热键。
func (w *Window) UnregisterHotkey(id int32) {
	if w == nil || w.Hwnd == 0 || id == 0 {
		return
	}
	if UnregisterHotKey != nil {
		UnregisterHotKey(w.Hwnd, id)
	}
	mu.Lock()
	if m := hotkeyMu[w.Hwnd]; m != nil {
		delete(m, id)
	}
	mu.Unlock()
}

// dispatchHotkey 由 wndProc 调用：把 WM_HOTKEY 派发给对应回调。
func dispatchHotkey(hwnd uintptr, id uint32) {
	mu.Lock()
	fn := hotkeyMu[hwnd][int32(id)]
	mu.Unlock()
	if fn != nil {
		fn()
	}
}

// releaseHotkeys 窗口销毁时清空回调（热键本身随窗口销毁自动失效）。
func releaseHotkeys(hwnd uintptr) {
	mu.Lock()
	delete(hotkeyMu, hwnd)
	mu.Unlock()
}

// ---- 电源管理 ----

// KeepAwake 阻止系统休眠/关屏（on=false 恢复默认行为）。
// 典型场景：长任务、播放器、演示。返回调用前的标志位。
func KeepAwake(on bool) uint32 {
	if SetThreadExecutionState == nil {
		return 0
	}
	if on {
		return SetThreadExecutionState(ES_CONTINUOUS | ES_SYSTEM_REQUIRED | ES_DISPLAY_REQUIRED)
	}
	return SetThreadExecutionState(ES_CONTINUOUS)
}

// ---- 资源管理器 / 回收站 ----

// RevealInExplorer 在资源管理器中定位并选中指定文件/文件夹。
func RevealInExplorer(path string) bool {
	if ShellExecuteW == nil || path == "" {
		return false
	}
	file, _ := syscall.UTF16PtrFromString("explorer.exe")
	op, _ := syscall.UTF16PtrFromString("open")
	params, _ := syscall.UTF16PtrFromString("/select,\"" + path + "\"")
	return ShellExecuteW(0, op, file, params, nil, SW_SHOW) > 32
}

// MoveToRecycleBin 把文件/文件夹移入回收站（FOF_ALLOWUNDO，静默无确认框）。
func MoveToRecycleBin(path string) bool {
	if SHFileOperationW == nil || path == "" {
		return false
	}
	// pFrom 必须以双 NUL 结尾
	from := append(syscall.StringToUTF16(path), 0)
	op := SHFILEOPSTRUCTW{
		WFunc:  FO_DELETE,
		PFrom:  &from[0],
		FFlags: FOF_ALLOWUNDO | FOF_NOCONFIRMATION | FOF_SILENT | FOF_NOERRORUI,
	}
	return SHFileOperationW(&op) == 0
}

// ---- 系统外观 ----

// SystemDarkMode 读取注册表判断系统是否使用深色模式（AppsUseLightTheme=0）。
// 读取失败时返回 true（与框架默认深色主题一致）。
func SystemDarkMode() bool {
	if RegOpenKeyExW == nil || RegQueryValueExW == nil {
		return true
	}
	sub, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)
	var key uintptr
	if RegOpenKeyExW(HKEY_CURRENT_USER, sub, 0, KEY_READ, &key) != 0 {
		return true
	}
	defer RegCloseKey(key)
	name, _ := syscall.UTF16PtrFromString("AppsUseLightTheme")
	var typ uint32
	var buf [4]byte
	cb := uint32(len(buf))
	if RegQueryValueExW(key, name, nil, &typ, &buf[0], &cb) != 0 {
		return true
	}
	return *(*uint32)(unsafe.Pointer(&buf[0])) == 0
}

// ---- 剪贴板：文件列表 ----

// DROPFILES CF_HDROP 头部（pFiles / pt / fNC / fWide，共 20 字节）。
type dropFiles struct {
	PFiles uint32
	Pt     POINT
	FNC    int32
	FWide  int32
}

// SetClipboardFiles 把文件列表写入剪贴板（CF_HDROP），可直接粘贴到资源管理器。
func SetClipboardFiles(paths []string) bool {
	if OpenClipboard == nil || OpenClipboard(0) == 0 {
		return false
	}
	defer CloseClipboard()
	EmptyClipboard()

	var list []uint16
	for _, p := range paths {
		if p == "" {
			continue
		}
		list = append(list, syscall.StringToUTF16(p)...)
	}
	if len(list) == 0 {
		return false
	}
	list = append(list, 0) // 结尾双 NUL

	hdr := uintptr(unsafe.Sizeof(dropFiles{}))
	// 头部需 20 字节对齐到 4 的倍数，路径紧跟在后面
	total := int(hdr) + len(list)*2
	h := GlobalAlloc(GMEMoveable, uintptr(total))
	if h == 0 {
		return false
	}
	p := GlobalLock(h)
	if p == 0 {
		GlobalFree(h)
		return false
	}
	df := (*dropFiles)(unsafe.Pointer(p))
	df.PFiles = uint32(hdr)
	df.FWide = 1
	// fNC = 0：pt 为客户区坐标
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p+hdr)), len(list))
	copy(dst, list)
	GlobalUnlock(h)
	if SetClipboardData(CF_HDROP, h) == 0 {
		GlobalFree(h)
		return false
	}
	return true
}
