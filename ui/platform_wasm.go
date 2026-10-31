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

	"gogui/internal/text"
)

// 浏览器端平台垫片（只放**真正 Web 特有**的实现）。
//
// 媒体 / 音乐 / 内嵌浏览器 / 视频等组件在 Web 端与其它非 Windows 平台共用同一套
// 空实现（media_stub.go / music_stub.go / webview_stub.go），因此这里不再重复定义，
// 避免出现「Web 版组件与桌面版不一致」的情况。
//
// 本文件负责：字体注入、窗口/链接/剪贴板、文件对话框、托盘、系统杂项。

// RegisterFont 注册一份字体数据（浏览器端唯一字体来源，如 fetch 回来的 TTF）。
func RegisterFont(name string, data []byte) { text.RegisterFont(name, data) }

// RegisterFontBold 注册粗体字重（以 "<name>/Bold" 命名，字体缓存按字重优先取它）。
func RegisterFontBold(name string, data []byte) { text.RegisterFont(name+"/Bold", data) }

// ---- 窗口 ----

// Current 返回当前活跃窗口。
func Current() *App { return currentApp.Load() }

// NewWindow 浏览器端只支持单窗口，直接复用当前窗口。
func NewWindow(opts Options, view func() *Node) (*App, error) { return OpenWindow(opts, view) }

// ---- 打开链接 / 剪贴板 ----

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

// ClipboardSetFiles 浏览器端不能写文件到剪贴板，退化为文本（路径列表）。
func ClipboardSetFiles(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	out := ""
	for i, p := range paths {
		if i > 0 {
			out += "\n"
		}
		out += p
	}
	return ClipboardSetText(out)
}

// ---- 文件对话框（浏览器沙箱内不可用，交由业务用 <input type=file> 自行实现）----

// errNoDialog 浏览器端没有系统文件对话框。
var errNoDialog = errors.New("ui: 浏览器端没有系统文件对话框（可用 HasCap(CapDialogs) 判断）")

// OpenFileDialog 打开文件对话框（浏览器端不支持同步返回，用 PickFile）。
func OpenFileDialog(title, filter string) (string, bool) { return "", false }

// PickFile 浏览器端选文件：用隐藏的 <input type=file> 异步实现。
//
// 图片（accept 为 image/*）回调 data URL，可直接交给 ui.ImageSized / ui.DecodeImage；
// 其它文件回调 blob: URL，可直接作为 <video src> / Link 地址。用户取消则不回调。
func PickFile(title, filter string, onPick func(path string)) {
	pickFiles(filter, false, func(paths []string) {
		if len(paths) > 0 && onPick != nil {
			onPick(paths[0])
		}
	})
}

// PickFilesAsync 浏览器端多选文件（回调顺序与用户选择顺序一致）。
func PickFilesAsync(title, filter string, onPick func(paths []string)) {
	pickFiles(filter, true, onPick)
}

// pickFiles 建一个隐藏的 <input type=file> 并触发点击。
//
// 图片读成 data URL（可直接交给 ui.ImageSized 解码），其它文件用 blob: URL
// （可直接作为 <video src>）。多选时按用户选择顺序回调。
func pickFiles(filter string, multiple bool, onPick func(paths []string)) {
	doc := js.Global().Get("document")
	in := doc.Call("createElement", "input")
	in.Set("type", "file")
	in.Get("style").Set("display", "none")
	if multiple {
		in.Set("multiple", true)
	}
	if accept := acceptOfFilter(filter); accept != "" {
		in.Set("accept", accept)
	}

	onChange := js.FuncOf(func(this js.Value, args []js.Value) any {
		files := in.Get("files")
		n := files.Length()
		if n == 0 {
			return nil
		}
		out := make([]string, n)
		ready := make([]bool, n)
		done := func() {
			for i := 0; i < n; i++ {
				if !ready[i] {
					return
				}
			}
			if p := in.Get("parentNode"); p.Type() == js.TypeObject {
				p.Call("removeChild", in)
			}
			var paths []string
			for _, s := range out {
				if s != "" {
					paths = append(paths, s)
				}
			}
			if len(paths) > 0 && onPick != nil {
				onPick(paths)
			}
			RequestRedraw()
		}
		for i := 0; i < n; i++ {
			idx := i
			f := files.Index(i)
			typ := strings.ToLower(f.Get("type").String())
			if strings.HasPrefix(typ, "image/") {
				readAsDataURL(f, func(url string) {
					out[idx] = url
					ready[idx] = true
					done()
				})
				continue
			}
			u := js.Global().Get("URL").Call("createObjectURL", f)
			out[idx] = u.String()
			ready[idx] = true
		}
		done()
		return nil
	})
	in.Call("addEventListener", "change", onChange)
	doc.Get("body").Call("appendChild", in)
	in.Call("click")
}

// readAsDataURL 把 File 读成 data URL（图片专用，几 MB 内开销可接受）。
func readAsDataURL(f js.Value, onDone func(url string)) {
	r := js.Global().Get("FileReader").New()
	load := js.FuncOf(func(this js.Value, args []js.Value) any {
		if res := r.Get("result"); res.Type() == js.TypeString {
			onDone(res.String())
		} else {
			onDone("")
		}
		return nil
	})
	r.Call("addEventListener", "load", load)
	r.Call("readAsDataURL", f)
}

// acceptOfFilter 把 "图片|*.png;*.jpg|所有文件|*.*" 转成 input 的 accept 值。
func acceptOfFilter(filter string) string {
	parts := strings.Split(filter, "|")
	var exts []string
	seen := map[string]bool{}
	for _, p := range parts {
		for _, one := range strings.Split(p, ";") {
			one = strings.TrimSpace(one)
			if !strings.HasPrefix(one, "*.") {
				continue
			}
			if one == "*.*" {
				continue
			}
			e := strings.TrimPrefix(one, "*")
			if !seen[e] {
				seen[e] = true
				exts = append(exts, e)
			}
		}
	}
	return strings.Join(exts, ",")
}

