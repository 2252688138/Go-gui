// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "strings"

// Mermaid 流程图（flowchart / graph）原生绘制。
//
// GoGUI 没有 JavaScript，也无法内嵌 mermaid.js，所以这里自己解析 flowchart 源码
// 并用「原生节点 + 正交连线」画出来：
//
//	解析：A[文本] -->|标签| B{判断}，支持链式 A --> B --> C
//	分层：按入边 BFS 分层（环用松弛兜底），TD/BT 纵排、LR/RL 横排
//	绘制：节点是圆角盒 + 文本，连线是轴对齐线段 + 箭头字形
//
// 仅覆盖 flow 图；时序、类图、甘特等仍降级为提示卡。

const (
	mmShapeRect = iota
	mmShapeRound
	mmShapeDiamond
	mmShapeCircle
	mmShapeStadium
	mmShapeSubroutine
)

// mmPad 画布内边距（节点坐标原点偏移）。
const mmPad = 14

type mmNodeT struct {
	id        string
	label     string
	shape     int
	w, h      float32
	layer     int
	x, y      float32
	firstSeen int
}

type mmEdgeT struct {
	from, to string
	label    string
	dashed   bool
}

type mmGraph struct {
	nodes []*mmNodeT
	edges []mmEdgeT
	dir   string
}

// mmArrowToks 支持的连线写法（长记号在前）。
var mmArrowToks = []string{"-.->", "-.-", "==>", "===", "-->", "---", "--o", "--x", "->"}

// mmParse 解析 flowchart 源码。
func mmParse(src string) (*mmGraph, bool) {
	g := &mmGraph{dir: "TD"}
	started := false
	seen := map[string]*mmNodeT{}
	get := func(id string) *mmNodeT {
		if n, ok := seen[id]; ok {
			return n
		}
		n := &mmNodeT{id: id, label: id, firstSeen: len(g.nodes)}
		seen[id] = n
		g.nodes = append(g.nodes, n)
		return n
	}

	for _, raw := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		low := strings.ToLower(line)
		switch {
		case strings.HasPrefix(low, "flowchart"), strings.HasPrefix(low, "graph"):
			started = true
			if f := strings.Fields(low); len(f) > 1 {
				switch strings.ToUpper(f[1]) {
				case "TD", "TB":
					g.dir = "TD"
				case "BT":
					g.dir = "BT"
				case "LR":
					g.dir = "LR"
				case "RL":
					g.dir = "RL"
				}
			}
			continue
		case strings.HasPrefix(low, "subgraph"), low == "end",
			strings.HasPrefix(low, "classdef"), strings.HasPrefix(low, "class "),
			strings.HasPrefix(low, "style"), strings.HasPrefix(low, "linkstyle"),
			strings.HasPrefix(low, "click"), strings.HasPrefix(low, "direction"):
			continue
		}
		if !started {
			continue
		}
		mmParseFlowLine(line, get, g)
	}
	if len(g.nodes) == 0 {
		return nil, false
	}
	return g, true
}

// mmParseFlowLine 解析一行 `A[文本] -->|标签| B`（支持链式）。
// 箭头标签属于「紧随其后创建的边」，所以先存成 pending 再随边落下。
func mmParseFlowLine(line string, get func(string) *mmNodeT, g *mmGraph) {
	rs := []rune(line)
	i := 0
	var prev *mmNodeT
	pendingLabel := ""
	pendingDashed := false
	for i < len(rs) {
		for i < len(rs) && (rs[i] == ' ' || rs[i] == '\t' || rs[i] == ';') {
			i++
		}
		if i >= len(rs) {
			return
		}
		start := i
		for i < len(rs) && !isMMStop(rs[i]) {
			i++
		}
		if i == start {
			i++
			continue
		}
		n := get(string(rs[start:i]))
		if label, shape, next, ok := mmReadShape(rs, i); ok {
			if label != "" {
				n.label = label
			}
			n.shape = shape
			i = next
		}
		if prev != nil {
			g.edges = append(g.edges, mmEdgeT{
				from: prev.id, to: n.id, label: pendingLabel, dashed: pendingDashed,
			})
			pendingLabel, pendingDashed = "", false
		}
		prev = n
		for i < len(rs) && (rs[i] == ' ' || rs[i] == '\t') {
			i++
		}
		label, dashed, next, ok := mmReadArrow(rs, i)
		if !ok {
			return
		}
		i = next
		pendingLabel, pendingDashed = label, dashed
	}
}

// isMMStop 节点 ID 的终止字符。
func isMMStop(r rune) bool {
	switch r {
	case ' ', '\t', '-', '=', '.', '[', '(', '{', '>', ']', ')', '}', '&', ';', ',':
		return true
	}
	return false
}

