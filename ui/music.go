//go:build windows || ((linux || freebsd) && !android)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gogui/internal/audio"
)

// 音乐播放：GoGUI 不做编解码，用外部 ffmpeg 子进程把音频解成 s16le PCM，
// 一路写入 waveOut 播放（设备样本数即播放位置），一路送进环形缓冲做 FFT。
// 与视频播放器的区别：没有画面，位置来自音频设备而非帧率，额外提供实时频谱。
//
// 典型用法：
//
//	src, err := ui.OpenMusic("song.mp3")
//	src.Play()
//	// 每帧（在 view 里）：
//	ui.MusicPlayer("mp", src, ui.MusicPlayerOptions{Track: ui.MusicTrack{Title: "曲名"}})

// musicExts 支持的纯音频扩展名（小写，含点）。
var musicExts = []string{
	".mp3", ".m4a", ".m4b", ".aac", ".flac", ".wav", ".ogg", ".oga", ".opus", ".wma", ".ape", ".aiff", ".aif", ".alac",
}

// MusicFilters 音乐文件对话框过滤器。
const MusicFilters = "音乐|*.mp3;*.m4a;*.m4b;*.aac;*.flac;*.wav;*.ogg;*.opus;*.wma;*.ape;*.aiff|" +
	"MP3|*.mp3|FLAC|*.flac|WAV|*.wav|M4A|*.m4a;*.m4b|AAC|*.aac|OGG / Opus|*.ogg;*.oga;*.opus|" +
	"WMA|*.wma|APE|*.ape|AIFF|*.aiff;*.aif|所有文件|*.*"

// IsMusicPath 判断路径是否为受支持的音频文件（按扩展名判断）。
func IsMusicPath(path string) bool {
	p := strings.ToLower(strings.TrimSpace(path))
	if p == "" {
		return false
	}
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	ext := filepath.Ext(p)
	for _, e := range musicExts {
		if ext == e {
			return true
		}
	}
	return false
}

// MusicFormats 支持的音频容器列表（界面展示用）。
func MusicFormats() []string {
	return []string{"mp3", "flac", "wav", "m4a", "aac", "ogg", "opus", "wma", "ape"}
}

// ---- 音乐解码源 ----

// MusicSource 纯音频解码源：ffmpeg → s16le PCM → waveOut，附带频谱分析。
// 播放位置以音频设备已播样本为准；没有音频设备时按墙钟推进（静音演示）。
type MusicSource struct {
	path string
	info MediaInfo
	cfg  MediaConfig

	rate, ch int

	mu   sync.Mutex
	cond *sync.Cond
	gen  int

	start   time.Duration // 当前窗口起点
	played  time.Duration // 窗口内已播时长
	lastTk  time.Time
	opened  bool
	cmd     *exec.Cmd
	audio   audio.Device
	audioOn bool

	paused  bool
	speed   float64
	vol     float32
	muted   bool
	loop    bool
	closed  bool
	eof     bool
	err     error
	lastPos time.Duration

	// 系统解码器后端（没装 ffmpeg 时使用；为 nil 表示 ffmpeg 路径）
	sys sysSource
	// sysWG 跟踪系统解码协程，关闭前等待退出
	sysWG sync.WaitGroup

	an *Spectrum
}

// OpenMusic 打开音频文件（默认跟随源采样率，声音开启）。
func OpenMusic(path string) (*MusicSource, error) {
	return OpenMusicWith(path, MediaConfig{})
}

// OpenMusicWith 按配置打开音频文件。支持 mp3 / flac / wav / m4a / aac / ogg / opus / wma / ape。
func OpenMusicWith(path string, cfg MediaConfig) (*MusicSource, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("音乐路径为空")
	}
	// 解码器回退：优先 ffmpeg；没安装时用系统解码器（Media Foundation）
	if !ffmpegReady(cfg) {
		return openMusicSystem(path, cfg)
	}
	info, err := ProbeMediaWith(path, cfg.FFprobe)
	if err != nil {
		return nil, err
	}
	info.Backend = "ffmpeg"
	if !info.HasAudio {
		return nil, errors.New("该文件没有音频流")
	}
	if toolPath(cfg.FFmpeg, "ffmpeg") == "" {
		return nil, errors.New("未找到 ffmpeg，请安装 FFmpeg 并把 bin 目录加入 PATH")
	}
	s := &MusicSource{path: path, info: *info, cfg: cfg, speed: 1, vol: 1}
	s.cond = sync.NewCond(&s.mu)
	rate := info.SampleRate
	if rate <= 0 {
		rate = 48000
	}
	s.rate, s.ch = rate, 2
	s.an = newSpectrum(SpectrumBands, rate)

	vol := cfg.Volume
	if vol <= 0 {
		vol = 1
	}
	s.vol = vol
	if !cfg.NoAudio {
		if a, err := audio.Open(rate, 2, 8, 2048); err == nil {
			s.audio = a
			s.audioOn = true
			a.SetVolume(vol)
		}
	}
	return s, nil
}

