// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"strings"
)

// 富文本编辑器：行内样式（粗/斜/下划线/删除线/代码/颜色/字号）+
// 选区 + 输入法 + 撤销。基于 RichText 的行内折行数据做光标定位与选区高亮。
//
//	st := ui.NewRichEditor("doc")
//	ui.RichToolbar(st, nil)                       // 工具栏
//	ui.RichEditor(st, 420, func() { markDirty() }) // 编辑区
//
// 限制：只编辑纯文本 + 行内样式，不含图片/表格（可作为后续扩展点）。

// RichStyle 行内样式。
type RichStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Code      bool
	Color     string
	Size      float32
	Link      string
}

// RichRun 一段连续同一样式的文字。
type RichRun struct {
	Text  string
	Style RichStyle
}

// RichEditorState 编辑器状态。
type RichEditorState struct {
	Key    string
	Runs   []RichRun
	Caret  int // rune 下标
	Anchor int // 选区锚点（与 Caret 不同则有选区）
	// Style 新输入文字的样式（工具栏修改）。
	Style RichStyle
	Undo  *UndoStack
}

// NewRichEditor 创建编辑器（默认一段空文本）。
func NewRichEditor(key string) *RichEditorState {
	st := &RichEditorState{Key: key, Undo: NewUndo()}
	st.Runs = []RichRun{{Text: "", Style: RichStyle{}}}
	return st
}

