package ui

import (
	"strconv"
	"strings"

	"gogui/internal/text"
)

// Markdown 渲染组件（声明式节点，每帧重建）。
//
// 参照 Web 版 frontend/src/components/Markdown.vue 的渲染思路：
//
//	块级解析 → 扩展块（tabs/details/card/gallery/badge/progress）
//	→ 行内样式（粗体/斜体/删除线/行内代码/链接/高亮/kbd）
//	→ 自绘排版（标题/段落/列表/引用/代码/表格/图片/分隔线）。
//
// 与 Web 版的差异（受 GoGUI 约束）：
//   - KText 只支持单一字体样式，行内多样式按「片段 + 手工折行」排版；
//   - 图片支持本地文件与 http(s) 远程地址（远程先下载到临时目录缓存）；
//   - video/audio 块降级为占位卡（真正播放请用 ui.VideoPlayer）；
//   - 链接点击走系统默认浏览器（win32.ShellOpen）。
//
// 支持的块：标题（含锚点）、段落、列表（含任务勾选）、引用/提示块、代码（行号/
// 着色/复制/折叠/diff 行底色/可选横向平移）、表格（数字列右对齐）、图片（含
// lightbox）、分隔线、页签、折叠、卡片、徽章、进度、图集、媒体、目录（可点击
// 滚动）、数学公式（LaTeX 子集真排版）、Mermaid 提示卡、定义列表、脚注。
//
// 用法：
//
//	ui.Scroll(ui.Markdown("md", src, ui.MarkdownOptions{Width: 760})).K("scroll-md")
//	// 图片放大层放到根 Stack 最后一层
//	ui.Stack(page, ui.MarkdownLightbox())
type MarkdownOptions struct {
	// Width 排版宽度（像素）。0 = min(ViewportW()-96, 860)。
	Width float32
	// BaseSize 正文字号，默认 13.5。
	BaseSize float32
	// FontFamily 正文字体（空 = 全局默认微软雅黑）。
	FontFamily string
	// MonoFamily 代码字体，默认 Consolas。
	MonoFamily string
	// LineHeight 正文行高，默认 BaseSize*1.65。
	LineHeight float32
	// MaxImageHeight 图片最大高度，默认 360。
	MaxImageHeight float32
	// CodeMaxLines 代码块折叠阈值（0 = 22，<0 不折叠）。
	CodeMaxLines int
	// CodeNoWrap 代码块不自动折行：超宽时横向平移查看（默认折行）。
	CodeNoWrap bool
	// OnLink 链接点击回调；nil 时用 win32.ShellOpen 打开。
	OnLink func(url string)
	// NoTOC 不渲染 [toc] 块。
	NoTOC bool
}

// ---- 主题 ----
//
// Markdown 自带一套色板（正文/引用/行内代码/代码块/语法高亮），
// applyTheme 时调 mdApplyTheme 按当前主题重算，所以切浅色后正文、
// 代码块、表格、提示框都会跟着变白，不再黑底白字。

var (
	mdColText    = "#dfe6f5"
	mdColDim     = "#b3bed6"
	mdColMuted   = "#7f8aa6"
	mdColAccent  = "#a5b4fc"
	mdColLink    = "#8ab4ff"
	mdColBorder  = "#243049"
	mdColBgSoft  = "#111a2b"
	mdColQuoteBg = "#0f1828"
	mdColInline  = "#1a2337"
	mdColMark    = "#4c5a92"
	mdColCodeBg  = "#0b121e"

	// 代码语法高亮
	mdSynKw   = "#c792ea"
	mdSynStr  = "#a5d6a7"
	mdSynCom  = "#5f6f8f"
	mdSynNum  = "#f0b37e"
	mdSynFn   = "#82aaff"
	mdSynTyp  = "#ffcb6b"
	mdSynBi   = "#89ddff"
	mdSynAttr = "#ffcb6b"
	mdSynTag  = "#f07178"
	mdSynPunc = "#93a1bd"
	mdSynText = "#d5deee"
)

// mdApplyTheme 根据主题刷新 Markdown 色板（由 applyTheme 调用）。
func mdApplyTheme(t Theme) {
	if t.Name == "light" {
		mdColText = "#1f2733"
		mdColDim = "#475569"
		mdColMuted = "#7c8aa0"
		mdColAccent = "#4f46e5"
		mdColLink = "#0369a1"
		mdColBorder = "#dbe2ee"
		mdColBgSoft = "#f2f5fb"
		mdColQuoteBg = "#f6f9fd"
		mdColInline = "#eef2f9"
		mdColMark = "#94a3b8"
		mdColCodeBg = "#f6f8fc"

		mdSynKw = "#cf222e"
		mdSynStr = "#0a7d33"
		mdSynCom = "#6e7781"
		mdSynNum = "#0550ae"
		mdSynFn = "#8250df"
		mdSynTyp = "#0550ae"
		mdSynBi = "#953800"
		mdSynAttr = "#0550ae"
		mdSynTag = "#cf222e"
		mdSynPunc = "#57606a"
		mdSynText = "#1f2328"
		return
	}

	mdColText = "#dfe6f5"
	mdColDim = "#b3bed6"
	mdColMuted = "#7f8aa6"
	mdColAccent = "#a5b4fc"
	mdColLink = "#8ab4ff"
	mdColBorder = "#243049"
	mdColBgSoft = "#111a2b"
	mdColQuoteBg = "#0f1828"
	mdColInline = "#1a2337"
	mdColMark = "#4c5a92"
	mdColCodeBg = "#0b121e"

	mdSynKw = "#c792ea"
	mdSynStr = "#a5d6a7"
	mdSynCom = "#5f6f8f"
	mdSynNum = "#f0b37e"
	mdSynFn = "#82aaff"
	mdSynTyp = "#ffcb6b"
	mdSynBi = "#89ddff"
	mdSynAttr = "#ffcb6b"
	mdSynTag = "#f07178"
	mdSynPunc = "#93a1bd"
	mdSynText = "#d5deee"
}

// ---- 交互状态（UI 线程独占） ----

// mdStateRev 交互状态版本号：页签切换 / 折叠展开 / 主题变化都会 +1。
// 这些状态会影响 renderBlocks 的输出，渲染树缓存用它做失效判断
// （渲染树是复用的同一批节点，不能靠指针比较判断是否需要重建）。
var mdStateRev uint64

// mdTouchState 标记渲染树缓存失效。
func mdTouchState() { mdStateRev++ }

// MarkdownResetState 清空页签/折叠/图片预览等交互状态。
func MarkdownResetState() {
	mdState.tabs = map[string]int{}
	mdState.details = map[string]bool{}
	mdState.folded = map[string]bool{}
	mdState.checks = map[string]bool{}
	mdState.codeX = map[string]float32{}
	mdState.anchors = map[string]*Node{}
	mdState.toast = ""
	mdState.viewer.Open = false
	mdTouchState()
}

func mdToast(msg string) {
	mdState.toast = msg
	mdState.toastAt = AnimTime()
	RequestRedraw()
}

// mdOpenURL 打开外部链接。
// mdLinkHandler 当前 Markdown 实例的链接处理器（MarkdownOptions.OnLink，可为空）。
var mdLinkHandler func(string)

func mdOpenURL(url string) {
	if url == "" {
		return
	}
	if mdLinkHandler != nil {
		mdLinkHandler(url)
		return
	}
	if strings.HasPrefix(url, "#") {
		mdToast("锚点：" + strings.TrimPrefix(url, "#"))
		return
	}
	ShellOpen(url)
}

// ---- 字体与测量 ----

var mdFallbackFonts *fontCache

// mdFont 取字体：优先当前窗口的字体缓存（受 LRU 上限约束），
// 无窗口环境（LayoutForTest / 单测）退回包级缓存。
func mdFont(size float32, weight int32, family string) *text.Font {
	if a := Current(); a != nil && a.fonts != nil {
		return a.fonts.getFamily(size, weight, family)
	}
	if mdFallbackFonts == nil {
		mdFallbackFonts = newFontCache("", 1)
	}
	return mdFallbackFonts.getFamily(size, weight, family)
}

// ---- 行内排版 ----

// mdSpan 行内片段的最终样式。
type mdSpan struct {
	size   float32
	weight int32
	family string
	color  string
	bg     string
	padX   float32
	padY   float32
	link   string
	strike bool
	code   bool
	fnRef  string
}

// mdPiece 排版 token（词 / 单字 / 空格 / 公式 / 上下标）。
type mdPiece struct {
	text string
	sp   mdSpan
	br   bool
	// math 行内公式（已排版，宽度从盒里取）
	math *matBox
	// script 0=普通 1=上标 2=下标
	script int
}

// mdBaseSpan 正文默认样式。
func mdBaseSpan(o MarkdownOptions) mdSpan {
	size := o.BaseSize
	if size <= 0 {
		size = 13.5
	}
	return mdSpan{size: size, weight: 400, family: o.FontFamily, color: mdColText}
}

