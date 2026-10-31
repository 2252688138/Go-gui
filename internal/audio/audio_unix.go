//go:build (linux || freebsd || openbsd || netbsd) && !android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package audio

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// Linux/BSD 端音频输出：把 s16le PCM 管道写进系统播放器的 stdin。
//
// 为什么用子进程而不是直接接 ALSA/PulseAudio：项目坚持零 cgo，而
// libasound / libpulse 都是 C 库；系统播放器到处都有，且自带混音与设备路由，
// 行为与桌面其它程序一致。写入管道天然有背压，因此能像设备一样限流。

var (
	// ErrClosed 设备已关闭。
	ErrClosed = errors.New("audio: 设备已关闭")
	// ErrNoDevice 本机找不到可用的音频输出后端。
	ErrNoDevice = errors.New("audio: 未找到音频输出后端（可安装 pulseaudio-utils / alsa-utils / pipewire）")
)

// Out 通过播放器子进程输出 PCM 的设备实现。
type Out struct {
	rate, ch int

	cmd  *exec.Cmd
	w    *io.PipeWriter
	done chan struct{}

	mu     sync.Mutex
	paused bool
	closed bool
	// vol 音量 0~1：非 1 时在 Write 里按样本缩放（播放器不提供运行时改音量）
	vol float32

	// 播放时钟：played 为已计满的时长，resumeAt 为上一次开始/恢复的时刻
	played   time.Duration
	resumeAt time.Time
	// latency 下游缓冲延迟估计，Position 会扣掉，避免画面/进度条领先声音
	latency time.Duration
}

// player 一种播放器后端的启动方式。
type player struct {
	name string
	args func(rate, ch int) []string
}

// players 按优先级列出可用后端及其参数（都从 stdin 读裸 PCM）。
//
//	paplay：PulseAudio 客户端（WSLg / 绝大多数桌面默认有 pulse 服务）；
//	aplay ：ALSA 工具（纯 ALSA 环境），"-" 表示 stdin；
//	pw-play：PipeWire 的播放器，"-" 同样表示 stdin。
func players() []player {
	return []player{
		{"paplay", func(r, c int) []string {
			return []string{"--raw", "--format=s16le", "--rate=" + strconv.Itoa(r), "--channels=" + strconv.Itoa(c)}
		}},
		{"aplay", func(r, c int) []string {
			return []string{"-q", "-t", "raw", "-f", "S16_LE", "-r", strconv.Itoa(r), "-c", strconv.Itoa(c), "-"}
		}},
		{"pw-play", func(r, c int) []string {
			return []string{"--raw", "--format=s16le", "--rate=" + strconv.Itoa(r), "--channels=" + strconv.Itoa(c), "-"}
		}},
	}
}

// Devices 本机可用的音频输出后端数量（0 = 无可用后端）。
func Devices() int {
	for _, p := range players() {
		if _, err := exec.LookPath(p.name); err == nil {
			return 1
		}
	}
	return 0
}

// startPlayer 拉起一个播放器子进程并返回其 stdin 管道。
func startPlayer(rate, ch int) (*exec.Cmd, *io.PipeWriter, chan struct{}, error) {
	var lastErr error
	for _, p := range players() {
		if _, err := exec.LookPath(p.name); err != nil {
			lastErr = err
			continue
		}
		pr, pw := io.Pipe()
		cmd := exec.Command(p.name, p.args(rate, ch)...)
		cmd.Stdin = pr
		if err := cmd.Start(); err != nil {
			_ = pr.Close()
			lastErr = err
			continue
		}
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		return cmd, pw, done, nil
	}
	if lastErr == nil {
		lastErr = ErrNoDevice
	}
	return nil, nil, nil, fmt.Errorf("%w：%v", ErrNoDevice, lastErr)
}

// Open 打开默认输出后端。参数与 Windows 端保持一致（buffers / framesPerBuffer
// 仅用于选择下游缓冲量级，管道实现由播放器自行排队）。
func Open(rate, ch, buffers, framesPerBuffer int) (Device, error) {
	if rate <= 0 {
		rate = 48000
	}
	if ch <= 0 {
		ch = 2
	}
	cmd, w, done, err := startPlayer(rate, ch)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &Out{
		rate: rate, ch: ch,
		cmd: cmd, w: w, done: done,
		vol:      1,
		resumeAt: now,
		// 播放器内部缓冲：aplay 默认约 500ms，pulse/pipewire 端到端约 100~300ms。
		// 取 250ms 作为折中，Position 扣掉它后音画/进度基本对齐。
		latency: 250 * time.Millisecond,
	}, nil
}

