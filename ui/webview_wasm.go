//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"fmt"
	"strings"
	"syscall/js"

	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// 浏览器端的内嵌浏览器与视频画面。
//
//   - WebView：用 <iframe> 承载网页，叠加在画布上（pointer-events:auto，可与网页交互），
//     前进后退用自建地址栈（iframe 跨域读取 location/history 会抛 SecurityError）。
//   - 视频：画面由 <video> 元素自己渲染，这里只负责按 UI 矩形定位并保留占位逻辑。

// errNoWebView 浏览器端创建内嵌浏览器失败。
var errNoWebView = errors.New("ui: 浏览器端无法创建内嵌浏览器")

// videoTexture 浏览器端不使用显存纹理（画面由 DOM 元素直接显示），仅保留类型占位。
type videoTexture struct{}

// intersectRect 两个矩形求交（无交集返回零矩形）。
func intersectRect(a, b geom.Rect) geom.Rect {
	x0 := maxF(a.X, b.X)
	y0 := maxF(a.Y, b.Y)
	x1 := minF(a.X+a.W, b.X+b.W)
	y1 := minF(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return geom.Rect{}
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// cssRadius 把四角圆角数组转成 CSS border-radius 值。
func cssRadius(r [4]float32) string {
	if r[0] == r[1] && r[1] == r[2] && r[2] == r[3] {
		return fmt.Sprintf("%.2fpx", r[0])
	}
	return fmt.Sprintf("%.2fpx %.2fpx %.2fpx %.2fpx", r[0], r[1], r[2], r[3])
}

// normalizeURL 补全协议（与桌面端一致）。
func normalizeURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "about:blank"
	}
	if strings.HasPrefix(s, "about:") || strings.HasPrefix(s, "data:") || strings.HasPrefix(s, "blob:") {
		return s
	}
	if strings.Contains(s, "://") {
		return s
	}
	return "https://" + s
}

// paintVideo 视频画面：把 <video> 元素放到画面矩形上；无媒体源时画占位。
func (a *App) paintVideo(n *Node, r geom.Rect, st *Style) {
	p := n.Video
	if p == nil {
		a.paintVideoPlaceholder(n, r, st)
		return
	}
	// 记录画面矩形，供控制条自动隐藏与点击判定（与桌面端一致）
	p.rect = rectBox{X: r.X, Y: r.Y, W: r.W, H: r.H}

	src, ok := p.Source.(*MediaSource)
	if !ok || src == nil || src.el.IsUndefined() || src.el.IsNull() {
		a.paintVideoPlaceholder(n, r, st)
		return
	}
	fit := "contain"
	switch p.Fit {
	case FitCover:
		fit = "cover"
	case FitFill:
		fit = "fill"
	}
	rot := p.RotationDeg()
	radius := cssRadius(st.Radius)
	clip := a.currentClip(r)
	placeOverlay(src.id, r, clip, func(el js.Value) {
		stx := el.Get("style")
		stx.Set("objectFit", fit)
		stx.Set("borderRadius", radius)
		if rot != 0 {
			stx.Set("transform", fmt.Sprintf("rotate(%ddeg)", rot))
		} else {
			stx.Set("transform", "none")
		}
	})
}

// paintVideoPlaceholder 无画面时的占位（暗底 + 提示文字）。
func (a *App) paintVideoPlaceholder(n *Node, r geom.Rect, st *Style) {
	a.r.Rect(gpu.Shape{
		Rect:   r,
		Radius: st.Radius,
		Fill:   paintOf(ColText, 0.06),
	})
	label := "无画面"
	if p := n.Video; p != nil {
		switch {
		case p.Err() != nil:
			label = "浏览器无法播放该源"
		case p.Source != nil:
			label = "解码中…"
		}
	}
	tmp := &Node{
		Kind:  KText,
		Text:  label,
		Style: Style{Opacity: 1, FontSize: 12, FontWeight: 500, Color: C("#6b7694"), CenterY: true, TextAlign: AlignCenter},
	}
	a.paintText(tmp, r, &tmp.Style)
}

