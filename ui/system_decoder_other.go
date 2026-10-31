//go:build (linux || freebsd) && !android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"time"
)

// Linux/BSD 端「系统解码器」垫片。
//
// 这些平台没有 Media Foundation 这类统一的系统解码器（GStreamer / VLC 需要
// 链接 .so 或装插件，不符合本项目零 cgo 的约束），因此系统解码器路径恒不可用：
//   - openMediaSystem / openMusicSystem 返回明确错误（提示安装 ffmpeg）；
//   - runSysLoop / runMusicSysAudio 永远进不去（s.sys 恒为 nil），
//     保留同签名空实现，让媒体层代码无需平台分支。

var errNoSystemDecoder = errors.New("本平台没有系统解码器，请安装 FFmpeg（ffmpeg / ffprobe 需在 PATH 中）")

// openMediaSystem 用系统解码器打开媒体（本平台无系统解码器）。
func openMediaSystem(path string, cfg MediaConfig) (*MediaSource, error) {
	return nil, errNoSystemDecoder
}

// openMusicSystem 用系统解码器打开音频（本平台无系统解码器）。
func openMusicSystem(path string, cfg MediaConfig) (*MusicSource, error) {
	return nil, errNoSystemDecoder
}

// runSysLoop 系统解码器的读帧协程（本平台不会进入）。
func (s *MediaSource) runSysLoop(gen int, pos time.Duration) {}

// runMusicSysAudio 系统解码器的音频协程（本平台不会进入）。
func (s *MusicSource) runMusicSysAudio(gen int, pos time.Duration) {}
