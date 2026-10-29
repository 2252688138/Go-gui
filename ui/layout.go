package ui

import (
	"gogui/internal/geom"
	"gogui/internal/text"
)

// Constraints 约束模型（Flutter 风格）。
type Constraints struct {
	MinW, MaxW, MinH, MaxH float32
}

func (c Constraints) loosen() Constraints {
	return Constraints{0, c.MaxW, 0, c.MaxH}
}

// 视口尺寸（最近一次 Layout 的尺寸），响应式分支读取。
var (
	viewportW float32 = 1180
	viewportH float32 = 760
)

// ViewportW 返回最近一次布局的视口宽度（逻辑像素），用于响应式分支。
func ViewportW() float32 { return viewportW }

// ViewportH 返回最近一次布局的视口高度（逻辑像素）。
func ViewportH() float32 { return viewportH }

// BP 按视口宽度选择断点值：<640 取 xs，<960 取 sm，<1280 取 md，其余 lg。
func BP[T any](xs, sm, md, lg T) T {
	switch w := viewportW; {
	case w < 640:
		return xs
	case w < 960:
		return sm
	case w < 1280:
		return md
	default:
		return lg
	}
}

// layoutCtx 布局上下文（字体测量 + 视口高度）。
type layoutCtx struct {
	fonts     *fontCache
	viewportH float32
}

func (lc *layoutCtx) measure(n *Node, maxW float32) (float32, float32) {
	f := lc.fonts.getFamily(n.Style.FontSize, n.Style.FontWeight, n.Style.FontFamily)
	if f == nil {
		return 0, 0
	}
	lh := n.Style.LineHeight
	if lh <= 0 {
		lh = f.LineHeight
	}
	// WrapOn：按可用宽度折行（与绘制共用折行缓存），高度 = 行数 × 行高。
	// 结果来自 wrapLinesCached，同一段落在测量/绘制之间只算一次。
	if n.Style.Wrap && maxW > 0 {
		n.wrapW = maxW
		lines := wrapLinesCached(f, n.Text, maxW)
		var w float32
		for _, ln := range lines {
			if v := f.Measure(ln); v > w {
				w = v
			}
		}
		return w, float32(len(lines)) * lh
	}
	n.wrapW = 0
	var w float32
	forEachLine(n.Text, func(ln string) {
		if v := f.Measure(ln); v > w {
			w = v
		}
	})
	return w, float32(lineCount(n.Text)) * lh
}

// Layout 对整棵树做布局：先 measure（算尺寸），再 arrange（定位）。
func Layout(root *Node, w, h float32, fonts *fontCache) {
	viewportW, viewportH = w, h
	lc := &layoutCtx{fonts: fonts, viewportH: h}
	root.Frame = geom.Rect{X: 0, Y: 0, W: w, H: h}
	measureNode(lc, root, Constraints{0, w, 0, h}, nil)
	root.Frame.W, root.Frame.H = w, h
	arrange(lc, root)
}

// arrange 递归定位（父节点位置已确定）。
func arrange(lc *layoutCtx, n *Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case KBox, KScroll, KStack, KGrid, KWrap, KMasonry:
		placeChildren(lc, n, n.Frame.W-n.Style.Pad.Horizontal(), n.Frame.H-n.Style.Pad.Vertical())
	}
}