// Rate 采样率。
func (o *Out) Rate() int { return o.rate }

// Write 写入 s16le PCM。下游缓冲满时阻塞，这是播放节奏的限流点。
func (o *Out) Write(pcm []byte) error {
	if len(pcm) == 0 {
		return nil
	}
	for len(pcm) > 0 {
		o.mu.Lock()
		closed, paused, vol := o.closed, o.paused, o.vol
		o.mu.Unlock()
		if closed {
			return ErrClosed
		}
		if paused {
			time.Sleep(4 * time.Millisecond)
			continue
		}
		chunk := pcm
		var scaled []byte
		if vol < 0.999 {
			scaled = scaleS16(pcm, vol)
			chunk = scaled
		}
		n, err := o.w.Write(chunk)
		pcm = pcm[n:]
		if err != nil {
			return err
		}
	}
	return nil
}

// scaleS16 按音量缩放 s16le 采样（返回新切片，不改动入参）。
func scaleS16(pcm []byte, vol float32) []byte {
	out := make([]byte, len(pcm))
	for i := 0; i+1 < len(pcm); i += 2 {
		s := int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8)
		v := int32(float32(s) * vol)
		if v > 32767 {
			v = 32767
		}
		if v < -32768 {
			v = -32768
		}
		out[i] = byte(uint16(int16(v)))
		out[i+1] = byte(uint16(int16(v)) >> 8)
	}
	return out
}

// Reset 立即停止并丢弃下游已排队数据：重启播放器进程，并把播放时钟归零。
func (o *Out) Reset() {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return
	}
	oldW, oldCmd, oldDone := o.w, o.cmd, o.done
	o.w, o.cmd, o.done = nil, nil, nil
	o.played = 0
	o.resumeAt = time.Now()
	o.mu.Unlock()

	stopPlayer(oldW, oldCmd, oldDone)

	cmd, w, done, err := startPlayer(o.rate, o.ch)
	if err != nil {
		return // 重启失败：后续 Write 会发现 w 为 nil 而报错，交给上层重建
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		stopPlayer(w, cmd, done)
		return
	}
	o.w, o.cmd, o.done = w, cmd, done
	o.mu.Unlock()
}

// stopPlayer 关闭管道并结束播放器（先 EOF 优雅退出，1 秒不退就强杀）。
func stopPlayer(w *io.PipeWriter, cmd *exec.Cmd, done chan struct{}) {
	if w != nil {
		_ = w.Close()
	}
	if cmd == nil || cmd.Process == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		_ = cmd.Process.Kill()
	}
}

// Pause 暂停/恢复：暂停期间 Write 阻塞、播放时钟冻结。
func (o *Out) Pause(v bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed || o.paused == v {
		return
	}
	now := time.Now()
	if v {
		o.played += now.Sub(o.resumeAt)
		o.paused = true
		return
	}
	o.resumeAt = now
	o.paused = false
}

// SetVolume 设置音量 0~1（管道无法运行时调音量，改为在 Write 里缩放样本）。
func (o *Out) SetVolume(v float32) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	o.mu.Lock()
	o.vol = v
	o.mu.Unlock()
}

// Position 已播放时长（按墙钟推算并扣掉下游缓冲延迟）。
func (o *Out) Position() (time.Duration, bool) {
	o.mu.Lock()
	closed, paused := o.closed, o.paused
	played, resumeAt := o.played, o.resumeAt
	o.mu.Unlock()
	if closed {
		return 0, false
	}
	pos := played
	if !paused {
		pos += time.Since(resumeAt)
	}
	pos -= o.latency
	if pos < 0 {
		pos = 0
	}
	return pos, true
}

// Close 关闭设备（幂等）。
func (o *Out) Close() error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	o.closed = true
	w, cmd, done := o.w, o.cmd, o.done
	o.w, o.cmd, o.done = nil, nil, nil
	o.mu.Unlock()
	stopPlayer(w, cmd, done)
	return nil
}
