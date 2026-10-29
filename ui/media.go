//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gogui/internal/win32"
)

// 媒体解码：GoGUI 自身不做编解码，而是调用外部 ffmpeg 子进程把任意容器解成
// RGBA 原始帧（画面）与 s16le PCM（声音），再把帧包装成 VideoSource。
// 支持 mp4 / mkv / mov / avi / webm / flv / ts / m3u8（远端链接亦可）。
//
// 工作方式：
//   - ffprobe 探测时长 / 尺寸 / 帧率 / 音频；
//   - 视频进程按「窗口」解码 rawvideo，读到内存帧环，按播放位置取帧；
//   - 音频进程解码 PCM 写入 waveOut，播放位置以音频设备时钟为准（A/V 同步）；
//   - 跳转时重启两个进程（-ss 输入定位），帧环与音频缓冲一并清空。

// mediaExts 支持的容器扩展名（小写，含点）。
var mediaExts = []string{
	".mp4", ".m4v", ".mkv", ".mov", ".avi", ".webm", ".flv", ".ts", ".m2ts", ".mts", ".m3u8",
}

// MediaFilters 文件对话框过滤器（含所有支持的容器）。
const MediaFilters = "视频|*.mp4;*.m4v;*.mkv;*.mov;*.avi;*.webm;*.flv;*.ts;*.m2ts;*.mts;*.m3u8|" +
	"MP4|*.mp4;*.m4v|MKV|*.mkv|MOV|*.mov|AVI|*.avi|WebM|*.webm|FLV|*.flv|" +
	"TS|*.ts;*.m2ts;*.mts|HLS 播放列表|*.m3u8|所有文件|*.*"

// IsMediaPath 判断路径/链接是否为受支持的媒体（按扩展名判断）。
func IsMediaPath(path string) bool {
	p := strings.ToLower(strings.TrimSpace(path))
	if p == "" {
		return false
	}
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	ext := filepath.Ext(p)
	for _, e := range mediaExts {
		if ext == e {
			return true
		}
	}
	return false
}

// MediaFormats 支持的容器列表（界面展示用）。
func MediaFormats() []string {
	return []string{"mp4", "mkv", "mov", "avi", "webm", "flv", "ts", "m3u8"}
}

// MediaInfo ffprobe（或系统解码器）探测结果。
type MediaInfo struct {
	Path       string
	Format     string
	Duration   time.Duration
	Width      int
	Height     int
	FPS        float64
	Bitrate    int64
	VideoCodec string
	AudioCodec string
	SampleRate int
	Channels   int
	HasVideo   bool
	HasAudio   bool
	// Backend 实际使用的解码后端："ffmpeg" 或 "system"（Windows Media Foundation）。
	Backend string
}

// MediaConfig 解码参数（零值即默认）。
type MediaConfig struct {
	// FFmpeg / FFprobe 可执行文件路径（空则用 PATH 里的 ffmpeg / ffprobe）。
	FFmpeg  string
	FFprobe string
	// MaxWidth / MaxHeight 输出上限；都 <= 0 时跟随源尺寸（不缩放）。
	MaxWidth  int
	MaxHeight int
	// FPS 输出帧率（0 = 跟随源帧率）。
	FPS float64
	// NoAudio 关闭声音解码。
	NoAudio bool
	// Volume 初始音量 0~1（默认 1）。
	Volume float32
	// HWAccel 尝试硬件解码（D3D11VA，失败由 ffmpeg 自动回退软解）。
	HWAccel bool
}

// DecoderPreference 解码器偏好（环境变量 GOGUI_DECODER）：
//   - "system" / "mf"：强制用系统解码器（没装 ffmpeg 时的回退路径，也方便验证）
//   - "ffmpeg"：强制用 ffmpeg（不可用时直接报错，不回退）
//   - 其它 / 未设置：auto（ffmpeg 优先，缺失时自动回退系统解码器）
func DecoderPreference() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GOGUI_DECODER"))) {
	case "system", "mf", "mediafoundation":
		return "system"
	case "ffmpeg":
		return "ffmpeg"
	}
	return "auto"
}

// ffmpegReady ffmpeg / ffprobe 是否都可用（false 时走系统解码器回退）。
func ffmpegReady(cfg MediaConfig) bool {
	ok := toolPath(cfg.FFmpeg, "ffmpeg") != "" && toolPath(cfg.FFprobe, "ffprobe") != ""
	if DecoderPreference() == "system" {
		return false
	}
	return ok
}

