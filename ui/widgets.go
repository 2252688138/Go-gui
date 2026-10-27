package ui

import (
	"math"
	"strconv"
	"strings"

	"gogui/internal/gpu"
)

// 组件库：在基础节点（Box/Row/Column/Stack/Text/Button）之上组合出的控件。
// 约定：所有可选/可编辑状态由调用方持有，回调内修改状态并调用 ui.RequestRedraw()。
// 交互能力对应 gio 的 widget：Clickable（Tap/TapAt/DragEvt）、Editor（Input）、
// List（Scroll 自带滚动条）、Switch/CheckBox/RadioButton、Slider、Table、Tooltip 等。

// 主题 token：定义在 theme.go（深/浅色两套色板 + SetTheme 切换）。
// 组件里直接用 ColText / ColBg / ColInput / ColPanel 等变量，不要写死颜色，
// 否则切浅色主题时会出现“白底深卡”。

// ---- 基础排版 ----

// Divider 分隔线。
func Divider() *Node {
	return Box().St(St().Wp(100).H(1).BgC(ColLine))
}

// Title 区块标题（带副标题可选）。
func Title(text, sub string) *Node {
	col := Column(Text(text).St(St().Text(16, 600, ColText)))
	if sub != "" {
		col.Add(Text(sub).St(St().Text(12, 400, ColTextMuted).PadOnly(4, 0, 0, 0)))
	}
	return col
}

// Card 卡片容器。
func Card(children ...*Node) *Node {
	return Column(children...).St(St().Padding(20).R(16).GapOf(12).
		BgC(ColBg).Stroke(1, ColStroke).Shadow(20, 8, 0, "rgba(0,0,0,0.32)"))
}

// Tag 标签 / 徽章文本。
func Tag(label, color string) *Node {
	return Text(label).St(St().Text(11, 600, color).PadXY(9, 4).R(7).
		BgC(ColHover).Stroke(1, ColLine))
}

// Dot 状态圆点。
func Dot(color string, size float32) *Node {
	return Box().St(St().W(size).H(size).R(size / 2).BgC(color))
}

// Avatar 圆形头像（文字首字）。
func Avatar(initial, color string) *Node {
	return Stack(
		Box().St(St().AbsFill().R(20).BgGrad(color, "rgba(255,255,255,0.06)", 120)),
		Text(initial).St(St().AbsFill().Text(14, 600, "#ffffff").CenterYOn().TAlign(AlignCenter)),
	).St(St().W(40).H(40))
}

// Badge 角标数字（附着在父节点右上角，需父节点为 Stack）。
func Badge(count int) *Node {
	label := strconv.Itoa(count)
	if count > 99 {
		label = "99+"
	}
	return Text(label).St(St().Text(10, 700, "#ffffff").PadXY(6, 2).R(9).
		BgC(ColDanger).AbsTR(-6, -6))
}

// Link 文字链接。
func Link(label string, onClick func()) *Node {
	return Text(label).St(St().Text(12.5, 500, "#8b93ff")).Tap(func() {
		if onClick != nil {
			onClick()
		}
		RequestRedraw()
	})
}

// KeyValue 左标题右值的一行。
func KeyValue(k, v string) *Node {
	return Row(
		Text(k).St(St().Text(12, 400, ColTextMuted)),
		Spacer(),
		Text(v).St(St().Text(12.5, 500, ColTextDim)),
	).St(St().PadOnly(0, 0, 6, 0)).AlignTo(AlignCenter)
}

// Empty 空状态占位。
func Empty(title, sub string) *Node {
	return Column(
		Stack(
			Box().St(St().W(54).H(54).R(16).BgC(ColHover).Stroke(1, ColStroke)),
			Text("—").St(St().AbsFill().Text(20, 500, ColPlaceholder).CenterYOn().TAlign(AlignCenter)),
		).St(St().W(54).H(54)),
		Text(title).St(St().Text(13.5, 600, ColText).PadOnly(14, 0, 0, 0)),
		Text(sub).St(St().Text(11.5, 400, ColTextMuted).WrapOn().TAlign(AlignCenter)),
	).St(St().GapOf(6).Padding(28)).AlignTo(AlignCenter)
}

