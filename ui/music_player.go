//go:build windows

package ui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 音乐播放器组件（流媒体站点风格）：封面/黑胶唱片 + 曲目信息 + 金色进度条 +
// 播放控制 + 实时频谱。播放状态由 MusicSource 持有，回调内改状态后 RequestRedraw()。
//
// 布局（横向三段）：
//
//	┌──────────────────────────────────────────────────────────────┐
//	│  封面/唱片   正在播放 / 曲名 / 歌手·专辑           频谱柱状图  │
//	│              全宽进度条                                        │
//	│              ⏮ ▶ ⏭   🔊 ────   循环  倍速   1:23 / 4:05       │
//	└──────────────────────────────────────────────────────────────┘

const (
	// musicGoldMusic 进度 / 主按钮高亮色
	musicGold = "#f5c518"
	// musicIcon 图标默认色
	musicIcon = "rgba(236,241,252,0.94)"
	// musicTrackBg 轨道底色
	musicTrackBg = "rgba(255,255,255,0.20)"
)

var (
	// musicScrub 正在拖动进度条（拖动期间只改显示，松手才定位）。
	musicScrub = map[string]bool{}
	// musicDrag 拖动中的进度比例（0~1）。
	musicDrag = map[string]float32{}
)

// MusicTrack 曲目信息（标题 / 歌手 / 封面均可空，空则回退到文件名与唱片占位）。
type MusicTrack struct {
	Path   string
	Title  string
	Artist string
	Album  string
	Cover  string
	Dur    time.Duration
}

// MusicPlayerOptions 音乐播放器外观与回调。
type MusicPlayerOptions struct {
	// Track 曲目信息（标题为空时用文件名）。
	Track MusicTrack
	// Bars 频谱柱数量（默认 40）。
	Bars int
	// Prev / Next 上一首 / 下一首（nil 时按钮置灰不可点）。
	Prev func()
	Next func()
	// HideSpectrum 不显示频谱（纯音频条模式）。
	HideSpectrum bool
}

// MusicPlayer 音乐播放器组件。
// 尺寸由调用方指定（默认 100% 宽，高度随内容）；key 用于区分多个播放器。
func MusicPlayer(key string, s *MusicSource, opt MusicPlayerOptions) *Node {
	if key == "" {
		key = "music"
	}
	if s == nil {
		return Empty("未绑定音源", "用 ui.OpenMusic(\"song.mp3\") 打开音频文件")
	}
	if !s.Tick() && opt.Next != nil {
		opt.Next()
	}
	if s.Playing() {
		markAnim()
	}

	pos, dur := s.Pos(), s.Duration()
	track := opt.Track
	if track.Title == "" {
		track = s.NowPlaying()
	}
	bars := opt.Bars
	if bars <= 0 {
		bars = SpectrumBands
	}

	cover := musicCover(key+"-cover", track, 104, s.Playing())

	info := Column(
		Text("正在播放 · NOW PLAYING").St(St().Text(10, 700, musicGold)),
		Text(track.Title).St(St().Text(19, 700, ColText).WrapOn()),
		Row(
			Text(musicSubtitle(track)).St(St().Text(12, 400, "rgba(226,232,240,0.60)").WrapOn()).Grow(1),
			Text(formatVideoTime(pos)+" / "+formatVideoTime(dur)).
				St(St().Text(11.5, 500, "rgba(226,232,240,0.84)")),
		).St(St().GapOf(10)).AlignTo(AlignCenter),
	).St(St().GapOf(5))

	ctrl := Row(
		videoSkipIcon(key+"-prev", true, func() {
			if opt.Prev != nil {
				opt.Prev()
			}
		}).DisabledOn(opt.Prev == nil),
		musicPlayBtn(key, s),
		videoSkipIcon(key+"-next", false, func() {
			if opt.Next != nil {
				opt.Next()
			}
		}).DisabledOn(opt.Next == nil),
		videoSpeaker(key+"-vol", s.Muted(), 28, func() { s.ToggleMute() }),
		musicVolBar(key+"-volbar", s, 62),
		Spacer(),
		videoTextBtn(key+"-loop", "循环", 40, s.Loop(), func() { s.SetLoop(!s.Loop()) }),
		videoTextBtn(key+"-speed", musicSpeedLabel(s.Speed()), 46, s.Speed() != 1, func() {
			s.SetSpeed(nextSpeed(s.Speed()))
		}),
	).St(St().GapOf(8)).AlignTo(AlignCenter)

	right := Column(
		info,
		Spacer(),
		musicSeekBar(key+"-seek", s, pos),
		ctrl,
	).St(St().GapOf(10)).Grow(1).JustifyTo(AlignEnd)

	row := Row(
		cover,
		right,
		If(!opt.HideSpectrum, func() *Node {
			return Column(
				Row(
					Text("频谱").St(St().Text(9.5, 700, ColTextMuted)),
					Spacer(),
					Text(formatVideoTime(dur)).St(St().Text(9.5, 500, ColTextMuted)),
				).St(St().Wp(100)).AlignTo(AlignCenter),
				MusicSpectrum(key+"-spec", s.Spectrum(), bars, "rgba(245,197,24,0.95)", "rgba(139,92,246,0.95)").
					St(St().Wp(100).H(78)),
			).St(St().W(240).GapOf(6))
		}),
	).St(St().GapOf(18)).AlignTo(AlignCenter)

	st := St().Wp(100).Padding(18).R(16).GapOf(14).
		BgGrad("rgba(26,31,46,0.96)", "rgba(11,14,23,0.98)", 150).
		Stroke(1, "rgba(255,255,255,0.08)").
		Shadow(26, 12, 0, "rgba(0,0,0,0.42)")
	return Box(row).St(st).K(key + "-card")
}