// OpenMusicPlayer 一步打开音频并自动播放（出错返回 nil + 错误）。
// 典型用法：
//
//	src, err := ui.OpenMusicPlayer("song.mp3")
//	if err == nil { defer src.Close() }
//	// view 里：ui.MusicPlayer("mp", src, ui.MusicPlayerOptions{})
func OpenMusicPlayer(path string) (*MusicSource, error) {
	src, err := OpenMusic(path)
	if err != nil {
		return nil, err
	}
	src.Play()
	return src, nil
}

// Path 音频路径。
func (s *MusicSource) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Info 探测信息（时长 / 编码 / 采样率等）。
func (s *MusicSource) Info() MediaInfo {
	if s == nil {
		return MediaInfo{}
	}
	return s.info
}

// Duration 总时长（未知返回 0）。
func (s *MusicSource) Duration() time.Duration {
	if s == nil {
		return 0
	}
	return s.info.Duration
}

// HasAudio 是否成功开启音频输出设备。
func (s *MusicSource) HasAudio() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.audioOn
}

// Err 最近一次解码错误。
func (s *MusicSource) Err() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Spectrum 频谱分析器（可为 nil）。
func (s *MusicSource) Spectrum() *Spectrum {
	if s == nil {
		return nil
	}
	return s.an
}

// Pos 当前播放位置。
func (s *MusicSource) Pos() time.Duration {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	if s.closed {
		pos := s.start + s.played
		s.mu.Unlock()
		return pos
	}
	if s.audio != nil && !s.paused {
		if d, ok := s.audio.Position(); ok {
			s.played, s.lastTk = d, time.Now()
			pos := s.start + d
			s.mu.Unlock()
			return s.clampPos(pos)
		}
	}
	if !s.paused && !s.lastTk.IsZero() {
		now := time.Now()
		s.played += now.Sub(s.lastTk)
		s.lastTk = now
	}
	pos := s.start + s.played
	s.mu.Unlock()
	return s.clampPos(pos)
}

// clampPos 把位置夹到 [0, 时长]（流媒体时长未知时不夹）。
func (s *MusicSource) clampPos(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if total := s.Duration(); total > 0 && d > total {
		return total
	}
	return d
}

// Progress 播放进度 0~1（时长未知返回 0）。
func (s *MusicSource) Progress() float32 {
	total := s.Duration()
	if total <= 0 {
		return 0
	}
	v := float32(s.Pos()) / float32(total)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Playing 是否正在播放。
func (s *MusicSource) Playing() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.paused && !s.closed && (s.opened || s.audioOn)
}

// Play 开始 / 继续播放。
func (s *MusicSource) Play() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	wasPaused := s.paused
	s.paused = false
	s.lastTk = time.Now()
	opened := s.opened
	total := s.info.Duration
	pos := s.start + s.played
	needReopen := !opened || (total > 0 && pos >= total)
	dev := s.audio
	s.mu.Unlock()

	if dev != nil {
		dev.Pause(false)
	}
	if needReopen {
		s.Seek(0)
	} else if wasPaused {
		RequestRedraw()
	}
	RequestRedraw()
}

// Pause 暂停。
func (s *MusicSource) Pause() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed || s.paused {
		s.mu.Unlock()
		return
	}
	// 先结算墙钟（无音频设备时位置靠它）
	if s.audio == nil && !s.lastTk.IsZero() {
		s.played += time.Since(s.lastTk)
	}
	s.paused = true
	s.lastTk = time.Now()
	dev := s.audio
	s.mu.Unlock()
	if dev != nil {
		dev.Pause(true)
	}
	RequestRedraw()
}

// TogglePlay 播放 / 暂停切换。
func (s *MusicSource) TogglePlay() {
	if s == nil {
		return
	}
	if s.Playing() {
		s.Pause()
	} else {
		s.Play()
	}
}