// measureNode 只计算尺寸，不定位。
func measureNode(lc *layoutCtx, n *Node, c Constraints, parent *Node) {
	if n == nil {
		return
	}
	n.Parent = parent
	st := &n.Style

	// 1. 自宽高
	cw := c.MaxW
	ch := c.MaxH

	// 绝对定位且两侧同时约束时，尺寸由父容器内边距区推导（如 AbsFill）
	if st.PosAbs {
		if st.HasL && st.HasR {
			if v := c.MaxW - st.AbsL - st.AbsR; v > 0 {
				cw = v
			}
		}
		if st.HasT && st.HasB {
			if v := c.MaxH - st.AbsT - st.AbsB; v > 0 {
				ch = v
			}
		}
	}

	if st.WidthPct > 0 {
		cw = c.MaxW * st.WidthPct / 100
	} else if st.Width > 0 {
		cw = st.Width
	}
	if st.HeightPct > 0 {
		ch = c.MaxH * st.HeightPct / 100
	} else if st.Height > 0 {
		ch = st.Height
	}

	innerMaxW := cw - st.Pad.Horizontal()
	if innerMaxW < 0 {
		innerMaxW = 0
	}
	innerMaxH := ch - st.Pad.Vertical()
	if innerMaxH < 0 {
		innerMaxH = 0
	}

	var contentW, contentH float32

	switch n.Kind {
	case KText, KButton:
		if st.Wrap {
			contentW, contentH = lc.measure(n, innerMaxW)
		} else {
			contentW, contentH = lc.measure(n, 0)
		}
		if st.Width > 0 || st.WidthPct > 0 {
			contentW = innerMaxW
		}
		if st.LineHeight > 0 {
			lines := float32(1)
			for _, r := range n.Text {
				if r == '\n' {
					lines++
				}
			}
			contentH = lines * st.LineHeight
		}

	case KImage:
		// 图片：默认取原图像素尺寸，可用 Style.Width/Height 覆盖
		w0, h0 := ImageSize(n.Src)
		contentW, contentH = float32(w0), float32(h0)
		if contentW <= 0 {
			contentW = 160
		}
		if contentH <= 0 {
			contentH = 120
		}
	case KVideo:
		// 视频：默认取画面像素尺寸（无画面时按 16:9 占位），可用 Style.Width/Height 覆盖
		w0, h0 := 0, 0
		if n.Video != nil {
			w0, h0 = n.Video.Size()
		}
		if w0 <= 0 || h0 <= 0 {
			w0, h0 = 640, 360
		}
		contentW, contentH = float32(w0), float32(h0)
	default:
		contentW, contentH = measureChildren(lc, n, innerMaxW, innerMaxH)
	}

	w := st.Width
	if st.WidthPct > 0 {
		w = c.MaxW * st.WidthPct / 100
	}
	if w == 0 {
		if st.PosAbs && st.HasL && st.HasR {
			w = c.MaxW - st.AbsL - st.AbsR
			if w < 0 {
				w = 0
			}
		} else {
			w = contentW + st.Pad.Horizontal()
		}
	}
	h := st.Height
	if st.HeightPct > 0 {
		h = c.MaxH * st.HeightPct / 100
	}
	if h == 0 {
		if st.PosAbs && st.HasT && st.HasB {
			h = c.MaxH - st.AbsT - st.AbsB
			if h < 0 {
				h = 0
			}
		} else {
			h = contentH + st.Pad.Vertical()
		}
	}

	if st.MinW > 0 && w < st.MinW {
		w = st.MinW
	}
	if st.MaxW > 0 && w > st.MaxW {
		w = st.MaxW
	}
	if st.MinH > 0 && h < st.MinH {
		h = st.MinH
	}
	if st.MaxH > 0 && h > st.MaxH {
		h = st.MaxH
	}
	if w > c.MaxW && c.MaxW > 0 {
		w = c.MaxW
	}
	// Wrap/Grid/Masonry 是内容型容器，高度按内容自然伸展（不受父高截断）；
	// 滚动容器（Scroll）的子项同样按内容自然伸展，否则内容会被截成视口高、无法滚动。
	content := n.Kind == KWrap || n.Kind == KGrid || n.Kind == KMasonry
	inScroll := parent != nil && parent.Kind == KScroll
	if c.MaxH > 0 && h > c.MaxH && !st.Scrollable && !content && !inScroll {
		h = c.MaxH
	}

	n.Frame.W = w
	n.Frame.H = h
}

// measureChildren 返回子节点总体尺寸。
func measureChildren(lc *layoutCtx, n *Node, availW, availH float32) (float32, float32) {
	switch n.Kind {
	case KStack:
		return measureStack(lc, n, availW, availH)
	case KGrid:
		return measureGrid(lc, n, availW, availH)
	case KWrap:
		return measureWrap(lc, n, availW, availH)
	case KMasonry:
		return measureMasonry(lc, n, availW, availH)
	}
	return measureFlex(lc, n, availW, availH)
}

// measureStack 叠层：容器尺寸取最大子项（绝对定位子项不撑开父级）。
func measureStack(lc *layoutCtx, n *Node, availW, availH float32) (float32, float32) {
	var maxW, maxH float32
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		measureNode(lc, c, Constraints{0, availW, 0, availH}, n)
		if c.Style.PosAbs {
			continue
		}
		if v := c.Frame.W + c.Style.Margin.Horizontal(); v > maxW {
			maxW = v
		}
		if v := c.Frame.H + c.Style.Margin.Vertical(); v > maxH {
			maxH = v
		}
	}
	return maxW, maxH
}