// mdRunSpan 把行内片段映射为样式。
func mdRunSpan(r mdInline, base mdSpan, o MarkdownOptions) mdSpan {
	sp := base
	if r.Bold {
		sp.weight = 700
	}
	if r.Italic {
		// 没有斜体字面：用更浅的字重 + 偏冷色近似（视觉上区别于正文）
		if sp.weight == 400 {
			sp.weight = 350
		}
		sp.color = mdColText
	}
	if r.Strike {
		sp.strike = true
		sp.color = mdColMuted
	}
	if r.Code {
		sp.code = true
		sp.family = mdMono(o)
		sp.size = base.size - 0.5
		sp.bg = mdColInline
		sp.padX, sp.padY = 3, 1
	}
	if r.Kbd {
		sp.code = true
		sp.family = mdMono(o)
		sp.size = base.size - 0.5
		sp.bg = mdColInline
		sp.padX, sp.padY = 5, 1
	}
	if r.Mark {
		sp.bg = mdColMark
		sp.padX = 2
	}
	if r.Link {
		sp.color = mdColLink
		sp.link = r.Href
	}
	return sp
}

func mdMono(o MarkdownOptions) string {
	if o.MonoFamily != "" {
		return o.MonoFamily
	}
	return "Consolas"
}

// mdIsRemote 是否远程图片地址。
func mdIsRemote(src string) bool {
	s := strings.ToLower(src)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// mdImagePath 把图片来源解析为本地路径（远程地址先下载到临时缓存）。
func mdImagePath(src string) (string, bool) {
	if mdIsRemote(src) {
		return mdRemoteImagePath(src)
	}
	return src, true
}

// mdPieceWidth 单个片段宽度（含内边距）。
func mdPieceWidth(p mdPiece) float32 {
	if p.math != nil {
		return p.math.w
	}
	if p.text == "" {
		return 0
	}
	f := mdFont(p.sp.size, p.sp.weight, p.sp.family)
	if f == nil {
		return 0
	}
	return f.Measure(p.text) + p.sp.padX*2
}

// mdTokens 把行内片段切成可断行的 token（空格单独成 token，CJK 逐字可断）。
func mdTokens(runs []mdInline, base mdSpan, o MarkdownOptions) []mdPiece {
	var out []mdPiece
	for _, r := range runs {
		if r.Br {
			out = append(out, mdPiece{br: true})
			continue
		}
		// 行内公式
		if r.Math != "" {
			if b, ok := mdMathBox(r.Math, base.size, base.family, base.color, false); ok {
				out = append(out, mdPiece{math: &b})
				continue
			}
			out = append(out, mdPiece{text: r.Math, sp: mdSpan{size: base.size, weight: 400, family: "Consolas", color: mdColDim, code: true, bg: mdColInline, padX: 3, padY: 1}})
			continue
		}
		// 上标 / 下标
		if r.Sup != "" || r.Sub != "" {
			sp := base
			sp.size = base.size * 0.72
			kind, txt := 1, r.Sup
			if r.Sub != "" {
				kind, txt = 2, r.Sub
			}
			out = append(out, mdPiece{text: txt, sp: sp, script: kind})
			continue
		}
		// 脚注引用（做成上标可点击）
		if r.FnRef != "" {
			sp := base
			sp.size = base.size * 0.7
			sp.color = mdColLink
			sp.link = "fn:" + r.FnRef
			out = append(out, mdPiece{text: "[" + r.FnRef + "]", sp: sp, script: 1})
			continue
		}
		sp := mdRunSpan(r, base, o)
		txt := r.Text
		if txt == "" {
			continue
		}
		// 行内代码 / kbd / 链接文本整体作为原子 token（过长时再拆）
		atomic := r.Code || r.Kbd
		rs := []rune(txt)
		var word []rune
		flushWord := func() {
			if len(word) > 0 {
				out = append(out, mdPiece{text: string(word), sp: sp})
				word = word[:0]
			}
		}
		for i := 0; i < len(rs); i++ {
			ch := rs[i]
			switch {
			case ch == ' ' || ch == '\t':
				flushWord()
				out = append(out, mdPiece{text: " ", sp: sp})
			case isWideRune(ch) && !atomic:
				flushWord()
				out = append(out, mdPiece{text: string(ch), sp: sp})
			default:
				word = append(word, ch)
			}
		}
		flushWord()
	}
	return out
}

// mdLayoutInline 贪心折行；宽度不足时对超长 token 逐字硬拆。
func mdLayoutInline(runs []mdInline, width float32, base mdSpan, o MarkdownOptions) [][]mdPiece {
	tokens := mdTokens(runs, base, o)
	var lines [][]mdPiece
	var cur []mdPiece
	var x float32

	flush := func() {
		// 去掉行尾空格
		for len(cur) > 0 && !cur[len(cur)-1].br && strings.TrimSpace(cur[len(cur)-1].text) == "" {
			cur = cur[:len(cur)-1]
		}
		if len(cur) > 0 {
			lines = append(lines, cur)
		}
		cur = nil
		x = 0
	}

	for _, tok := range tokens {
		if tok.br {
			flush()
			lines = append(lines, nil) // 空行保留（硬换行）
			continue
		}
		w := mdPieceWidth(tok)
		if x+w <= width || (len(cur) == 0 && w <= width) {
			cur = append(cur, tok)
			x += w
			continue
		}
		if len(cur) == 0 {
			// 单 token 超宽：按字符拆成多行（首行之后逐段 flush）
			parts := mdSplitByWidth(tok, width)
			for i, part := range parts {
				cur = append(cur, part)
				x += mdPieceWidth(part)
				if i < len(parts)-1 {
					flush()
				}
			}
			continue
		}
		flush()
		// 换行后重新放置
		if w <= width {
			cur = append(cur, tok)
			x = w
		} else {
			parts := mdSplitByWidth(tok, width)
			for i, part := range parts {
				if i == 0 {
					cur = append(cur, part)
					x += mdPieceWidth(part)
				} else {
					flush()
					cur = append(cur, part)
					x = mdPieceWidth(part)
				}
			}
		}
	}
	flush()
	// 去掉开头的空格 token
	for i := range lines {
		for len(lines[i]) > 0 && strings.TrimSpace(lines[i][0].text) == "" {
			lines[i] = lines[i][1:]
		}
	}
	return lines
}

// mdSplitByWidth 把超宽 token 拆成多段（逐字符累积到宽度）。
func mdSplitByWidth(p mdPiece, width float32) []mdPiece {
	rs := []rune(p.text)
	var out []mdPiece
	var buf []rune
	var w float32
	for _, r := range rs {
		one := mdPiece{text: string(r), sp: p.sp}
		rw := mdPieceWidth(one)
		if w+rw > width && len(buf) > 0 {
			out = append(out, mdPiece{text: string(buf), sp: p.sp})
			buf = buf[:0]
			w = 0
		}
		buf = append(buf, r)
		w += rw
	}
	if len(buf) > 0 {
		out = append(out, mdPiece{text: string(buf), sp: p.sp})
	}
	return out
}

// mdPieceNode 把一个排版片段转成节点。
func mdPieceNode(p mdPiece) *Node {
	// 行内公式：直接返回固定尺寸的排版盒
	if p.math != nil {
		return matBoxNode(*p.math)
	}
	if p.text == "" {
		return nil
	}
	st := St().Text(p.sp.size, p.sp.weight, p.sp.color)
	if p.sp.family != "" {
		st.Font(p.sp.family)
	}
	if p.sp.bg != "" {
		st.BgC(p.sp.bg).R(4)
		if p.sp.padX > 0 || p.sp.padY > 0 {
			st.PadOnly(p.sp.padY, p.sp.padX, p.sp.padY, p.sp.padX)
		}
	}
	n := Text(p.text).St(st)
	if p.sp.link != "" {
		link := p.sp.link
		if strings.HasPrefix(link, "fn:") {
			id := strings.TrimPrefix(link, "fn:")
			n.Tap(func() { mdScrollToAnchor("fn-" + id) })
			TooltipOf(n, "查看脚注 "+id)
		} else {
			n.Tap(func() { mdOpenURL(link) })
			TooltipOf(n, link)
		}
	}
	if p.sp.strike {
		w := mdPieceWidth(mdPiece{text: p.text, sp: p.sp})
		line := Box().St(St().W(w).H(1).BgC(mdColMuted).AbsCenter())
		n = Stack(n, line)
	}
	// 上标 / 下标：外框固定高度，文字用绝对定位上下微移
	if p.script != 0 {
		w := mdPieceWidth(p)
		ih := p.sp.size * 1.2
		dy := -p.sp.size * 0.46
		h := p.sp.size * 0.95
		if p.script == 2 {
			dy = p.sp.size * 0.30
			h = p.sp.size * 1.15
		}
		n.St(St().AbsTL(0, dy).W(w).H(ih).Lh(ih))
		return Stack(n).St(St().W(w + 0.5).H(h))
	}
	return n
}

// mdLineNode 一行片段 → Row（不做二次折行）。
func mdLineNode(pieces []mdPiece) *Node {
	var kids []*Node
	for _, p := range pieces {
		if p.text == "" {
			continue
		}
		if n := mdPieceNode(p); n != nil {
			kids = append(kids, n)
		}
	}
	if len(kids) == 0 {
		return nil
	}
	return Row(kids...).AlignTo(AlignStart)
}

// mdInlineNodes 行内片段 → 多行节点。
func mdInlineNodes(runs []mdInline, width float32, base mdSpan, o MarkdownOptions, lineGap float32) []*Node {
	lines := mdLayoutInline(runs, width, base, o)
	out := make([]*Node, 0, len(lines))
	for _, ln := range lines {
		if n := mdLineNode(ln); n != nil {
			out = append(out, n)
		}
	}
	return out
}

// ---- 渲染器 ----

type mdRenderer struct {
	doc   *mdDoc
	opts  MarkdownOptions
	width float32
	base  mdSpan
	key   string
}

// Markdown 渲染 Markdown 文档。
//
// 渲染树按 (窗口, 节点 Key, 源文, 宽度, 渲染缩放, 主题, 交互状态版本, 排版选项) 缓存：
// 旧实现每帧都把整篇文档重新排版成上千个节点（字符串拼接 + 语法高亮 + 文本测量），
// 长文档滚动时这一项就吃掉大半帧预算。缓存后每帧只做一次 map 查找 + 复制一层子节点指针。
//
// 复用同一批 *Node 是安全的：布局每帧都会重写 Frame/Parent/wrapW/ContentH，
// 悬停/按下由 App 按指针写入、本库不读取 Node.Hover/Pressed，
// 交互状态（页签/折叠/详情）通过 mdStateRev 参与缓存键。
func Markdown(key, src string, opts ...MarkdownOptions) *Node {
	o := MarkdownOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	base := mdBaseSpan(o)
	w := o.Width
	if w <= 0 {
		w = ViewportW() - 96
		if w > 860 {
			w = 860
		}
		if w < 240 {
			w = 240
		}
	}
	app := Current()
	mk := mdTreeKey{
		app:    app,
		key:    key,
		src:    src,
		width:  int32(w*2 + 0.5),
		scale:  int32(mdScaleOf(app)*100 + 0.5),
		theme:  curTheme.Name,
		rev:    mdStateRev,
		base:   int32(base.size*2 + 0.5),
		line:   int32(o.LineHeight*2 + 0.5),
		maxImg: int32(o.MaxImageHeight*2 + 0.5),
		codes:  o.CodeMaxLines,
		family: o.FontFamily,
		mono:   o.MonoFamily,
		noTOC:  o.NoTOC,
	}

	kids, hit := mdTreeCache[mk]
	if !hit {
		prevLink := mdLinkHandler
		if o.OnLink != nil {
			mdLinkHandler = o.OnLink
		}
		r := &mdRenderer{doc: mdParseCached(src), opts: o, width: w, base: base, key: key}
		kids = r.renderBlocks(r.doc.Blocks)
		// 文末脚注区
		if fn := r.footnotes(); fn != nil {
			kids = append(kids, Box().St(St().H(8)), fn)
		}
		mdLinkHandler = prevLink
		// 超大文档不缓存：避免一篇几万节点的文档把渲染树常驻在内存里
		if mdCountNodes(kids) <= mdTreeCacheMaxNodes {
			if len(mdTreeCache) >= mdTreeCacheMaxKeys {
				mdTreeCache = map[mdTreeKey][]*Node{}
			}
			mdTreeCache[mk] = kids
		}
	}

	// 每帧复制一层子节点指针：返回的 Children 是独立切片，
	// 追加 toast 或下游布局都不会污染缓存
	out := make([]*Node, 0, len(kids)+1)
	out = append(out, kids...)
	if t := mdToastNode(); t != nil {
		out = append(out, t)
	}
	return Column(out...).St(St().W(w).GapOf(0)).K(key)
}

// ---- 渲染树缓存 ----

// mdTreeKey 渲染树缓存键。任何影响排版结果的输入都必须在此出现，
// 否则会出现「切主题/改宽度/换页签后画面不更新」。
type mdTreeKey struct {
	app    *App // 多窗口隔离：节点 Frame/Parent 是每窗口各自的布局状态
	key    string
	src    string
	width  int32 // 量化到 0.5px，避免浮点抖动导致永不命中
	scale  int32 // 渲染缩放 ×100：字形度量参与折行/列宽计算
	theme  string
	rev    uint64 // mdStateRev：页签/折叠/详情等交互状态
	base   int32
	line   int32
	maxImg int32
	codes  int
	family string
	mono   string
	noTOC  bool
}

const (
	// mdTreeCacheMaxKeys 缓存文档数上限（超出整体丢弃）。
	mdTreeCacheMaxKeys = 6
	// mdTreeCacheMaxNodes 单篇文档节点数上限（超过则不缓存）。
	mdTreeCacheMaxNodes = 6000
)

var mdTreeCache = map[mdTreeKey][]*Node{}

// mdDropTreeCache 丢弃某个窗口的渲染树缓存（窗口销毁时调用）。
// 缓存键里有 *App，不清掉会让已关闭窗口的节点树与 App 一直常驻。
func mdDropTreeCache(a *App) {
	if a == nil || len(mdTreeCache) == 0 {
		return
	}
	for k := range mdTreeCache {
		if k.app == a {
			delete(mdTreeCache, k)
		}
	}
}

// mdScaleOf 取窗口的渲染缩放（无窗口环境按 1 处理，保证单测稳定）。
func mdScaleOf(a *App) float32 {
	if a == nil {
		return 1
	}
	if s := a.effScale(); s > 0 {
		return s
	}
	return 1
}

// mdCountNodes 统计子树节点数（仅缓存未命中时调用一次）。
func mdCountNodes(ns []*Node) int {
	total := 0
	var walk func(n *Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		total++
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, n := range ns {
		walk(n)
	}
	return total
}

// ---- 解析缓存（同一内容不重复解析，流式输出时收益明显） ----

func mdParseCached(src string) *mdDoc {
	if d, ok := mdParseCache.m[src]; ok {
		return d
	}
	d := parseMarkdown(src)
	if len(mdParseCache.m) >= mdParseCache.max {
		// 简单淘汰：清空（文档级缓存，命中率主要来自同一篇内容的重复渲染）
		mdParseCache.m = map[string]*mdDoc{}
	}
	mdParseCache.m[src] = d
	return d
}

// MarkdownParse 解析但不渲染（调试/目录提取用）。
func MarkdownParse(src string) *mdDoc { return parseMarkdown(src) }

// ---- 块渲染 ----

func (r *mdRenderer) renderBlocks(blocks []*mdBlock) []*Node {
	out := make([]*Node, 0, len(blocks)*2)
	for i, b := range blocks {
		n := r.renderBlock(b, i)
		if n == nil {
			continue
		}
		if len(out) > 0 {
			out = append(out, Box().St(St().H(mdBlockGap(b))))
		}
		out = append(out, n)
	}
	return out
}

// mdBlockGap 块前间距。
func mdBlockGap(b *mdBlock) float32 {
	switch b.Kind {
	case mdHeading:
		if b.Level <= 2 {
			return 18
		}
		return 14
	case mdCode, mdTable, mdTabs, mdDetails, mdCard, mdGallery, mdMath, mdMermaid, mdDefList:
		return 12
	case mdHR:
		return 10
	}
	return 6
}

func (r *mdRenderer) renderBlock(b *mdBlock, idx int) *Node {
	switch b.Kind {
	case mdHeading:
		return r.heading(b, idx)
	case mdParagraph:
		return r.paragraph(b)
	case mdCode:
		return r.codeBlock(b, idx)
	case mdList:
		return r.list(b, idx, 0)
	case mdQuote:
		return r.quote(b, idx)
	case mdHR:
		return Divider()
	case mdImage:
		return r.image(b)
	case mdTable:
		return r.table(b)
	case mdTabs:
		return r.tabs(b, idx)
	case mdDetails:
		return r.details(b, idx)
	case mdCard:
		return r.card(b, idx)
	case mdBadge:
		return r.badge(b)
	case mdProgress:
		return r.progress(b)
	case mdGallery:
		return r.gallery(b, idx)
	case mdMedia:
		return r.media(b)
	case mdTOC:
		if r.opts.NoTOC {
			return nil
		}
		return r.toc()
	case mdMath:
		return r.mathBlock(b)
	case mdMermaid:
		return r.mermaid(b)
	case mdDefList:
		return r.defList(b)
	}
	return nil
}

// mathBlock 块级公式：居中展示 + 复制源码。
func (r *mdRenderer) mathBlock(b *mdBlock) *Node {
	src := strings.TrimSpace(b.Math)
	if src == "" {
		return nil
	}
	size := r.base.size + 1.5
	node := mdMathTypeset(src, size, r.base.family, mdColText, true)
	if node == nil {
		// 排版失败（语法过偏）：退化为等宽代码块，至少内容不丢
		fallback := &mdBlock{Kind: mdCode, Lang: "latex", Code: src}
		return r.codeBlock(fallback, -1)
	}
	copyBtn := Text("复制").St(St().Text(10.5, 500, mdColMuted).PadXY(7, 3).R(6).BgC(ColHover))
	code := src
	copyBtn.Tap(func() {
		if ClipboardSetText(code) {
			mdToast("公式已复制")
		} else {
			mdToast("复制失败")
		}
	})
	return Column(
		Row(Spacer(), copyBtn).St(St().PadOnly(6, 10, 0, 0)),
		Row(node).Grow(1).JustifyTo(AlignCenter).St(St().Padding(12)),
	).St(St().R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder))
}

// mermaid 图表块：GoGUI 没有 JS 运行环境，降级为「图表提示卡」，
// 展示图类型、行数与源码（可复制），并提示到浏览器版本查看。
func (r *mdRenderer) mermaid(b *mdBlock) *Node {
	kind, desc := mdMermaidKind(b.Code)
	// flowchart / graph：用原生节点 + 正交连线真正画出来
	if g, ok := mmParse(b.Code); ok {
		if view, ok2 := mmView(g, r.width-24, r.base.size+0.5, r.opts.FontFamily,
			r.key+"-mm-"+strconv.Itoa(len(b.Code))); ok2 {
			copy2 := Text("复制源码").St(St().Text(11, 500, mdColAccent).PadXY(9, 4).R(7).
				BgC("rgba(129,140,248,0.12)"))
			src := b.Code
			copy2.Tap(func() {
				if ClipboardSetText(src) {
					mdToast("Mermaid 源码已复制")
				} else {
					mdToast("复制失败")
				}
			})
			return Column(
				Row(
					Text("◇").St(St().Text(14, 600, "#67e8f9")),
					Text(kind).St(St().Text(r.base.size-0.5, 600, mdColText)),
					Text("· "+strconv.Itoa(len(g.nodes))+" 节点 / "+strconv.Itoa(len(g.edges))+" 连线").
						St(St().Text(r.base.size-2, 400, mdColMuted)),
					Spacer(),
					copy2,
				).St(St().GapOf(8)).AlignTo(AlignCenter),
				Text(desc).St(St().Text(r.base.size-2, 400, mdColMuted).WrapOn().PadOnly(4, 0, 0, 0)),
				Row(view).JustifyTo(AlignCenter).St(St().PadOnly(8, 0, 2, 0)),
			).St(St().Padding(12).GapOf(6).R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder))
		}
	}
	lines := strings.Count(strings.TrimSpace(b.Code), "\n") + 1
	code := b.Code
	copyBtn := Text("复制源码").St(St().Text(11, 500, mdColAccent).PadXY(9, 4).R(7).
		BgC("rgba(129,140,248,0.12)"))
	copyBtn.Tap(func() {
		if ClipboardSetText(code) {
			mdToast("Mermaid 源码已复制")
		} else {
			mdToast("复制失败")
		}
	})
	// 源码预览：最多 8 行
	preview := strings.Split(strings.TrimSpace(b.Code), "\n")
	if len(preview) > 8 {
		preview = append(preview[:8], "…")
	}
	var codeLines []*Node
	for _, l := range preview {
		codeLines = append(codeLines, Text(l).St(St().Text(r.base.size-2, 400, mdColMuted).
			Font(mdMono(r.opts)).WrapOn()))
	}
	return Column(
		Row(
			Text("◇").St(St().Text(14, 600, "#67e8f9")),
			Text(kind).St(St().Text(r.base.size-0.5, 600, mdColText)),
			Text("· "+strconv.Itoa(lines)+" 行").St(St().Text(r.base.size-2, 400, mdColMuted)),
			Spacer(),
			copyBtn,
		).St(St().GapOf(8)).AlignTo(AlignCenter),
		Text(desc).St(St().Text(r.base.size-2, 400, mdColMuted).WrapOn().PadOnly(4, 0, 0, 0)),
		Column(codeLines...).St(St().GapOf(1).PadOnly(8, 10, 8, 10).R(8).BgC(mdColCodeBg).Clipped()),
		Text("桌面端原生绘制 flowchart / graph；时序图、类图、甘特图请复制源码到 mermaid.live 查看").St(
			St().Text(r.base.size-2.5, 400, mdColMuted).WrapOn()),
	).St(St().Padding(12).GapOf(8).R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder))
}

