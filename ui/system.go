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
)

// 系统能力（平台无关部分）：文件拖入区域、路径展示、置顶、热键、拖入回调等
// 纯逻辑组件放在这里；真正落到系统 API 的部分（剪贴板、文件对话框、托盘、
// 休眠抑制、回收站、深色模式检测）在 system_win32.go / system_unix.go / platform_*.go。

// ---- 文件拖入（DROP）----
//
// 链路：窗口层注册接收拖入 → 系统在松手时投递事件 → App 在下一帧用落点命中
// 节点树 → 最近的设置了 OnDrop 的节点优先，否则交给应用级回调。

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
// 数值与 Win32 的 MOD_* 一致，因此同一份配置在所有平台含义相同。
const (
	HotAlt      = 0x0001
	HotCtrl     = 0x0002
	HotShift    = 0x0004
	HotWin      = 0x0008
	HotNoRepeat = 0x4000
)

// RegisterHotkey 注册系统级热键（窗口最小化/失焦时同样触发）。
// vk 为虚拟键码，字母键直接用 'S' 这样的字符。返回 id（0 表示注册失败，
// 通常是该组合已被其它程序占用，或当前平台不支持）。
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

// 小工具：把路径列表压成一行展示（超出 n 个用 … 省略）。
func joinPaths(paths []string, n int) string {
	if len(paths) <= n {
		return strings.Join(paths, "，")
	}
	return strings.Join(paths[:n], "，") + " 等 " + strconv.Itoa(len(paths)) + " 项"
}