// measureFlex 弹性容器（Row / Column）测量。
// 两遍：先按父可用尺寸量全部子项并统计固定项占用，再把剩余主轴空间分配给
// Grow 子项重新测量——这样可伸缩容器内部（如 Scroll 里的百分比宽度）不会按
// 父级满宽计算而偏大。
func measureFlex(lc *layoutCtx, n *Node, availW, availH float32) (float32, float32) {
	st := &n.Style
	horiz := n.Dir == Horizontal
	gap := st.Gap

	// 测量约束：主轴方向不设上限（与 CSS flex 一致），交叉轴按父可用尺寸。
	// 若主轴也给上限，子项内部的百分比尺寸会按父级满宽计算，导致自动宽容器被撑大。
	childC := func() Constraints {
		if horiz {
			return Constraints{0, 0, 0, availH}
		}
		return Constraints{0, availW, 0, 0}
	}

	var fixedMain, growSum float32
	first := true
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if c.Style.PosAbs {
			// 绝对定位不占据流空间，只预先量尺寸
			measureNode(lc, c, Constraints{0, availW, 0, availH}, n)
			continue
		}
		if !first && gap > 0 {
			fixedMain += gap
		}
		first = false
		cst := &c.Style
		measureNode(lc, c, childC(), n)
		if cst.Grow > 0 {
			growSum += cst.Grow
			continue
		}
		if horiz {
			fixedMain += c.Frame.W + cst.Margin.Horizontal()
		} else {
			fixedMain += c.Frame.H + cst.Margin.Vertical()
		}
	}

	if growSum > 0 {
		free := availW - fixedMain
		if !horiz {
			free = availH - fixedMain
		}
		if free < 0 {
			free = 0
		}
		for _, c := range n.Children {
			if c == nil || c.Style.PosAbs || c.Style.Grow <= 0 {
				continue
			}
			cst := &c.Style
			// 主轴显式定尺寸（Width/Height/Basis）的项不需要重测
			if horiz {
				if cst.Width > 0 || cst.WidthPct > 0 || cst.Basis > 0 {
					continue
				}
			} else if cst.Height > 0 || cst.HeightPct > 0 || cst.Basis > 0 {
				continue
			}
			share := free * cst.Grow / growSum
			cc := Constraints{0, share, 0, availH}
			if !horiz {
				cc = Constraints{0, availW, 0, share}
			}
			measureNode(lc, c, cc, n)
		}
	}

	var sumMain, maxCross float32
	first = true
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		if !first && gap > 0 {
			sumMain += gap
		}
		first = false
		cst := &c.Style
		if horiz {
			sumMain += c.Frame.W + cst.Margin.Horizontal()
			if cross := c.Frame.H + cst.Margin.Vertical(); cross > maxCross {
				maxCross = cross
			}
		} else {
			sumMain += c.Frame.H + cst.Margin.Vertical()
			if cross := c.Frame.W + cst.Margin.Horizontal(); cross > maxCross {
				maxCross = cross
			}
		}
	}
	if horiz {
		return sumMain, maxCross
	}
	return maxCross, sumMain
}

// ---- 网格 / 换行 / 瀑布流 ----

// gridCols 计算列数：Cols > 0 用固定列数；ColW > 0 按列宽自动算；否则按 220 最小列宽自适应。
func gridCols(st *Style, availW float32) int {
	if st.Cols > 0 {
		return st.Cols
	}
	minW := st.ColW
	if minW <= 0 {
		minW = 220
	}
	g := st.Gap
	c := int((availW + g) / (minW + g))
	if c < 1 {
		c = 1
	}
	return c
}

// gridColW 计算单列宽度。
func gridColW(st *Style, availW float32, cols int) float32 {
	if st.ColW > 0 && st.Cols <= 0 {
		return st.ColW
	}
	w := (availW - st.Gap*float32(cols-1)) / float32(cols)
	if w < 0 {
		w = 0
	}
	return w
}

// gridRowHeights 按列数切行并返回每行高度。
func gridRowHeights(n *Node, cols int) []float32 {
	rows := make([]float32, 0, len(n.Children)/cols+1)
	idx := 0
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		r := idx / cols
		for len(rows) <= r {
			rows = append(rows, 0)
		}
		if v := c.Frame.H + c.Style.Margin.Vertical(); v > rows[r] {
			rows[r] = v
		}
		idx++
	}
	return rows
}