// ---- 按钮 ----

// ButtonKind 按钮变体。
type ButtonKind int

const (
	BtnPrimary   ButtonKind = iota // 主行动：渐变实心
	BtnSecondary                   // 次要：深色描边
	BtnGhost                       // 幽灵：无背景
	BtnDanger
	BtnSuccess
)

// Btn 带变体与禁用态的按钮。
func Btn(label string, kind ButtonKind, onClick func()) *Node {
	st := St().Text(13, 600, "#ffffff").PadXY(16, 8).R(9)
	switch kind {
	case BtnPrimary:
		st.BgGrad(ColPrimary, ColPrimary2, 90)
	case BtnSecondary:
		st.Text(13, 500, ColText).BgC(ColBgSoft).Stroke(1, ColStroke)
	case BtnGhost:
		st.Text(13, 500, ColTextMuted)
	case BtnDanger:
		st.BgGrad("#ef4444", "#f87171", 90)
	case BtnSuccess:
		st.BgGrad("#10b981", "#34d399", 90)
	}
	return Button(label, onClick).St(st)
}

// IconButton 方形图标按钮（glyph 建议用 + - × → ← ↑ ↓ 等有字形符号）。
func IconButton(glyph string, onClick func()) *Node {
	return Button(glyph, onClick).St(St().W(34).H(34).MinWidth(34).
		Text(14, 600, ColTextDim).R(9).BgC(ColBgSoft).Stroke(1, ColStroke))
}

// ---- 表单控件 ----

// Checkbox 复选框。
func Checkbox(key, label string, checked bool, onChange func(bool)) *Node {
	box := ColHover
	stroke := ColStroke
	mark := ""
	if checked {
		box = ColPrimary
		stroke = "#8b93ff"
		mark = "√"
	}
	indicator := Stack(
		Box().St(St().AbsFill().R(6).BgC(box).Stroke(1.5, stroke)),
		Text(mark).St(St().AbsFill().Text(12, 700, "#ffffff").CenterYOn().TAlign(AlignCenter)),
	).St(St().W(18).H(18))
	row := Row(indicator, Text(label).St(St().Text(13, 500, ColTextDim))).
		St(St().GapOf(10)).AlignTo(AlignCenter)
	return row.K(key).Tap(func() {
		checked = !checked
		if onChange != nil {
			onChange(checked)
		}
		RequestRedraw()
	})
}

// Radio 单选按钮。
func Radio(key, label string, selected bool, onPick func()) *Node {
	outer := ColLine
	inner := "transparent"
	if selected {
		outer = ColPrimary
		inner = ColPrimary
	}
	ind := Stack(
		Box().St(St().AbsFill().R(9).Stroke(1.5, outer)),
		Box().St(St().W(8).H(8).R(4).BgC(inner).AbsCenter()),
	).St(St().W(18).H(18))
	row := Row(ind, Text(label).St(St().Text(13, 500, ColTextDim))).
		St(St().GapOf(10)).AlignTo(AlignCenter)
	return row.K(key).Tap(func() {
		if onPick != nil {
			onPick()
		}
		RequestRedraw()
	})
}

// switchAnim 记录开关切换时刻，用于旋钮过渡动画。
var switchAnim = map[string]float32{}

