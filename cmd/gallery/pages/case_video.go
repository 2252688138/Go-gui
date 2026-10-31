// Go-gui — gallery 页面拆分：视频播放器
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"fmt"
	"gogui/ui"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ---- 案例：视频播放器 ----

var (
	videoPl       *ui.Player
	videoEp       = 1
	videoDemoClip *ui.VideoClip
	// videoURL 默认播放地址（HLS 直链），启动后自动载入
	videoURL       = "https://v6.ppqrrs.com/wjv6/202610/25/4qUXBp64qg90/video/index.m3u8"
	videoAutoTried bool
	videoMsg       = "点击画面播放/暂停；拖动进度条跳转；聚焦画面后 ←/→ 快退快进 5 秒"
)

// demoSource 内置程序化片段（懒生成，多个演示播放器共用同一帧序列）。
func demoSource() *ui.VideoClip {
	if videoDemoClip == nil {
		videoDemoClip = demoClip()
	}
	return videoDemoClip
}

// videoHooks 给播放器挂上演示用回调。
func videoHooks(p *ui.Player) *ui.Player {
	p.OnPlay = func() { videoMsg = "开始播放" }
	p.OnPause = func() { videoMsg = "已暂停" }
	p.OnLoopChange = func(loop bool) {
		if loop {
			videoMsg = "已开启循环播放"
		} else {
			videoMsg = "已关闭循环播放"
		}
	}
	p.OnVolume = func(v float32, muted bool) {
		if muted {
			videoMsg = "已静音"
			return
		}
		videoMsg = fmt.Sprintf("音量 %.0f%%", v*100)
	}
	p.OnFullscreen = func() {
		// 播放器铺满窗口 + 窗口真全屏（覆盖桌面）
		if p.Fullscreen {
			ui.SetFocus("gvf-surface") // 让 ESC 交给播放器/窗口处理
			videoMsg = "已全屏：播放器铺满整个桌面（ESC 或再点全屏按钮退出）"
		} else {
			ui.SetFocus("")
			videoMsg = "已退出全屏"
		}
	}
	return p
}

// demoPlayer 首次访问时生成程序化片段（320×180 / 24fps / 4s）。
func demoPlayer() *ui.Player {
	if videoPl == nil {
		videoPl = videoHooks(ui.NewPlayer(demoSource()))
		videoPl.AutoHide = true
	}
	return videoPl
}

// demoClip 程序化生成一段循环动画片段（无需外部素材）。
func demoClip() *ui.VideoClip {
	const w, h, fps, secs = 320, 180, 24, 4
	total := fps * secs
	frames := make([]ui.VideoFrame, 0, total)
	dots := [][3]byte{{255, 214, 130}, {130, 226, 255}, {255, 140, 205}, {150, 245, 170}}
	for i := 0; i < total; i++ {
		t := float64(i) / fps
		pix := make([]byte, w*h*4)
		// 背景：极坐标波纹 + 中心辉光
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dx := float64(x) - w/2
				dy := float64(y) - h/2
				dist := math.Hypot(dx, dy) / (h * 0.62)
				if dist > 1 {
					dist = 1
				}
				ang := math.Atan2(dy, dx) + t*0.9
				wave := 0.5 + 0.5*math.Sin(ang*2+t*1.4)
				glow := 1 - dist
				o := (y*w + x) * 4
				pix[o] = byte(18 + 90*wave*glow)
				pix[o+1] = byte(20 + 70*glow)
				pix[o+2] = byte(46 + 130*glow)
				pix[o+3] = 255
			}
		}
		// 四个绕圈的运动圆点
		for k := 0; k < 4; k++ {
			ang := t*1.1 + float64(k)*math.Pi/2
			cx := int(float64(w)/2 + 96*math.Cos(ang))
			cy := int(float64(h)/2 + 58*math.Sin(ang))
			fillDisc(pix, w, h, cx, cy, 8, dots[k])
		}
		// 中心呼吸方块
		side := int(10 + 6*math.Sin(t*3))
		fillRect(pix, w, h, w/2-side/2, h/2-side/2, side, side, [3]byte{255, 255, 255})
		// 底部进度条（随帧增长的亮条）
		fillRect(pix, w, h, 0, h-5, w*(i+1)/total, 5, [3]byte{120, 132, 255})
		frames = append(frames, ui.VideoFrame{Pix: pix, W: w, H: h})
	}
	return ui.NewVideoClip(frames, fps, true)
}

