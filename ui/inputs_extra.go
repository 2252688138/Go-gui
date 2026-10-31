// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// 输入类组件补充：AutoComplete / TagInput / Rate / RangeSlider / KeyCapture /
// TimePicker / Cascader / Upload / ColorPicker。
//
// 浮层统一约定：控件本体渲染在树里，浮层由对应的 *Layers() 放在根 Stack 末尾，
// 锚点取控件上一帧矩形（LastFrameOf），定位不依赖调用方算坐标。

// ---- 浮层定位助手 ----

// anchorBelow 把浮层锚定到 key 节点下方（越界时上翻 / 左右夹取）。
func anchorBelow(key string, panel *Node, w, h float32) *Node {
	if panel == nil {
		return nil
	}
	r, ok := LastFrameOf(key)
	if !ok {
		return nil
	}
	x := r.X
	y := r.Y + r.H + 4
	vw, vh := ViewportW(), ViewportH()
	if vw <= 0 {
		vw = 1024
	}
	if vh <= 0 {
		vh = 700
	}
	if x+w > vw-8 {
		x = vw - w - 8
	}
	if x < 8 {
		x = 8
	}
	if y+h > vh-8 {
		if up := r.Y - 4 - h; up > 8 {
			y = up
		}
	}
	return panel.St(St().AbsTL(x, y))
}

// layerCatcher 浮层背后点一下关闭的透明遮罩。
func layerCatcher(key string, closeAll func()) *Node {
	return Box().St(St().AbsFill()).K(key).Tap(func() {
		if closeAll != nil {
			closeAll()
		}
		RequestRedraw()
	})
}

// ---- AutoComplete ----

// AutoCompleteState 输入联想状态。
type AutoCompleteState struct {
	Key   string
	Open  bool
	Hover int
	Value string

	onChange func(string)
	opts     []string
	items    []string
}

var autoStates = map[string]*AutoCompleteState{}

// NewAutoComplete 创建（或复用）联想状态。
func NewAutoComplete(key string) *AutoCompleteState {
	if s, ok := autoStates[key]; ok {
		return s
	}
	s := &AutoCompleteState{Key: key, Hover: -1}
	autoStates[key] = s
	return s
}

// Close 收起浮层。
func (s *AutoCompleteState) Close() {
	if s == nil {
		return
	}
	s.Open = false
	s.Hover = -1
	CloseOverlay(s.Key + "-ac")
}

// filter 按输入过滤候选（不区分大小写包含匹配，最多 8 条）。
func (s *AutoCompleteState) filter() {
	q := strings.ToLower(strings.TrimSpace(s.Value))
	s.items = s.items[:0]
	for _, o := range s.opts {
		if q == "" || strings.Contains(strings.ToLower(o), q) {
			s.items = append(s.items, o)
			if len(s.items) >= 8 {
				break
			}
		}
	}
	if s.Hover >= len(s.items) {
		s.Hover = len(s.items) - 1
	}
}

// pick 选中第 i 项。
func (s *AutoCompleteState) pick(i int) {
	if i < 0 || i >= len(s.items) {
		return
	}
	s.Value = s.items[i]
	s.Close()
	inputCaret[s.Key] = len([]rune(s.Value))
	if s.onChange != nil {
		s.onChange(s.Value)
	}
	RequestRedraw()
}

// AutoComplete 输入联想框（↑↓ 选择，Enter 确认，Esc 关闭）。
func AutoComplete(st *AutoCompleteState, opts []string, value, placeholder string, onChange func(string)) *Node {
	if st == nil {
		return nil
	}
	st.opts = opts
	st.Value = value
	st.onChange = onChange
	st.filter()
	field := Input(st.Key, value, placeholder, func(v string) {
		st.Value = v
		st.Open = true
		st.filter()
		if onChange != nil {
			onChange(v)
		}
		RequestRedraw()
	})
	n := Stack(
		field,
		Text("▾").St(St().Text(10, 500, ColTextMuted).W(22).H(FieldH).CenterYOn().
			TAlign(AlignCenter).AbsTR(6, 0)),
	).St(St().Wp(100).H(FieldH))
	f := field
	f.OnKey = func(k uint32) {
		switch k {
		case VK_DOWN:
			st.Open = true
			if len(st.items) > 0 {
				st.Hover = (st.Hover + 1) % len(st.items)
			}
		case VK_UP:
			st.Open = true
			if len(st.items) > 0 {
				st.Hover = (st.Hover - 1 + len(st.items)) % len(st.items)
			}
		case VK_RETURN:
			if st.Open && st.Hover >= 0 {
				st.pick(st.Hover)
			}
		case VK_ESCAPE:
			st.Close()
		case 0x08, 0x2E, 0x25, 0x27, 0x24, 0x23:
			// 光标与删除交给输入框自身
			callInputKey(st.Key, k)
			st.filter()
			st.Open = true
		}
		RequestRedraw()
	}
	n.OnClick = f.OnClick
	return n
}

// AutoCompleteLayers 渲染当前展开的联想浮层（根 Stack 末尾）。
func AutoCompleteLayers() *Node {
	layers := make([]*Node, 0, len(autoStates))
	for _, s := range autoStates {
		if !s.Open || len(s.items) == 0 {
			continue
		}
		key := s.Key
		r, ok := LastFrameOf(key)
		if !ok {
			continue
		}
		w := r.W
		for _, it := range s.items {
			if tw := TextWidth(it, 13, 400) + 30; tw > w {
				w = tw
			}
		}
		if w < 180 {
			w = 180
		}
		if w > 340 {
			w = 340
		}
		rows := make([]*Node, 0, len(s.items))
		for i, it := range s.items {
			idx := i
			label := it
			if idx == s.Hover {
				label = "▸ " + label
			}
			row := Text(label).St(St().W(w-16).H(28).PadXY(8, 0).R(7).
				Text(12.5, 400, ColTextDim).CenterYOn())
			if idx == s.Hover {
				row.St(St().BgC(paintWithAlpha(ColPrimary, 0.18)))
			}
			row.K(fmt.Sprintf("%s-ac-%d", s.Key, i)).Tap(func() { s.pick(idx) })
			rows = append(rows, row)
		}
		h := float32(len(rows))*30 + 12
		panel := Column(rows...).St(St().W(w).Padding(6).GapOf(2).R(12).BgC(ColPanel).
			Stroke(1, ColStroke).Shadow(28, 12, 0, "rgba(0,0,0,0.55)"))
		if p := anchorBelow(key, panel, w, h); p != nil {
			layers = append(layers, p)
		}
	}
	if len(layers) == 0 {
		return nil
	}
	catcher := layerCatcher("ac-catcher", func() {
		for _, s := range autoStates {
			s.Close()
		}
	})
	return Stack(append([]*Node{catcher}, layers...)...).St(St().AbsFill())
}

