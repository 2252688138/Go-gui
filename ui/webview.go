//go:build windows

package ui

import (
	"errors"
	"math"
	"strings"

	"gogui/internal/geom"
	"gogui/internal/gpu"
	"gogui/internal/win32"
)

// WebView 内嵌浏览器节点（系统 WebView2 / Edge 内核）。
//
// 渲染方式（当前为 A 阶段 · HWND 托管）：网页由 Chromium 的 GPU 进程渲染到
// 自己的子窗口上，子窗口位置 = 节点矩形（按 DPI 换算成物理像素），
// 因此：
//   - 帧率与交互是浏览器原生的，网页里的滚动、输入法、视频、canvas 全部真实可用；
//   - 但网页像素不经过 GoGUI 的 OpenGL 批次，**无法**被自绘内容裁剪 / 圆角 / 覆盖。
//
// 想让它像普通控件一样被裁剪、圆角、被菜单盖住，需要升级到 DirectComposition
// 合成（B 阶段，见 COMPONENTS.md 的 roadmap）。
type WebView struct {
	core *win32.WebView2
	app  *App

	// 上一帧的逻辑矩形与可见状态（调试 / 命中测试用）
	rect    geom.Rect
	visible bool
	// radius 网页区域圆角（逻辑像素；>0 时用窗口区域真裁剪，而非画圆角）
	radius float32
	// 合成模式（B 阶段）：网页是 DComp 图层，位置/裁剪走视觉属性
	compose    bool
	compVisual uintptr
}

// WebViewOptions 创建内嵌浏览器参数。
type WebViewOptions struct {
	// URL 初始地址（http/https/file 均可）。
	URL string
	// HTML 直接加载 HTML（优先于 URL，适合离线页面 / 自绘模板）。
	HTML string
	// UserDataDir 用户数据目录（cookie / 缓存），空 = %LOCALAPPDATA%\GoGUI\WebView2。
	UserDataDir string
	// Zoom 初始缩放（1 = 100%）。
	Zoom float64
	// Transparent 背景透明（网页自身透明处透出下层自绘内容）。
	Transparent bool
	// OnChange 页面状态变化（加载开始/完成、地址、标题、历史）时回调。
	OnChange func()
}

// NewWebView 在当前窗口创建内嵌浏览器（须在 UI 线程调用，如按钮回调里）。
func NewWebView(opts WebViewOptions) (*WebView, error) {
	return NewWebViewIn(currentApp.Load(), opts)
}

// NewWebViewIn 在指定窗口创建内嵌浏览器。
func NewWebViewIn(a *App, opts WebViewOptions) (*WebView, error) {
	if a == nil {
		return nil, errors.New("ui: 没有活动窗口，无法创建 WebView")
	}
	var compRoot uintptr
	if a.comp != nil {
		if v, verr := a.comp.NewVisual(); verr == nil {
			compRoot = v
		}
	}
	core, err := win32.NewWebView2(win32.WebView2Options{
		Parent:      a.HWND(),
		URL:         opts.URL,
		HTML:        opts.HTML,
		UserDataDir: opts.UserDataDir,
		Zoom:        opts.Zoom,
		Transparent: opts.Transparent,
		CompositionRoot: compRoot,
		OnChange: func() {
			a.Invalidate()
			if opts.OnChange != nil {
				opts.OnChange()
			}
		},
	})
	if err != nil {
		return nil, err
	}
	wv := &WebView{core: core, app: a, compose: compRoot != 0, compVisual: compRoot}
	a.registerWebView(wv)
	return wv, nil
}

// Available WebView2 运行时是否可用（未安装时组件降级为占位卡片）。
func WebViewAvailable() bool { return win32.WebView2RuntimeAvailable() }

// Ready 浏览器内核是否已就绪。
func (wv *WebView) Ready() bool { return wv != nil && wv.core != nil && wv.core.Ready() }

// Err 创建错误（无则 nil）。
func (wv *WebView) Err() error {
	if wv == nil || wv.core == nil {
		return errors.New("WebView 未初始化")
	}
	return wv.core.Err()
}

// Navigate 打开地址。
func (wv *WebView) Navigate(url string) {
	if wv == nil || wv.core == nil || url == "" {
		return
	}
	wv.core.Navigate(normalizeURL(url))
	RequestRedraw()
}