// fillDisc 在 RGBA 缓冲里画实心圆。
func fillDisc(pix []byte, w, h, cx, cy, rad int, c [3]byte) {
	for y := cy - rad; y <= cy+rad; y++ {
		if y < 0 || y >= h {
			continue
		}
		for x := cx - rad; x <= cx+rad; x++ {
			if x < 0 || x >= w {
				continue
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy > rad*rad {
				continue
			}
			o := (y*w + x) * 4
			pix[o], pix[o+1], pix[o+2], pix[o+3] = c[0], c[1], c[2], 255
		}
	}
}

// fillRect 在 RGBA 缓冲里填充矩形（越界自动裁剪）。
func fillRect(pix []byte, w, h, x0, y0, rw, rh int, c [3]byte) {
	for y := y0; y < y0+rh; y++ {
		if y < 0 || y >= h {
			continue
		}
		for x := x0; x < x0+rw; x++ {
			if x < 0 || x >= w {
				continue
			}
			o := (y*w + x) * 4
			pix[o], pix[o+1], pix[o+2], pix[o+3] = c[0], c[1], c[2], 255
		}
	}
}

// openVideoGIF 选择本地 GIF 作为新片源。
func openVideoGIF() {
	ui.PickFile("选择 GIF 动图", "GIF|*.gif|所有文件|*.*", func(path string) {
		clip, err := ui.LoadGIF(path)
		if err != nil {
			videoMsg = "载入失败：" + err.Error()
			ui.RequestRedraw()
			return
		}
		p := demoPlayer()
		p.SetClip(clip)
		p.Play()
		videoMsg = fmt.Sprintf("已载入 GIF（%d 帧）", clip.FrameCount())
		ui.RequestRedraw()
	})
}

// useBuiltinClip 切回内置合成片段（网络失败或想跑离线演示时）。
func useBuiltinClip() {
	old := videoPl
	p := videoHooks(ui.NewPlayer(demoSource()))
	p.AutoHide = true
	p.Fullscreen = false
	videoPl = p
	if old != nil && old != p {
		old.Close()
	}
	p.Play()
	videoMsg = "已切换到内置片段（320×180 / 24fps）"
	ui.RequestRedraw()
}

// openVideoFile 选择本地视频（mp4 / mkv / mov / avi / webm / flv / ts / m3u8）。
func openVideoFile() {
	ui.PickFile("选择视频", ui.MediaFilters, func(path string) {
		loadMedia(path)
	})
}

// loadMedia 用 ffmpeg 解码打开媒体：ffprobe 探测在后台 goroutine，完成后回 UI 线程换源。
func loadMedia(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	videoMsg = "正在载入 " + path + " …"
	ui.RequestRedraw()
	cur := ui.Current()
	// wasm 单线程：平台包会直接同步执行；桌面端放后台 goroutine 避免阻塞界面
	ui.RunAsync(func() {
		p, err := ui.NewMediaPlayerWith(path, ui.MediaConfig{MaxWidth: 1280, MaxHeight: 720})
		apply := func() {
			if err != nil {
				videoMsg = "载入失败：" + err.Error()
				ui.RequestRedraw()
				return
			}
			old := videoPl
			p.Fullscreen = false
			p.AutoHide = true
			videoHooks(p)
			videoPl = p
			if old != nil && old != p {
				old.Close()
			}
			p.Play()
			info := p.Info()
			w, h := p.Size()
			audio := "无音频"
			if info.HasAudio {
				audio = info.AudioCodec
				if !p.HasAudio() {
					audio += "（无音频设备）"
				}
			}
			videoMsg = fmt.Sprintf("已载入 %s · %s %s · %d×%d · %s · %.2f fps · %s",
				filepath.Base(path), strings.ToUpper(info.Format), info.VideoCodec,
				w, h, audio, info.FPS, fmtDur(info.Duration))
			ui.RequestRedraw()
		}
		if cur != nil {
			cur.Post(apply)
		} else {
			apply()
		}
	})
}

// playVideoURL 播放输入框里的直链（mp4 / m3u8 等）。
func playVideoURL() {
	u := strings.TrimSpace(videoURL)
	if u == "" {
		videoMsg = "请先粘贴视频直链（mp4 / m3u8 等）"
		ui.RequestRedraw()
		return
	}
	loadMedia(u)
}

// fmtDur 时长文本。
func fmtDur(d time.Duration) string {
	if d <= 0 {
		return "时长未知"
	}
	t := int(d.Seconds())
	if t >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", t/3600, t%3600/60, t%60)
	}
	return fmt.Sprintf("%d:%02d", t/60, t%60)
}