// measureGrid 网格：按列等宽，行高取该行最高子项。
func measureGrid(lc *layoutCtx, n *Node, availW, availH float32) (float32, float32) {
	st := &n.Style
	cols := gridCols(st, availW)
	colW := gridColW(st, availW, cols)

	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if c.Style.PosAbs {
			measureNode(lc, c, Constraints{0, availW, 0, availH}, n)
			continue
		}
		measureNode(lc, c, Constraints{MaxW: colW, MaxH: availH}, n)
	}
	rows := gridRowHeights(n, cols)
	var h float32
	for i, v := range rows {
		h += v
		if i > 0 {
			h += st.Gap
		}
	}
	w := float32(cols)*colW + st.Gap*float32(cols-1)
	if w > availW {
		w = availW
	}
	return w, h
}

// placeGrid 网格摆放：单元格等宽，子项默认填满单元格（显式宽高优先）。
func placeGrid(lc *layoutCtx, n *Node, ox, oy, innerW, innerH float32) {
	st := &n.Style
	cols := gridCols(st, innerW)
	colW := gridColW(st, innerW, cols)
	rows := gridRowHeights(n, cols)

	idx := 0
	y := oy
	for ri, rowH := range rows {
		for ci := 0; ci < cols; ci++ {
			if idx >= len(n.Children) {
				break
			}
			c := n.Children[idx]
			if c == nil {
				idx++
				ci--
				continue
			}
			cst := &c.Style
			if cst.PosAbs {
				idx++
				ci--
				continue
			}
			if cst.Width == 0 && cst.WidthPct == 0 {
				c.Frame.W = colW - cst.Margin.Horizontal()
			}
			align := n.Align
			if cst.AlignSelf != AlignStart {
				align = cst.AlignSelf
			}
			if align == AlignStretch && cst.Height == 0 && cst.HeightPct == 0 && cst.MaxH == 0 {
				if v := rowH - cst.Margin.Vertical(); v > 0 {
					c.Frame.H = v
				}
			}
			c.Frame.X = ox + float32(ci)*(colW+st.Gap) + cst.Margin.Left
			cy := y + cst.Margin.Top
			avail := rowH - cst.Margin.Vertical()
			switch align {
			case AlignCenter:
				cy = y + cst.Margin.Top + (avail-c.Frame.H)/2
			case AlignEnd:
				cy = y + cst.Margin.Top + avail - c.Frame.H
			}
			c.Frame.Y = cy
			arrange(lc, c)
			idx++
		}
		y += rowH + st.Gap
		_ = ri
	}
	placeAbsChildren(lc, n, ox, oy, innerW, innerH)
	clampChildren(n)
}

// wrapLines 把子项切分为若干行（流式换行），返回每行子项、行宽、行高。
type wrapLine struct {
	items []*Node
	w, h  float32
}

func wrapLines(n *Node, innerW float32) []*wrapLine {
	st := &n.Style
	lines := make([]*wrapLine, 0, 4)
	cur := &wrapLine{}
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		cst := &c.Style
		cw := c.Frame.W + cst.Margin.Horizontal()
		ch := c.Frame.H + cst.Margin.Vertical()
		if len(cur.items) > 0 && cur.w+st.Gap+cw > innerW {
			lines = append(lines, cur)
			cur = &wrapLine{}
		}
		if len(cur.items) > 0 {
			cur.w += st.Gap
		}
		cur.items = append(cur.items, c)
		cur.w += cw
		if ch > cur.h {
			cur.h = ch
		}
	}
	if len(cur.items) > 0 {
		lines = append(lines, cur)
	}
	return lines
}

// measureWrap 流式换行：一行放不下就换行（类似 flex-wrap: wrap）。
func measureWrap(lc *layoutCtx, n *Node, availW, availH float32) (float32, float32) {
	st := &n.Style
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if c.Style.PosAbs {
			measureNode(lc, c, Constraints{0, availW, 0, availH}, n)
			continue
		}
		measureNode(lc, c, Constraints{0, availW, 0, availH}, n)
	}
	// 先有尺寸才能切行
	lines := wrapLines(n, availW)
	var w, h float32
	for i, ln := range lines {
		if ln.w > w {
			w = ln.w
		}
		h += ln.h
		if i > 0 {
			h += st.Gap
		}
	}
	if w > availW {
		w = availW
	}
	return w, h
}