// Switch 开关（轨道 + 旋钮，切换时 140ms 过渡）。
func Switch(key string, on bool, onChange func(bool)) *Node {
	const trackW, trackH, knob float32 = 44, 24, 18
	t := float32(1)
	if start, ok := switchAnim[key]; ok {
		t = (AnimTime() - start) / 0.14
		if t >= 1 {
			t = 1
		}
	}
	if t < 1 {
		RequestRedraw()
	}
	target := float32(0)
	if on {
		target = 1
	}
	// 由目标值倒推动画起点：t=0 时位于旧位置
	from := target
	if t < 1 {
		from = 1 - target
	}
	k := from + (target-from)*easeOut(t)
	offX := 3 + k*(trackW-knob-6)
	bg := Blend(C(ColTrack), C(ColPrimary), k)
	knobC := Blend(C("#8b93ad"), C("#ffffff"), k)

	n := Stack(
		Box().St(St().AbsFill().R(trackH/2).BgC(bg)),
		Box().St(St().W(knob).H(knob).R(knob/2).BgC(knobC).
			Shadow(8, 2, 0, "rgba(0,0,0,0.45)").AbsTL(offX, (trackH-knob)/2)),
	).St(St().W(trackW).H(trackH))
	if t < 1 {
		n.St(St().AnimOn())
	}
	return n.K(key).Tap(func() {
		switchAnim[key] = AnimTime()
		on = !on
		if onChange != nil {
			onChange(on)
		}
		RequestRedraw()
	})
}

func easeOut(t float32) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	u := 1 - t
	return 1 - u*u*u
}

