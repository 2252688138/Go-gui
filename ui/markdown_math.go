package ui

import (
	"strconv"
	"strings"
)

// 数学公式排版（LaTeX 子集）。
//
// 框架只有「单一样式文本节点 + 绝对定位」，没有富文本 run，也没有 HTML/MathML，
// 所以公式必须自己算度量再摆坐标：
//
//	源码 → token → 表达式树 → 按字体度量做基线排版（matBox）→ Stack + AbsTL 节点
//
// 支持：分式 \frac、根式 \sqrt[n]、上下标 ^ _、求和/连乘/积分（含上下限）、
// 括号 \left( \right)、矩阵与 cases、重音 \hat \bar \vec \tilde \dot、
// 希腊字母与常用关系/运算符号、\text \mathbf \mathrm、函数名 \sin \log 等。

// matCtx 排版上下文（字号/字体/颜色）。
type matCtx struct {
	size   float32
	family string
	color  string
}

// matBox 排版盒：宽、高、基线（自顶部起算）。
type matBox struct {
	w, h, base float32
	kids       []matKid
}

// matKid 盒内一个已定位的子节点。
type matKid struct {
	x, y float32
	n    *Node
}

// ---- 盒运算 ----

func matShift(b matBox, dx, dy float32) matBox {
	out := matBox{w: b.w, h: b.h, base: b.base, kids: make([]matKid, 0, len(b.kids))}
	for _, k := range b.kids {
		out.kids = append(out.kids, matKid{x: k.x + dx, y: k.y + dy, n: k.n})
	}
	return out
}

// matHBox 横向拼接：按基线对齐，项间插入 gap。
func matHBox(items []matBox, gap float32) matBox {
	base := float32(0)
	for _, it := range items {
		if it.base > base {
			base = it.base
		}
	}
	w := float32(0)
	for i, it := range items {
		if i > 0 {
			w += gap
		}
		w += it.w
	}
	h := base
	for _, it := range items {
		if v := it.h - it.base + base; v > h {
			h = v
		}
	}
	out := matBox{w: w, h: h, base: base}
	x := float32(0)
	for i, it := range items {
		if i > 0 {
			x += gap
		}
		s := matShift(it, x, base-it.base)
		out.kids = append(out.kids, s.kids...)
		x += it.w
	}
	return out
}

// matVBox 纵向堆叠（各项水平居中），返回的基线为 0（顶部对齐用）。
func matVBox(items []matBox, gap float32) matBox {
	w := float32(0)
	h := float32(0)
	for i, it := range items {
		if i > 0 {
			h += gap
		}
		if it.w > w {
			w = it.w
		}
		h += it.h
	}
	out := matBox{w: w, h: h, base: h}
	y := float32(0)
	for i, it := range items {
		if i > 0 {
			y += gap
		}
		s := matShift(it, (w-it.w)/2, y)
		out.kids = append(out.kids, s.kids...)
		y += it.h
	}
	return out
}

// matRect 线段（分式横线、根号上横线）。
func matRect(w, h float32, color string) matBox {
	n := Box().St(St().W(w).H(h).BgC(color).AbsTL(0, 0))
	return matBox{w: w, h: h, base: h, kids: []matKid{{0, 0, n}}}
}

func matConcat(boxes ...matBox) matBox {
	var kids []matKid
	w, h := float32(0), float32(0)
	for _, b := range boxes {
		kids = append(kids, b.kids...)
		if b.w > w {
			w = b.w
		}
		if b.h > h {
			h = b.h
		}
	}
	return matBox{w: w, h: h, base: h, kids: kids}
}

// ---- 原子 ----

// atom 一段直立文本（自带度量）。
func (c *matCtx) atom(s string, size float32, weight int32) matBox {
	if s == "" {
		return matBox{}
	}
	f := mdFont(size, weight, c.family)
	w := size * float32(len([]rune(s))) * 0.5
	asc, desc := size*0.78, size*0.24
	if f != nil {
		w = f.Measure(s)
		asc, desc = f.Ascent, f.Descent
	}
	h := asc + desc
	st := St().Text(size, weight, c.color).W(w+0.6).H(h).Lh(h).AbsTL(0, 0)
	if c.family != "" {
		st.Font(c.family)
	}
	n := Text(s).St(st)
	return matBox{w: w, h: h, base: asc, kids: []matKid{{0, 0, n}}}
}

