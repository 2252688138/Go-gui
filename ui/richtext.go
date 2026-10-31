// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"gogui/internal/geom"
	"gogui/internal/gpu"
	"gogui/internal/text"
)

// 富文本（span 树）：一段文字内混排不同字号 / 粗细 / 颜色 / 链接 / 行内代码。
//
// 与 Text 的区别：Text 一个节点只有一种样式；RichText 把若干 TextSpan 排成同一段
// 可折行的文本，测量、绘制、命中复用同一份行/片段数据，因此「看到的字形范围」
// 就是「点击命中的范围」，链接可以精确到字。
//
// 用法：
//
//	ui.RichText([]ui.TextSpan{
//	    {Text: "共 ", Color: ui.ColTextDim},
//	    {Text: "12", Weight: 700, Color: ui.ColPrimary},
//	    {Text: " 个文件，", Color: ui.ColTextDim},
//	    {Text: "查看详情", Link: "detail", Underline: true},
//	}, ui.RichTextOptions{Size: 13}).OnSpanClickEvt(func(sp ui.TextSpan) {
//	    if sp.Link == "detail" { openDetail() }
//	})

// TextSpan 行内片段。零值字段表示继承 RichTextOptions / 节点样式。
type TextSpan struct {
	Text string
	// Color 文字颜色（空 = 继承基础色）。
	Color string
	// BgColor 片段底色（高亮）。
	BgColor string
	// Size 字号（0 = 继承）。
	Size float32
	// Weight 字重（0 = 继承）。
	Weight int32
	// Family 字体家族（空 = 继承）。
	Family string
	// Link 链接标识（非空时该片段可点击，悬停显示下划线）。
	Link string
	// Underline / Strike 下划线 / 删除线。
	Underline bool
	Strike    bool
	// Code 行内代码：等宽字体 + 底色。
	Code bool
}

// RichTextOptions 整段富文本的默认样式。
type RichTextOptions struct {
	Size       float32
	Weight     int32
	Color      string
	Family     string
	LineHeight float32
	// MaxLines > 0 时超出部分截断并追加省略号。
	MaxLines int
	// Align 行内对齐（默认左对齐）。
	Align Align
	// LinkColor 链接默认颜色（空 = #8b93ff）。
	LinkColor string
	// CodeBg 行内代码底色（空 = 主题悬停色）。
	CodeBg string
	// NoWrap 关闭自动折行（需要与 Style.Width 配合裁剪）。
	NoWrap bool
}

// richPiece 一行内的一个片段（坐标相对内容区左上角）。
type richPiece struct {
	span int
	text string
	x, w float32
	h    float32
}

// richLine 折行后的一行。
type richLine struct {
	y, h, w float32
	pieces  []richPiece
}

// richLayoutData 测量结果（布局阶段写入，绘制 / 命中共用）。
type richLayoutData struct {
	lines []richLine
	w, h  float32
	maxW  float32
	valid bool
}

// RichText 构建富文本节点（默认自动折行）。
func RichText(spans []TextSpan, opts RichTextOptions) *Node {
	if opts.Size <= 0 {
		opts.Size = 13
	}
	if opts.Weight <= 0 {
		opts.Weight = 400
	}
	if opts.Color == "" {
		opts.Color = ColText
	}
	if opts.LinkColor == "" {
		opts.LinkColor = "#8b93ff"
	}
	if opts.CodeBg == "" {
		opts.CodeBg = ColHover
	}
	return &Node{
		Kind:     KRichText,
		Spans:    spans,
		RichOpts: opts,
		Style: Style{
			Opacity:    1,
			Wrap:       !opts.NoWrap,
			FontSize:   opts.Size,
			FontWeight: opts.Weight,
			Color:      C(opts.Color),
			FontFamily: opts.Family,
			LineHeight: opts.LineHeight,
		},
	}
}

// OnSpanClickEvt 行内片段点击回调（链接 / 行内代码均可命中）。
func (n *Node) OnSpanClickEvt(fn func(TextSpan)) *Node { n.OnSpanClick = fn; return n }

// spanFont 返回片段实际使用的字体（nil 表示字体缺失）。
func (lc *layoutCtx) spanFont(sp TextSpan, base *Style) *text.Font {
	size := sp.Size
	if size <= 0 {
		size = base.FontSize
	}
	if size <= 0 {
		size = 13
	}
	weight := sp.Weight
	if weight <= 0 {
		weight = base.FontWeight
	}
	if weight <= 0 {
		weight = 400
	}
	fam := sp.Family
	if fam == "" {
		fam = base.FontFamily
	}
	if sp.Code && fam == "" {
		fam = "Consolas"
	}
	if lc == nil || lc.fonts == nil {
		return nil
	}
	return lc.fonts.getFamily(size, weight, fam)
}

// spanColor 片段文字颜色（链接未显式配色时用链接色）。
func spanColor(sp TextSpan, opts RichTextOptions, base *Style) gpu.Paint {
	if sp.Color != "" {
		return C(sp.Color)
	}
	if sp.Link != "" {
		return C(opts.LinkColor)
	}
	if base.Color.A > 0 {
		return base.Color
	}
	return C(ColText)
}

