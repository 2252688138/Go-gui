//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// WinMM waveOut 音频输出（纯 Go 动态绑定，无 cgo）。
// 用法：NewAudioOut(48000, 2) → Write(pcm16) 阻塞直到有空闲缓冲 → Position() 取已播样本。

const (
	WAVE_MAPPER     = 0xFFFFFFFF
	WAVE_FORMAT_PCM = 1

	WHDR_DONE     = 0x00000001
	WHDR_PREPARED = 0x00000002

	CALLBACK_NULL = 0

	TIME_SAMPLES = 0x0002

	WAVEOUT_ERR_CLOSED = "waveOut 已关闭"
)

// WAVEFORMATEX 标准 PCM 格式描述。
type WAVEFORMATEX struct {
	WFormatTag      uint16
	NChannels       uint16
	NSamplesPerSec  uint32
	NAvgBytesPerSec uint32
	NBlockAlign     uint16
	WBitsPerSample  uint16
	CbSize          uint16
}

// WAVEHDR 一块输出缓冲的描述。
type WAVEHDR struct {
	LpData          uintptr
	DwBufferLength  uint32
	DwBytesRecorded uint32
	DwUser          uintptr
	DwFlags         uint32
	DwLoops         uint32
	LpNext          uintptr
	Reserved        uintptr
}

// MMTIME 播放位置（wType=TIME_SAMPLES 时 U 为样本数）。
// sizeof(MMTIME) = 4 + 8 = 12（union 为 SMPTE 占 8 字节），传 8 会被 waveOutGetPosition 拒绕。
type MMTIME struct {
	WType uint32
	U     uint32
	Pad   uint32
}

var libWinmm uintptr

var (
	waveOutOpen            func(hwo *uintptr, deviceID uint32, format *WAVEFORMATEX, callback, instance uintptr, flags uint32) uint32
	waveOutClose           func(hwo uintptr) uint32
	waveOutPrepareHeader   func(hwo uintptr, hdr *WAVEHDR, size uint32) uint32
	waveOutUnprepareHeader func(hwo uintptr, hdr *WAVEHDR, size uint32) uint32
	waveOutWrite           func(hwo uintptr, hdr *WAVEHDR, size uint32) uint32
	waveOutReset           func(hwo uintptr) uint32
	waveOutPause           func(hwo uintptr) uint32
	waveOutRestart         func(hwo uintptr) uint32
	waveOutSetVolume       func(hwo uintptr, volume uint32) uint32
	waveOutGetPosition     func(hwo uintptr, mmt *MMTIME, size uint32) uint32
	waveOutGetNumDevs      func() uint32
)

func init() {
	libWinmm = loadLib("winmm.dll")
	bind(libWinmm, &waveOutOpen, "waveOutOpen")
	bind(libWinmm, &waveOutClose, "waveOutClose")
	bind(libWinmm, &waveOutPrepareHeader, "waveOutPrepareHeader")
	bind(libWinmm, &waveOutUnprepareHeader, "waveOutUnprepareHeader")
	bind(libWinmm, &waveOutWrite, "waveOutWrite")
	bind(libWinmm, &waveOutReset, "waveOutReset")
	bind(libWinmm, &waveOutPause, "waveOutPause")
	bind(libWinmm, &waveOutRestart, "waveOutRestart")
	bind(libWinmm, &waveOutSetVolume, "waveOutSetVolume")
	bind(libWinmm, &waveOutGetPosition, "waveOutGetPosition")
	bind(libWinmm, &waveOutGetNumDevs, "waveOutGetNumDevs")
}

// ErrAudioClosed 音频设备已关闭。
var ErrAudioClosed = errors.New(WAVEOUT_ERR_CLOSED)

// AudioOut waveOut 音频输出：固定数量的循环缓冲，Write 阻塞等待空闲缓冲（天然按实时节奏限流）。
type AudioOut struct {
	mu     sync.Mutex
	hwo    uintptr
	rate   int
	ch     int
	bits   int
	bufs   [][]byte
	hdrs   []WAVEHDR
	used   []bool
	vol    float32
	paused bool
	closed bool
}

// AudioDevices 系统波形输出设备数量（0 表示无可用音频设备）。
func AudioDevices() int {
	if waveOutGetNumDevs == nil {
		return 0
	}
	return int(waveOutGetNumDevs())
}

