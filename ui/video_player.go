package ui

import (
	"fmt"
	"math"
	"time"

	"gogui/internal/geom"
)

// 播放器组件（参考流媒体站点：金色进度条 + 无底图标按钮 + 右侧倍速面板）。
// 约定与其它组件一致：状态由业务持有的 Player 保存，回调里改状态后 RequestRedraw()。

const (
	// videoGold 已播放进度/高亮色
	videoGold = "#f5c518"
	// videoIcon 控制条图标默认色
	videoIcon = "rgba(236,241,252,0.94)"
	// videoTrack 进度/音量轨道底色
	videoTrack = "rgba(255,255,255,0.24)"
)

var (
	// videoScrub 正在拖动进度条（拖动期间暂停走时钟，松手后恢复）。
	videoScrub = map[string]bool{}
	// videoScrubAt 最近一次拖动回调的时间（判断拖动结束）。
	videoScrubAt = map[string]float32{}
	// videoShowAt 控制条最近一次因鼠标移动而显示的时间（自动隐藏）。
	videoShowAt = map[string]float32{}
	// videoMouse 各播放器上一次的鼠标位置（判断鼠标是否移动）。
	videoMouse = map[string][2]float32{}
	// videoSpeedOpen 倍速面板是否展开。
	videoSpeedOpen = map[string]bool{}
	// videoVolOpen 音量滑块浮层最后一次交互的时间（悬停后显示 0.8s）。
	videoVolOpen = map[string]float32{}
)

// VideoPlayer 视频播放器：画面 + 底部渐变控制层。
//
// 布局（自下而上）：全宽进度条（金色已播放段 + 白色圆点）、
// 图标行（播放/暂停、上一集、下一集、音量图标 + 音量滑块、时间、倍速、全屏）。
// 暂停时画面中央显示双环播放按钮；右侧倍速面板列出 0.5x~2x。
//
// 尺寸由调用方通过 .St(St().H(...)) 指定；默认 100% 宽 × 360 高。
func VideoPlayer(key string, p *Player) *Node {
	if key == "" {
		key = "video"
	}
	if p == nil {
		return Empty("未绑定播放器", "用 ui.NewPlayer(clip) 创建播放状态")
	}
	now := AnimTime()
	// 兜底：鼠标移出窗口等情况收不到松手回调，超时无拖动事件即视为已松开
	if videoScrub[key] && now-videoScrubAt[key] > 0.3 {
		videoReleaseScrub(key, p)
	}
	p.scrub = videoScrub[key]
	p.mediaScrub()
	p.advance(now)
	if p.scrub {
		// 拖动期间持续重绘：后台抓回的预览帧才能立刻上屏
		markAnim()
	}

	// 窗口已退出全屏（用户按 ESC）时同步退出播放器全屏；无窗口（测试/离屏）不做同步
	if full, known := windowFullscreenState(); p.Fullscreen && known && !full {
		p.Fullscreen = false
		SetFocus("")
	}

	// ---- 画面 ----
	surface := Video(p).St(St().AbsFill().BgC("#000000").FocusOn()).K(key + "-surface")
	surface.OnClick = func() {
		// 锁定后点画面不切播放，避免误触
		if p.Locked {
			return
		}
		p.TogglePlay()
	}
	surface.OnKey = func(code uint32) {
		// 锁定后除 ESC 外的快捷键失效
		if p.Locked && code != 0x1B {
			return
		}
		switch code {
		case 0x20: // Space
			p.TogglePlay()
		case 0x25: // ←
			p.SeekBy(-5 * time.Second)
		case 0x27: // →
			p.SeekBy(5 * time.Second)
		case 0x26: // ↑
			p.SetVolume(p.Volume + 0.1)
		case 0x28: // ↓
			p.SetVolume(p.Volume - 0.1)
		case 0x24: // Home
			p.SetPos(0)
		case 0x23: // End
			p.SetPos(p.Duration())
		case 0x1B: // Esc：退出播放器全屏
			if p.Fullscreen {
				p.Fullscreen = false
				RequestRedraw()
			}
		}
	}

	children := []*Node{surface}

	// ---- 字幕（贴画面底部居中，压在控制层之上）----
	if p.Subtitle != "" {
		children = append(children, Text(p.Subtitle).St(St().
			AbsBottom(70).Wp(92).Text(14, 500, "#ffffff").
			TAlign(AlignCenter).WrapOn()))
	}

	// ---- 暂停时的中央播放按钮（锁定时不显示，避免误触）----
	if !p.Playing && !p.HideControls && !p.Locked {
		children = append(children, videoCenterPlay(key, p))
	}

	// ---- 底部控制层 ----
	if !p.HideControls {
		target := float32(0)
		if videoControlsShown(key, p, now) {
			target = 1
		}
		op := Tween(key+"-ctrl", target, 0.18, EaseOutCubic)
		if op > 0.02 {
			children = append(children, videoSideTools(key, p, op))
			children = append(children, videoControlLayer(key, p, op))
		}
	}

	// ---- 音量浮层（挂在根层，避免 28px 图标格裁切）----
	if at, ok := videoVolOpen[key]; ok && AnimTime()-at < 0.8 {
		if pop := videoVolumePopup(key, p); pop != nil {
			children = append(children, pop)
		}
	}

	// ---- 倍速面板 ----
	if videoSpeedOpen[key] {
		children = append(children, videoSpeedPanel(key, p))
	}

	st := St().Wp(100).H(360).R(10).Clipped().
		BgC("#000000").Stroke(1, ColLine)
	if p.Fullscreen {
		// 全屏：铺满窗口根 Stack（去掉圆角与描边）
		st = St().BgC("#000000")
		st.PosAbs = true
		st.HasL, st.AbsL = true, 0
		st.HasT, st.AbsT = true, 0
		st.HasR, st.AbsR = true, 0
		st.HasB, st.AbsB = true, 0
	}
	return Stack(children...).St(st).K(key + "-root")
}