// placeWrap 流式摆放（行内支持主轴分布与交叉轴对齐；交叉轴 Stretch 按起始对齐，避免拉伸控件）。
func placeWrap(lc *layoutCtx, n *Node, ox, oy, innerW, innerH float32) {
	st := &n.Style
	lines := wrapLines(n, innerW)
	y := oy
	for _, ln := range lines {
		sp := distribute(n.Justify, innerW-ln.w, len(ln.items))
		x := ox + sp.start
		for i, c := range ln.items {
			if i > 0 {
				x += st.Gap + sp.extra
			}
			cst := &c.Style
			c.Frame.X = x + cst.Margin.Left
			align := n.Align
			if cst.AlignSelf != AlignStart {
				align = cst.AlignSelf
			}
			avail := ln.h - cst.Margin.Vertical()
			c.Frame.Y = y + cst.Margin.Top
			switch align {
			case AlignCenter:
				c.Frame.Y = y + cst.Margin.Top + (avail-c.Frame.H)/2
			case AlignEnd:
				c.Frame.Y = y + cst.Margin.Top + avail - c.Frame.H
			}
			arrange(lc, c)
			x += c.Frame.W + cst.Margin.Horizontal()
		}
		y += ln.h + st.Gap
	}
	placeAbsChildren(lc, n, ox, oy, innerW, innerH)
	clampChildren(n)
}

// measureMasonry 瀑布流：子项落到当前最矮的列。
func measureMasonry(lc *layoutCtx, n *Node, availW, availH float32) (float32, float32) {
	st := &n.Style
	cols := gridCols(st, availW)
	colW := gridColW(st, availW, cols)
	heights := make([]float32, cols)

	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if c.Style.PosAbs {
			measureNode(lc, c, Constraints{0, availW, 0, availH}, n)
			continue
		}
		measureNode(lc, c, Constraints{MaxW: colW, MaxH: availH}, n)
		k := shortest(heights)
		heights[k] += c.Frame.H + c.Style.Margin.Vertical() + st.Gap
	}
	var maxH float32
	for _, v := range heights {
		if v > maxH {
			maxH = v
		}
	}
	if maxH > 0 {
		maxH -= st.Gap
	}
	w := float32(cols)*colW + st.Gap*float32(cols-1)
	if w > availW {
		w = availW
	}
	return w, maxH
}

// placeMasonry 瀑布流摆放（列宽等分，列内顺序堆叠）。
func placeMasonry(lc *layoutCtx, n *Node, ox, oy, innerW, innerH float32) {
	st := &n.Style
	cols := gridCols(st, innerW)
	colW := gridColW(st, innerW, cols)
	tops := make([]float32, cols)
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		cst := &c.Style
		k := shortest(tops)
		if cst.Width == 0 && cst.WidthPct == 0 {
			c.Frame.W = colW - cst.Margin.Horizontal()
		}
		c.Frame.X = ox + float32(k)*(colW+st.Gap) + cst.Margin.Left
		c.Frame.Y = oy + tops[k] + cst.Margin.Top
		arrange(lc, c)
		tops[k] += c.Frame.H + cst.Margin.Vertical() + st.Gap
	}
	placeAbsChildren(lc, n, ox, oy, innerW, innerH)
	clampChildren(n)
}

func shortest(v []float32) int {
	k := 0
	for i := 1; i < len(v); i++ {
		if v[i] < v[k]-0.5 {
			k = i
		}
	}
	return k
}

// spread 主轴分布结果：起始偏移 + 每项之间的额外间距。
type spread struct {
	start, extra float32
}

// distribute 按对齐方式把剩余空间分配到主轴（支持 space-between / around / evenly）。
func distribute(a Align, free float32, count int) spread {
	if free <= 0 || count <= 0 {
		return spread{}
	}
	switch a {
	case AlignCenter:
		return spread{start: free / 2}
	case AlignEnd:
		return spread{start: free}
	case AlignSpaceBetween:
		if count > 1 {
			return spread{extra: free / float32(count-1)}
		}
	case AlignSpaceAround:
		u := free / float32(count)
		return spread{start: u / 2, extra: u}
	case AlignSpaceEvenly:
		u := free / float32(count+1)
		return spread{start: u, extra: u}
	}
	return spread{}
}