// musicSubtitle 副标题：歌手 · 专辑（都空时显示容器信息）。
func musicSubtitle(t MusicTrack) string {
	switch {
	case t.Artist != "" && t.Album != "":
		return t.Artist + " · " + t.Album
	case t.Artist != "":
		return t.Artist
	case t.Album != "":
		return t.Album
	}
	if t.Path != "" {
		return "本地文件 · " + t.Path
	}
	return "未命名曲目"
}

// musicCover 封面：有图片用图片，否则画一张黑胶唱片（播放时标记点随节奏旋转）。
func musicCover(key string, t MusicTrack, size float32, playing bool) *Node {
	if t.Cover != "" {
		if st, err := os.Stat(t.Cover); err == nil && !st.IsDir() {
			return ImageSized(t.Cover, size, size).St(St().R(12).Clipped())
		}
	}
	dots := make([]*Node, 0, 6)
	r := size * 0.33
	ang0 := AnimTime() * 1.7
	if playing {
		markAnim()
	}
	for i := 0; i < 5; i++ {
		ang := float64(ang0) + float64(i)*2*math.Pi/5
		x := size/2 + float32(math.Cos(ang))*r - 1.6
		y := size/2 + float32(math.Sin(ang))*r - 1.6
		dots = append(dots, Box().St(St().AbsTL(x, y).W(3.2).H(3.2).R(1.6).
			BgC("rgba(245,197,24,0.80)")))
	}
	parts := []*Node{
		// 盘身
		Box().St(St().AbsFill().R(size/2).BgGrad(ColBgSoft, ColBgDeep, 150).
			Stroke(1, "rgba(255,255,255,0.10)")),
		// 纹路
		Box().St(St().AbsCenter().W(size*0.70).H(size*0.70).R(size*0.35).
			Stroke(1, "rgba(255,255,255,0.05)")),
		Box().St(St().AbsCenter().W(size*0.50).H(size*0.50).R(size*0.25).
			Stroke(1, "rgba(255,255,255,0.05)")),
		// 中心标签
		Box().St(St().AbsCenter().W(size*0.32).H(size*0.32).R(size*0.16).
			BgGrad("#f5c518", "#8b5cf6", 120)),
		// 中心孔
		Box().St(St().AbsCenter().W(5).H(5).R(2.5).BgC(ColBg)),
	}
	parts = append(parts, dots...)
	return Stack(parts...).St(St().W(size).H(size)).K(key)
}

