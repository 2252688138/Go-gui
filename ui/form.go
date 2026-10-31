// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 表单组件：FormItem / Select（下拉浮层）/ Textarea（多行输入）/ NumberInput /
// SearchInput / Chip / DatePicker（日历浮层）/ DateField（只读展示）+ 轻量校验。
//
// 下拉与日历是浮层：调用方需在根 Stack 末尾（和 TooltipLayer 同一层）调一次
// ui.SelectLayers() / ui.DateLayers()，否则浮层会被滚动容器裁剪或被内容盖住。
// 浮层锚点取控件上一帧布局矩形（LastFrameOf），因此定位不依赖调用方算坐标。

const (
	// FieldH 单行控件高度（与 Input 一致）。
	FieldH float32 = 38
	// LabelW 行内表单标签列宽。
	LabelW float32 = 92
)

// ---- 布局 ----

// FormItem 纵向表单行：标签（可带必填星号）+ 控件 + 错误提示。
func FormItem(label string, required bool, errMsg string, child *Node) *Node {
	col := Column().St(St().GapOf(6))
	if label != "" {
		head := Row(Text(label).St(St().Text(12.5, 500, ColTextDim)))
		if required {
			head.Add(Text("*").St(St().Text(12.5, 600, ColDanger).MarOnly(0, 0, 0, 4)))
		}
		col.Add(head.AlignTo(AlignCenter))
	}
	if child != nil {
		col.Add(child)
	}
	if errMsg != "" {
		col.Add(Text(errMsg).St(St().Text(11.5, 400, ColDanger)))
	}
	return col
}

// FormItemInline 横向表单行：左侧固定宽度标签，右侧控件占满剩余宽度。
func FormItemInline(label string, required bool, errMsg string, child *Node) *Node {
	head := Row(Text(label).St(St().Text(12.5, 500, ColTextDim)))
	if required {
		head.Add(Text("*").St(St().Text(12.5, 600, ColDanger).MarOnly(0, 0, 0, 4)))
	}
	right := Column().St(St().GapOf(4).GrowTo(1))
	if child != nil {
		right.Add(child)
	}
	if errMsg != "" {
		right.Add(Text(errMsg).St(St().Text(11.5, 400, ColDanger)))
	}
	return Row(head.St(St().W(LabelW)), right).St(St().GapOf(12)).AlignTo(AlignStart)
}

// FormFooter 表单操作行（右对齐按钮组）。
func FormFooter(buttons ...*Node) *Node {
	items := append([]*Node{Spacer()}, buttons...)
	return Row(items...).St(St().GapOf(10)).AlignTo(AlignCenter)
}

// ---- 校验 ----

// ValidateRequired 必填校验：空串返回错误文案。
func ValidateRequired(label, v string) string {
	if strings.TrimSpace(v) == "" {
		return "请填写" + label
	}
	return ""
}

// ValidateMinLen 最小长度校验（len 按 rune 计）。
func ValidateMinLen(label, v string, n int) string {
	if l := len([]rune(strings.TrimSpace(v))); l < n {
		return fmt.Sprintf("%s至少 %d 个字符", label, n)
	}
	return ""
}

// ValidateEmail 邮箱格式校验（够用的宽松规则）。
func ValidateEmail(label, v string) string {
	s := strings.TrimSpace(v)
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 || !strings.Contains(s[at+1:], ".") {
		return label + "格式不正确"
	}
	return ""
}

// ---- 下拉框 ----

// SelectOption 下拉选项。
type SelectOption struct {
	Value    string
	Label    string
	Hint     string // 右侧附加说明（可选）
	Disabled bool
}

// SelectState 下拉框状态（调用方长期持有，用 NewSelect 创建）。
type SelectState struct {
	Key   string
	Open  bool
	Hover int // 键盘/鼠标高亮项索引，-1 表示无
	Value string

	opts     []SelectOption
	onChange func(string)
}

// selectStates 已创建的 Select 状态（SelectLayers 统一渲染浮层）。
var selectStates = map[string]*SelectState{}