// videoSideTools 画面左侧中间的旋转 / 锁定竖排按钮（形状图标，不依赖字形）。
func videoSideTools(key string, p *Player, op float32) *Node {
	rot := videoRoundBtn(key+"-rotate", videoIconRotate, p.Rotation&3 != 0, func() { p.RotateBy(1) })
	TooltipOf(rot, fmt.Sprintf("旋转画面 90°（当前 %d°）", p.RotationDeg()))
	lock := videoRoundBtn(key+"-lock", videoIconLock, p.Locked, func() { p.ToggleLock() })
	tip := "锁定画面（禁止点击与快捷键）"
	if p.Locked {
		tip = "已锁定：点此解锁"
	}
	TooltipOf(lock, tip)
	col := Column(rot, lock).St(St().AbsTL(10, 0).Hp(100).GapOf(8).Fade(op))
	col.JustifyTo(AlignCenter)
	return col
}

// videoRoundBtn 悬浮圆形工具按钮（激活态金色）。
func videoRoundBtn(key string, icon func(color string) *Node, active bool, onClick func()) *Node {
	bg := C("rgba(8,12,20,0.52)")
	fg := "rgba(236,241,252,0.92)"
	border := "rgba(255,255,255,0.22)"
	if active {
		bg = C("rgba(245,197,24,0.92)")
		fg = ColTextMuted
		border = "rgba(245,197,24,0.95)"
	}
	if IsHovered(key) {
		if active {
			bg = C("rgba(255,214,64,1)")
		} else {
			bg = C("rgba(28,36,52,0.88)")
			fg = "#ffffff"
		}
	}
	n := Stack(icon(fg)).St(St().W(30).H(30).R(15).BgC(bg).Stroke(1, border))
	n.K(key).Tap(func() {
		if onClick != nil {
			onClick()
		}
		RequestRedraw()
	})
	return n
}