// Text 编辑器全文。
func (st *RichEditorState) Text() string {
	var b strings.Builder
	for _, r := range st.Runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

// Selection 返回选区 [lo, hi)（无选区时 lo==hi==caret）。
func (st *RichEditorState) Selection() (int, int) {
	if st == nil {
		return 0, 0
	}
	lo, hi := st.Caret, st.Anchor
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi
}

func runeLen(s string) int { return len([]rune(s)) }

// totalRunes 全文长度。
func (st *RichEditorState) totalRunes() int {
	n := 0
	for _, r := range st.Runs {
		n += runeLen(r.Text)
	}
	return n
}

// replace 用 text 替换选区并把样式记为 style。
func (st *RichEditorState) replace(text string, style RichStyle, record bool) {
	if st == nil {
		return
	}
	lo, hi := st.Selection()
	before := st.Text()
	// 展平成 rune + style 序列
	type cs struct {
		r rune
		s RichStyle
	}
	var flat []cs
	for _, run := range st.Runs {
		for _, r := range run.Text {
			flat = append(flat, cs{r: r, s: run.Style})
		}
	}
	if lo < 0 {
		lo = 0
	}
	if hi > len(flat) {
		hi = len(flat)
	}
	var next []cs
	next = append(next, flat[:lo]...)
	for _, r := range text {
		next = append(next, cs{r: r, s: style})
	}
	caret := lo + runeLen(text)
	next = append(next, flat[hi:]...)
	// 合并为 runs
	runs := make([]RichRun, 0, 8)
	for _, c := range next {
		if n := len(runs); n > 0 && runs[n-1].Style == c.s {
			runs[n-1].Text += string(c.r)
			continue
		}
		runs = append(runs, RichRun{Text: string(c.r), Style: c.s})
	}
	if len(runs) == 0 {
		runs = []RichRun{{Text: "", Style: style}}
	}
	st.Runs = runs
	st.Caret, st.Anchor = caret, caret
	after := st.Text()
	if record && st.Undo != nil && before != after {
		st.Undo.Push("编辑", func() { st.setPlain(before) }, func() { st.setPlain(after) })
	}
	RequestRedraw()
}

// setPlain 整体替换文本（撤销用，保留首段样式）。
func (st *RichEditorState) setPlain(text string) {
	style := RichStyle{}
	if len(st.Runs) > 0 {
		style = st.Runs[0].Style
	}
	st.Runs = []RichRun{{Text: text, Style: style}}
	if st.Caret > runeLen(text) {
		st.Caret = runeLen(text)
	}
	st.Anchor = st.Caret
	RequestRedraw()
}

// Insert 在光标处插入文字。
func (st *RichEditorState) Insert(text string) { st.replace(text, st.Style, true) }

// Toggle 切换当前样式（有选区时应用到选区，否则作用于后续输入）。
func (st *RichEditorState) Toggle(apply func(*RichStyle)) {
	if st == nil {
		return
	}
	apply(&st.Style)
	lo, hi := st.Selection()
	if lo == hi {
		RequestRedraw()
		return
	}
	// 对选区内的 run 应用样式
	pos := 0
	for i := range st.Runs {
		rs := []rune(st.Runs[i].Text)
		start, end := pos, pos+len(rs)
		pos = end
		if end <= lo || start >= hi {
			continue
		}
		apply(&st.Runs[i].Style)
	}
	RequestRedraw()
}

// Bold 切换粗体。
func (st *RichEditorState) Bold() { st.Toggle(func(s *RichStyle) { s.Bold = !s.Bold }) }

// Italic 切换斜体。
func (st *RichEditorState) Italic() { st.Toggle(func(s *RichStyle) { s.Italic = !s.Italic }) }

// Underline 切换下划线。
func (st *RichEditorState) Underline() { st.Toggle(func(s *RichStyle) { s.Underline = !s.Underline }) }

// Strike 切换删除线。
func (st *RichEditorState) Strike() { st.Toggle(func(s *RichStyle) { s.Strike = !s.Strike }) }

// Code 切换行内代码。
func (st *RichEditorState) Code() { st.Toggle(func(s *RichStyle) { s.Code = !s.Code }) }

// SetColor 设置文字颜色。
func (st *RichEditorState) SetColor(c string) { st.Toggle(func(s *RichStyle) { s.Color = c }) }

// SetSize 设置字号。
func (st *RichEditorState) SetSize(v float32) { st.Toggle(func(s *RichStyle) { s.Size = v }) }

// spans 把 runs 转成 RichText 片段（为选区插入高亮与光标）。
func (st *RichEditorState) spans(focused bool) []TextSpan {
	lo, hi := st.Selection()
	var out []TextSpan
	pos := 0
	caretShown := focused && int(AnimTime()*2)%2 == 0
	for _, run := range st.Runs {
		rs := []rune(run.Text)
		start, end := pos, pos+len(rs)
		pos = end
		sp := TextSpan{
			Color: run.Style.Color, Size: run.Style.Size,
			Link: run.Style.Link, Underline: run.Style.Underline,
			Strike: run.Style.Strike, Code: run.Style.Code,
			Weight: floatWeight(run.Style),
		}
		if caretShown && st.Caret >= start && st.Caret <= end && lo == hi {
			cut := st.Caret - start
			if cut <= 0 {
				out = append(out, TextSpan{Text: "│", Color: ColPrimary, Weight: 600})
				out = append(out, withText(sp, run.Text))
			} else {
				out = append(out, withText(sp, string(rs[:cut])))
				out = append(out, TextSpan{Text: "│", Color: ColPrimary, Weight: 600})
				out = append(out, withText(sp, string(rs[cut:])))
			}
			continue
		}
		if lo != hi && start < hi && end > lo {
			// 选中部分加高亮底色
			cutLo := 0
			if lo > start {
				cutLo = lo - start
			}
			cutHi := len(rs)
			if hi < end {
				cutHi = hi - start
			}
			if cutLo > 0 {
				out = append(out, withText(sp, string(rs[:cutLo])))
			}
			sel := sp
			sel.BgColor = "rgba(99,102,241,0.30)"
			out = append(out, withText(sel, string(rs[cutLo:cutHi])))
			if cutHi < len(rs) {
				out = append(out, withText(sp, string(rs[cutHi:])))
			}
			continue
		}
		out = append(out, withText(sp, run.Text))
	}
	if len(out) == 0 {
		out = append(out, TextSpan{Text: "", Color: ColText})
	}
	return out
}

func withText(sp TextSpan, s string) TextSpan {
	sp.Text = s
	return sp
}

func floatWeight(s RichStyle) int32 {
	if s.Bold {
		return 700
	}
	return 400
}

// RichEditor 编辑区（可聚焦，支持输入法 / 方向键 / 选区 / 快捷键）。
func RichEditor(st *RichEditorState, height float32, onChange func()) *Node {
	if st == nil {
		return nil
	}
	if height <= 0 {
		height = 320
	}
	focused := IsFocused(st.Key)
	box := RichText(st.spans(focused), RichTextOptions{Size: 13, MaxLines: 0}).
		St(St().Wp(100).Padding(12).R(10).BgC(ColInput).
			Stroke(1, focusStroke(st.Key)).FocusOn().Clipped())
	rt := box
	n := Stack(
		Scroll(rt).K(st.Key + "-scroll").St(St().AbsFill().NoFocusRing()),
	).St(St().Wp(100).H(height)).K(st.Key)
	if st.Caret == 0 && !focused {
		n.Add(Text("开始输入…").St(St().AbsTL(14, 13).Text(13, 400, ColPlaceholder)))
	}
	changed := func() {
		if onChange != nil {
			onChange()
		}
		RequestRedraw()
	}
	n.OnClick = func() {
		SetFocus(st.Key)
		// 点击定位光标：按上一帧行内片段估算
		x, y := MousePos()
		if r, ok := LastFrameOf(st.Key + "-scroll"); ok {
			lx, ly := x-r.X-12, y-r.Y-12
			if rt.rich.valid {
				st.Caret = caretAtRich(rt, st, lx, ly)
				st.Anchor = st.Caret
			}
		}
		RequestRedraw()
	}
	n.OnChar = func(r rune) {
		if r == 8 {
			st.backspace()
			changed()
			return
		}
		if r == 127 {
			st.deleteForward()
			changed()
			return
		}
		if r == '\r' || r == '\n' {
			st.Insert("\n")
			changed()
			return
		}
		st.Insert(string(r))
		changed()
	}
	n.OnKey = func(k uint32) {
		shift := ShiftDown()
		ctrl := CtrlDown()
		switch k {
		case VK_LEFT:
			if ctrl {
				st.Caret = wordLeft(st.Text(), st.Caret)
			} else {
				st.Caret--
			}
			if !shift {
				st.Anchor = st.Caret
			}
		case VK_RIGHT:
			if ctrl {
				st.Caret = wordRight(st.Text(), st.Caret)
			} else {
				st.Caret++
			}
			if !shift {
				st.Anchor = st.Caret
			}
		case 0x24:
			st.Caret = lineStartOf(st.Text(), st.Caret)
			if !shift {
				st.Anchor = st.Caret
			}
		case 0x23:
			st.Caret = lineEndOf(st.Text(), st.Caret)
			if !shift {
				st.Anchor = st.Caret
			}
		case VK_BACK:
			st.backspace()
			changed()
		case VK_DELETE:
			st.deleteForward()
			changed()
		case 'B':
			if ctrl {
				st.Bold()
			}
		case 'I':
			if ctrl {
				st.Italic()
			}
		case 'U':
			if ctrl {
				st.Underline()
			}
		case VK_ESCAPE:
			SetFocus("")
		}
		st.clampCaret()
		RequestRedraw()
	}
	return n
}

func (st *RichEditorState) clampCaret() {
	n := st.totalRunes()
	if st.Caret < 0 {
		st.Caret = 0
	}
	if st.Caret > n {
		st.Caret = n
	}
	if st.Anchor < 0 {
		st.Anchor = 0
	}
	if st.Anchor > n {
		st.Anchor = n
	}
}

func (st *RichEditorState) backspace() {
	lo, hi := st.Selection()
	if lo == hi {
		if lo == 0 {
			return
		}
		st.Caret = lo - 1
		st.Anchor = st.Caret
		lo = st.Caret
	}
	st.deleteRange(lo, hi)
}

func (st *RichEditorState) deleteForward() {
	lo, hi := st.Selection()
	if lo == hi {
		if hi >= st.totalRunes() {
			return
		}
		hi++
		st.Anchor = st.Caret
	}
	st.deleteRange(lo, hi)
}

// deleteRange 删除 [lo,hi)（含撤销记录）。
func (st *RichEditorState) deleteRange(lo, hi int) {
	if lo >= hi {
		return
	}
	before := st.Text()
	var flat []rune
	for _, r := range st.Runs {
		flat = append(flat, []rune(r.Text)...)
	}
	flat = append(flat[:lo], flat[hi:]...)
	st.Runs = []RichRun{{Text: string(flat), Style: st.Runs[0].Style}}
	st.Caret, st.Anchor = lo, lo
	after := st.Text()
	if st.Undo != nil {
		st.Undo.Push("删除", func() { st.setPlain(before) }, func() { st.setPlain(after) })
	}
	RequestRedraw()
}

// caretAtRich 点击位置 → rune 下标（按行内片段近似）。
func caretAtRich(rt *Node, st *RichEditorState, x, y float32) int {
	if rt == nil || !rt.rich.valid {
		return st.Caret
	}
	line := 0
	for i, ln := range rt.rich.lines {
		if y >= ln.y && y <= ln.y+ln.h {
			line = i
			break
		}
	}
	best := 0
	if line < len(rt.rich.lines) {
		ln := rt.rich.lines[line]
		acc := 0
		for _, p := range ln.pieces {
			if x <= p.x+p.w/2 {
				break
			}
			acc += runeLen(p.text)
		}
		// 累加之前各行的字符数
		for i := 0; i < line && i < len(rt.rich.lines); i++ {
			for _, p := range rt.rich.lines[i].pieces {
				best += runeLen(p.text)
			}
		}
		best += acc
	}
	if best > st.totalRunes() {
		best = st.totalRunes()
	}
	return best
}

func wordLeft(s string, pos int) int {
	rs := []rune(s)
	if pos > len(rs) {
		pos = len(rs)
	}
	for pos > 0 && !isWordRune(rs[pos-1]) {
		pos--
	}
	for pos > 0 && isWordRune(rs[pos-1]) {
		pos--
	}
	return pos
}

func wordRight(s string, pos int) int {
	rs := []rune(s)
	if pos < 0 {
		pos = 0
	}
	for pos < len(rs) && !isWordRune(rs[pos]) {
		pos++
	}
	for pos < len(rs) && isWordRune(rs[pos]) {
		pos++
	}
	return pos
}

func isWordRune(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= 0x4E00 && r <= 0x9FFF
}

func lineStartOf(s string, pos int) int {
	rs := []rune(s)
	if pos > len(rs) {
		pos = len(rs)
	}
	for pos > 0 && rs[pos-1] != '\n' {
		pos--
	}
	return pos
}

func lineEndOf(s string, pos int) int {
	rs := []rune(s)
	if pos < 0 {
		pos = 0
	}
	for pos < len(rs) && rs[pos] != '\n' {
		pos++
	}
	return pos
}

// ---- 工具栏 ----

// RichToolbar 编辑器工具栏（粗/斜/下划线/删除线/代码/颜色/字号/撤销重做）。
func RichToolbar(st *RichEditorState, onChange func()) *Node {
	if st == nil {
		return nil
	}
	btn := func(label string, active bool, fn func()) *Node {
		color := ColTextDim
		var bg any = "transparent"
		if active {
			color = "#ffffff"
			bg = paintWithAlpha(ColPrimary, 0.35)
		}
		return Text(label).St(St().Text(13, 600, color).W(30).H(28).R(6).
			CenterYOn().TAlign(AlignCenter).BgC(bg)).Tap(func() {
			fn()
			if onChange != nil {
				onChange()
			}
			RequestRedraw()
		})
	}
	colors := make([]*Node, 0, 6)
	for _, c := range []string{ColText, "#8b93ff", ColSuccess, ColWarn, ColDanger} {
		col := c
		colors = append(colors, Box().St(St().W(18).H(18).R(5).BgC(col).Stroke(1, ColStroke)).
			Tap(func() {
				st.SetColor(col)
				if onChange != nil {
					onChange()
				}
			}))
	}
	return Row(
		btn("B", st.Style.Bold, st.Bold),
		btn("I", st.Style.Italic, st.Italic),
		btn("U", st.Style.Underline, st.Underline),
		btn("S", st.Style.Strike, st.Strike),
		btn("<>", st.Style.Code, st.Code),
		Box().St(St().W(1).H(20).BgC(ColLine)),
		Row(colors...).St(St().GapOf(4)),
		Spacer(),
		btn("↶", false, func() {
			if st.Undo != nil {
				st.Undo.Undo()
			}
		}),
		btn("↷", false, func() {
			if st.Undo != nil {
				st.Undo.Redo()
			}
		}),
	).St(St().GapOf(4).Padding(6).R(8).BgC(ColBgSoft).Stroke(1, ColStroke)).
		AlignTo(AlignCenter)
}

// RichEditorEx 工具栏 + 编辑区（常用组合）。
func RichEditorEx(st *RichEditorState, height float32, onChange func()) *Node {
	return Column(
		RichToolbar(st, onChange),
		RichEditor(st, height, onChange),
	).St(St().GapOf(8))
}

// RichEditorText 调试用：带样式标记的纯文本导出（"**粗**" 等）。
func RichEditorText(st *RichEditorState) string {
	if st == nil {
		return ""
	}
	var b strings.Builder
	for _, r := range st.Runs {
		t := r.Text
		if r.Style.Code {
			t = "`" + t + "`"
		}
		if r.Style.Bold {
			t = "**" + t + "**"
		}
		if r.Style.Italic {
			t = "_" + t + "_"
		}
		b.WriteString(t)
	}
	return b.String()
}

// RichEditorDebug 打印编辑器结构（调试）。
func RichEditorDebug(st *RichEditorState) string {
	if st == nil {
		return ""
	}
	return fmt.Sprintf("runs=%d caret=%d anchor=%d", len(st.Runs), st.Caret, st.Anchor)
}