// ---- TagInput ----

type tagInputState struct {
	Text string
}

var tagStates = map[string]*tagInputState{}

// TagInput 标签输入：回车 / 逗号添加，退格在空输入时删除末项，自动去重。
func TagInput(key string, tags []string, placeholder string, onChange func([]string)) *Node {
	ts := tagStates[key]
	if ts == nil {
		ts = &tagInputState{}
		tagStates[key] = ts
	}
	if placeholder == "" {
		placeholder = "输入后回车添加"
	}
	add := func() {
		v := strings.TrimSpace(ts.Text)
		if v == "" {
			return
		}
		for _, t := range tags {
			if t == v {
				ts.Text = ""
				return
			}
		}
		tags = append(tags, v)
		ts.Text = ""
		inputCaret[key+"-tag"] = 0
		if onChange != nil {
			onChange(tags)
		}
		RequestRedraw()
	}
	remove := func(i int) {
		if i < 0 || i >= len(tags) {
			return
		}
		tags = append(tags[:i], tags[i+1:]...)
		if onChange != nil {
			onChange(tags)
		}
		RequestRedraw()
	}

	chips := make([]*Node, 0, len(tags)+1)
	for i, t := range tags {
		idx := i
		chips = append(chips, ChipRemovable(t, func() { remove(idx) }))
	}
	field := Input(key+"-tag", ts.Text, placeholder, func(v string) {
		ts.Text = v
	})
	field.St(St().W(150).NoFocusRing())
	field.OnKey = func(k uint32) {
		switch k {
		case VK_RETURN:
			add()
		case VK_BACK:
			if ts.Text == "" && len(tags) > 0 {
				remove(len(tags) - 1)
			} else {
				callInputKey(key+"-tag", k)
			}
		case VK_ESCAPE:
			SetFocus("")
		default:
			callInputKey(key+"-tag", k)
		}
		RequestRedraw()
	}
	field.OnChar = func(r rune) {
		if r == ',' || r == '，' {
			add()
			return
		}
		callInputChar(key+"-tag", r)
		RequestRedraw()
	}
	chips = append(chips, field)
	return Wrap(chips...).St(St().Wp(100).MinHeight(FieldH).Padding(6).GapOf(6).R(10).
		BgC(ColInput).Stroke(1, focusStroke(key+"-tag")).Clipped())
}

// focusStroke 焦点描边色。
func focusStroke(key string) string {
	if IsFocused(key) {
		return ColPrimary
	}
	return ColStroke
}

// ---- Rate ----

var rateHover = map[string]float32{}

// Rate 星级评分（默认整星，allowHalf 支持半星）。
func Rate(key string, v float32, max int, allowHalf bool, onChange func(float32)) *Node {
	if max <= 0 {
		max = 5
	}
	show := v
	if hv, ok := rateHover[key]; ok && hv > 0 {
		show = hv
	}
	items := make([]*Node, 0, max)
	for i := 0; i < max; i++ {
		idx := i
		on := show >= float32(i+1)
		half := !on && show >= float32(i)+0.5
		glyph := "☆"
		color := ColPlaceholder
		if on || half {
			glyph = "★"
			color = "#f5c518"
		}
		star := Text(glyph).St(St().Text(20, 400, color).W(26).H(26).TAlign(AlignCenter))
		if half {
			// 半星：底层空心星 + 左半实心星裁剪。两层字形宽都固定为 26，上层文字
			// 在外层 13 宽容器里仍按 26 宽排版，才能在 x=r.W/2 处精确切成左右两半
			// （若让裁剪容器收缩文字 Frame，居中文字会被重新居中 → 半星错位）。
			star = Stack(
				Text("☆").St(St().AbsFill().Text(20, 400, ColPlaceholder).TAlign(AlignCenter)),
				Row(Text("★").St(St().W(26).H(26).Text(20, 400, "#f5c518").TAlign(AlignCenter))).St(
					St().W(13).H(26).Clipped()),
			).St(St().W(26).H(26))
		}
		star.K(fmt.Sprintf("%s-star-%d", key, i))
		star.OnHoverEvt(func(h bool) {
			if h {
				if allowHalf {
					x, _ := MousePos()
					if r, ok := LastFrameOf(fmt.Sprintf("%s-star-%d", key, idx)); ok {
						if x-r.X > r.W/2 {
							rateHover[key] = float32(idx) + 1
						} else {
							rateHover[key] = float32(idx) + 0.5
						}
					}
				} else {
					rateHover[key] = float32(idx) + 1
				}
			} else if rateHover[key] == float32(idx)+1 || rateHover[key] == float32(idx)+0.5 {
				delete(rateHover, key)
			}
			RequestRedraw()
		})
		star.TapAt(func(x, _ float32) {
			nv := float32(idx) + 1
			if allowHalf && x < 13 {
				nv = float32(idx) + 0.5
			}
			if nv == v {
				nv = 0 // 再次点击同一星清除
			}
			if onChange != nil {
				onChange(nv)
			}
			delete(rateHover, key)
			RequestRedraw()
		})
		items = append(items, star)
	}
	return Row(items...).St(St().GapOf(2))
}

// ---- RangeSlider ----