// space 水平空白。
func (c *matCtx) space(em float32, size float32) matBox {
	return matBox{w: em * size, h: 0, base: 0}
}

// ---- 复杂结构 ----

// frac 分式：分子在上、横线居中、分母在下。
func (c *matCtx) frac(num, den matBox, size float32) matBox {
	axis := size * 0.26 // 横线中心距基线高度
	thick := float32(1.2)
	gap := float32(2)
	barY := num.h + gap
	denY := barY + thick + gap
	base := barY + thick/2 + axis
	w := num.w
	if den.w > w {
		w = den.w
	}
	w += size * 0.24
	h := denY + den.h
	if v := base + size*0.24; v > h {
		h = v
	}
	out := matBox{w: w, h: h, base: base}
	out.kids = append(out.kids, matShift(num, (w-num.w)/2, 0).kids...)
	out.kids = append(out.kids, matShift(den, (w-den.w)/2, denY).kids...)
	bar := matRect(w-2, thick, c.color)
	out.kids = append(out.kids, matShift(bar, 1, barY).kids...)
	return out
}

// fracPack 把一组表达式包成盒并做分式（displaystyle 时尺寸不缩）。
func (c *matCtx) fracPack(num, den []matExpr, size float32) matBox {
	return c.frac(c.layout(num, size), c.layout(den, size), size)
}

// sqrt 根式（index 为可选根指数）。
func (c *matCtx) sqrt(body matBox, index []matExpr, size float32) matBox {
	rootSize := body.h + size*0.55
	if rootSize < size {
		rootSize = size
	}
	root := c.atom("√", rootSize, 400)
	overY := float32(1.5)
	line := matRect(body.w+2, 1.2, c.color)
	bodyTop := overY + 2
	bodyX := root.w - 2
	h := bodyTop + body.h + 1
	if root.h > h {
		h = root.h
	}
	out := matBox{w: bodyX + body.w + 2, h: h, base: bodyTop + body.base}
	if len(index) > 0 {
		idx := c.layout(index, size*0.55)
		out.kids = append(out.kids, matShift(idx, 0, 0).kids...)
		out.kids = append(out.kids, matShift(root, idx.w*0.7, bodyTop-root.h+body.h+2).kids...)
	} else {
		out.kids = append(out.kids, matShift(root, 0, bodyTop-root.h+body.h+2).kids...)
	}
	out.kids = append(out.kids, matShift(body, bodyX, bodyTop).kids...)
	out.kids = append(out.kids, matShift(line, bodyX, overY).kids...)
	return out
}

// bigOp 大运算符（∑ ∏ ∫ 等）：limits=true 时上下限在正上下方，否则在右侧。
func (c *matCtx) bigOp(glyph string, upper, lower []matExpr, size float32) matBox {
	gl := c.atom(glyph, size*1.42, 400)
	up := c.layout(upper, size*0.62)
	lo := c.layout(lower, size*0.62)
	w := gl.w
	if up.w > w {
		w = up.w
	}
	if lo.w > w {
		w = lo.w
	}
	h := up.h + gl.h + lo.h
	if len(upper) == 0 {
		h -= up.h
	}
	if len(lower) == 0 {
		h -= lo.h
	}
	base := h - lo.h
	out := matBox{w: w, h: h, base: base}
	y := float32(0)
	if len(upper) > 0 {
		out.kids = append(out.kids, matShift(up, (w-up.w)/2, 0).kids...)
		y += up.h
	}
	gy := y + (base - y - gl.base)
	out.kids = append(out.kids, matShift(gl, (w-gl.w)/2, gy).kids...)
	if len(lower) > 0 {
		out.kids = append(out.kids, matShift(lo, (w-lo.w)/2, base+gl.h-gl.base-1).kids...)
	}
	return out
}

