// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"math"
	"sync"
)

// 频谱分析（平台无关）：从 music.go（Windows 解码源）里独立出来，
// 让所有平台都能用 NewSpectrum + Spectrum 驱动频谱组件；
// 非 Windows 平台由 music_stub.go 的 MusicSource 往这里推 PCM（如有解码器），
// 没有解码器时 Band 自然衰减到 0，界面不会报错。

// ---- 频谱分析 ----

// Spectrum FFT 频谱分析器：解码线程写 PCM，UI 线程每帧读频段能量。
// 频段按对数频率划分（20Hz~20kHz），输出 0~1 的归一化能量，附带峰值回落，
// 可直接驱动柱状 / 镜像频谱。
type Spectrum struct {
	mu    sync.Mutex
	ring  []float32 // 单声道环形缓冲（最新样本覆盖最旧）
	pos   int
	total uint64
	rate  int

	// 以下字段只在 UI 线程访问
	bands  []float32
	peaks  []float32
	lastT  uint64
	hann   []float64
	re, im []float64
	snap   []float32
	edges  []int // 每个频段的 FFT bin 边界
	lo, hi int
}

const (
	// spectrumWin FFT 窗口长度（样本数，2 的幂）。
	spectrumWin = 2048
	// SpectrumBands 默认频段数。
	SpectrumBands = 40
)

// NewSpectrum 创建频谱分析器（bands <= 0 时用默认 40 段）。
func NewSpectrum(bands int) *Spectrum {
	return newSpectrum(bands, 48000)
}

func newSpectrum(bands, rate int) *Spectrum {
	if bands <= 0 {
		bands = SpectrumBands
	}
	if rate <= 0 {
		rate = 48000
	}
	sp := &Spectrum{
		ring:  make([]float32, spectrumWin),
		bands: make([]float32, bands),
		peaks: make([]float32, bands),
		hann:  make([]float64, spectrumWin),
		re:    make([]float64, spectrumWin),
		im:    make([]float64, spectrumWin),
		snap:  make([]float32, spectrumWin),
		rate:  rate,
		lo:    20,
		hi:    20000,
	}
	for i := 0; i < spectrumWin; i++ {
		sp.hann[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(spectrumWin-1))
	}
	sp.buildEdges()
	return sp
}

// buildEdges 按对数频率给每个频段分配 FFT bin 区间（至少 1 个 bin）。
func (sp *Spectrum) buildEdges() {
	n := len(sp.bands)
	sp.edges = make([]int, n+1)
	binHz := float64(sp.rate) / float64(spectrumWin)
	ratio := float64(sp.hi) / float64(sp.lo)
	for i := 0; i <= n; i++ {
		f := float64(sp.lo) * math.Pow(ratio, float64(i)/float64(n))
		b := int(f / binHz)
		if b < 1 {
			b = 1
		}
		if i > 0 && b <= sp.edges[i-1] {
			b = sp.edges[i-1] + 1
		}
		if b > spectrumWin/2-1 {
			b = spectrumWin/2 - 1
		}
		sp.edges[i] = b
	}
}

// PushPCM 写入一段 s16le PCM（解码线程调用，可多声道：取各声道平均）。
func (sp *Spectrum) PushPCM(pcm []byte, channels, rate int) {
	if sp == nil || len(pcm) < 2 {
		return
	}
	if channels <= 0 {
		channels = 2
	}
	if rate > 0 && rate != sp.rate {
		sp.mu.Lock()
		sp.rate = rate
		sp.mu.Unlock()
		sp.buildEdges()
	}
	n := len(pcm) / 2 / channels
	if n <= 0 {
		return
	}
	sp.mu.Lock()
	for i := 0; i < n; i++ {
		var sum int32
		for c := 0; c < channels; c++ {
			o := (i*channels + c) * 2
			sum += int32(int16(uint16(pcm[o]) | uint16(pcm[o+1])<<8))
		}
		v := float32(sum) / float32(channels) / 32768
		sp.ring[sp.pos] = v
		sp.pos = (sp.pos + 1) % spectrumWin
	}
	sp.total += uint64(n)
	sp.mu.Unlock()
}