// RangeSlider 双滑块区间选择（lo/hi 均为 0..1）。
func RangeSlider(key string, lo, hi float32, onChange func(lo, hi float32)) *Node {
	const width, trackH, thumb = float32(220), float32(18), float32(14)
	clamp := func(v float32) float32 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	lo, hi = clamp(lo), clamp(hi)
	if lo > hi {
		lo, hi = hi, lo
	}
	usable := width - thumb
	posOf := func(v float32) float32 { return v * usable }
	dragKey := key + "-drag"
	// 记录当前拖动的是哪个把手（-1 未开始，0 低值，1 高值）
	if _, ok := splitState[dragKey]; !ok {
		splitState[dragKey] = -1
	}
	set := func(x float32) {
		nv := clamp((x - thumb/2) / usable)
		which := splitState[dragKey]
		if which < 0 {
			if math.Abs(float64(nv-lo)) <= math.Abs(float64(nv-hi)) {
				which = 0
			} else {
				which = 1
			}
			splitState[dragKey] = which
		}
		if which == 0 {
			lo = nv
			if lo > hi {
				lo = hi
			}
		} else {
			hi = nv
			if hi < lo {
				hi = lo
			}
		}
		if onChange != nil {
			onChange(lo, hi)
		}
		RequestRedraw()
	}
	trackY := (trackH - 4) / 2
	thumbY := (trackH - thumb) / 2
	n := Stack(
		Box().St(St().W(width).H(4).R(2).BgC(ColTrack).AbsTL(0, trackY)),
		Box().St(St().W(posOf(hi)-posOf(lo)+thumb).H(4).R(2).
			BgGrad(ColPrimary, ColPrimary2, 90).AbsTL(posOf(lo), trackY)),
		Box().St(St().W(thumb).H(thumb).R(thumb/2).BgC("#ffffff").Stroke(3, ColPrimary).
			Shadow(10, 2, 0, "rgba(99,102,241,0.55)").AbsTL(posOf(lo), thumbY)),
		Box().St(St().W(thumb).H(thumb).R(thumb/2).BgC("#ffffff").Stroke(3, ColPrimary).
			Shadow(10, 2, 0, "rgba(99,102,241,0.55)").AbsTL(posOf(hi), thumbY)),
	).St(St().W(width).H(trackH))
	n.K(key).TapAt(func(x, _ float32) { set(x) }).
		DragEvt(func(x, _ float32) { set(x) }).
		DragEndEvt(func() { splitState[dragKey] = -1 })
	return n
}

// ---- KeyCapture ----

// KeyCapture 快捷键录制框：点击后按下组合键写入；Esc 取消，Backspace 清空。
func KeyCapture(key, binding string, onChange func(string)) *Node {
	recording := keyCapture.recording(key)
	label := binding
	color := ColText
	if recording {
		label = "请按下快捷键…"
		color = ColPrimary
	} else if label == "" {
		label = "未绑定"
		color = ColPlaceholder
	}
	n := Row(
		Text(label).St(St().Text(12.5, 500, color)).Grow(1),
		Text("⌨").St(St().Text(13, 400, ColTextMuted)),
	).St(St().Wp(100).H(FieldH).PadXY(12, 0).GapOf(8).R(10).
		BgC(ColInput).Stroke(1, focusStroke(key)).FocusOn()).AlignTo(AlignCenter).K(key)
	n.OnClick = func() {
		keyCapture.set(key, true)
		SetFocus(key)
		RequestRedraw()
	}
	n.OnKey = func(k uint32) {
		switch k {
		case VK_ESCAPE:
			keyCapture.set(key, false)
			SetFocus("")
		case VK_BACK, VK_DELETE:
			keyCapture.set(key, false)
			if onChange != nil {
				onChange("")
			}
		case VK_CONTROL, VK_SHIFT, VK_MENU:
			return // 只按修饰键不结束录制
		default:
			combo := comboString(k)
			keyCapture.set(key, false)
			if onChange != nil {
				onChange(combo)
			}
		}
		RequestRedraw()
	}
	return n
}

// keyCaptureState 录制状态（按 key）。
type keyCaptureState struct{ m map[string]bool }

var keyCapture = &keyCaptureState{}

func (s *keyCaptureState) set(k string, v bool) {
	if s.m == nil {
		s.m = map[string]bool{}
	}
	s.m[k] = v
}

func (s *keyCaptureState) recording(k string) bool { return s.m[k] }

// comboString 用当前修饰键 + 主键生成 "Ctrl+Shift+K" 形式。
func comboString(vk uint32) string {
	var parts []string
	if CtrlDown() {
		parts = append(parts, "Ctrl")
	}
	if ShiftDown() {
		parts = append(parts, "Shift")
	}
	if AltDown() {
		parts = append(parts, "Alt")
	}
	parts = append(parts, KeyName(vk))
	return strings.Join(parts, "+")
}

// CtrlDown / ShiftDown / AltDown 当前修饰键状态（快捷键组件用）。
func CtrlDown() bool {
	if a := currentApp.Load(); a != nil {
		return a.ctrlDown
	}
	return false
}

func ShiftDown() bool {
	if a := currentApp.Load(); a != nil {
		return a.shiftDown
	}
	return false
}

func AltDown() bool {
	if a := currentApp.Load(); a != nil {
		return a.altDown
	}
	return false
}

// KeyName 虚拟键码转显示名（快捷键展示 / 序列化）。
func KeyName(vk uint32) string {
	switch vk {
	case VK_RETURN:
		return "Enter"
	case VK_ESCAPE:
		return "Esc"
	case VK_TAB:
		return "Tab"
	case VK_SPACE:
		return "Space"
	case VK_BACK:
		return "Backspace"
	case VK_DELETE:
		return "Delete"
	case VK_LEFT:
		return "Left"
	case VK_RIGHT:
		return "Right"
	case VK_UP:
		return "Up"
	case VK_DOWN:
		return "Down"
	case 0x24:
		return "Home"
	case 0x23:
		return "End"
	case 0x21:
		return "PageUp"
	case 0x22:
		return "PageDown"
	case VK_F11:
		return "F11"
	}
	if vk >= 0x70 && vk <= 0x7B {
		return "F" + strconv.Itoa(int(vk-0x6F))
	}
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	return fmt.Sprintf("VK%d", vk)
}

// ---- TimePicker ----

// TimeState 时间选择状态。
type TimeState struct {
	Key  string
	Open bool
	// Second 是否显示秒列。
	Second  bool
	H, M, S int

	onChange func(string)
}