// NewSelect 创建下拉框状态（同一个 key 重复创建会覆盖注册项）。
func NewSelect(key string) *SelectState {
	s := &SelectState{Key: key, Hover: -1}
	selectStates[key] = s
	return s
}

// Close 收起下拉浮层。
func (s *SelectState) Close() {
	if s == nil {
		return
	}
	s.Open = false
	s.Hover = -1
	CloseOverlay(s.Key + "-sel")
}

// Toggle 展开 / 收起。
func (s *SelectState) Toggle() {
	if s == nil {
		return
	}
	s.Open = !s.Open
	if !s.Open {
		s.Hover = -1
		return
	}
	s.Hover = s.index()
	RequestRedraw()
}

func (s *SelectState) index() int {
	for i, o := range s.opts {
		if o.Value == s.Value {
			return i
		}
	}
	return -1
}

// pick 选中第 i 项并回调。
func (s *SelectState) pick(i int) {
	if i < 0 || i >= len(s.opts) || s.opts[i].Disabled {
		return
	}
	s.Close()
	s.Value = s.opts[i].Value
	if s.onChange != nil {
		s.onChange(s.Value)
	}
	RequestRedraw()
}

// move 键盘上下移动高亮项（跳过禁用项）。
func (s *SelectState) move(d int) {
	n := len(s.opts)
	if n == 0 {
		return
	}
	i := s.Hover
	for k := 0; k < n; k++ {
		i += d
		if i < 0 {
			i = n - 1
		}
		if i >= n {
			i = 0
		}
		if !s.opts[i].Disabled {
			s.Hover = i
			RequestRedraw()
			return
		}
	}
}

// selectKey 下拉框键盘操作：↑↓ 切换、回车确认、Esc 关闭、Home/End 跳首尾。
func selectKey(s *SelectState, k uint32) {
	switch k {
	case 0x28: // ↓
		if !s.Open {
			s.Open = true
		}
		s.move(1)
	case 0x26: // ↑
		if !s.Open {
			s.Open = true
		}
		s.move(-1)
	case 0x0D: // Enter
		if s.Open {
			s.pick(s.Hover)
		} else {
			s.Open = true
			s.Hover = s.index()
			RequestRedraw()
		}
	case 0x1B: // Esc
		s.Close()
		RequestRedraw()
	case 0x24: // Home
		s.Hover = 0
		s.move(0)
		if s.Hover < 0 {
			s.move(1)
		}
	case 0x23: // End
		s.Hover = len(s.opts) - 1
		if s.Hover >= 0 && s.opts[s.Hover].Disabled {
			s.move(-1)
		}
		RequestRedraw()
	}
	// 高亮项滚入可视区
	if s.Open {
		scrollListInto(s.Key+"-opt", s.Hover, selectRowH)
	}
}

// SelectBox 下拉框控件本体。
func SelectBox(s *SelectState, opts []SelectOption, value, placeholder string, onChange func(string)) *Node {
	if s == nil {
		return nil
	}
	s.opts, s.Value, s.onChange = opts, value, onChange
	label, color := placeholder, ColPlaceholder
	for i, o := range opts {
		if o.Value == value {
			label, color = o.Label, ColText
			if s.Hover < 0 {
				s.Hover = i
			}
			break
		}
	}
	if label == "" {
		label = "请选择"
	}
	stroke := ColStroke
	if s.Open || IsFocused(s.Key) {
		stroke = ColPrimary
	}
	caret := "▼"
	if s.Open {
		caret = "▲"
	}
	box := Row(
		Text(label).St(St().Text(13, 400, color)),
		Spacer(),
		Text(suffixHint(opts, value)).St(St().Text(11, 400, ColTextMuted)),
		Text(caret).St(St().Text(8, 400, ColTextMuted)),
	).St(St().Wp(100).H(FieldH).PadXY(12, 0).GapOf(8).R(10).
		BgC(ColInput).Stroke(1, stroke).FocusOn().Clipped()).AlignTo(AlignCenter)

	n := box.K(s.Key)
	n.OnClick = func() {
		for _, o := range selectStates {
			if o != s {
				o.Close()
			}
		}
		s.Open = !s.Open
		if s.Open {
			SetFocus(s.Key)
			s.Hover = s.index()
			if s.Hover < 0 {
				s.Hover = 0
				s.move(1)
			}
			OpenOverlay(s.Key+"-sel", func() bool {
				s.Close()
				RequestRedraw()
				return true
			})
		} else {
			s.Hover = -1
			CloseOverlay(s.Key + "-sel")
		}
		RequestRedraw()
	}
	n.OnKey = func(k uint32) { selectKey(s, k) }
	return n
}