// mmReadShape 读取节点形状与标签。
func mmReadShape(rs []rune, i int) (label string, shape int, next int, ok bool) {
	if i >= len(rs) {
		return "", 0, i, false
	}
	two := ""
	if i+1 < len(rs) {
		two = string(rs[i : i+2])
	}
	switch two {
	case "((":
		return mmReadDelim(rs, i+2, "))", mmShapeCircle)
	case "([":
		return mmReadDelim(rs, i+2, "])", mmShapeStadium)
	case "[[":
		return mmReadDelim(rs, i+2, "]]", mmShapeSubroutine)
	case "{{":
		return mmReadDelim(rs, i+2, "}}", mmShapeRect)
	}
	switch rs[i] {
	case '[':
		return mmReadDelim(rs, i+1, "]", mmShapeRect)
	case '(':
		return mmReadDelim(rs, i+1, ")", mmShapeRound)
	case '{':
		return mmReadDelim(rs, i+1, "}", mmShapeDiamond)
	case '>':
		return mmReadDelim(rs, i+1, "]", mmShapeRect)
	}
	return "", 0, i, false
}

// mmReadDelim 读取到闭合标记为止的标签。
func mmReadDelim(rs []rune, from int, close string, shape int) (string, int, int, bool) {
	if from > len(rs) {
		return "", 0, from, false
	}
	body := string(rs[from:])
	k := strings.Index(body, close)
	if k < 0 {
		return "", 0, from, false
	}
	text := strings.TrimSpace(body[:k])
	text = strings.Trim(text, "\"'")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	next := from + len([]rune(body[:k])) + len([]rune(close))
	return text, shape, next, true
}

// mmReadArrow 读取箭头与可选标签。
func mmReadArrow(rs []rune, i int) (label string, dashed bool, next int, ok bool) {
	rest := string(rs[i:])
	tok := ""
	for _, t := range mmArrowToks {
		if strings.HasPrefix(rest, t) {
			tok = t
			break
		}
	}
	if tok == "" {
		return "", false, i, false
	}
	i += len([]rune(tok))
	dashed = strings.HasPrefix(tok, "-.")
	if i < len(rs) && rs[i] == '|' {
		j := i + 1
		for j < len(rs) && rs[j] != '|' {
			j++
		}
		if j < len(rs) {
			label = strings.TrimSpace(string(rs[i+1 : j]))
			i = j + 1
		}
	}
	return label, dashed, i, true
}

// ---- 布局 ----

