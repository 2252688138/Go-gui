// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"gogui/internal/geom"
)

// 数据展示组件补充：Carousel / Timeline / Steps / Segmented / Descriptions /
// Statistic / Calendar / Watermark / ImagePreview / AvatarGroup / Tour。

// ---- Carousel ----

// CarouselOptions 轮播参数。
type CarouselOptions struct {
	Height   float32
	Interval float32 // 自动播放间隔（秒，默认 4）
	Autoplay bool
	Loop     bool
	Arrows   bool
	Dots     bool
}

type carouselState struct {
	idx  int
	last float32

	// 自动播放用「单次定时器 + 一帧重绘」驱动，而不是每帧 RequestRedraw。
	// 旧实现每个轮播实例在每次构建时都请求重绘，导致整个页面常驻满帧重建。
	mu    sync.Mutex
	timer *time.Timer
	tick  bool
}

var carousels = map[string]*carouselState{}

// arm 安排下一次自动播放：单次定时器，触发后只请求一帧重绘。
// 比“每次构建都 RequestRedraw”少一整棵树的重复构建。
func (st *carouselState) arm(interval float32) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.timer != nil {
		return
	}
	d := time.Duration(float64(interval) * float64(time.Second))
	if d < 50*time.Millisecond {
		d = 50 * time.Millisecond
	}
	st.timer = time.AfterFunc(d, func() {
		defer recoverReport("carousel-timer")
		st.mu.Lock()
		st.timer = nil
		st.tick = true
		st.mu.Unlock()
		RequestRedraw()
	})
}

// takeTick 取走一次到期的自动播放信号。
func (st *carouselState) takeTick() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.tick {
		return false
	}
	st.tick = false
	return true
}

// Carousel 轮播图：指示器、左右箭头、自动播放、循环。
func Carousel(key string, slides []*Node, opts CarouselOptions) *Node {
	if opts.Height <= 0 {
		opts.Height = 220
	}
	if opts.Interval <= 0 {
		opts.Interval = 4
	}
	st := carousels[key]
	if st == nil {
		st = &carouselState{last: AnimTime()}
		carousels[key] = st
	}
	n := len(slides)
	if n == 0 {
		return EmptyState("", "暂无内容", "", "", nil).St(St().H(opts.Height))
	}
	next := func(d int) {
		st.idx += d
		if opts.Loop {
			st.idx = (st.idx%n + n) % n
		} else if st.idx < 0 {
			st.idx = 0
		} else if st.idx >= n {
			st.idx = n - 1
		}
		st.last = AnimTime()
		ResetAnim(key + "-slide")
		RequestRedraw()
	}
	if opts.Autoplay {
		if st.takeTick() {
			if opts.Loop || st.idx < n-1 {
				next(1)
			} else {
				st.last = AnimTime()
			}
		}
		st.arm(opts.Interval)
	}
	if st.idx >= n {
		st.idx = n - 1
	}
	t := TweenFrom(key+"-slide", 0, 1, 0.35, EaseOutCubic)
	active := Box(slides[st.idx]).St(St().AbsFill().Clipped().Translate((1-t)*26, 0))

	layers := []*Node{active}
	if opts.Arrows {
		prev := Text("‹").St(St().Text(26, 500, "#ffffff").W(36).H(36).R(18).
			CenterYOn().TAlign(AlignCenter).BgC(paintWithAlpha("#000000", 0.35)).
			AbsTL(10, opts.Height/2-18)).Tap(func() { next(-1) })
		nf := Text("›").St(St().Text(26, 500, "#ffffff").W(36).H(36).R(18).
			CenterYOn().TAlign(AlignCenter).BgC(paintWithAlpha("#000000", 0.35)).
			AbsTR(10, opts.Height/2-18)).Tap(func() { next(1) })
		layers = append(layers, prev, nf)
	}
	if opts.Dots {
		dots := make([]*Node, 0, n)
		for i := 0; i < n; i++ {
			idx := i
			col := paintWithAlpha("#ffffff", 0.35)
			w := float32(7)
			if idx == st.idx {
				col = C("#ffffff")
				w = 16
			}
			// 指示点没有持续动画，不能标 AnimOn：否则每帧都被判定为“动画中”，
			// 整页（含表格 / 打印预览）会被无限重绘。
			dots = append(dots, Box().St(St().W(w).H(7).R(4).BgC(col)).
				K(fmt.Sprintf("%s-dot-%d", key, i)).Tap(func() {
				st.idx = idx
				st.last = AnimTime()
				RequestRedraw()
			}))
		}
		layers = append(layers, Row(dots...).St(St().GapOf(6).AbsBottom(10)))
	}
	return Stack(layers...).St(St().Wp(100).H(opts.Height).R(14).Clipped())
}

