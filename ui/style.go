// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// Align 主轴/交叉轴对齐。
type Align uint8

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
	// AlignSpaceBetween / Around / Evenly 仅用于主轴分配（JustifyTo）。
	AlignSpaceBetween
	AlignSpaceAround
	AlignSpaceEvenly
)

// Direction 弹性方向。
type Direction uint8

const (
	Vertical Direction = iota
	Horizontal
)

// Style 声明式样式（类似 CSS 的子集）。
type Style struct {
	Width, Height       float32 // 0 = auto
	WidthPct, HeightPct float32 // >0 = 占父容器百分比
	MinW, MaxW          float32
	MinH, MaxH          float32
	Pad                 geom.Insets
	Margin              geom.Insets
	Gap                 float32

	Grow      float32
	Basis     float32 // 主轴基准尺寸（配合 Grow 实现等宽/等高分栏，类似 flex-basis）
	AlignSelf Align

	// Cols / ColW 网格与瀑布流列参数：Cols>0 固定列数；ColW>0 固定列宽（自动算列数）。
	Cols int
	ColW float32

	Bg      gpu.Paint
	BgTo    gpu.Paint // 有 alpha > 0 时启用线性渐变
	BgAngle float32

	Radius      [4]float32
	Border      float32
	BorderColor gpu.Paint

	ShadowBlur   float32
	ShadowY      float32
	ShadowSpread float32
	ShadowColor  gpu.Paint

	FontSize   float32
	FontWeight int32
	// FontFamily 字体家族名（空 = 默认 Microsoft YaHei UI）。
	FontFamily string
	Color      gpu.Paint
	LineHeight float32
	TextAlign  Align
	// CenterY 为 true 时文字在自身高度内垂直居中（按钮默认开启）。
	CenterY bool
	Wrap    bool

	Opacity    float32
	Clip       bool
	Scrollable bool

	// 绝对定位：相对父容器内边距区域定位（Stack 叠层与浮层用）
	PosAbs                 bool
	AbsL, AbsT, AbsR, AbsB float32
	HasL, HasT, HasR, HasB bool
	AbsCenterX, AbsCenterY bool

	// Focusable 可获取键盘焦点（输入框等）；Animated 每帧重绘（加载动画/骨架屏）
	Focusable bool
	Animated  bool

	// BackdropBlur > 0 时对该节点区域应用毛玻璃背景模糊（离屏 FBO + Kawase）
	BackdropBlur float32
	// BackdropTint 毛玻璃上的叠加色
	BackdropTint gpu.Paint

	// Scrollbar 滚动条可见（默认可见）
	Scrollbar bool
}

// St 新建样式。
func St() *Style { return &Style{Opacity: 1} }

func (s *Style) W(v float32) *Style         { s.Width = v; return s }
func (s *Style) H(v float32) *Style         { s.Height = v; return s }
func (s *Style) Wp(v float32) *Style        { s.WidthPct = v; return s }
func (s *Style) Hp(v float32) *Style        { s.HeightPct = v; return s }
func (s *Style) MinWidth(v float32) *Style  { s.MinW = v; return s }
func (s *Style) MaxWidth(v float32) *Style  { s.MaxW = v; return s }
func (s *Style) MinHeight(v float32) *Style { s.MinH = v; return s }
func (s *Style) MaxHeight(v float32) *Style { s.MaxH = v; return s }

func (s *Style) Padding(v float32) *Style          { s.Pad = geom.All(v); return s }
func (s *Style) PadXY(h, v float32) *Style         { s.Pad = geom.Sym(h, v); return s }
func (s *Style) PadOnly(t, r, b, l float32) *Style { s.Pad = geom.Only(t, r, b, l); return s }
func (s *Style) Mar(v float32) *Style              { s.Margin = geom.All(v); return s }
func (s *Style) MarXY(h, v float32) *Style         { s.Margin = geom.Sym(h, v); return s }
func (s *Style) MarOnly(t, r, b, l float32) *Style { s.Margin = geom.Only(t, r, b, l); return s }
func (s *Style) GapOf(v float32) *Style            { s.Gap = v; return s }

