//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"strings"
	"syscall/js"
	"time"

	"gogui/internal/geom"
	"gogui/internal/text"
)

// RegisterFont 注册一份字体数据（浏览器端唯一字体来源，如 fetch 回来的 TTF）。
func RegisterFont(name string, data []byte) { text.RegisterFont(name, data) }

// RegisterFontBold 注册粗体字重（以 "<name>/Bold" 命名，字体缓存按字重优先取它）。
func RegisterFontBold(name string, data []byte) { text.RegisterFont(name+"/Bold", data) }

// errUnsupportedMedia 浏览器端不支持的解码能力。
var errUnsupportedMedia = errors.New("浏览器端不支持媒体解码")

// 浏览器端平台能力降级：桌面专属功能（WebView2、托盘、系统对话框、媒体解码、
// 内存统计等）在 Web 里没有对应实现，这里给出同名空实现，保证同一套业务代码
// 与组件树可以原样编译运行。

// WebView 内嵌浏览器（浏览器端原生就是网页，无需子窗口）。
type WebView struct {
	url     string
	visible bool
}

func (w *WebView) SetURL(u string)  { w.url = u }
func (w *WebView) URL() string      { return w.url }
func (w *WebView) SetVisible(v bool) { w.visible = v }
func (w *WebView) Visible() bool    { return w.visible }
func (w *WebView) Close()           {}

// syncWebViews 桌面端同步子窗口位置；浏览器端为空操作。
func (a *App) syncWebViews() {}

// closeWebViews 释放内嵌浏览器。
func (a *App) closeWebViews() {
	a.webviews = nil
	a.wvRects = nil
	a.wvShown = nil
}

// logMemStats 内存统计（浏览器端不启用）。
func (a *App) logMemStats() {}

// ResumeMediaAll 最小化恢复后重建解码器（浏览器端无解码器）。
func ResumeMediaAll() {}

// resumeMediaAll 供 App.step 内部调用。
func resumeMediaAll() {}

// MediaInfo / MediaConfig 桌面端解码参数与探测结果；浏览器端仅保留类型以便编译。
type MediaInfo struct {
	Path       string
	Format     string
	Duration   time.Duration
	Width      int
	Height     int
	FPS        float64
	Bitrate    int64
	VideoCodec string
	AudioCodec string
	SampleRate int
	Channels   int
	HasVideo   bool
	HasAudio   bool
	Backend    string
}

type MediaConfig struct {
	FFmpeg    string
	FFprobe   string
	MaxWidth  int
	MaxHeight int
	FPS       float64
	NoAudio   bool
	Volume    float32
	HWAccel   bool
}

// MediaSource 媒体源（浏览器端用 <video> 标签，这里仅保留类型）。
type MediaSource struct {
	Path   string
	Info   MediaInfo
	Config MediaConfig
}

// OpenMedia 打开媒体（浏览器端不支持本地文件解码）。
func OpenMedia(path string, cfg MediaConfig) (*MediaSource, error) { return nil, errUnsupportedMedia }

// OpenMediaWith 带探测结果打开媒体。
func OpenMediaWith(path string, cfg MediaConfig, info MediaInfo) (*MediaSource, error) {
	return nil, errUnsupportedMedia
}
func MediaFormats() []string {
	return []string{"mp4", "webm", "m4a", "mp3", "wav", "ogg"}
}

// ---- 平台小工具（桌面端在 platform.go / system.go 里用 Win32 实现）----

// Current 返回当前活跃窗口。
func Current() *App { return currentApp.Load() }

// NewWindow 浏览器端只支持单窗口，直接复用当前窗口。
func NewWindow(opts Options, view func() *Node) (*App, error) { return OpenWindow(opts, view) }

// ShellOpen 用新标签页打开链接。
func ShellOpen(path string) {
	js.Global().Call("open", path, "_blank")
}

// ClipboardSetText 写入剪贴板（异步 API，成功与否以调用是否抛错为准）。
func ClipboardSetText(s string) bool {
	nav := js.Global().Get("navigator")
	cb := nav.Get("clipboard")
	if cb.IsNull() || cb.IsUndefined() {
		return false
	}
	cb.Call("writeText", s)
	return true
}

// ClipboardText 浏览器端需要用户授权，这里返回空串（组件会走「无文本」分支）。
func ClipboardText() string { return "" }

// dumpNode 布局树导出（浏览器端仅输出节点数量，用于排障）。
func dumpNode(sb *strings.Builder, n *Node, depth int) {
	if n == nil || depth > 8 {
		return
	}
	sb.WriteString("node\n")
	for _, c := range n.Children {
		dumpNode(sb, c, depth+1)
	}
}

// paintWebView 内嵌浏览器没有独立子窗口，浏览器端直接跳过。
func (a *App) paintWebView(n *Node, r geom.Rect, st *Style) {}