// mdMermaidKind 识别 Mermaid 图类型（首行关键字）。
func mdMermaidKind(src string) (string, string) {
	first := ""
	for _, l := range strings.Split(src, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			first = strings.ToLower(t)
			break
		}
	}
	switch {
	case strings.HasPrefix(first, "graph"), strings.HasPrefix(first, "flowchart"):
		return "流程图 Flowchart", "节点与连线的有向/无向图"
	case strings.HasPrefix(first, "sequencediagram"):
		return "时序图 Sequence", "参与者之间的消息时序"
	case strings.HasPrefix(first, "classdiagram"):
		return "类图 Class", "类型、属性与继承关系"
	case strings.HasPrefix(first, "statediagram"):
		return "状态图 State", "状态机迁移"
	case strings.HasPrefix(first, "erdiagram"):
		return "实体关系 ER", "表结构与关联"
	case strings.HasPrefix(first, "gantt"):
		return "甘特图 Gantt", "任务排期与依赖"
	case strings.HasPrefix(first, "pie"):
		return "饼图 Pie", "占比构成"
	case strings.HasPrefix(first, "journey"):
		return "用户旅程 Journey", "分阶段体验流程"
	case strings.HasPrefix(first, "gitgraph"):
		return "Git 图", "分支与提交历史"
	case strings.HasPrefix(first, "mindmap"):
		return "思维导图 Mindmap", "中心主题的层级展开"
	}
	return "Mermaid 图表", "未识别的图类型，按源码展示"
}

