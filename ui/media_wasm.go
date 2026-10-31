//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"syscall/js"
	"time"
)

// 浏览器端媒体后端：把解码与渲染交给浏览器的 <video> 元素。
//
// 与桌面端（Windows 走 Media Foundation / ffmpeg）的区别：
//   - 画面由浏览器原生解码，直接以 DOM 元素叠在 WebGL 画布上（见 overlay_wasm.go），
//     因此 MediaSource.Frame 恒返回 nil，paintVideo 只负责定位元素；
//   - 播放进度以 video.currentTime 为准（MediaClock），拖动/跳转通过写 currentTime 完成；
//   - 只支持浏览器能解的容器与编码（mp4/h264、webm、部分 hls 直链 .m3u8）。

// errUnsupportedMedia 浏览器端无法打开该媒体。
var errUnsupportedMedia = errors.New("ui: 浏览器无法播放该媒体（可用 HasCap(CapVideo) 判断）")

// MediaFilters 文件对话框用的媒体过滤器（与桌面端一致）。
const MediaFilters = "视频|*.mp4;*.m4v;*.mkv;*.mov;*.avi;*.webm;*.flv;*.ts;*.m2ts;*.mts;*.m3u8|" +
	"音频|*.mp3;*.flac;*.wav;*.m4a;*.aac;*.ogg;*.opus|所有文件|*.*"

// mediaExts 浏览器端可尝试的媒体扩展名。
var mediaExts = []string{
	".mp4", ".m4v", ".mkv", ".mov", ".avi", ".webm", ".flv", ".ts", ".m2ts", ".mts", ".m3u8",
	".mp3", ".flac", ".wav", ".m4a", ".aac", ".ogg", ".opus",
}

// IsMediaPath 按扩展名判断是否媒体文件（blob:/data: 地址按视频处理）。
func IsMediaPath(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "blob:") || strings.HasPrefix(lower, "data:video/") || strings.HasPrefix(lower, "data:audio/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	// blob/data URL 里可能带查询串，取扩展名失败时再宽松匹配一次
	for _, e := range mediaExts {
		if ext == e {
			return true
		}
	}
	return false
}

// MediaFormats 浏览器端支持的容器列表（界面展示用）。
func MediaFormats() []string {
	return []string{"mp4", "m4v", "webm", "mov", "m3u8", "mp3", "m4a", "wav", "ogg"}
}

// MediaInfo 媒体探测结果（字段与桌面端一致）。
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
	// Backend 实际使用的解码后端；浏览器端恒为 "browser"。
	Backend string
}

// MediaConfig 解码参数（浏览器端只用到 NoAudio / Volume）。
type MediaConfig struct {
	FFmpeg    string
	FFprobe   string
	MaxWidth  int
	MaxHeight int
	FPS       float64
	NoAudio   bool
	Volume    float32
	HWAccel   bool
}

// DecoderPreference 解码器偏好（浏览器端固定走原生解码）。
func DecoderPreference() string { return "browser" }

// ProbeMedia 探测媒体信息（浏览器端只填路径与容器名，尺寸/时长在元数据到达后回填）。
func ProbeMedia(path string) (*MediaInfo, error) { return ProbeMediaWith(path, "") }

// ProbeMediaWith 指定 ffprobe 路径探测（浏览器端忽略该参数）。
func ProbeMediaWith(path, ffprobe string) (*MediaInfo, error) {
	info := MediaInfo{Path: path, Backend: "browser"}
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		info.Format = strings.TrimPrefix(ext, ".")
	}
	return &info, nil
}

// MediaSource 媒体源：一个 <video> 元素 + 元数据缓存。
type MediaSource struct {
	path string
	cfg  MediaConfig
	el   js.Value
	id   string
	info MediaInfo
	err  error

	ready bool
	funcs []js.Func

	// 最近一次 seek 的时间与目标，拖动进度条时做节流
	lastSeekAt float64
	lastSeekTo time.Duration

	// wantVol / wantMuted 业务期望的音量；浏览器自动播放策略要求「静音起播」，
	// 所以 play() 成功后再按期望值恢复（见 MediaPlay / applyWanted）。
	wantVol   float32
	wantMuted bool

	// hls hls.js 实例（播放 .m3u8 时创建；Safari 等原生支持 HLS 的浏览器为空）
	hls js.Value
}

// OpenMedia 打开媒体（mp4 / webm / m3u8 / blob: / data:）。
func OpenMedia(path string) (*MediaSource, error) { return OpenMediaWith(path, MediaConfig{}) }