// ---- Timeline ----

// TimelineItem 时间线节点。
type TimelineItem struct {
	Time  string
	Title string
	Sub   string
	Color string
	Body  *Node
}

// Timeline 垂直时间线。
func Timeline(items []TimelineItem) *Node {
	rows := make([]*Node, 0, len(items))
	for i, it := range items {
		color := it.Color
		if color == "" {
			color = ColPrimary
		}
		last := i == len(items)-1
		lineColor := paintWithAlpha(ColStroke, 1)
		marker := Column(
			Box().St(St().W(10).H(10).R(5).BgC(color).Stroke(2, ColBgDeep)),
		).St(St().H(10))
		if !last {
			line := Box().St(St().W(2).GrowTo(1).BgC(lineColor).MarOnly(4, 0, 0, 0))
			marker.Add(line)
		}
		body := Column(
			Row(
				Text(it.Title).St(St().Text(12.5, 600, ColText)).Grow(1),
				Text(it.Time).St(St().Text(10.5, 400, ColTextMuted)),
			).St(St().GapOf(8)).AlignTo(AlignCenter),
		)
		if it.Sub != "" {
			body.Add(Text(it.Sub).St(St().Text(11.5, 400, ColTextMuted).WrapOn()))
		}
		if it.Body != nil {
			body.Add(it.Body)
		}
		rows = append(rows, Row(marker, body).St(St().GapOf(12).PadOnly(0, 0, 14, 0)).AlignTo(AlignStart))
	}
	return Column(rows...).St(St().GapOf(0))
}

// ---- Steps ----

// StepItem 步骤。
type StepItem struct {
	Title string
	Sub   string
}

// Steps 步骤条（横向；active 为当前步，0 起）。
func Steps(items []StepItem, active int) *Node {
	cells := make([]*Node, 0, len(items)*2)
	for i, it := range items {
		color := ColTextMuted
		ring := ColStroke
		var bg any = "transparent"
		num := strconv.Itoa(i + 1)
		switch {
		case i < active:
			color, ring = ColSuccess, ColSuccess
			bg = paintWithAlpha(ColSuccess, 0.18)
			num = "✓"
		case i == active:
			color, ring = ColPrimary, ColPrimary
			bg = paintWithAlpha(ColPrimary, 0.18)
		}
		circle := Text(num).St(St().Text(11, 700, color).W(24).H(24).R(12).
			CenterYOn().TAlign(AlignCenter).BgC(bg).Stroke(1.5, ring))
		label := Column(Text(it.Title).St(St().Text(12.5, 600, ColTextDim)))
		if it.Sub != "" {
			label.Add(Text(it.Sub).St(St().Text(10.5, 400, ColTextMuted)))
		}
		cells = append(cells, Row(circle, label).St(St().GapOf(9)).AlignTo(AlignCenter))
		if i < len(items)-1 {
			col := ColStroke
			if i < active {
				col = ColSuccess
			}
			cells = append(cells, Box().St(St().H(1.5).MinWidth(24).GrowTo(1).BgC(col).
				MarXY(10, 0).Self(AlignCenter)))
		}
	}
	return Row(cells...).St(St().GapOf(4)).AlignTo(AlignCenter)
}

// ---- Segmented ----