func placeChildren(lc *layoutCtx, n *Node, innerW, innerH float32) {
	st := &n.Style
	ox := n.Frame.X + st.Pad.Left
	oy := n.Frame.Y + st.Pad.Top

	switch n.Kind {
	case KStack:
		placeStack(lc, n, ox, oy, innerW, innerH)
		return
	case KGrid:
		placeGrid(lc, n, ox, oy, innerW, innerH)
		return
	case KWrap:
		placeWrap(lc, n, ox, oy, innerW, innerH)
		return
	case KMasonry:
		placeMasonry(lc, n, ox, oy, innerW, innerH)
		return
	}

	horiz := n.Dir == Horizontal

	// 主轴剩余空间分配给 Grow 子项（Basis > 0 时优作为基准尺寸）
	limit := innerH
	if horiz {
		limit = innerW
	}
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		cst := &c.Style
		// 百分比尺寸按父容器最终内边距区重算：父容器尺寸可能在 arrange 阶段被拉伸/收缩。
		// 仅在父容器尺寸是“确定”的时候重算，避免百分比指代按内容撑开的循环尺寸。
		defW := st.Width > 0 || st.WidthPct > 0 || n.Kind == KScroll || (st.PosAbs && st.HasL && st.HasR)
		defH := st.Height > 0 || st.HeightPct > 0 || (st.PosAbs && st.HasT && st.HasB)
		if defW && cst.WidthPct > 0 {
			c.Frame.W = innerW * cst.WidthPct / 100
		}
		if defH && cst.HeightPct > 0 {
			c.Frame.H = innerH * cst.HeightPct / 100
		}
		if cst.Basis > 0 {
			if horiz {
				c.Frame.W = cst.Basis
			} else {
				c.Frame.H = cst.Basis
			}
			continue
		}
		// 主轴未显式定尺寸的 Grow 子项：先归零，避免把按父宽测量的自然尺寸再叠加一次
		if cst.Grow > 0 {
			if horiz && cst.Width == 0 && cst.WidthPct == 0 {
				c.Frame.W = 0
			}
			if !horiz && cst.Height == 0 && cst.HeightPct == 0 {
				c.Frame.H = 0
			}
		}
	}
	var fixed, growSum float32
	first := true
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		if !first && st.Gap > 0 {
			fixed += st.Gap
		}
		first = false
		if horiz {
			fixed += c.Frame.W + c.Style.Margin.Horizontal()
		} else {
			fixed += c.Frame.H + c.Style.Margin.Vertical()
		}
		growSum += c.Style.Grow
	}
	if growSum > 0 {
		free := limit - fixed
		if free > 0 {
			for _, c := range n.Children {
				if c == nil || c.Style.PosAbs || c.Style.Grow <= 0 {
					continue
				}
				extra := free * c.Style.Grow / growSum
				if horiz {
					c.Frame.W += extra
				} else {
					c.Frame.H += extra
				}
			}
		}
	}

	// 主轴分布（Justify）：center / end / space-between / space-around / space-evenly
	var usedMain float32
	count := 0
	first = true
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		if !first && st.Gap > 0 {
			usedMain += st.Gap
		}
		first = false
		count++
		if horiz {
			usedMain += c.Frame.W + c.Style.Margin.Horizontal()
		} else {
			usedMain += c.Frame.H + c.Style.Margin.Vertical()
		}
	}
	sp := distribute(n.Justify, limit-usedMain, count)

	cursor := oy + sp.start
	if horiz {
		cursor = ox + sp.start
	}
	first = true

	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		if !first && st.Gap > 0 {
			cursor += st.Gap
		}
		first = false
		cst := &c.Style

		align := n.Align
		if cst.AlignSelf != AlignStart {
			align = cst.AlignSelf
		}

		if horiz {
			c.Frame.X = cursor + cst.Margin.Left
			c.Frame.Y = oy + cst.Margin.Top
			crossAvail := innerH - cst.Margin.Vertical()
			switch align {
			case AlignCenter:
				c.Frame.Y = oy + cst.Margin.Top + (crossAvail-c.Frame.H)/2
			case AlignEnd:
				c.Frame.Y = oy + crossAvail - c.Frame.H + cst.Margin.Top
			case AlignStretch:
				if cst.HeightPct > 0 {
					c.Frame.H = crossAvail * cst.HeightPct / 100
				} else if cst.Height == 0 && cst.MaxH == 0 {
					c.Frame.H = crossAvail
				}
			}
			cursor += c.Frame.W + cst.Margin.Horizontal() + sp.extra
		} else {
			c.Frame.X = ox + cst.Margin.Left
			c.Frame.Y = cursor + cst.Margin.Top
			crossAvail := innerW - cst.Margin.Horizontal()
			switch align {
			case AlignCenter:
				c.Frame.X = ox + cst.Margin.Left + (crossAvail-c.Frame.W)/2
			case AlignEnd:
				c.Frame.X = ox + crossAvail - c.Frame.W + cst.Margin.Left
			case AlignStretch:
				if cst.WidthPct > 0 {
					c.Frame.W = crossAvail * cst.WidthPct / 100
				} else if cst.Width == 0 && cst.MaxW == 0 {
					c.Frame.W = crossAvail
				}
			}
			cursor += c.Frame.H + cst.Margin.Vertical() + sp.extra
		}

		// 尺寸可能被 stretch 修改，重排内部
		arrange(lc, c)
	}

	if n.Kind == KScroll {
		n.ContentH = cursor - oy + st.Pad.Bottom
	}

	placeAbsChildren(lc, n, ox, oy, innerW, innerH)

	// 默认不超出父容器（非滚动）
	if n.Kind != KScroll && n.Style.Clip {
		clampChildren(n)
	}
}