// richAtom 折行用的最小单位（一个 rune + 其字体度量）。
type richAtom struct {
	r      rune
	span   int
	adv    float32
	brkAft bool
	brkBef bool
	nl     bool
}

// layoutRich 富文本测量与折行：写入 n.rich，返回内容尺寸。
//
// 断行优先级与 wrapTextRuns 保持一致：空格 / 连字符 / 斜杠优先，
// CJK 逐字可断，收尾标点断在其之前。这样 RichText 与 Text 的排版观感一致。
func (lc *layoutCtx) layoutRich(n *Node, maxW float32) (float32, float32) {
	opts := &n.RichOpts
	base := &n.Style
	if maxW <= 0 {
		maxW = 1
	}
	fonts := make([]*text.Font, len(n.Spans))
	atoms := make([]richAtom, 0, 64)
	for i := range n.Spans {
		sp := n.Spans[i]
		f := lc.spanFont(sp, base)
		fonts[i] = f
		sc := float32(1)
		if f != nil && f.Scale() > 0 {
			sc = f.Scale()
		}
		for _, r := range sp.Text {
			at := richAtom{r: r, span: i, adv: 0}
			if f != nil {
				at.adv = f.GlyphFor(r).Advance / sc
			}
			switch {
			case r == '\n':
				at.nl = true
			case r == ' ' || r == '\t' || r == '-' || r == '/' || r == '\u00a0':
				at.brkAft = true
			case isWideRune(r):
				if isCJKPunct(r) {
					at.brkBef = true
				} else {
					at.brkAft = true
				}
			}
			atoms = append(atoms, at)
		}
	}

	noWrap := opts.NoWrap
	var lines []richLine
	appendLine := func(from, to int) {
		ln := richLine{}
		maxH := float32(0)
		x := float32(0)
		for start := from; start < to; {
			sp := atoms[start].span
			end := start + 1
			for end < to && atoms[end].span == sp {
				end++
			}
			w := float32(0)
			txt := make([]rune, 0, end-start)
			for k := start; k < end; k++ {
				w += atoms[k].adv
				txt = append(txt, atoms[k].r)
			}
			h := float32(0)
			if f := fonts[sp]; f != nil {
				h = f.LineHeight
			}
			if h > maxH {
				maxH = h
			}
			ln.pieces = append(ln.pieces, richPiece{span: sp, text: string(txt), x: x, w: w, h: h})
			x += w
			start = end
		}
		ln.w = x
		if maxH <= 0 {
			maxH = 16
		}
		ln.h = maxH
		lines = append(lines, ln)
	}

	if noWrap {
		appendLine(0, len(atoms))
	} else {
		start, width, lastBreak := 0, float32(0), -1
		for i := 0; i < len(atoms); i++ {
			at := atoms[i]
			if at.nl {
				appendLine(start, i)
				start = i + 1
				width, lastBreak = 0, -1
				continue
			}
			if width+at.adv > maxW && i > start {
				cut := i
				if lastBreak > start && lastBreak < i {
					cut = lastBreak
				}
				appendLine(start, cut)
				start = cut
				for start < len(atoms) && atoms[start].r == ' ' {
					start++
				}
				i = start - 1
				width, lastBreak = 0, -1
				continue
			}
			width += at.adv
			if at.brkAft {
				lastBreak = i + 1
			} else if at.brkBef {
				lastBreak = i
			}
		}
		appendLine(start, len(atoms))
	}

	// 超出行数上限：截断并在末尾追加省略号
	if opts.MaxLines > 0 && len(lines) > opts.MaxLines {
		lines = lines[:opts.MaxLines]
		last := &lines[len(lines)-1]
		ef := lc.spanFont(TextSpan{}, base)
		ew := float32(8)
		if ef != nil && ef.Scale() > 0 {
			ew = ef.Measure("…")
		}
		for last.w+ew > maxW && len(last.pieces) > 0 {
			p := &last.pieces[len(last.pieces)-1]
			rs := []rune(p.text)
			if len(rs) <= 1 {
				last.w -= p.w
				last.pieces = last.pieces[:len(last.pieces)-1]
				continue
			}
			rs = rs[:len(rs)-1]
			nw := float32(0)
			if f := fonts[p.span]; f != nil && f.Scale() > 0 {
				for _, r := range rs {
					nw += f.GlyphFor(r).Advance / f.Scale()
				}
			}
			last.w += nw - p.w
			p.text = string(rs)
			p.w = nw
		}
		if len(last.pieces) > 0 {
			p := &last.pieces[len(last.pieces)-1]
			p.text += "…"
			p.w += ew
			last.w += ew
		} else {
			last.pieces = append(last.pieces, richPiece{x: 0, w: ew, h: last.h, span: -1, text: "…"})
			last.w = ew
		}
	}

	// 纵向累计 + 行内对齐
	var y float32
	var maxLineW float32
	lhBase := base.LineHeight
	for i := range lines {
		ln := &lines[i]
		h := ln.h
		if lhBase > h {
			h = lhBase
		}
		ln.y = y
		ln.h = h
		y += h
		if ln.w > maxLineW {
			maxLineW = ln.w
		}
		align := opts.Align
		if align == AlignStart {
			align = base.TextAlign
		}
		off := float32(0)
		switch align {
		case AlignCenter:
			off = (maxW - ln.w) / 2
		case AlignEnd:
			off = maxW - ln.w
		}
		if off > 0 {
			for k := range ln.pieces {
				ln.pieces[k].x += off
			}
		}
	}
	if len(lines) == 0 {
		lines = append(lines, richLine{h: 16})
		y = 16
	}
	if noWrap && maxLineW > maxW {
		maxLineW = maxW
	}
	n.rich = richLayoutData{lines: lines, w: maxLineW, h: y, maxW: maxW, valid: true}
	return maxLineW, y
}

