//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gogui/internal/win32"
)

// ---- 文件拖入（DROP）----
//
// 链路：窗口 DragAcceptFiles 注册 → 系统在松手时投递 WM_DROPFILES →
// win32 取出 HDROP 里的路径与落点 → App 在下一帧用落点命中节点树 →
// 最近的设置了 OnDrop 的节点优先，否则交给应用级回调。
//
// 说明：WM_DROPFILES 只在「松手」时到达，没有 dragenter/dragover 过程，
// 因此不存在悬停高亮；需要悬停高亮得改用 OLE IDropTarget，本框架未提供。

// SetOnDropFiles 给当前所有窗口设置应用级文件拖入回调。
// x/y 为落下点（逻辑像素，客户区坐标）；拖到没有 OnDrop 的区域时才触发。
func SetOnDropFiles(fn func(paths []string, x, y float32)) {
	for _, a := range snapshotApps() {
		a.onDrop = fn
	}
}

// SetAcceptDrops 开启/关闭当前窗口接收文件拖入（默认开启）。
func SetAcceptDrops(on bool) {
	if a := currentApp.Load(); a != nil {
		a.SetAcceptDrops(on)
	}
}

// DropZone 文件拖入区域：把文件/文件夹拖进框内松手即触发 onDrop。
// height <= 0 时用默认 132。返回的节点可继续 .St()/.K() 调整外观。
func DropZone(key, hint string, height float32, onDrop func(paths []string)) *Node {
	if height <= 0 {
		height = 132
	}
	if hint == "" {
		hint = "把文件拖到这里"
	}
	icon := Stack(
		Box().St(St().W(46).H(46).R(14).BgC(ColHover).Stroke(1, ColLine)),
		Text("↓").St(St().AbsFill().Text(20, 600, ColPlaceholder).CenterYOn().TAlign(AlignCenter)),
	).St(St().W(46).H(46))

	body := Column(
		icon,
		Text(hint).St(St().Text(12.5, 500, ColTextDim).TAlign(AlignCenter)),
		Text("拖入文件或文件夹 · 支持多选").St(St().Text(11, 400, ColPlaceholder).TAlign(AlignCenter)),
	)
	body.Align = AlignCenter
	body.Justify = AlignCenter
	body.OnDrop = onDrop
	body.K(key)
	return body.St(St().GapOf(8).Padding(18).H(height).R(14).
		BgC(ColBgSoft).Stroke(1, ColStroke))
}

// FormatBytes 把字节数格式化成 1.2 MB 这样的可读文本。
func FormatBytes(n int64) string {
	f := float64(n)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return strconv.FormatInt(n, 10) + " B"
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

// FileRow 拖入结果的一行：文件名 + 大小 + 所在目录。
// onClick 为 nil 时默认用系统默认程序打开。
func FileRow(path string, onClick func()) *Node {
	name := filepath.Base(path)
	info, err := os.Stat(path)
	size, isDir := "", false
	if err == nil {
		if info.IsDir() {
			isDir = true
			size = "文件夹"
		} else {
			size = FormatBytes(info.Size())
		}
	} else {
		size = "不可访问"
	}
	dir := filepath.Dir(path)
	glyph := "·"
	if isDir {
		glyph = "▸"
	}
	open := onClick
	if open == nil {
		p := path
		open = func() { ShellOpen(p) }
	}
	n := Row(
		Box().St(St().W(26).H(26).R(8).BgC(ColHover).Stroke(1, ColLine)).Add(
			Text(glyph).St(St().AbsFill().Text(12, 600, ColPrimary).CenterYOn().TAlign(AlignCenter)),
		),
		Column(
			Text(name).St(St().Text(12.5, 500, ColText)),
			Text(dir).St(St().Text(10.5, 400, ColPlaceholder)),
		).St(St().GrowTo(1).GapOf(2)),
		Text(size).St(St().Text(11, 500, ColTextMuted)),
	).St(St().GapOf(10).PadXY(10, 7).R(9).BgC(ColBg))
	n.OnClick = func() {
		open()
		RequestRedraw()
	}
	return n
}

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

// ---- 窗口 ----

// SetAlwaysOnTop 窗口置顶 / 取消置顶。
func SetAlwaysOnTop(on bool) {
	if a := currentApp.Load(); a != nil {
		a.SetAlwaysOnTop(on)
	}
}

// IsAlwaysOnTop 当前窗口是否置顶。
func IsAlwaysOnTop() bool {
	if a := currentApp.Load(); a != nil {
		return a.IsAlwaysOnTop()
	}
	return false
}

// 全局热键修饰键（RegisterHotkey 的 mods 参数）。
const (
	HotAlt      = win32.MOD_ALT
	HotCtrl     = win32.MOD_CONTROL
	HotShift    = win32.MOD_SHIFT
	HotWin      = win32.MOD_WIN
	HotNoRepeat = win32.MOD_NOREPEAT
)

// RegisterHotkey 注册系统级热键（窗口最小化/失焦时同样触发）。
// vk 为虚拟键码，字母键直接用 'S' 这样的字符。返回 id（0 表示注册失败，
// 通常是该组合已被其它程序占用）。
func RegisterHotkey(mods uint32, vk uint32, fn func()) int32 {
	if a := currentApp.Load(); a != nil {
		return a.RegisterHotkey(mods, vk, fn)
	}
	return 0
}

// UnregisterHotkey 注销热键。
func UnregisterHotkey(id int32) {
	if a := currentApp.Load(); a != nil {
		a.UnregisterHotkey(id)
	}
}

// ---- 电源 / 资源管理器 ----

// SetKeepAwake 阻止系统休眠与关屏（false = 恢复默认）。返回调用前的标志位。
func SetKeepAwake(on bool) uint32 { return win32.KeepAwake(on) }

// RevealInExplorer 在资源管理器中定位并选中给定路径。
func RevealInExplorer(path string) bool { return win32.RevealInExplorer(path) }

// MoveToRecycleBin 把文件/文件夹移入回收站（不弹确认框，可从回收站还原）。
func MoveToRecycleBin(path string) bool { return win32.MoveToRecycleBin(path) }

// SystemDarkMode 系统是否处于深色模式（读注册表 AppsUseLightTheme）。
func SystemDarkMode() bool { return win32.SystemDarkMode() }

// ---- 托盘通知 ----

// Notify 在托盘图标上弹出气泡提示（title 可空）。
func (t *Tray) Notify(title, msg string) {
	if t == nil || t.t == nil {
		return
	}
	t.t.Notify(title, msg)
}

// 小工具：把路径列表压成一行展示（超出 n 个用 … 省略）。
func joinPaths(paths []string, n int) string {
	if len(paths) <= n {
		return strings.Join(paths, "，")
	}
	return strings.Join(paths[:n], "，") + " 等 " + strconv.Itoa(len(paths)) + " 项"
}