// Segmented 分段控制器（滑动指示条的 Tabs 单选变体）。
func Segmented(keys, labels []string, active string, onPick func(string)) *Node {
	if len(keys) == 0 {
		return nil
	}
	idx := 0
	for i, k := range keys {
		if k == active {
			idx = i
		}
	}
	n := float32(len(keys))
	pad := float32(4)
	_ = n
	items := make([]*Node, 0, len(keys))
	for i, k := range keys {
		label := k
		if i < len(labels) {
			label = labels[i]
		}
		color := ColTextMuted
		weight := int32(500)
		if k == active {
			color, weight = "#ffffff", 600
		}
		id := k
		txt := Text(label).St(St().Text(12.5, weight, color).Wp(100).H(30).
			CenterYOn().TAlign(AlignCenter))
		txt.Tap(func() {
			if onPick != nil {
				onPick(id)
			}
			RequestRedraw()
		})
		items = append(items, txt)
	}
	// 指示条：每段等宽容器内放一个相对位移块
	segs := make([]*Node, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		var inner *Node
		if i == idx {
			inner = Box().St(St().AbsFill().Mar(6).R(8).BgC(ColPrimary))
		} else {
			inner = Box()
		}
		segs = append(segs, inner)
	}
	return Stack(
		Row(segs...).St(St().AbsFill().PadXY(pad, pad)),
		Row(items...).St(St().AbsFill().PadXY(pad, pad)),
	).St(St().Wp(100).H(38).R(10).BgC(ColInput).Stroke(1, ColStroke))
}

// ---- Descriptions ----

// KV 键值对。
type KV struct{ K, V string }

// Descriptions 键值详情列表（columns 列，自适应）。
func Descriptions(items []KV, columns int) *Node {
	if columns <= 0 {
		columns = 2
	}
	rows := make([]*Node, 0, (len(items)+columns-1)/columns)
	for i := 0; i < len(items); i += columns {
		cells := make([]*Node, 0, columns)
		for j := 0; j < columns; j++ {
			if i+j >= len(items) {
				cells = append(cells, Box().Grow(1))
				continue
			}
			kv := items[i+j]
			cell := Column(
				Text(kv.K).St(St().Text(10.5, 500, ColTextMuted)),
				Text(kv.V).St(St().Text(12.5, 500, ColText).WrapOn()),
			).St(St().GapOf(3)).Grow(1)
			cells = append(cells, cell)
		}
		rows = append(rows, Row(cells...).St(St().GapOf(16).PadOnly(0, 0, 14, 0)).AlignTo(AlignStart))
	}
	return Column(rows...).St(St().GapOf(0))
}

// ---- Statistic ----

// StatOptions 指标卡参数。
type StatOptions struct {
	Prefix   string
	Suffix   string
	Decimals int
	Trend    float32 // 正数上升 / 负数下降 / 0 无
	Animate  bool
}

// Statistic 指标数值（可选滚动动效 + 趋势）。
func Statistic(key, label string, value float64, opts StatOptions) *Node {
	v := value
	if opts.Animate {
		v = AnimatedNumber("stat-"+key, value, 0.8, opts.Decimals)
	}
	num := opts.Prefix + FormatNumber(v, opts.Decimals) + opts.Suffix
	col := Column(
		Text(label).St(St().Text(11.5, 500, ColTextMuted)),
		Text(num).St(St().Text(24, 700, ColText)),
	)
	if opts.Trend != 0 {
		color := ColSuccess
		glyph := "▲"
		if opts.Trend < 0 {
			color, glyph = ColDanger, "▼"
		}
		col.Add(Row(
			Text(glyph).St(St().Text(9, 600, color)),
			Text(fmt.Sprintf("%.1f%%", absF(opts.Trend))).St(St().Text(11.5, 600, color)),
		).St(St().GapOf(4)).AlignTo(AlignCenter))
	}
	return col.St(St().GapOf(4))
}

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// ---- Calendar ----

// CalendarEvent 日历事件点。
type CalendarEvent struct {
	Date  string // YYYY-MM-DD
	Label string
	Color string
}

