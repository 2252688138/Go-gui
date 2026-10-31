//go:build !windows && !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"

	"gogui/internal/geom"
)

// 非 Windows 桌面端的内嵌浏览器垫片。
//
// Windows 走 WebView2（Chromium 子窗口 + DirectComposition 图层）；这些平台需要
// WKWebView（macOS）/ WebKitGTK（Linux）等系统组件，尚未接入。
// 这里保留**与 Windows 完全一致的 API**，创建时返回明确错误，组件降级为占位卡片，
// 业务代码无需任何改动即可编译运行。

// errNoWebView 本平台没有可用的内嵌浏览器。
var errNoWebView = errors.New("ui: 当前平台尚未接入内嵌浏览器（可用 HasCap(CapWebView) 判断）")

// WebViewOptions 内嵌浏览器选项（字段与 Windows 端一致）。
type WebViewOptions struct {
	URL         string
	HTML        string
	UserDataDir string
	Zoom        float64
	Transparent bool
	OnChange    func()
}

// WebView 内嵌浏览器（本平台占位：记录状态但不产生画面）。
type WebView struct {
	url     string
	html    string
	title   string
	loading bool
	visible bool
	zoom    float64
	radius  float32
}

// NewWebView 在当前窗口创建内嵌浏览器（本平台返回不支持）。
func NewWebView(opts WebViewOptions) (*WebView, error) {
	return NewWebViewIn(currentApp.Load(), opts)
}

// NewWebViewIn 在指定窗口创建内嵌浏览器（本平台返回不支持）。
func NewWebViewIn(a *App, opts WebViewOptions) (*WebView, error) {
	wv := &WebView{url: opts.URL, html: opts.HTML, zoom: 1, visible: true}
	if opts.OnChange != nil {
		opts.OnChange()
	}
	return wv, errNoWebView
}

// WebViewAvailable 当前平台是否可用内嵌浏览器。
func WebViewAvailable() bool { return false }

// Ready 是否已就绪（占位实现：恒 false）。
func (wv *WebView) Ready() bool { return false }

// Err 当前错误。
func (wv *WebView) Err() error { return errNoWebView }

// Navigate 打开地址。
func (wv *WebView) Navigate(url string) {
	if wv != nil {
		wv.url, wv.loading = url, false
	}
}

// LoadHTML 加载 HTML。
func (wv *WebView) LoadHTML(html string) {
	if wv != nil {
		wv.html, wv.loading = html, false
	}
}

// Back 后退（空操作）。
func (wv *WebView) Back() {}

// Forward 前进（空操作）。
func (wv *WebView) Forward() {}

// Reload 刷新（空操作）。
func (wv *WebView) Reload() {}

// Stop 停止加载（空操作）。
func (wv *WebView) Stop() {}

// Focus 聚焦（空操作）。
func (wv *WebView) Focus() {}

// SetRadius 圆角半径。
func (wv *WebView) SetRadius(r float32) {
	if wv != nil {
		wv.radius = r
	}
}

// SetZoom 缩放。
func (wv *WebView) SetZoom(z float64) {
	if wv != nil {
		wv.zoom = z
	}
}

// ExecuteScript 注入脚本（空操作）。
func (wv *WebView) ExecuteScript(js string) {}

// URL 当前地址。
func (wv *WebView) URL() string {
	if wv == nil {
		return ""
	}
	return wv.url
}

// Title 页面标题。
func (wv *WebView) Title() string {
	if wv == nil {
		return ""
	}
	return wv.title
}

// Loading 是否加载中。
func (wv *WebView) Loading() bool { return wv != nil && wv.loading }

// CanGoBack 能否后退。
func (wv *WebView) CanGoBack() bool { return false }

// CanGoForward 能否前进。
func (wv *WebView) CanGoForward() bool { return false }

// Close 关闭浏览器。
func (wv *WebView) Close() {
	if wv != nil {
		wv.visible = false
	}
}

// Browser 浏览器组合节点（地址栏 + 页面区域）。
func Browser(wv *WebView) *Node {
	return Column(
		WebViewBar(wv, WebViewBarOptions{}),
		WebViewNode(wv),
	).St(St().GrowTo(1).GapOf(8))
}

// WebViewNode 页面区域节点（本平台渲染占位卡片）。
func WebViewNode(wv *WebView) *Node {
	return Card(
		Empty("内嵌浏览器未启用", "当前平台尚未接入 WebView（Windows 使用 WebView2）"),
	).St(St().GrowTo(1).PadXY(16, 16))
}

// WebViewBarOptions 地址栏选项（字段与 Windows 端一致）。
type WebViewBarOptions struct {
	Address   string
	Key       string
	OnAddress func(string)
	OnGo      func()
	ShowTitle bool
}

// WebViewBar 自绘浏览器工具条（本平台给出禁用态，保持布局一致）。
func WebViewBar(wv *WebView, o WebViewBarOptions) *Node {
	addr := o.Address
	if addr == "" && wv != nil {
		addr = wv.URL()
	}
	return Row(
		Button("←", nil).St(St().W(32)),
		Button("→", nil).St(St().W(32)),
		Button("↻", nil).St(St().W(32)),
		Box().St(St().GrowTo(1).H(32).R(8).BgC(ColBgSoft).Stroke(1, ColStroke)).Add(
			Text(addr).St(St().AbsFill().Text(12, 400, ColPlaceholder).CenterYOn().PadXY(10, 0)),
		),
		Button("打开", nil).St(St().W(56)),
	).St(St().GapOf(8).H(40))
}

// WebViewStatus 状态标签（本平台提示不可用）。
func WebViewStatus(wv *WebView) *Node {
	return Tag("未启用 WebView", ColWarn)
}

// syncWebViews 同步子窗口位置（本平台无子窗口，空操作）。
func (a *App) syncWebViews() {}

// closeWebViews 释放内嵌浏览器。
func (a *App) closeWebViews() {
	a.webviews = nil
	a.wvRects = nil
	a.wvShown = nil
}

// paintWebView 内嵌浏览器绘制：本平台无独立子窗口，跳过。
func (a *App) paintWebView(n *Node, r geom.Rect, st *Style) {}