// WebViewOptions 内嵌浏览器选项。
type WebViewOptions struct {
	URL         string
	HTML        string
	UserDataDir string
	Zoom        float64
	Transparent bool
	OnChange    func()
}

// WebView 浏览器端内嵌浏览器（<iframe>）。
type WebView struct {
	url     string
	html    string
	title   string
	loading bool
	visible bool
	zoom    float64
	radius  float32

	id    string
	el    js.Value
	funcs []js.Func
	stack []string
	idx   int
}

// NewWebView 在当前窗口创建内嵌浏览器。
func NewWebView(opts WebViewOptions) (*WebView, error) {
	return NewWebViewIn(currentApp.Load(), opts)
}

// NewWebViewIn 在指定窗口创建内嵌浏览器。
func NewWebViewIn(a *App, opts WebViewOptions) (*WebView, error) {
	wv := &WebView{zoom: 1, visible: true, loading: true, idx: -1}
	if opts.Zoom > 0 {
		wv.zoom = opts.Zoom
	}
	wv.id = overlayID("webview")
	el := overlayEl(wv.id, "iframe")
	el.Set("frameborder", "0")
	el.Set("allow", "fullscreen; autoplay; clipboard-read; clipboard-write")
	st := el.Get("style")
	st.Set("pointerEvents", "auto")
	st.Set("background", "#ffffff")
	st.Set("borderRadius", fmt.Sprintf("%.2fpx", wv.radius))
	wv.el = el

	load := js.FuncOf(func(this js.Value, args []js.Value) any {
		wv.loading = false
		// iframe.src 属性跨域可读，用它跟踪页面内跳转
		if s := el.Get("src").String(); s != "" && s != "about:blank" {
			wv.url = s
			if wv.idx < 0 || wv.stack[wv.idx] != s {
				wv.push(s)
			}
		}
		RequestRedraw()
		return nil
	})
	wv.funcs = append(wv.funcs, load)
	el.Call("addEventListener", "load", load)

	switch {
	case opts.HTML != "":
		wv.LoadHTML(opts.HTML)
	case opts.URL != "":
		wv.Navigate(opts.URL)
	}
	if opts.OnChange != nil {
		opts.OnChange()
	}
	return wv, nil
}

// push 把地址压入自建栈（用于跨域前进后退）。
func (wv *WebView) push(url string) {
	if wv == nil || url == "" {
		return
	}
	wv.stack = append(wv.stack[:wv.idx+1], url)
	wv.idx = len(wv.stack) - 1
}

// WebViewAvailable 浏览器端始终可用。
func WebViewAvailable() bool { return true }

// Ready 是否已就绪。
func (wv *WebView) Ready() bool { return wv != nil && !wv.el.IsUndefined() && !wv.el.IsNull() }

// Err 当前错误。
func (wv *WebView) Err() error {
	if wv == nil {
		return errNoWebView
	}
	return nil
}

// Navigate 打开地址。
func (wv *WebView) Navigate(url string) {
	if wv == nil {
		return
	}
	url = normalizeURL(url)
	wv.url = url
	wv.loading = true
	wv.push(url)
	if !wv.el.IsUndefined() && !wv.el.IsNull() {
		wv.el.Set("srcdoc", "")
		wv.el.Set("src", url)
	}
	RequestRedraw()
}

// LoadHTML 加载 HTML 片段（iframe srcdoc）。
func (wv *WebView) LoadHTML(html string) {
	if wv == nil {
		return
	}
	wv.html = html
	wv.loading = true
	wv.url = "about:srcdoc"
	if !wv.el.IsUndefined() && !wv.el.IsNull() {
		wv.el.Set("src", "about:blank")
		wv.el.Set("srcdoc", html)
	}
	RequestRedraw()
}

// Back 后退。
func (wv *WebView) Back() {
	if wv == nil || wv.idx <= 0 {
		return
	}
	wv.idx--
	wv.navTo(wv.stack[wv.idx])
}

// Forward 前进。
func (wv *WebView) Forward() {
	if wv == nil || wv.idx+1 >= len(wv.stack) {
		return
	}
	wv.idx++
	wv.navTo(wv.stack[wv.idx])
}