// scripts 给主体加右上/右下标。
func (c *matCtx) scripts(body matBox, sup, sub []matExpr, size float32) matBox {
	ss := size * 0.72
	sp := c.layout(sup, ss)
	sb := c.layout(sub, ss)
	out := matBox{w: body.w, h: body.h, base: body.base}
	out.kids = append(out.kids, body.kids...)

	supY := float32(0)
	if len(sup) > 0 {
		supY = body.base - size*0.42 - sp.h
		// 上标超出盒顶时整体下移，保持内容不被裁掉
		if supY < 0 {
			sh := -supY
			out.kids = matShift(matBox{kids: out.kids}, 0, sh).kids
			out.base += sh
			out.h += sh
			supY = 0
		}
		out.kids = append(out.kids, matShift(sp, body.w+size*0.06, supY).kids...)
		out.w = body.w + size*0.06 + sp.w
	}
	if len(sub) > 0 {
		y := body.base + size*0.18
		if v := y + sb.h; v > out.h {
			out.h = v
		}
		out.kids = append(out.kids, matShift(sb, body.w+size*0.06, y).kids...)
		if w := body.w + size*0.06 + sb.w; w > out.w {
			out.w = w
		}
	}
	return out
}

// accent 重音（帽 / 横线 / 向量箭头 / 波浪 / 点）。
func (c *matCtx) accent(kind string, body matBox, size float32) matBox {
	glyph := ""
	switch kind {
	case "hat":
		glyph = "^"
	case "bar":
		glyph = "‾"
	case "vec":
		glyph = "→"
	case "tilde":
		glyph = "~"
	case "dot":
		glyph = "·"
	}
	acc := c.atom(glyph, size*0.82, 400)
	w := body.w
	if acc.w > w {
		w = acc.w
	}
	h := acc.h*0.85 + body.h
	out := matBox{w: w, h: h, base: acc.h*0.85 + body.base}
	out.kids = append(out.kids, matShift(acc, (w-acc.w)/2, 0).kids...)
	out.kids = append(out.kids, matShift(body, (w-body.w)/2, acc.h*0.85).kids...)
	return out
}

// matrix 矩阵 / cases：rows 为逐行逐列的表达式。
func (c *matCtx) matrix(rows [][]matExpr, left, right string, size float32) matBox {
	nr := len(rows)
	if nr == 0 {
		return matBox{}
	}
	// 各列自然宽度
	nc := 0
	cells := make([][]matBox, nr)
	for i, row := range rows {
		cells[i] = make([]matBox, len(row))
		for j, e := range row {
			cells[i][j] = c.layoutOne(e, size)
		}
		if len(row) > nc {
			nc = len(row)
		}
	}
	colW := make([]float32, nc)
	rowH := make([]float32, nr)
	rowB := make([]float32, nr)
	for i := range cells {
		for j := range cells[i] {
			if cells[i][j].w > colW[j] {
				colW[j] = cells[i][j].w
			}
			if cells[i][j].h > rowH[i] {
				rowH[i] = cells[i][j].h
			}
			if cells[i][j].base > rowB[i] {
				rowB[i] = cells[i][j].base
			}
		}
	}
	gapX, gapY := size*0.85, size*0.45
	w := float32(0)
	for j := 0; j < nc; j++ {
		w += colW[j]
		if j < nc-1 {
			w += gapX
		}
	}
	h := float32(0)
	for i := 0; i < nr; i++ {
		h += rowH[i]
		if i < nr-1 {
			h += gapY
		}
	}
	out := matBox{w: w, h: h, base: h}
	y := float32(0)
	for i := range cells {
		x := float32(0)
		for j := range cells[i] {
			cell := cells[i][j]
			out.kids = append(out.kids, matShift(cell, x+(colW[j]-cell.w)/2, y+(rowB[i]-cell.base)).kids...)
			x += colW[j]
			if j < nc-1 {
				x += gapX
			}
		}
		y += rowH[i] + gapY
	}
	// 括号
	if left != "" || right != "" {
		hs := h * 1.02
		fs := size
		if hs > size*1.3 {
			fs = size * hs / (size * 1.3)
		}
		var parts []matBox
		if left != "" {
			parts = append(parts, c.atom(left, fs, 400))
		}
		parts = append(parts, out)
		if right != "" {
			parts = append(parts, c.atom(right, fs, 400))
		}
		return matHBox(parts, size*0.10)
	}
	return out
}

