// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 开关组件族（多样式）：默认样式的 Switch 仍在 widgets.go，
// 这里提供按需组合的 SwitchEx + 预设样式，覆盖
// 尺寸（小/标准/大）、左侧图标、轨道内文字、渐变轨道、外发光、
// 外框胶囊（图标 + 轨道，参考「✦ 图标 + 紫色发光」样式）与禁用态。
//
// 约定：状态由调用方持有，回调里改状态并 RequestRedraw；
// 动画进度沿用 widgets.go 的 switchAnim（按 key 记录切换时刻）。

// SwitchSize 开关尺寸预设。
type SwitchSize int

const (
	SwitchNormal SwitchSize = iota // 44×24，旋钮 18（与默认 Switch 一致，零值即默认）
	SwitchSmall                    // 34×18，旋钮 14
	SwitchLarge                    // 58×30，旋钮 22
)

// geo 尺寸预设换算：轨道宽高、旋钮直径、内边距。
func (s SwitchSize) geo() (tw, th, knob, pad float32) {
	switch s {
	case SwitchSmall:
		return 34, 18, 14, 2
	case SwitchLarge:
		return 58, 30, 22, 4
	default:
		return 44, 24, 18, 3
	}
}

// labelFont 轨道内文字字号。
func (s SwitchSize) labelFont() float32 {
	switch s {
	case SwitchSmall:
		return 9.5
	case SwitchLarge:
		return 12
	default:
		return 10.5
	}
}

// SwitchStyle 开关外观配置。零值 = 默认样式（等于 widgets.go 的 Switch）。
type SwitchStyle struct {
	Size     SwitchSize
	Icon     string // 左侧图标字形（Boxed 时显示在轨道左侧）
	IconC    any    // 图标颜色，默认琥珀色
	OnLabel  string // 打开态轨道内文字
	OffLabel string // 关闭态轨道内文字
	OnFrom   any    // 打开态轨道渐变起点，默认 ColPrimary
	OnTo     any    // 打开态轨道渐变终点，默认 ColPrimary2
	OffBg    any    // 关闭态轨道底色，默认 ColTrack
	KnobC    any    // 旋钮颜色，默认白色
	Border   any    // 外框描边色（Boxed 用）
	Glow     bool   // 打开态外发光
	Boxed    bool   // 外框胶囊：图标 + 轨道
	Disabled bool   // 禁用态：半透明且不响应点击

	// ---- 自定义装饰（均可选，为空时回落到上面的字形/文字字段）----
	IconNode  *Node   // 自定义图标节点（替代 Icon 字形，渲染在外框左侧 th×th 区域内）
	OnNode    *Node   // 自定义打开态轨道内容（任意节点，替代 OnLabel）
	OffNode   *Node   // 自定义关闭态轨道内容（任意节点，替代 OffLabel）
	LabelW    float32 // 轨道内容区宽度（0 = 按 OnLabel/OffLabel 文字宽度自动）
	LabelSize float32 // 轨道文字字号（0 = 按尺寸预设）
	OnTextC   any     // 打开态文字颜色，默认 #ffffff
	OffTextC  any     // 关闭态文字颜色，默认 #e8ecff
}

// SwitchStyleDefault 默认样式（与 Switch 视觉一致）。
func SwitchStyleDefault() SwitchStyle { return SwitchStyle{} }

// SwitchStyleSparkle 「✦ 图标 + 紫色渐变 + 发光 + 外框」样式。
func SwitchStyleSparkle() SwitchStyle {
	return SwitchStyle{
		Size:   SwitchLarge,
		Icon:   "✦",
		IconC:  "#fbbf24",
		OnFrom: "#7c3aed",
		OnTo:   "#3b82f6",
		Boxed:  true,
		Glow:   true,
		Border: "rgba(139,92,246,0.55)",
	}
}

// SwitchStyleLabeled 轨道内 ON/OFF 文字样式。
func SwitchStyleLabeled(onLabel, offLabel string) SwitchStyle {
	return SwitchStyle{
		Size:     SwitchNormal,
		OnLabel:  onLabel,
		OffLabel: offLabel,
		OnFrom:   "#10b981",
		OnTo:     "#34d399",
		Glow:     true,
	}
}

// SwitchStyleGradient 纯渐变 + 发光样式。
func SwitchStyleGradient(from, to any) SwitchStyle {
	return SwitchStyle{Size: SwitchNormal, OnFrom: from, OnTo: to, Glow: true}
}

// ---- 自定义能力（值拷贝返回新样式，便于链式构建） ----

// WithIcon 自定义图标字形与颜色。
func (st SwitchStyle) WithIcon(glyph string, color any) SwitchStyle {
	st.Icon, st.IconC = glyph, color
	return st
}

// WithIconNode 用任意节点作为图标（自动开启外框胶囊），节点在 th×th 区域内居中。
func (st SwitchStyle) WithIconNode(n *Node) SwitchStyle {
	st.IconNode, st.Boxed = n, true
	return st
}

// WithLabels 自定义开 / 关文案（轨道内文字，自动按文字宽度撑开轨道）。
func (st SwitchStyle) WithLabels(onLabel, offLabel string) SwitchStyle {
	st.OnLabel, st.OffLabel = onLabel, offLabel
	return st
}