// mmLayout 分层与定位，返回总宽高。
func (g *mmGraph) layout(size float32, family string) (float32, float32) {
	for _, n := range g.nodes {
		w := size * 2.4
		f := mdFont(size, 600, family)
		for _, ln := range strings.Split(n.label, "\n") {
			if f != nil {
				if v := f.Measure(ln); v > w {
					w = v
				}
			} else if v := float32(len([]rune(ln))) * size * 0.55; v > w {
				w = v
			}
		}
		lines := float32(strings.Count(n.label, "\n") + 1)
		h := size*1.6*lines + 14
		if n.shape == mmShapeDiamond {
			w += size * 2
			h += 4
		}
		if n.shape == mmShapeCircle && w < h*1.5 {
			w = h * 1.5
		}
		n.w = w + 22
		n.h = h
	}

	layer := map[string]int{}
	indeg := map[string]int{}
	adj := map[string][]string{}
	for _, e := range g.edges {
		adj[e.from] = append(adj[e.from], e.to)
		indeg[e.to]++
	}
	queue := make([]string, 0, len(g.nodes))
	for _, n := range g.nodes {
		if indeg[n.id] == 0 {
			layer[n.id] = 0
			queue = append(queue, n.id)
		}
	}
	for i := 0; i < len(queue); i++ {
		cur := queue[i]
		for _, to := range adj[cur] {
			if v := layer[cur] + 1; v > layer[to] {
				layer[to] = v
			}
			indeg[to]--
			if indeg[to] <= 0 {
				queue = append(queue, to)
			}
		}
	}
	for _, n := range g.nodes {
		n.layer = layer[n.id]
	}
	// 环：按边松弛，保证每条边至少向下走一层
	for iter := 0; iter < len(g.nodes)+2; iter++ {
		changed := false
		for _, e := range g.edges {
			a, b := g.node(e.from), g.node(e.to)
			if a == nil || b == nil || a == b {
				continue
			}
			if b.layer <= a.layer && b.layer < 60 {
				b.layer = a.layer + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	maxLayer := 0
	for _, n := range g.nodes {
		if n.layer > maxLayer {
			maxLayer = n.layer
		}
	}
	rows := make([][]*mmNodeT, maxLayer+1)
	for _, n := range g.nodes {
		rows[n.layer] = append(rows[n.layer], n)
	}

	vertical := g.dir == "TD" || g.dir == "BT"
	gapMain, gapCross := float32(48), float32(34)
	var totalW, totalH float32
	if vertical {
		for li, row := range rows {
			var rh, rw float32
			for _, n := range row {
				if n.h > rh {
					rh = n.h
				}
			}
			for i, n := range row {
				if i > 0 {
					rw += gapCross
				}
				rw += n.w
			}
			if rw > totalW {
				totalW = rw
			}
			if li > 0 {
				totalH += gapMain
			}
			totalH += rh
		}
		y := float32(0)
		for _, row := range rows {
			var rh, rw float32
			for _, n := range row {
				if n.h > rh {
					rh = n.h
				}
			}
			for i, n := range row {
				if i > 0 {
					rw += gapCross
				}
				rw += n.w
			}
			x := (totalW - rw) / 2
			for _, n := range row {
				n.x = x
				n.y = y + (rh-n.h)/2
				x += n.w + gapCross
			}
			y += rh + gapMain
		}
	} else {
		for li, row := range rows {
			var cw, ch float32
			for _, n := range row {
				if n.w > cw {
					cw = n.w
				}
			}
			for i, n := range row {
				if i > 0 {
					ch += gapCross
				}
				ch += n.h
			}
			if ch > totalH {
				totalH = ch
			}
			if li > 0 {
				totalW += gapMain
			}
			totalW += cw
		}
		x := float32(0)
		for _, row := range rows {
			var cw, ch float32
			for _, n := range row {
				if n.w > cw {
					cw = n.w
				}
			}
			for i, n := range row {
				if i > 0 {
					ch += gapCross
				}
				ch += n.h
			}
			y := (totalH - ch) / 2
			for _, n := range row {
				n.x = x + (cw-n.w)/2
				n.y = y
				y += n.h + gapCross
			}
			x += cw + gapMain
		}
	}
	if g.dir == "BT" {
		for _, n := range g.nodes {
			n.y = totalH - n.y - n.h
		}
	}
	if g.dir == "RL" {
		for _, n := range g.nodes {
			n.x = totalW - n.x - n.w
		}
	}
	return totalW, totalH
}

func (g *mmGraph) node(id string) *mmNodeT {
	for _, n := range g.nodes {
		if n.id == id {
			return n
		}
	}
	return nil
}

// ---- 绘制 ----

// mmView 渲染流程图。
func mmView(g *mmGraph, width, size float32, family, key string) (*Node, bool) {
	totalW, totalH := g.layout(size, family)
	if totalW <= 0 || totalH <= 0 {
		return nil, false
	}
	canvasW := totalW + mmPad*2
	canvasH := totalH + mmPad*2

	lineCol := "rgba(148,163,184,0.55)"
	kids := make([]*Node, 0, len(g.nodes)+len(g.edges)*4)
	for _, e := range g.edges {
		from, to := g.node(e.from), g.node(e.to)
		if from == nil || to == nil || from == to {
			continue
		}
		kids = append(kids, mmEdgeNodes(g, from, to, e, lineCol, size)...)
	}
	for _, n := range g.nodes {
		kids = append(kids, mmNodeBox(n, size, family))
	}
	inner := Stack(kids...).St(St().W(canvasW).H(canvasH))

	if width > 0 && canvasW > width {
		off := mdState.codeX["mm-x-"+key]
		maxOff := canvasW - width
		if off > maxOff {
			off = maxOff
		}
		if off < 0 {
			off = 0
		}
		inner.St(St().AbsTL(-off, 0))
		view := Box(inner).St(St().W(width).H(canvasH).Clipped())
		bar := Row(
			mmPanBtn("‹", key, -width*0.7, maxOff),
			Text("拖动 / 点击左右查看").St(St().Text(10, 400, mdColMuted)),
			mmPanBtn("›", key, width*0.7, maxOff),
		).St(St().GapOf(8)).AlignTo(AlignCenter)
		return Column(view, bar).St(St().GapOf(4)), true
	}
	return Box(inner).St(St().W(canvasW).H(canvasH)), true
}

func mmPanBtn(label, key string, delta, maxOff float32) *Node {
	n := Text(label).St(St().Text(11, 700, mdColMuted).PadXY(8, 2).R(6).BgC(ColHover))
	n.Tap(func() {
		x := mdState.codeX["mm-x-"+key] + delta
		if x < 0 {
			x = 0
		}
		if x > maxOff {
			x = maxOff
		}
		mdState.codeX["mm-x-"+key] = x
		RequestRedraw()
	})
	return n
}

// mmNodeBox 节点盒（圆角盒 / 胶囊 / 菱形用不同底色与圆角近似）。
func mmNodeBox(n *mmNodeT, size float32, family string) *Node {
	x, y := n.x+mmPad, n.y+mmPad
	bg, bd := mdColCodeBg, mdColAccent
	radius := float32(8)
	switch n.shape {
	case mmShapeRound, mmShapeStadium, mmShapeCircle:
		radius = n.h / 2
	case mmShapeDiamond:
		bd, bg = "#fbbf24", "rgba(251,191,36,0.10)"
	case mmShapeSubroutine:
		bd = "rgba(52,211,153,0.75)"
	}
	box := Box().St(St().W(n.w).H(n.h).R(radius).BgC(bg).Stroke(1.2, bd).AbsTL(x, y))
	txtSt := St().Text(size, 600, mdColText).WrapOn().TAlign(AlignCenter).CenterYOn().
		W(n.w-14).H(n.h).AbsTL(x+7, y)
	if family != "" {
		txtSt.Font(family)
	}
	return Stack(box, Text(n.label).St(txtSt))
}

// mmEdgeNodes 一条边的线段 + 箭头 + 标签。
func mmEdgeNodes(g *mmGraph, from, to *mmNodeT, e mmEdgeT, color string, size float32) []*Node {
	thick := float32(1.5)
	var out []*Node
	vertical := g.dir == "TD" || g.dir == "BT"

	if vertical {
		sx := from.x + mmPad + from.w/2
		tx := to.x + mmPad + to.w/2
		var sy, ty float32
		if g.dir == "TD" {
			sy, ty = from.y+mmPad+from.h, to.y+mmPad
		} else {
			sy, ty = from.y+mmPad, to.y+mmPad+to.h
		}
		mid := (sy + ty) / 2
		out = append(out, mmVLine(sx, sy, mid, thick, color))
		out = append(out, mmHLine(minf(sx, tx), maxf(sx, tx), mid, thick, color))
		out = append(out, mmVLine(tx, mid, ty, thick, color))
		arrow := "▼"
		if g.dir == "BT" {
			arrow = "▲"
		}
		out = append(out, Text(arrow).St(St().Text(size*0.8, 700, mdColAccent).
			W(12).H(12).TAlign(AlignCenter).AbsTL(tx-6, ty-7)))
		if e.label != "" {
			out = append(out, Text(e.label).St(St().Text(size*0.76, 500, mdColDim).
				PadXY(4, 1).R(4).BgC(mdColBgSoft).
				AbsTL((sx+tx)/2-float32(len([]rune(e.label)))*size*0.24, mid-size*0.62)))
		}
		return out
	}

	sy := from.y + mmPad + from.h/2
	ty := to.y + mmPad + to.h/2
	var sx, tx float32
	if g.dir == "LR" {
		sx, tx = from.x+mmPad+from.w, to.x+mmPad
	} else {
		sx, tx = from.x+mmPad, to.x+mmPad+to.w
	}
	mid := (sx + tx) / 2
	out = append(out, mmHLine(sx, mid, sy, thick, color))
	out = append(out, mmVLine(mid, minf(sy, ty), maxf(sy, ty), thick, color))
	out = append(out, mmHLine(minf(mid, tx), maxf(mid, tx), ty, thick, color))
	arrow := "▶"
	if g.dir == "RL" {
		arrow = "◀"
	}
	out = append(out, Text(arrow).St(St().Text(size*0.8, 700, mdColAccent).
		W(12).H(12).TAlign(AlignCenter).AbsTL(tx-6, ty-7)))
	if e.label != "" {
		out = append(out, Text(e.label).St(St().Text(size*0.76, 500, mdColDim).
			PadXY(4, 1).R(4).BgC(mdColBgSoft).
			AbsTL(mid-size*0.4, (sy+ty)/2-size*0.7)))
	}
	return out
}

// mmHLine / mmVLine 轴对齐线段（画布坐标，横竖都按粗度居中）。
func mmHLine(x0, x1, y, thick float32, color string) *Node {
	return Box().St(St().W(maxf(x1-x0, thick)).H(thick).BgC(color).AbsTL(x0, y-thick/2))
}

func mmVLine(x, y0, y1, thick float32, color string) *Node {
	return Box().St(St().W(thick).H(maxf(y1-y0, thick)).BgC(color).AbsTL(x-thick/2, y0))
}

// mmHasFlow 是否包含流程图定义。
func mmHasFlow(src string) bool {
	for _, l := range strings.Split(src, "\n") {
		t := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(t, "flowchart") || strings.HasPrefix(t, "graph") {
			return true
		}
	}
	return false
}