// Slider 滑块（0~1，支持点击定位与拖动，拖动过程中实时跟手）。
func Slider(key string, v float32, onChange func(float32)) *Node {
	const trackW, trackH, thumb float32 = 200, 18, 14
	usable := trackW - thumb
	// 轨道与滑块都按容器高度垂直居中（轨道 4px 中心、滑块 14px 中心同在 trackH/2）
	trackY := (trackH - 4) / 2
	thumbY := (trackH - thumb) / 2
	set := func(x float32) {
		nv := (x - thumb/2) / usable
		if nv < 0 {
			nv = 0
		}
		if nv > 1 {
			nv = 1
		}
		if onChange != nil {
			onChange(nv)
		}
		RequestRedraw()
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	n := Stack(
		Box().St(St().W(trackW).H(4).R(2).BgC(ColTrack).AbsTL(0, trackY)),
		Box().St(St().W(v*usable+thumb/2).H(4).R(2).BgGrad(ColPrimary, ColPrimary2, 0).AbsTL(0, trackY)),
		Box().St(St().W(thumb).H(thumb).R(thumb/2).BgC("#ffffff").
			Stroke(3, ColPrimary).Shadow(10, 2, 0, "rgba(99,102,241,0.55)").
			AbsTL(v*usable, thumbY)),
	).St(St().W(trackW).H(trackH))
	return n.K(key).TapAt(func(x, _ float32) { set(x) }).DragEvt(func(x, _ float32) { set(x) })
}

// Progress 进度条（0~1）。宽度由父容器决定（Column 拉伸 / Row 内 Grow），
// 不写死 100% 宽度，避免在不定宽容器里按父级满宽测量而溢出。
func Progress(v float32, color string) *Node {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return Stack(
		Box().St(St().AbsFill().R(4).BgC(ColTrack)),
		Box().St(St().Wp(v*100).Hp(100).R(4).BgGrad(color, "#ffffff33", 90)),
	).St(St().H(8))
}

// Ring 环形进度（分段圆点近似，gio 风格的小尺寸指示器）。
func Ring(v float32, size float32, color string) *Node {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	const segments = 40
	rad := float64(size/2 - 3)
	items := make([]*Node, 0, segments)
	for i := 0; i < segments; i++ {
		ang := -math.Pi/2 + float64(i)*2*math.Pi/segments
		x := float32(float64(size)/2+rad*math.Cos(ang)) - 2
		y := float32(float64(size)/2+rad*math.Sin(ang)) - 2
		c := ColHover
		if float32(i) < v*segments {
			c = color
		}
		items = append(items, Box().St(St().W(4).H(4).R(2).BgC(c).AbsTL(x, y)))
	}
	return Stack(items...).St(St().W(size).H(size))
}

// ---- 输入框（gio Editor 的简化版） ----

// inputCaret 记录每个输入框的光标位置（按 rune）。
var inputCaret = map[string]int{}

// inputSubmit 记录每个输入框的回车提交回调（地址栏等需要「回车即执行」的场景）。
var inputSubmit = map[string]func(){}

// InputSubmit 带回车提交的单行输入框：回车触发 onSubmit（地址栏、搜索框用）。
func InputSubmit(key, value, placeholder string, onChange func(string), onSubmit func()) *Node {
	if onSubmit != nil {
		inputSubmit[key] = onSubmit
	}
	return inputNode(key, value, placeholder, false, onChange)
}

// Input 单行输入框：点击获焦、光标闪烁、左右/Home/End/退格/删除。
func Input(key, value, placeholder string, onChange func(string)) *Node {
	return inputNode(key, value, placeholder, false, onChange)
}

// PasswordInput 密码输入框（掩码显示）。
func PasswordInput(key, value, placeholder string, onChange func(string)) *Node {
	return inputNode(key, value, placeholder, true, onChange)
}

func inputNode(key, value, placeholder string, password bool, onChange func(string)) *Node {
	focused := IsFocused(key)
	runes := []rune(value)
	caret, ok := inputCaret[key]
	if !ok || caret > len(runes) {
		caret = len(runes)
	}
	if caret < 0 {
		caret = 0
	}
	inputCaret[key] = caret

	// 显示文本：密码掩码
	display := value
	if password && len(runes) > 0 {
		display = strings.Repeat("•", len(runes))
	}

	stroke := ColStroke
	if focused {
		stroke = ColPrimary
	}

	var content *Node
	if focused {
		// 光标闪烁：1.0s 周期，动画节点驱动逐帧重绘
		showCaret := int(AnimTime()*2)%2 == 0
		before := string(runes[:caret])
		after := string(runes[caret:])
		if password {
			before = strings.Repeat("•", caret)
			after = strings.Repeat("•", len(runes)-caret)
		}
		cells := []*Node{Text(before).St(St().Text(13, 400, ColText))}
		// IME 组合串：高亮背景显示，光标不闪烁（输入法正在编辑）
		if comp := ComposingText(); comp != "" && !password {
			cp := ComposingCaret()
			cr := []rune(comp)
			if cp < 0 || cp > len(cr) {
				cp = len(cr)
			}
			cells = append(cells, Text(string(cr[:cp])).St(St().Text(13, 400, ColText).BgC(ColBgSoft).R(3)))
			cells = append(cells, Text(string(cr[cp:])).St(St().Text(13, 400, ColText).BgC(ColHover).R(3)))
			showCaret = false
		}
		if showCaret {
			cells = append(cells, Box().St(St().W(1.5).H(16).BgC(ColPrimary).MarXY(0.5, 0)))
		}
		cells = append(cells, Text(after).St(St().Text(13, 400, ColText)))
		content = Row(cells...).St(St().AnimOn())
	} else if value == "" {
		content = Row(Text(placeholder).St(St().Text(13, 400, ColPlaceholder)))
	} else {
		content = Row(Text(display).St(St().Text(13, 400, ColText)))
	}

	box := Stack(
		Row(content).St(St().AbsFill().PadXY(12, 0)).AlignTo(AlignCenter),
	).St(St().Wp(100).H(38).R(10).BgC(ColInput).Stroke(1, stroke).FocusOn().Clipped())

	n := box.K(key)
	n.OnClick = func() { SetFocus(key) }
	n.OnChar = func(r rune) {
		switch r {
		case '\r', '\n', '\t':
			return
		case 8, 127: // Backspace / Delete
			if caret > 0 {
				runes = append(runes[:caret-1], runes[caret:]...)
				caret--
			}
		default:
			runes = append(runes, 0)
			copy(runes[caret+1:], runes[caret:])
			runes[caret] = r
			caret++
		}
		inputCaret[key] = caret
		value = string(runes)
		if onChange != nil {
			onChange(value)
		}
		RequestRedraw()
	}
	n.OnKey = func(k uint32) {
		switch k {
		case 0x0D: // Enter：提交（不录入文本，由回调决定动作）
			if fn := inputSubmit[key]; fn != nil {
				fn()
			}
			return
		case 0x25: // Left
			if caret > 0 {
				caret--
			}
		case 0x27: // Right
			if caret < len(runes) {
				caret++
			}
		case 0x24: // Home
			caret = 0
		case 0x23: // End
			caret = len(runes)
		case 0x2E: // Delete
			if caret < len(runes) {
				runes = append(runes[:caret], runes[caret+1:]...)
			}
		default:
			return
		}
		inputCaret[key] = caret
		value = string(runes)
		if onChange != nil {
			onChange(value)
		}
		RequestRedraw()
	}
	return n
}

// ---- 加载与反馈 ----

// Spinner 旋转加载指示器（逐帧动画）。
func Spinner(size float32, color string) *Node {
	const segments = 8
	center := size / 2
	rad := size/2 - 2
	items := make([]*Node, 0, segments)
	t := AnimTime()
	head := int(t*8) % segments
	for i := 0; i < segments; i++ {
		ang := float64(i) * math.Pi / 4
		x := float32(float64(center)+float64(rad)*math.Cos(ang)) - 2
		y := float32(float64(center)+float64(rad)*math.Sin(ang)) - 2
		alpha := 0.15 + 0.85*float64((i-head+segments)%segments)/float64(segments)
		c := C(color).WithAlpha(float32(alpha))
		items = append(items, Box().St(St().W(4).H(4).R(2).BgC(c).AbsTL(x, y)))
	}
	return Stack(items...).St(St().W(size).H(size))
}

// Skeleton 骨架屏占位（呼吸高亮动画）。
func Skeleton(w, h float32) *Node {
	t := AnimTime()
	k := 0.05 + 0.05*(1+float32(math.Sin(float64(t)*2.2)))/2
	return Box().St(St().W(w).H(h).R(6).BgC(whiteA(k)))
}

// Toast 轻提示（悬浮于容器底部中央，需放在 Stack 中）。
func Toast(msg string) *Node {
	if msg == "" {
		return nil
	}
	return Row(
		Box().St(St().W(6).H(6).R(3).BgC(ColSuccess)),
		Text(msg).St(St().Text(12.5, 500, ColText)),
	).St(St().AbsBottom(24).PadXY(14, 10).R(10).GapOf(8).
		BgC(ColPanel).Stroke(1, ColStroke).Shadow(24, 8, 0, "rgba(0,0,0,0.5)"))
}

// Banner 信息条（左侧色条 + 标题/正文 + 关闭）。
func Banner(kind, title, body string, onClose func()) *Node {
	color := ColInfo
	switch kind {
	case "success":
		color = ColSuccess
	case "warn":
		color = ColWarn
	case "error":
		color = ColDanger
	}
	row := Row(
		Box().St(St().W(3).Hp(100).R(2).BgC(color)),
		Column(
			Text(title).St(St().Text(12.5, 600, ColText)),
			Text(body).St(St().Text(11.5, 400, ColTextMuted).WrapOn().PadOnly(3, 0, 0, 0)),
		).Grow(1).St(St().GapOf(2)),
	)
	if onClose != nil {
		row.Add(Text("×").St(St().Text(15, 500, ColTextMuted)).Tap(onClose))
	}
	return row.St(St().GapOf(12).Padding(12).R(10).
		BgC(ColHover).Stroke(1, ColStroke)).AlignTo(AlignCenter)
}

// Modal 模态框（遮罩 + 卡片，放在 Stack 中并已 AbsFill）。
func Modal(open bool, title, body string, onClose func()) *Node {
	return ModalEx(open, title, body, "取消", "确定", onClose, onClose, onClose, 420)
}

// ModalEx 可定制按钮与宽度的模态框。
func ModalEx(open bool, title, body, cancelLabel, okLabel string, onCancel, onConfirm, onClose func(), width float32) *Node {
	if !open {
		return nil
	}
	if width <= 0 {
		width = 420
	}
	scrim := Box().St(St().AbsFill().BgC(ColScrim)).Tap(onClose)
	dialog := Column(
		Row(
			Text(title).St(St().Text(16, 600, ColText)).Grow(1),
			Text("×").St(St().Text(15, 500, ColTextMuted)).Tap(onClose),
		).AlignTo(AlignCenter),
		Divider(),
		Text(body).St(St().Text(13, 400, ColTextDim).WrapOn()),
		Row(
			Spacer(),
			Btn(cancelLabel, BtnSecondary, onCancel),
			Btn(okLabel, BtnPrimary, onConfirm),
		).St(St().GapOf(10)).AlignTo(AlignCenter),
	).St(St().W(width).Padding(22).R(18).GapOf(14).
		BgC(ColPanelDeep).Stroke(1, ColStroke).Shadow(48, 18, 0, "rgba(0,0,0,0.55)").
		AbsCenter())

	return Stack(scrim, dialog).St(St().AbsFill())
}

// ---- 导航与列表 ----

// Tabs 分段页签。
func Tabs(keys, labels []string, active string, onPick func(string)) *Node {
	items := make([]*Node, 0, len(keys))
	for i, k := range keys {
		label := k
		if i < len(labels) {
			label = labels[i]
		}
		st := St().Text(12.5, 500, ColTextMuted).PadXY(14, 7).R(8)
		if k == active {
			st.Text(12.5, 600, ColText).BgC(ColBgSoft).Stroke(1, ColStroke)
		}
		id := k
		items = append(items, Text(label).St(st).Tap(func() {
			if onPick != nil {
				onPick(id)
			}
			RequestRedraw()
		}))
	}
	return Row(items...).St(St().GapOf(4).Padding(4).R(12).BgC(ColInput).Stroke(1, ColStroke))
}

// TabsBar 下划线风格页签（gio 常见导航样式）。
func TabsBar(keys, labels []string, active string, onPick func(string)) *Node {
	items := make([]*Node, 0, len(keys))
	for i, k := range keys {
		label := k
		if i < len(labels) {
			label = labels[i]
		}
		color, weight := ColTextMuted, int32(500)
		if k == active {
			color, weight = ColText, 600
		}
		cells := []*Node{Text(label).St(St().Text(13, weight, color).PadXY(2, 8))}
		if k == active {
			cells = append(cells, Box().St(St().Wp(100).H(2).R(1).BgC(ColPrimary)))
		}
		col := Column(cells...).St(St().GapOf(6))
		id := k
		items = append(items, col.Tap(func() {
			if onPick != nil {
				onPick(id)
			}
			RequestRedraw()
		}))
	}
	return Row(items...).St(St().GapOf(22)).AlignTo(AlignStart)
}

// Breadcrumb 面包屑（点击项回调，最后一项为当前页）。
func Breadcrumb(items []string, onPick func(int)) *Node {
	cells := make([]*Node, 0, len(items)*2)
	for i, s := range items {
		if i > 0 {
			cells = append(cells, Text("/").St(St().Text(11.5, 400, ColPlaceholder)))
		}
		idx := i
		color := ColTextMuted
		if i == len(items)-1 {
			color = ColTextDim
		}
		cells = append(cells, Text(s).St(St().Text(12, 500, color)).Tap(func() {
			if onPick != nil {
				onPick(idx)
			}
			RequestRedraw()
		}))
	}
	return Row(cells...).St(St().GapOf(8)).AlignTo(AlignCenter)
}

// Accordion 折叠面板。
func Accordion(key, title, body string, open bool, onToggle func()) *Node {
	head := Row(
		Text(title).St(St().Text(13, 600, ColText)).Grow(1),
		Text(map[bool]string{true: "−", false: "+"}[open]).St(St().Text(14, 600, ColTextMuted)),
	).St(St().Padding(14).R(10).BgC(ColBgSoft))
	head.K(key + "-head").Tap(func() {
		if onToggle != nil {
			onToggle()
		}
		RequestRedraw()
	})
	col := Column(head)
	if open {
		col.Add(Text(body).St(St().Text(12.5, 400, ColTextDim).WrapOn().PadOnly(12, 14, 12, 14)))
	}
	return col.St(St().GapOf(4))
}

// ListItem 可点击列表项（左侧色点 + 标题 + 右侧说明）。
func ListItem(key, title, sub, color string, onClick func()) *Node {
	row := Row(
		Dot(color, 8),
		Column(
			Text(title).St(St().Text(13, 500, ColText)),
			Text(sub).St(St().Text(11.5, 400, ColTextMuted).PadOnly(3, 0, 0, 0)),
		).Grow(1),
		Text("→").St(St().Text(13, 500, ColPlaceholder)),
	).St(St().GapOf(12).Padding(14).R(12).BgC(ColBgSoft).Stroke(1, ColStroke)).AlignTo(AlignCenter)
	return row.K(key).Tap(func() {
		if onClick != nil {
			onClick()
		}
		RequestRedraw()
	})
}

// Table 数据表格：固定/弹性列宽、表头、斑马纹、整行点击。
func Table(headers []string, widths []float32, rows [][]*Node, onRow func(int)) *Node {
	head := Row().St(St().GapOf(12).PadXY(16, 10).BgC(ColHeader))
	for i, h := range headers {
		cell := Text(h).St(St().Text(11.5, 600, ColTextMuted))
		if i < len(widths) && widths[i] > 0 {
			cell.St(St().W(widths[i]))
		} else {
			cell.Grow(1)
		}
		head.Add(cell)
	}
	body := make([]*Node, 0, len(rows))
	for ri, cells := range rows {
		bg := ColRow
		if ri%2 == 1 {
			bg = ColRowAlt
		}
		r := Row().St(St().GapOf(12).PadXY(16, 11).BgC(bg)).AlignTo(AlignCenter)
		for ci, c := range cells {
			if c == nil {
				c = Text("—").St(St().Text(12, 400, ColTextMuted))
			}
			if ci < len(widths) && widths[ci] > 0 {
				c.St(St().W(widths[ci]))
			} else {
				c.Grow(1)
			}
			r.Add(c)
		}
		idx := ri
		r.K("row-" + strconv.Itoa(ri)).Tap(func() {
			if onRow != nil {
				onRow(idx)
			}
			RequestRedraw()
		})
		body = append(body, r)
	}
	return Column(append([]*Node{Divider(), head, Divider()}, body...)...).
		St(St().R(12).Stroke(1, ColStroke).Clipped())
}

// ---- 浮层 ----

// TooltipOf 给节点绑定悬停提示。
func TooltipOf(n *Node, text string) *Node {
	if n == nil || text == "" {
		return n
	}
	n.OnHover = func(hover bool) {
		if hover {
			x, y := MousePos()
			SetTooltip(text, x, y)
		} else {
			SetTooltip("", 0, 0)
		}
	}
	return n
}

// TooltipLayer 悬停提示浮层（放在根 Stack 最后一层）。
func TooltipLayer() *Node {
	if tooltipText == "" {
		return nil
	}
	return Text(tooltipText).St(St().Text(11.5, 500, ColText).PadXY(9, 6).R(7).
		BgC(ColPanel).Stroke(1, ColStroke).Shadow(18, 6, 0, "rgba(0,0,0,0.5)").
		AbsTL(tooltipX+14, tooltipY+18))
}

// whiteA 构造白色半透明（骨架屏呼吸层）。
func whiteA(a float32) gpu.Paint { return gpu.Paint{R: 1, G: 1, B: 1, A: a} }