// defList 定义列表。
func (r *mdRenderer) defList(b *mdBlock) *Node {
	rows := make([]*Node, 0, len(b.Defs)*2)
	for _, d := range b.Defs {
		term := Column(mdInlineNodes(d.Term, r.width-r.base.size*6, mdSpan{
			size: r.base.size, weight: 600, family: r.opts.FontFamily, color: mdColText,
		}, r.opts, 0)...).St(St().GapOf(mdLineGap(r.opts, r.base.size)))
		defs := make([]*Node, 0, len(d.Defs)*2)
		for i, def := range d.Defs {
			body := mdInlineNodes(def, r.width-r.base.size*7, r.base, r.opts, 0)
			defs = append(defs, Row(
				Text(mdDefMark(i)).St(St().Text(r.base.size, 500, mdColAccent).W(r.base.size*1.2)),
				Column(body...).Grow(1).St(St().GapOf(mdLineGap(r.opts, r.base.size))),
			).AlignTo(AlignStart))
		}
		rows = append(rows, term, Column(defs...).St(St().GapOf(2).PadOnly(3, 0, 5, r.base.size*1.4)))
	}
	return Column(rows...).St(St().GapOf(4))
}

func mdDefMark(i int) string {
	if i == 0 {
		return "："
	}
	return "·"
}

// heading 标题（字号/字重随级别变化，带锚点色条）。
func (r *mdRenderer) heading(b *mdBlock, idx int) *Node {
	size := r.base.size
	weight := int32(700)
	color := mdColText
	switch b.Level {
	case 1:
		size += 10.5
	case 2:
		size += 6.5
	case 3:
		size += 3.5
	case 4:
		size += 2
	case 5:
		weight = 600
		size += 1
	case 6:
		weight = 600
		size += 0.5
		color = mdColMuted
	}
	if b.Level == 1 {
		// h1 下面带一条渐隐分隔线
		kids := mdInlineNodes(b.Inl, r.width, mdSpan{size: size, weight: weight, family: r.opts.FontFamily, color: color}, r.opts, 0)
		return r.anchorNode(b, Column(kids...).St(St().GapOf(1)))
	}
	base := r.base
	base.size = size
	base.weight = weight
	base.color = color
	return r.anchorNode(b, Column(mdInlineNodes(b.Inl, r.width, base, r.opts, 0)...))
}

// anchorNode 给标题挂锚点 Key 并登记，供目录点击滚动定位。
func (r *mdRenderer) anchorNode(b *mdBlock, n *Node) *Node {
	if b.Key == "" {
		return n
	}
	n.K(mdAnchorID(r.key, b.Key))
	if mdState.anchors == nil {
		mdState.anchors = map[string]*Node{}
	}
	if len(mdState.anchors) > 4096 {
		mdState.anchors = map[string]*Node{}
	}
	mdState.anchors[mdAnchorID(r.key, b.Key)] = n
	return n
}

// mdAnchorID 锚点在全局表里的唯一键（同一篇文档多个实例共存时按渲染 Key 区分）。
func mdAnchorID(docKey, anchor string) string { return docKey + "|" + anchor }

// mdScrollToAnchor 把锚点节点滚动到最近的滚动容器顶部。
// 节点 Frame 是上一帧布局结果，点击回调发生在事件分发阶段，二者一致。
func mdScrollToAnchor(id string) {
	n := mdState.anchors[id]
	if n == nil {
		mdToast("锚点未找到")
		return
	}
	a := Current()
	if a == nil {
		return
	}
	var sc *Node
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Kind == KScroll && p.Key != "" {
			sc = p
			break
		}
	}
	if sc == nil {
		mdToast("当前不在滚动容器内")
		return
	}
	off := n.Frame.Y - sc.Frame.Y + a.scrolls[sc.Key] - 10
	if off < 0 {
		off = 0
	}
	max := sc.ContentH - sc.Frame.H
	if max < 0 {
		max = 0
	}
	a.setScrollTarget(sc.Key, off, max)
	a.scrollIdle = 0
	a.dirty = true
	RequestRedraw()
}