// richAt 命中：返回坐标处的片段（坐标相对内容区左上角）。
func richAt(n *Node, x, y float32) (TextSpan, bool) {
	if n == nil || !n.rich.valid {
		return TextSpan{}, false
	}
	for _, ln := range n.rich.lines {
		if y < ln.y || y > ln.y+ln.h {
			continue
		}
		for _, p := range ln.pieces {
			if x < p.x || x > p.x+p.w {
				continue
			}
			if p.span < 0 || p.span >= len(n.Spans) {
				return TextSpan{}, false
			}
			return n.Spans[p.span], true
		}
	}
	return TextSpan{}, false
}

// paintRichText 绘制富文本（App.paint 的 KRichText 分支）。
func (a *App) paintRichText(n *Node, r geom.Rect, st *Style) {
	if n == nil || !n.rich.valid || len(n.Spans) == 0 {
		return
	}
	lx := r.X + st.Pad.Left + st.DrawX
	ly := r.Y + st.Pad.Top + st.DrawY
	opts := n.RichOpts

	hoverSpan := -1
	if a.mouseIn {
		mx, my := a.mouseX-lx, a.mouseY-ly
		for _, ln := range n.rich.lines {
			if my < ln.y || my > ln.y+ln.h {
				continue
			}
			for _, p := range ln.pieces {
				if mx >= p.x && mx <= p.x+p.w {
					hoverSpan = p.span
				}
			}
		}
	}

	for _, ln := range n.rich.lines {
		for _, p := range ln.pieces {
			sp := TextSpan{Text: p.text}
			if p.span >= 0 && p.span < len(n.Spans) {
				sp = n.Spans[p.span]
			}
			f := a.fontForSpan(sp, st)
			if f == nil {
				continue
			}
			col := spanColor(sp, opts, st)
			hot := p.span == hoverSpan && sp.Link != ""
			if hot {
				col = Blend(col, C("#ffffff"), 0.3)
			}
			ty := ly + ln.y
			if p.h > f.LineHeight {
				ty += (p.h - f.LineHeight) / 2
			}
			if sp.Code || sp.BgColor != "" {
				bg := C(opts.CodeBg)
				if sp.BgColor != "" {
					bg = C(sp.BgColor)
				}
				a.r.Rect(gpu.Shape{
					Rect:   geom.Rect{X: lx + p.x - 2, Y: ty, W: p.w + 4, H: f.LineHeight},
					Radius: [4]float32{3, 3, 3, 3},
					Fill:   bg,
				})
			}
			a.r.TextRun(gpu.TextRun{X: lx + p.x, Y: ty, Text: p.text, Font: f, Color: col})
			if sp.Underline || hot {
				a.r.Rect(gpu.Shape{
					Rect:   geom.Rect{X: lx + p.x, Y: ty + f.LineHeight - 3, W: p.w, H: 1},
					Radius: [4]float32{0.5, 0.5, 0.5, 0.5},
					Fill:   col,
				})
			}
			if sp.Strike {
				a.r.Rect(gpu.Shape{
					Rect:   geom.Rect{X: lx + p.x, Y: ty + f.LineHeight/2, W: p.w, H: 1},
					Radius: [4]float32{0.5, 0.5, 0.5, 0.5},
					Fill:   col,
				})
			}
		}
	}
}

// fontForSpan 绘制阶段取片段字体（与测量同一套参数）。
func (a *App) fontForSpan(sp TextSpan, base *Style) *text.Font {
	size := sp.Size
	if size <= 0 {
		size = base.FontSize
	}
	if size <= 0 {
		size = 13
	}
	weight := sp.Weight
	if weight <= 0 {
		weight = base.FontWeight
	}
	if weight <= 0 {
		weight = 400
	}
	fam := sp.Family
	if fam == "" {
		fam = base.FontFamily
	}
	if sp.Code && fam == "" {
		fam = "Consolas"
	}
	return a.fonts.getFamily(size, weight, fam)
}