// musicPlayBtn 播放 / 暂停圆形主按钮（金色，暂停时用双竖条形状绘制）。
func musicPlayBtn(key string, s *MusicSource) *Node {
	var inner *Node
	if s.Playing() {
		inner = Row(
			Box().St(St().W(4).H(13).R(1).BgC(ColTextMuted)),
			Box().St(St().W(4).H(13).R(1).BgC(ColTextMuted)),
		).St(St().AbsFill().GapOf(4)).JustifyTo(AlignCenter).AlignTo(AlignCenter)
	} else {
		inner = Text("▶").St(St().AbsFill().Text(13, 700, ColTextMuted).
			TAlign(AlignCenter).CenterYOn())
	}
	bg := C(musicGold)
	if IsHovered(key + "-play") {
		bg = C("#ffd84d")
	}
	n := Stack(inner).St(St().W(38).H(38).R(19).BgC(bg).
		Shadow(14, 4, 0, "rgba(245,197,24,0.28)"))
	n.K(key + "-play").Tap(func() {
		s.TogglePlay()
		RequestRedraw()
	})
	return n
}

// musicVolBar 横向音量条（点击 / 拖动调整，0 即静音）。
func musicVolBar(key string, s *MusicSource, w float32) *Node {
	v := s.Volume()
	if s.Muted() {
		v = 0
	}
	n := Stack(
		Box().St(St().AbsTL(0, 5).Wp(100).H(4).R(2).BgC(musicTrackBg)),
		Box().St(St().AbsTL(0, 5).Wp(v*100).H(4).R(2).BgC(musicGold)),
	).St(St().W(w).H(14))
	set := func(x float32) {
		if n.Frame.W > 0 {
			s.SetVolume(clampf(x/n.Frame.W, 0, 1))
		}
	}
	n.K(key)
	n.OnClickAt = func(x, _ float32) { set(x) }
	n.OnDrag = func(x, _ float32) { set(x) }
	return n
}

// musicSeekBar 全宽进度条：金色已播放段 + 白色圆点。
// 拖动期间只更新显示（避免每个移动事件都重启解码进程），松手后一次定位。
func musicSeekBar(key string, s *MusicSource, pos time.Duration) *Node {
	prog := s.Progress()
	if musicScrub[key] {
		prog = musicDrag[key]
	}
	grow := musicScrub[key] || IsHovered(key)
	h := float32(3)
	knob := float32(9)
	if grow {
		h, knob = 4, 12
	}
	n := Stack(
		Box().St(St().AbsTL(0, 8-h/2).Wp(100).H(h).R(h/2).BgC(musicTrackBg)),
		Box().St(St().AbsTL(0, 8-h/2).Wp(prog*100).H(h).R(h/2).BgC(musicGold)),
		Row(
			Box().St(St().W(knob).H(knob).R(knob/2).BgC(ColText).
				Shadow(6, 1, 0, "rgba(0,0,0,0.55)")),
		).St(St().Wp(prog*100)).JustifyTo(AlignEnd).AlignTo(AlignCenter),
	).St(St().H(16))

	n.K(key)
	n.OnClickAt = func(x, _ float32) {
		if n.Frame.W > 0 {
			s.SeekRatio(clampf(x/n.Frame.W, 0, 1))
		}
	}
	n.OnDrag = func(x, _ float32) {
		if n.Frame.W <= 0 {
			return
		}
		musicScrub[key] = true
		musicDrag[key] = clampf(x/n.Frame.W, 0, 1)
		RequestRedraw()
	}
	n.OnDragEnd = func() {
		if !musicScrub[key] {
			return
		}
		musicScrub[key] = false
		if v, ok := musicDrag[key]; ok {
			delete(musicDrag, key)
			s.SeekRatio(v)
		}
		RequestRedraw()
	}
	return n
}

// MusicSpectrum 频谱柱状图：对数频段能量驱动柱高（底部对齐），
// 颜色由 from（底部）到 to（顶部）渐变。组件本身不带高度，
// 调用方用 .St(St().H(78)) 之类指定高度。
func MusicSpectrum(key string, sp *Spectrum, bars int, from, to string) *Node {
	if sp == nil {
		return Box().St(St().Wp(100).Hp(100))
	}
	bands := sp.Bands()
	n := bars
	if n <= 0 {
		n = len(bands)
	}
	if n > len(bands) {
		n = len(bands)
	}
	cols := make([]*Node, 0, n)
	for i := 0; i < n; i++ {
		v := float32(0)
		if i < len(bands) {
			v = bands[i]
		}
		// 静音时保留一条细线，视觉上不塌陷
		h := 3 + v*97
		alpha := 0.55 + v*0.45
		cols = append(cols, Box().Grow(1).St(St().Hp(h).R(1.5).
			BgGrad(to, from, 180).Fade(alpha)))
		if v > 0.55 {
			// 高能量柱顶部加一点金色高光
			cols[len(cols)-1] = Stack(
				cols[len(cols)-1],
				Box().St(St().AbsTL(0, 0).Wp(100).H(2).R(1).BgC("rgba(255,255,255,0.55)")),
			).Grow(1).St(St().Hp(h))
		}
	}
	return Row(cols...).St(St().Wp(100).H(64).GapOf(3)).AlignTo(AlignEnd).K(key)
}

