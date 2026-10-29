//go:build windows

package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"sync"
	"time"
)

// 视频播放：GoGUI 不做编解码，只负责「帧 → 纹理 → 画面 + 播放控制」。
// 帧来源有两类：
//  1. VideoClip：内存中的帧序列（GIF、图片序列、解码器逐帧结果）；
//  2. VideoSource：外部解码器按播放位置取帧（FFmpeg / MediaFoundation 等）。
// 播放状态由业务持有的 Player 保存，VideoPlayer 每帧推进并渲染。

// VideoFrame 一帧画面：RGBA8 非预乘像素（行优先，长度 = W*H*4），Delay 为本帧停留时长。
type VideoFrame struct {
	Pix   []byte
	W, H  int
	Delay time.Duration
}

// Valid 画面是否可用（尺寸合法且像素足够）。
func (f *VideoFrame) Valid() bool {
	return f != nil && f.W > 0 && f.H > 0 && len(f.Pix) >= f.W*f.H*4
}

// VideoClip 一段可播放的帧序列：GIF、图片序列、逐帧解码结果都归一到这个载体。
// 播放位置到帧号的映射由 Delay 累加成的时间轴决定（起点表缓存 + 二分查找）。
type VideoClip struct {
	Frames []VideoFrame
	// FPS 在 Delay 全为 0 时使用的帧率（默认 24）。
	FPS  float64
	Loop bool

	once   sync.Once
	starts []time.Duration
	total  time.Duration
	w, h   int
}

// NewVideoClip 用等间隔帧构造片段（fps <= 0 时按 24 帧/秒处理）。
func NewVideoClip(frames []VideoFrame, fps float64, loop bool) *VideoClip {
	return &VideoClip{Frames: frames, FPS: fps, Loop: loop}
}

// FrameDelay 返回第 i 帧的停留时长（Delay <= 0 时按 FPS 折算）。
func (c *VideoClip) FrameDelay(i int) time.Duration {
	if c == nil || i < 0 || i >= len(c.Frames) {
		return 0
	}
	if d := c.Frames[i].Delay; d > 0 {
		return d
	}
	fps := c.FPS
	if fps <= 0 {
		fps = 24
	}
	return time.Duration(float64(time.Second) / fps)
}

// build 构建时间轴缓存（起点表 + 总时长 + 画面尺寸）。
func (c *VideoClip) build() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		n := len(c.Frames)
		c.starts = make([]time.Duration, n+1)
		for i := 0; i < n; i++ {
			c.starts[i] = c.total
			c.total += c.FrameDelay(i)
			if f := c.Frames[i]; f.W > c.w {
				c.w, c.h = f.W, f.H
			} else if c.w == 0 && f.H > 0 {
				c.w, c.h = f.W, f.H
			}
		}
		c.starts[n] = c.total
	})
}

// Duration 片段总时长。
func (c *VideoClip) Duration() time.Duration {
	if c == nil {
		return 0
	}
	c.build()
	return c.total
}

// Size 画面像素尺寸（取首帧，帧尺寸不一致时取最大）。
func (c *VideoClip) Size() (int, int) {
	if c == nil {
		return 0, 0
	}
	c.build()
	return c.w, c.h
}

// FrameCount 帧数。
func (c *VideoClip) FrameCount() int {
	if c == nil {
		return 0
	}
	return len(c.Frames)
}

// FrameStart 第 i 帧的起始时间。
func (c *VideoClip) FrameStart(i int) time.Duration {
	if c == nil {
		return 0
	}
	c.build()
	if i < 0 || i >= len(c.starts) {
		return 0
	}
	return c.starts[i]
}