// videoIconRotate 顺时针圆形箭头（圆点弧 + 箭头尾巴）。
func videoIconRotate(color string) *Node {
	const rad, dot = 5.4, 2.2
	items := make([]*Node, 0, 10)
	for i := 0; i < 8; i++ {
		a := (-160.0 + float64(i)*36) * math.Pi / 180
		x := 15 + rad*math.Cos(a)
		y := 15 + rad*math.Sin(a)
		items = append(items, Box().St(St().
			AbsTL(float32(x)-dot/2, float32(y)-dot/2).W(dot).H(dot).R(dot/2).BgC(color)))
	}
	// 箭头：缺口处一根短横 + 一个小圆点，读作“旋转”
	items = append(items,
		Box().St(St().AbsTL(19.4, 11.2).W(4.6).H(1.8).R(0.9).BgC(color)),
		Box().St(St().AbsTL(19.4, 11.2).W(1.8).H(5.4).R(0.9).BgC(color)),
	)
	return Stack(items...).St(St().W(30).H(30))
}

// videoIconLock 锁头（抖环 + 锁体 + 锁孔）。
func videoIconLock(color string) *Node {
	shackle := Box().St(St().AbsTL(10.5, 5.6).W(9).H(8).R(4.5).
		BgC(C("transparent")).Stroke(1.8, color))
	body := Box().St(St().AbsTL(8, 11).W(14).H(10).R(2.6).BgC(color))
	hole := Box().St(St().AbsTL(13.6, 14).W(2.8).H(2.8).R(1.4).BgC("rgba(0,0,0,0.62)"))
	return Stack(shackle, body, hole).St(St().W(30).H(30))
}

// videoControlsShown 控制条是否显示：暂停时始终显示；播放中鼠标移入画面后显示 2.5s。
func videoControlsShown(key string, p *Player, now float32) bool {
	if !p.AutoHide || !p.Playing {
		return true
	}
	x, y := MousePos()
	inside := p.rect.contains(x, y)
	last, ok := videoMouse[key]
	if inside && (!ok || last[0] != x || last[1] != y) {
		videoShowAt[key] = now
	}
	videoMouse[key] = [2]float32{x, y}
	at, ok := videoShowAt[key]
	if !ok {
		videoShowAt[key] = now
		return true
	}
	return now-at < 2.5
}

// videoCenterPlay 画面中央的双环播放按钮（内圆 + 外圈微光，参考网站播放器）。
func videoCenterPlay(key string, p *Player) *Node {
	core := Stack(
		Box().St(St().W(56).H(56).R(28).BgC("rgba(14,18,28,0.82)").
			Stroke(1.5, "rgba(255,255,255,0.86)")),
		Text("▶").St(St().AbsFill().Text(18, 600, "#ffffff").TAlign(AlignCenter).CenterYOn()),
	).St(St().W(56).H(56).AbsCenter())

	n := Stack(
		Box().St(St().W(82).H(82).R(41).BgC("rgba(8,12,20,0.28)").
			Stroke(1, "rgba(255,255,255,0.18)")),
		core,
	).St(St().W(82).H(82).AbsCenter())
	n.K(key + "-bigplay").Tap(func() { p.Play() })
	return n
}

