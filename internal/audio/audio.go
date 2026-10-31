// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package audio 平台无关的 16bit PCM 音频输出。
//
// 媒体层（ui/media.go、ui/music.go）只依赖本包的 Device 接口：
//   - Windows：WinMM waveOut（internal/win32 的 AudioOut，循环缓冲 + 设备时钟）；
//   - Linux/BSD：把 PCM 管道写进系统播放器（paplay / pw-play / aplay）的 stdin；
//   - 其它平台：Open 返回错误，调用方按「无音频输出」降级（画面照常播放）。
//
// Device 的 Write 是**阻塞**语义：下游缓冲满了就等，天然按实时节奏限流，
// 因此调用方可以像写文件一样直接把解码结果灌进来。
package audio

import "time"

// Device 音频输出设备。
type Device interface {
	// Rate 采样率（Hz）。
	Rate() int
	// Write 写入 s16le PCM（阻塞到下游能接收；关闭后返回错误）。
	Write(pcm []byte) error
	// Reset 立即停止并丢弃已排队数据（跳转时调用），随后播放时钟归零。
	Reset()
	// Pause 暂停/恢复（暂停期间 Write 阻塞，播放时钟冻结）。
	Pause(v bool)
	// SetVolume 音量 0~1。
	SetVolume(v float32)
	// Position 已播放时长（自 Open/Reset 起）。ok=false 表示时钟不可用。
	Position() (time.Duration, bool)
	// Close 关闭设备（幂等）。
	Close() error
}
