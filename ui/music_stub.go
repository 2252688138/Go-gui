//go:build !windows && (!(linux || freebsd) || android)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"path/filepath"
	"strings"
	"time"
)

// 非 Windows 桌面端的音乐解码源垫片。
//
// Windows 端是「ffmpeg → s16le PCM → waveOut」+ 频谱；这些平台尚未接入音频输出，
// 这里保留同一份 API 与推进逻辑（按墙钟走时间轴），让 MusicPlayer 组件能正常渲染、
// 进度条能走、频谱为空（自然衰减），用户点播放不会报错。

// MusicFilters 文件对话框用的音频过滤器。
const MusicFilters = "音频|*.mp3;*.flac;*.wav;*.m4a;*.aac;*.ogg;*.opus;*.wma|所有文件|*.*"

var musicExts = []string{".mp3", ".flac", ".wav", ".m4a", ".aac", ".ogg", ".opus", ".wma", ".ape"}

// IsMusicPath 按扩展名判断是否音频文件。
func IsMusicPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range musicExts {
		if ext == e {
			return true
		}
	}
	return false
}

// MusicSource 纯音频解码源（本平台为占位实现：无声音、按墙钟推进）。
type MusicSource struct {
	path  string
	info  MediaInfo
	cfg   MediaConfig
	an    *Spectrum
	pos   time.Duration // 当前播放位置
	last  time.Time
	rate  float64
	vol   float32
	muted bool
	loop  bool
	play  bool
	ended bool
}

// OpenMusic 打开音频文件（本平台无解码后端，返回占位源）。
func OpenMusic(path string) (*MusicSource, error) { return OpenMusicWith(path, MediaConfig{}) }

// OpenMusicWith 按配置打开音频文件（本平台无解码后端，返回占位源）。
func OpenMusicWith(path string, cfg MediaConfig) (*MusicSource, error) {
	return &MusicSource{
		path: path,
		cfg:  cfg,
		an:   newSpectrum(SpectrumBands, 48000),
		rate: 1,
		vol:  1,
		last: time.Now(),
	}, nil
}

// OpenMusicPlayer 打开音频并立即播放（本平台无音频输出，仅推进时间轴）。
func OpenMusicPlayer(path string) (*MusicSource, error) {
	s, err := OpenMusic(path)
	if err != nil {
		return nil, err
	}
	s.Play()
	return s, nil
}

// Path 音频路径。
func (s *MusicSource) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Info 探测信息。
func (s *MusicSource) Info() MediaInfo {
	if s == nil {
		return MediaInfo{}
	}
	return s.info
}

// Duration 时长（无解码后端 → 0，界面显示 --:--）。
func (s *MusicSource) Duration() time.Duration { return 0 }

// HasAudio 是否有音轨。
func (s *MusicSource) HasAudio() bool { return s != nil }

// Err 错误（占位源无错误）。
func (s *MusicSource) Err() error { return nil }

// Spectrum 频谱分析器（无声源，能量恒为 0）。
func (s *MusicSource) Spectrum() *Spectrum {
	if s == nil {
		return nil
	}
	return s.an
}

// tick 按墙钟推进播放位置（无解码器时的统一实现）。
func (s *MusicSource) tick() {
	now := time.Now()
	if !s.play || s.ended {
		s.last = now
		return
	}
	dt := now.Sub(s.last)
	s.last = now
	s.pos += time.Duration(float64(dt) * s.rate)
	if d := s.Duration(); d > 0 && s.pos >= d {
		if s.loop {
			s.pos = 0
		} else {
			s.pos = d
			s.play = false
			s.ended = true
		}
	}
}

// Pos 当前播放位置。
func (s *MusicSource) Pos() time.Duration {
	if s == nil {
		return 0
	}
	s.tick()
	return s.pos
}

// Progress 播放进度 0~1。
func (s *MusicSource) Progress() float32 {
	if s == nil {
		return 0
	}
	d := s.Duration()
	if d <= 0 {
		return 0
	}
	return float32(s.Pos()) / float32(d)
}

// Playing 是否正在播放。
func (s *MusicSource) Playing() bool { return s != nil && s.play }

// Play 开始播放。
func (s *MusicSource) Play() {
	if s == nil {
		return
	}
	s.play, s.ended, s.last = true, false, time.Now()
}

// Pause 暂停。
func (s *MusicSource) Pause() {
	if s == nil {
		return
	}
	s.tick()
	s.play = false
}

// TogglePlay 切换播放/暂停。
func (s *MusicSource) TogglePlay() {
	if s == nil {
		return
	}
	if s.play {
		s.Pause()
	} else {
		s.Play()
	}
}

// Seek 跳转到指定位置。
func (s *MusicSource) Seek(pos time.Duration) {
	if s == nil {
		return
	}
	if pos < 0 {
		pos = 0
	}
	s.pos, s.ended, s.last = pos, false, time.Now()
	if s.an != nil {
		s.an.Reset()
	}
}

// SeekRatio 按比例跳转 0~1。
func (s *MusicSource) SeekRatio(v float32) {
	if s == nil {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	if d := s.Duration(); d > 0 {
		s.Seek(time.Duration(float32(d) * v))
	}
}

// Speed 播放速率。
func (s *MusicSource) Speed() float32 {
	if s == nil {
		return 1
	}
	return float32(s.rate)
}

// SetSpeed 设置播放速率（<=0 视为 1）。
func (s *MusicSource) SetSpeed(v float32) {
	if s == nil {
		return
	}
	if v <= 0 {
		v = 1
	}
	s.rate = float64(v)
}

// Volume 音量 0~1。
func (s *MusicSource) Volume() float32 {
	if s == nil {
		return 0
	}
	return s.vol
}

// Muted 是否静音。
func (s *MusicSource) Muted() bool { return s != nil && s.muted }

// SetVolume 设置音量。
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
	s.vol = v
}

// ToggleMute 切换静音。
func (s *MusicSource) ToggleMute() {
	if s == nil {
		return
	}
	s.muted = !s.muted
}

// Loop 是否循环。
func (s *MusicSource) Loop() bool { return s != nil && s.loop }

// SetLoop 设置循环。
func (s *MusicSource) SetLoop(v bool) {
	if s == nil {
		return
	}
	s.loop = v
}

// Tick 每帧推进；返回 true 表示当前曲目已播完（组件据此自动下一首）。
func (s *MusicSource) Tick() bool {
	if s == nil {
		return false
	}
	wasEnded := s.ended
	s.tick()
	return !wasEnded && s.ended
}

// Close 释放音源。
func (s *MusicSource) Close() {
	if s == nil {
		return
	}
	s.play = false
	s.ended = true
}

// NowPlaying 当前曲目信息（标题回退文件名）。
func (s *MusicSource) NowPlaying() MusicTrack {
	if s == nil {
		return MusicTrack{}
	}
	title := s.path
	if i := strings.LastIndexAny(title, `/\`); i >= 0 {
		title = title[i+1:]
	}
	if i := strings.LastIndex(title, "."); i > 0 {
		title = title[:i]
	}
	return MusicTrack{Path: s.path, Title: title}
}