// videoControlLayer 底部渐变控制层：全宽进度条 + 图标行。
// 锁定后只保留（只读）进度条，菜单/倍速/全屏等按钮全部隐藏，防止误触。
func videoControlLayer(key string, p *Player, op float32) *Node {
	if p.Locked {
		st := St().PadOnly(2, 16, 14, 16).
			BgGrad("rgba(4,6,12,0)", "rgba(4,6,12,0.55)", 180).
			Fade(op)
		st.PosAbs = true
		st.HasL, st.AbsL = true, 0
		st.HasR, st.AbsR = true, 0
		st.HasB, st.AbsB = true, 0
		return Column(videoSeekBar(key+"-seek", p, true)).St(st).K(key + "-ctrl-ro")
	}
	glyph, tip := "▶", "播放"
	if p.Playing {
		glyph, tip = "", "暂停"
	}
	play := videoIconBtn(key+"-play", glyph, 34, func() { p.TogglePlay() })
	TooltipOf(play, tip)

	cells := []*Node{play}
	if p.OnPrev != nil {
		prev := videoSkipIcon(key+"-prev", true, func() { p.OnPrev() })
		TooltipOf(prev, "上一集")
		cells = append(cells, prev)
	}
	if p.OnNext != nil {
		next := videoSkipIcon(key+"-next", false, func() { p.OnNext() })
		TooltipOf(next, "下一集")
		cells = append(cells, next)
	}
	cells = append(cells,
		videoVolumeGroup(key, p),
		Text(formatVideoTime(p.Pos())+" / "+formatVideoTime(p.Duration())).
			St(St().Text(11.5, 500, "rgba(226,232,240,0.88)")),
		Spacer(),
		videoTextBtn(key+"-loop", "循环", 40, p.Loop, func() { p.SetLoop(!p.Loop) }),
		videoTextBtn(key+"-speed", "倍速", 44, videoSpeedOpen[key], func() {
			videoSpeedOpen[key] = !videoSpeedOpen[key]
			RequestRedraw()
		}),
		videoFullBtn(key, p),
	)

	row := Row(cells...).St(St().GapOf(12)).AlignTo(AlignCenter)

	// 底部渐变（透明 → 深色），与网站播放器一致
	st := St().PadOnly(4, 16, 10, 16).GapOf(2).
		BgGrad("rgba(4,6,12,0)", "rgba(4,6,12,0.88)", 180).
		Fade(op)
	st.PosAbs = true
	st.HasL, st.AbsL = true, 0
	st.HasR, st.AbsR = true, 0
	st.HasB, st.AbsB = true, 0
	return Column(videoSeekBar(key+"-seek", p, false), row).St(st)
}

// videoSeekBar 全宽进度条：轨道 + 金色已播放段 + 白色圆点（点击/拖动跳转）。
// readonly 为 true 时只展示进度，不响应点击/拖动（锁定状态）。
func videoSeekBar(key string, p *Player, readonly bool) *Node {
	prog := p.Progress()
	grow := !readonly && (videoScrub[key] || IsHovered(key))
	h := float32(3)
	knob := float32(9)
	if grow {
		h, knob = 4, 12
	}
	n := Stack(
		Box().St(St().AbsTL(0, 9-h/2).Wp(100).H(h).R(h/2).BgC(videoTrack)),
		Box().St(St().AbsTL(0, 9-h/2).Wp(prog*100).H(h).R(h/2).BgC(videoGold)),
		Row(
			Box().St(St().W(knob).H(knob).R(knob/2).BgC("#ffffff").
				Shadow(6, 1, 0, "rgba(0,0,0,0.55)")),
		).St(St().Wp(prog*100)).JustifyTo(AlignEnd).AlignTo(AlignCenter),
	).St(St().Wp(100).H(18))

	n.K(key)
	if readonly {
		return n
	}
	n.OnClickAt = func(x, _ float32) {
		if n.Frame.W > 0 {
			p.SeekRatio(clampf(x/n.Frame.W, 0, 1))
		}
	}
	// 拖动：开始即进入拖动模式（声音暂停、只记目标位置），坐标变化时更新目标并请求预览
	n.OnDrag = func(x, _ float32) {
		if !videoScrub[key] {
			videoScrub[key] = true
			p.scrub = true
			p.mediaScrub()
		}
		videoScrubAt[key] = AnimTime()
		if n.Frame.W > 0 {
			p.SeekRatio(clampf(x/n.Frame.W, 0, 1))
		}
	}
	// 松手：事件驱动提交（参考 gio Clickable.Pressed 变 false），不再靠超时猜测
	n.OnDragEnd = func() { videoReleaseScrub(key, p) }
	return n
}

// videoReleaseScrub 结束拖动：清状态并让解码源一次性定位（画面与声音一起跳到目标）。
func videoReleaseScrub(key string, p *Player) {
	if !videoScrub[key] {
		return
	}
	videoScrub[key] = false
	delete(videoScrubAt, key)
	if p != nil {
		p.scrub = false
		p.mediaScrub()
	}
	RequestRedraw()
}

