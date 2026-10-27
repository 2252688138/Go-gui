package ui

import (
	"strconv"
	"strings"

	"gogui/internal/text"
	"gogui/internal/win32"
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
//   - 图片只支持本地文件（远程 URL 显示为占位卡）；
//   - video/audio 块降级为占位卡（真正播放请用 ui.VideoPlayer）；
//   - 链接点击走系统默认浏览器（win32.ShellOpen）。
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

// MarkdownResetState 清空页签/折叠/图片预览等交互状态。
func MarkdownResetState() {
	mdState.tabs = map[string]int{}
	mdState.details = map[string]bool{}
	mdState.folded = map[string]bool{}
	mdState.toast = ""
	mdState.viewer.Open = false
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
	win32.ShellOpen(url)
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
}

// mdPiece 排版 token（词 / 单字 / 空格）。
type mdPiece struct {
	text string
	sp   mdSpan
	br   bool
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

// mdPieceWidth 单个片段宽度（含内边距）。
func mdPieceWidth(p mdPiece) float32 {
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
		href := p.sp.link
		n.Tap(func() { mdOpenURL(href) })
		TooltipOf(n, href)
	}
	if p.sp.strike {
		w := mdPieceWidth(mdPiece{text: p.text, sp: p.sp})
		line := Box().St(St().W(w).H(1).BgC(mdColMuted).AbsCenter())
		n = Stack(n, line)
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
	prevLink := mdLinkHandler
	if o.OnLink != nil {
		mdLinkHandler = o.OnLink
	}
	r := &mdRenderer{doc: mdParseCached(src), opts: o, width: w, base: base, key: key}
	kids := r.renderBlocks(r.doc.Blocks)
	mdLinkHandler = prevLink
	if t := mdToastNode(); t != nil {
		kids = append(kids, t)
	}
	return Column(kids...).St(St().W(w).GapOf(0)).K(key)
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
	case mdCode, mdTable, mdTabs, mdDetails, mdCard, mdGallery:
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
	}
	return nil
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
		return Column(kids...).St(St().GapOf(1))
	}
	base := r.base
	base.size = size
	base.weight = weight
	base.color = color
	return Column(mdInlineNodes(b.Inl, r.width, base, r.opts, 0)...)
}