// delim \left( ... \right)：括号按内容高度拉伸。
func (c *matCtx) delim(left, right string, body matBox, size float32) matBox {
	mk := func(s string) matBox {
		if s == "" || s == "." {
			return matBox{}
		}
		fs := size
		if body.h > size*1.25 {
			fs = size * body.h / (size * 1.25)
		}
		return c.atom(s, fs, 400)
	}
	return matHBox([]matBox{mk(left), body, mk(right)}, size*0.10)
}

// ---- 表达式树 ----

const (
	meAtom = iota
	meSeq
	meFrac
	meSqrt
	meBig
	meMatrix
	meAccent
	meDelim
	meSpace
	meScript
)

type matExpr struct {
	kind  int
	text  string
	items []matExpr
	num   []matExpr
	den   []matExpr
	base  []matExpr
	sup   []matExpr
	sub   []matExpr
	upper []matExpr
	lower []matExpr
	rows  [][]matExpr
	left  string
	right string
	acc   string
	em    float32
}

// layoutOne 排版单个表达式（其宽度/高度/基线独立计算）。
func (c *matCtx) layoutOne(e matExpr, size float32) matBox {
	switch e.kind {
	case meSeq, 0:
		if len(e.items) == 0 && e.text == "" {
			return matBox{}
		}
		if len(e.items) == 0 {
			return c.atom(e.text, size, 400)
		}
		var boxes []matBox
		for i, it := range e.items {
			b := c.layoutOne(it, size)
			if i > 0 && matNeedsGap(e.items[i-1], it) {
				boxes = append(boxes, c.space(0.16, size))
			}
			boxes = append(boxes, b)
		}
		return matHBox(boxes, 0)
	case meFrac:
		return c.frac(c.layout(e.num, size), c.layout(e.den, size), size)
	case meSqrt:
		return c.sqrt(c.layout(e.base, size), e.items, size)
	case meBig:
		return c.bigOp(e.text, e.upper, e.lower, size)
	case meMatrix:
		return c.matrix(e.rows, e.left, e.right, size)
	case meAccent:
		return c.accent(e.acc, c.layout(e.base, size), size)
	case meScript:
		return c.scripts(c.layout(e.base, size), e.sup, e.sub, size)
	case meDelim:
		return c.delim(e.left, e.right, c.layout(e.base, size), size)
	case meSpace:
		return c.space(e.em, size)
	}
	return c.atom(e.text, size, 400)
}

// layout 排版一串表达式（顺序拼接）。
func (c *matCtx) layout(list []matExpr, size float32) matBox {
	if len(list) == 0 {
		return matBox{}
	}
	return c.layoutOne(matExpr{kind: meSeq, items: list}, size)
}

// matNeedsGap 关系/二元运算符两侧留白。
func matNeedsGap(prev, cur matExpr) bool {
	return matIsBinOp(prev) || matIsBinOp(cur)
}

func matIsBinOp(e matExpr) bool {
	switch e.text {
	case "=", "+", "-", "±", "∓", "<", ">", "≤", "≥", "≠", "≈", "~", "→", "←", "⇒", "⇔", "∈", "≡",
		"×", "÷", "·", "∪", "∩", "⊂", "⊃", "⊆", "⊇", "∥", "⊥", "∝", "≪", "≫", "≅", "⊕", "⊗":
		return e.kind == meAtom || e.kind == 0
	}
	return false
}

// ---- 词法 ----

type matTok struct {
	c    rune   // '{' '}' '^' '_' '&' '\\' '~' 'r'(普通字符) 'w'(控制词)
	name string // 控制词名
	ch   rune   // 普通字符
}