// OpenMediaWith 按配置打开媒体。
func OpenMediaWith(path string, cfg MediaConfig) (*MediaSource, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errUnsupportedMedia
	}
	s := &MediaSource{path: path, cfg: cfg}
	s.info = MediaInfo{Path: path, Backend: "browser"}
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		s.info.Format = strings.TrimPrefix(ext, ".")
	}
	s.id = overlayID("video")

	el := overlayEl(s.id, "video")
	el.Set("preload", "auto")
	el.Set("playsInline", true)
	el.Set("controls", false)
	// 初始静音：浏览器只允许静音媒体自动播放，播放成功后再恢复业务音量
	el.Set("muted", true)
	st := el.Get("style")
	st.Set("objectFit", "contain")
	st.Set("background", "#000")
	st.Set("pointerEvents", "none")
	s.el = el

	// .m3u8（HLS）：Chrome/Edge/Firefox 需要 hls.js，Safari 原生支持。
	switch {
	case !isHLSPath(path):
		el.Set("src", path)
	case nativeHLS(el):
		el.Set("src", path)
	default:
		if err := s.attachHLS(); err != nil {
			el.Set("src", path)
		}
	}

	meta := js.FuncOf(func(this js.Value, args []js.Value) any {
		w := el.Get("videoWidth").Int()
		h := el.Get("videoHeight").Int()
		d := el.Get("duration").Float()
		if w > 0 {
			s.info.Width, s.info.Height, s.info.HasVideo = w, h, true
		}
		if d > 0 && !isInf(d) {
			s.info.Duration = time.Duration(d * float64(time.Second))
		}
		s.info.HasAudio = !cfg.NoAudio
		s.ready = true
		s.applyVolume()
		RequestRedraw()
		return nil
	})
	fail := js.FuncOf(func(this js.Value, args []js.Value) any {
		s.err = fmt.Errorf("ui: 无法播放 %s（浏览器原生格式仅 mp4/webm 等；HLS 已尝试 hls.js，请检查流地址与网络）",
			filepath.Base(path))
		RequestRedraw()
		return nil
	})
	s.funcs = append(s.funcs, meta, fail)
	el.Call("addEventListener", "loadedmetadata", meta)
	el.Call("addEventListener", "loadeddata", meta)
	el.Call("addEventListener", "error", fail)
	return s, nil
}

// isInf 判断无穷大（直播流的 duration 为 Infinity）。
func isInf(f float64) bool {
	return f > 1e17 || f < -1e17
}

// Info 探测信息（元数据到达后包含尺寸与时长）。
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

// Size 画面尺寸（元数据未知时 0,0）。
func (s *MediaSource) Size() (int, int) {
	if s == nil {
		return 0, 0
	}
	return s.info.Width, s.info.Height
}

// Duration 时长（未知返回 0）。
func (s *MediaSource) Duration() time.Duration {
	if s == nil {
		return 0
	}
	return s.info.Duration
}

// HasAudio 是否有音轨。
func (s *MediaSource) HasAudio() bool {
	return s != nil && s.info.HasAudio && !s.cfg.NoAudio
}

// Err 打开/解码错误。
func (s *MediaSource) Err() error {
	if s == nil {
		return errUnsupportedMedia
	}
	return s.err
}

// Frame 取指定时间点画面：浏览器端画面由 DOM 元素显示，这里只负责把
// 播放位置同步给 <video>（跳转/拖动进度条），返回 nil 表示「无新帧要绘制」。
func (s *MediaSource) Frame(pos time.Duration) (*VideoFrame, error) {
	if s == nil || s.el.IsUndefined() || s.el.IsNull() {
		return nil, nil
	}
	if s.err != nil {
		return nil, s.err
	}
	cur := s.el.Get("currentTime").Float()
	want := pos.Seconds()
	if diff := cur - want; diff > 0.35 || diff < -0.35 {
		// 拖动进度条时 60fps 连续 seek 会让浏览器解码器排队，做 120ms 节流；
		// 松手后位置不再变化，节流窗口内的最后一次会补上。
		now := float64(time.Now().UnixNano()) / 1e9
		if now-s.lastSeekAt > 0.12 || pos == s.lastSeekTo {
			s.el.Set("currentTime", want)
			s.lastSeekAt = now
			s.lastSeekTo = pos
		}
	}
	return nil, nil
}

// MediaResume 最小化恢复后重建解码器（浏览器端无需处理）。
func (s *MediaSource) MediaResume() {}

// MediaClock 解码器时钟：以 <video> 的真实播放位置为准（A/V 同步）。
func (s *MediaSource) MediaClock() (time.Duration, bool) {
	if s == nil || s.el.IsUndefined() || s.el.IsNull() || !s.ready {
		return 0, false
	}
	t := s.el.Get("currentTime").Float()
	if t < 0 || isInf(t) {
		return 0, false
	}
	return time.Duration(t * float64(time.Second)), true
}

// MediaScrub 拖动定位状态（浏览器端即时 seek，无需额外处理）。
func (s *MediaSource) MediaScrub(v bool) {}