// videoFitDemo 同一片段的三种填充方式（只显示画面，不带控制条）。
func videoFitDemo(label string, fit ui.VideoFit) *ui.Node {
	q := videoFitPlayer(fit)
	q.Fit = fit
	q.HideControls = true
	q.SetPos(demoPlayer().Pos())
	return ui.Column(
		ui.Row(
			ui.Text(label).St(ui.St().Text(11.5, 600, ui.ColTextDim)),
			ui.Spacer(),
			ui.Text(map[ui.VideoFit]string{ui.FitContain: "留边", ui.FitCover: "裁切", ui.FitFill: "拉伸"}[fit]).
				St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().Wp(100)).AlignTo(ui.AlignCenter),
		ui.Video(q).St(ui.St().Wp(100).H(110).R(10).Clipped().BgC("#000000")),
	).St(ui.St().GrowTo(1).GapOf(6))
}

// videoFitPlayer 三种填充方式的演示播放器（按 Fit 缓存）。
// 旧实现每帧 ui.NewPlayer(demoSource()) 新建播放器，而 App 里每个播放器都带一张
// 帧纹理且从不回收，纹理与 Player 随帧累积——这是示例页内存涨到 GB 级的主因。
var videoFitPlays = map[ui.VideoFit]*ui.Player{}

func videoFitPlayer(fit ui.VideoFit) *ui.Player {
	if q := videoFitPlays[fit]; q != nil {
		return q
	}
	q := ui.NewPlayer(demoSource())
	q.Fit = fit
	q.HideControls = true
	videoFitPlays[fit] = q
	return q
}

// backendOf 播放器当前使用的解码后端（ffmpeg / system）。
func backendOf(p *ui.Player) string {
	if p != nil {
		if b := p.Info().Backend; b != "" {
			return b
		}
	}
	return ui.BackendName()
}

// videoMetaTags 顶部信息标签：内置片段显示帧数，媒体源显示容器 / 编码 / 音频 / 时长。
func videoMetaTags(p *ui.Player, state string) *ui.Node {
	w, h := p.Size()
	tags := []*ui.Node{ui.Tag(fmt.Sprintf("尺寸 %d×%d", w, h), "#a5b4fc")}
	info := p.Info()
	switch {
	case info.HasVideo:
		name := strings.ToUpper(info.Format)
		if name == "" {
			name = "视频"
		}
		tags = append(tags,
			ui.Tag(name+" · "+info.VideoCodec, "#22d3ee"),
			ui.Tag(fmt.Sprintf("%.2f fps", info.FPS), "#34d399"),
		)
		if info.HasAudio {
			tags = append(tags, ui.Tag("音频 "+info.AudioCodec, "#f472b6"))
		} else {
			tags = append(tags, ui.Tag("无声", "#94a3b8"))
		}
	case p.Clip != nil:
		tags = append(tags,
			ui.Tag("内置片段", "#22d3ee"),
			ui.Tag(strconv.Itoa(p.Clip.FrameCount())+" 帧", "#34d399"),
		)
	}
	tags = append(tags,
		ui.Tag("解码 "+ui.FormatBackend(backendOf(p)), "#38bdf8"),
		ui.Tag(fmtDur(p.Duration()), "#fbbf24"),
		ui.Tag(state, "#818cf8"),
	)
	return ui.Row(tags...).St(ui.St().GapOf(8))
}