func matLex(s string) []matTok {
	rs := []rune(s)
	var out []matTok
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch r {
		case ' ', '\t', '\n', '\r':
			continue
		case '\\':
			if i+1 >= len(rs) {
				continue
			}
			n := rs[i+1]
			if n == '\\' {
				out = append(out, matTok{c: '\\'})
				i++
				continue
			}
			if !isIdentChar(n) {
				// \, \; \! 等间距命令直接忽略
				i++
				continue
			}
			j := i + 1
			for j < len(rs) && isIdentChar(rs[j]) {
				j++
			}
			out = append(out, matTok{c: 'w', name: string(rs[i+1 : j])})
			i = j - 1
		case '{':
			out = append(out, matTok{c: '{'})
		case '}':
			out = append(out, matTok{c: '}'})
		case '^':
			out = append(out, matTok{c: '^'})
		case '_':
			out = append(out, matTok{c: '_'})
		case '&':
			out = append(out, matTok{c: '&'})
		case '~':
			out = append(out, matTok{c: '~'})
		default:
			out = append(out, matTok{c: 'r', ch: r})
		}
	}
	return out
}

// ---- 语法 ----

type matParser struct {
	toks []matTok
	pos  int
}

var matSymbols = map[string]string{
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ε", "varepsilon": "ε",
	"zeta": "ζ", "eta": "η", "theta": "θ", "vartheta": "ϑ", "iota": "ι", "kappa": "κ",
	"lambda": "λ", "mu": "μ", "nu": "ν", "xi": "ξ", "pi": "π", "rho": "ρ", "sigma": "σ",
	"tau": "τ", "upsilon": "υ", "phi": "φ", "varphi": "φ", "chi": "χ", "psi": "ψ", "omega": "ω",
	"Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ", "Xi": "Ξ", "Pi": "Π",
	"Sigma": "Σ", "Phi": "Φ", "Psi": "Ψ", "Omega": "Ω",
	"times": "×", "div": "÷", "cdot": "·", "pm": "±", "mp": "∓", "ast": "∗",
	"le": "≤", "leq": "≤", "ge": "≥", "geq": "≥", "ne": "≠", "neq": "≠", "approx": "≈",
	"equiv": "≡", "sim": "∼", "propto": "∝", "ll": "≪", "gg": "≫", "cong": "≅",
	"to": "→", "rightarrow": "→", "leftarrow": "←", "Rightarrow": "⇒", "Leftarrow": "⇐",
	"leftrightarrow": "↔", "Leftrightarrow": "⇔", "mapsto": "↦", "uparrow": "↑", "downarrow": "↓",
	"in": "∈", "notin": "∉", "ni": "∋", "subset": "⊂", "supset": "⊃", "subseteq": "⊆",
	"supseteq": "⊇", "cup": "∪", "cap": "∩", "emptyset": "∅", "varnothing": "∅",
	"forall": "∀", "exists": "∃", "nexists": "∄", "neg": "¬", "land": "∧", "lor": "∨",
	"infty": "∞", "partial": "∂", "nabla": "∇", "prime": "′", "degree": "°",
	"cdots": "⋯", "ldots": "…", "dots": "…", "vdots": "⋮", "ddots": "⋱",
	"angle": "∠", "triangle": "△", "square": "□", "circ": "∘", "bullet": "•",
	"sum": "∑", "prod": "∏", "int": "∫", "iint": "∬", "iiint": "∭", "oint": "∮",
	"lim": "lim", "log": "log", "ln": "ln", "lg": "lg", "exp": "exp",
	"sin": "sin", "cos": "cos", "tan": "tan", "cot": "cot", "sec": "sec", "csc": "csc",
	"arcsin": "arcsin", "arccos": "arccos", "arctan": "arctan",
	"sinh": "sinh", "cosh": "cosh", "tanh": "tanh", "max": "max", "min": "min",
	"sup": "sup", "inf": "inf", "det": "det", "dim": "dim", "ker": "ker", "deg": "deg",
	"gcd": "gcd", "arg": "arg", "mod": "mod", "bmod": "mod",
	"quad": " ", "qquad": "  ", "lvert": "|", "rvert": "|", "vert": "|", "mid": "|",
	"langle": "⟨", "rangle": "⟩", "lfloor": "⌊", "rfloor": "⌋", "lceil": "⌈", "rceil": "⌉",
	"star": "⋆", "oplus": "⊕", "otimes": "⊗", "perp": "⊥", "parallel": "∥", "because": "∵",
	"therefore": "∴", "aleph": "ℵ", "hbar": "ℏ", "ell": "ℓ", "Re": "ℜ", "Im": "ℑ",
}