// suffixHint 当前选项的右侧说明。
func suffixHint(opts []SelectOption, value string) string {
	for _, o := range opts {
		if o.Value == value {
			return o.Hint
		}
	}
	return ""
}

// 下拉浮层几何（绘制与命中一致）。
const (
	selectRowH float32 = 30
	selectMaxH float32 = 240
	selectMinW float32 = 160
	selectPadY float32 = 6
)

// SelectLayers 渲染所有展开的下拉浮层（放在根 Stack 末尾）。
func SelectLayers() *Node {
	layers := make([]*Node, 0, len(selectStates))
	for _, s := range selectStates {
		if n := selectPanel(s); n != nil {
			layers = append(layers, n)
		}
	}
	if len(layers) == 0 {
		return nil
	}
	catcher := Box().St(St().AbsFill()).Tap(func() {
		for _, s := range selectStates {
			s.Close()
		}
		RequestRedraw()
	}).K("select-catcher")
	return Stack(append([]*Node{catcher}, layers...)...).St(St().AbsFill())
}

// selectPanel 单个下拉浮层：锚定控件上一帧矩形，超出视口时上翻。
func selectPanel(s *SelectState) *Node {
	if s == nil || !s.Open {
		return nil
	}
	r, ok := LastFrameOf(s.Key)
	if !ok {
		return nil
	}
	w := r.W
	for _, o := range s.opts {
		if tw := TextWidth(o.Label, 13, 400) + TextWidth(o.Hint, 11, 400) + 56; tw > w {
			w = tw
		}
	}
	if w < selectMinW {
		w = selectMinW
	}
	if w > 320 {
		w = 320
	}
	listH := float32(len(s.opts))*selectRowH + float32(len(s.opts)-1)*2 + selectPadY*2
	if listH < selectRowH+selectPadY*2 {
		listH = selectRowH + selectPadY*2
	}
	x, y := r.X, r.Y+r.H+4
	if x+w > ViewportW()-8 {
		x = ViewportW() - w - 8
	}
	if x < 8 {
		x = 8
	}
	panelH := listH
	if panelH > selectMaxH {
		panelH = selectMaxH
	}
	if y+panelH > ViewportH()-8 {
		if up := r.Y - 4 - panelH; up > 8 {
			y = up
		}
	}
	rows := make([]*Node, 0, len(s.opts))
	for i, o := range s.opts {
		rows = append(rows, selectRowNode(s, i, o, w-selectPadY*2))
	}
	body := Column(rows...).St(St().GapOf(2))
	if listH > selectMaxH {
		body = Column(Scroll(rows...).K(s.Key + "-opt").St(St().H(selectMaxH - selectPadY*2))).
			St(St().W(w - selectPadY*2))
		if s.Open {
			RequestRedraw() // 浮层内含滚动状态，保持帧推进
		}
	}
	return Column(body).St(St().W(w).Padding(selectPadY).R(12).BgC(ColPanel).
		Stroke(1, ColStroke).Shadow(28, 12, 0, "rgba(0,0,0,0.55)").AbsTL(x, y))
}

