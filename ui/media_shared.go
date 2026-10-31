// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"sync"
	"time"
)

// 媒体层跨平台共用的小工具：系统解码器句柄接口与并发等待。
// 具体后端（Windows Media Foundation）见 system_decoder.go，
// 非 Windows 平台实现见 system_decoder_other.go（恒无系统解码器）。

// sysSource 系统解码器句柄。
//
// 只有「有系统解码器」的平台才会给 s.sys 赋非 nil 值（目前是 Windows 的
// Media Foundation），其它平台恒为 nil，媒体层据此走 ffmpeg 路径。
// 接口只描述媒体层/系统解码协程用到的能力，避免把 win32 类型泄漏到跨平台代码里。
type sysSource interface {
	Seek(d time.Duration) error
	VideoStreamIndex() int
	AudioStreamIndex() int
	ReadStream(stream int) (kind int, data []byte, ts time.Duration, eof bool, err error)
	Size() (int, int)
	Close()
}

// waitDone 等待 WaitGroup 结束，最多等 d；返回是否在超时前等到。
func waitDone(wg *sync.WaitGroup, d time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}