// LoadHTML 加载一段 HTML。
func (wv *WebView) LoadHTML(html string) {
	if wv == nil || wv.core == nil {
		return
	}
	wv.core.LoadHTML(html)
	RequestRedraw()
}

// Back / Forward / Reload / Stop 导航控制。
func (wv *WebView) Back() {
	if wv == nil || wv.core == nil {
		return
	}
	wv.core.Back()
}

func (wv *WebView) Forward() {
	if wv == nil || wv.core == nil {
		return
	}
	wv.core.Forward()
}

func (wv *WebView) Reload() {
	if wv == nil || wv.core == nil {
		return
	}
	wv.core.Reload()
}

func (wv *WebView) Stop() {
	if wv == nil || wv.core == nil {
		return
	}
	wv.core.StopLoad()
}

// Focus 把焦点交回网页。
func (wv *WebView) Focus() {
	if wv != nil && wv.core != nil {
		wv.core.Focus()
	}
}

// SetRadius 网页区域圆角（逻辑像素，>0 生效）。
// 实现方式是对网页子窗口 SetWindowRgn 区域裁剪，所以圆角是真的裁掉而不是遮罩。
func (wv *WebView) SetRadius(r float32) {
	if wv != nil {
		wv.radius = r
		RequestRedraw()
	}
}

// SetZoom 缩放（1 = 100%）。
func (wv *WebView) SetZoom(z float64) {
	if wv != nil && wv.core != nil {
		wv.core.SetZoom(z)
	}
}

// ExecuteScript 在页面执行 JS。
func (wv *WebView) ExecuteScript(js string) {
	if wv != nil && wv.core != nil {
		wv.core.ExecuteScript(js)
	}
}

// URL 当前地址。
func (wv *WebView) URL() string {
	if wv == nil || wv.core == nil {
		return ""
	}
	return wv.core.URL()
}

// Title 当前页面标题。
func (wv *WebView) Title() string {
	if wv == nil || wv.core == nil {
		return ""
	}
	return wv.core.Title()
}

// Loading 是否正在加载。
func (wv *WebView) Loading() bool { return wv != nil && wv.core != nil && wv.core.Loading() }

// CanGoBack / CanGoForward 历史状态。
func (wv *WebView) CanGoBack() bool { return wv != nil && wv.core != nil && wv.core.CanGoBack() }

func (wv *WebView) CanGoForward() bool {
	return wv != nil && wv.core != nil && wv.core.CanGoForward()
}

// Close 关闭浏览器（窗口销毁时会自动调用，也可手动提前释放）。
func (wv *WebView) Close() {
	if wv == nil || wv.core == nil {
		return
	}
	wv.core.Close()
	RequestRedraw()
}

// normalizeURL 补全协议：输入 "example.com" 视为 https。
func normalizeURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	low := strings.ToLower(s)
	for _, p := range []string{"http://", "https://", "file:", "about:", "data:", "edge:"} {
		if strings.HasPrefix(low, p) {
			return s
		}
	}
	if strings.Contains(s, "://") {
		return s
	}
	return "https://" + s
}

// Browser WebView 节点（网页区域的占位节点；真实像素由浏览器子窗口绘制）。
func Browser(wv *WebView) *Node {
	return WebViewNode(wv)
}

// WebViewNode WebView 节点，用法同 ui.Video(p)：尺寸由样式决定，
// 通常 `.St(ui.St().GrowTo(1))` 或 `.St(ui.St().Wp(100).H(520))`。
func WebViewNode(wv *WebView) *Node {
	return &Node{Kind: KWebView, WebView: wv, Style: Style{Opacity: 1}}
}

// ---- 自绘浏览器工具条 ----

// WebViewBarOptions 工具条参数。
type WebViewBarOptions struct {
	// Address 地址栏当前文本（由调用方持有）。
	Address string
	// Key 地址输入框的 key（同一界面内唯一）。
	Key string
	// OnAddress 地址文本变化。
	OnAddress func(string)
	// OnGo 回车或点击「打开」时触发（通常调 wv.Navigate(addr)）。
	OnGo func()
	// ShowTitle 在地址栏右侧显示页面标题。
	ShowTitle bool
}