var timeStates = map[string]*TimeState{}

// NewTime 创建时间选择状态。
func NewTime(key string) *TimeState {
	if s, ok := timeStates[key]; ok {
		return s
	}
	s := &TimeState{Key: key}
	timeStates[key] = s
	return s
}

// TimeValue 当前时间 HH:MM 或 HH:MM:SS。
func (s *TimeState) TimeValue() string {
	if s == nil {
		return ""
	}
	if s.Second {
		return fmt.Sprintf("%02d:%02d:%02d", s.H, s.M, s.S)
	}
	return fmt.Sprintf("%02d:%02d", s.H, s.M)
}

// TimePicker 时间选择框（点击展开时/分/秒列）。
func TimePicker(st *TimeState, value string, onChange func(string)) *Node {
	if st == nil {
		return nil
	}
	st.onChange = onChange
	if value == "" {
		value = st.TimeValue()
	}
	label := value
	color := ColText
	if label == "" {
		label = "选择时间"
		color = ColPlaceholder
	}
	n := Row(
		Text(label).St(St().Text(13, 400, color)),
		Spacer(),
		Text("◷").St(St().Text(12, 400, ColTextMuted)),
	).St(St().Wp(100).H(FieldH).PadXY(12, 0).GapOf(8).R(10).
		BgC(ColInput).Stroke(1, focusStroke(st.Key)).FocusOn()).AlignTo(AlignCenter).K(st.Key)
	n.OnClick = func() {
		st.Open = !st.Open
		if st.Open {
			SetFocus(st.Key)
			if s := value; s != "" {
				parts := strings.Split(s, ":")
				if len(parts) >= 2 {
					st.H, _ = strconv.Atoi(parts[0])
					st.M, _ = strconv.Atoi(parts[1])
					if len(parts) > 2 {
						st.S, _ = strconv.Atoi(parts[2])
					}
				}
			}
			OpenOverlay(st.Key+"-time", func() bool {
				st.Open = false
				RequestRedraw()
				return true
			})
		} else {
			CloseOverlay(st.Key + "-time")
		}
		RequestRedraw()
	}
	n.OnKey = func(k uint32) {
		switch k {
		case VK_RETURN:
			n.OnClick()
		case VK_ESCAPE:
			st.Open = false
			CloseOverlay(st.Key + "-time")
			RequestRedraw()
		}
	}
	return n
}

// TimeLayers 时间浮层（根 Stack 末尾）。
func TimeLayers() *Node {
	layers := make([]*Node, 0, len(timeStates))
	for _, s := range timeStates {
		if !s.Open {
			continue
		}
		cols := []*Node{
			timeColumn(s, "时", 24, s.H, func(v int) { s.H = v }),
			timeColumn(s, "分", 60, s.M, func(v int) { s.M = v }),
		}
		if s.Second {
			cols = append(cols, timeColumn(s, "秒", 60, s.S, func(v int) { s.S = v }))
		}
		now := Text("现在").St(St().Text(11.5, 500, ColPrimary).W(46).H(26).R(6).
			CenterYOn().TAlign(AlignCenter).BgC(ColHover))
		now.Tap(func() {
			t := nowTimeParts()
			s.H, s.M, s.S = t[0], t[1], t[2]
			s.onChange(s.TimeValue())
			RequestRedraw()
		})
		ok := Text("确定").St(St().Text(11.5, 600, "#ffffff").W(46).H(26).R(6).
			CenterYOn().TAlign(AlignCenter).BgGrad(ColPrimary, ColPrimary2, 90))
		ok.Tap(func() {
			s.Open = false
			CloseOverlay(s.Key + "-time")
			if s.onChange != nil {
				s.onChange(s.TimeValue())
			}
			RequestRedraw()
		})
		w := float32(3*62 + 20)
		panel := Column(
			Row(cols...).St(St().GapOf(4)),
			Row(now, ok).St(St().GapOf(8).MarOnly(8, 0, 0, 0)).JustifyTo(AlignEnd),
		).St(St().W(w).Padding(10).GapOf(8).R(12).BgC(ColPanel).
			Stroke(1, ColStroke).Shadow(28, 12, 0, "rgba(0,0,0,0.55)"))
		if p := anchorBelow(s.Key, panel, w, 260); p != nil {
			layers = append(layers, p)
		}
	}
	if len(layers) == 0 {
		return nil
	}
	catcher := layerCatcher("time-catcher", func() {
		for _, s := range timeStates {
			s.Open = false
			CloseOverlay(s.Key + "-time")
		}
	})
	return Stack(append([]*Node{catcher}, layers...)...).St(St().AbsFill())
}

// timeColumn 时/分/秒列（滚动选择）。
func timeColumn(s *TimeState, label string, n, cur int, set func(int)) *Node {
	rows := make([]*Node, 0, n)
	for i := 0; i < n; i++ {
		v := i
		txt := fmt.Sprintf("%02d", v)
		color := ColTextDim
		var bg any = "transparent"
		if v == cur {
			color = "#ffffff"
			bg = paintWithAlpha(ColPrimary, 0.35)
		}
		row := Text(txt).St(St().Text(12, 500, color).W(56).H(24).R(6).
			CenterYOn().TAlign(AlignCenter).BgC(bg))
		row.K(fmt.Sprintf("%s-%s-%d", s.Key, label, v)).Tap(func() {
			set(v)
			RequestRedraw()
		})
		rows = append(rows, row)
	}
	scrollKey := fmt.Sprintf("%s-%s-scroll", s.Key, label)
	// 打开时滚到当前值附近
	if s.Open {
		if _, ok := scrolledOnce[scrollKey]; !ok {
			scrolledOnce[scrollKey] = true
			ScrollTo(scrollKey, float32(cur-2)*24)
		}
	}
	body := Scroll(rows...).K(scrollKey).St(St().H(150))
	return Column(
		Text(label).St(St().Text(10.5, 500, ColTextMuted).W(56).TAlign(AlignCenter)),
		body,
	).St(St().GapOf(4))
}

var scrolledOnce = map[string]bool{}