// navTo 直接切换地址（不压栈）。
func (wv *WebView) navTo(url string) {
	wv.url = url
	wv.loading = true
	if !wv.el.IsUndefined() && !wv.el.IsNull() {
		wv.el.Set("srcdoc", "")
		wv.el.Set("src", url)
	}
	RequestRedraw()
}

// Reload 刷新。
func (wv *WebView) Reload() {
	if wv == nil {
		return
	}
	if wv.html != "" && wv.url == "about:srcdoc" {
		wv.LoadHTML(wv.html)
		return
	}
	wv.navTo(wv.URL())
}

// Stop 停止加载（iframe 无法中断导航，移除 srcdoc/src 之外的等效做法是置 about:blank）。
func (wv *WebView) Stop() {
	if wv == nil || wv.el.IsUndefined() {
		return
	}
	wv.loading = false
	RequestRedraw()
}

// Focus 聚焦到网页。
func (wv *WebView) Focus() {
	if wv == nil || wv.el.IsUndefined() {
		return
	}
	wv.el.Call("focus")
}

// SetRadius 圆角半径。
func (wv *WebView) SetRadius(r float32) {
	if wv == nil {
		return
	}
	wv.radius = r
	if !wv.el.IsUndefined() && !wv.el.IsNull() {
		wv.el.Get("style").Set("borderRadius", fmt.Sprintf("%.2fpx", r))
	}
}

// SetZoom 缩放（CSS zoom）。
func (wv *WebView) SetZoom(z float64) {
	if wv == nil {
		return
	}
	if z <= 0 {
		z = 1
	}
	wv.zoom = z
	if !wv.el.IsUndefined() && !wv.el.IsNull() {
		wv.el.Get("style").Set("zoom", z)
	}
}

// ExecuteScript 注入脚本（跨域会被浏览器拒绝，调用方需自行保证同源）。
func (wv *WebView) ExecuteScript(js string) {
	if wv == nil || wv.el.IsUndefined() || wv.el.IsNull() {
		return
	}
	if doc := wv.el.Get("contentDocument"); !doc.IsNull() && !doc.IsUndefined() {
		doc.Call("defaultView").Call("eval").Invoke(js)
	}
}

// URL 当前地址。
func (wv *WebView) URL() string {
	if wv == nil {
		return ""
	}
	return wv.url
}

// Title 页面标题（跨域读不到，返回地址）。
func (wv *WebView) Title() string {
	if wv == nil {
		return ""
	}
	if wv.title != "" {
		return wv.title
	}
	return wv.url
}

// Loading 是否加载中。
func (wv *WebView) Loading() bool { return wv != nil && wv.loading }

// CanGoBack 能否后退。
func (wv *WebView) CanGoBack() bool { return wv != nil && wv.idx > 0 }

// CanGoForward 能否前进。
func (wv *WebView) CanGoForward() bool { return wv != nil && wv.idx+1 < len(wv.stack) }

// Close 关闭浏览器（移除 DOM 元素）。
func (wv *WebView) Close() {
	if wv == nil {
		return
	}
	wv.visible = false
	for _, f := range wv.funcs {
		f.Release()
	}
	wv.funcs = nil
	if wv.id != "" {
		dropOverlayEl(wv.id)
	}
	if a := currentApp.Load(); a != nil {
		delete(a.wvRects, wv)
		delete(a.wvShown, wv)
		delete(a.webviews, wv)
	}
}

// Browser 浏览器组合节点（地址栏 + 页面区域）。
func Browser(wv *WebView) *Node {
	return Column(
		WebViewBar(wv, WebViewBarOptions{}),
		WebViewNode(wv),
	).St(St().GrowTo(1).GapOf(8))
}

// WebViewNode 页面区域节点：真实像素由叠加的 <iframe> 绘制。
func WebViewNode(wv *WebView) *Node {
	return &Node{Kind: KWebView, WebView: wv, Style: Style{Opacity: 1}}
}

// WebViewBarOptions 地址栏选项。
type WebViewBarOptions struct {
	Address   string
	Key       string
	OnAddress func(string)
	OnGo      func()
	ShowTitle bool
}