// WebViewBar 自绘浏览器工具条：← → ↻ + 地址栏 + 打开 + 加载指示。
// 放在 WebViewNode 正上方即可组成完整浏览器界面。
func WebViewBar(wv *WebView, o WebViewBarOptions) *Node {
	key := o.Key
	if key == "" {
		key = "wv-addr"
	}
	canBack, canFwd, loading := false, false, false
	if wv != nil {
		canBack, canFwd, loading = wv.CanGoBack(), wv.CanGoForward(), wv.Loading()
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

	left := navBtn("‹", canBack && wv != nil, func() {
		if wv != nil {
			wv.Back()
		}
	})
	right := navBtn("›", canFwd && wv != nil, func() {
		if wv != nil {
			wv.Forward()
		}
	})
	reloadGlyph := "⟳"
	if loading {
		reloadGlyph = "×"
	}
	reload := navBtn(reloadGlyph, wv != nil, func() {
		if wv == nil {
			return
		}
		if wv.Loading() {
			wv.Stop()
		} else {
			wv.Reload()
		}
	})

	addr := Box(InputSubmit(key, o.Address, "输入网址，回车打开", func(v string) {
		if o.OnAddress != nil {
			o.OnAddress(v)
		}
	}, func() {
		if o.OnGo != nil {
			o.OnGo()
		}
	}).St(St().H(34).R(9))).Grow(1)

	cells := []*Node{left, right, reload, addr}
	if loading {
		cells = append(cells, Spinner(16, ColPrimary))
	}
	if o.ShowTitle && wv != nil && wv.Title() != "" {
		cells = append(cells, Text(ellipsize(wv.Title(), 34)).St(St().Text(11.5, 500, ColTextMuted)))
	}
	cells = append(cells, Btn("打开", BtnPrimary, func() {
		if o.OnGo != nil {
			o.OnGo()
		}
	}))

	return Row(cells...).St(St().GapOf(8).Padding(8).R(12).
		BgC(ColBgSoft).Stroke(1, ColStroke)).AlignTo(AlignCenter)
}

// WebViewStatus 页面状态行：地址 / 标题 / 加载中提示。
func WebViewStatus(wv *WebView) *Node {
	if wv == nil {
		return nil
	}
	label := wv.Title()
	if label == "" {
		label = wv.URL()
	}
	if label == "" {
		label = "空白页"
	}
	dot := ColSuccess
	switch {
	case wv.Err() != nil:
		dot = ColDanger
		label = "WebView2 不可用：" + wv.Err().Error()
	case wv.Loading():
		dot = ColWarn
	case !wv.Ready():
		dot = ColWarn
		label = "正在启动浏览器内核…"
	}
	return Row(
		Dot(dot, 7),
		Text(ellipsize(label, 76)).St(St().Text(11.5, 500, ColTextMuted)),
		Spacer(),
		Text(ellipsize(wv.URL(), 52)).St(St().Text(11, 400, ColPlaceholder)),
	).St(St().GapOf(8).PadXY(12, 6).R(9).BgC(ColBgDeep).Stroke(1, ColStroke)).
		AlignTo(AlignCenter)
}

func ellipsize(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

// ---- 绘制层 ----

// paintWebView 绘制网页节点：记录本帧矩形（供 frame 同步子窗口），
// 并在内核未就绪 / 失败时画占位内容（网页窗口盖上去之后就看不见了）。
func (a *App) paintWebView(n *Node, r geom.Rect, st *Style) {
	wv := n.WebView
	if wv == nil || wv.core == nil {
		a.paintWebViewCard(r, st, "WebView 未创建", ColTextMuted, false)
		return
	}
	// 裁剪到祖先裁剪区（滚动视口 / Clipped 容器）：HWND 本身无法裁剪，
	// 这里按交集缩小窗口，粗略还原「被裁掉」的效果。
	vr := r
	if a.clipN > 0 {
		vr = intersectRect(vr, a.clipStack[a.clipN-1])
	}
	if vr.W < 1 || vr.H < 1 {
		return
	}
	if a.wvRects == nil {
		a.wvRects = map[*WebView]geom.Rect{}
	}
	a.wvRects[wv] = vr
	wv.rect = vr

	if err := wv.core.Err(); err != nil {
		a.paintWebViewCard(r, st, "WebView2 暂不可用\n"+err.Error(), ColDanger, false)
		return
	}
	if !wv.core.Ready() {
		// 内核启动中：暗底 + 呼吸提示，逐帧重绘推进动画
		a.paintWebViewCard(r, st, "正在启动浏览器内核…", ColTextMuted, true)
		a.animating = true
	}
}

// paintWebViewCard 网页区域占位卡片（暗底 + 圆角 + 居中提示）。
func (a *App) paintWebViewCard(r geom.Rect, st *Style, label string, color string, pulse bool) {
	alpha := float32(0.04)
	if pulse {
		alpha = 0.03 + 0.02*(1+float32(math.Sin(float64(AnimTime())*2.4)))/2
	}
	a.r.Rect(gpu.Shape{
		Rect:   r,
		Radius: st.Radius,
		Fill:   paintOf(ColText, alpha),
	})
	if label == "" {
		return
	}
	tmp := &Node{
		Kind: KText,
		Text: label,
		Style: Style{
			Opacity: 1, FontSize: 12, FontWeight: 500, Color: C(color),
			CenterY: true, TextAlign: AlignCenter,
		},
	}
	a.paintText(tmp, r, &tmp.Style)
}

// intersectRect 求两个矩形交集（无交集时宽高为 0）。
func intersectRect(a, b geom.Rect) geom.Rect {
	x0 := maxF(a.X, b.X)
	y0 := maxF(a.Y, b.Y)
	x1 := minF(a.Right(), b.Right())
	y1 := minF(a.Bottom(), b.Bottom())
	if x1 <= x0 || y1 <= y0 {
		return geom.Rect{}
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// ---- 窗口同步 ----

// registerWebView 登记实例（窗口销毁时统一关闭）。
func (a *App) registerWebView(wv *WebView) {
	if a == nil || wv == nil {
		return
	}
	if a.webviews == nil {
		a.webviews = map[*WebView]bool{}
	}
	a.webviews[wv] = true
}

// syncWebViews 把本帧绘制到的 WebView 矩形换算成物理像素交给浏览器子窗口，
// 本帧没绘制的（被滚出视口 / 切页 / 隐藏）一律隐藏。
func (a *App) syncWebViews() {
	if a == nil || a.win == nil {
		return
	}
	scale := a.effScale()
	for wv := range a.webviews {
		if wv == nil || wv.core == nil {
			continue
		}
		r, ok := a.wvRects[wv]
		if !ok || r.W < 1 || r.H < 1 {
			if a.wvShown[wv] {
				wv.core.SetVisible(false)
				wv.visible = false
				delete(a.wvShown, wv)
			}
			continue
		}
		wv.core.Refresh()
		x := int32(math.Round(float64(r.X * scale)))
		y := int32(math.Round(float64(r.Y * scale)))
		w := int32(math.Round(float64(r.W * scale)))
		h := int32(math.Round(float64(r.H * scale)))
		if wv.compose && a.comp != nil {
// B 阶段：网页是 DComp 图层——位置由视觉偏移决定，bounds 从 (0,0) 起算，
// 圆角交给 IDCompositionRectangleClip（抗锯齿），不再用窗口区域硬裁。
a.comp.SetVisualOffset(wv.compVisual, r.X*scale, r.Y*scale)
wv.core.SetBounds(0, 0, w, h)
wv.core.SetVisible(true)
if wv.radius > 0 {
_ = a.comp.SetRoundedClip(wv.compVisual, 0, 0, float32(w), float32(h), wv.radius*scale)
}
} else {
wv.core.SetBounds(x, y, w, h)
wv.core.SetVisible(true)
if wv.radius > 0 {
wv.core.SetRoundedRegion(int32(math.Round(float64(wv.radius * scale))))
}
}
		wv.visible = true
		if a.wvShown == nil {
			a.wvShown = map[*WebView]bool{}
		}
		a.wvShown[wv] = true
	}
}

// closeWebViews 关闭本窗口的所有浏览器实例（dispose 时调用）。
func (a *App) closeWebViews() {
	for wv := range a.webviews {
		if wv != nil && wv.core != nil {
			wv.core.Close()
		}
	}
	a.webviews = nil
	a.wvRects = nil
	a.wvShown = nil
}