// FrameIndexAt 返回时间点 pos 对应的帧号（空片段返回 -1）。
func (c *VideoClip) FrameIndexAt(pos time.Duration) int {
	n := c.FrameCount()
	if n == 0 {
		return -1
	}
	c.build()
	if pos < 0 {
		pos = 0
	}
	if c.total <= 0 {
		return 0
	}
	if pos >= c.total {
		return n - 1
	}
	// 二分：找最大的 i 使 starts[i] <= pos
	lo, hi := 0, n-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if c.starts[mid] <= pos {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// LoadGIF 解码 GIF 的全部画面为片段（按 disposal 规则合成为完整 RGBA 帧）。
func LoadGIF(path string) (*VideoClip, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, err
	}
	return GIFToClip(g)
}

// GIFToClip 把已解码的 GIF 转成片段。
func GIFToClip(g *gif.GIF) (*VideoClip, error) {
	if g == nil || len(g.Image) == 0 {
		return nil, errors.New("gif 没有画面")
	}
	w, h := g.Config.Width, g.Config.Height
	if w <= 0 || h <= 0 {
		b := g.Image[0].Bounds()
		w, h = b.Dx(), b.Dy()
	}
	if w <= 0 || h <= 0 {
		return nil, errors.New("gif 尺寸非法")
	}

	// GIF 每帧只记录变化区域，需要在前一帧基础上合成。
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	transparent := image.NewUniform(color.RGBA{0, 0, 0, 0})

	frames := make([]VideoFrame, 0, len(g.Image))
	for i, src := range g.Image {
		if i > 0 && disposal(g, i-1) == gif.DisposalBackground {
			prev := g.Image[i-1].Bounds()
			draw.Draw(canvas, prev, transparent, prev.Min, draw.Src)
		}
		draw.Draw(canvas, src.Bounds(), src, src.Bounds().Min, draw.Over)

		pix := make([]byte, w*h*4)
		copy(pix, canvas.Pix)
		var delay time.Duration
		if i < len(g.Delay) && g.Delay[i] > 0 {
			delay = time.Duration(g.Delay[i]) * 10 * time.Millisecond
		}
		frames = append(frames, VideoFrame{Pix: pix, W: w, H: h, Delay: delay})
	}
	return &VideoClip{Frames: frames, FPS: 24, Loop: true}, nil
}

func disposal(g *gif.GIF, i int) byte {
	if g == nil || i < 0 || i >= len(g.Disposal) {
		return gif.DisposalNone
	}
	return g.Disposal[i]
}

// LoadImageSequence 把多张图片按固定帧率组成片段（帧序按传入顺序）。
func LoadImageSequence(paths []string, fps float64, loop bool) (*VideoClip, error) {
	if len(paths) == 0 {
		return nil, errors.New("图片序列为空")
	}
	frames := make([]VideoFrame, 0, len(paths))
	for _, p := range paths {
		img, err := DecodeImage(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		b := img.Bounds()
		frames = append(frames, VideoFrame{Pix: img.Pix, W: b.Dx(), H: b.Dy()})
	}
	return &VideoClip{Frames: frames, FPS: fps, Loop: loop}, nil
}

// VideoFit 画面填充方式。
type VideoFit uint8

const (
	// FitContain 等比缩放完整显示（多余部分留黑边）。
	FitContain VideoFit = iota
	// FitCover 等比缩放铺满（超出部分裁切）。
	FitCover
	// FitFill 拉伸铺满（不保持比例）。
	FitFill
)

// VideoSource 外部解码器接口（FFmpeg / MediaFoundation / 自研解码器）。
// 播放器每帧按当前播放位置取画面；同一位置可能被连续请求，实现应自行缓存。
type VideoSource interface {
	// Size 画面像素尺寸。
	Size() (int, int)
	// Duration 总时长（未知返回 0，此时进度条按未知处理）。
	Duration() time.Duration
	// Frame 返回时间点 pos 的画面；返回 nil, nil 表示该位置暂无画面（保留上一帧）。
	Frame(pos time.Duration) (*VideoFrame, error)
}

// Player 播放状态（业务持有并跨帧复用；由 VideoPlayer 每帧推进）。
type Player struct {
	// Clip 内存帧序列；与 Source 二选一（同时存在时优先 Clip）。
	Clip *VideoClip
	// Source 外部帧来源（流式解码）。
	Source VideoSource

	Playing bool
	Loop    bool
	// Speed 播放速率（<=0 视为 1）。
	Speed float32
	// Volume 音量 0~1；渲染层不发声，由业务在 OnVolume 中消费。
	Volume float32
	Muted  bool
	Fit    VideoFit
	// StepMode 逐帧模式：按帧序号 + 毫秒精确定位。
	StepMode bool
	// Rotation 画面旋转（0~3，单位 90° 顺时针，控制条左侧旋转按钮修改）。
	Rotation int
	// Locked 锁定画面：点击画面不切播放、快捷键（空格/方向键）失效，避免误触。
	Locked bool
	// AutoHide 播放中自动隐藏控制条（鼠标移动/暂停时显示）。
	AutoHide bool
	// HideControls 不显示控制条（纯画面）。
	HideControls bool
	// Fullscreen 播放器铺满窗口（VideoPlayer 需放在根 Stack 中才会铺满整窗）。
	Fullscreen bool

	// ShowControls 之外的展示项
	// Subtitle 底部字幕（空则不显示；中间的「·」等符号照常渲染）
	Subtitle string

	// 回调
	OnPlay       func()
	OnPause      func()
	OnSeek       func(time.Duration)
	OnVolume     func(v float32, muted bool)
	OnLoopChange func(loop bool)
	OnFullscreen func()
	// OnPrev / OnNext 上一集 / 下一集（非 nil 时控制条显示跳集按钮）
	OnPrev func()
	OnNext func()

	pos      time.Duration
	lastTick float32
	started  bool
	scrub    bool
	err      error
	frame    *VideoFrame
	frameIdx int
	frameSig uint64
	rect     rectBox // 上一帧画面矩形（控制条自动隐藏判定用）
}

// rectBox 简单矩形（画面命中与自动隐藏判定用）。
type rectBox struct{ X, Y, W, H float32 }

func (r rectBox) contains(x, y float32) bool {
	return r.W > 0 && r.H > 0 && x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H
}

// NewPlayer 用内存帧序列创建播放器（默认速率 1，循环设置继承片段）。
func NewPlayer(clip *VideoClip) *Player {
	p := &Player{Clip: clip, Speed: 1, Volume: 1, Fit: FitContain}
	if clip != nil {
		p.Loop = clip.Loop
	}
	return p
}

// NewPlayerOf 用外部帧来源创建播放器。
func NewPlayerOf(src VideoSource) *Player {
	return &Player{Source: src, Speed: 1, Volume: 1, Fit: FitContain}
}

// NewMediaPlayer 打开媒体文件/链接（mp4 / mkv / mov / avi / webm / flv / ts / m3u8 等）
// 并返回可播放的播放器。解码由外部 ffmpeg / ffprobe 完成（需在 PATH 中），
// 画面跟随源尺寸解码，声音经 waveOut 输出；用完后调用 Player.Close 释放进程与设备。
func NewMediaPlayer(path string) (*Player, error) {
	return NewMediaPlayerWith(path, MediaConfig{})
}

// NewMediaPlayerWith 按配置打开媒体并创建播放器。
func NewMediaPlayerWith(path string, cfg MediaConfig) (*Player, error) {
	src, err := OpenMediaWith(path, cfg)
	if err != nil {
		return nil, err
	}
	p := NewPlayerOf(src)
	// 默认循环关闭：长视频播完即停（短片源可在业务里 SetLoop(true)）
	p.Loop = false
	p.mediaVolume()
	return p, nil
}

// OpenMediaInto 在已有播放器上打开新媒体（旧解码源自动关闭）。
func OpenMediaInto(p *Player, path string) error {
	if p == nil {
		return errors.New("播放器为空")
	}
	src, err := OpenMedia(path)
	if err != nil {
		return err
	}
	p.Close()
	p.Source = src
	p.Clip = nil
	p.pos = 0
	p.err = nil
	p.frame = nil
	p.frameIdx = -1
	p.frameSig = 0
	p.mediaVolume()
	RequestRedraw()
	return nil
}

// mediaPlay / mediaVolume / mediaSpeed 把播放状态同步给解码源（若支持）。
func (p *Player) mediaPlay() {
	if m, ok := p.Source.(interface{ MediaPlay(bool) }); ok && m != nil {
		m.MediaPlay(p.Playing)
	}
}

func (p *Player) mediaVolume() {
	if p == nil {
		return
	}
	if m, ok := p.Source.(interface{ MediaVolume(float32, bool) }); ok && m != nil {
		m.MediaVolume(p.Volume, p.Muted)
	}
}

func (p *Player) mediaSpeed() {
	if p == nil {
		return
	}
	if m, ok := p.Source.(interface{ MediaSpeed(float32) }); ok && m != nil {
		m.MediaSpeed(p.SpeedValue())
	}
}

// Close 释放外部解码源（ffmpeg 子进程 / 音频设备），播放器本身仍可继续使用内存片段。
func (p *Player) Close() {
	if p == nil {
		return
	}
	if c, ok := p.Source.(interface{ Close() }); ok && c != nil {
		c.Close()
	}
	p.Source = nil
	p.frame = nil
	p.frameSig = 0
}

// HasAudio 当前解码源是否成功开启声音输出。
func (p *Player) HasAudio() bool {
	if p == nil {
		return false
	}
	if a, ok := p.Source.(interface{ HasAudio() bool }); ok && a != nil {
		return a.HasAudio()
	}
	return false
}

// Info 媒体探测信息（非媒体源返回零值）。
func (p *Player) Info() MediaInfo {
	if p == nil {
		return MediaInfo{}
	}
	if s, ok := p.Source.(*MediaSource); ok && s != nil {
		return s.Info()
	}
	return MediaInfo{}
}

// SetClip 切换片源（重置播放位置，保持播放状态与循环设置）。
func (p *Player) SetClip(c *VideoClip) {
	if p == nil {
		return
	}
	// 切换片源时释放外部解码器（ffmpeg 进程 / 音频设备）
	if p.Clip != c {
		if cl, ok := p.Source.(interface{ Close() }); ok && cl != nil {
			cl.Close()
		}
		p.Source = nil
	}
	p.Clip = c
	p.pos = 0
	p.err = nil
	p.frame = nil
	p.frameIdx = -1
	p.frameSig = 0
	if c != nil {
		p.Loop = c.Loop
		// 只继承一次：之后以 Player.Loop 为准，避免 Clip.Loop 绕过循环开关
	}
	RequestRedraw()
}

// Pos 当前播放位置。
func (p *Player) Pos() time.Duration {
	if p == nil {
		return 0
	}
	return p.pos
}

// Duration 总时长（未知返回 0）。
func (p *Player) Duration() time.Duration {
	if p == nil {
		return 0
	}
	if p.Clip != nil {
		return p.Clip.Duration()
	}
	if p.Source != nil {
		return p.Source.Duration()
	}
	return 0
}

// Size 画面像素尺寸。
func (p *Player) Size() (int, int) {
	if p == nil {
		return 0, 0
	}
	if p.Clip != nil {
		return p.Clip.Size()
	}
	if p.Source != nil {
		return p.Source.Size()
	}
	return 0, 0
}

// Progress 播放进度 0~1（时长未知返回 0）。
func (p *Player) Progress() float32 {
	d := p.Duration()
	if d <= 0 {
		return 0
	}
	v := float32(p.pos) / float32(d)
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return v
}

// mediaScrub 把拖动状态同步给解码源：拖动中不重启 ffmpeg，松手后一次性定位（大幅提升拖动响应）。
func (p *Player) mediaScrub() {
	if p == nil {
		return
	}
	if m, ok := p.Source.(interface{ MediaScrub(bool) }); ok && m != nil {
		m.MediaScrub(p.scrub)
	}
}

// SetRotation 设置画面旋转（自动折到 0~3 的 90° 份数），返回生效值。
func (p *Player) SetRotation(q int) int {
	if p == nil {
		return 0
	}
	p.Rotation = ((q % 4) + 4) % 4
	RequestRedraw()
	return p.Rotation
}

// RotateBy 按 90° 份数旋转（可正可负，如 -1 表示逆时针 90°）。
func (p *Player) RotateBy(delta int) int {
	if p == nil {
		return 0
	}
	return p.SetRotation(p.Rotation + delta)
}

// RotationDeg 当前旋转角度（度）。
func (p *Player) RotationDeg() int {
	if p == nil {
		return 0
	}
	return (p.Rotation & 3) * 90
}

// SetLocked 锁定/解锁画面（锁定后点击与快捷键不再改播放状态）。
func (p *Player) SetLocked(v bool) {
	if p == nil {
		return
	}
	p.Locked = v
	RequestRedraw()
}

// ToggleLock 切换锁定状态。
func (p *Player) ToggleLock() {
	if p == nil {
		return
	}
	p.SetLocked(!p.Locked)
}

// Err 最近一次取帧错误。
func (p *Player) Err() error {
	if p == nil {
		return nil
	}
	return p.err
}

// PlayingNow 是否正在播放。
func (p *Player) PlayingNow() bool { return p != nil && p.Playing }

// SpeedValue 生效的播放速率（<=0 视为 1）。
func (p *Player) SpeedValue() float32 {
	if p == nil || p.Speed <= 0 {
		return 1
	}
	return p.Speed
}

// loopOn 生效的循环设置（Player.Loop 是唯一开关，SetLoop 能真正关掉循环）。
func (p *Player) loopOn() bool {
	return p != nil && p.Loop
}

// Play 开始播放；已播放到末尾时从头开始。
func (p *Player) Play() {
	if p == nil {
		return
	}
	if d := p.Duration(); d > 0 && p.pos >= d {
		p.pos = 0
	}
	if !p.Playing {
		p.Playing = true
		p.started = false
		p.mediaPlay()
		if p.OnPlay != nil {
			p.OnPlay()
		}
	}
	RequestRedraw()
}

// Pause 暂停。
func (p *Player) Pause() {
	if p == nil || !p.Playing {
		return
	}
	p.Playing = false
	p.mediaPlay()
	if p.OnPause != nil {
		p.OnPause()
	}
	RequestRedraw()
}

// TogglePlay 播放/暂停切换。
func (p *Player) TogglePlay() {
	if p == nil {
		return
	}
	if p.Playing {
		p.Pause()
	} else {
		p.Play()
	}
}

// SetPos 跳转到指定播放位置（自动夹到 [0, 时长]）。
func (p *Player) SetPos(d time.Duration) {
	if p == nil {
		return
	}
	total := p.Duration()
	if d < 0 {
		d = 0
	}
	if total > 0 && d > total {
		d = total
	}
	p.pos = d
	p.resolveFrame()
	if p.OnSeek != nil {
		p.OnSeek(d)
	}
	RequestRedraw()
}

// SeekBy 相对跳转（负值快退）。
func (p *Player) SeekBy(d time.Duration) {
	if p == nil {
		return
	}
	p.SetPos(p.pos + d)
}

// SeekRatio 按比例跳转（0~1，进度条点击/拖动使用）。
func (p *Player) SeekRatio(v float32) {
	if p == nil {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	d := p.Duration()
	if d <= 0 {
		return
	}
	p.SetPos(time.Duration(float32(d) * v))
}

// SetSpeed 设置播放速率（夹到 0.25~4）。
func (p *Player) SetSpeed(v float32) {
	if p == nil {
		return
	}
	if v < 0.25 {
		v = 0.25
	}
	if v > 4 {
		v = 4
	}
	p.Speed = v
	p.mediaSpeed()
	RequestRedraw()
}

// SetVolume 设置音量 0~1（音量 0 自动静音）。
func (p *Player) SetVolume(v float32) {
	if p == nil {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.Volume = v
	p.Muted = v <= 0
	p.mediaVolume()
	if p.OnVolume != nil {
		p.OnVolume(p.Volume, p.Muted)
	}
	RequestRedraw()
}

// ToggleMute 静音切换（保留静音前音量）。
func (p *Player) ToggleMute() {
	if p == nil {
		return
	}
	p.Muted = !p.Muted
	p.mediaVolume()
	if p.OnVolume != nil {
		p.OnVolume(p.Volume, p.Muted)
	}
	RequestRedraw()
}

// SetLoop 设置循环播放。
func (p *Player) SetLoop(v bool) {
	if p == nil {
		return
	}
	p.Loop = v
	if p.OnLoopChange != nil {
		p.OnLoopChange(v)
	}
	RequestRedraw()
}

// Frame 当前画面（可能为 nil）。
func (p *Player) Frame() *VideoFrame {
	if p == nil {
		return nil
	}
	return p.frame
}

// FrameIndex 当前帧号（Clip 有效，否则 -1）。
func (p *Player) FrameIndex() int {
	if p == nil || p.Clip == nil {
		return -1
	}
	return p.frameIdx
}

// advance 按真实时间推进播放位置并解析当前帧（VideoPlayer 每帧调用）。
func (p *Player) advance(now float32) {
	if p == nil {
		return
	}
	dt := float32(0)
	if p.started {
		dt = now - p.lastTick
	}
	p.started = true
	p.lastTick = now
	if dt < 0 {
		dt = 0
	}
	if dt > 0.25 {
		// 掉帧或窗口挂起后不一次性跳过太多内容
		dt = 0.25
	}

	total := p.Duration()
	if p.Playing && !p.scrub {
		// 有声音时以音频设备时钟为准（A/V 同步），否则按墙钟推进
		if c, ok := p.Source.(interface {
			MediaClock() (time.Duration, bool)
		}); ok && c != nil {
			if cp, valid := c.MediaClock(); valid {
				p.pos = cp
			} else {
				p.pos += time.Duration(float64(dt) * float64(p.SpeedValue()) * float64(time.Second))
			}
		} else {
			p.pos += time.Duration(float64(dt) * float64(p.SpeedValue()) * float64(time.Second))
		}
		if total > 0 && p.pos >= total {
			if p.loopOn() {
				p.pos = time.Duration(int64(p.pos) % int64(total))
			} else {
				p.pos = total
				p.Playing = false
				if p.OnPause != nil {
					p.OnPause()
				}
			}
		}
		if p.Playing {
			markAnim()
		}
	}
	p.resolveFrame()
}

// resolveFrame 按当前播放位置取帧。
func (p *Player) resolveFrame() {
	if p.Clip != nil {
		idx := p.Clip.FrameIndexAt(p.pos)
		if idx >= 0 && idx < len(p.Clip.Frames) {
			p.frame = &p.Clip.Frames[idx]
			p.frameIdx = idx
			p.frameSig = uint64(idx) + 1
		}
		return
	}
	if p.Source != nil {
		f, err := p.Source.Frame(p.pos)
		if err != nil {
			p.err = err
			return
		}
		p.err = nil
		if f != nil {
			p.frame = f
			p.frameSig = uint64(p.pos) + 1
		}
	}
}

// formatVideoTime 播放时间文本（<1h: m:ss，>=1h: h:mm:ss）。
func formatVideoTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d / time.Second)
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
