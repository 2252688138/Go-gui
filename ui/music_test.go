//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestIsMusicPath 音频扩展名识别（含查询串/大小写）。
func TestIsMusicPath(t *testing.T) {
	ok := []string{"a.mp3", "B.FLAC", "song.m4a?token=1", "x.wav", "y.opus", "z.ape"}
	for _, p := range ok {
		if !IsMusicPath(p) {
			t.Fatalf("IsMusicPath(%q) = false, want true", p)
		}
	}
	no := []string{"", "a.mp4", "a.mkv", "a.txt", "a.mp3.old"}
	for _, p := range no {
		if IsMusicPath(p) {
			t.Fatalf("IsMusicPath(%q) = true, want false", p)
		}
	}
}

// TestSpectrumPicksTone 1kHz 正弦应当点亮对应频段（低频段应接近 0）。
func TestSpectrumPicksTone(t *testing.T) {
	const rate = 48000
	sp := newSpectrum(SpectrumBands, rate)
	tone := 1000.0
	pcm := make([]byte, 0, 8192)
	for i := 0; i < 4096; i++ {
		v := int16(16000 * math.Sin(2*math.Pi*tone*float64(i)/rate))
		pcm = append(pcm, byte(uint16(v)), byte(uint16(v)>>8))
	}
	sp.PushPCM(pcm, 1, rate)
	bands := sp.Bands()
	if len(bands) != SpectrumBands {
		t.Fatalf("频段数 = %d, want %d", len(bands), SpectrumBands)
	}
	// 找能量最高的频段，换算回频率应落在 1kHz 附近
	best, bestV := 0, float32(0)
	for i, v := range bands {
		if v > bestV {
			best, bestV = i, v
		}
	}
	if bestV <= 0.2 {
		t.Fatalf("1kHz 正弦未点亮频段（最大值 %.3f）", bestV)
	}
	ratio := float64(sp.hi) / float64(sp.lo)
	f := float64(sp.lo) * math.Pow(ratio, (float64(best)+0.5)/float64(len(bands)))
	if f < 500 || f > 2000 {
		t.Fatalf("能量最高频段 ≈ %.0fHz, want ≈1kHz", f)
	}
	if bands[0] > 0.2 {
		t.Fatalf("最低频段被误点亮：%.3f", bands[0])
	}
	// 没有新样本时能量应当衰减
	prev := bestV
	for i := 0; i < 40; i++ {
		bands = sp.Bands()
	}
	if bands[best] >= prev {
		t.Fatalf("静音后能量未衰减：%.3f → %.3f", prev, bands[best])
	}
}

// TestSpectrumMirror 镜像频谱：每列上下两段高度之和不超过整高，且能拿到 56 列。
func TestSpectrumMirror(t *testing.T) {
	sp := newSpectrum(SpectrumBands, 48000)
	pcm := make([]byte, 0, 8192)
	for i := 0; i < 4096; i++ {
		v := int16(12000 * math.Sin(2*math.Pi*440*float64(i)/48000))
		pcm = append(pcm, byte(uint16(v)), byte(uint16(v)>>8))
	}
	sp.PushPCM(pcm, 1, 48000)
	sp.Bands()

	root := Column(
		MusicSpectrum("sp", sp, 40, "#f5c518", "#8b5cf6").St(St().Wp(100).H(78)),
		MusicMirrorSpectrum("spm", sp, 56, "#22d3ee", "#818cf8").St(St().Wp(100).H(72)),
	).St(St().W(600).Padding(10))
	LayoutForTest(root, 600, 400, "Microsoft YaHei UI", 1)

	if root.Children[0].Frame.H < 70 {
		t.Fatalf("柱状频谱高度 = %.0f, want ≈78", root.Children[0].Frame.H)
	}
	if root.Children[1].Frame.H < 64 {
		t.Fatalf("镜像频谱高度 = %.0f, want ≈72", root.Children[1].Frame.H)
	}
	bar := root.Children[0].Children[0]
	if bar == nil || bar.Frame.H <= 0 {
		t.Fatalf("频谱柱高度非法: %+v", bar)
	}
	if len(root.Children[0].Children) != 40 {
		t.Fatalf("柱数 = %d, want 40", len(root.Children[0].Children))
	}
	mcol := root.Children[1].Children[0]
	if mcol == nil || mcol.Frame.H <= 0 {
		t.Fatalf("镜像列高度非法: %+v", mcol)
	}
	up, down := mcol.Children[0], mcol.Children[1]
	if up.Frame.Y+mcol.Frame.Y < mcol.Frame.Y-1 {
		t.Fatalf("上半段越出容器顶部: %+v / %+v", up.Frame, mcol.Frame)
	}
	if down.Frame.Y+down.Frame.H > mcol.Frame.Y+mcol.Frame.H+1 {
		t.Fatalf("下半段越出容器底部: %+v / %+v", down.Frame, mcol.Frame)
	}
}