func (r *mdRenderer) paragraph(b *mdBlock) *Node {
	// 单独成段的裸链接 → 链接卡片
	if n := r.linkCard(b); n != nil {
		return n
	}
	kids := mdInlineNodes(b.Inl, r.width, r.base, r.opts, 0)
	return Column(kids...).St(St().GapOf(mdLineGap(r.opts, r.base.size)))
}

// linkCard 单独成段的 URL 渲染为链接卡（域名 + 路径 + 打开提示）。
func (r *mdRenderer) linkCard(b *mdBlock) *Node {
	url := ""
	for _, run := range b.Inl {
		if run.Br || run.FnRef != "" || run.Math != "" {
			return nil
		}
		href := strings.TrimSpace(run.Href)
		if run.Link && href != "" && strings.TrimSpace(run.Text) == href && mdLooseURLRe.MatchString(href) {
			if url != "" {
				return nil
			}
			url = href
			continue
		}
		if strings.TrimSpace(run.Text) != "" {
			return nil
		}
	}
	if url == "" {
		return nil
	}
	domain, path := mdSplitURL(url)
	row := Row(
		Text("🔗").St(St().Text(r.base.size+3, 400, mdColMuted).W(26).TAlign(AlignCenter)),
		Column(
			Text(domain).St(St().Text(r.base.size, 600, mdColLink)),
			Text(path).St(St().Text(r.base.size-2, 400, mdColMuted).WrapOn()),
		).Grow(1).St(St().GapOf(2)),
		Text("打开 ↗").St(St().Text(r.base.size-2, 500, mdColMuted)),
	).St(St().GapOf(10).Padding(11).R(9).BgC(mdColBgSoft).Stroke(1, mdColBorder)).
		AlignTo(AlignCenter).Tap(func() { mdOpenURL(url) })
	TooltipOf(row, "在浏览器中打开 "+url)
	return row
}

// mdSplitURL 拆出域名与路径（不引入 net/url，避免无谓分配）。
func mdSplitURL(u string) (string, string) {
	s := u
	for _, pre := range []string{"https://", "http://"} {
		s = strings.TrimPrefix(s, pre)
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		return s[:i], u
	}
	return s, u
}

// mdLineGap 行间距：优先用 MarkdownOptions.LineHeight，否则按字号推算。
func mdLineGap(o MarkdownOptions, size float32) float32 {
	if o.LineHeight > size {
		if g := o.LineHeight - size*1.45; g > 1 {
			return g
		}
	}
	g := size * 0.55
	if g < 4 {
		g = 4
	}
	return g
}

// codeBlock 代码块：语言标签 + 复制 + 折叠 + 行号 + 轻量着色。
func (r *mdRenderer) codeBlock(b *mdBlock, idx int) *Node {
	key := r.key + "-code-" + strconv.Itoa(idx)
	lines := mdHighlight(b.Lang, b.Code)

	maxLines := r.opts.CodeMaxLines
	if maxLines == 0 {
		maxLines = 22
	}
	folded := false
	if maxLines > 0 && len(lines) > maxLines && !mdState.folded[key] {
		folded = true
	}
	shown := lines
	if folded {
		shown = lines[:maxLines]
	}
	if len(shown) == 0 {
		shown = [][]mdHLSpan{{}}
	}

	size := r.base.size - 1
	if size < 11 {
		size = 11
	}
	mono := mdMono(r.opts)
	monoFont := mdFont(size, 400, mono)
	lineH := size * 1.55

	// 行号列宽
	digits := len(strconv.Itoa(len(lines)))
	numW := float32(digits)*size*0.62 + 14
	if monoFont != nil {
		numW = monoFont.Measure(strings.Repeat("0", digits)) + 16
	}
	codeAvail := r.width - numW - 28
	codeW := codeAvail
	if r.opts.CodeNoWrap {
		// 不折行：按最长行撑开宽度，超出的部分通过横向平移查看
		maxLine := float32(0)
		for _, spans := range shown {
			var lw float32
			for _, s := range spans {
				if f := mdFont(size, 400, mono); f != nil {
					lw += f.Measure(s.Text)
				}
			}
			if lw > maxLine {
				maxLine = lw
			}
		}
		if maxLine > codeW {
			codeW = maxLine + 2
		}
	}

	var rows []*Node
	for i, spans := range shown {
		num := Text(strconv.Itoa(i + 1)).St(St().Text(size-0.5, 400, mdColMuted).Font(mono).W(numW).TAlign(AlignEnd).Lh(lineH))
		code := r.codeLine(spans, codeW, mono, size, lineH)
		rowSt := St().GapOf(10)
		switch mdDiffKind(spans) {
		case "add":
			rowSt.BgC("rgba(52,211,153,0.10)")
		case "del":
			rowSt.BgC("rgba(248,113,113,0.10)")
		case "meta":
			rowSt.BgC("rgba(56,189,248,0.08)")
		}
		rows = append(rows, Row(num, code).St(rowSt).AlignTo(AlignStart))
	}

	body := Column(rows...).St(St().GapOf(1).PadXY(14, 12))

	// 横向平移（不折行模式）：内容比可视区宽时，用头部左右按钮/拖动移动
	panKey := key + "-x"
	maxPan := codeW - codeAvail
	if r.opts.CodeNoWrap && maxPan > 0 {
		x := mdState.codeX[panKey]
		if x > maxPan {
			x = maxPan
		}
		if x < 0 {
			x = 0
		}
		bodyH := float32(0)
		for range rows {
			bodyH += lineH + 1
		}
		inner := body.St(St().AbsTL(-x, 0).W(codeW + 28))
		view := Box(inner).St(St().W(codeAvail + 28).H(bodyH + 2).Clipped())
		// 拖动平移：按下后按水平位移比例移动
		view.DragEvt(func(px, _ float32) {
			if codeAvail <= 0 {
				return
			}
			nx := x + (px/maxPan-1)*4
			if nx < 0 {
				nx = 0
			}
			if nx > maxPan {
				nx = maxPan
			}
			mdState.codeX[panKey] = nx
			RequestRedraw()
		})
		body = view
	}

	// 头部：语言 + 复制 + 折叠
	langLabel := strings.ToUpper(strings.TrimSpace(b.Lang))
	if langLabel == "" {
		langLabel = "TEXT"
	}
	copyBtn := Text("复制").St(St().Text(11, 500, mdColMuted).PadXY(8, 3).R(6).
		BgC(ColHover).Stroke(1, ColHover))
	code := b.Code
	copyBtn.Tap(func() {
		if ClipboardSetText(code) {
			mdToast("代码已复制")
		} else {
			mdToast("复制失败")
		}
	})
	headKids := []*Node{
		Text(langLabel).St(St().Text(10.5, 600, mdColMuted).Font(mono)),
		Spacer(),
	}
	if r.opts.CodeNoWrap && codeW > codeAvail {
		pan := func(delta float32) {
			x := mdState.codeX[panKey] + delta
			maxPan2 := codeW - codeAvail
			if x < 0 {
				x = 0
			}
			if x > maxPan2 {
				x = maxPan2
			}
			mdState.codeX[panKey] = x
			RequestRedraw()
		}
		l := Text("‹").St(St().Text(12, 700, mdColMuted).PadXY(7, 2).R(6).BgC(ColHover))
		l.Tap(func() { pan(-codeAvail * 0.6) })
		rr := Text("›").St(St().Text(12, 700, mdColMuted).PadXY(7, 2).R(6).BgC(ColHover))
		rr.Tap(func() { pan(codeAvail * 0.6) })
		headKids = append(headKids, l, rr)
	}
	if folded {
		n := len(lines) - maxLines
		expand := Text("展开 " + strconv.Itoa(n) + " 行").St(St().Text(11, 500, mdColAccent).PadXY(8, 3).R(6).
			BgC("rgba(129,140,248,0.10)"))
		k := key
		expand.Tap(func() {
			mdState.folded[k] = true
			mdTouchState()
			RequestRedraw()
		})
		headKids = append(headKids, expand)
	} else if maxLines > 0 && len(lines) > maxLines {
		collapse := Text("收起").St(St().Text(11, 500, mdColMuted).PadXY(8, 3).R(6).
			BgC(ColHover))
		k := key
		collapse.Tap(func() {
			delete(mdState.folded, k)
			mdTouchState()
			RequestRedraw()
		})
		headKids = append(headKids, collapse)
	}
	headKids = append(headKids, copyBtn)
	head := Row(headKids...).St(St().GapOf(8).PadXY(14, 7).BgC(mdColBgSoft)).AlignTo(AlignCenter)

	return Column(head, Divider(), body).St(St().R(10).Stroke(1, mdColBorder).BgC(mdColCodeBg).Clipped())
}