// clampChildren 把子项裁剪到父框内（配合 Clipped）。
func clampChildren(n *Node) {
	if !n.Style.Clip || n.Kind == KScroll {
		return
	}
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		c.Frame = clampRect(c.Frame, n.Frame)
	}
}

// placeStack 叠层布局：子节点全部叠放在同一区域（后声明者在上层）。
func placeStack(lc *layoutCtx, n *Node, ox, oy, innerW, innerH float32) {
	for _, c := range n.Children {
		if c == nil || c.Style.PosAbs {
			continue
		}
		cst := &c.Style
		// 尺寸：百分比 > 固定 > 铺满
		if cst.WidthPct > 0 {
			c.Frame.W = (innerW - cst.Margin.Horizontal()) * cst.WidthPct / 100
		} else if cst.Width == 0 && cst.MaxW == 0 && !cst.HasL && !cst.HasR {
			c.Frame.W = innerW - cst.Margin.Horizontal()
		}
		if cst.HeightPct > 0 {
			c.Frame.H = (innerH - cst.Margin.Vertical()) * cst.HeightPct / 100
		} else if cst.Height == 0 && cst.MaxH == 0 && !cst.HasT && !cst.HasB {
			c.Frame.H = innerH - cst.Margin.Vertical()
		}
		align := n.Align
		if cst.AlignSelf != AlignStart {
			align = cst.AlignSelf
		}
		x := ox + cst.Margin.Left
		switch align {
		case AlignCenter:
			x = ox + cst.Margin.Left + (innerW-cst.Margin.Horizontal()-c.Frame.W)/2
		case AlignEnd:
			x = ox + innerW - cst.Margin.Right - c.Frame.W
		}
		sp := distribute(n.Justify, innerH-cst.Margin.Vertical()-c.Frame.H, 1)
		y := oy + cst.Margin.Top + sp.start
		c.Frame.X, c.Frame.Y = x, y
		arrange(lc, c)
	}
	placeAbsChildren(lc, n, ox, oy, innerW, innerH)
}