// SaveFileDialog 保存文件对话框（浏览器端不支持：用 a[download] 代替）。
func SaveFileDialog(title, filter, defExt string) (string, bool) { return "", false }

// PickFolderDialog 选择文件夹（浏览器端不支持）。
func PickFolderDialog(title string) (string, bool) { return "", false }

// PickFiles 多选文件（浏览器端不支持：用 <input type=file multiple>）。
func PickFiles(title, filter string) ([]string, bool) { return nil, false }

// RunAsync 在「可安全调用平台 API 的线程」上执行 fn：浏览器端是单线程，
// 直接同步执行（另起 goroutine 调 DOM 反而会把渲染循环卡住）。
func RunAsync(fn func()) {
	if fn != nil {
		fn()
	}
}

// ---- 启动参数 ----

// StartupQuery 读取 Web 端启动参数（?key=value，例如 ?case=video）；
// 桌面端没有 URL，恒返回空串。
func StartupQuery(key string) string {
	if key == "" {
		return ""
	}
	loc := js.Global().Get("location")
	if loc.IsUndefined() || loc.IsNull() {
		return ""
	}
	search := loc.Get("search").String()
	if search == "" {
		return ""
	}
	q := js.Global().Get("URLSearchParams").New(strings.TrimPrefix(search, "?"))
	v := q.Call("get", key)
	if v.IsUndefined() || v.IsNull() {
		return ""
	}
	return v.String()
}

// ---- 托盘（浏览器端不存在）----

// TrayItem 托盘菜单项（字段与桌面端一致，保证同一份菜单代码可编译）。
type TrayItem struct {
	ID        int
	Label     string
	Separator bool
	Checked   bool
	Disabled  bool
	Submenu   []TrayItem
}

// Tray 托盘图标（浏览器端占位）。
type Tray struct {
	tooltip string
}

// NewTray 创建托盘（浏览器端永远返回 "不支持"）。
func NewTray(tooltip string, onClick func(), items []TrayItem, onMenu func(id int)) (*Tray, error) {
	return &Tray{tooltip: tooltip}, errors.New("ui: 浏览器端没有系统托盘")
}

// SetTooltip 更新提示。
func (t *Tray) SetTooltip(s string) {
	if t != nil {
		t.tooltip = s
	}
}

// SetItems 更新菜单。
func (t *Tray) SetItems(items []TrayItem) {}

// Notify 气泡通知（不支持）。
func (t *Tray) Notify(title, msg string) {}

// Remove 移除托盘。
func (t *Tray) Remove() { t = nil }

// ---- 系统杂项 ----

// RevealInExplorer 在文件管理器中定位（浏览器端不支持）。
func RevealInExplorer(path string) bool { return false }

// MoveToRecycleBin 移入回收站（浏览器端不支持）。
func MoveToRecycleBin(path string) bool { return false }

// SetKeepAwake 阻止休眠：浏览器端用 Screen Wake Lock API（存在才启用）。
func SetKeepAwake(on bool) uint32 {
	return platformKeepAwake(on)
}

// SystemDarkMode 系统是否深色模式（matchMedia prefers-color-scheme）。
func SystemDarkMode() bool { return platformPrefersDark() }

// platformPrefersDark 读取浏览器主题偏好；不可用时默认深色（与桌面一致）。
func platformPrefersDark() bool {
	mq := js.Global().Call("matchMedia", "(prefers-color-scheme: light)")
	if mq.IsNull() || mq.IsUndefined() {
		return true
	}
	matches := mq.Get("matches")
	if matches.IsUndefined() || matches.IsNull() {
		return true
	}
	return !matches.Bool() // 匹配「浅色」说明当前是浅色，取其反
}

// wakeLock 保存已申请的 Wake Lock（浏览器端防止屏幕休眠）。
var wakeLock js.Value

// platformKeepAwake 通过 Screen Wake Lock API 阻止休眠。返回调用前的状态。
func platformKeepAwake(on bool) uint32 {
	prev := uint32(0)
	if !wakeLock.IsUndefined() && !wakeLock.IsNull() {
		prev = 1
	}
	nav := js.Global().Get("navigator")
	if nav.IsUndefined() || nav.IsNull() {
		return prev
	}
	wl := nav.Get("wakeLock")
	if wl.IsUndefined() || wl.IsNull() {
		return prev
	}
	if !on {
		if prev == 1 {
			wakeLock.Call("release")
			wakeLock = js.Undefined()
		}
		return prev
	}
	if prev == 0 {
		// request() 返回 Promise<WakeLockSentinel>
		then := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				wakeLock = args[0]
			}
			return nil
		})
		wl.Call("request", "screen").Call("then", then)
	}
	return prev
}

// ---- 解码后端命名（浏览器端无本地解码器）----

// BackendName 当前可用的解码后端名称（浏览器端走 <video> 原生解码）。
func BackendName() string { return "browser" }

// BackendLabel 解码后端的中文标签。
func BackendLabel(name string) string {
	switch name {
	case "ffmpeg":
		return "ffmpeg 解码"
	case "system":
		return "系统解码"
	case "browser":
		return "浏览器原生解码（<video>）"
	}
	return "无可用解码器（浏览器端）"
}

// FormatBackend 后端名短标签。
func FormatBackend(name string) string {
	switch name {
	case "system":
		return "系统"
	case "ffmpeg":
		return "ffmpeg"
	case "browser":
		return "浏览器"
	}
	return "—"
}