// WithLabelNodes 用任意节点作为开 / 关轨道内容（宽度由 WithLabelWidth 指定）。
func (st SwitchStyle) WithLabelNodes(on, off *Node) SwitchStyle {
	st.OnNode, st.OffNode = on, off
	return st
}

// WithLabelStyle 设置轨道文字颜色与字号（size <= 0 保持尺寸预设字号）。
func (st SwitchStyle) WithLabelStyle(onColor, offColor any, size float32) SwitchStyle {
	st.OnTextC, st.OffTextC, st.LabelSize = onColor, offColor, size
	return st
}

// WithLabelWidth 指定轨道内容区宽度（自定义节点建议显式指定，文字默认自动测量）。
func (st SwitchStyle) WithLabelWidth(w float32) SwitchStyle {
	st.LabelW = w
	return st
}

// SwitchEx 通用开关：按 SwitchStyle 渲染，切换时 160ms 缓动过渡。
func SwitchEx(key string, on bool, st SwitchStyle, onChange func(bool)) *Node {
	tw, th, knob, pad := st.Size.geo()
	font := st.LabelSize
	if font <= 0 {
		font = st.Size.labelFont()
	}

	// 过渡进度：t=1 表示动画结束
	t := float32(1)
	if start, ok := switchAnim[key]; ok {
		t = (AnimTime() - start) / 0.16
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

	// 颜色默认值 + 开关态按进度插值（关闭态 → 打开态）
	offBg, onFrom, onTo, knobC := st.OffBg, st.OnFrom, st.OnTo, st.KnobC
	if offBg == nil {
		offBg = ColTrack
	}
	if onFrom == nil {
		onFrom = ColPrimary
	}
	if onTo == nil {
		onTo = ColPrimary2
	}
	if knobC == nil {
		knobC = "#ffffff"
	}
	offP := C(offBg)
	cFrom := Blend(offP, C(onFrom), k)
	cTo := Blend(offP, C(onTo), k)

	// 轨道内容区宽度：自定义节点用 LabelW，文字按实测宽度；再按需撑开轨道避免旋钮压字
	hasLabel := st.OnLabel != "" || st.OffLabel != "" || st.OnNode != nil || st.OffNode != nil
	lw := st.LabelW
	if lw <= 0 {
		lw = TextWidth(st.OnLabel, font, 600)
		if w := TextWidth(st.OffLabel, font, 600); w > lw {
			lw = w
		}
		if lw <= 0 && (st.OnNode != nil || st.OffNode != nil) {
			lw = knob // 自定义节点未给宽度时的最小保留宽度
		}
	}
	if hasLabel {
		if need := lw + knob + pad*4; need > tw {
			tw = need
		}
	}

	knobX := pad + k*(tw-knob-pad*2)
	cells := []*Node{
		Box().St(St().AbsFill().R(th/2).BgGrad(cFrom, cTo, 90).Stroke(1, Blend(offP, C(onTo), k*0.6))),
	}
	// 关闭态内容靠右、打开态内容靠左（旋钮让位），两段内容互相淡入淡出；
	// 文字与自定义节点（OnNode / OffNode）二选一，节点优先。
	onC, offC := st.OnTextC, st.OffTextC
	if onC == nil {
		onC = "#ffffff"
	}
	if offC == nil {
		offC = "#e8ecff"
	}
	if st.OffNode != nil {
		cells = append(cells, switchLabelArea(st.OffNode, lw, th, (1-k)*0.9, AlignEnd).
			St(St().AbsTR(pad+1, 0)))
	} else if st.OffLabel != "" {
		cells = append(cells, Text(st.OffLabel).St(St().Text(font, 600, offC).H(th).
			CenterYOn().AbsTR(pad+1, 0).Fade((1-k)*0.9)))
	}
	if st.OnNode != nil {
		cells = append(cells, switchLabelArea(st.OnNode, lw, th, k*0.95, AlignStart).
			St(St().AbsTL(pad+1, 0)))
	} else if st.OnLabel != "" {
		cells = append(cells, Text(st.OnLabel).St(St().Text(font, 600, onC).H(th).
			CenterYOn().AbsTL(pad+1, 0).Fade(k*0.95)))
	}
	cells = append(cells, Box().St(St().W(knob).H(knob).R(knob/2).BgC(knobC).
		Shadow(10, 2, 0, "rgba(0,0,0,0.45)").AbsTL(knobX, (th-knob)/2)))
	track := Stack(cells...).St(St().W(tw).H(th))
	if st.Glow && k > 0.05 {
		track.St(St().Shadow(16*k, 0, 0, C(onTo).WithAlpha(0.45*k)))
	}

	n := track
	if st.Boxed {
		// 图标：自定义节点优先，其次 Icon 字形（可为空 = 不带图标）
		var icon *Node
		if st.IconNode != nil {
			icon = switchIconArea(st.IconNode, th, 0.5+0.5*k)
		} else if st.Icon != "" {
			iconC := st.IconC
			if iconC == nil {
				iconC = "#fbbf24"
			}
			icon = switchGlyph(st.Icon, th*0.66, th, iconC, 0.5+0.5*k)
		}
		border := st.Border
		if border == nil {
			border = ColStroke
		}
		padOut := float32(6)
		cells := make([]*Node, 0, 2)
		if icon != nil {
			cells = append(cells, icon)
		}
		cells = append(cells, track)
		n = Row(cells...).St(St().H(th+padOut*2).PadXY(padOut+4, padOut).GapOf(7).
			R((th+padOut*2)/2).BgC(ColBgSoft).Stroke(1, border)).AlignTo(AlignCenter)
	}

	if st.Disabled {
		n.St(St().Fade(0.42).NoFocusRing())
		return n.K(key)
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

// switchGlyph 在固定尺寸方框内光学垂直居中的图标字形。
// CenterY 按行高居中，而行高含降部留白，符号字形（✦ / ☾ 等）会整体偏上；
// 这里改用字形位图包围盒（TextGlyphBounds）算偏移，真正对齐视觉中心。
func switchGlyph(glyph string, size, box float32, color any, alpha float32) *Node {
	dy := (box - size) / 2 // 兜底：无应用上下文（无窗口布局）时按字号估算
	if top, h := TextGlyphBounds(glyph, size, 700); h > 0 {
		// 字形顶部在行顶下方 top，要让它落到 (box-h)/2 处：
		// 负偏移表示向上，不能用 AbsTL（布局期会当越界），改为绘制期 Translate。
		dy = (box-h)/2 - top
	}
	return Stack(
		Text(glyph).St(St().Text(size, 700, color).W(box).TAlign(AlignCenter).Translate(0, dy)),
	).St(St().W(box).H(box).Fade(alpha))
}

// switchBoxCenter 在固定尺寸方框内居中任意节点；纯文本按字形包围盒做光学居中，
// 避免 ✓ / ✕ / ★ 这类符号因行高降部留白而整体偏下（align 控制水平对齐）。
func switchBoxCenter(n *Node, w, h, alpha float32, align Align) *Node {
	if n == nil {
		return Stack().St(St().W(w).H(h).Fade(alpha))
	}
	if n.Kind == KText && len(n.Children) == 0 {
		dy := float32(0)
		if top, gh := TextGlyphBounds(n.Text, n.Style.FontSize, n.Style.FontWeight); gh > 0 {
			dy = (h-gh)/2 - top
		}
		return Stack(n.St(St().W(w).H(h).TAlign(align).Translate(0, dy))).
			St(St().W(w).H(h).Fade(alpha))
	}
	return Stack(Center(n)).St(St().W(w).H(h).Fade(alpha))
}

// switchIconArea 自定义图标区域：th×th 方框，内容水平垂直居中，随切换进度淡入淡出。
func switchIconArea(n *Node, box, alpha float32) *Node {
	return switchBoxCenter(n, box, box, alpha, AlignCenter)
}

// switchLabelArea 轨道内自定义内容区：宽 w 高 h，水平按 align 对齐。
func switchLabelArea(n *Node, w, h, alpha float32, align Align) *Node {
	return switchBoxCenter(n, w, h, alpha, align)
}

// ---- 预设封装（常用样式一行调用） ----

// SwitchDefault 默认样式的开关（与 Switch 一致，便于统一走 SwitchEx 体系）。
func SwitchDefault(key string, on bool, onChange func(bool)) *Node {
	return SwitchEx(key, on, SwitchStyleDefault(), onChange)
}

// SwitchSized 按尺寸预设渲染的默认样式开关。
func SwitchSized(key string, size SwitchSize, on bool, onChange func(bool)) *Node {
	st := SwitchStyleDefault()
	st.Size = size
	return SwitchEx(key, on, st, onChange)
}

// SwitchIcon 左侧带图标的胶囊开关（图标 + 轨道）。
func SwitchIcon(key, icon string, on bool, onChange func(bool)) *Node {
	st := SwitchStyleDefault()
	st.Icon, st.Boxed, st.Size = icon, true, SwitchLarge
	return SwitchEx(key, on, st, onChange)
}

// SwitchSparkle 「✦ 紫色发光」胶囊开关（SwitchStyleSparkle 的一行封装）。
func SwitchSparkle(key string, on bool, onChange func(bool)) *Node {
	return SwitchEx(key, on, SwitchStyleSparkle(), onChange)
}

// SwitchLabeled 轨道内带 ON / OFF 文字的开关。
func SwitchLabeled(key, onLabel, offLabel string, on bool, onChange func(bool)) *Node {
	return SwitchEx(key, on, SwitchStyleLabeled(onLabel, offLabel), onChange)
}

// SwitchGradient 渐变发光开关（自定义起止色）。
func SwitchGradient(key string, from, to any, on bool, onChange func(bool)) *Node {
	return SwitchEx(key, on, SwitchStyleGradient(from, to), onChange)
}

// SwitchDisabled 禁用态开关（半透明、不可点击）。
func SwitchDisabled(key string, on bool) *Node {
	st := SwitchStyleDefault()
	st.Disabled = true
	return SwitchEx(key, on, st, nil)
}
