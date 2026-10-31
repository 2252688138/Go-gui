//go:build !windows && !((linux || freebsd || openbsd || netbsd) && !android)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package audio

import "errors"

// ErrNoDevice 本平台尚无音频输出后端（macOS / 移动端 / 浏览器）。
var ErrNoDevice = errors.New("audio: 本平台尚未接入音频输出后端")

// Open 本平台无音频输出后端，返回错误让调用方降级为「无声音、仅画面」。
func Open(rate, ch, buffers, framesPerBuffer int) (Device, error) {
	return nil, ErrNoDevice
}

// Devices 本平台音频输出设备数量（恒为 0）。
func Devices() int { return 0 }