// placeAbsChildren 摆放绝对定位子节点（在流布局之后，保证覆盖在上层）。
func placeAbsChildren(lc *layoutCtx, n *Node, ox, oy, innerW, innerH float32) {
	for _, c := range n.Children {
		if c == nil || !c.Style.PosAbs {
			continue
		}
		cst := &c.Style
		// 父容器尺寸可能在 arrange 阶段被拉伸/收缩，两侧同时约束的子项（AbsFill 等）
		// 需按最终内边距区重算尺寸，否则会保留 measure 阶段的旧尺寸而溢出。
		if cst.HasL && cst.HasR && cst.Width == 0 && cst.WidthPct == 0 {
			c.Frame.W = innerW - cst.AbsL - cst.AbsR - cst.Margin.Horizontal()
			if c.Frame.W < 0 {
				c.Frame.W = 0
			}
		}
		if cst.HasT && cst.HasB && cst.Height == 0 && cst.HeightPct == 0 && !cst.Scrollable {
			c.Frame.H = innerH - cst.AbsT - cst.AbsB - cst.Margin.Vertical()
			if c.Frame.H < 0 {
				c.Frame.H = 0
			}
		}
		x := ox + cst.Margin.Left
		switch {
		case cst.HasL:
			x = ox + cst.AbsL + cst.Margin.Left
		case cst.HasR:
			x = ox + innerW - cst.AbsR - c.Frame.W - cst.Margin.Right
		case cst.AbsCenterX:
			x = ox + (innerW-c.Frame.W)/2 + cst.Margin.Left
		}
		y := oy + cst.Margin.Top
		switch {
		case cst.HasT:
			y = oy + cst.AbsT + cst.Margin.Top
		case cst.HasB:
			y = oy + innerH - cst.AbsB - c.Frame.H - cst.Margin.Bottom
		case cst.AbsCenterY:
			y = oy + (innerH-c.Frame.H)/2 + cst.Margin.Top
		}
		c.Frame.X, c.Frame.Y = x, y
		arrange(lc, c)
	}
}

func clampRect(r, box geom.Rect) geom.Rect {
	return r.Intersect(box)
}

// fontCache 按字体家族/字号/字重缓存字体。
type fontCache struct {
	fonts map[fontKey]*text.Font
	// age 最近使用序号（LRU 回收用）
	age   map[fontKey]uint64
	tick  uint64
	face  string
	scale float32
	// onEvict 字体被回收时通知上层释放对应图集纹理
	onEvict func(*text.Font)
}

// maxFonts 字体实例缓存上限。每个字体自带 4MB 字形图集 + opentype face，
// 不限制数量时「字号×字重×家族」组合会让内存无界增长（旧版 16MB/字体）。
const maxFonts = 24

type fontKey struct {
	family string
	size   int16
	weight int32
}

func newFontCache(face string, scale float32) *fontCache {
	return &fontCache{fonts: make(map[fontKey]*text.Font), age: make(map[fontKey]uint64), face: face, scale: scale}
}

func (fc *fontCache) get(size float32, weight int32) *text.Font {
	return fc.getFamily(size, weight, "")
}

// getFamily 按字体家族取字体；family 为空时用默认家族。
func (fc *fontCache) getFamily(size float32, weight int32, family string) *text.Font {
	if size <= 0 {
		size = 14
	}
	if weight == 0 {
		weight = 400
	}
	face := family
	if face == "" {
		face = fc.face
	}
	key := fontKey{face, int16(size * 4), weight}
	fc.tick++
	if f, ok := fc.fonts[key]; ok {
		fc.age[key] = fc.tick
		return f
	}
	if len(fc.fonts) >= maxFonts {
		fc.evictOldest()
	}
	f := text.NewFont(face, size, weight, fc.scale)
	fc.fonts[key] = f
	fc.age[key] = fc.tick
	return f
}

// evictOldest 回收最久未使用的字体（同时释放其图集纹理）。
func (fc *fontCache) evictOldest() {
	var oldKey fontKey
	var oldest uint64
	found := false
	for k, t := range fc.age {
		if !found || t < oldest {
			oldKey, oldest, found = k, t, true
		}
	}
	if !found {
		return
	}
	if f, ok := fc.fonts[oldKey]; ok {
		if fc.onEvict != nil {
			fc.onEvict(f)
		}
		f.Close()
		delete(fc.fonts, oldKey)
	}
	delete(fc.age, oldKey)
	// 折行缓存以 *Font 为键，字体已关闭必须整体失效，否则缓存会拖住字体与图集
	clearWrapMemo()
}

func (fc *fontCache) setScale(scale float32) {
	if scale <= 0 || scale == fc.scale {
		return
	}
	fc.scale = scale
	for _, f := range fc.fonts {
		f.SetScale(scale)
	}
	// 字形尺寸变了，按字体缓存的折行结果全部失效
	clearWrapMemo()
}

func (fc *fontCache) close() {
	for _, f := range fc.fonts {
		f.Close()
	}
	fc.fonts = map[fontKey]*text.Font{}
	fc.age = map[fontKey]uint64{}
	clearWrapMemo()
}

// atlasBytes 当前图集占用总字节（字体数 × 每张图集大小）。
func (fc *fontCache) atlasBytes() int {
	n := 0
	for _, f := range fc.fonts {
		n += f.AtlasBytes()
	}
	return n
}