// WebViewBar 自绘浏览器工具条：← → ↻ + 地址栏 + 打开 + 新标签。
func WebViewBar(wv *WebView, o WebViewBarOptions) *Node {
	key := o.Key
	if key == "" {
		key = "wv-addr"
	}
	canBack, canFwd := false, false
	addr := o.Address
	if wv != nil {
		canBack, canFwd = wv.CanGoBack(), wv.CanGoForward()
		if addr == "" {
			addr = wv.URL()
		}
	}
	navBtn := func(glyph string, enabled bool, fn func()) *Node {
		st := St().W(32).H(32).MinWidth(32).Text(13.5, 600, ColTextDim).R(9).
			BgC(ColBgSoft).Stroke(1, ColStroke)
		if !enabled {
			st = St().W(32).H(32).MinWidth(32).Text(13.5, 600, "#3a4358").R(9).
				BgC(ColInput).Stroke(1, ColStroke)
		}
		n := Button(glyph, fn).St(st)
		if !enabled {
			n.Disabled = true
		}
		return n
	}
	return Row(
		navBtn("‹", canBack, func() {
			if wv != nil {
				wv.Back()
			}
		}),
		navBtn("›", canFwd, func() {
			if wv != nil {
				wv.Forward()
			}
		}),
		navBtn("↻", wv != nil, func() {
			if wv != nil {
				wv.Reload()
			}
		}),
		Input(key, addr, "输入网址后回车…", func(v string) {
			if o.OnAddress != nil {
				o.OnAddress(v)
			}
		}).St(St().GrowTo(1).H(32).R(8).PadOnly(0, 10, 0, 10)),
		Btn("打开", BtnPrimary, func() {
			if o.OnGo != nil {
				o.OnGo()
			} else if wv != nil {
				wv.Navigate(addr)
			}
		}).St(St().W(56).H(32)),
	).St(St().GapOf(8).H(40)).AlignTo(AlignCenter)
}

// WebViewStatus 状态标签。
func WebViewStatus(wv *WebView) *Node {
	if wv == nil {
		return Tag("浏览器不可用", ColWarn)
	}
	if wv.Loading() {
		return Tag("加载中…", ColWarn)
	}
	return Tag("iframe 直连 · 跨站限制见下", ColSuccess)
}

// paintWebView 把 <iframe> 定位到页面区域矩形（绘制阶段记录，Swap 时写入 DOM）。
func (a *App) paintWebView(n *Node, r geom.Rect, st *Style) {
	wv := n.WebView
	if wv == nil || wv.el.IsUndefined() || wv.el.IsNull() {
		a.paintWebViewCard(r, st, "内嵌浏览器未创建", ColTextMuted)
		return
	}
	if a.wvRects == nil {
		a.wvRects = map[*WebView]geom.Rect{}
	}
	a.wvRects[wv] = r
	clip := a.currentClip(r)
	radius := cssRadius(st.Radius)
	if st.Radius == [4]float32{} {
		radius = fmt.Sprintf("%.2fpx", wv.radius)
	}
	placeOverlay(wv.id, r, clip, func(el js.Value) {
		stx := el.Get("style")
		stx.Set("pointerEvents", "auto")
		stx.Set("borderRadius", radius)
	})
}

// paintWebViewCard 未创建/不可用时的暗底卡片。
func (a *App) paintWebViewCard(r geom.Rect, st *Style, label, color string) {
	a.r.Rect(gpu.Shape{
		Rect:   r,
		Radius: st.Radius,
		Fill:   paintOf(ColText, 0.06),
	})
	tmp := &Node{
		Kind:  KText,
		Text:  label,
		Style: Style{Opacity: 1, FontSize: 12, FontWeight: 500, Color: C(color), CenterY: true, TextAlign: AlignCenter},
	}
	a.paintText(tmp, r, &tmp.Style)
}

// syncWebViews 浏览器端不需要同步子窗口位置（已在 Swap 里统一处理）。
func (a *App) syncWebViews() {}

// closeWebViews 释放内嵌浏览器。
func (a *App) closeWebViews() {
	for wv := range a.webviews {
		wv.Close()
	}
	a.webviews = nil
	a.wvRects = nil
	a.wvShown = nil
}