// codeLine 一行代码（超宽自动折行，保留缩进）。
func (r *mdRenderer) codeLine(spans []mdHLSpan, maxW float32, mono string, size, lineH float32) *Node {
	// 先测总宽
	var total float32
	for _, s := range spans {
		f := mdFont(size, 400, mono)
		if f != nil {
			total += f.Measure(s.Text)
		}
	}
	if total <= maxW || maxW <= 0 {
		var kids []*Node
		for _, s := range spans {
			if s.Text == "" {
				continue
			}
			kids = append(kids, Text(s.Text).St(St().Text(size, 400, mdCodeColor(s.Kind)).Font(mono).Lh(lineH)))
		}
		if len(kids) == 0 {
			kids = append(kids, Text(" ").St(St().Text(size, 400, mdColMuted).Font(mono).Lh(lineH)))
		}
		return Row(kids...).AlignTo(AlignStart)
	}

	// 超宽：按字符累积成多行
	var out []*Node
	var cur []*Node
	var curText strings.Builder
	var x float32
	flush := func() {
		if curText.Len() > 0 {
			out = append(out, Row(cur...).AlignTo(AlignStart))
			cur = nil
			curText.Reset()
			x = 0
		}
	}
	for _, s := range spans {
		runes := []rune(s.Text)
		var buf []rune
		flushBuf := func() {
			if len(buf) > 0 {
				t := string(buf)
				cur = append(cur, Text(t).St(St().Text(size, 400, mdCodeColor(s.Kind)).Font(mono).Lh(lineH)))
				curText.WriteString(t)
				buf = buf[:0]
			}
		}
		for _, ch := range runes {
			f := mdFont(size, 400, mono)
			var adv float32
			if f != nil {
				adv = f.GlyphFor(ch).Advance / f.Scale()
			}
			if x+adv > maxW && (len(cur) > 0 || len(buf) > 0) {
				flushBuf()
				flush()
			}
			buf = append(buf, ch)
			x += adv
		}
		flushBuf()
	}
	flush()
	if len(out) == 0 {
		out = append(out, Text(" ").St(St().Text(size, 400, mdColMuted).Font(mono).Lh(lineH)))
	}
	return Column(out...).St(St().GapOf(0))
}

func mdCodeColor(kind string) string {
	switch kind {
	case "kw":
		return mdSynKw
	case "str":
		return mdSynStr
	case "com":
		return mdSynCom
	case "num":
		return mdSynNum
	case "fn":
		return mdSynFn
	case "typ":
		return mdSynTyp
	case "bi":
		return mdSynBi
	case "attr":
		return mdSynAttr
	case "tag":
		return mdSynTag
	case "punc":
		return mdSynPunc
	case "add":
		return "#7ee2a8"
	case "del":
		return "#ff9aa2"
	case "meta":
		return mdSynBi
	}
	return mdSynText
}

// mdDiffKind 取整行的 diff 语义（用于行底色）。
func mdDiffKind(spans []mdHLSpan) string {
	if len(spans) == 0 {
		return ""
	}
	switch spans[0].Kind {
	case "add", "del", "meta":
		return spans[0].Kind
	}
	return ""
}

// list 列表（支持嵌套、有序、任务列表）。
func (r *mdRenderer) list(b *mdBlock, idx, depth int) *Node {
	items := make([]*Node, 0, len(b.Items))
	for i, it := range b.Items {
		marker := "•"
		if b.Ordered {
			marker = strconv.Itoa(b.Start+i) + "."
		} else {
			switch depth {
			case 1:
				marker = "◦"
			case 2:
				marker = "▪"
			}
		}
		var mk *Node
		if it.Task >= 0 {
			tk := r.key + "-task-" + strconv.Itoa(idx) + "-" + strconv.Itoa(i)
			checked := it.Task == 1
			if v, ok := mdState.checks[tk]; ok {
				checked = v
			}
			box := ColHover
			mark := ""
			if checked {
				box = mdColAccent
				mark = "√"
			}
			mk = Stack(
				Box().St(St().AbsFill().R(4).BgC(box).Stroke(1, mdColBorder)),
				Text(mark).St(St().AbsFill().Text(10, 700, "#ffffff").CenterYOn().TAlign(AlignCenter)),
			).St(St().W(14).H(14).MarOnly(4, 6, 0, 6)).Tap(func() {
				if mdState.checks == nil {
					mdState.checks = map[string]bool{}
				}
				mdState.checks[tk] = !checked
				mdTouchState()
				RequestRedraw()
			})
		} else {
			mk = Text(marker).St(St().Text(r.base.size, 500, mdColAccent).W(22).TAlign(AlignEnd).
				Lh(r.base.size*1.65).MarOnly(0, 8, 0, 0))
		}
		contentW := r.width - 30
		kids := mdInlineNodes(it.Inl, contentW, r.base, r.opts, 0)
		col := Column(kids...).St(St().GapOf(2))
		for _, sub := range it.Body {
			n := r.renderBlock(sub, idx*100+i)
			if n != nil {
				col.Add(n)
			}
		}
		row := Row(mk, col.Grow(1)).St(St().GapOf(0)).AlignTo(AlignStart)
		if depth > 0 {
			row.St(St().PadOnly(0, 0, 0, 0))
		}
		items = append(items, row)
	}
	return Column(items...).St(St().GapOf(5))
}

// quote 引用 / 提示块。
func (r *mdRenderer) quote(b *mdBlock, idx int) *Node {
	contentW := r.width - 18
	inner := &mdRenderer{doc: r.doc, opts: r.opts, width: contentW, base: r.base, key: r.key + "-q" + strconv.Itoa(idx)}
	kids := inner.renderBlocks(b.Quote)

	barColor := mdColBorder
	bg := "transparent"
	icon := ""
	switch b.Accent {
	case "tip":
		barColor, bg, icon = "#34d399", "rgba(52,211,153,0.07)", "💡"
	case "warning":
		barColor, bg, icon = "#fbbf24", "rgba(251,191,36,0.07)", "⚠️"
	case "success":
		barColor, bg, icon = "#34d399", "rgba(52,211,153,0.07)", "✅"
	case "error":
		barColor, bg, icon = "#f87171", "rgba(248,113,113,0.07)", "❌"
	case "important":
		barColor, bg, icon = "#a78bfa", "rgba(167,139,250,0.07)", "🎯"
	case "note":
		barColor, bg, icon = "#38bdf8", "rgba(56,189,248,0.07)", "📝"
	}

	body := Column(kids...).St(St().GapOf(0))
	if icon != "" {
		body = Column(
			Row(Text(icon).St(St().Text(r.base.size+1, 400, mdColText)),
				Column(kids...).Grow(1).St(St().GapOf(0)),
			).St(St().GapOf(8)).AlignTo(AlignStart),
		)
	}
	bar := Box().St(St().W(3).BgC(barColor).R(2))
	row := Row(bar, body.Grow(1)).St(St().GapOf(14)).AlignTo(AlignStart)
	st := St().PadOnly(10, 14, 10, 14).R(8)
	if bg != "transparent" {
		st.BgC(bg)
	}
	return row.St(st)
}

// image 单张图片。
func (r *mdRenderer) image(b *mdBlock) *Node {
	return r.imageNode(b.Src, b.Alt, mdColText)
}

func (r *mdRenderer) imageNode(src, alt, captionColor string) *Node {
	maxH := r.opts.MaxImageHeight
	if maxH <= 0 {
		maxH = 360
	}
	local, ok := mdImagePath(src)
	if !ok {
		return r.imagePlaceholder(src, alt)
	}
	w0, h0 := ImageSize(local)
	if w0 <= 0 || h0 <= 0 {
		return r.imagePlaceholder(src, alt)
	}
	w, h := float32(w0), float32(h0)
	if h > maxH {
		w *= maxH / h
		h = maxH
	}
	if w > r.width {
		h *= r.width / w
		w = r.width
	}
	img := ImageSized(local, w, h).St(St().R(8))
	s := local
	a := alt
	img.Tap(func() {
		mdState.viewer.Open = true
		mdState.viewer.Src = s
		mdState.viewer.Alt = a
		RequestRedraw()
	})
	col := Column(img)
	if alt != "" {
		col.Add(Text(alt).St(St().Text(11.5, 400, captionColor).TAlign(AlignCenter).WrapOn()))
	}
	return col.St(St().GapOf(6))
}

// imagePlaceholder 无法显示的图片 → 占位卡（区分「下载中 / 失败 / 非本地路径」）。
func (r *mdRenderer) imagePlaceholder(src, alt string) *Node {
	hint := "GoGUI 仅支持本地图片（PNG / JPEG / GIF / BMP）"
	icon := "🖼"
	if mdIsRemote(src) {
		switch {
		case mdRemotePending(src):
			hint = "正在下载远程图片…（完成后自动刷新画面）"
			RequestRedraw()
		default:
			if msg := mdRemoteError(src); msg != "" {
				hint = "远程图片加载失败：" + msg
			} else {
				hint = "远程图片未就绪"
			}
		}
		icon = "🌐"
	}
	col := Column(
		Row(
			Text(icon).St(St().Text(13, 400, mdColMuted)),
			Text(alt).St(St().Text(11.5, 600, mdColMuted).WrapOn()),
		).St(St().GapOf(6)).AlignTo(AlignStart),
		Text(src).St(St().Text(10.5, 400, mdColMuted).WrapOn()),
		Text(hint).St(St().Text(10.5, 400, mdColMuted).WrapOn()),
	).St(St().GapOf(4).Padding(14).R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder))
	return col
}

