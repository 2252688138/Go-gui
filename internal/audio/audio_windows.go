//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package audio

import "gogui/internal/win32"

// Open 打开默认波形输出设备（WinMM waveOut）。
// rate 采样率，ch 声道数，buffers 缓冲块数，framesPerBuffer 每块帧数。
func Open(rate, ch, buffers, framesPerBuffer int) (Device, error) {
	a, err := win32.NewAudioOut(rate, ch, buffers, framesPerBuffer)
	if err != nil {
		return nil, err
	}
	// *win32.AudioOut 的方法集已满足 Device 接口，无需包装。
	return a, nil
}

// Devices 系统波形输出设备数量（0 表示无可用音频设备）。
func Devices() int { return win32.AudioDevices() }