// nowTimeParts 当前时分秒。
func nowTimeParts() [3]int {
	t := nowLocal()
	return [3]int{t.Hour(), t.Minute(), t.Second()}
}

// ---- Cascader ----

// CascadeOption 级联选项。
type CascadeOption struct {
	Value    string
	Label    string
	Disabled bool
	Children []CascadeOption
	// Load 懒加载子级（Children 为空时点击调用一次）。
	Load func() []CascadeOption
}

// CascadeState 级联选择状态。
type CascadeState struct {
	Key   string
	Open  bool
	Path  []int
	Value []string

	opts     []CascadeOption
	onChange func([]string)
}

var cascadeStates = map[string]*CascadeState{}

// NewCascade 创建级联状态。
func NewCascade(key string) *CascadeState {
	if s, ok := cascadeStates[key]; ok {
		return s
	}
	s := &CascadeState{Key: key}
	cascadeStates[key] = s
	return s
}

// levels 返回当前路径上的各级选项列表。
func (s *CascadeState) levels() [][]CascadeOption {
	out := [][]CascadeOption{s.opts}
	cur := s.opts
	for _, idx := range s.Path {
		if idx < 0 || idx >= len(cur) {
			break
		}
		o := cur[idx]
		if o.Load != nil && len(o.Children) == 0 {
			o.Children = o.Load()
			cur[idx] = o
		}
		cur = o.Children
		if len(cur) == 0 {
			break
		}
		out = append(out, cur)
	}
	return out
}

// CascaderPath 当前选中的标签路径。
func (s *CascadeState) CascaderPath() string {
	if s == nil || len(s.Value) == 0 {
		return ""
	}
	return strings.Join(s.Value, " / ")
}

// Cascader 级联选择框（省市区 / 分类等）。
func Cascader(st *CascadeState, opts []CascadeOption, path []string, placeholder string, onChange func([]string)) *Node {
	if st == nil {
		return nil
	}
	st.opts = opts
	st.onChange = onChange
	st.Value = path
	label := st.CascaderPath()
	color := ColText
	if label == "" {
		if placeholder == "" {
			placeholder = "请选择"
		}
		label, color = placeholder, ColPlaceholder
	}
	n := Row(
		Text(label).St(St().Text(13, 400, color)),
		Spacer(),
		Text("›").St(St().Text(12, 500, ColTextMuted)),
	).St(St().Wp(100).H(FieldH).PadXY(12, 0).GapOf(8).R(10).
		BgC(ColInput).Stroke(1, focusStroke(st.Key)).FocusOn()).AlignTo(AlignCenter).K(st.Key)
	n.OnClick = func() {
		st.Open = !st.Open
		if st.Open {
			SetFocus(st.Key)
			OpenOverlay(st.Key+"-cascade", func() bool {
				st.Open = false
				RequestRedraw()
				return true
			})
		} else {
			CloseOverlay(st.Key + "-cascade")
		}
		RequestRedraw()
	}
	n.OnKey = func(k uint32) {
		if k == VK_ESCAPE {
			st.Open = false
			CloseOverlay(st.Key + "-cascade")
			RequestRedraw()
		}
	}
	return n
}

// CascaderLayers 级联浮层（根 Stack 末尾）。
func CascaderLayers() *Node {
	layers := make([]*Node, 0, len(cascadeStates))
	for _, s := range cascadeStates {
		if !s.Open {
			continue
		}
		levels := s.levels()
		cols := make([]*Node, 0, len(levels))
		for li, list := range levels {
			level := li
			rows := make([]*Node, 0, len(list))
			activeIdx := -1
			if level < len(s.Path) {
				activeIdx = s.Path[level]
			}
			for i, o := range list {
				idx := i
				hasChild := len(o.Children) > 0 || o.Load != nil
				color := ColTextDim
				if idx == activeIdx {
					color = "#ffffff"
				}
				if o.Disabled {
					color = ColPlaceholder
				}
				label := o.Label
				if hasChild {
					label += "  ›"
				}
				row := Text(label).St(St().Text(12.5, 400, color).W(120).H(28).R(6).
					PadXY(8, 0).CenterYOn())
				if idx == activeIdx {
					row.St(St().BgC(paintWithAlpha(ColPrimary, 0.22)))
				}
				row.K(fmt.Sprintf("%s-cas-%d-%d", s.Key, level, i)).Tap(func() {
					if o.Disabled {
						return
					}
					if hasChild {
						s.Path = append(s.Path[:level], idx)
						RequestRedraw()
						return
					}
					// 叶子：提交完整路径
					var labels []string
					for l, ii := range s.Path {
						lv := s.levels()
						if l < len(lv) && ii < len(lv[l]) {
							labels = append(labels, lv[l][ii].Label)
						}
					}
					if level < len(s.Path) {
						labels = labels[:level]
					}
					labels = append(labels, o.Label)
					s.Value = labels
					s.Open = false
					CloseOverlay(s.Key + "-cascade")
					if s.onChange != nil {
						s.onChange(labels)
					}
					RequestRedraw()
				})
				rows = append(rows, row)
			}
			cols = append(cols, Column(rows...).St(St().GapOf(2)))
		}
		w := float32(len(cols)*126 + 16)
		h := float32(300)
		panel := Row(cols...).St(St().W(w).H(240).Padding(8).GapOf(2).R(12).BgC(ColPanel).
			Stroke(1, ColStroke).Shadow(28, 12, 0, "rgba(0,0,0,0.55)").Clipped())
		if p := anchorBelow(s.Key, panel, w, h); p != nil {
			layers = append(layers, p)
		}
	}
	if len(layers) == 0 {
		return nil
	}
	catcher := layerCatcher("cascade-catcher", func() {
		for _, s := range cascadeStates {
			s.Open = false
			CloseOverlay(s.Key + "-cascade")
		}
	})
	return Stack(append([]*Node{catcher}, layers...)...).St(St().AbsFill())
}

// ---- Upload ----

// UploadFile 上传项。
type UploadFile struct {
	Path     string
	Name     string
	Size     int64
	Progress float32 // 0..1
	Err      string
}