// MediaPlay 播放/暂停。
func (s *MediaSource) MediaPlay(playing bool) {
	if s == nil || s.el.IsUndefined() || s.el.IsNull() {
		return
	}
	if !playing {
		s.el.Call("pause")
		return
	}
	// 浏览器自动播放策略：有声 play() 会被拒绝（NotAllowedError）。
	// 统一「静音起播」，promise 成功后立刻按业务期望恢复音量与静音状态。
	s.el.Set("muted", true)
	res := s.el.Call("play")
	if res.IsUndefined() || res.IsNull() {
		s.applyWanted()
		return
	}
	ok := js.FuncOf(func(this js.Value, args []js.Value) any {
		s.applyWanted()
		return nil
	})
	bad := js.FuncOf(func(this js.Value, args []js.Value) any {
		// 仍被拒绝（例如没有用户手势且被策略拦截）：保持静音后再试一次
		s.el.Set("muted", true)
		s.el.Call("play")
		RequestRedraw()
		return nil
	})
	s.funcs = append(s.funcs, ok, bad)
	res.Call("then", ok)
	res.Call("catch", bad)
}

// isHLSPath 是否 HLS 播放列表（.m3u8，忽略查询串）。
func isHLSPath(path string) bool {
	p := strings.ToLower(path)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	return strings.HasSuffix(p, ".m3u8")
}

// nativeHLS 浏览器是否原生支持 HLS（Safari / iOS WebKit 支持，Chrome 系返回空串）。
func nativeHLS(el js.Value) bool {
	v := el.Call("canPlayType", "application/vnd.apple.mpegurl")
	return v.Type() == js.TypeString && v.String() != ""
}

// attachHLS 用 hls.js 把 m3u8 挂到 <video> 上；hls.js 未加载或浏览器不支持 MSE 时返回错误。
func (s *MediaSource) attachHLS() error {
	hls := js.Global().Get("Hls")
	if hls.IsUndefined() || hls.IsNull() {
		return errors.New("ui: hls.js 未加载（把 hls.min.js 放到 index.html 同目录）")
	}
	if ok := hls.Call("isSupported"); ok.Type() == js.TypeBoolean && !ok.Bool() {
		return errors.New("ui: 当前浏览器不支持 MSE，无法用 hls.js 播放 HLS")
	}
	inst := hls.New()
	inst.Call("loadSource", s.path)
	inst.Call("attachMedia", s.el)
	s.hls = inst

	errFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 2 {
			return nil
		}
		if fatal := args[1].Get("fatal"); fatal.Type() == js.TypeBoolean && fatal.Bool() {
			detail := args[1].Get("details")
			s.err = fmt.Errorf("ui: HLS 播放失败（%s）", detail.String())
			RequestRedraw()
		}
		return nil
	})
	s.funcs = append(s.funcs, errFn)
	if events := hls.Get("Events"); !events.IsUndefined() && !events.IsNull() {
		inst.Call("on", events.Get("ERROR"), errFn)
	}
	return nil
}

// applyWanted 把业务期望的音量/静音写入元素（播放已放行，不会再被策略拦截）。
// 没有用户手势时保持静音：Chrome 会把「无手势的有声播放」直接暂停。
func (s *MediaSource) applyWanted() {
	if s == nil || s.el.IsUndefined() || s.el.IsNull() {
		return
	}
	v := s.wantVol
	if v <= 0 {
		v = 1
	}
	s.el.Set("volume", float64(v))
	muted := s.wantMuted || s.cfg.NoAudio
	if !muted {
		nav := js.Global().Get("navigator")
		if !nav.IsUndefined() && !nav.IsNull() {
			if ua := nav.Get("userActivation"); !ua.IsUndefined() && !ua.IsNull() {
				if act := ua.Get("hasBeenActive"); act.Type() == js.TypeBoolean && !act.Bool() {
					muted = true
				}
			}
		}
	}
	s.el.Set("muted", muted)
}

// MediaVolume 音量与静音（0~1）。播放中立即生效，未播放时先记录。
func (s *MediaSource) MediaVolume(v float32, muted bool) {
	if s == nil || s.el.IsUndefined() || s.el.IsNull() {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	s.wantVol, s.wantMuted = v, muted
	if paused := s.el.Get("paused"); paused.Type() == js.TypeBoolean && paused.Bool() {
		return
	}
	s.applyWanted()
}

// applyVolume 元数据到达后补写一次音量。
func (s *MediaSource) applyVolume() {
	if s.wantVol > 0 || s.cfg.NoAudio || s.wantMuted {
		s.applyWanted()
	}
}

// MediaSpeed 播放速率。
func (s *MediaSource) MediaSpeed(v float32) {
	if s == nil || s.el.IsUndefined() || s.el.IsNull() {
		return
	}
	if v <= 0 {
		v = 1
	}
	s.el.Set("playbackRate", float64(v))
}

// Close 关闭媒体源：停止播放并移除 DOM 元素。
func (s *MediaSource) Close() {
	if s == nil {
		return
	}
	if !s.hls.IsUndefined() && !s.hls.IsNull() {
		s.hls.Call("destroy")
		s.hls = js.Undefined()
	}
	if !s.el.IsUndefined() && !s.el.IsNull() {
		s.el.Call("pause")
		s.el.Set("src", "")
		s.el.Call("load")
	}
	for _, f := range s.funcs {
		f.Release()
	}
	s.funcs = nil
	if s.id != "" {
		dropOverlayEl(s.id)
		s.id = ""
	}
}