type probeDoc struct {
	Streams []struct {
		CodecType    string            `json:"codec_type"`
		CodecName    string            `json:"codec_name"`
		Width        int               `json:"width"`
		Height       int               `json:"height"`
		AvgFrameRate string            `json:"avg_frame_rate"`
		RFrameRate   string            `json:"r_frame_rate"`
		SampleRate   string            `json:"sample_rate"`
		Channels     int               `json:"channels"`
		Disposition  map[string]int    `json:"disposition"`
		SideData     []json.RawMessage `json:"side_data_list"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
}

// parseProbeJSON 解析 ffprobe -print_format json 输出。
func parseProbeJSON(path string, raw []byte) (*MediaInfo, error) {
	var doc probeDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}
	info := &MediaInfo{Path: path, Format: doc.Format.FormatName}
	if f, err := strconv.ParseFloat(doc.Format.Duration, 64); err == nil && f > 0 {
		info.Duration = time.Duration(f * float64(time.Second))
	}
	if b, err := strconv.ParseInt(doc.Format.BitRate, 64, 64); err == nil {
		info.Bitrate = b
	}
	for _, st := range doc.Streams {
		switch st.CodecType {
		case "video":
			// 跳过封面图（attached_pic）
			if st.Disposition["attached_pic"] == 1 || info.HasVideo {
				continue
			}
			w, h := st.Width, st.Height
			if w <= 0 || h <= 0 {
				continue
			}
			info.HasVideo = true
			info.VideoCodec = st.CodecName
			info.Width, info.Height = w, h
			info.FPS = parseFrameRate(st.AvgFrameRate)
			if info.FPS <= 0 {
				info.FPS = parseFrameRate(st.RFrameRate)
			}
			// 旋转元数据：显示尺寸宽高互换
			for _, sd := range st.SideData {
				var m struct {
					Rotation float64 `json:"rotation"`
				}
				if json.Unmarshal(sd, &m) == nil && m.Rotation != 0 {
					r := int(m.Rotation) % 180
					if r == 90 || r == -90 {
						info.Width, info.Height = h, w
					}
				}
			}
		case "audio":
			if info.HasAudio {
				continue
			}
			info.HasAudio = true
			info.AudioCodec = st.CodecName
			if sr, err := strconv.Atoi(st.SampleRate); err == nil {
				info.SampleRate = sr
			}
			info.Channels = st.Channels
		}
	}
	if !info.HasVideo && !info.HasAudio {
		return nil, errors.New("没有可播放的音视频流")
	}
	return info, nil
}

// parseFrameRate 解析 "30000/1001" 形式的帧率。
func parseFrameRate(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0/0" {
		return 0
	}
	if i := strings.IndexByte(s, '/'); i > 0 {
		num, err1 := strconv.ParseFloat(s[:i], 64)
		den, err2 := strconv.ParseFloat(s[i+1:], 64)
		if err1 == nil && err2 == nil && den > 0 && num > 0 {
			return num / den
		}
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// ProbeMedia 用 ffprobe 探测媒体信息。
func ProbeMedia(path string) (*MediaInfo, error) { return ProbeMediaWith(path, "") }

// ProbeMediaWith 指定 ffprobe 路径探测（空则用 PATH）。
func ProbeMediaWith(path, ffprobe string) (*MediaInfo, error) {
	exe := toolPath(ffprobe, "ffprobe")
	if exe == "" {
		return nil, errors.New("未找到 ffprobe，请安装 FFmpeg 并把 bin 目录加入 PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	args := []string{"-v", "error", "-hide_banner", "-print_format", "json", "-show_format", "-show_streams"}
	if isRemotePath(path) {
		args = append(args, "-rw_timeout", "15000000")
	}
	args = append(args, path)
	cmd := exec.CommandContext(ctx, exe, args...)
	hideConsole(cmd)
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New("ffprobe 失败: " + firstLines(msg, 3))
	}
	return parseProbeJSON(path, []byte(out.String()))
}

// ---- 工具函数 ----

// toolPath 定位外部工具：显式路径 > PATH。
func toolPath(explicit, name string) string {
	if explicit != "" {
		return explicit
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	if p, err := exec.LookPath(name + ".exe"); err == nil {
		return p
	}
	return ""
}

// isRemotePath 是否远端链接。
func isRemotePath(p string) bool {
	lp := strings.ToLower(p)
	for _, s := range []string{"http://", "https://", "rtmp://", "rtsp://", "udp://", "rtp://", "ftp://"} {
		if strings.HasPrefix(lp, s) {
			return true
		}
	}
	return false
}

// hideConsole 隐藏子进程控制台窗口（发布构建下不闪黑框）。
func hideConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}

// firstLines 取前 n 行非空文本（错误提示用）。
func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]string, 0, n)
	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
			if len(out) >= n {
				break
			}
		}
	}
	return strings.Join(out, " / ")
}

// secArg 秒数文本（ffmpeg 参数用）。
func secArg(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}

// atempoChain 把任意倍速拆成 ffmpeg atempo 支持的范围（每级 0.5~2）。
func atempoChain(v float64) []string {
	if v <= 0 {
		return nil
	}
	var out []string
	for v > 2 {
		out = append(out, "atempo=2")
		v /= 2
	}
	for v < 0.5 {
		out = append(out, "atempo=0.5")
		v /= 0.5
	}
	out = append(out, "atempo="+strconv.FormatFloat(v, 'f', 3, 64))
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---- 解码源 ----

// mediaFrame 已解码帧 + 帧号（相对当前解码窗口起点）。
type mediaFrame struct {
	idx int
	f   *VideoFrame
}

// MediaSource ffmpeg 解码的帧来源，实现 VideoSource。
// 首次取帧时启动解码进程；跳转超出已缓冲范围时自动重启窗口。
type MediaSource struct {
	path string
	info MediaInfo
	cfg  MediaConfig

	outW, outH int
	fps        float64
	frameBytes int
	maxFrames  int
	// pool 帧缓冲复用池（最多 4 块，每块 frameBytes）。
	// 1080p 一帧 8MB，旧实现「make 读缓冲 + append 复制」每帧分配两次，
	// 25fps 下就是 400MB/s 的分配量，Go 堆被迫膨胀、GC 每两秒一轮；
	// 现在像素缓冲跟帧一起流转：帧被丢弃时归还，解码线程直接复用。
	pool [][]byte

	mu   sync.Mutex
	cond *sync.Cond

	gen  int // 视频窗口代际
	agen int // 音频窗口代际

	start      time.Duration // 当前窗口的媒体起点（音频时钟基准）
	vstart     time.Duration // 视频窗口起点（最小化恢复只重开视频，与音频起点解耦）
	opened     bool          // 是否已启动过解码窗口
	frames     []mediaFrame
	vcmd       *exec.Cmd
	acmd       *exec.Cmd
	audio      *win32.AudioOut
	audioOn    bool
	audioDevOn bool
	speed      float64
	paused     bool
	lastPos    time.Duration
	// 拖动进度中：只记录目标位置不重启解码，松手后一次性定位
	scrubbing   bool
	pendingSeek time.Duration
	hasPending  bool
	// 拖动中的单帧预览（last-wins：最新目标位置覆盖旧请求，单进程串行抓帧）
	previewFrame *VideoFrame
	previewAt    time.Duration
	previewBusy  bool
	previewWant  time.Duration
	previewHas   bool
	lastOpen    time.Time
	closed      bool
	eof         bool
	err         error

	// 系统解码器后端（没装 ffmpeg 时使用；为 nil 表示 ffmpeg 路径）
	sys *win32.MFSourceReader
	// sysWG 跟踪系统解码协程，关闭前等待退出（避免归还 reader 后仍在调用）
	sysWG sync.WaitGroup
}

// OpenMedia 打开媒体文件/链接（默认：跟随源尺寸 + 声音）。
func OpenMedia(path string) (*MediaSource, error) {
	return OpenMediaWith(path, MediaConfig{})
}

// OpenMediaWith 按配置打开媒体。
func OpenMediaWith(path string, cfg MediaConfig) (*MediaSource, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("媒体路径为空")
	}
	// 解码器回退：优先 ffmpeg / ffprobe；没安装时改用系统解码器（Media Foundation）
	if !ffmpegReady(cfg) {
		return openMediaSystem(path, cfg)
	}
	info, err := ProbeMediaWith(path, cfg.FFprobe)
	if err != nil {
		return nil, err
	}
	if !info.HasVideo {
		return nil, errors.New("该文件没有视频流")
	}
	if toolPath(cfg.FFmpeg, "ffmpeg") == "" {
		return nil, errors.New("未找到 ffmpeg，请安装 FFmpeg 并把 bin 目录加入 PATH")
	}

	s := &MediaSource{path: path, info: *info, cfg: cfg, speed: 1}
	s.cond = sync.NewCond(&s.mu)

	// 输出尺寸：默认跟随源尺寸
	w, h := info.Width, info.Height
	if w <= 0 || h <= 0 {
		w, h = 1280, 720
	}
	if cfg.MaxWidth > 0 && w > cfg.MaxWidth {
		h = int(float64(h) * float64(cfg.MaxWidth) / float64(w))
		w = cfg.MaxWidth
	}
	if cfg.MaxHeight > 0 && h > cfg.MaxHeight {
		w = int(float64(w) * float64(cfg.MaxHeight) / float64(h))
		h = cfg.MaxHeight
	}
	s.outW, s.outH = clampInt(w, 2, 7680), clampInt(h, 2, 4320)
	s.frameBytes = s.outW * s.outH * 4

	// 输出帧率：跟随源（夹到 1~120）
	fps := cfg.FPS
	if fps <= 0 {
		fps = info.FPS
	}
	if fps <= 0 {
		fps = 25
	}
	s.fps = clampFloat(fps, 1, 120)
	info.Backend = "ffmpeg"
	s.info = *info

	// 帧环容量：约 48MB 且不超过 2 秒，至少 3 帧。
	// 1080p 一帧约 8MB，旧上限（96MB/4s）会让播放时时常驻 ~90MB；
	// 本地解码速度远超播放速度，200ms 缓冲已足够，内存优先。
	byBytes := (48 << 20) / s.frameBytes
	byTime := int(s.fps * 2)
	s.maxFrames = clampInt(minInt(byBytes, byTime), 3, 240)

	if !cfg.NoAudio && info.HasAudio {
		vol := cfg.Volume
		if vol <= 0 {
			vol = 1
		}
		if a, err := win32.NewAudioOut(48000, 2, 8, 2048); err == nil {
			s.audio = a
			s.audioDevOn = true
			a.SetVolume(vol)
		}
	}
	registerMedia(s)
	return s, nil
}

// Info 探测到的媒体信息。
func (s *MediaSource) Info() MediaInfo {
	if s == nil {
		return MediaInfo{}
	}
	return s.info
}

// Path 媒体路径。
func (s *MediaSource) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Size 解码输出尺寸。
func (s *MediaSource) Size() (int, int) {
	if s == nil {
		return 0, 0
	}
	return s.outW, s.outH
}

// Duration 总时长（流媒体可能为 0）。
func (s *MediaSource) Duration() time.Duration {
	if s == nil {
		return 0
	}
	return s.info.Duration
}

// HasAudio 是否成功开启音频输出。
func (s *MediaSource) HasAudio() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.audioDevOn
}

// Err 最近一次解码错误。
func (s *MediaSource) Err() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Frame 取时间点 pos 的画面（VideoSource 实现）。
func (s *MediaSource) Frame(pos time.Duration) (*VideoFrame, error) {
	if s == nil {
		return nil, nil
	}
	if pos < 0 {
		pos = 0
	}
	fps := s.fps
	idx := int(float64(pos) / float64(time.Second) * fps)

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errors.New("媒体已关闭")
	}
	s.lastPos = pos
	needOpen := !s.opened
	// 拖动中：记下目标位置（松手后一次性定位）并请求单帧预览，立即返回不阻塞 UI 线程
	if s.scrubbing && !needOpen {
		s.pendingSeek, s.hasPending = pos, true
		s.mu.Unlock()
		return s.scrubFrame(pos, idx), nil
	}
	s.mu.Unlock()

	if needOpen {
		s.openWindow(pos)
	}

	// 等待预算：本地短一些，远端（HLS 建连 / 拉首片）略长；
	// 超时就返回当前可用帧（无则 nil，界面显示“解码中…”），避免阻塞 UI 线程。
	// 已经显示过画面时预算再压低：解码短暂落后就先返回上一帧，
	// 不要每帧都等几百毫秒把 UI 线程拖住。
	budget := 600 * time.Millisecond
	remote := isRemotePath(s.path)
	if remote {
		// 远端预算要小：Frame 在 UI 线程上调用，阻塞太久窗口会“未响应”。
		// HLS 首帧靠“不重启窗口 + 放宽停滞容忍”来等，而不是靠阻塞 UI。
		budget = 300 * time.Millisecond
	}
	s.mu.Lock()
	hasFrame := len(s.frames) > 0
	s.mu.Unlock()
	if hasFrame {
		budget = 150 * time.Millisecond
		if remote {
			budget = 250 * time.Millisecond
		}
	}
	deadline := time.Now().Add(budget)
	// 停滞检测：帧环被填满 / 管道阻塞时解码器不会自己往下走，
	// 长时间没有新帧就直接重建窗口（最小化恢复、系统挂起后最常见的场景）。
	stallLast, stallAt := -2, time.Now()
	for time.Now().Before(deadline) {
		s.mu.Lock()
		if f := s.lookup(idx); f != nil {
			s.dropBefore(idx)
			s.mu.Unlock()
			return f, nil
		}
		base := int(float64(s.vstart) / float64(time.Second) * fps)
		rel := idx - base
		last := s.lastBuffered()
		full := len(s.frames) >= s.maxFrames
		decoding := (s.vcmd != nil || s.sys != nil) && !s.eof
		lastOpen := s.lastOpen
		start := s.vstart
		s.mu.Unlock()

		if last != stallLast {
			stallLast, stallAt = last, time.Now()
		}

		// 解码器还在拆包 / 追帧（含首帧未到）：等待推进
		if decoding && (rel > last || last < 0) {
			// 帧环已满且目标帧在缓冲之后：读帧 goroutine 正卡在 waitRoom 等空位，
			// 不消费旧帧就永远拿不到新帧（画面会停在最小化前那一帧不动）。
			if full && rel > last {
				if remote {
					// 远端（HLS/直播）解码只能按实时速度推进，而播放时钟总会稍稍领先，
					// 于是“目标帧落在缓冲之外”是常态。旧逻辑会走下面的“离太远就重开窗口”，
					// 每 1.5s 杀一次 ffmpeg 重连，永远拿不到首帧（界面卡在“解码中”）。
					// 这里改成：丢掉旧帧腾空间，直接把最新帧交出去，下一帧继续追。
					s.trimBuffer(idx)
					s.mu.Lock()
					var nf *VideoFrame
					if n := len(s.frames); n > 0 {
						nf = s.frames[n-1].f
					}
					s.mu.Unlock()
					if nf != nil {
						return nf, nil
					}
					continue
				}
				if rel-last > s.maxFrames {
					// 目标离缓冲太远：直接重开窗口，比逐帧追更快
					s.openWindow(pos)
					stallAt = time.Now()
					continue
				}
				s.trimBuffer(idx)
			}
			if time.Since(stallAt) > stallLimit(remote) {
				// 解码器长时间没有推进：重建窗口，避免永久显示旧帧
				s.openWindow(pos)
				stallAt = time.Now()
				continue
			}
			if s.waitFrame(idx, 60*time.Millisecond) {
				s.mu.Lock()
				if f := s.lookup(idx); f != nil {
					s.dropBefore(idx)
					s.mu.Unlock()
					return f, nil
				}
				s.mu.Unlock()
			}
			continue
		}
		// 解码进程尚未启动（刚开窗）或已结束：已结束则不再等，否则等启动
		if !decoding {
			s.mu.Lock()
			ended := s.eof || s.err != nil || s.closed
			s.mu.Unlock()
			if ended {
				break
			}
			if s.waitStart(1500 * time.Millisecond) {
				continue
			}
			break
		}
		// 目标比缓冲旧（跳转）或解码器已停止推进：重启窗口，避免原地反复重启
		// 远端刚开窗的 2 秒内不重建：HLS 建连 + 拉首片本来就要时间，
		// 重建只会把已经建立的连接白白丢掉、永远出不了首帧。
		if remote && time.Since(lastOpen) < remoteGrace {
			break
		}
		if time.Since(lastOpen) < 300*time.Millisecond && pos == start {
			break
		}
		s.openWindow(pos)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	if n := len(s.frames); n > 0 {
		return s.frames[n-1].f, nil
	}
	return nil, nil
}

// remoteGrace 远端新建解码窗口后的“建连宽限期”：这段时间内即使一时没有新帧，
// 也不重建窗口（HLS 要下载 m3u8 / 按序拉切片 / 重连，重建只会白丢连接）。
const remoteGrace = 10 * time.Second

// stallLimit 解码器长时间无新帧的容忍时间：本地 400ms 足够（解码远快于播放），
// 远端 HLS 切片 / 重连 / 网络抖动需要缓冲时间，给到 10 秒才重建窗口。
func stallLimit(remote bool) time.Duration {
	if remote {
		return remoteGrace
	}
	return 400 * time.Millisecond
}

// ---- 媒体源注册表（最小化恢复时统一唤醒）----

var (
	mediaMu   sync.Mutex
	mediaLive = make(map[*MediaSource]bool)
)

func registerMedia(s *MediaSource) {
	if s == nil {
		return
	}
	mediaMu.Lock()
	mediaLive[s] = true
	mediaMu.Unlock()
}

func unregisterMedia(s *MediaSource) {
	if s == nil {
		return
	}
	mediaMu.Lock()
	delete(mediaLive, s)
	mediaMu.Unlock()
}

// ResumeMediaAll 窗口从最小化恢复时调用：让所有存活的媒体源重建视频解码窗口。
// 最小化期间 UI 不再取帧，解码器早把帧环填满并阻塞在管道上，
// 不重建就会一直绘着最小化前的那一帧不动。
func ResumeMediaAll() {
	mediaMu.Lock()
	list := make([]*MediaSource, 0, len(mediaLive))
	for s := range mediaLive {
		list = append(list, s)
	}
	mediaMu.Unlock()
	for _, s := range list {
		s.MediaResume()
	}
}

// MediaResume 重建视频解码窗口到当前播放位置（音频进程与时钟保持连续）。
func (s *MediaSource) MediaResume() {
	if s == nil {
		return
	}
	s.mu.Lock()
	opened, closed, scrubbing := s.opened, s.closed, s.scrubbing
	pos := s.lastPos
	s.mu.Unlock()
	if closed || !opened || scrubbing {
		return
	}
	// 有声音时以音频设备时钟为准（最小化期间声音仍在播放，位置已经前进了）
	if d, ok := s.MediaClock(); ok {
		pos = d
	}
	s.reopenVideo(pos)
}

// reopenVideo 只重建视频解码窗口：不动音频进程与 s.start（音频时钟基准），
// 视频定位基准单独用 s.vstart，避免重开视频把声音时钟弄乱。
func (s *MediaSource) reopenVideo(pos time.Duration) {
	if pos < 0 {
		pos = 0
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.gen++
	gen := s.gen
	s.vstart = pos
	s.lastOpen = time.Now()
	s.opened = true
	s.releaseFrames(s.frames)
	s.frames = nil
	s.eof = false
	s.err = nil
	s.clearPreview()
	oldV := s.vcmd
	s.vcmd = nil
	s.mu.Unlock()
	s.cond.Broadcast()
	killProcess(oldV)
	if s.sys != nil {
		s.sysWG.Add(1)
		s.goAsync(func() {
			defer s.sysWG.Done()
			s.runSysLoop(gen, pos)
		})
		return
	}
	s.goAsync(func() { s.runVideo(gen, pos) })
}

// scrubFrame 拖动中的画面：优先缓冲命中帧，其次最近的单帧预览，最后保留上一帧。
// 拖动中绝不等待解码（避免 UI 线程卡顿），画面靠预览抓帧跟上手指。
func (s *MediaSource) scrubFrame(pos time.Duration, idx int) *VideoFrame {
	s.requestPreview(pos)
	s.mu.Lock()
	defer s.mu.Unlock()
	if f := s.lookup(idx); f != nil {
		return f
	}
	if s.previewFrame != nil && s.previewFrame.Valid() {
		return s.previewFrame
	}
	if n := len(s.frames); n > 0 {
		return s.frames[n-1].f
	}
	return nil
}

// requestPreview 请求拖动预览帧：只记最新目标（last-wins），同一时刻只跑一个抓帧进程，
// 拖动再快也不会拉起一堆 ffmpeg。
func (s *MediaSource) requestPreview(pos time.Duration) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.previewWant, s.previewHas = pos, true
	if s.previewBusy {
		s.mu.Unlock()
		return
	}
	s.previewBusy = true
	s.mu.Unlock()
	go s.previewWorker()
}

// previewWorker 预览抓帧循环：取最新目标 → 解一帧 → 更新预览（不影响主解码窗口）。
func (s *MediaSource) previewWorker() {
	for {
		s.mu.Lock()
		if s.closed || !s.scrubbing || !s.previewHas {
			s.previewHas = false
			s.previewBusy = false
			s.mu.Unlock()
			return
		}
		at := s.previewWant
		s.previewHas = false
		gen := s.gen
		s.mu.Unlock()

		f := s.grabFrame(at)
		if f == nil {
			time.Sleep(30 * time.Millisecond)
			continue
		}

		s.mu.Lock()
		if gen == s.gen && !s.closed {
			// 旧预览帧同样占 8MB，换帧前先归还
			if s.previewFrame != nil && s.previewFrame != f {
				s.releaseFrame(s.previewFrame)
			}
			s.previewFrame, s.previewAt = f, at
		}
		stale := gen != s.gen || s.closed
		if stale {
			s.previewBusy = false
		}
		s.mu.Unlock()
		if stale {
			return
		}
	}
}

// grabFrame 单帧抓取（拖动预览用）：-ss 快速定位后只解一帧 RGBA。
func (s *MediaSource) grabFrame(pos time.Duration) *VideoFrame {
	if pos < 0 {
		pos = 0
	}
	exe := toolPath(s.cfg.FFmpeg, "ffmpeg")
	if exe == "" {
		return nil
	}
	args := []string{"-v", "error", "-nostdin", "-hide_banner"}
	if isRemotePath(s.path) {
		args = append(args, "-rw_timeout", "8000000")
	} else {
		args = append(args, "-analyzeduration", "1000000", "-probesize", "2000000")
	}
	args = append(args, "-ss", secArg(pos), "-i", s.path,
		"-frames:v", "1", "-an", "-sn", "-dn",
		"-vf", fmt.Sprintf("scale=%d:%d:flags=fast_bilinear,setsar=1", s.outW, s.outH),
		"-pix_fmt", "rgba", "-f", "rawvideo", "pipe:1")

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	hideConsole(cmd)
	cmd.Stderr = newErrRing()
	out, err := cmd.Output()
	if err != nil || len(out) < s.frameBytes {
		return nil
	}
	return &VideoFrame{
		Pix:   out[:s.frameBytes],
		W:     s.outW,
		H:     s.outH,
		Delay: time.Duration(float64(time.Second) / s.fps),
	}
}

// lookup 在缓冲里找帧号 idx（相对窗口起点）。
func (s *MediaSource) lookup(idx int) *VideoFrame {
	rel := idx - int(float64(s.vstart)/float64(time.Second)*s.fps)
	for i := range s.frames {
		if s.frames[i].idx == rel {
			return s.frames[i].f
		}
	}
	return nil
}

// lastBuffered 缓冲中最新帧号（无则 -1）。
func (s *MediaSource) lastBuffered() int {
	if len(s.frames) == 0 {
		return -1
	}
	return s.frames[len(s.frames)-1].idx
}

// dropBefore 丢弃过旧帧（保留一帧备用，避免取帧抖动）。
func (s *MediaSource) dropBefore(idx int) {
	rel := idx - int(float64(s.vstart)/float64(time.Second)*s.fps)
	cut := 0
	for cut < len(s.frames) && s.frames[cut].idx < rel-1 {
		cut++
	}
	if cut > 0 {
		s.releaseFrames(s.frames[:cut])
		// 原地左移：避免每帧重新分配切片
		n := copy(s.frames, s.frames[cut:])
		s.frames = s.frames[:n]
		s.cond.Broadcast()
	}
}

// trimBuffer 腾出帧环空间：缓冲已满且目标帧在缓冲之后时，解码读帧 goroutine
// 正阻塞在 waitRoom 等空位——不消费旧帧就永远拿不到新帧（最小化/长时间不取帧后
// 画面会永久停在旧的一帧）。这里丢掉除最后一帧以外的旧帧并唤醒等待方。
func (s *MediaSource) trimBuffer(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.frames) > 1 {
		s.releaseFrames(s.frames[:len(s.frames)-1])
		last := s.frames[len(s.frames)-1]
		s.frames = s.frames[:1]
		s.frames[0] = last
	}
	s.cond.Broadcast()
}

// waitStart 等待解码进程启动（或已结束），超时返回 false。
func (s *MediaSource) waitStart(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for {
		s.mu.Lock()
		ready := s.vcmd != nil || s.sys != nil || s.eof || s.err != nil || s.closed
		s.mu.Unlock()
		if ready {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(3 * time.Millisecond)
	}
}

// waitFrame 等解码器推进到指定帧号（超时返回 false）。
func (s *MediaSource) waitFrame(idx int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return false
		}
		if s.lookup(idx) != nil {
			s.mu.Unlock()
			return true
		}
		if time.Now().After(deadline) {
			s.mu.Unlock()
			return false
		}
		s.mu.Unlock()
		time.Sleep(4 * time.Millisecond)
	}
}

// openWindow 从 pos 重新启动解码窗口（视频 + 音频）。
func (s *MediaSource) openWindow(pos time.Duration) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.gen++
	s.agen++
	gen, agen := s.gen, s.agen
	s.start = pos
	s.vstart = pos
	s.lastPos = pos
	s.lastOpen = time.Now()
	s.opened = true
	s.hasPending = false
	s.releaseFrames(s.frames)
	s.frames = nil
	s.eof = false
	s.err = nil
	s.clearPreview()
	oldV, oldA := s.vcmd, s.acmd
	s.vcmd, s.acmd = nil, nil
	if s.audio != nil {
		// 新窗口：清空排队声音，并按播放状态恢复（拖动时已被暂停）
		s.audio.Reset()
		s.audio.Pause(s.paused)
	}
	s.mu.Unlock()
	s.cond.Broadcast()

	killProcess(oldV)
	killProcess(oldA)

	if s.sys != nil {
		// 系统解码器：单 reader 交错读画面 + 声音，重新定位后重跑读循环
		s.sysWG.Add(1)
		s.goAsync(func() {
			defer s.sysWG.Done()
			s.runSysLoop(gen, pos)
		})
		return
	}

	s.goAsync(func() { s.runVideo(gen, pos) })
	if s.audioDevOn {
		s.goAsync(func() { s.runAudio(agen, pos) })
	}
}

// goAsync 启动后台解码 goroutine（Close 后自行退出）。
func (s *MediaSource) goAsync(fn func()) { go fn() }

// takeFrameBuf 取一块帧缓冲（调用方需持有 s.mu）：优先复用回收池里的块。
func (s *MediaSource) takeFrameBuf() []byte {
	if n := len(s.pool); n > 0 {
		b := s.pool[n-1]
		s.pool[n-1] = nil
		s.pool = s.pool[:n-1]
		if len(b) == s.frameBytes {
			return b
		}
	}
	return make([]byte, s.frameBytes)
}

// putFrameBuf 归还帧缓冲（调用方需持有 s.mu）：池上限 4 块，多余交给 GC。
func (s *MediaSource) putFrameBuf(b []byte) {
	if b == nil || len(b) < s.frameBytes || len(s.pool) >= 4 {
		return
	}
	s.pool = append(s.pool, b[:s.frameBytes])
}

// releaseFrame 归还一帧的像素缓冲并断开引用（调用方需持有 s.mu）。
// 断开引用可防止播放器仍指向已复用的缓冲，同时避免同一块被重复归还。
func (s *MediaSource) releaseFrame(f *VideoFrame) {
	if f == nil || f.Pix == nil {
		return
	}
	s.putFrameBuf(f.Pix)
	f.Pix = nil
}

// releaseFrames 归还一批帧的像素缓冲（调用方需持有 s.mu）。
func (s *MediaSource) releaseFrames(frames []mediaFrame) {
	for i := range frames {
		s.releaseFrame(frames[i].f)
	}
}

// clearPreview 丢弃拖动预览帧并归还缓冲（调用方需持有 s.mu）。
func (s *MediaSource) clearPreview() {
	s.releaseFrame(s.previewFrame)
	s.previewFrame = nil
	s.previewHas = false
}

// runVideo 启动视频解码进程并读帧。
func (s *MediaSource) runVideo(gen int, start time.Duration) {
	cmd := exec.Command(toolPath(s.cfg.FFmpeg, "ffmpeg"), s.videoArgs(start)...)
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
	s.vcmd = cmd
	s.mu.Unlock()

	s.mu.Lock()
	buf := s.takeFrameBuf()
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.putFrameBuf(buf)
		s.mu.Unlock()
	}()
	for {
		if _, err := io.ReadFull(stdout, buf); err != nil {
			s.mu.Lock()
			if gen == s.gen {
				s.eof = true
				if !s.closed {
					if msg := stderr.String(); msg != "" {
						s.err = errors.New(firstLines(msg, 2))
					}
				}
			}
			s.mu.Unlock()
			break
		}
		if !s.waitRoom(gen) {
			break
		}
		// 缓冲所有权直接交给帧环（不再复制一份 8MB），下一帧另取一块
		f := &VideoFrame{
			Pix:   buf,
			W:     s.outW,
			H:     s.outH,
			Delay: time.Duration(float64(time.Second) / s.fps),
		}
		s.mu.Lock()
		if gen != s.gen || s.closed {
			s.mu.Unlock()
			break
		}
		idx := 0
		if n := len(s.frames); n > 0 {
			idx = s.frames[n-1].idx + 1
		}
		s.frames = append(s.frames, mediaFrame{idx: idx, f: f})
		s.cond.Broadcast()
		buf = s.takeFrameBuf()
		s.mu.Unlock()
	}
	_ = cmd.Wait()
	s.mu.Lock()
	if s.vcmd == cmd {
		s.vcmd = nil
	}
	s.mu.Unlock()
}

// waitRoom 等待帧环空位（跳转 / 关闭时立即返回 false）。
func (s *MediaSource) waitRoom(gen int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.frames) >= s.maxFrames && gen == s.gen && !s.closed {
		s.cond.Wait()
	}
	return gen == s.gen && !s.closed
}

// runAudio 启动音频解码进程并写入 waveOut。
func (s *MediaSource) runAudio(agen int, start time.Duration) {
	dev := s.audio
	if dev == nil {
		return
	}
	cmd := exec.Command(toolPath(s.cfg.FFmpeg, "ffmpeg"), s.audioArgs(start)...)
	hideConsole(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	cmd.Stderr = newErrRing()
	if err := cmd.Start(); err != nil {
		return
	}
	s.mu.Lock()
	if agen != s.agen || s.closed {
		s.mu.Unlock()
		killProcess(cmd)
		_ = cmd.Wait()
		return
	}
	s.acmd = cmd
	paused := s.paused
	s.mu.Unlock()
	if paused {
		dev.Pause(true)
	}

	buf := make([]byte, 16384)
	for {
		n, rerr := stdout.Read(buf)
		if n > 0 {
			s.mu.Lock()
			stale := agen != s.agen || s.closed
			s.mu.Unlock()
			if stale {
				break
			}
			if err := dev.Write(buf[:n]); err != nil {
				break
			}
		}
		if rerr != nil {
			break
		}
	}
	_ = cmd.Wait()
	s.mu.Lock()
	if s.acmd == cmd {
		s.acmd = nil
	}
	s.mu.Unlock()
}

// decodeThreads ffmpeg 解码线程数（string，直接拼进参数）。
// ffmpeg 默认按 CPU 核数开线程，高分辨率时线程栈 + 帧缓冲叠加会让子进程
// 常驻上百 MB；限制到 CPU 核数的一半（2~4）足够实时解码且更省内存。
func decodeThreads() string {
	n := runtime.NumCPU() / 2
	if n < 2 {
		n = 2
	}
	if n > 4 {
		n = 4
	}
	return strconv.Itoa(n)
}

// videoArgs 视频解码命令行。
func (s *MediaSource) videoArgs(start time.Duration) []string {
	args := []string{"-v", "error", "-nostdin", "-hide_banner"}
	args = append(args, "-threads", decodeThreads(), "-filter_threads", "1")
	if isRemotePath(s.path) {
		args = append(args, "-rw_timeout", "15000000",
			"-reconnect", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "4")
	}
	if s.cfg.HWAccel {
		args = append(args, "-hwaccel", "d3d11va")
	}
	if !isRemotePath(s.path) {
		// 本地文件缩短输入分析时间，降低 seek 后拉起进程的延迟
		args = append(args, "-analyzeduration", "1000000", "-probesize", "2000000")
	}
	if start > 0 {
		args = append(args, "-ss", secArg(start))
	}
	args = append(args, "-i", s.path,
		"-map", "0:v:0", "-an", "-sn", "-dn",
		"-threads", decodeThreads(),
		"-vf", fmt.Sprintf("scale=%d:%d:flags=bilinear,setsar=1", s.outW, s.outH),
		"-r", strconv.FormatFloat(s.fps, 'f', 3, 64),
		"-pix_fmt", "rgba", "-f", "rawvideo", "pipe:1")
	return args
}

// audioArgs 音频解码命令行（s16le / 48kHz / 立体声）。
func (s *MediaSource) audioArgs(start time.Duration) []string {
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
	args = append(args, "-ar", "48000", "-ac", "2", "-acodec", "pcm_s16le", "-f", "s16le", "pipe:1")
	return args
}

func (s *MediaSource) speedValue() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.speed <= 0 {
		return 1
	}
	return s.speed
}

// fail 记录解码错误。
func (s *MediaSource) fail(gen int, err error) {
	s.mu.Lock()
	if gen == s.gen {
		s.err = err
		s.eof = true
	}
	s.mu.Unlock()
}

// MediaClock 播放时钟：有声音时以音频设备已播样本为准（A/V 同步），否则无效。
func (s *MediaSource) MediaClock() (time.Duration, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	dev, start, on, paused, closed := s.audio, s.start, s.audioDevOn, s.paused, s.closed
	s.mu.Unlock()
	if !on || dev == nil || paused || closed {
		return 0, false
	}
	if d, ok := dev.Position(); ok {
		return start + d, true
	}
	return 0, false
}

// MediaScrub 拖动进度状态：拖动中不动解码（画面走单帧预览、声音暂停），
// 松手后一次性定位到最终位置，画面与声音同时对齐。
func (s *MediaSource) MediaScrub(v bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.scrubbing == v {
		s.mu.Unlock()
		return
	}
	s.scrubbing = v
	dev := s.audio
	paused := s.paused
	var seekTo time.Duration
	doseek := false
	if v {
		// 进入拖动：丢弃旧的预览请求，声音先停（否则画面不动声音还在跑）
		s.previewHas = false
	} else if s.hasPending {
		seekTo, doseek = s.pendingSeek, true
		s.hasPending = false
		s.clearPreview()
	}
	s.mu.Unlock()

	if v {
		if dev != nil {
			dev.Pause(true)
		}
		RequestRedraw()
		return
	}
	if doseek {
		s.openWindow(seekTo)
	} else if dev != nil {
		dev.Pause(paused)
	}
	RequestRedraw()
}

// scrubNow 当前是否处于拖动进度中。
func (s *MediaSource) scrubNow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scrubbing
}

// MediaPlay 播放 / 暂停（Player 状态变化时调用）。
func (s *MediaSource) MediaPlay(playing bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.paused = !playing
	dev := s.audio
	s.mu.Unlock()
	if dev != nil {
		dev.Pause(!playing)
	}
}

// MediaVolume 音量 / 静音。
func (s *MediaSource) MediaVolume(v float32, muted bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	dev := s.audio
	s.mu.Unlock()
	if dev == nil {
		return
	}
	if muted {
		v = 0
	}
	dev.SetVolume(v)
}

// MediaSpeed 播放速率：音频用 atempo 滤镜重采样，需要重启音频进程。
func (s *MediaSource) MediaSpeed(v float32) {
	if s == nil {
		return
	}
	if v <= 0 {
		v = 1
	}
	s.mu.Lock()
	changed := s.speed != float64(v)
	s.speed = float64(v)
	dev := s.audio
	pos := s.lastPos
	paused := s.paused
	closed := s.closed
	// 系统解码器不支持 atempo 重采样，倍速仅记录不重开（避免声音与画面脱节）
	sysMode := s.sys != nil
	if !changed || dev == nil || closed || sysMode {
		s.mu.Unlock()
		return
	}
	s.agen++
	agen := s.agen
	oldCmd := s.acmd
	s.acmd = nil
	s.mu.Unlock()
	killProcess(oldCmd)
	dev.Reset()
	dev.Pause(paused)
	s.goAsync(func() { s.runAudio(agen, pos) })
}

// Close 关闭解码进程与音频设备（幂等）。
func (s *MediaSource) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	vcmd, acmd, dev := s.vcmd, s.acmd, s.audio
	sv := s.sys
	s.vcmd, s.acmd, s.audio, s.sys = nil, nil, nil, nil
	s.releaseFrames(s.frames)
	s.frames = nil
	s.clearPreview()
	s.pool = nil
	s.mu.Unlock()
	unregisterMedia(s)
	s.cond.Broadcast()
	killProcess(vcmd)
	killProcess(acmd)
	// 系统解码协程可能正在 MF 内部调用：等它退出后再释放 reader（否则 use-after-free）。
	// 能等到就立即释放，等不到就交给后台等（宁可透一会儿也不再碰同一个 MF 平台）。
	if sv != nil {
		if waitDone(&s.sysWG, 1500*time.Millisecond) {
			sv.Close()
		} else {
			go func() {
				s.sysWG.Wait()
				sv.Close()
			}()
		}
	}
	if dev != nil {
		_ = dev.Close()
	}
}

// killProcess 结束子进程（由读取方 Wait 回收）。
func killProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}

// errRing 子进程 stderr 的滑动缓冲（只保留尾部若干字节）。
type errRing struct {
	mu  sync.Mutex
	buf []byte
}

const errRingCap = 8 << 10

func newErrRing() *errRing { return &errRing{} }

func (e *errRing) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.buf = append(e.buf, p...)
	if len(e.buf) > errRingCap {
		e.buf = append([]byte(nil), e.buf[len(e.buf)-errRingCap:]...)
	}
	return len(p), nil
}

func (e *errRing) String() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return strings.TrimSpace(string(e.buf))
}