// videoIconBtn 无底色图标按钮（悬停时叠一层浅色圆角，播放/暂停用形状绘制）。
func videoIconBtn(key, glyph string, size float32, onClick func()) *Node {
	return videoIconBtnT(key, glyph, size, videoIcon, onClick)
}

// videoIconBtnT 指定图标颜色的无底色按钮。
func videoIconBtnT(key, glyph string, size float32, iconColor string, onClick func()) *Node {
	bg := C("transparent")
	c := iconColor
	if IsHovered(key) {
		bg = C("rgba(255,255,255,0.14)")
		c = "#ffffff"
	} else if IsPressed(key) {
		bg = C("rgba(255,255,255,0.22)")
	}

	var inner *Node
	if glyph == "" {
		// 暂停：两根竖条（形状绘制，任何字体都不会缺字形）
		inner = Row(
			Box().St(St().W(4).H(13).R(1).BgC(c)),
			Box().St(St().W(4).H(13).R(1).BgC(c)),
		).St(St().AbsFill().GapOf(4)).JustifyTo(AlignCenter).AlignTo(AlignCenter)
	} else {
		inner = Text(glyph).St(St().AbsFill().Text(13, 600, c).
			TAlign(AlignCenter).CenterYOn())
	}

	n := Stack(inner).St(St().W(size).H(size).R(size / 2).BgC(bg))
	n.K(key).Tap(func() {
		if onClick != nil {
			onClick()
		}
		RequestRedraw()
	})
	return n
}

// videoSkipIcon 上一集 / 下一集图标（三角 + 竖条，形状与字形混合绘制）。
func videoSkipIcon(key string, prev bool, onClick func()) *Node {
	c := videoIcon
	if IsHovered(key) {
		c = "#ffffff"
	}
	tri := Text("◀").St(St().Text(11, 600, c).CenterYOn())
	bar := Box().St(St().W(2).H(10).R(1).BgC(c))
	units := []*Node{tri, bar}
	if prev {
		// 上一集：竖条在左，三角向左
		units = []*Node{bar, tri}
		tri.Text = "◀"
	} else {
		tri.Text = "▶"
	}
	bg := C("transparent")
	if IsHovered(key) {
		bg = C("rgba(255,255,255,0.14)")
	}
	n := Stack(
		Row(units...).St(St().AbsFill().GapOf(1)).JustifyTo(AlignCenter).AlignTo(AlignCenter),
	).St(St().W(30).H(30).R(15).BgC(bg))
	n.K(key).Tap(func() {
		if onClick != nil {
			onClick()
		}
		RequestRedraw()
	})
	return n
}

// videoVolumeGroup 音量图标（点击静音）+ 图标上方弹出的竖向音量滑块。
// videoSpeaker 喇叭图标按钮（纯形状绘制：底座 + 渐高的喇叭口 + 声波）。
// 与播放/暂停、全屏同构（固定方框内居中），保证与同行按钮垂直对齐一致。
func videoSpeaker(key string, muted bool, size float32, onClick func()) *Node {
	bg := C("transparent")
	col := C(videoIcon)
	if IsHovered(key) {
		bg = C("rgba(255,255,255,0.14)")
		col = C("#ffffff")
	}
	if muted {
		col = col.WithAlpha(0.40)
	}

	const iw, ih float32 = 16, 14
	parts := []*Node{
		Box().St(St().W(3).H(5).R(0.8).BgC(col).AbsTL(0, 4.5)),
		Box().St(St().W(4).H(9).R(0.8).BgC(col).AbsTL(3, 2.5)),
		Box().St(St().W(5).H(13).R(1).BgC(col).AbsTL(7, 0.5)),
	}
	if !muted {
		// 声波：两段高度递增的细条
		parts = append(parts,
			Box().St(St().W(1.5).H(4).R(0.75).BgC(col).AbsTL(13, 5)),
			Box().St(St().W(1.5).H(8).R(0.75).BgC(col).AbsTL(15, 3)),
		)
	}
	icon := Stack(parts...).St(St().W(iw).H(ih))
	n := Stack(
		Stack(icon).St(St().AbsCenter()),
	).St(St().W(size).H(size).R(size / 2).BgC(bg))
	n.K(key).Tap(func() {
		if onClick != nil {
			onClick()
		}
		RequestRedraw()
	})
	return n
}