func matIsBigOp(name string) (string, bool) {
	switch name {
	case "sum":
		return "∑", true
	case "prod":
		return "∏", true
	case "int", "iint", "iiint", "oint":
		return matSymbols[name], true
	}
	return "", false
}

// parse 解析整段公式。
func matParse(src string) []matExpr {
	p := &matParser{toks: matLex(src)}
	list, _ := p.parseSeq(false)
	return list
}

// parseSeq 解析到终点；inGroup 时遇到 '}' 结束。
func (p *matParser) parseSeq(inGroup bool) ([]matExpr, bool) {
	var out []matExpr
	for p.pos < len(p.toks) {
		t := p.toks[p.pos]
		switch t.c {
		case '}':
			if inGroup {
				p.pos++
				return out, true
			}
			p.pos++
			continue
		case '&', '\\':
			return out, false
		case '^', '_':
			p.pos++
			arg := p.parseArg()
			if len(out) == 0 {
				out = append(out, matExpr{kind: meSeq})
			}
			last := out[len(out)-1]
			wrap := matExpr{kind: meScript, base: []matExpr{last}}
			if last.kind == meScript {
				wrap = last
			}
			if t.c == '^' {
				wrap.sup = arg
			} else {
				wrap.sub = arg
			}
			out[len(out)-1] = wrap
			continue
		}
		e, ok := p.parseOne()
		if !ok {
			p.pos++
			continue
		}
		out = append(out, e)
	}
	return out, false
}

// parseArg 解析一个参数：{...} 或单个 token。
func (p *matParser) parseArg() []matExpr {
	if p.pos >= len(p.toks) {
		return nil
	}
	if p.toks[p.pos].c == '{' {
		p.pos++
		list, _ := p.parseSeq(true)
		return list
	}
	e, ok := p.parseOne()
	if !ok {
		p.pos++
		return nil
	}
	return []matExpr{e}
}

// parseBrace 读取 {...} 内的表达式（允许缺失）。
func (p *matParser) parseBrace() []matExpr {
	for p.pos < len(p.toks) && (p.toks[p.pos].c == ' ' || p.toks[p.pos].c == '~') {
		p.pos++
	}
	if p.pos >= len(p.toks) {
		return nil
	}
	return p.parseArg()
}

func (p *matParser) parseOne() (matExpr, bool) {
	if p.pos >= len(p.toks) {
		return matExpr{}, false
	}
	t := p.toks[p.pos]
	switch t.c {
	case 'r':
		p.pos++
		return matExpr{kind: meAtom, text: string(t.ch)}, true
	case '{':
		p.pos++
		list, _ := p.parseSeq(true)
		return matExpr{kind: meSeq, items: list}, true
	case '&':
		p.pos++
		return matExpr{kind: meAtom, text: "&"}, true
	case '\\':
		p.pos++
		return matExpr{kind: meAtom, text: " "}, true
	case '~':
		p.pos++
		return matExpr{kind: meSpace, em: 0.35}, true
	case 'w':
		p.pos++
		return p.parseCmd(t.name), true
	}
	p.pos++
	return matExpr{kind: meAtom, text: string(t.c)}, true
}

