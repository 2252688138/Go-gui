//go:build !windows && !js && (!(linux || freebsd) || android)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// 非 Windows 桌面端（Linux / macOS / BSD / 移动端）的媒体能力垫片。
//
// 解码后端（Windows 走 Media Foundation / ffmpeg 进程 + waveOut）尚未接入这些平台，
// 这里给出与 Windows 端**完全一致的类型与方法签名**，保证同一套 UI 代码可以原样
// 编译、原样运行：调用结果是「无媒体 / 无画面」，界面走占位与降级分支，绝不 panic。
// 业务代码可用 HasCap(CapVideo) / HasCap(CapAudio) 判断后再决定是否给出入口。

// errUnsupportedMedia 当前平台没有媒体解码后端。
var errUnsupportedMedia = errors.New("ui: 当前平台尚未接入媒体解码后端（可用 HasCap(CapVideo) 判断）")

// MediaFilters 文件对话框用的媒体过滤器（与 Windows 端一致）。
const MediaFilters = "视频|*.mp4;*.m4v;*.mkv;*.mov;*.avi;*.webm;*.flv;*.ts;*.m2ts;*.mts;*.m3u8|" +
	"音频|*.mp3;*.flac;*.wav;*.m4a;*.aac;*.ogg;*.opus|所有文件|*.*"

var mediaExts = []string{
	".mp4", ".m4v", ".mkv", ".mov", ".avi", ".webm", ".flv",
	".ts", ".m2ts", ".mts", ".m3u8", ".mp3", ".flac", ".wav", ".m4a", ".aac", ".ogg", ".opus",
}

// IsMediaPath 按扩展名判断是否媒体文件。
func IsMediaPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range mediaExts {
		if ext == e {
			return true
		}
	}
	return false
}

// MediaFormats 支持的媒体容器列表（界面展示用）。
func MediaFormats() []string {
	return []string{"mp4", "mkv", "mov", "avi", "webm", "flv", "ts", "m3u8", "mp3", "flac", "wav", "m4a"}
}

// MediaInfo 媒体探测结果（字段与 Windows 端一致）。
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
	// Backend 实际使用的解码后端；本平台无后端，恒为 ""。
	Backend string
}

// MediaConfig 解码参数（与 Windows 端一致，本平台忽略）。
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

// DecoderPreference 解码器偏好（本平台无后端）。
func DecoderPreference() string { return "" }

// ProbeMedia 探测媒体信息（本平台无后端，返回不支持）。
func ProbeMedia(path string) (*MediaInfo, error) { return ProbeMediaWith(path, "") }

// ProbeMediaWith 指定 ffprobe 路径探测（本平台无后端，返回不支持）。
func ProbeMediaWith(path, ffprobe string) (*MediaInfo, error) { return nil, errUnsupportedMedia }

// MediaSource 媒体源：占位实现，实现 VideoSource 接口但不产生画面。
type MediaSource struct {
	path string
	info MediaInfo
	cfg  MediaConfig
	err  error
}

// OpenMedia 打开媒体（本平台无后端）。
func OpenMedia(path string) (*MediaSource, error) { return OpenMediaWith(path, MediaConfig{}) }

// OpenMediaWith 按配置打开媒体（本平台无后端）。
func OpenMediaWith(path string, cfg MediaConfig) (*MediaSource, error) {
	return &MediaSource{path: path, cfg: cfg, err: errUnsupportedMedia}, nil
}

// Info 探测信息（零值）。
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

// Size 画面尺寸（无画面 → 0,0）。
func (s *MediaSource) Size() (int, int) { return 0, 0 }

// Duration 时长（未知 → 0）。
func (s *MediaSource) Duration() time.Duration { return 0 }

// HasAudio 是否有音轨。
func (s *MediaSource) HasAudio() bool { return false }

// Err 打开/解码错误。
func (s *MediaSource) Err() error {
	if s == nil {
		return errUnsupportedMedia
	}
	return s.err
}

// Frame 取指定时间点画面（无解码器 → nil, nil，保留上一帧）。
func (s *MediaSource) Frame(pos time.Duration) (*VideoFrame, error) { return nil, nil }

// MediaResume 最小化恢复后重建解码器（空操作）。
func (s *MediaSource) MediaResume() {}

// MediaClock 解码器时钟（无效）。
func (s *MediaSource) MediaClock() (time.Duration, bool) { return 0, false }

// MediaScrub 拖动定位状态（空操作）。
func (s *MediaSource) MediaScrub(v bool) {}

// MediaPlay 播放/暂停（空操作）。
func (s *MediaSource) MediaPlay(playing bool) {}

// Close 关闭媒体源。
func (s *MediaSource) Close() {}