// videoVolumeGroup 音量图标（点击静音）+ 悬停时记录位置供根层弹出竖向滑块。
func videoVolumeGroup(key string, p *Player) *Node {
	muted := p.Muted || p.Volume <= 0
	icon := videoSpeaker(key+"-vol", muted, 28, func() { p.ToggleMute() })

	// 悬停在图标/浮层上时记录时间（浮层在播放器根层绘制，不受 28px 图标格约束）
	if IsHovered(key+"-vol") || IsHovered(key+"-volpop") || IsHovered(key+"-volbar") {
		videoVolOpen[key] = AnimTime()
	}
	return Stack(icon).St(St().W(28).H(28).Self(AlignCenter))
}

// windowFullscreenState 返回（窗口是否全屏, 是否存在窗口）。
func windowFullscreenState() (bool, bool) {
	a := currentApp.Load()
	if a == nil {
		return false, false
	}
	return a.win.IsFullscreen(), true
}

// volumeGlyph 音量图标改由 videoSpeaker 形状绘制（喇叭），不再依赖 Emoji 字形基线。

// videoVolumePopup 音量竖向滑块浮层（挂在播放器根层，按图标上一帧位置定位）。
func videoVolumePopup(key string, p *Player) *Node {
	icon, okIcon := LastFrameOf(key + "-vol")
	root, okRoot := LastFrameOf(key + "-root")
	if !okIcon || !okRoot {
		return nil
	}
	const panelW, trackW, knob float32 = 34, 4, 10
	trackH := float32(96)
	v := p.Volume
	if p.Muted {
		v = 0
	}
	inner := panelW - 8 // 左右内边距各 4
	fillH := trackH * v
	bar := Stack(
		Box().St(St().AbsTL((inner-trackW)/2, 0).W(trackW).H(trackH).R(trackW/2).BgC(videoTrack)),
		Box().St(St().AbsTL((inner-trackW)/2, trackH-fillH).W(trackW).H(fillH).R(trackW/2).BgC(videoGold)),
		Box().St(St().W(knob).H(knob).R(knob/2).BgC("#ffffff").
			Shadow(6, 1, 0, "rgba(0,0,0,0.6)").
			AbsTL((inner-knob)/2, trackH-fillH-knob/2)),
	).St(St().W(inner).H(trackH))

	set := func(y float32) {
		if bar.Frame.H > 0 {
			p.SetVolume(clampf(1-y/bar.Frame.H, 0, 1))
		}
	}
	bar.K(key + "-volbar")
	bar.OnClickAt = func(_, y float32) { set(y) }
	bar.OnDrag = func(_, y float32) { set(y) }

	// 面板贴图标上方：水平居中于图标，底部距图标顶部 6px
	x, y := videoVolPanelPos(icon, root, panelW, trackH+16)
	return Column(bar).St(St().
		AbsTL(x, y).
		PadXY(4, 8).
		BgC(ColBgSoft).R(8).
		Stroke(1, ColLine).
		Shadow(20, 6, 0, "rgba(0,0,0,0.5)")).K(key + "-volpop")
}

// videoTextBtn 文字胶囊按钮（倍速入口）。
func videoTextBtn(key, label string, w float32, active bool, onClick func()) *Node {
	bg := C("rgba(255,255,255,0.10)")
	color := videoIcon
	if active {
		bg = C("rgba(245,197,24,0.18)")
		color = videoGold
	} else if IsHovered(key) {
		bg = C("rgba(255,255,255,0.18)")
		color = "#ffffff"
	}
	n := Text(label).St(St().W(w).H(23).R(6).BgC(bg).
		Text(11.5, 600, color).TAlign(AlignCenter).CenterYOn().
		Stroke(1, "rgba(255,255,255,0.16)"))
	n.K(key).Tap(func() {
		if onClick != nil {
			onClick()
		}
		RequestRedraw()
	})
	return n
}