// selectRowNode 下拉选项行。
func selectRowNode(s *SelectState, i int, o SelectOption, width float32) *Node {
	color := ColTextDim
	if o.Value == s.Value {
		color = "#ffffff"
	}
	if o.Disabled {
		color = ColPlaceholder
	}
	label := o.Label
	if o.Value == s.Value {
		label = "✓ " + label
	}
	cells := []*Node{Text(label).St(St().Text(13, 400, color))}
	if o.Hint != "" {
		cells = append(cells, Spacer(), Text(o.Hint).St(St().Text(11, 400, ColTextMuted)))
	}
	row := Row(cells...).St(St().W(width).H(selectRowH).PadXY(9, 0).GapOf(6).R(7)).AlignTo(AlignCenter)
	if i == s.Hover {
		row.St(St().BgC("rgba(99,102,241,0.18)"))
	} else if IsHovered(fmt.Sprintf("%s-opt-%d", s.Key, i)) {
		row.St(St().BgC(ColHover))
	}
	idx := i
	row.K(fmt.Sprintf("%s-opt-%d", s.Key, i))
	row.OnHoverEvt(func(h bool) {
		if h && s.Hover != idx {
			s.Hover = idx
			RequestRedraw()
		}
	})
	row.Tap(func() { s.pick(idx) })
	return row
}

// scrollListInto 让列表中第 idx 项滚入可视区（key 为 Scroll 的 Key）。
func scrollListInto(key string, idx int, rowH float32) {
	if key == "" || idx < 0 {
		return
	}
	a := currentApp.Load()
	if a == nil || a.root == nil {
		return
	}
	n := findKey(a.root, key)
	if n == nil || n.Frame.H <= 0 {
		return
	}
	top := float32(idx) * (rowH + 2)
	h := n.Frame.H
	cur := a.scrolls[key]
	if top < cur {
		a.setScrollTarget(key, top, n.ContentH-h)
	} else if top+rowH > cur+h {
		a.setScrollTarget(key, top+rowH-h, n.ContentH-h)
	}
}

// ---- 多行输入 ----

// textareaCaret 每个多行输入的插入光标位置（按 rune）。
var textareaCaret = map[string]int{}