// makeTestAudio 用 ffmpeg 生成 2 秒 440Hz WAV（无 ffmpeg 时跳过）。
func makeTestAudio(t *testing.T) string {
	t.Helper()
	ff := toolPath("", "ffmpeg")
	if ff == "" {
		t.Skip("未安装 ffmpeg，跳过音频集成测试")
	}
	path := filepath.Join(t.TempDir(), "tone.wav")
	cmd := exec.Command(ff, "-v", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2", path)
	hideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("生成测试音频失败（%v）: %s", err, out)
	}
	return path
}

// TestMusicSourceDecode 端到端：打开 WAV → 播放推进 → 频谱有能量 → 跳转 → 关闭。
func TestMusicSourceDecode(t *testing.T) {
	path := makeTestAudio(t)
	src, err := OpenMusic(path)
	if err != nil {
		t.Fatalf("OpenMusic: %v", err)
	}
	defer src.Close()

	if d := src.Duration(); d < 1900*time.Millisecond || d > 2100*time.Millisecond {
		t.Fatalf("时长 = %v, want ~2s", d)
	}
	if info := src.Info(); !info.HasAudio || info.SampleRate <= 0 {
		t.Fatalf("探测信息异常: %+v", info)
	}
	src.Play()
	if !src.Playing() {
		t.Fatal("Play 后 Playing() = false")
	}

	// 等到有实际播放进度，同时频谱应当点亮
	deadline := time.Now().Add(3 * time.Second)
	var bands []float32
	for time.Now().Before(deadline) {
		time.Sleep(60 * time.Millisecond)
		src.Pos()
		bands = src.Spectrum().Bands()
		if src.Pos() > 200*time.Millisecond && maxBand(bands) > 0.2 {
			break
		}
	}
	if p := src.Pos(); p <= 0 {
		t.Fatalf("播放位置未推进: %v", p)
	}
	if maxBand(bands) <= 0.2 {
		t.Fatalf("频谱未点亮（max=%.3f）", maxBand(bands))
	}

	// 跳转
	src.Seek(1500 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)
	if p := src.Pos(); p < 1400*time.Millisecond || p > 2100*time.Millisecond {
		t.Fatalf("Seek 后位置 = %v, want ≈1.5s", p)
	}
	if src.Progress() <= 0 {
		t.Fatal("Progress 应大于 0")
	}

	// 音量 / 倍速 / 循环 / 暂停
	src.SetVolume(0.5)
	if v := src.Volume(); v < 0.49 || v > 0.51 {
		t.Fatalf("Volume = %.2f, want 0.5", v)
	}
	src.ToggleMute()
	if !src.Muted() {
		t.Fatal("ToggleMute 后应为静音")
	}
	src.ToggleMute()
	src.SetLoop(true)
	if !src.Loop() {
		t.Fatal("SetLoop(true) 未生效")
	}
	src.SetSpeed(1.5)
	if v := src.Speed(); v < 1.49 || v > 1.51 {
		t.Fatalf("Speed = %.2f, want 1.5", v)
	}
	src.Pause()
	if src.Playing() {
		t.Fatal("Pause 后仍在播放")
	}
}