// NewAudioOut 打开默认波形输出设备。rate 采样率，ch 声道数，buffers 缓冲数量，framesPerBuffer 每块帧数。
func NewAudioOut(rate, ch, buffers, framesPerBuffer int) (*AudioOut, error) {
	if rate <= 0 {
		rate = 48000
	}
	if ch <= 0 {
		ch = 2
	}
	if buffers < 2 {
		buffers = 2
	}
	if framesPerBuffer < 256 {
		framesPerBuffer = 256
	}
	if AudioDevices() == 0 {
		return nil, errors.New("没有可用的音频输出设备")
	}
	f := &WAVEFORMATEX{
		WFormatTag:      WAVE_FORMAT_PCM,
		NChannels:       uint16(ch),
		NSamplesPerSec:  uint32(rate),
		WBitsPerSample:  16,
		NBlockAlign:     uint16(ch * 2),
		NAvgBytesPerSec: uint32(rate * ch * 2),
	}
	var hwo uintptr
	if r := waveOutOpen(&hwo, WAVE_MAPPER, f, 0, 0, CALLBACK_NULL); r != 0 {
		return nil, fmt.Errorf("waveOutOpen 失败（code %d）", r)
	}
	a := &AudioOut{hwo: hwo, rate: rate, ch: ch, bits: 16, vol: 1}
	size := framesPerBuffer * ch * 2
	for i := 0; i < buffers; i++ {
		buf := make([]byte, size)
		hdr := WAVEHDR{LpData: uintptr(unsafe.Pointer(&buf[0]))}
		if r := waveOutPrepareHeader(hwo, &hdr, uint32(unsafe.Sizeof(hdr))); r != 0 {
			a.Close()
			return nil, fmt.Errorf("waveOutPrepareHeader 失败（code %d）", r)
		}
		a.bufs = append(a.bufs, buf)
		a.hdrs = append(a.hdrs, hdr)
		a.used = append(a.used, false)
	}
	a.SetVolume(1)
	return a, nil
}

// Rate 采样率。
func (a *AudioOut) Rate() int { return a.rate }

// Write 写入 16bit PCM（阻塞到有空闲缓冲；关闭后返回 ErrAudioClosed）。
func (a *AudioOut) Write(pcm []byte) error {
	for len(pcm) > 0 {
		i, err := a.acquire()
		if err != nil {
			return err
		}
		n := copy(a.bufs[i], pcm)
		h := &a.hdrs[i]
		h.DwBufferLength = uint32(n)
		h.DwFlags &^= WHDR_DONE
		if r := waveOutWrite(a.hwo, h, uint32(unsafe.Sizeof(*h))); r != 0 {
			return fmt.Errorf("waveOutWrite 失败（code %d）", r)
		}
		pcm = pcm[n:]
	}
	return nil
}

// acquire 取一块已完成播放的缓冲（暂停时等待）。
func (a *AudioOut) acquire() (int, error) {
	for {
		a.mu.Lock()
		if a.closed {
			a.mu.Unlock()
			return -1, ErrAudioClosed
		}
		if a.paused {
			a.mu.Unlock()
			time.Sleep(4 * time.Millisecond)
			continue
		}
		found := -1
		for i := range a.hdrs {
			if !a.used[i] || a.hdrs[i].DwFlags&WHDR_DONE != 0 {
				found = i
				a.used[i] = true
				break
			}
		}
		a.mu.Unlock()
		if found >= 0 {
			return found, nil
		}
		time.Sleep(3 * time.Millisecond)
	}
}

// Reset 立即停止播放并丢弃所有排队数据（跳转时调用）。
func (a *AudioOut) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.hwo == 0 {
		return
	}
	waveOutReset(a.hwo)
	for i := range a.hdrs {
		a.hdrs[i].DwFlags |= WHDR_DONE
		a.used[i] = false
	}
}

// Pause 暂停/恢复播放（暂停期间 Write 阻塞）。
func (a *AudioOut) Pause(v bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.hwo == 0 || a.paused == v {
		return
	}
	a.paused = v
	if v {
		waveOutPause(a.hwo)
	} else {
		waveOutRestart(a.hwo)
	}
}

// SetVolume 设置音量 0~1（同时写左右声道）。
func (a *AudioOut) SetVolume(v float32) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	a.mu.Lock()
	a.vol = v
	hwo := a.hwo
	a.mu.Unlock()
	if hwo == 0 {
		return
	}
	level := uint32(v*65535) * 0x10001
	waveOutSetVolume(hwo, level)
}

// Position 已播放时长（自 Reset 起按设备样本计数）。
func (a *AudioOut) Position() (time.Duration, bool) {
	a.mu.Lock()
	hwo, rate, paused := a.hwo, a.rate, a.paused
	a.mu.Unlock()
	if hwo == 0 || paused {
		return 0, false
	}
	mm := MMTIME{WType: TIME_SAMPLES}
	if waveOutGetPosition(hwo, &mm, uint32(unsafe.Sizeof(mm))) != 0 {
		return 0, false
	}
	return time.Duration(float64(mm.U) / float64(rate) * float64(time.Second)), true
}

// Close 关闭设备（幂等）。
func (a *AudioOut) Close() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	hwo := a.hwo
	a.mu.Unlock()
	if hwo == 0 {
		return nil
	}
	waveOutReset(hwo)
	for i := range a.hdrs {
		waveOutUnprepareHeader(hwo, &a.hdrs[i], uint32(unsafe.Sizeof(a.hdrs[i])))
	}
	waveOutClose(hwo)
	a.mu.Lock()
	a.hwo = 0
	a.mu.Unlock()
	return nil
}