// Textarea 多行文本输入：换行 / 上下左右 / Home-End / 退格删除 / 光标闪烁，
// 内容超过可视高度时自动跟随光标滚动（不折行，超出部分裁剪）。
func Textarea(key, value, placeholder string, height float32, onChange func(string)) *Node {
	if height <= 0 {
		height = 96
	}
	const (
		lineH = float32(20)
		padX  = float32(12)
		padY  = float32(8)
	)
	focused := IsFocused(key)
	runes := []rune(value)
	caret, ok := textareaCaret[key]
	if !ok || caret > len(runes) || caret < 0 {
		caret = len(runes)
	}
	textareaCaret[key] = caret

	// 光标所在行 / 列（列按 rune 计）
	line, col := 0, 0
	for i := 0; i < caret && i < len(runes); i++ {
		if runes[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	lines := strings.Split(value, "\n")
	shown := int((height - padY*2) / lineH)
	if shown < 1 {
		shown = 1
	}
	top := 0
	if line >= shown {
		top = line - shown + 1
	}
	if top > len(lines)-shown {
		top = len(lines) - shown
	}
	if top < 0 {
		top = 0
	}

	showCaret := focused && int(AnimTime()*2)%2 == 0
	cells := make([]*Node, 0, shown)
	if value == "" && !focused {
		cells = append(cells, Row(Text(placeholder).St(St().Text(13, 400, ColPlaceholder))).St(St().H(lineH)))
	}
	for i := top; i < len(lines) && i < top+shown; i++ {
		ln := []rune(lines[i])
		var row *Node
		if showCaret && i == line {
			c := col
			if c > len(ln) {
				c = len(ln)
			}
			row = Row(
				Text(string(ln[:c])).St(St().Text(13, 400, ColText)),
				Box().St(St().W(1.5).H(16).BgC(ColPrimary).MarXY(0.5, 0)),
				Text(string(ln[c:])).St(St().Text(13, 400, ColText)),
			).St(St().H(lineH).AnimSlowOn()).AlignTo(AlignCenter)
		} else {
			txt := string(ln)
			if txt == "" {
				txt = " "
			}
			row = Row(Text(txt).St(St().Text(13, 400, ColText).CenterYOn())).St(St().H(lineH))
		}
		cells = append(cells, row)
	}

	stroke := ColStroke
	if focused {
		stroke = ColPrimary
	}
	box := Stack(
		Column(cells...).St(St().AbsTL(padX, padY)),
	).St(St().Wp(100).H(height).R(10).BgC(ColInput).Stroke(1, stroke).FocusOn().Clipped())

	n := box.K(key)
	n.OnClick = func() {
		SetFocus(key)
		RequestRedraw()
	}
	// set 统一回写状态与光标
	set := func(rs []rune, c int) {
		if c < 0 {
			c = 0
		}
		if c > len(rs) {
			c = len(rs)
		}
		textareaCaret[key] = c
		if onChange != nil {
			onChange(string(rs))
		}
		RequestRedraw()
	}
	n.OnChar = func(r rune) {
		switch r {
		case '\r', '\n':
			return // 换行由 OnKey 处理，避免重复插入
		case '\t':
			r = ' '
		case 8: // 退格
			if caret > 0 {
				rs := append(runes[:caret-1:caret-1], runes[caret:]...)
				set(rs, caret-1)
			}
			return
		case 127: // Delete
			if caret < len(runes) {
				rs := append(runes[:caret:caret], runes[caret+1:]...)
				set(rs, caret)
			}
			return
		}
		rs := make([]rune, 0, len(runes)+1)
		rs = append(rs, runes[:caret]...)
		rs = append(rs, r)
		rs = append(rs, runes[caret:]...)
		set(rs, caret+1)
	}
	n.OnKey = func(k uint32) {
		switch k {
		case 0x0D: // Enter 换行
			rs := make([]rune, 0, len(runes)+1)
			rs = append(rs, runes[:caret]...)
			rs = append(rs, '\n')
			rs = append(rs, runes[caret:]...)
			set(rs, caret+1)
		case 0x1B: // Esc 失焦
			SetFocus("")
			RequestRedraw()
		case 0x25: // ←
			set(runes, caret-1)
		case 0x27: // →
			set(runes, caret+1)
		case 0x26: // ↑ 上一行同列
			set(runes, taVertical(runes, caret, -1))
		case 0x28: // ↓ 下一行同列
			set(runes, taVertical(runes, caret, 1))
		case 0x24: // Home 行首
			set(runes, taLineStart(runes, caret))
		case 0x23: // End 行尾
			set(runes, taLineEnd(runes, caret))
		case 0x2E: // Delete
			if caret < len(runes) {
				rs := append(runes[:caret:caret], runes[caret+1:]...)
				set(rs, caret)
			}
		}
	}
	return n
}

// taLineStart 光标所在行首的 rune 索引。
func taLineStart(rs []rune, caret int) int {
	for caret > 0 && rs[caret-1] != '\n' {
		caret--
	}
	return caret
}

// taLineEnd 光标所在行尾（换行符之前）的 rune 索引。
func taLineEnd(rs []rune, caret int) int {
	for caret < len(rs) && rs[caret] != '\n' {
		caret++
	}
	return caret
}

// taVertical 上下移动一行，尽量保持列位置。
func taVertical(rs []rune, caret, dir int) int {
	start := taLineStart(rs, caret)
	col := caret - start
	if dir < 0 {
		if start == 0 {
			return caret
		}
		pEnd := start - 1 // 上一行换行符位置
		pStart := taLineStart(rs, pEnd)
		if pStart+col > pEnd {
			return pEnd
		}
		return pStart + col
	}
	end := taLineEnd(rs, caret)
	if end >= len(rs) {
		return caret
	}
	nStart := end + 1
	nEnd := taLineEnd(rs, nStart)
	if nStart+col > nEnd {
		return nEnd
	}
	return nStart + col
}

// ---- 数字 / 搜索 / 标签 / 日期 ----

// NumberInput 数字输入：左右 +/- 步进按钮 + 中间可编辑输入框。
func NumberInput(key string, v float64, min, max, step float64, onChange func(float64)) *Node {
	if step == 0 {
		step = 1
	}
	clamp := func(x float64) float64 {
		if x < min {
			x = min
		}
		if x > max {
			x = max
		}
		return x
	}
	dec := 0
	if step < 1 && step > 0 {
		dec = 2
	}
	fmtV := func() string { return strconv.FormatFloat(v, 'f', dec, 64) }
	bump := func(d float64) {
		nv := clamp(mathRound(v+d, dec))
		if onChange != nil {
			onChange(nv)
		}
		RequestRedraw()
	}
	minus := Text("−").St(St().Text(14, 600, ColTextDim).W(30).H(FieldH).CenterYOn().TAlign(AlignCenter)).
		K(key + "-minus").Tap(func() { bump(-step) })
	plus := Text("+").St(St().Text(14, 600, ColTextDim).W(30).H(FieldH).CenterYOn().TAlign(AlignCenter)).
		K(key + "-plus").Tap(func() { bump(step) })
	mid := Box(Input(key, fmtV(), "", func(s string) {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return
		}
		if onChange != nil {
			onChange(clamp(f))
		}
	})).Grow(1)
	row := Row(minus, mid, plus).St(St().GapOf(2).Padding(3).R(10).
		BgC(ColInput).Stroke(1, ColStroke)).AlignTo(AlignCenter)
	return row.K(key + "-num")
}

// mathRound 四舍五入到 dec 位小数（避免浮点尾巴）。
func mathRound(v float64, dec int) float64 {
	p := 1.0
	for i := 0; i < dec; i++ {
		p *= 10
	}
	if v >= 0 {
		return float64(int64(v*p+0.5)) / p
	}
	return float64(int64(v*p-0.5)) / p
}

// SearchInput 搜索框：左侧放大镜（矢量方块拼）+ 有内容时右侧清除按钮。
func SearchInput(key, value, placeholder string, onChange func(string)) *Node {
	if placeholder == "" {
		placeholder = "搜索…"
	}
	field := Input(key, value, placeholder, onChange)
	field.St(St().PadOnly(0, 30, 0, 0))
	glass := Stack(
		Box().St(St().W(9).H(9).R(5).Stroke(1.6, ColTextMuted).AbsTL(0, 0)),
		Box().St(St().W(5).H(1.6).R(1).BgC(ColTextMuted).AbsTL(8, 9)),
	).St(St().W(14).H(12))
	cells := []*Node{Box(glass).St(St().AbsTL(12, 13)), field}
	if value != "" {
		cells = append(cells, Text("×").St(St().Text(14, 500, ColTextMuted).W(22).H(FieldH).
			CenterYOn().TAlign(AlignCenter)).K(key+"-clear").Tap(func() {
			inputClear(key)
			if onChange != nil {
				onChange("")
			}
			RequestRedraw()
		}).St(St().AbsTR(8, 0)))
	}
	return Stack(cells...).St(St().Wp(100).H(FieldH))
}

// inputClear 清空输入框文本与光标（供清空按钮用）。
func inputClear(key string) {
	inputCaret[key] = 0
}

// Chip 可选中的标签胶囊。
func Chip(label string, active bool, onToggle func()) *Node {
	bg, stroke, color := "rgba(255,255,255,0.05)", "rgba(255,255,255,0.10)", ColTextDim
	if active {
		bg, stroke, color = "rgba(99,102,241,0.20)", "rgba(129,140,248,0.55)", "#e0e7ff"
	}
	return Text(label).St(St().Text(11.5, 600, color).PadXY(11, 6).R(9).
		BgC(bg).Stroke(1, stroke)).K("chip-" + label).Tap(func() {
		if onToggle != nil {
			onToggle()
		}
		RequestRedraw()
	})
}

// ChipRemovable 带删除按钮的标签（用于多选结果）。
func ChipRemovable(label string, onRemove func()) *Node {
	return Row(
		Text(label).St(St().Text(11.5, 600, ColTextDim)),
		Text("×").St(St().Text(13, 500, ColTextMuted)).Tap(func() {
			if onRemove != nil {
				onRemove()
			}
			RequestRedraw()
		}),
	).St(St().GapOf(6).PadXY(10, 5).R(9).BgC("rgba(99,102,241,0.16)").
		Stroke(1, "rgba(129,140,248,0.4)")).AlignTo(AlignCenter)
}

// DateField 只读日期展示框（带日历图标与小锁提示）。
func DateField(label, value string) *Node {
	cell := func(c string) *Node {
		return Text(c).St(St().Text(13, 500, ColTextDim))
	}
	return Row(
		cell(value),
		Spacer(),
		Text(label).St(St().Text(11, 400, ColTextMuted)),
	).St(St().Wp(100).H(FieldH).PadXY(12, 0).GapOf(8).R(10).
		BgC(ColBgDeep).Stroke(1, ColStroke)).AlignTo(AlignCenter)
}

// DateState 日历浮层状态。
type DateState struct {
	Key   string
	Open  bool
	Y, M  int // 面板当前显示的年月
	Value string

	onChange func(string)
}

// dateStates 已创建的日历状态（DateLayers 统一渲染）。
var dateStates = map[string]*DateState{}

// NewDate 创建日历状态。
func NewDate(key string) *DateState {
	s := &DateState{Key: key}
	dateStates[key] = s
	return s
}

// DateBox 日期选择控件（点击展开日历浮层，选择后回调 YYYY-MM-DD）。
func DateBox(s *DateState, value string, onChange func(string)) *Node {
	if s == nil {
		return nil
	}
	s.Value, s.onChange = value, onChange
	color := ColText
	if value == "" {
		value, color = "请选择日期", "#4d5670"
	}
	stroke := ColStroke
	if s.Open || IsFocused(s.Key) {
		stroke = ColPrimary
	}
	n := Row(
		Text(value).St(St().Text(13, 400, color)),
		Spacer(),
		Text("▦").St(St().Text(11, 400, ColTextMuted)),
	).St(St().Wp(100).H(FieldH).PadXY(12, 0).GapOf(8).R(10).
		BgC(ColInput).Stroke(1, stroke).FocusOn().Clipped()).
		AlignTo(AlignCenter).K(s.Key)
	n.OnClick = func() {
		s.Open = !s.Open
		if s.Open {
			SetFocus(s.Key)
			s.Y, s.M = dateParse(value)
			for _, o := range dateStates {
				if o != s {
					o.Open = false
				}
			}
			OpenOverlay(s.Key+"-date", func() bool {
				s.Open = false
				RequestRedraw()
				return true
			})
		} else {
			CloseOverlay(s.Key + "-date")
		}
		RequestRedraw()
	}
	n.OnKey = func(k uint32) {
		switch k {
		case 0x0D:
			s.Open = !s.Open
		case 0x1B:
			s.Open = false
		}
		RequestRedraw()
	}
	return n
}

// dateParse 解析 YYYY-MM-DD，失败时返回当前年月。
func dateParse(v string) (int, int) {
	parts := strings.Split(strings.TrimSpace(v), "-")
	if len(parts) >= 2 {
		y, e1 := strconv.Atoi(parts[0])
		m, e2 := strconv.Atoi(parts[1])
		if e1 == nil && e2 == nil && m >= 1 && m <= 12 {
			return y, m
		}
	}
	now := time.Now()
	return now.Year(), int(now.Month())
}

// DateLayers 渲染所有展开的日历浮层（放在根 Stack 末尾）。
func DateLayers() *Node {
	layers := make([]*Node, 0, len(dateStates))
	for _, s := range dateStates {
		if n := datePanel(s); n != nil {
			layers = append(layers, n)
		}
	}
	if len(layers) == 0 {
		return nil
	}
	catcher := Box().St(St().AbsFill()).Tap(func() {
		for _, s := range dateStates {
			s.Open = false
		}
		RequestRedraw()
	}).K("date-catcher")
	return Stack(append([]*Node{catcher}, layers...)...).St(St().AbsFill())
}

// datePanel 单个月历面板（6 行 × 7 列）。
func datePanel(s *DateState) *Node {
	if s == nil || !s.Open {
		return nil
	}
	r, ok := LastFrameOf(s.Key)
	if !ok {
		return nil
	}
	const (
		cellW = float32(34)
		cellH = float32(28)
		padX  = float32(10)
		padY  = float32(8)
		weeks = 6
	)
	w := cellW*7 + padX*2
	y0, m0 := s.Y, s.M
	first := time.Date(y0, time.Month(m0), 1, 0, 0, 0, 0, time.Local)
	lead := int(first.Weekday()) // 周日=0
	prevY, prevM := m0-1, y0
	if prevM < 1 {
		prevM, prevY = 12, y0-1
	}
	nextY, nextM := m0+1, y0
	if nextM > 12 {
		nextM, nextY = 1, y0+1
	}
	daysIn := func(y, m int) int { return time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.Local).Day() }
	prevDays := daysIn(prevY, prevM)
	curDays := daysIn(y0, m0)

	cells := make([]*Node, 0, weeks*7+1)
	// 星期表头
	head := Row()
	for _, d := range []string{"日", "一", "二", "三", "四", "五", "六"} {
		head.Add(Text(d).St(St().Text(10.5, 500, ColTextMuted).W(cellW).CenterYOn().TAlign(AlignCenter)))
	}
	cells = append(cells, head.St(St().H(20)))

	today := time.Now()
	grid := Row()
	for i := 0; i < weeks*7; i++ {
		day := i - lead + 1
		dy, dm, dd := y0, m0, day
		muted := false
		if day < 1 {
			dy, dm, dd = prevY, prevM, prevDays+day
			muted = true
		} else if day > curDays {
			dy, dm, dd = nextY, nextM, day-curDays
			muted = true
		}
		label := strconv.Itoa(dd)
		color := ColTextDim
		bg := "transparent"
		stroke := "transparent"
		if muted {
			color = ColPlaceholder
		}
		if dy == today.Year() && int(today.Month()) == dm && today.Day() == dd {
			color = "#ffffff"
			bg = "rgba(99,102,241,0.28)"
		}
		if fmt.Sprintf("%04d-%02d-%02d", dy, dm, dd) == s.Value {
			color = "#ffffff"
			bg = ColPrimary
			stroke = "rgba(255,255,255,0.35)"
		}
		cy, cm, cd := dy, dm, dd
		cell := Text(label).St(St().Text(11.5, 500, color).W(cellW).H(cellH).R(7).
			CenterYOn().TAlign(AlignCenter).BgC(bg).Stroke(1, stroke))
		cell.K(fmt.Sprintf("%s-day-%04d%02d%02d", s.Key, cy, cm, cd)).Tap(func() {
			s.Value = fmt.Sprintf("%04d-%02d-%02d", cy, cm, cd)
			s.Open = false
			CloseOverlay(s.Key + "-date")
			if s.onChange != nil {
				s.onChange(s.Value)
			}
			RequestRedraw()
		})
		grid.Add(cell)
		if (i+1)%7 == 0 && i < weeks*7-1 {
			cells = append(cells, grid.St(St().GapOf(0)))
			grid = Row()
		}
	}
	cells = append(cells, grid.St(St().GapOf(0)))

	navBtn := func(label string, dy, dm int) *Node {
		return Text(label).St(St().Text(13, 600, ColTextDim).W(26).H(22).R(6).
			CenterYOn().TAlign(AlignCenter)).Tap(func() {
			s.Y, s.M = dy, dm
			RequestRedraw()
		})
	}
	header := Row(
		navBtn("‹", prevY, prevM),
		Text(fmt.Sprintf("%d 年 %d 月", y0, m0)).St(St().Text(12, 600, ColText).TAlign(AlignCenter)).Grow(1),
		navBtn("›", nextY, nextM),
	).St(St().GapOf(4)).AlignTo(AlignCenter)

	panel := Column(
		header,
		Column(cells...).St(St().GapOf(2)),
	).St(St().W(w).Padding(padX).GapOf(6).R(12).BgC(ColPanel).
		Stroke(1, ColStroke).Shadow(28, 12, 0, "rgba(0,0,0,0.55)"))

	x, py := r.X, r.Y+r.H+4
	if x+w > ViewportW()-8 {
		x = ViewportW() - w - 8
	}
	if x < 8 {
		x = 8
	}
	ph := 300
	if py+float32(ph) > ViewportH()-8 {
		if up := r.Y - 4 - float32(ph); up > 8 {
			py = up
		}
	}
	return panel.St(St().AbsTL(x, py))
}