func (s *Style) GrowTo(v float32) *Style    { s.Grow = v; return s }
func (s *Style) FlexBasis(v float32) *Style { s.Basis = v; return s }
func (s *Style) Self(a Align) *Style        { s.AlignSelf = a; return s }

// ColsN 设置网格/瀑布流列数（0 = 自动）。
func (s *Style) ColsN(n int) *Style { s.Cols = n; return s }

// ColWidth 设置固定列宽（>0 且未指定列数时按可用宽度自动算列数）。
func (s *Style) ColWidth(v float32) *Style { s.ColW = v; return s }


func (s *Style) BgC(v any) *Style { s.Bg = C(v); return s }
func (s *Style) BgGrad(from, to any, angle float32) *Style {
	s.Bg = C(from)
	s.BgTo = C(to)
	s.BgAngle = angle
	return s
}

func (s *Style) R(v float32) *Style {
	s.Radius = [4]float32{v, v, v, v}
	return s
}
func (s *Style) Rx(tl, tr, br, bl float32) *Style {
	s.Radius = [4]float32{tl, tr, br, bl}
	return s
}
func (s *Style) Stroke(w float32, c any) *Style {
	s.Border = w
	s.BorderColor = C(c)
	return s
}
func (s *Style) Shadow(blur, y, spread float32, c any) *Style {
	s.ShadowBlur, s.ShadowY, s.ShadowSpread = blur, y, spread
	s.ShadowColor = C(c)
	return s
}
func (s *Style) Text(size float32, weight int32, c any) *Style {
	s.FontSize, s.FontWeight, s.Color = size, weight, C(c)
	return s
}
func (s *Style) Lh(v float32) *Style   { s.LineHeight = v; return s }

// Font 指定字体家族（需要在系统字体目录可解析）。
func (s *Style) Font(family string) *Style { s.FontFamily = family; return s }

// Backdrop 毛玻璃背景模糊（blur 为逻辑像素半径，tint 为叠加色）。
func (s *Style) Backdrop(blur float32, tint any) *Style {
	s.BackdropBlur = blur
	if tint != nil {
		s.BackdropTint = C(tint)
	}
	return s
}
func (s *Style) TAlign(a Align) *Style { s.TextAlign = a; return s }
func (s *Style) WrapOn() *Style        { s.Wrap = true; return s }

// CenterYOn 文字在自身高度内垂直居中。
func (s *Style) CenterYOn() *Style     { s.CenterY = true; return s }
func (s *Style) Fade(v float32) *Style { s.Opacity = v; return s }
func (s *Style) Clipped() *Style       { s.Clip = true; return s }
func (s *Style) ScrollY() *Style       { s.Scrollable = true; return s }

// AbsFill 铺满父容器内边距区域。
func (s *Style) AbsFill() *Style {
	s.PosAbs = true
	s.HasL, s.HasT, s.HasR, s.HasB = true, true, true, true
	s.AbsL, s.AbsT, s.AbsR, s.AbsB = 0, 0, 0, 0
	return s
}

// AbsTL 左上角绝对定位。
func (s *Style) AbsTL(l, t float32) *Style {
	s.PosAbs = true
	s.HasL, s.HasT = true, true
	s.AbsL, s.AbsT = l, t
	return s
}

// AbsTR 右上角绝对定位。
func (s *Style) AbsTR(r, t float32) *Style {
	s.PosAbs = true
	s.HasR, s.HasT = true, true
	s.AbsR, s.AbsT = r, t
	return s
}

// AbsBL 左下角绝对定位。
func (s *Style) AbsBL(l, b float32) *Style {
	s.PosAbs = true
	s.HasL, s.HasB = true, true
	s.AbsL, s.AbsB = l, b
	return s
}

// AbsBR 右下角绝对定位。
func (s *Style) AbsBR(r, b float32) *Style {
	s.PosAbs = true
	s.HasR, s.HasB = true, true
	s.AbsR, s.AbsB = r, b
	return s
}

// AbsCenter 在父容器内水平+垂直居中。
func (s *Style) AbsCenter() *Style {
	s.PosAbs = true
	s.AbsCenterX, s.AbsCenterY = true, true
	return s
}