// CalendarState 日历状态。
type CalendarState struct {
	Key string
	Y   int
	M   int
}

var calendarStates = map[string]*CalendarState{}

// NewCalendar 创建日历状态。
func NewCalendar(key string) *CalendarState {
	if s, ok := calendarStates[key]; ok {
		return s
	}
	now := nowLocal()
	s := &CalendarState{Key: key, Y: now.Year(), M: int(now.Month())}
	calendarStates[key] = s
	return s
}

// Calendar 独立日历组件（月视图 + 事件点）。
func Calendar(st *CalendarState, events []CalendarEvent, value string, onChange func(string)) *Node {
	if st == nil {
		return nil
	}
	if st.Y == 0 {
		now := nowLocal()
		st.Y, st.M = now.Year(), int(now.Month())
	}
	const cellW, cellH = float32(42), float32(38)
	daysIn := func(y, m int) int { return time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.Local).Day() }
	first := time.Date(st.Y, time.Month(st.M), 1, 0, 0, 0, 0, time.Local)
	lead := int(first.Weekday())
	curDays := daysIn(st.Y, st.M)
	prevY, prevM := st.M-1, st.Y
	if prevM < 1 {
		prevM, prevY = 12, st.Y-1
	}
	nextY, nextM := st.M+1, st.Y
	if nextM > 12 {
		nextM, nextY = 1, st.Y+1
	}
	evColor := map[string]string{}
	for _, e := range events {
		c := e.Color
		if c == "" {
			c = ColPrimary
		}
		evColor[e.Date] = c
	}
	nav := func(label string, y, m int) *Node {
		return Text(label).St(St().Text(13, 600, ColTextDim).W(28).H(26).R(6).
			CenterYOn().TAlign(AlignCenter)).Tap(func() {
			st.Y, st.M = y, m
			RequestRedraw()
		})
	}
	head := Row(
		nav("‹", prevY, prevM),
		Text(fmt.Sprintf("%d 年 %d 月", st.Y, st.M)).St(St().Text(13, 600, ColText).
			TAlign(AlignCenter)).Grow(1),
		nav("›", nextY, nextM),
	).St(St().GapOf(6)).AlignTo(AlignCenter)

	week := Row()
	for _, d := range []string{"日", "一", "二", "三", "四", "五", "六"} {
		week.Add(Text(d).St(St().Text(10.5, 500, ColTextMuted).W(cellW).H(22).
			CenterYOn().TAlign(AlignCenter)))
	}
	cells := []*Node{week}
	today := nowLocal()
	grid := Row()
	for i := 0; i < 42; i++ {
		day := i - lead + 1
		dy, dm, dd := st.Y, st.M, day
		muted := false
		if day < 1 {
			dy, dm = prevY, prevM
			dd = daysIn(prevY, prevM) + day
			muted = true
		} else if day > curDays {
			dy, dm = nextY, nextM
			dd = day - curDays
			muted = true
		}
		ds := fmt.Sprintf("%04d-%02d-%02d", dy, dm, dd)
		color := ColTextDim
		if muted {
			color = ColPlaceholder
		}
		var bg any = "transparent"
		var stroke any = "transparent"
		if dy == today.Year() && int(today.Month()) == dm && today.Day() == dd {
			bg = paintWithAlpha(ColPrimary, 0.18)
		}
		if ds == value {
			bg = ColPrimary
			color = "#ffffff"
			stroke = paintWithAlpha("#ffffff", 0.35)
		}
		cell := Stack(
			Box().St(St().AbsFill().R(8).BgC(bg).Stroke(1, stroke)),
			Text(strconv.Itoa(dd)).St(St().AbsFill().Text(11.5, 500, color).
				TAlign(AlignCenter).CenterYOn()),
		).St(St().W(cellW).H(cellH).R(8).NoFocusRing())
		if c, ok := evColor[ds]; ok {
			cell.Add(Box().St(St().W(4).H(4).R(2).BgC(c).AbsBottom(5)))
		}
		cell.K(fmt.Sprintf("%s-day-%s", st.Key, ds))
		d := ds
		cell.Tap(func() {
			if onChange != nil {
				onChange(d)
			}
			RequestRedraw()
		})
		grid.Add(cell)
		if (i+1)%7 == 0 {
			cells = append(cells, grid.St(St().GapOf(2)))
			grid = Row()
		}
	}
	return Column(append([]*Node{head}, cells...)...).St(St().GapOf(6).Padding(14).R(14).
		BgC(ColPanel).Stroke(1, ColStroke))
}