// UploadOptions 上传组件参数。
type UploadOptions struct {
	Title   string
	Filter  string // 形如 "图片|*.png;*.jpg|所有文件|*.*"
	MaxSize int64  // 字节，0 表示不限制
	Height  float32
}

// Upload 上传列表：选择文件、进度、重试、移除、拖入。
func Upload(key string, files []UploadFile, opts UploadOptions, onPick func(), onRemove func(int), onRetry func(int)) *Node {
	if opts.Title == "" {
		opts.Title = "点击选择文件，或拖到这里"
	}
	if opts.Height <= 0 {
		opts.Height = 96
	}
	pick := func() {
		if onPick != nil {
			onPick()
			return
		}
		PickFilesAsync("选择文件", opts.Filter, func(paths []string) {
			for _, p := range paths {
				_ = p
			}
			RequestRedraw()
		})
	}
	zone := Stack(
		Box().St(St().AbsFill().R(12).BgC(ColInput).Stroke(1, ColStroke)),
		Column(
			Text("⇪").St(St().Text(22, 500, ColPlaceholder)),
			Text(opts.Title).St(St().Text(12, 500, ColTextMuted)),
			Text("支持多选"+sizeHint(opts.MaxSize)).St(St().Text(10.5, 400, ColPlaceholder)),
		).St(St().GapOf(4)).AlignTo(AlignCenter).St(St().AbsCenter()),
	).St(St().Wp(100).H(opts.Height).Clipped().NoFocusRing())
	zone.K(key + "-zone").Tap(pick)
	zone.OnDrop = func(paths []string) {
		if onPick != nil {
			onPick()
		}
	}

	rows := make([]*Node, 0, len(files))
	for i, f := range files {
		idx := i
		name := f.Name
		if name == "" {
			name = baseName(f.Path)
		}
		status := "完成"
		color := ColSuccess
		switch {
		case f.Err != "":
			status, color = "失败", ColDanger
		case f.Progress < 1:
			status, color = fmt.Sprintf("%.0f%%", f.Progress*100), ColInfo
		}
		row := Row(
			Text("▤").St(St().Text(13, 500, ColTextMuted)),
			Column(
				Text(name).St(St().Text(12, 500, ColText)),
				Progress(f.Progress, color).St(St().Wp(100)),
			).Grow(1).St(St().GapOf(5)),
			Text(status).St(St().Text(11, 500, color)),
			Text("×").St(St().Text(14, 500, ColTextMuted).W(22).TAlign(AlignCenter)).Tap(func() {
				if onRemove != nil {
					onRemove(idx)
				}
				RequestRedraw()
			}),
		).St(St().GapOf(10).Padding(10).R(8).BgC(ColBgSoft)).AlignTo(AlignCenter)
		if f.Err != "" && onRetry != nil {
			row.Add(Text("重试").St(St().Text(11, 600, ColPrimary)).Tap(func() {
				onRetry(idx)
				RequestRedraw()
			}))
		}
		rows = append(rows, row)
	}
	col := Column(zone)
	if len(rows) > 0 {
		col.Add(Column(rows...).St(St().GapOf(6).MarOnly(8, 0, 0, 0)))
	}
	return col.St(St().GapOf(2)).K(key)
}

// sizeHint 大小限制提示。
func sizeHint(max int64) string {
	if max <= 0 {
		return ""
	}
	return "，单个不超过 " + FormatBytes(max)
}