func (p *matParser) parseCmd(name string) matExpr {
	switch name {
	case "frac", "dfrac", "tfrac":
		num := p.parseBrace()
		den := p.parseBrace()
		return matExpr{kind: meFrac, num: num, den: den}
	case "sqrt":
		var idx []matExpr
		if p.pos < len(p.toks) && p.toks[p.pos].c == 'r' && p.toks[p.pos].ch == '[' {
			p.pos++
			var buf []matExpr
			for p.pos < len(p.toks) && !(p.toks[p.pos].c == 'r' && p.toks[p.pos].ch == ']') {
				e, ok := p.parseOne()
				if !ok {
					break
				}
				buf = append(buf, e)
			}
			p.pos++ // ']'
			idx = buf
		}
		body := p.parseBrace()
		return matExpr{kind: meSqrt, base: body, items: idx}
	case "text", "mathrm", "operatorname":
		return matExpr{kind: meAtom, text: p.rawBraceText()}
	case "mathbf", "textbf":
		return matExpr{kind: meAtom, text: p.rawBraceText()}
	case "mathbb":
		s := p.rawBraceText()
		return matExpr{kind: meAtom, text: matBlackboard(s)}
	case "left":
		left := p.delimTok()
		var buf []matExpr
		for p.pos < len(p.toks) {
			if p.toks[p.pos].c == 'w' && p.toks[p.pos].name == "right" {
				p.pos++
				break
			}
			if p.toks[p.pos].c == '}' {
				break
			}
			e, ok := p.parseOne()
			if !ok {
				p.pos++
				break
			}
			for p.pos < len(p.toks) && (p.toks[p.pos].c == '^' || p.toks[p.pos].c == '_') {
				kind := p.toks[p.pos].c
				p.pos++
				arg := p.parseArg()
				wrap := matExpr{kind: meScript, base: []matExpr{e}}
				if kind == '^' {
					wrap.sup = arg
				} else {
					wrap.sub = arg
				}
				e = wrap
			}
			buf = append(buf, e)
		}
		right := p.delimTok()
		return matExpr{kind: meDelim, base: buf, left: left, right: right}
	case "right":
		return matExpr{kind: meAtom, text: p.delimTok()}
	case "hat", "widehat":
		return matExpr{kind: meAccent, acc: "hat", base: p.parseBrace()}
	case "bar", "overline":
		return matExpr{kind: meAccent, acc: "bar", base: p.parseBrace()}
	case "vec":
		return matExpr{kind: meAccent, acc: "vec", base: p.parseBrace()}
	case "tilde", "widetilde":
		return matExpr{kind: meAccent, acc: "tilde", base: p.parseBrace()}
	case "dot":
		return matExpr{kind: meAccent, acc: "dot", base: p.parseBrace()}
	case "begin":
		env := p.rawBraceText()
		return p.parseEnv(env)
	case "end":
		p.rawBraceText()
		return matExpr{}
	}
	if g, big := matIsBigOp(name); big {
		upper := p.collectLimits('^')
		lower := p.collectLimits('_')
		return matExpr{kind: meBig, text: g, upper: upper, lower: lower}
	}
	if s, ok := matSymbols[name]; ok {
		return matExpr{kind: meAtom, text: s}
	}
	return matExpr{kind: meAtom, text: name}
}

// collectLimits 读取 ^ / _ 后面的表达式（大运算符上下限）。
func (p *matParser) collectLimits(kind rune) []matExpr {
	if p.pos < len(p.toks) && p.toks[p.pos].c == kind {
		p.pos++
		return p.parseArg()
	}
	return nil
}

// rawBraceText 读取 {...} 内的原始文本（\text / \mathrm 用）。
func (p *matParser) rawBraceText() string {
	for p.pos < len(p.toks) && p.toks[p.pos].c == '~' {
		p.pos++
	}
	if p.pos >= len(p.toks) || p.toks[p.pos].c != '{' {
		if p.pos < len(p.toks) {
			t := p.toks[p.pos]
			p.pos++
			if t.c == 'w' {
				return t.name
			}
			if t.c == 'r' {
				return string(t.ch)
			}
		}
		return ""
	}
	p.pos++
	var sb strings.Builder
	depth := 1
	for p.pos < len(p.toks) {
		t := p.toks[p.pos]
		p.pos++
		switch t.c {
		case '{':
			depth++
			sb.WriteString("{")
		case '}':
			depth--
			if depth == 0 {
				return sb.String()
			}
			sb.WriteString("}")
		case 'w':
			if s, ok := matSymbols[t.name]; ok {
				sb.WriteString(s)
			} else {
				sb.WriteString(t.name)
			}
		case 'r':
			sb.WriteRune(t.ch)
		case '~':
			sb.WriteString(" ")
		case '^', '_':
			sb.WriteRune(t.c)
		case '&':
			sb.WriteString("&")
		}
	}
	return sb.String()
}

// delimTok 读取 \left 后面的括号字符。
func (p *matParser) delimTok() string {
	if p.pos >= len(p.toks) {
		return ""
	}
	t := p.toks[p.pos]
	p.pos++
	switch t.c {
	case 'r':
		return string(t.ch)
	case 'w':
		if s, ok := matSymbols[t.name]; ok {
			return s
		}
		return t.name
	}
	return ""
}