func (r *mdRenderer) paragraph(b *mdBlock) *Node {
	kids := mdInlineNodes(b.Inl, r.width, r.base, r.opts, 0)
	return Column(kids...).St(St().GapOf(mdLineGap(r.opts, r.base.size)))
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
	codeW := r.width - numW - 28

	var rows []*Node
	for i, spans := range shown {
		num := Text(strconv.Itoa(i + 1)).St(St().Text(size-0.5, 400, mdColMuted).Font(mono).W(numW).TAlign(AlignEnd).Lh(lineH))
		code := r.codeLine(spans, codeW, mono, size, lineH)
		rows = append(rows, Row(num, code).St(St().GapOf(10)).AlignTo(AlignStart))
	}

	body := Column(rows...).St(St().GapOf(1).PadXY(14, 12))

	// 头部：语言 + 复制 + 折叠
	langLabel := strings.ToUpper(strings.TrimSpace(b.Lang))
	if langLabel == "" {
		langLabel = "TEXT"
	}
	copyBtn := Text("复制").St(St().Text(11, 500, mdColMuted).PadXY(8, 3).R(6).
		BgC(ColHover).Stroke(1, ColHover))
	code := b.Code
	copyBtn.Tap(func() {
		if win32.SetClipboardText(code) {
			mdToast("代码已复制")
		} else {
			mdToast("复制失败")
		}
	})
	headKids := []*Node{
		Text(langLabel).St(St().Text(10.5, 600, mdColMuted).Font(mono)),
		Spacer(),
	}
	if folded {
		n := len(lines) - maxLines
		expand := Text("展开 " + strconv.Itoa(n) + " 行").St(St().Text(11, 500, mdColAccent).PadXY(8, 3).R(6).
			BgC("rgba(129,140,248,0.10)"))
		k := key
		expand.Tap(func() {
			mdState.folded[k] = true
			RequestRedraw()
		})
		headKids = append(headKids, expand)
	} else if maxLines > 0 && len(lines) > maxLines {
		collapse := Text("收起").St(St().Text(11, 500, mdColMuted).PadXY(8, 3).R(6).
			BgC(ColHover))
		k := key
		collapse.Tap(func() {
			delete(mdState.folded, k)
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
	}
	return mdSynText
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
			box := ColHover
			mark := ""
			if it.Task == 1 {
				box = mdColAccent
				mark = "√"
			}
			mk = Stack(
				Box().St(St().AbsFill().R(4).BgC(box).Stroke(1, mdColBorder)),
				Text(mark).St(St().AbsFill().Text(10, 700, "#ffffff").CenterYOn().TAlign(AlignCenter)),
			).St(St().W(14).H(14).MarOnly(4, 6, 0, 6))
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
	w0, h0 := ImageSize(src)
	if w0 <= 0 || h0 <= 0 {
		// 远程 / 不存在的图片：显示占位卡
		return Column(
			Text("🖼  图片").St(St().Text(11.5, 600, mdColMuted)),
			Text(src).St(St().Text(11, 400, mdColMuted).WrapOn()),
			Text("GoGUI 仅支持本地图片（PNG / JPEG / GIF / BMP）").St(St().Text(10.5, 400, mdColMuted)),
		).St(St().GapOf(4).Padding(14).R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder))
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
	img := ImageSized(src, w, h).St(St().R(8))
	s := src
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

	var rows []*Node
	headCells := make([]*Node, 0, cols)
	for i, c := range b.Table.Header {
		sp := r.base
		sp.weight = 600
		sp.color = mdColText
		kids := mdInlineNodes(c.Inl, widths[i]-padX*2, sp, r.opts, 0)
		headCells = append(headCells, Column(kids...).St(St().W(widths[i]).PadXY(padX, 8).TAlign(c.Align)))
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
			kids := mdInlineNodes(c.Inl, widths[i]-padX*2, r.base, r.opts, 0)
			cells = append(cells, Column(kids...).St(St().W(widths[i]).PadXY(padX, 8)))
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
		if w, h := ImageSize(c.Image); w > 0 && h > 0 {
			ih := float32(150)
			iw := r.width * ih / float32(h)
			if iw < r.width*0.3 {
				iw = r.width * 0.3
			}
			if iw > r.width {
				iw = r.width
			}
			col.Add(ImageSized(c.Image, iw, ih))
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
		w0, h0 := ImageSize(img.Src)
		if w0 <= 0 || h0 <= 0 {
			items = append(items, Column(
				Text("🖼").St(St().Text(16, 400, mdColMuted).TAlign(AlignCenter)),
				Text(img.Alt).St(St().Text(10, 400, mdColMuted).WrapOn().TAlign(AlignCenter)),
			).St(St().W(thumbW).H(thumbH).Padding(6).R(8).BgC(mdColBgSoft)))
			continue
		}
		src, alt := img.Src, img.Alt
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
		items = append(items, Row(
			Box().St(St().W(4).H(4).R(2).BgC(mdColAccent)),
			Text(h.Text).St(St().Text(size, 500, mdColDim).WrapOn()),
		).St(St().GapOf(8).PadOnly(2, 0, 2, indent)).AlignTo(AlignStart))
	}
	return Column(
		Text("目录").St(St().Text(r.base.size-1, 600, mdColMuted)),
		Column(items...).St(St().GapOf(3).PadOnly(8, 0, 0, 0)),
	).St(St().Padding(12).R(10).BgC(mdColBgSoft).Stroke(1, mdColBorder).GapOf(6))
}

// ---- 复制提示 ----

func mdToastNode() *Node {
	if mdState.toast == "" {
		return nil
	}
	age := AnimTime() - mdState.toastAt
	if age > 1.6 {
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