// table 表格（列宽按内容比例分配，单元格内折行）。
func (r *mdRenderer) table(b *mdBlock) *Node {
	if b.Table == nil || len(b.Table.Header) == 0 {
		return nil
	}
	cols := len(b.Table.Header)
	padX := float32(10)
	desired := make([]float32, cols)
	for i, c := range b.Table.Header {
		desired[i] = mdPlainWidth(c.Inl, r.base, r.opts) + padX*2
	}
	for _, row := range b.Table.Rows {
		for i, c := range row {
			if i < cols {
				if v := mdPlainWidth(c.Inl, r.base, r.opts) + padX*2; v > desired[i] {
					desired[i] = v
				}
			}
		}
	}
	widths := mdDistributeWidths(desired, r.width, 64)

	// 纯数字列自动右对齐（含百分号/单位后缀也算数字）
	numCol := make([]bool, cols)
	for i := range numCol {
		numCol[i] = true
		seen := 0
		for _, row := range b.Table.Rows {
			if i >= len(row) {
				continue
			}
			txt := strings.TrimSpace(inlineText(row[i].Inl))
			if txt == "" {
				continue
			}
			seen++
			if !mdIsNumeric(txt) {
				numCol[i] = false
				break
			}
		}
		if seen == 0 {
			numCol[i] = false
		}
	}

	var rows []*Node
	headCells := make([]*Node, 0, cols)
	for i, c := range b.Table.Header {
		sp := r.base
		sp.weight = 600
		sp.color = mdColText
		align := c.Align
		if numCol[i] && align == AlignStart {
			align = AlignEnd
		}
		kids := mdInlineNodes(c.Inl, widths[i]-padX*2, sp, r.opts, 0)
		headCells = append(headCells, Column(kids...).St(St().W(widths[i]).PadXY(padX, 8).TAlign(align)))
	}
	rows = append(rows, Row(headCells...).St(St().BgC(mdColBgSoft)))

	for ri, row := range b.Table.Rows {
		bg := "transparent"
		if ri%2 == 1 {
			bg = ColHover
		}
		cells := make([]*Node, 0, cols)
		for i := range b.Table.Header {
			c := mdCell{Align: b.Table.Header[i].Align}
			if i < len(row) {
				c = row[i]
			}
			align := c.Align
			if numCol[i] && align == AlignStart {
				align = AlignEnd
			}
			kids := mdInlineNodes(c.Inl, widths[i]-padX*2, r.base, r.opts, 0)
			cells = append(cells, Column(kids...).St(St().W(widths[i]).PadXY(padX, 8).TAlign(align)))
		}
		rows = append(rows, Row(cells...).St(St().BgC(bg)))
		rows = append(rows, Divider())
	}
	if len(rows) > 2 {
		rows = rows[:len(rows)-1]
	}
	head := rows[0]
	body := Column(rows[1:]...)
	return Column(head, Divider(), body).St(St().R(8).Stroke(1, mdColBorder).Clipped())
}

// mdPlainWidth 行内文本自然宽度。
func mdPlainWidth(runs []mdInline, base mdSpan, o MarkdownOptions) float32 {
	var w float32
	for _, p := range mdTokens(runs, base, o) {
		w += mdPieceWidth(p)
	}
	return w
}

// mdIsNumeric 判断是否为纯数字（表格列右对齐用）。
func mdIsNumeric(s string) bool {
	if s == "" {
		return false
	}
	digits := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '.' || r == ',' || r == '%' || r == '+' || r == '-' || r == ' ' || r == '/':
			continue
		default:
			return false
		}
	}
	return digits > 0
}

// mdDistributeWidths 按期望宽度比例分配可用宽度（带最小值）。
func mdDistributeWidths(desired []float32, avail, minW float32) []float32 {
	n := len(desired)
	out := make([]float32, n)
	var sum float32
	for _, d := range desired {
		sum += d
	}
	if sum <= 0 {
		for i := range out {
			out[i] = avail / float32(n)
		}
		return out
	}
	if sum <= avail {
		// 富余空间按比例摊开
		extra := avail / sum
		for i, d := range desired {
			out[i] = d * extra
		}
		return out
	}
	// 超宽：按比例压缩 + 最小值
	fixed := float32(0)
	growIdx := make([]int, 0, n)
	for i, d := range desired {
		v := avail * d / sum
		if v < minW {
			out[i] = minW
			fixed += minW
		} else {
			out[i] = v
			fixed += v
			growIdx = append(growIdx, i)
		}
	}
	if fixed > 0 && len(growIdx) > 0 {
		k := avail / fixed
		for i := range out {
			out[i] *= k
			if out[i] < 32 {
				out[i] = 32
			}
		}
	}
	return out
}

// tabs 页签块。
func (r *mdRenderer) tabs(b *mdBlock, idx int) *Node {
	key := r.key + "-tabs-" + strconv.Itoa(idx)
	if len(b.Tabs) == 0 {
		return nil
	}
	active := mdState.tabs[key]
	if active < 0 || active >= len(b.Tabs) {
		active = 0
	}
	heads := make([]*Node, 0, len(b.Tabs))
	for i, t := range b.Tabs {
		i := i
		color := mdColMuted
		bg := "transparent"
		if i == active {
			color = "#ffffff"
			bg = mdColInline
		}
		n := Text(t.Label).St(St().Text(r.base.size-0.5, 500, color).PadXY(12, 6).R(7).BgC(bg))
		n.Tap(func() {
			mdState.tabs[key] = i
			mdTouchState()
			RequestRedraw()
		})
		heads = append(heads, n)
	}
	inner := &mdRenderer{doc: r.doc, opts: r.opts, width: r.width - 28, base: r.base, key: key}
	body := Column(inner.renderBlocks(b.Tabs[active].Blocks)...)
	return Column(
		Row(heads...).St(St().GapOf(4).Padding(5)),
		Divider(),
		body.St(St().Padding(14)),
	).St(St().R(10).Stroke(1, mdColBorder).BgC(ColInput).Clipped())
}

// details 折叠块。
func (r *mdRenderer) details(b *mdBlock, idx int) *Node {
	key := r.key + "-det-" + strconv.Itoa(idx)
	open, ok := mdState.details[key]
	if !ok {
		open = b.Open
	}
	arrow := "▸"
	if open {
		arrow = "▾"
	}
	title := b.Title
	if title == "" {
		title = "详情"
	}
	head := Row(
		Text(arrow).St(St().Text(11, 600, mdColAccent)),
		Text(title).St(St().Text(r.base.size, 600, mdColText)),
		Spacer(),
		Text(map[bool]string{true: "收起", false: "展开"}[open]).St(St().Text(10.5, 400, mdColMuted)),
	).St(St().GapOf(8).Padding(11)).AlignTo(AlignCenter)
	k := key
	head.Tap(func() {
		mdState.details[k] = !open
		mdTouchState()
		RequestRedraw()
	})
	col := Column(head.St(St().R(8).BgC(mdColBgSoft)))
	if open {
		inner := &mdRenderer{doc: r.doc, opts: r.opts, width: r.width - 28, base: r.base, key: key}
		body := Column(inner.renderBlocks(b.Quote)...)
		col.Add(Divider())
		col.Add(body.St(St().Padding(14)))
	}
	return col.St(St().R(10).Stroke(1, mdColBorder).BgC(mdColCodeBg).Clipped())
}

// card 卡片块。
func (r *mdRenderer) card(b *mdBlock, idx int) *Node {
	c := b.Card
	if c == nil {
		return nil
	}
	col := Column()
	if c.Image != "" {
		local, ok := mdImagePath(c.Image)
		if ok {
			if w, h := ImageSize(local); w > 0 && h > 0 {
				ih := float32(150)
				iw := r.width * ih / float32(h)
				if iw < r.width*0.3 {
					iw = r.width * 0.3
				}
				if iw > r.width {
					iw = r.width
				}
				col.Add(ImageSized(local, iw, ih))
			}
		}
	}
	head := Column()
	title := c.Title
	if title == "" {
		title = "卡片"
	}
	tn := Text(title).St(St().Text(r.base.size+3, 700, mdColText).WrapOn())
	if c.URL != "" {
		url := c.URL
		tn = Text(title).St(St().Text(r.base.size+3, 700, mdColLink).WrapOn())
		tn.Tap(func() { mdOpenURL(url) })
	}
	head.Add(tn)
	if c.Subtitle != "" {
		head.Add(Text(c.Subtitle).St(St().Text(r.base.size-1, 400, mdColMuted).WrapOn()))
	}
	if len(c.Blocks) > 0 {
		inner := &mdRenderer{doc: r.doc, opts: r.opts, width: r.width - 32, base: r.base, key: r.key + "-card" + strconv.Itoa(idx)}
		body := Column(inner.renderBlocks(c.Blocks)...)
		head.Add(body.St(St().PadOnly(6, 0, 0, 0)))
	}
	col.Add(Column(head.Children...).St(St().Padding(16).GapOf(4)))
	return col.St(St().R(12).Stroke(1, mdColBorder).BgC(mdColBgSoft).Clipped())
}

// badge 徽章。
func (r *mdRenderer) badge(b *mdBlock) *Node {
	if b.Badge == nil || b.Badge.Text == "" {
		return nil
	}
	fg, bg := mdColText, ColHover
	switch b.Badge.Color {
	case "primary":
		fg, bg = "#c7d2fe", "rgba(99,102,241,0.18)"
	case "success":
		fg, bg = "#6ee7b7", "rgba(52,211,153,0.16)"
	case "warning":
		fg, bg = "#fcd34d", "rgba(251,191,36,0.16)"
	case "error", "danger":
		fg, bg = "#fca5a5", "rgba(248,113,113,0.16)"
	case "info":
		fg, bg = "#7dd3fc", "rgba(56,189,248,0.16)"
	}
	return Row(Text(b.Badge.Text).St(St().Text(r.base.size-1.5, 600, fg).PadXY(10, 4).R(10).BgC(bg))).
		St(St().PadOnly(2, 0, 2, 0))
}