// videoFullBtn 全屏按钮（四角形状绘制，避免依赖字体里是否有全屏字形）。
func videoFullBtn(key string, p *Player) *Node {
	c := videoIcon
	if IsHovered(key) {
		c = "#ffffff"
	}
	const arm, thick, size float32 = 6, 1.5, 14
	corner := func(ax, ay float32, hx, vy float32) *Node {
		return Stack(
			Box().St(St().W(arm).H(thick).R(thick/2).BgC(c).AbsTL(ax, ay)),
			Box().St(St().W(thick).H(arm).R(thick/2).BgC(c).AbsTL(hx, vy)),
		).St(St().W(size).H(size))
	}
	icon := Stack(
		corner(0, 0, 0, 0),
		corner(size-arm, 0, size-thick, 0),
		corner(0, size-thick, 0, size-arm),
		corner(size-arm, size-thick, size-thick, size-arm),
	).St(St().W(size).H(size))

	bg := C("transparent")
	if IsHovered(key) {
		bg = C("rgba(255,255,255,0.14)")
	}
	n := Stack(
		Stack(icon).St(St().AbsCenter()),
	).St(St().W(28).H(28).R(14).BgC(bg))
	n.K(key + "-full").Tap(func() {
		// 播放器铺满窗口 + 系统窗口真全屏（覆盖当前显示器）
		p.Fullscreen = !p.Fullscreen
		if cur := currentApp.Load(); cur != nil {
			cur.SetFullscreen(p.Fullscreen)
		}
		if p.OnFullscreen != nil {
			p.OnFullscreen()
		}
		RequestRedraw()
	})
	return n
}

// videoSpeedPanel 倍速面板（右侧竖排 0.5x~2x，当前速率高亮）。
func videoSpeedPanel(key string, p *Player) *Node {
	speeds := []float32{0.5, 0.75, 1, 1.25, 1.5, 2}
	items := make([]*Node, 0, len(speeds))
	for _, sp := range speeds {
		s := sp
		label := videoSpeedLabel(s)
		active := p.SpeedValue() == s
		bg := C("transparent")
		color := "rgba(226,232,240,0.86)"
		weight := int32(500)
		if active {
			bg = C("rgba(255,255,255,0.10)")
			color = "#ffffff"
			weight = 700
		} else if IsHovered(key + "-sp-" + videoSpeedLabel(s)) {
			bg = C(ColLine)
			color = "#ffffff"
		}
		row := Text(label).St(St().Wp(100).H(24).R(5).BgC(bg).
			Text(11.5, weight, color).TAlign(AlignCenter).CenterYOn())
		row.K(key + "-sp-" + label)
		row.Tap(func() {
			p.SetSpeed(s)
			videoSpeedOpen[key] = false
			RequestRedraw()
		})
		items = append(items, row)
	}
	return Column(items...).St(St().AbsBR(50, 54).W(76).Padding(5).GapOf(2).R(8).
		BgC(ColBgSoft).Stroke(1, ColLine).
		Shadow(26, 10, 0, "rgba(0,0,0,0.55)"))
}

// videoVolPanelPos 计算音量浮层左上角（相对播放器根节点）：水平居中于图标，
// 底部距图标顶部 6px。参数为几何矩形，便于单独测试。
func videoVolPanelPos(icon, root geom.Rect, panelW, panelH float32) (float32, float32) {
	x := icon.X - root.X + icon.W/2 - panelW/2
	y := icon.Y - root.Y - panelH - 6
	return x, y
}

// videoSpeedLabel 倍速文本（1 显示为「正常」，其余为 0.5x、1.25x 等）。
func videoSpeedLabel(v float32) string {
	if v == 1 {
		return "正常"
	}
	if v == float32(int(v)) {
		return fmt.Sprintf("%dx", int(v))
	}
	return fmt.Sprintf("%gx", v)
}