func videoCase() *ui.Node {
	// 首次进入时自动载入默认播放地址（m3u8 直链），失败仍会保留内置片段
	if !videoAutoTried && strings.TrimSpace(videoURL) != "" {
		videoAutoTried = true
		loadMedia(videoURL)
	}
	p := demoPlayer()
	// 字幕与上/下一集：控制条只在回调非空时显示跳集按钮
	p.Subtitle = fmt.Sprintf("第 %d 集 · 正常人才干不出那种事", videoEp)
	p.OnPrev = func() {
		if videoEp > 1 {
			videoEp--
		}
		p.SetPos(0)
		p.Play()
		videoMsg = fmt.Sprintf("已切到第 %d 集（示例：同一个片段）", videoEp)
		ui.RequestRedraw()
	}
	p.OnNext = func() {
		videoEp++
		p.SetPos(0)
		p.Play()
		videoMsg = fmt.Sprintf("已切到第 %d 集（示例：同一个片段）", videoEp)
		ui.RequestRedraw()
	}
	state := "暂停"
	if p.PlayingNow() {
		state = "播放中"
	}
	posText := fmt.Sprintf("%.2f / %.2f s",
		float64(p.Pos().Milliseconds())/1000, float64(p.Duration().Milliseconds())/1000)
	loopText := "关闭"
	if p.Loop {
		loopText = "开"
	}

	return ui.Column(
		pageHead("视频播放器", "mp4 / mkv / mov / avi / webm / flv / ts / m3u8 均可播放（ffmpeg 解码）+ 逐帧纹理、拖动进度、倍速、音量、循环、全屏"),
		videoMetaTags(p, state),
		// 全屏时由根层铺满播放，这里不再重复绘制
		ui.If(!p.Fullscreen, func() *ui.Node {
			return ui.VideoPlayer("gv", p).St(ui.St().H(380).BgC("#000000"))
		}),
		ui.Row(
			ui.Btn("◀ 上一集", ui.BtnSecondary, func() { p.OnPrev() }),
			ui.Spacer(),
			ui.Tag(fmt.Sprintf("第 %d 集 · 实时源 · HLS 播放 · HD 国语", videoEp), "#a5b4fc"),
			ui.Spacer(),
			ui.Btn("下一集 ▶", ui.BtnSecondary, func() { p.OnNext() }),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Text(videoMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
		ui.Row(
			ui.Btn("打开本地视频", ui.BtnPrimary, openVideoFile),
			ui.Btn("打开 GIF 动图", ui.BtnSecondary, openVideoGIF),
			ui.Btn("内置片段", ui.BtnGhost, useBuiltinClip),
			ui.Btn("暂停 / 播放", ui.BtnSecondary, func() {
				p.TogglePlay()
				ui.RequestRedraw()
			}),
			ui.Btn("回到第一帧", ui.BtnGhost, func() { p.SetPos(0) }),
			ui.Btn("旋转 90°", ui.BtnGhost, func() { p.RotateBy(1) }),
			ui.Btn(map[bool]string{true: "解锁画面", false: "锁定画面"}[p.Locked], ui.BtnGhost, func() { p.ToggleLock() }),
			ui.Btn("全屏演示", ui.BtnGhost, func() {
				p.Fullscreen = true
				if cur := ui.Current(); cur != nil {
					cur.SetFullscreen(true)
				}
				ui.SetFocus("gvf-surface") // 让 ESC 走播放器
				videoMsg = "已全屏：播放器铺满整个桌面（ESC 或右上角按钮退出）"
				ui.RequestRedraw()
			}),
		).St(ui.St().GapOf(10)),
		ui.Row(
			ui.Box(ui.Input("video-url", videoURL, "粘贴视频直链（https://… .mp4 / .m3u8）", func(v string) { videoURL = v })).Grow(1),
			ui.Btn("播放链接", ui.BtnSecondary, playVideoURL),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Card(
				ui.Title("支持格式", "全部交给 ffmpeg demux / decode，不依赖系统解码器"),
				ui.Row(ui.Tag("mp4", "#a5b4fc"), ui.Tag("mkv", "#22d3ee"), ui.Tag("mov", "#f472b6"),
					ui.Tag("avi", "#34d399"), ui.Tag("webm", "#fbbf24"), ui.Tag("flv", "#fb923c")).St(ui.St().GapOf(6)),
				ui.Row(ui.Tag("ts", "#c084fc"), ui.Tag("m2ts", "#818cf8"), ui.Tag("m3u8 / HLS", "#38bdf8"),
					ui.Tag("远端直链", "#94a3b8")).St(ui.St().GapOf(6)),
				ui.KeyValue("解码", "ffmpeg 子进程 → rawvideo RGBA"),
				ui.KeyValue("声音", "waveOut 输出，以音频时钟同步画面"),
				ui.KeyValue("分辨率", "跟随源尺寸（可用 MediaConfig 限高）"),
			).St(ui.St().GrowTo(1)),
			ui.Card(
				ui.Title("操作方式", "鼠标 + 键盘（画面获焦后）"),
				ui.KeyValue("播放 / 暂停", "点击画面 / 控制条按钮 / Space"),
				ui.KeyValue("进度跳转", "点击或拖动进度条"),
				ui.KeyValue("快退 / 快进", "← / →（各 5 秒）"),
				ui.KeyValue("音量", "↑ / ↓ 或拖动音量条"),
				ui.KeyValue("倍速 / 循环 / 全屏", "控制条右侧胶囊按钮"),
			).St(ui.St().GrowTo(1)),
		).St(ui.St().GapOf(14)),
		ui.Row(
			ui.Card(
				ui.Title("实时状态", "每帧从 Player 读取"),
				ui.KeyValue("当前帧", strconv.Itoa(p.FrameIndex())),
				ui.KeyValue("播放位置", posText),
				ui.KeyValue("播放速率", fmt.Sprintf("%g×", p.SpeedValue())),
				ui.KeyValue("循环", loopText),
				ui.KeyValue("旋转", fmt.Sprintf("%d°", p.RotationDeg())),
				ui.KeyValue("锁定", map[bool]string{true: "已锁定", false: "未锁"}[p.Locked]),
				ui.KeyValue("音量", fmt.Sprintf("%.0f%%", p.Volume*100)),
			).St(ui.St().GrowTo(1)),
		).St(ui.St().GapOf(14)),
		ui.Card(
			ui.Title("画面填充方式", "Player.Fit：Contain 留边 / Cover 裁切 / Fill 拉伸"),
			ui.Row(
				videoFitDemo("Contain", ui.FitContain),
				videoFitDemo("Cover", ui.FitCover),
				videoFitDemo("Fill", ui.FitFill),
			).St(ui.St().GapOf(12)),
		),
		ui.Card(
			ui.Title("接入真实视频", "三种帧源：内存帧序列 / 外部解码器 / 内置 ffmpeg"),
			ui.Text("ui.LoadGIF(path) → VideoClip；ui.LoadImageSequence(paths, fps, loop) → VideoClip；"+
				"视频文件直接 ui.NewMediaPlayer(path)（或 ui.OpenMedia(path) + ui.NewPlayerOf），"+
				"内部用 ffprobe 探测 + ffmpeg 解 rawvideo RGBA 帧、s16le 音频经 waveOut 输出，"+
				"播放位置以音频设备时钟为基准。需 PATH 中有 ffmpeg/ffprobe；播放完调 Player.Close 释放进程与设备。").
				St(ui.St().Text(12, 400, ui.ColTextDim).WrapOn()),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

// VideoLayer 播放器全屏浮层：在窗口根 Stack 里铺满，不改系统窗口。
func VideoLayer() *ui.Node {
	// 视频已并入「媒体播放」分类页（mediaTab = video），全屏浮层仍只在视频页签下生效。
	return ui.If(Current == "media" && mediaTab == "video" && videoPl != nil && videoPl.Fullscreen, func() *ui.Node {
		return ui.Stack(
			ui.VideoPlayer("gvf", videoPl),
			ui.Row(
				ui.Text("退出全屏（Esc）").St(ui.St().Text(11.5, 600, "#ffffff").
					PadXY(12, 6).R(7).BgC("rgba(8,12,20,0.72)").
					Stroke(1, "rgba(255,255,255,0.18)")),
			).St(ui.St().AbsTL(16, 14)).Tap(func() {
				videoPl.Fullscreen = false
				ui.SetFocus("")
				if cur := ui.Current(); cur != nil {
					cur.SetFullscreen(false)
				}
				videoMsg = "已退出全屏"
				ui.RequestRedraw()
			}),
		).St(ui.St().AbsFill())
	})
}