// MusicMirrorSpectrum 镜像频谱：以中线为轴上下对称（音乐 App 常见的“声浪”效果）。
// 高度与颜色参数同 MusicSpectrum。
func MusicMirrorSpectrum(key string, sp *Spectrum, bars int, from, to string) *Node {
	if sp == nil {
		return Box().St(St().Wp(100).Hp(100))
	}
	bands := sp.Bands()
	n := bars
	if n <= 0 {
		n = len(bands)
	}
	if n > len(bands) {
		n = len(bands)
	}
	cols := make([]*Node, 0, n)
	for i := 0; i < n; i++ {
		v := float32(0)
		if i < len(bands) {
			v = bands[i]
		}
		half := 2 + v*46
		cols = append(cols, Column(
			Box().St(St().Wp(100).Hp(half).R(1.5).BgGrad(from, to, 0).Fade(0.55+v*0.45)),
			Box().St(St().Wp(100).Hp(half).R(1.5).BgGrad(to, from, 180).Fade(0.55+v*0.45)),
		).Grow(1).St(St().Hp(100)).JustifyTo(AlignCenter))
	}
	return Row(cols...).St(St().Wp(100).H(72).GapOf(3)).AlignTo(AlignCenter).K(key)
}

// MusicPlaylist 播放列表组件：每行标题 + 副标题，当前曲目高亮并带「播放中」标签。
// tracks 的 Title 为空时回退到文件名；onPick 传回点击的下标（nil 时不可点）。
func MusicPlaylist(key string, tracks []MusicTrack, active int, onPick func(i int)) *Node {
	if len(tracks) == 0 {
		return Text("播放列表为空").St(St().Text(12, 400, ColTextMuted).Padding(10))
	}
	rows := make([]*Node, 0, len(tracks))
	for i, t := range tracks {
		i, t := i, t
		on := i == active
		bg := "rgba(255,255,255,0.03)"
		titleColor := ColTextDim
		idxColor := ColPlaceholder
		if on {
			bg = "rgba(245,197,24,0.12)"
			titleColor = ColText
			idxColor = musicGold
		}
		title := t.Title
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(t.Path), filepath.Ext(t.Path))
			if title == "" {
				title = "未命名曲目"
			}
		}
		sub := musicSubtitle(t)
		row := Row(
			Text(strconv.Itoa(i+1)).St(St().W(18).Text(11, 700, C(idxColor))),
			Column(
				Text(title).St(St().Text(12.5, 600, C(titleColor))),
				Text(sub).St(St().Text(10.5, 400, ColTextMuted)),
			).Grow(1).St(St().GapOf(2)),
			If(on, func() *Node { return Tag("播放中", musicGold) }),
		).St(St().GapOf(10).PadXY(10, 8).R(9).BgC(C(bg)))
		row.K(fmt.Sprintf("%s-row-%d", key, i))
		if onPick != nil {
			row.Tap(func() {
				onPick(i)
				RequestRedraw()
			})
		}
		rows = append(rows, row)
	}
	return Column(rows...).St(St().GapOf(4)).K(key)
}

// musicSpeedLabel 倍速文本（1 显示为「正常」，其余为 0.5x / 1.25x 等）。
func musicSpeedLabel(v float32) string { return videoSpeedLabel(v) }

// musicSpeedSteps 倍速档位（点击循环切换）。
var musicSpeedSteps = []float32{0.5, 0.75, 1, 1.25, 1.5, 2}

// nextSpeed 下一个倍速档位。
func nextSpeed(v float32) float32 {
	for i, s := range musicSpeedSteps {
		if math.Abs(float64(s-v)) < 0.01 {
			return musicSpeedSteps[(i+1)%len(musicSpeedSteps)]
		}
	}
	return 1
}