// progress 进度条。
func (r *mdRenderer) progress(b *mdBlock) *Node {
	if b.Prog == nil {
		return nil
	}
	p := b.Prog
	if p.Max <= 0 {
		p.Max = 100
	}
	v := p.Value / p.Max
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	color := ColPrimary
	switch {
	case v >= 0.8:
		color = ColSuccess
	case v >= 0.5:
		color = ColWarn
	case v < 0.3:
		color = ColDanger
	}
	label := p.Label
	if label == "" {
		label = "进度"
	}
	return Column(
		Row(
			Text(label).St(St().Text(r.base.size-1, 500, mdColDim)),
			Spacer(),
			Text(strconv.Itoa(int(v*100))+"%").St(St().Text(r.base.size-1, 600, mdColMuted)),
		).AlignTo(AlignCenter),
		Progress(v, color).St(St().PadOnly(6, 0, 0, 0)),
	).St(St().GapOf(0))
}

// gallery 图片集。
func (r *mdRenderer) gallery(b *mdBlock, idx int) *Node {
	if len(b.Gallery) == 0 {
		return nil
	}
	thumbW := float32(150)
	cols := int((r.width + 8) / (thumbW + 8))
	if cols < 1 {
		cols = 1
	}
	thumbW = (r.width - float32(cols-1)*8) / float32(cols)
	thumbH := thumbW * 0.66
	items := make([]*Node, 0, len(b.Gallery))
	for _, img := range b.Gallery {
		local, ok := mdImagePath(img.Src)
		w0, h0 := 0, 0
		if ok {
			w0, h0 = ImageSize(local)
		}
		if w0 <= 0 || h0 <= 0 {
			items = append(items, Column(
				Text("🖼").St(St().Text(16, 400, mdColMuted).TAlign(AlignCenter)),
				Text(img.Alt).St(St().Text(10, 400, mdColMuted).WrapOn().TAlign(AlignCenter)),
			).St(St().W(thumbW).H(thumbH).Padding(6).R(8).BgC(mdColBgSoft)))
			continue
		}
		src, alt := local, img.Alt
		im := ImageSized(src, thumbW, thumbH).St(St().R(8))
		im.Tap(func() {
			mdState.viewer.Open = true
			mdState.viewer.Src = src
			mdState.viewer.Alt = alt
			RequestRedraw()
		})
		items = append(items, im)
	}
	return Column(
		Row(
			Text("🖼").St(St().Text(11, 400, mdColMuted)),
			Text("图片集 ("+strconv.Itoa(len(b.Gallery))+" 张)").St(St().Text(r.base.size-1.5, 600, mdColDim)),
		).St(St().GapOf(6)).AlignTo(AlignCenter),
		Wrap(items...).St(St().GapOf(8).PadOnly(8, 0, 0, 0)),
	).St(St().Padding(12).R(10).BgC(ColInput).Stroke(1, mdColBorder))
}

// media 音视频占位卡。
func (r *mdRenderer) media(b *mdBlock) *Node {
	icon := "▶"
	label := "视频"
	if b.Lang == "audio" {
		icon = "♫"
		label = "音频"
	}
	title := b.Title
	if title == "" {
		title = label
	}
	col := Column(
		Row(
			Text(icon).St(St().Text(15, 600, mdColAccent).W(26).TAlign(AlignCenter)),
			Column(
				Text(title).St(St().Text(r.base.size, 600, mdColText)),
				Text(b.URL).St(St().Text(r.base.size-2, 400, mdColMuted).WrapOn()),
			).Grow(1).St(St().GapOf(2)),
		).St(St().GapOf(10)).AlignTo(AlignCenter),
		Text("GoGUI 版以占位展示；播放入口：ui.VideoPlayer / ui.MusicPlayer").St(St().Text(r.base.size-2, 400, mdColMuted).WrapOn()),
	).St(St().Padding(14).R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder).GapOf(8))
	return col
}

// toc 目录。
func (r *mdRenderer) toc() *Node {
	if len(r.doc.Headings) == 0 {
		return nil
	}
	items := make([]*Node, 0, len(r.doc.Headings))
	for _, h := range r.doc.Headings {
		indent := float32((h.Level - 1) * 14)
		size := r.base.size - 1
		if h.Level == 1 {
			size = r.base.size
		}
		anchor := mdAnchorID(r.key, h.Key)
		row := Row(
			Box().St(St().W(4).H(4).R(2).BgC(mdColAccent)),
			Text(h.Text).St(St().Text(size, 500, mdColDim).WrapOn()),
		).St(St().GapOf(8).PadOnly(2, 0, 2, indent)).AlignTo(AlignStart).
			Tap(func() { mdScrollToAnchor(anchor) })
		TooltipOf(row, "跳转到该小节")
		items = append(items, row)
	}
	return Column(
		Row(
			Text("目录").St(St().Text(r.base.size-1, 600, mdColMuted)),
			Spacer(),
			Text(strconv.Itoa(len(r.doc.Headings))+" 节").St(St().Text(r.base.size-2, 400, mdColMuted)),
		).AlignTo(AlignCenter),
		Column(items...).St(St().GapOf(3).PadOnly(8, 0, 0, 0)),
	).St(St().Padding(12).R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder).GapOf(6))
}

// footnotes 脚注区：文末汇总所有 [^id] 定义。
func (r *mdRenderer) footnotes() *Node {
	if len(r.doc.Footnotes) == 0 {
		return nil
	}
	items := make([]*Node, 0, len(r.doc.Footnotes)*2)
	for _, f := range r.doc.Footnotes {
		body := mdInlineNodes(f.Inl, r.width-30, r.base, r.opts, 0)
		row := Row(
			Text(f.ID).St(St().Text(r.base.size-2, 600, mdColLink).W(18)),
			Column(body...).Grow(1).St(St().GapOf(mdLineGap(r.opts, r.base.size))),
		).AlignTo(AlignStart)
		if mdState.anchors == nil {
			mdState.anchors = map[string]*Node{}
		}
		id := mdAnchorID(r.key, "fn-"+f.ID)
		row.K(id)
		mdState.anchors[id] = row
		items = append(items, row)
	}
	return Column(
		Divider(),
		Text("脚注").St(St().Text(r.base.size-1.5, 600, mdColMuted)),
		Column(items...).St(St().GapOf(5).PadOnly(6, 0, 0, 0)),
	).St(St().GapOf(8).PadOnly(10, 0, 0, 0))
}

// ---- 复制提示 ----

func mdToastNode() *Node {
	if mdState.toast == "" {
		return nil
	}
	age := AnimTime() - mdState.toastAt
	if age >= ToastDefaultSeconds {
		return nil
	}
	RequestRedraw()
	return Toast(mdState.toast)
}

// ---- 图片放大层 ----

// MarkdownOpenImage 打开图片预览（供外部调用）。
func MarkdownOpenImage(src, alt string) {
	mdState.viewer.Open = true
	mdState.viewer.Src = src
	mdState.viewer.Alt = alt
	RequestRedraw()
}

// MarkdownLightbox 图片预览浮层：放在根 Stack 最后一层，
// 点击遮罩或图片关闭（对应 Web 版 Teleport 到 body 的 image-viewer）。
func MarkdownLightbox() *Node {
	if !mdState.viewer.Open {
		return nil
	}
	vw, vh := ViewportW(), ViewportH()
	maxW := vw * 0.86
	maxH := vh * 0.78
	w0, h0 := ImageSize(mdState.viewer.Src)
	w, h := maxW, maxH
	if w0 > 0 && h0 > 0 {
		s := maxW / float32(w0)
		if s2 := maxH / float32(h0); s2 < s {
			s = s2
		}
		w, h = float32(w0)*s, float32(h0)*s
	}
	scrim := Box().St(St().AbsFill().BgC("rgba(4,7,14,0.94)")).Tap(func() {
		mdState.viewer.Open = false
		RequestRedraw()
	})
	var pic *Node
	if w0 > 0 && h0 > 0 {
		pic = ImageSized(mdState.viewer.Src, w, h).St(St().R(10))
	} else {
		pic = Text("图片不存在：" + mdState.viewer.Src).St(St().Text(13, 400, mdColMuted))
	}
	title := mdState.viewer.Alt
	if title == "" {
		title = mdState.viewer.Src
	}
	closeBtn := Text("✕").St(St().Text(15, 600, "#ffffff").PadXY(10, 6).R(8).
		BgC("rgba(255,255,255,0.10)").AbsTR(18, 16))
	closeBtn.Tap(func() {
		mdState.viewer.Open = false
		RequestRedraw()
	})
	box := Stack(
		Stack(pic).St(St().AbsCenter()),
		Text(title).St(St().Text(12, 500, "#cbd5e1").AbsBottom(26)),
		closeBtn,
	).St(St().AbsFill())

	return Stack(scrim, box).St(St().AbsFill())
}