// Reset 清空缓冲与平滑状态（切换曲目 / 跳转时调用）。
func (sp *Spectrum) Reset() {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	for i := range sp.ring {
		sp.ring[i] = 0
	}
	sp.pos, sp.total = 0, 0
	sp.mu.Unlock()
	for i := range sp.bands {
		sp.bands[i], sp.peaks[i] = 0, 0
	}
	sp.lastT = 0
}

// Bands 返回各频段能量（0~1，已做 attack/release 平滑）。UI 线程每帧调用一次。
func (sp *Spectrum) Bands() []float32 {
	if sp == nil {
		return nil
	}
	sp.mu.Lock()
	total, pos := sp.total, sp.pos
	copy(sp.snap, sp.ring[pos:])
	copy(sp.snap[spectrumWin-pos:], sp.ring[:pos])
	sp.mu.Unlock()

	// 没有新样本（暂停 / 播放结束）：能量自然衰减到 0
	if total == sp.lastT {
		sp.decay()
		return sp.bands
	}
	sp.lastT = total

	// Hann 窗 + FFT
	for i := 0; i < spectrumWin; i++ {
		sp.re[i] = float64(sp.snap[i]) * sp.hann[i]
		sp.im[i] = 0
	}
	fftInPlace(sp.re, sp.im)

	norm := func(lo, hi int) float32 {
		var sum float64
		cnt := 0
		for b := lo; b < hi && b < spectrumWin/2; b++ {
			sum += math.Hypot(sp.re[b], sp.im[b])
			cnt++
		}
		if cnt == 0 {
			return 0
		}
		// 幅度归一化 → dB → 0~1（-70dB 静音，0dB 满刻度）
		mag := sum / float64(cnt) * 2 / float64(spectrumWin)
		db := 20 * math.Log10(mag+1e-7)
		v := (db + 70) / 70
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		return float32(v)
	}

	for i := range sp.bands {
		target := norm(sp.edges[i], sp.edges[i+1])
		cur := sp.bands[i]
		// attack 快（跟得上鼓点）、release 慢（不闪烁）
		if target > cur {
			cur += (target - cur) * 0.62
		} else {
			cur += (target - cur) * 0.16
		}
		if cur < 0.002 {
			cur = 0
		}
		sp.bands[i] = cur
		if cur >= sp.peaks[i] {
			sp.peaks[i] = cur
		}
	}
	sp.fallPeaks()
	return sp.bands
}

// Peaks 返回频段峰值（0~1，缓慢回落），用于绘制顶端峰值点。
func (sp *Spectrum) Peaks() []float32 {
	if sp == nil {
		return nil
	}
	return sp.peaks
}

// decay 无新样本时把能量平滑降到 0。
func (sp *Spectrum) decay() {
	for i := range sp.bands {
		sp.bands[i] *= 0.86
		if sp.bands[i] < 0.002 {
			sp.bands[i] = 0
		}
	}
	sp.fallPeaks()
}

// fallPeaks 峰值缓慢下落（约 0.9/秒）。
func (sp *Spectrum) fallPeaks() {
	for i := range sp.peaks {
		if sp.peaks[i] > sp.bands[i] {
			sp.peaks[i] -= 0.012
			if sp.peaks[i] < sp.bands[i] {
				sp.peaks[i] = sp.bands[i]
			}
			if sp.peaks[i] < 0 {
				sp.peaks[i] = 0
			}
		}
	}
}

// fftInPlace 基 2 迭代 FFT（就地，长度需为 2 的幂）。
func fftInPlace(re, im []float64) {
	n := len(re)
	if n < 2 {
		return
	}
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}
	for length := 2; length <= n; length <<= 1 {
		ang := -2 * math.Pi / float64(length)
		wRe, wIm := math.Cos(ang), math.Sin(ang)
		half := length >> 1
		for i := 0; i < n; i += length {
			cRe, cIm := 1.0, 0.0
			for j := 0; j < half; j++ {
				a, b := i+j, i+j+half
				vRe := re[b]*cRe - im[b]*cIm
				vIm := re[b]*cIm + im[b]*cRe
				re[b], im[b] = re[a]-vRe, im[a]-vIm
				re[a], im[a] = re[a]+vRe, im[a]+vIm
				cRe, cIm = cRe*wRe-cIm*wIm, cRe*wIm+cIm*wRe
			}
		}
	}
}
