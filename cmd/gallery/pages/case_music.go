// Go-gui — gallery 页面拆分：音乐播放器
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"fmt"
	"gogui/ui"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// musicDemo 内置演示曲目（程序化合成 WAV）。
type musicDemo struct {
	title, artist, album string
	root                 float64 // 基频（Hz）
	bpm                  int
	seed                 int
}

var musicDemos = []musicDemo{
	{"Sunrise Drive", "GoGUI Synth", "Demo Album · Vol.1", 220.0, 112, 1},
	{"Neon Nights", "GoGUI Synth", "Demo Album · Vol.1", 164.8, 128, 2},
	{"Midnight Rain", "GoGUI Synth", "Demo Album · Vol.2", 196.0, 92, 3},
}

var (
	musicIndex int
	musicAuto  bool
	musicSrc   *ui.MusicSource
	musicMsg   = "点播放听到的是程序化合成的示例曲目（无需外部素材）；频谱随声音实时跳动"
	musicPaths = map[int]string{}
)

// musicDemoPath 生成 / 复用内置 WAV（放在系统缓存目录）。
func musicDemoPath(i int) string {
	if p := musicPaths[i]; p != "" {
		return p
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "gogui-gallery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	d := musicDemos[i]
	p := filepath.Join(dir, fmt.Sprintf("demo-%d-%d.wav", i, d.seed))
	if st, serr := os.Stat(p); serr != nil || st.Size() < 4096 {
		if werr := writeDemoWAV(p, d); werr != nil {
			return ""
		}
	}
	musicPaths[i] = p
	return p
}

// writeDemoWAV 生成一段简单合成器循环（底鼓 + 贝斯 + 琶音 + 踩镲），
// 44.1kHz / 单声道 / 16bit，总长 16 秒，可直接被 ffmpeg 解码播放。
func writeDemoWAV(path string, d musicDemo) error {
	const rate = 44100
	const secs = 16.0
	n := int(secs * rate)
	samples := make([]int16, n)
	beat := 60.0 / float64(d.bpm)
	// 大调音阶（主音 → 八度），琶音按 1-3-5 跳进
	scale := []float64{1, 9.0 / 8, 5.0 / 4, 3.0 / 2, 5.0 / 3, 2, 5.0 / 2}
	rnd := uint32(2463534242 + d.seed*7919)
	noise := func() float64 {
		rnd ^= rnd << 13
		rnd ^= rnd >> 17
		rnd ^= rnd << 5
		return float64(rnd)/float64(1<<31) - 1
	}
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		var v float64

		// 主旋律：每个 1/8 拍换一个音，指数衰减包络
		eighth := beat / 2
		si := int(t / eighth)
		st := t - float64(si)*eighth
		note := d.root * scale[(si*3+int(d.seed)*2)%len(scale)]
		env := math.Exp(-st*6) * 0.28
		v += math.Sin(2*math.Pi*note*t) * env
		v += math.Sin(2*math.Pi*note*2*t) * env * 0.16 // 泛音，频谱更丰富

		// 贝斯：每拍一个低音（正弦 + 三次谐波）
		bi := int(t / beat)
		bt := t - float64(bi)*beat
		bf := d.root / 4 * scale[bi%4]
		benv := math.Exp(-bt*3.2) * 0.34
		v += math.Sin(2*math.Pi*bf*t) * benv
		v += math.Sin(2*math.Pi*bf*3*t) * benv * 0.12

		// 底鼓：每拍 120Hz→45Hz 扫频 + 指数衰减
		if bt < 0.18 {
			kf := 45 + 75*math.Exp(-bt*28)
			v += math.Sin(2*math.Pi*kf*bt) * math.Exp(-bt*16) * 0.55
		}
		// 踩镲：半拍处的短噪声
		hi := t - float64(int(t/(beat/2)))*(beat/2)
		if hi < 0.05 {
			v += noise() * math.Exp(-hi*90) * 0.12
		}

		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		samples[i] = int16(v * 32000)
	}

	data := make([]byte, 44+len(samples)*2)
	putU32 := func(off int, v uint32) {
		data[off], data[off+1], data[off+2], data[off+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	putU16 := func(off int, v uint16) {
		data[off], data[off+1] = byte(v), byte(v>>8)
	}
	copy(data[0:4], "RIFF")
	putU32(4, uint32(len(data)-8))
	copy(data[8:12], "WAVE")
	copy(data[12:16], "fmt ")
	putU32(16, 16)
	putU16(20, 1) // PCM
	putU16(22, 1) // 单声道
	putU32(24, rate)
	putU32(28, rate*2)
	putU16(32, 2)
	putU16(34, 16)
	copy(data[36:40], "data")
	putU32(40, uint32(len(samples)*2))
	for i, s := range samples {
		putU16(44+i*2, uint16(s))
	}
	return os.WriteFile(path, data, 0o644)
}

// currentMusicTrack 当前曲目信息。
func currentMusicTrack() ui.MusicTrack {
	if musicIndex < 0 || musicIndex >= len(musicDemos) {
		return ui.MusicTrack{}
	}
	d := musicDemos[musicIndex]
	t := ui.MusicTrack{Title: d.title, Artist: d.artist, Album: d.album}
	if musicSrc != nil {
		t.Path = musicSrc.Path()
		t.Dur = musicSrc.Duration()
	}
	return t
}

// openMusicDemo 打开指定下标的示例曲目。
func openMusicDemo(i int) {
	n := len(musicDemos)
	if i < 0 {
		i = n - 1
	}
	if i >= n {
		i = 0
	}
	p := musicDemoPath(i)
	if p == "" {
		musicMsg = "示例音频生成失败（缓存目录不可写？）"
		ui.RequestRedraw()
		return
	}
	musicIndex = i
	openMusicFile(p, true)
	musicMsg = fmt.Sprintf("正在播放示例：%s · %s（%s）",
		musicDemos[i].title, musicDemos[i].artist, fmtDur(musicSrc.Duration()))
	ui.RequestRedraw()
}

// openMusicFile 打开音频文件并（可选）自动播放，旧音源自动关闭。
func openMusicFile(path string, play bool) {
	if strings.TrimSpace(path) == "" {
		return
	}
	src, err := ui.OpenMusic(path)
	if err != nil {
		musicMsg = "打开失败：" + err.Error()
		ui.RequestRedraw()
		return
	}
	old := musicSrc
	musicSrc = src
	if play {
		src.Play()
	}
	if old != nil && old != src {
		old.Close()
	}
	info := src.Info()
	name := filepath.Base(path)
	if info.AudioCodec != "" {
		musicMsg = fmt.Sprintf("已打开 %s · %s · %d Hz · %s",
			name, info.AudioCodec, info.SampleRate, fmtDur(src.Duration()))
	} else {
		musicMsg = fmt.Sprintf("已打开 %s（%s）", name, fmtDur(src.Duration()))
	}
	ui.RequestRedraw()
}

// openMusicDialog 选择本地音乐文件。
func openMusicDialog() {
	ui.PickFile("选择音乐", ui.MusicFilters, func(path string) {
		openMusicFile(path, true)
	})
}

// musicRow 播放列表一行（当前项高亮）。
func musicRow(i int, d musicDemo) *ui.Node {
	active := i == musicIndex
	bg := "rgba(255,255,255,0.03)"
	color := ui.ColTextDim
	idxColor := ui.ColTextMuted
	if active {
		bg = "rgba(245,197,24,0.12)"
		color = "#ffffff"
		idxColor = "#f5c518"
	}
	row := ui.Row(
		ui.Text(fmt.Sprint(i+1)).St(ui.St().W(18).Text(11, 700, idxColor)),
		ui.Column(
			ui.Text(d.title).St(ui.St().Text(12.5, 600, color)),
			ui.Text(fmt.Sprintf("%s · %d BPM", d.artist, d.bpm)).
				St(ui.St().Text(10.5, 400, ui.ColTextMuted)),
		).Grow(1).St(ui.St().GapOf(2)),
		ui.If(active, func() *ui.Node { return ui.Tag("播放中", "#f5c518") }),
	).St(ui.St().GapOf(10).PadXY(10, 8).R(9).BgC(bg))
	row.K(fmt.Sprintf("music-row-%d", i)).Tap(func() { openMusicDemo(i) })
	return row
}

// musicSpectrumCard 频谱实验室：大号柱状 + 镜像频谱与统计标签。
func musicSpectrumCard(src *ui.MusicSource) *ui.Node {
	if src == nil || src.Spectrum() == nil {
		return ui.Card(ui.Title("频谱特效", "打开音源后显示实时频谱"))
	}
	sp := src.Spectrum()
	bands := sp.Bands()
	var peak, sum float32
	for _, v := range bands {
		if v > peak {
			peak = v
		}
		sum += v
	}
	avg := float32(0)
	if len(bands) > 0 {
		avg = sum / float32(len(bands))
	}
	return ui.Card(
		ui.Title("频谱特效", "FFT（2048 点 Hann 窗）→ 40 个对数频段 → attack/release 平滑"),
		ui.MusicSpectrum("gmp-spec-big", sp, 40, "rgba(245,197,24,0.95)", "rgba(139,92,246,0.95)").
			St(ui.St().Wp(100).H(96)),
		ui.MusicMirrorSpectrum("gmp-spec-mirror", sp, 56, "rgba(34,211,238,0.92)", "rgba(129,140,248,0.92)").
			St(ui.St().Wp(100).H(72)),
		ui.Row(
			ui.Tag(fmt.Sprintf("峰值 %.2f", peak), "#fbbf24"),
			ui.Tag(fmt.Sprintf("均值 %.2f", avg), "#22d3ee"),
			ui.Tag(fmt.Sprintf("%d 段", len(bands)), "#a5b4fc"),
			ui.Tag(fmt.Sprintf("%d Hz", src.Info().SampleRate), "#34d399"),
		).St(ui.St().GapOf(8)),
		ui.Text("频谱数据来自正在播放的 PCM（与 waveOut 同一份数据流），暂停时能量自动衰减；"+
			"ui.MusicSpectrum / ui.MusicMirrorSpectrum 可嵌到任意界面，也可单独用 ui.NewSpectrum(64) 分析自己的 PCM。").
			St(ui.St().Text(12, 400, ui.ColTextDim).WrapOn()),
	)
}

func musicCase() *ui.Node {
	// 首次进入自动准备并播放第一首示例（同步生成 16s WAV，几十毫秒）
	if !musicAuto {
		musicAuto = true
		openMusicDemo(0)
	}
	src := musicSrc
	if src == nil {
		return ui.Column(
			pageHead("音乐播放器", "mp3 / flac / wav / m4a / aac / ogg / opus / wma / ape（ffmpeg 解码）+ 实时 FFT 频谱"),
			ui.Empty("暂无音源", "点下方「打开本地音乐」选择文件，或「播放示例」听程序化生成的曲目"),
			ui.Row(
				ui.Btn("打开本地音乐", ui.BtnPrimary, openMusicDialog),
				ui.Btn("播放示例", ui.BtnSecondary, func() { openMusicDemo(musicIndex) }),
			).St(ui.St().GapOf(10)),
			ui.Text(musicMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
		).St(ui.St().Padding(26).GapOf(16))
	}

	state := "暂停"
	if src.Playing() {
		state = "播放中"
	}
	rows := make([]*ui.Node, 0, len(musicDemos))
	for i, d := range musicDemos {
		rows = append(rows, musicRow(i, d))
	}
	info := src.Info()
	loopText := "循环关"
	if src.Loop() {
		loopText = "循环开"
	}

	return ui.Column(
		pageHead("音乐播放器", "mp3 / flac / wav / m4a / aac / ogg / opus / wma / ape（ffmpeg 解码）+ 实时 FFT 频谱特效"),
		ui.Row(
			ui.Tag(strings.ToUpper(info.Format), "#22d3ee"),
			ui.Tag("解码 "+ui.FormatBackend(info.Backend), "#38bdf8"),
			ui.Tag(info.AudioCodec, "#f472b6"),
			ui.Tag(fmt.Sprintf("%d Hz", info.SampleRate), "#34d399"),
			ui.Tag(fmtDur(src.Duration()), "#fbbf24"),
			ui.Tag(state, "#818cf8"),
			ui.Tag(fmt.Sprintf("倍速 %g×", src.Speed()), "#a5b4fc"),
			ui.Tag(loopText, "#94a3b8"),
		).St(ui.St().GapOf(8)),
		ui.MusicPlayer("gmp", src, ui.MusicPlayerOptions{
			Track: currentMusicTrack(),
			Bars:  40,
			Prev:  func() { openMusicDemo(musicIndex - 1) },
			Next:  func() { openMusicDemo(musicIndex + 1) },
		}),
		ui.Row(
			ui.Btn("打开本地音乐", ui.BtnPrimary, openMusicDialog),
			ui.Btn("播放 / 暂停", ui.BtnSecondary, func() {
				src.TogglePlay()
				ui.RequestRedraw()
			}),
			ui.Btn("回到开头", ui.BtnGhost, func() { src.Seek(0) }),
			ui.Btn("单曲循环", ui.BtnGhost, func() {
				src.SetLoop(!src.Loop())
				ui.RequestRedraw()
			}),
			ui.Btn("上一首", ui.BtnGhost, func() { openMusicDemo(musicIndex - 1) }),
			ui.Btn("下一首", ui.BtnGhost, func() { openMusicDemo(musicIndex + 1) }),
		).St(ui.St().GapOf(10)),
		ui.Text(musicMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
		ui.Row(
			ui.Card(
				ui.Title("播放列表", "点击切换（示例曲目为程序化合成，也可换成自己的文件）"),
				ui.Column(rows...).St(ui.St().GapOf(4)),
			).Grow(1),
			ui.Card(
				ui.Title("接入真实音乐", "OpenMusic → MusicPlayer"),
				ui.KeyValue("打开", "ui.OpenMusic(path) / ui.OpenMusicWith(path, cfg)"),
				ui.KeyValue("组件", "ui.MusicPlayer(key, src, opt)"),
				ui.KeyValue("频谱", "src.Spectrum() → ui.MusicSpectrum"),
				ui.KeyValue("解码", "ffmpeg → s16le PCM → waveOut"),
				ui.KeyValue("释放", "src.Close()（退出时释放进程与设备）"),
				ui.Text("音乐与视频互不影响：视频用 ui.NewMediaPlayer，音乐用 ui.OpenMusic。"+
					"两者都以音频设备时钟为播放位置，拖动进度条会一次性定位并重建解码窗口。").
					St(ui.St().Text(12, 400, ui.ColTextDim).WrapOn()),
			).Grow(1),
		).St(ui.St().GapOf(14)),
		musicSpectrumCard(src),
		ui.Card(
			ui.Title("操作方式", "鼠标点击即可"),
			ui.KeyValue("播放 / 暂停", "中间金色圆形按钮"),
			ui.KeyValue("上一首 / 下一首", "两侧跳曲按钮（或点播放列表）"),
			ui.KeyValue("进度", "点击或拖动进度条"),
			ui.KeyValue("音量", "喇叭图标静音 / 拖动音量条"),
			ui.KeyValue("循环 / 倍速", "右侧胶囊按钮（0.5x~2x 循环切换）"),
		),
	).St(ui.St().Padding(26).GapOf(16))
}