// Seek 跳转到指定位置（重开解码窗口，声音与频谱一起定位）。
func (s *MusicSource) Seek(pos time.Duration) {
	if s == nil {
		return
	}
	if pos < 0 {
		pos = 0
	}
	if total := s.Duration(); total > 0 && pos > total {
		pos = total
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.gen++
	gen := s.gen
	s.start = pos
	s.played = 0
	s.lastTk = time.Now()
	s.lastPos = pos
	s.opened = true
	s.eof = false
	s.err = nil
	old := s.cmd
	s.cmd = nil
	dev := s.audio
	paused := s.paused
	s.mu.Unlock()

	if s.an != nil {
		s.an.Reset()
	}
	if dev != nil {
		dev.Reset()
		dev.Pause(paused)
	}
	killProcess(old)
	if s.sys != nil {
		s.sysWG.Add(1)
		go func() {
			defer s.sysWG.Done()
			s.runMusicSysAudio(gen, pos)
		}()
		RequestRedraw()
		return
	}
	go s.runAudio(gen, pos)
	RequestRedraw()
}

// SeekRatio 按比例跳转（0~1）。
func (s *MusicSource) SeekRatio(v float32) {
	if s == nil {
		return
	}
	total := s.Duration()
	if total <= 0 {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	s.Seek(time.Duration(float32(total) * v))
}

// Speed 当前倍速（<=0 视为 1）。
func (s *MusicSource) Speed() float32 {
	if s == nil {
		return 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.speed <= 0 {
		return 1
	}
	return float32(s.speed)
}

// SetSpeed 设置倍速（0.5~2），用 atempo 重采样后重开音频窗口。
func (s *MusicSource) SetSpeed(v float32) {
	if s == nil {
		return
	}
	if v < 0.5 {
		v = 0.5
	}
	if v > 2 {
		v = 2
	}
	s.mu.Lock()
	if s.closed || s.speed == float64(v) {
		s.mu.Unlock()
		return
	}
	s.speed = float64(v)
	pos := s.start + s.played
	dev := s.audio
	s.mu.Unlock()
	if dev != nil {
		dev.Reset()
	}
	s.Seek(pos)
}

// Volume 当前音量（0~1）。
func (s *MusicSource) Volume() float32 {
	if s == nil {
		return 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vol
}

// Muted 是否静音。
func (s *MusicSource) Muted() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.muted
}

// SetVolume 设置音量 0~1（0 视为静音）。
func (s *MusicSource) SetVolume(v float32) {
	if s == nil {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	s.mu.Lock()
	s.vol = v
	s.muted = v <= 0
	dev := s.audio
	muted := s.muted
	s.mu.Unlock()
	if dev != nil {
		if muted {
			dev.SetVolume(0)
		} else {
			dev.SetVolume(v)
		}
	}
	RequestRedraw()
}

// ToggleMute 静音切换（保留静音前音量）。
func (s *MusicSource) ToggleMute() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.muted = !s.muted
	vol, muted, dev := s.vol, s.muted, s.audio
	if vol <= 0 && !muted {
		vol = 0.8
		s.vol = vol
	}
	s.mu.Unlock()
	if dev != nil {
		if muted {
			dev.SetVolume(0)
		} else {
			dev.SetVolume(vol)
		}
	}
	RequestRedraw()
}

// Loop 是否单曲循环。
func (s *MusicSource) Loop() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loop
}

// SetLoop 设置单曲循环。
func (s *MusicSource) SetLoop(v bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.loop = v
	s.mu.Unlock()
	RequestRedraw()
}

// Tick 供每帧调用：播放到结尾时按循环设置重播或停在结尾。
// 返回 false 表示本曲已播完（业务可据此切下一首）。
func (s *MusicSource) Tick() bool {
	if s == nil || !s.Playing() {
		return true
	}
	total := s.Duration()
	if total <= 0 {
		return true
	}
	if s.Pos() < total-30*time.Millisecond {
		return true
	}
	if s.Loop() {
		s.Seek(0)
		return true
	}
	// 停在结尾：声音暂停、位置钉在总时长（不重开解码进程，避免多余的一次 seek）
	s.Pause()
	s.mu.Lock()
	s.start, s.played, s.eof = total, 0, true
	s.mu.Unlock()
	return false
}

// Close 释放解码进程与音频设备（幂等）。
func (s *MusicSource) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	cmd, dev, sys := s.cmd, s.audio, s.sys
	s.cmd, s.audio, s.sys = nil, nil, nil
	s.mu.Unlock()
	killProcess(cmd)
	if sys != nil {
		// 等系统解码协程退出后再释放（避免 use-after-free）
		if waitDone(&s.sysWG, 1500*time.Millisecond) {
			sys.Close()
		} else {
			go func() {
				s.sysWG.Wait()
				sys.Close()
			}()
		}
	}
	if dev != nil {
		_ = dev.Close()
	}
}

// fail 记录解码错误。
func (s *MusicSource) fail(gen int, err error) {
	s.mu.Lock()
	if gen == s.gen {
		s.err = err
		s.eof = true
	}
	s.mu.Unlock()
}

// runAudio 启动音频解码进程并写入 waveOut（同时把 PCM 送去频谱分析）。
func (s *MusicSource) runAudio(gen int, start time.Duration) {
	dev := s.audio
	cmd := exec.Command(toolPath(s.cfg.FFmpeg, "ffmpeg"), s.args(start)...)
	hideConsole(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.fail(gen, err)
		return
	}
	stderr := newErrRing()
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		s.fail(gen, err)
		return
	}
	s.mu.Lock()
	if gen != s.gen || s.closed {
		s.mu.Unlock()
		killProcess(cmd)
		_ = cmd.Wait()
		return
	}
	s.cmd = cmd
	paused := s.paused
	rate, ch := s.rate, s.ch
	s.mu.Unlock()
	if dev != nil {
		dev.Pause(paused)
	}

	buf := make([]byte, 16384)
	for {
		n, rerr := stdout.Read(buf)
		if n > 0 {
			pcm := buf[:n]
			if s.an != nil {
				s.an.PushPCM(pcm, ch, rate)
			}
			if dev != nil {
				if err := dev.Write(pcm); err != nil {
					break
				}
			} else {
				// 没有音频设备：按 PCM 时长限速，频谱与进度仍按实时节奏推进
				time.Sleep(pcmDuration(n, rate, ch))
			}
		}
		s.mu.Lock()
		stale := gen != s.gen || s.closed
		s.mu.Unlock()
		if stale {
			break
		}
		if rerr != nil {
			s.mu.Lock()
			if gen == s.gen {
				s.eof = true
				if msg := stderr.String(); msg != "" && !s.closed {
					s.err = errors.New(firstLines(msg, 2))
				}
			}
			s.mu.Unlock()
			break
		}
	}
	_ = cmd.Wait()
	s.mu.Lock()
	if s.cmd == cmd {
		s.cmd = nil
	}
	s.mu.Unlock()
	RequestRedraw()
}