// maxBand 频段最大值。
func maxBand(b []float32) float32 {
	m := float32(0)
	for _, v := range b {
		if v > m {
			m = v
		}
	}
	return m
}

// TestMusicPlayerLayout 组件布局：封面 / 进度条 / 控制行都在卡片内且不溢出。
func TestMusicPlayerLayout(t *testing.T) {
	sp := newSpectrum(SpectrumBands, 48000)
	src := &MusicSource{info: MediaInfo{Format: "wav", AudioCodec: "pcm_s16le", SampleRate: 48000,
		Duration: 3 * time.Second, HasAudio: true}, rate: 48000, ch: 2, speed: 1, vol: 1, an: sp, closed: true}
	root := Column(
		MusicPlayer("mp", src, MusicPlayerOptions{Track: MusicTrack{Title: "曲名", Artist: "歌手"}, Bars: 40}),
	).St(St().W(900).Padding(20))
	LayoutForTest(root, 900, 600, "Microsoft YaHei UI", 1)

	card := root.Children[0]
	if card == nil || card.Frame.W < 800 || card.Frame.H < 120 {
		t.Fatalf("卡片尺寸异常: %+v", card)
	}
	for _, key := range []string{"mp-cover", "mp-seek", "mp-play", "mp-prev", "mp-next", "mp-spec"} {
		n := findNodeKey(card, key)
		if n == nil {
			t.Fatalf("找不到节点 %s", key)
		}
		if n.Frame.W <= 0 || n.Frame.H <= 0 {
			t.Fatalf("%s 尺寸非法: %+v", key, n.Frame)
		}
		if n.Frame.Bottom() > card.Frame.Bottom()+1 {
			t.Fatalf("%s 溢出卡片底部: %+v / %+v", key, n.Frame, card.Frame)
		}
	}
	if spec := findNodeKey(card, "mp-spec"); spec.Frame.H < 60 {
		t.Fatalf("频谱高度 = %.0f, want ≈64", spec.Frame.H)
	}
}

// findNodeKey 在节点树里按键查找节点（测试辅助）。
func findNodeKey(n *Node, key string) *Node {
	if n == nil {
		return nil
	}
	if n.Key == key {
		return n
	}
	for _, c := range n.Children {
		if r := findNodeKey(c, key); r != nil {
			return r
		}
	}
	return nil
}

// writeTestWAV 写一个 1 秒 440Hz 单声道 16bit WAV（验证 WAV 头与探测链路）。
func writeTestWAV(path string) error {
	const rate = 44100
	n := rate
	data := make([]byte, 44+n*2)
	putU32 := func(off int, v uint32) {
		data[off], data[off+1], data[off+2], data[off+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	putU16 := func(off int, v uint16) {
		data[off], data[off+1] = byte(v), byte(v>>8)
	}
	copy(data[0:4], "RIFF")
	putU32(4, uint32(len(data)-8))
	copy(data[8:12], "WAVE")
	copy(data[12:16], "fmt ")
	putU32(16, 16)
	putU16(20, 1)
	putU16(22, 1)
	putU32(24, rate)
	putU32(28, rate*2)
	putU16(32, 2)
	putU16(34, 16)
	copy(data[36:40], "data")
	putU32(40, uint32(n*2))
	for i := 0; i < n; i++ {
		v := int16(15000 * math.Sin(2*math.Pi*440*float64(i)/rate))
		putU16(44+i*2, uint16(v))
	}
	return os.WriteFile(path, data, 0o644)
}

// TestWAVProbe画廊内置 WAV 生成器产出的文件可被 ffprobe 解析。
func TestWAVProbe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.wav")
	if err := writeTestWAV(path); err != nil {
		t.Fatalf("生成 WAV: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() < 44100 {
		t.Fatalf("WAV 文件异常: %v size=%d", err, st.Size())
	}
	if toolPath("", "ffprobe") == "" {
		return
	}
	info, err := ProbeMedia(path)
	if err != nil {
		t.Fatalf("ffprobe 解析失败: %v", err)
	}
	if !info.HasAudio || info.AudioCodec == "" {
		t.Fatalf("探测信息异常: %+v", info)
	}
}