// AbsBottom 水平居中、贴底定位（Toast 等）。
func (s *Style) AbsBottom(b float32) *Style {
	s.PosAbs = true
	s.HasB, s.AbsB = true, b
	s.AbsCenterX = true
	return s
}

// FocusOn 允许获取键盘焦点。
func (s *Style) FocusOn() *Style { s.Focusable = true; return s }

// AnimOn 标记为需要逐帧重绘。
func (s *Style) AnimOn() *Style { s.Animated = true; return s }

// merge 覆盖默认样式（零值字段不覆盖）。
func (s Style) merge(o Style) Style {
	if o.Width != 0 {
		s.Width = o.Width
	}
	if o.Height != 0 {
		s.Height = o.Height
	}
	if o.WidthPct != 0 {
		s.WidthPct = o.WidthPct
	}
	if o.HeightPct != 0 {
		s.HeightPct = o.HeightPct
	}
	if o.MinW != 0 {
		s.MinW = o.MinW
	}
	if o.MaxW != 0 {
		s.MaxW = o.MaxW
	}
	if o.MinH != 0 {
		s.MinH = o.MinH
	}
	if o.MaxH != 0 {
		s.MaxH = o.MaxH
	}
	if o.Pad != (geom.Insets{}) {
		s.Pad = o.Pad
	}
	if o.Margin != (geom.Insets{}) {
		s.Margin = o.Margin
	}
	if o.Gap != 0 {
		s.Gap = o.Gap
	}
	if o.Grow != 0 {
		s.Grow = o.Grow
	}
	if o.Basis != 0 {
		s.Basis = o.Basis
	}
	if o.Cols != 0 {
		s.Cols = o.Cols
	}
	if o.ColW != 0 {
		s.ColW = o.ColW
	}
	if o.Bg.A > 0 {
		s.Bg = o.Bg
	}
	if o.BgTo.A > 0 {
		s.BgTo = o.BgTo
		s.BgAngle = o.BgAngle
	}
	if o.Radius != [4]float32{} {
		s.Radius = o.Radius
	}
	if o.Border != 0 {
		s.Border = o.Border
		s.BorderColor = o.BorderColor
	}
	if o.ShadowColor.A > 0 {
		s.ShadowBlur, s.ShadowY, s.ShadowSpread, s.ShadowColor = o.ShadowBlur, o.ShadowY, o.ShadowSpread, o.ShadowColor
	}
	if o.FontSize != 0 {
		s.FontSize = o.FontSize
	}
	if o.FontWeight != 0 {
		s.FontWeight = o.FontWeight
	}
	if o.FontFamily != "" {
		s.FontFamily = o.FontFamily
	}
	if o.BackdropBlur != 0 {
		s.BackdropBlur = o.BackdropBlur
	}
	if o.BackdropTint.A > 0 {
		s.BackdropTint = o.BackdropTint
	}
	if o.Scrollbar {
		s.Scrollbar = true
	}
	if o.Color.A > 0 {
		s.Color = o.Color
	}
	if o.LineHeight != 0 {
		s.LineHeight = o.LineHeight
	}
	if o.Opacity != 0 {
		s.Opacity = o.Opacity
	}
	if o.Clip {
		s.Clip = true
	}
	if o.Scrollable {
		s.Scrollable = true
	}
	if o.Wrap {
		s.Wrap = true
	}
	if o.CenterY {
		s.CenterY = true
	}
	if o.AlignSelf != AlignStart {
		s.AlignSelf = o.AlignSelf
	}
	if o.TextAlign != AlignStart {
		s.TextAlign = o.TextAlign
	}
	if o.PosAbs {
		s.PosAbs = true
	}
	if o.HasL {
		s.HasL, s.AbsL = true, o.AbsL
	}
	if o.HasT {
		s.HasT, s.AbsT = true, o.AbsT
	}
	if o.HasR {
		s.HasR, s.AbsR = true, o.AbsR
	}
	if o.HasB {
		s.HasB, s.AbsB = true, o.AbsB
	}
	if o.AbsCenterX {
		s.AbsCenterX = true
	}
	if o.AbsCenterY {
		s.AbsCenterY = true
	}
	if o.Focusable {
		s.Focusable = true
	}
	if o.Animated {
		s.Animated = true
	}
	return s
}