// ---- Watermark ----

// Watermark 区域水印（平铺文字，低透明度，不拦截交互）。
func Watermark(content string, children ...*Node) *Node {
	if content == "" {
		return Box(children...).St(St().Wp(100))
	}
	col := C(ColText)
	col.A = 0.06
	const stepX, stepY = float32(180), float32(110)
	layer := Stack().St(St().AbsFill().Clipped().NoFocusRing())
	// 固定 6×4 个平铺点，跟随容器裁剪
	for r := 0; r < 4; r++ {
		for c := 0; c < 6; c++ {
			layer.Add(Text(content).St(St().Text(12.5, 500, col).NoFocusRing().
				AbsTL(float32(c)*stepX+float32(r%2)*60, float32(r)*stepY+8)))
		}
	}
	return Stack(append(children, layer)...).St(St().Wp(100))
}

// ---- ImagePreview ----

// PreviewImage 预览图片。
type PreviewImage struct {
	Src   string
	Title string
}

var previewState struct {
	open   bool
	images []PreviewImage
	idx    int
	zoom   float32
}

// ImagePreviewOpen 打开通用图片灯箱。
func ImagePreviewOpen(images []PreviewImage, idx int) {
	if len(images) == 0 {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(images) {
		idx = len(images) - 1
	}
	previewState.open = true
	previewState.images = images
	previewState.idx = idx
	previewState.zoom = 1
	OpenOverlay("image-preview", func() bool {
		ImagePreviewClose()
		return true
	})
	RequestRedraw()
}

// ImagePreviewOpenFiles 用图片路径打开灯箱。
func ImagePreviewOpenFiles(paths []string, idx int) {
	imgs := make([]PreviewImage, 0, len(paths))
	for _, p := range paths {
		imgs = append(imgs, PreviewImage{Src: p})
	}
	ImagePreviewOpen(imgs, idx)
}

// ImagePreviewClose 关闭灯箱。
func ImagePreviewClose() {
	previewState.open = false
	previewState.zoom = 1
	CloseOverlay("image-preview")
	RequestRedraw()
}

// ImagePreviewLayer 灯箱浮层（根 Stack 末尾）。
func ImagePreviewLayer() *Node {
	visible, t := KeepAlive("image-preview-box", previewState.open, 0.16)
	if !visible {
		return nil
	}
	if previewState.open {
		RequestRedraw()
	}
	idx := previewState.idx
	n := len(previewState.images)
	if idx < 0 || idx >= n {
		return nil
	}
	cur := previewState.images[idx]
	next := func(d int) {
		previewState.idx = (previewState.idx + d + n) % n
		previewState.zoom = 1
		RequestRedraw()
	}
	img := Image(cur.Src).St(St().MaxWidth(ViewportW() * 0.86).MaxHeight(ViewportH() * 0.78).
		AbsCenter())
	// 缩放采用尺寸百分比（不支持负边距时的近似做法）
	if z := previewState.zoom; z > 0 && z != 1 {
		img.St(St().Wp(minF2(92*z, 92)).Hp(minF2(84*z, 84)).AbsCenter())
	}
	bar := Row(
		Text(fmt.Sprintf("%d / %d", idx+1, n)).St(St().Text(11.5, 500, "#ffffff")),
		Spacer(),
		Text("−").St(St().Text(16, 600, "#ffffff").W(32).H(32).R(8).CenterYOn().
			TAlign(AlignCenter).BgC(paintWithAlpha("#ffffff", 0.12))).Tap(func() {
			if previewState.zoom > 0.5 {
				previewState.zoom -= 0.2
			}
			RequestRedraw()
		}),
		Text("+").St(St().Text(16, 600, "#ffffff").W(32).H(32).R(8).CenterYOn().
			TAlign(AlignCenter).BgC(paintWithAlpha("#ffffff", 0.12))).Tap(func() {
			if previewState.zoom < 3 {
				previewState.zoom += 0.2
			}
			RequestRedraw()
		}),
		Text("×").St(St().Text(16, 600, "#ffffff").W(32).H(32).R(8).CenterYOn().
			TAlign(AlignCenter).BgC(paintWithAlpha("#ffffff", 0.12))).Tap(ImagePreviewClose),
	).St(St().AbsTL(20, 16).W(ViewportW() - 40).GapOf(8)).AlignTo(AlignCenter)
	nav := func(glyph string, d int, right bool) *Node {
		st := St().W(44).H(44).R(22).Text(26, 500, "#ffffff").CenterYOn().
			TAlign(AlignCenter).BgC(paintWithAlpha("#ffffff", 0.10))
		if right {
			st.AbsTR(16, ViewportH()/2-22)
		} else {
			st.AbsTL(16, ViewportH()/2-22)
		}
		return Text(glyph).St(st).Tap(func() { next(d) })
	}
	layers := []*Node{
		img,
		nav("‹", -1, false),
		nav("›", 1, true),
		bar,
	}
	if cur.Title != "" {
		layers = append(layers, Text(cur.Title).St(St().Text(12.5, 500, "#ffffff").
			AbsBottom(24)))
	}
	scrim := Box().St(St().AbsFill().BgC(paintWithAlpha("#060a12", 0.92*t))).Tap(ImagePreviewClose)
	return Stack(append([]*Node{scrim}, layers...)...).St(St().AbsFill().Fade(t))
}

func minF2(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// ---- AvatarGroup ----

// AvatarItem 头像项。
type AvatarItem struct {
	Initial string
	Color   string
	Src     string
}

// AvatarGroup 头像叠放组（超出 max 显示 +N）。
func AvatarGroup(items []AvatarItem, max int) *Node {
	if max <= 0 {
		max = 4
	}
	shown := items
	more := 0
	if len(items) > max {
		shown = items[:max]
		more = len(items) - max
	}
	cells := make([]*Node, 0, len(shown)+1)
	for i, it := range shown {
		var a *Node
		if it.Src != "" {
			a = Stack(
				Image(it.Src).St(St().AbsFill().R(18)),
				Box().St(St().AbsFill().R(18).Stroke(2, ColBgDeep)),
			).St(St().W(36).H(36))
		} else {
			color := it.Color
			if color == "" {
				color = ColPrimary
			}
			a = Avatar(it.Initial, color)
			a.St(St().W(36).H(36).R(18))
		}
		if i > 0 {
			a.St(St().MarOnly(0, 0, 0, -10))
		}
		cells = append(cells, a)
	}
	if more > 0 {
		cells = append(cells, Text(fmt.Sprintf("+%d", more)).St(St().W(36).H(36).R(18).
			Text(11, 600, ColTextDim).CenterYOn().TAlign(AlignCenter).
			BgC(ColBgSoft).Stroke(1, ColStroke).MarOnly(0, 0, 0, -10)))
	}
	return Row(cells...).St(St().GapOf(0)).AlignTo(AlignCenter)
}

// ---- Tour ----

// TourStep 引导步骤。
type TourStep struct {
	Title  string
	Body   string
	Target string // 目标节点 Key（为空时居中显示）
	Place  string // bottom / top / right / left
}

// TourState 引导状态。
type TourState struct {
	Key  string
	Open bool
	Step int
	Seen map[string]bool
}

var tourStates = map[string]*TourState{}

// NewTour 创建引导状态（Seen 持久化到偏好）。
func NewTour(key string) *TourState {
	if s, ok := tourStates[key]; ok {
		return s
	}
	s := &TourState{Key: key, Seen: map[string]bool{}}
	if GetPrefBool("tour."+key, false) {
		s.Seen["done"] = true
	}
	tourStates[key] = s
	return s
}

// StartTour 开始引导。
func (s *TourState) Start() {
	if s == nil {
		return
	}
	s.Open = true
	s.Step = 0
	OpenOverlay("tour:"+s.Key, func() bool {
		s.Open = false
		RequestRedraw()
		return true
	})
	RequestRedraw()
}

// Finish 结束并记录已看。
func (s *TourState) Finish() {
	if s == nil {
		return
	}
	s.Open = false
	s.Seen["done"] = true
	SetPrefBool("tour."+s.Key, true)
	CloseOverlay("tour:" + s.Key)
	RequestRedraw()
}

// TourLayer 引导浮层（根 Stack 末尾）。targets 用于显示目标高亮。
func TourLayer(st *TourState, steps []TourStep) *Node {
	if st == nil || !st.Open || len(steps) == 0 {
		return nil
	}
	if st.Step >= len(steps) {
		st.Step = len(steps) - 1
	}
	if st.Step < 0 {
		st.Step = 0
	}
	step := steps[st.Step]
	hole := geom.Rect{}
	if step.Target != "" {
		if r, ok := LastFrameOf(step.Target); ok {
			hole = r
		}
	}
	// 遮罩挖空：四个方向的暗色块
	dark := paintWithAlpha("#04070e", 0.72)
	shade := func(l, t, w, h float32) *Node {
		return Box().St(St().AbsTL(l, t).W(w).H(h).BgC(dark)).Tap(func() {})
	}
	vw, vh := ViewportW(), ViewportH()
	pad := float32(6)
	hx, hy := hole.X-pad, hole.Y-pad
	hw, hh := hole.W+pad*2, hole.H+pad*2
	if hole.W <= 0 {
		hx, hy, hw, hh = 0, 0, 0, 0
	}
	layers := []*Node{
		shade(0, 0, vw, hy),
		shade(0, hy, hx, hh),
		shade(hx+hw, hy, vw-hx-hw, hh),
		shade(0, hy+hh, vw, vh-hy-hh),
	}
	if hole.W > 0 {
		layers = append(layers, Box().St(St().AbsTL(hx, hy).W(hw).H(hh).R(8).
			Stroke(2, ColPrimary)))
	}
	// 气泡
	bw := float32(300)
	bubble := Column(
		Text(step.Title).St(St().Text(13.5, 600, ColText)),
		Text(step.Body).St(St().Text(12, 400, ColTextDim).WrapOn()),
		Row(
			Text(fmt.Sprintf("%d / %d", st.Step+1, len(steps))).St(St().Text(10.5, 400, ColTextMuted)),
			Spacer(),
			Text("跳过").St(St().Text(11.5, 500, ColTextMuted)).Tap(st.Finish),
			Btn("下一步", BtnPrimary, func() {
				if st.Step < len(steps)-1 {
					st.Step++
					RequestRedraw()
					return
				}
				st.Finish()
			}),
		).St(St().GapOf(10)).AlignTo(AlignCenter),
	).St(St().W(bw).Padding(16).GapOf(10).R(12).BgC(ColPanelDeep).
		Stroke(1, ColStroke).Shadow(36, 14, 0, "rgba(0,0,0,0.6)"))
	pos := bubble
	switch step.Place {
	case "top":
		pos.St(St().AbsTL(clampf(hx, 12, vw-bw-12), hy-150))
	case "right":
		pos.St(St().AbsTL(hx+hw+12, clampf(hy, 12, vh-190)))
	case "left":
		pos.St(St().AbsTL(hx-bw-12, clampf(hy, 12, vh-190)))
	default:
		if hole.W > 0 {
			pos.St(St().AbsTL(clampf(hx, 12, vw-bw-12), hy+hh+12))
		} else {
			pos.St(St().AbsCenter())
		}
	}
	layers = append(layers, pos)
	return Stack(layers...).St(St().AbsFill())
}

// geomRectZero 已移除：Tour 直接使用 geom.Rect。