// parseEnv 解析 \begin{...}...\end{...} 环境（矩阵 / cases / aligned）。
// 用 '&' 切列、'\\' 切行；行内多余的 '&' 忽略，缺列时补空。
func (p *matParser) parseEnv(env string) matExpr {
	var rows [][]matExpr
	var row []matExpr
	var cell []matExpr
	flushCell := func() {
		row = append(row, matExpr{kind: meSeq, items: cell})
		cell = nil
	}
	for p.pos < len(p.toks) {
		t := p.toks[p.pos]
		if t.c == 'w' && t.name == "end" {
			p.pos++
			p.rawBraceText()
			break
		}
		if t.c == '\\' {
			p.pos++
			flushCell()
			rows = append(rows, row)
			row = nil
			continue
		}
		if t.c == '&' {
			p.pos++
			flushCell()
			continue
		}
		e, ok := p.parseOne()
		if !ok {
			p.pos++
			continue
		}
		for p.pos < len(p.toks) && (p.toks[p.pos].c == '^' || p.toks[p.pos].c == '_') {
			kind := p.toks[p.pos].c
			p.pos++
			arg := p.parseArg()
			wrap := matExpr{kind: meScript, base: []matExpr{e}}
			if kind == '^' {
				wrap.sup = arg
			} else {
				wrap.sub = arg
			}
			e = wrap
		}
		cell = append(cell, e)
	}
	if len(cell) > 0 || len(row) > 0 {
		flushCell()
		rows = append(rows, row)
	}

	left, right := "", ""
	switch env {
	case "pmatrix":
		left, right = "(", ")"
	case "bmatrix":
		left, right = "[", "]"
	case "Bmatrix":
		left, right = "{", "}"
	case "vmatrix":
		left, right = "|", "|"
	case "Vmatrix":
		left, right = "‖", "‖"
	case "cases":
		left, right = "{", "."
	}
	return matExpr{kind: meMatrix, rows: rows, left: left, right: right}
}

// matBlackboard 黑板粗体（常见集合符号）。
func matBlackboard(s string) string {
	m := map[rune]rune{'R': 'ℝ', 'N': 'ℕ', 'Z': 'ℤ', 'Q': 'ℚ', 'C': 'ℂ', 'P': 'ℙ', 'E': '𝔼'}
	var sb strings.Builder
	for _, r := range s {
		if v, ok := m[r]; ok {
			sb.WriteRune(v)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ---- 对外 ----

// matBoxNode 把排版盒转成固定尺寸节点（Stack + 绝对定位子项）。
func matBoxNode(b matBox) *Node {
	kids := make([]*Node, 0, len(b.kids))
	for _, k := range b.kids {
		if k.n == nil {
			continue
		}
		k.n.St(St().AbsTL(k.x, k.y))
		kids = append(kids, k.n)
	}
	return Stack(kids...).St(St().W(b.w + 1).H(b.h + 1))
}

// mdMathTypeset 排版公式并返回节点（失败返回 nil）。
func mdMathTypeset(src string, size float32, family, color string, display bool) *Node {
	b, ok := mdMathBox(src, size, family, color, display)
	if !ok {
		return nil
	}
	return matBoxNode(b)
}

// mdMathBox 排版公式并返回盒（供行内排版取宽高）。
func mdMathBox(src string, size float32, family, color string, display bool) (matBox, bool) {
	src = strings.TrimSpace(src)
	if src == "" {
		return matBox{}, false
	}
	if display {
		size *= 1.06
	}
	c := &matCtx{size: size, family: family, color: color}
	list := matParse(src)
	if len(list) == 0 {
		return matBox{}, false
	}
	b := c.layout(list, size)
	if b.w <= 0 {
		return matBox{}, false
	}
	return b, true
}

// mdMathTextWidth 估算公式宽度（无窗口/失败时的兜底）。
func mdMathTextWidth(src string, size float32) float32 {
	return float32(len([]rune(src))) * size * 0.55
}

// matNumber 数字/单位拼接（调试与测试辅助）。
func matNumber(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}