// baseName 路径取文件名（同时兼容 / 与 \）。
func baseName(p string) string {
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// ---- ColorPicker ----

// ColorState HSV 取色器状态。
type ColorState struct {
	Key   string
	Open  bool
	H     float32 // 0..360
	S     float32 // 0..1
	V     float32 // 0..1
	Value string

	// Picking 吸管拾色中：面板收起，全屏轮询鼠标位置，点击取色。
	// pickWaitRelease 记下起始时左键是否还按着（那次按下不能算取色）。
	Picking         bool
	pickWaitRelease bool
	pickPreview     string

	onChange func(string)
}

var colorStates = map[string]*ColorState{}

// NewColor 创建取色器状态。
func NewColor(key string) *ColorState {
	if s, ok := colorStates[key]; ok {
		return s
	}
	s := &ColorState{Key: key, S: 0.7, V: 0.9}
	colorStates[key] = s
	return s
}

// SetColor 设置颜色（HEX / #RGB），同步 HSV。
func (s *ColorState) SetColor(hex string) {
	if s == nil {
		return
	}
	r, g, b, ok := parseHexRGB(hex)
	if !ok {
		return
	}
	s.H, s.S, s.V = rgbToHSV(r, g, b)
	s.Value = hexOf(C(hex))
}

// Hex 当前颜色 HEX。
func (s *ColorState) Hex() string {
	if s == nil {
		return "#000000"
	}
	r, g, b := hsvToRGB(s.H, s.S, s.V)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// ColorPicker 取色器（点击展开 SV 面板 + 色相条 + HEX 输入）。
func ColorPicker(st *ColorState, value string, onChange func(string)) *Node {
	if st == nil {
		return nil
	}
	st.onChange = onChange
	if value != "" && value != st.Value {
		st.SetColor(value)
		st.Value = value
	}
	swatch := Box().St(St().W(20).H(20).R(6).BgC(st.Hex()).Stroke(1, ColStroke))
	label := st.Hex()
	if value != "" {
		label = value
	}
	n := Row(
		swatch,
		Text(label).St(St().Text(12.5, 400, ColText)),
		Spacer(),
		Text("▾").St(St().Text(9, 400, ColTextMuted)),
	).St(St().Wp(100).H(FieldH).PadXY(10, 0).GapOf(10).R(10).
		BgC(ColInput).Stroke(1, focusStroke(st.Key)).FocusOn()).AlignTo(AlignCenter).K(st.Key)
	n.OnClick = func() {
		st.Open = !st.Open
		if st.Open {
			SetFocus(st.Key)
			OpenOverlay(st.Key+"-color", func() bool {
				st.Open = false
				RequestRedraw()
				return true
			})
		} else {
			CloseOverlay(st.Key + "-color")
		}
		RequestRedraw()
	}
	n.OnKey = func(k uint32) {
		if k == VK_ESCAPE {
			st.Open = false
			CloseOverlay(st.Key + "-color")
			RequestRedraw()
		}
	}
	return n
}

// colorPickLayer 吸管拾色层：轮询指针位置与左键状态，点击即取该点屏幕颜色。
// 面板在拾色期间收起，取色成功后重新展开并回调 onChange。
// 层上带 AnimOn()，保证鼠标停在窗口外时仍有动画帧可用（否则没有帧就轮询不到点击）。
func colorPickLayer() *Node {
	var cur *ColorState
	for _, s := range colorStates {
		if s.Picking {
			cur = s
		}
	}
	if cur == nil {
		return nil
	}
	if ScreenPixelAt == nil || CursorAt == nil || LeftButtonDown == nil {
		cur.Picking = false
		return nil
	}
	done := func(ok bool) {
		cur.Picking = false
		cur.Open = ok
		if ok {
			// 重新登记浮层：点击面板外仍能收起
			OpenOverlay(cur.Key+"-color", func() bool {
				cur.Open = false
				RequestRedraw()
				return true
			})
			if cur.onChange != nil {
				cur.onChange(cur.Value)
			}
		}
		RequestRedraw()
	}
	if mx, my, ok := CursorAt(); ok {
		preview := cur.pickPreview
		if c, pok := ScreenPixelAt(mx, my); pok {
			preview = c
			cur.pickPreview = c
		}
		if cur.pickWaitRelease {
			if !LeftButtonDown() {
				cur.pickWaitRelease = false // 起手那次按下松开了，之后的点击才算取色
			}
		} else if LeftButtonDown() && preview != "" {
			cur.SetColor(preview)
			cur.Value = hexOf(C(preview))
			done(true)
			return nil
		}
	}
	cancel := (RightButtonDown != nil && RightButtonDown()) ||
		(KeyDown != nil && KeyDown(int(VK_ESCAPE)))
	if cancel {
		done(false)
		return nil
	}
	chip := cur.pickPreview
	if chip == "" {
		chip = cur.Hex()
	}
	hint := Row(
		Box().St(St().W(18).H(18).R(5).BgC(chip).Stroke(1, ColStroke)),
		Text("吸管取色：点击任意位置取色 · Esc / 右键取消").St(St().Text(12, 500, ColText)),
	).St(St().GapOf(8).PadXY(14, 10).R(10).BgC(ColPanel).Stroke(1, ColStroke).
		Shadow(24, 8, 0, "rgba(0,0,0,0.5)"))
	return hint.St(St().AbsTL(24, 24).AnimOn().NoFocusRing())
}

// ColorLayers 取色浮层（根 Stack 末尾）。
func ColorLayers() *Node {
	pick := colorPickLayer()
	layers := make([]*Node, 0, len(colorStates))
	switch {
	case pick != nil:
		// 拾色中：面板全部收起，只留拾色提示层
		layers = append(layers, pick)
	default:
		for _, s := range colorStates {
			if !s.Open {
				continue
			}
			const svW, svH = float32(200), float32(140)
			hue := s.H
			base := Box().St(St().AbsFill().BgC(hsvHex(hue, 1, 1)))
			// 渐变角度按 CSS 约定：0=向上 / 90=向右 / 180=向下。
			// 白→透明只能横向（90），黑→透明必须纵向（180），两者叠出标准 SV 面板；
			// 透明端点带 BgToSet 标记，不会被样式 merge 当成“未设置”丢掉。
			white := Box().St(St().AbsFill().BgGrad("rgba(255,255,255,1)", "rgba(255,255,255,0)", 90))
			black := Box().St(St().AbsFill().BgGrad("rgba(0,0,0,0)", "rgba(0,0,0,0.92)", 180))
			cursor := Box().St(St().W(10).H(10).R(5).Stroke(2, "#ffffff").
				BgC("transparent").AbsTL(s.S*svW-5, (1-s.V)*svH-5))
			sv := Stack(base, white, black, cursor).St(St().W(svW).H(svH).R(8).Clipped().NoFocusRing())
			sv.K(s.Key + "-sv")
			sv.OnClickAt = func(x, y float32) {
				ns, nv := clamp01(x/svW), 1-clamp01(y/svH)
				s.S, s.V = ns, nv
				s.Value = s.Hex()
				if s.onChange != nil {
					s.onChange(s.Value)
				}
				RequestRedraw()
			}
			sv.OnDrag = func(x, y float32) {
				s.S, s.V = clamp01(x/svW), 1-clamp01(y/svH)
				s.Value = s.Hex()
				if s.onChange != nil {
					s.onChange(s.Value)
				}
				RequestRedraw()
			}

			// 色相条：12 段横向渐变。必须用 Row 顺排——Stack 会把各段重叠在同一位置，
			// 结果只剩最后一段（红）可见，色相条退化成一个小色块。
			const hueH = float32(12)
			segW := svW / 12
			segs := make([]*Node, 0, 13)
			for i := 0; i < 12; i++ {
				a := hsvHex(float32(i)*30, 1, 1)
				b := hsvHex(float32(i+1)*30, 1, 1)
				w := segW
				if i == 11 {
					w = svW - segW*11 // 末段补齐，避免累计误差留缝
				}
				segs = append(segs, Box().St(St().W(w).H(hueH).BgGrad(a, b, 90)))
			}
			// 当前色相指示线（白色竖条 + 描边，亮色段上也看得见）
			markX := s.H / 360 * svW
			if markX < 1.5 {
				markX = 1.5
			}
			if markX > svW-1.5 {
				markX = svW - 1.5
			}
			segs = append(segs, Box().St(St().W(3).H(hueH).R(1.5).BgC("#ffffff").
				Stroke(0.5, "rgba(0,0,0,0.55)").AbsTL(markX-1.5, 0)))
			hueBar := Row(segs...).St(St().W(svW).H(hueH).R(5).Clipped().NoFocusRing())
			hueBar.K(s.Key + "-hue")
			hueBar.OnClickAt = func(x, _ float32) {
				s.H = clamp01(x/svW) * 360
				s.Value = s.Hex()
				if s.onChange != nil {
					s.onChange(s.Value)
				}
				RequestRedraw()
			}
			hueBar.OnDrag = func(x, _ float32) {
				s.H = clamp01(x/svW) * 360
				s.Value = s.Hex()
				if s.onChange != nil {
					s.onChange(s.Value)
				}
				RequestRedraw()
			}

			hexIn := Input(s.Key+"-hex", s.Hex(), "#rrggbb", func(v string) {
				if r, g, b, ok := parseHexRGB(v); ok {
					s.H, s.S, s.V = rgbToHSV(r, g, b)
					s.Value = hexOf(C(v))
					if s.onChange != nil {
						s.onChange(s.Value)
					}
				}
			})
			// Input 自带 Wp(100)，而 Style.merge 会忽略零值字段，W(110) 盖不掉百分比：
			// 结果宽度被算成 0，浮层里根本看不到 HEX 输入框。这里直接清掉百分比再给固定宽高。
			hexIn.Style.WidthPct = 0
			hexIn.Style.Width = 100
			hexIn.Style.Height = 28
			hexIn.Style.NoRing = true

			var eyedrop *Node
			if ScreenPixelAt != nil && CursorAt != nil && LeftButtonDown != nil {
				eyedrop = Text("吸管").St(St().Text(11, 500, ColTextDim).W(46).H(28).R(7).
					CenterYOn().TAlign(AlignCenter).BgC(ColBgSoft).Stroke(1, ColStroke))
				eyedrop.K(s.Key + "-eyedrop")
				eyedrop.Tap(func() {
					// 进入拾色模式：先收起面板（否则遮住要取色的区域），
					// 之后由 ColorLayers 里的拾色层轮询全屏鼠标/左键完成取色。
					s.Picking = true
					s.pickWaitRelease = LeftButtonDown()
					s.pickPreview = ""
					s.Open = false
					CloseOverlay(s.Key + "-color")
					RequestRedraw()
				})
			}

			presets := []string{"#6366f1", "#8b5cf6", "#ec4899", "#f59e0b", "#10b981", "#06b6d4", "#ef4444", "#64748b"}
			sw := make([]*Node, 0, len(presets))
			for _, p := range presets {
				col := p
				box := Box().St(St().W(18).H(18).R(5).BgC(col).Stroke(1, ColStroke))
				box.K(s.Key + "-preset-" + col).Tap(func() {
					s.SetColor(col)
					s.Value = col
					if s.onChange != nil {
						s.onChange(col)
					}
					RequestRedraw()
				})
				sw = append(sw, box)
			}
			footer := Row(append([]*Node{hexIn}, append([]*Node{Spacer()}, eyedrop)...)...).
				St(St().GapOf(6)).AlignTo(AlignCenter)
			panel := Column(
				sv,
				hueBar,
				footer,
				Row(sw...).St(St().GapOf(6)),
			).St(St().W(svW+20).Padding(10).GapOf(8).R(12).BgC(ColPanel).
				Stroke(1, ColStroke).Shadow(28, 12, 0, "rgba(0,0,0,0.55)"))
			if p := anchorBelow(s.Key, panel, svW+20, 260); p != nil {
				layers = append(layers, p)
			}
		}
	}
	if len(layers) == 0 {
		return nil
	}
	catcher := layerCatcher("color-catcher", func() {
		for _, s := range colorStates {
			s.Open = false
			CloseOverlay(s.Key + "-color")
		}
	})
	return Stack(append([]*Node{catcher}, layers...)...).St(St().AbsFill())
}

// 屏幕取色钩子（Windows 由 system_extra_windows.go 的 init 赋值；其他平台为 nil，
// 此时取色面板不显示「吸管」按钮）。
var (
	// ScreenPickColor 取当前鼠标位置的颜色（兼容旧接口）。
	ScreenPickColor func() (string, bool)
	// ScreenPixelAt 取指定屏幕坐标的颜色。
	ScreenPixelAt func(x, y int) (string, bool)
	// CursorAt 鼠标指针的屏幕坐标（物理像素）。
	CursorAt func() (x, y int, ok bool)
	// LeftButtonDown 鼠标左键当前是否按下（拾色模式点击判定）。
	LeftButtonDown func() bool
	// RightButtonDown 鼠标右键当前是否按下（拾色模式取消）。
	RightButtonDown func() bool
	// KeyDown 指定虚拟键是否按下（Esc 取消拾色）。
	KeyDown func(vk int) bool
)

// ---- 颜色工具 ----

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// parseHexRGB 解析 #rgb / #rrggbb。
func parseHexRGB(s string) (int, int, int, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v>>16) & 0xff, int(v>>8) & 0xff, int(v) & 0xff, true
}

// hsvHex HSV → HEX。
func hsvHex(h, s, v float32) string {
	r, g, b := hsvToRGB(h, s, v)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// hsvToRGB HSV（H 0..360）转 RGB。
func hsvToRGB(h, s, v float32) (int, int, int) {
	h = float32(math.Mod(float64(h), 360))
	if h < 0 {
		h += 360
	}
	s, v = clamp01(s), clamp01(v)
	c := v * s
	x := c * (1 - float32(math.Abs(math.Mod(float64(h/60), 2)-1)))
	m := v - c
	var r, g, b float32
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return int((r + m) * 255), int((g + m) * 255), int((b + m) * 255)
}

// rgbToHSV RGB 转 HSV（H 0..360，S/V 0..1）。
func rgbToHSV(r, g, b int) (h, s, v float32) {
	rf, gf, bf := float32(r)/255, float32(g)/255, float32(b)/255
	max := rf
	if gf > max {
		max = gf
	}
	if bf > max {
		max = bf
	}
	min := rf
	if gf < min {
		min = gf
	}
	if bf < min {
		min = bf
	}
	v = max
	d := max - min
	if max > 0 {
		s = d / max
	}
	if d == 0 {
		return 0, s, v
	}
	switch max {
	case rf:
		h = 60 * float32(math.Mod(float64((gf-bf)/d), 6))
	case gf:
		h = 60 * ((bf-rf)/d + 2)
	default:
		h = 60 * ((rf-gf)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	return h, s, v
}

// ---- 小工具 ----

// nowLocal 可被测试替换的当前时间。
var nowLocal = func() time.Time { return time.Now() }