// pcmDuration 一段 s16le PCM 的时长。
func pcmDuration(n, rate, ch int) time.Duration {
	if rate <= 0 || ch <= 0 || n <= 0 {
		return 0
	}
	return time.Duration(float64(n) / float64(rate*ch*2) * float64(time.Second))
}

// args 音频解码命令行（s16le / 源采样率 / 立体声 → 管道）。
func (s *MusicSource) args(start time.Duration) []string {
	args := []string{"-v", "error", "-nostdin", "-hide_banner", "-threads", "1"}
	if isRemotePath(s.path) {
		args = append(args, "-rw_timeout", "15000000",
			"-reconnect", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "4")
	}
	if start > 0 {
		args = append(args, "-ss", secArg(start))
	}
	args = append(args, "-i", s.path, "-map", "0:a:0", "-vn", "-sn", "-dn")
	if sp := s.speedValue(); sp != 1 {
		args = append(args, "-af", strings.Join(atempoChain(sp), ","))
	}
	args = append(args,
		"-ar", strconv.Itoa(s.rate), "-ac", "2",
		"-acodec", "pcm_s16le", "-f", "s16le", "pipe:1")
	return args
}

func (s *MusicSource) speedValue() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.speed <= 0 {
		return 1
	}
	return s.speed
}

// NowPlaying 便捷：曲目信息（标题 / 歌手 / 封面）。
// 元数据读取交给 ffprobe；这里只返回基于文件名与容器信息的兜底结果。
func (s *MusicSource) NowPlaying() MusicTrack {
	if s == nil {
		return MusicTrack{}
	}
	base := filepath.Base(s.path)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	return MusicTrack{
		Path:  s.path,
		Title: title,
		Album: strings.ToUpper(s.info.Format),
		Dur:   s.Duration(),
	}
}
