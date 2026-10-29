package ui

import (
	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// Kind 节点类型。
type Kind uint8

const (
	KBox Kind = iota
	KText
	KButton
	KScroll
	KImage
	KStack
	KGrid
	KWrap
	KMasonry
	// KVideo 视频画面（由 Player 提供当前帧，绘制层逐帧上传纹理）。
	KVideo
	// KWebView 内嵌浏览器画面（系统 WebView2 子窗口，绘制层只负责布局与占位）。
	KWebView
	// KArrow 子菜单指示三角（矢量纹理绘制，不依赖字体字形）。
	KArrow
	// KChart 图表（折线 / 面积 / 柱状 / 饼 / 环形 / 雷达 / 散点）。
	KChart
)

// Node 声明式节点（每帧重建，交互状态按 Key 继承）。
type Node struct {
	Kind     Kind
	Key      string
	Style    Style
	Dir      Direction
	Align    Align
	Justify  Align
	Text     string
	Children []*Node
	// Src 为图片路径（Kind == KImage 时生效）。
	Src string
	// Video 视频播放器状态（Kind == KVideo 时生效）。
	Video   *Player
	// WebView 内嵌浏览器状态（Kind == KWebView 时生效）。
	WebView *WebView
	// Chart 图表数据与缓存（Kind == KChart 时生效）。
	Chart *Chart
	OnClick func()
	// OnClickAt 带点击坐标（相对节点左上角）的点击回调，滑块等需要。
	OnClickAt func(x, y float32)
	// OnHover 悬停变化回调。
	OnHover   func(bool)
	// OnPress 鼠标按下即触发（在按下后的第一帧分发一次，早于 OnDrag/OnClick）。
	// 自绘标题栏拖动、菜单按下等需要“按下就响应”的场景用它。
	OnPress   func()
	OnScroll  func(dy float32)
	// OnChar / OnKey 仅在节点获得键盘焦点时收到事件。
	OnChar func(rune)
	OnKey  func(uint32)
	// OnDrag 按下并拖动时回调（坐标相对节点左上角）；Disabled 禁用点击与拖动。
	OnDrag func(x, y float32)
	// OnDragEnd 拖动结束（松手）时回调，对应 gio Clickable.Pressed() 由 true→false；
	// 拖动类组件据此提交最终值，不必靠超时猜测松手。
	OnDragEnd func()
	// OnDrop 文件拖入回调：拖入落点命中本节点区域时触发，paths 为拖入的文件/文件夹路径。
	// 与 OnClick 一样沿祖先冒泡，未设置 OnDrop 的子节点由最近的祖先接管。
	OnDrop func(paths []string)
	Disabled bool

	// 运行时状态
	Hover, Pressed, Focus bool
	ScrollY               float32
	// wrapW 测量阶段的折行宽度：WrapOn 文本由布局写入，绘制阶段复用同一宽度，
	// 保证「测量行数」与「实际绘制行数」一致。
	wrapW    float32
	ContentH float32
	Frame                 geom.Rect
	Parent                *Node

	tag string
}

// Build 用于封装局部 view。
type Build func() *Node

// newBox 创建带默认样式的容器（Opacity 默认 1，否则会被绘制层跳过）。
func newBox(dir Direction, align Align) *Node {
	return &Node{Kind: KBox, Dir: dir, Align: align, Style: Style{Opacity: 1}}
}

// Box 全能容器。
func Box(children ...*Node) *Node {
	n := newBox(Vertical, AlignStretch)
	n.Children = children
	return n
}

// Column 纵向弹性容器。
func Column(children ...*Node) *Node {
	n := newBox(Vertical, AlignStretch)
	n.Children = children
	return n
}

// Row 横向弹性容器（交叉轴默认拉伸，与 CSS flex align-items:stretch 一致）。
func Row(children ...*Node) *Node {
	n := newBox(Horizontal, AlignStretch)
	n.Children = children
	return n
}

// Center 居中容器。
func Center(child *Node) *Node {
	n := newBox(Vertical, AlignCenter)
	n.Justify = AlignCenter
	n.Children = []*Node{child}
	return n
}

// Spacer 可伸缩空白。
func Spacer() *Node {
	n := newBox(Vertical, AlignStretch)
	n.Style.Grow = 1
	return n
}

// Text 文本节点。
func Text(s string) *Node {
	return &Node{Kind: KText, Text: s, Style: Style{Opacity: 1}}
}

// Rich 与 Text 相同，语义别名。
func Rich(s string) *Node { return Text(s) }

// Image 图片节点（PNG/JPEG/GIF/BMP，尺寸默认取原图像素）。
func Image(path string) *Node {
	return &Node{Kind: KImage, Src: path, Style: Style{Opacity: 1}}
}

// ImageSized 指定显示尺寸的图片节点。
func ImageSized(path string, w, h float32) *Node {
	return Image(path).St(St().W(w).H(h))
}

// Video 视频画面节点（尺寸默认取画面像素，可用 Style 覆盖；填充方式由 Player.Fit 决定）。
// 只负责绘制，播放控制请用 VideoPlayer 组合或自行调用 Player 方法。
func Video(p *Player) *Node {
	return &Node{Kind: KVideo, Video: p, Style: Style{Opacity: 1}}
}

// VideoSized 指定显示尺寸的视频画面节点。
func VideoSized(p *Player, w, h float32) *Node {
	return Video(p).St(St().W(w).H(h))
}

// Button 按钮（可点击）。
func Button(label string, onClick func()) *Node {
	n := &Node{Kind: KButton, Text: label, OnClick: onClick, Style: Style{Opacity: 1, CenterY: true}}
	n.Style.Height = 36
	n.Style.MinW = 44
	return n
}

// Scroll 纵向滚动容器。
func Scroll(children ...*Node) *Node {
	n := newBox(Vertical, AlignStretch)
	n.Kind = KScroll
	n.Children = children
	return n
}

// Stack 叠层容器：子节点互相重叠，后声明的绘制在上层（用于开关旋钮、进度条、浮层）。
func Stack(children ...*Node) *Node {
	n := newBox(Vertical, AlignStretch)
	n.Kind = KStack
	n.Children = children
	return n
}

// Grid 网格布局：等宽列，行高取该行最高子项。
// 列数：.St(St().ColsN(3)) 固定列数；或 .St(St().ColWidth(240)) 按列宽自适应列数。
func Grid(children ...*Node) *Node {
	n := newBox(Vertical, AlignStretch)
	n.Kind = KGrid
	n.Children = children
	return n
}

// Wrap 流式换行布局（flex-wrap: wrap）：子项按声明顺序排布，一行放不下自动折行。
func Wrap(children ...*Node) *Node {
	n := newBox(Horizontal, AlignStretch)
	n.Kind = KWrap
	n.Children = children
	return n
}

// Masonry 瀑布流布局：每项落入当前最矮的一列（列数同 Grid）。
func Masonry(children ...*Node) *Node {
	n := newBox(Vertical, AlignStretch)
	n.Kind = KMasonry
	n.Children = children
	return n
}

// EqualRow 横向等分栏：每个子项占等宽（Grow=1）。
func EqualRow(children ...*Node) *Node {
	for _, c := range children {
		if c != nil && c.Style.Grow == 0 {
			c.Style.Grow = 1
		}
	}
	n := newBox(Horizontal, AlignStretch)
	n.Children = children
	return n
}

// EqualCol 纵向等分栏：每个子项占等高（Grow=1）。
func EqualCol(children ...*Node) *Node {
	for _, c := range children {
		if c != nil && c.Style.Grow == 0 {
			c.Style.Grow = 1
		}
	}
	n := newBox(Vertical, AlignStretch)
	n.Children = children
	return n
}

// Pick 响应式选择：按视口宽度断点（<640 / <960 / <1280 / 其余）执行对应构造函数。
// 传 nil 时回退到前一个分支。
func Pick(xs, sm, md, lg func() *Node) *Node {
	w := ViewportW()
	fn := lg
	switch {
	case w < 640:
		fn = xs
	case w < 960:
		fn = sm
	case w < 1280:
		fn = md
	}
	for fn == nil {
		if lg != nil {
			fn = lg
			break
		}
		if md != nil {
			fn = md
			break
		}
		if sm != nil {
			fn = sm
			break
		}
		if xs != nil {
			fn = xs
		}
		break
	}
	if fn == nil {
		return nil
	}
	return fn()
}

// Abs 绝对定位容器（脱离 flex 流，尺寸由 Style 决定）。
func Abs(child *Node) *Node {
	n := newBox(Vertical, AlignStretch)
	n.Children = []*Node{child}
	return n
}

func (n *Node) St(s *Style) *Node {
	if s != nil {
		n.Style = n.Style.merge(*s)
	}
	return n
}

func (n *Node) Add(c ...*Node) *Node {
	n.Children = append(n.Children, c...)
	return n
}

func (n *Node) K(key string) *Node { n.Key = key; return n }

func (n *Node) Grow(v float32) *Node { n.Style.Grow = v; return n }

func (n *Node) GapElastic(v float32) *Node { n.Style.Gap = v; return n }

func (n *Node) AlignTo(a Align) *Node { n.Align = a; return n }

func (n *Node) JustifyTo(a Align) *Node { n.Justify = a; return n }

func (n *Node) Horizontal() *Node { n.Dir = Horizontal; return n }

func (n *Node) Tap(fn func()) *Node { n.OnClick = fn; return n }

// TapAt 带相对坐标的点击回调（滑块、进度跳转等）。
func (n *Node) TapAt(fn func(x, y float32)) *Node { n.OnClickAt = fn; return n }

// OnCharEvt 键盘字符输入回调（需配合 .St(St().FocusOn())）。
// OnCharEvt 键盘字符输入回调（需配合 .St(St().FocusOn())）。
func (n *Node) OnCharEvt(fn func(rune)) *Node { n.OnChar = fn; return n }

func (n *Node) OnKeyEvt(fn func(uint32)) *Node { n.OnKey = fn; return n }

// DragEvt 按下拖动回调（需设置 Key）。
func (n *Node) DragEvt(fn func(x, y float32)) *Node { n.OnDrag = fn; return n }

// DragEndEvt 拖动结束（松手）回调（需设置 Key 并配合 DragEvt）。
func (n *Node) DragEndEvt(fn func()) *Node { n.OnDragEnd = fn; return n }

// DropEvt 文件拖入回调（paths 为拖入的文件/文件夹路径，落点在本节点内时触发）。
func (n *Node) DropEvt(fn func(paths []string)) *Node { n.OnDrop = fn; return n }

// DisabledOn 禁用控件（不响应点击/拖动，外观由样式决定）。
func (n *Node) DisabledOn(v bool) *Node { n.Disabled = v; return n }

func (n *Node) OnHoverEvt(fn func(bool)) *Node { n.OnHover = fn; return n }

// OnPressEvt 鼠标按下回调（需设置 Key，可用于自绘标题栏拖动）。
func (n *Node) OnPressEvt(fn func()) *Node { n.OnPress = fn; return n }

func (n *Node) ScrollEvt(fn func(float32)) *Node { n.OnScroll = fn; return n }

// If 条件构造。
func If(cond bool, fn func() *Node) *Node {
	if cond {
		return fn()
	}
	return nil
}

// Each 列表构造。
func Each[T any](items []T, fn func(T, int) *Node) []*Node {
	out := make([]*Node, 0, len(items))
	for i, it := range items {
		if n := fn(it, i); n != nil {
			out = append(out, n)
		}
	}
	return out
}

// Instrs 把渲染指令收集为 slice（供后端 Draw 使用）。
type painter struct {
	r *gpu.Renderer
}
