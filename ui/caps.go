// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 平台能力查询：同一套 UI 代码要在 Windows / Linux / macOS / Web / 移动端编译运行，
// 各端能力必然不齐（内嵌浏览器、托盘、媒体解码、全局热键…）。
//
// 约定：**符号永不缺失**（没有实现就给空实现），能力用 HasCap 查询，
// 业务代码据此决定是否显示入口或走降级分支：
//
//	if ui.HasCap(ui.CapVideo) {
//		p, _ := ui.NewMediaPlayer(path)
//		// …
//	} else {
//		// 显示占位卡片
//	}
type Cap int

const (
	// CapVideo 视频解码与绘制。
	CapVideo Cap = iota
	// CapAudio 音频播放。
	CapAudio
	// CapWebView 内嵌浏览器（系统 WebView）。
	CapWebView
	// CapTray 系统托盘图标与菜单。
	CapTray
	// CapClipboard 剪贴板读写。
	CapClipboard
	// CapDialogs 文件/文件夹选择对话框。
	CapDialogs
	// CapHotkey 全局热键。
	CapHotkey
	// CapDragDrop 文件拖入窗口。
	CapDragDrop
	// CapCompose 窗口级 GPU 合成（如 Windows DirectComposition）。
	CapCompose
	// CapSystemDark 读取系统深色模式偏好。
	CapSystemDark
	// CapKeepAwake 阻止系统休眠/关屏。
	CapKeepAwake
	// CapRecycleBin 移入回收站。
	CapRecycleBin
	// CapTouch 触摸/手势输入。
	CapTouch
	// CapPrint 系统打印对话框与 GDI 输出。
	CapPrint
	// CapScreenCapture 屏幕截图 / 吸管取色。
	CapScreenCapture
	// CapClipboardWatch 剪贴板变化监听。
	CapClipboardWatch
	// CapMultiMonitor 多显示器枚举与工作区。
	CapMultiMonitor
	// CapSingleInstance 单实例锁 + 二次启动参数。
	CapSingleInstance
	// CapModernNotify 应用内通知中心（按钮/进度）。
	CapModernNotify
)

// CapLast 最后一个能力值（遍历 / 测试用；新增能力时同步更新）。
const CapLast = CapModernNotify

// capNames 能力名称（调试与界面展示用）。
var capNames = map[Cap]string{
	CapVideo:          "视频解码",
	CapAudio:          "音频播放",
	CapWebView:        "内嵌浏览器",
	CapTray:           "系统托盘",
	CapClipboard:      "剪贴板",
	CapDialogs:        "文件对话框",
	CapHotkey:         "全局热键",
	CapDragDrop:       "文件拖入",
	CapCompose:        "GPU 合成",
	CapSystemDark:     "深色模式检测",
	CapKeepAwake:      "禁止休眠",
	CapRecycleBin:     "回收站",
	CapTouch:          "触摸输入",
	CapPrint:          "打印",
	CapScreenCapture:  "屏幕捕获",
	CapClipboardWatch: "剪贴板监听",
	CapMultiMonitor:   "多显示器",
	CapSingleInstance: "单实例锁",
	CapModernNotify:   "通知中心",
}

// CapName 返回能力的中文名称。
func CapName(c Cap) string {
	if n, ok := capNames[c]; ok {
		return n
	}
	return "未知能力"
}

// Capabilities 当前平台支持的能力列表（由各平台文件提供）。
func Capabilities() []Cap { return platformCaps() }

// HasCap 当前平台是否支持某能力。
func HasCap(c Cap) bool {
	for _, v := range platformCaps() {
		if v == c {
			return true
		}
	}
	return false
}

// UnsupportedCaps 当前平台缺失的能力（便于界面上统一提示）。
func UnsupportedCaps() []Cap {
	var out []Cap
	for c := CapVideo; c <= CapLast; c++ {
		if !HasCap(c) {
			out = append(out, c)
		}
	}
	return out
}
