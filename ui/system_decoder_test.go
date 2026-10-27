//go:build windows

package ui

import (
	"os"
	"testing"
	"time"

	"gogui/internal/win32"
)

// withDecoder 临时设置解码器偏好（GOGUI_DECODER），返回清理函数。
func withDecoder(t *testing.T, v string) {
	t.Helper()
	old, had := os.LookupEnv("GOGUI_DECODER")
	if err := os.Setenv("GOGUI_DECODER", v); err != nil {
		t.Fatalf("Setenv: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("GOGUI_DECODER", old)
		} else {
			_ = os.Unsetenv("GOGUI_DECODER")
		}
	})
}

// TestDecoderPreference GOGUI_DECODER 环境变量解析。
func TestDecoderPreference(t *testing.T) {
	cases := []struct{ env, want string }{
		{"", "auto"},
		{"system", "system"},
		{"MF", "system"},
		{"ffmpeg", "ffmpeg"},
		{"whatever", "auto"},
	}
	for _, c := range cases {
		withDecoder(t, c.env)
		if got := DecoderPreference(); got != c.want {
			t.Fatalf("DecoderPreference(%q) = %q, want %q", c.env, got, c.want)
		}
	}
}

// TestAutoBackend 自动模式：有 ffmpeg 时仍优先 ffmpeg（保证回退不改变默认行为）。
func TestAutoBackend(t *testing.T) {
	if toolPath("", "ffmpeg") == "" || toolPath("", "ffprobe") == "" {
		t.Skip("未安装 ffmpeg，跳过自动模式测试")
	}
	withDecoder(t, "")
	if got := BackendName(); got != "ffmpeg" {
		t.Fatalf("BackendName() = %q, want ffmpeg", got)
	}
	path := makeTestVideo(t)
	src, err := OpenMediaWith(path, MediaConfig{NoAudio: true})
	if err != nil {
		t.Fatalf("OpenMediaWith: %v", err)
	}
	defer src.Close()
	if got := src.Info().Backend; got != "ffmpeg" {
		t.Fatalf("自动模式 Backend = %q, want ffmpeg", got)
	}
}

// TestSystemBackend 系统解码器回退（Media Foundation）端到端：
// 视频取帧 + 转 RGBA + 跳转；音频 PCM + 频谱 + 跳转。
//
// 注意：Media Foundation 的 SourceReader 在同一进程反复创建/销毁存在平台级竞态
// （0xc0000005），因此这里在一个测试内顺序验完，共用一次 MF 生命周期。
// 业务代码中一次只用一个 reader，不受影响。
func TestSystemBackend(t *testing.T) {
	if !win32.MFAvailable() {
		t.Skip("系统解码器（Media Foundation）不可用，跳过回退测试")
	}
	videoPath := makeTestVideo(t) // 仅用于生成测试素材（H.264 + AAC）
	audioPath := makeTestAudio(t)
	withDecoder(t, "system")

	// ---- 视频 ----
	src, err := OpenMediaWith(videoPath, MediaConfig{NoAudio: true})
	if err != nil {
		t.Fatalf("系统解码器打开视频失败: %v", err)
	}
	info := src.Info()
	if info.Backend != "system" {
		t.Fatalf("Backend = %q, want system", info.Backend)
	}
	if !info.HasVideo || info.Width <= 0 || info.Height <= 0 {
		t.Fatalf("探测信息异常: %+v", info)
	}

	var frame *VideoFrame
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		frame, err = src.Frame(0)
		if err == nil && frame != nil && frame.Valid() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("系统解码取帧失败: %v", err)
	}
	if frame == nil || !frame.Valid() {
		t.Fatal("系统解码未取到画面")
	}
	if frame.W != info.Width || frame.H != info.Height {
		t.Fatalf("帧尺寸 = %dx%d, want %dx%d", frame.W, frame.H, info.Width, info.Height)
	}
	// 像素不能全黑（测试素材是彩色 testsrc）
	nonZero := 0
	for i := 0; i < len(frame.Pix); i += 997 {
		if frame.Pix[i] != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatal("系统解码画面全黑")
	}
	// 跳转后仍能取到画面
	var f2 *VideoFrame
	deadline = time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		f2, err = src.Frame(1200 * time.Millisecond)
		if err == nil && f2 != nil && f2.Valid() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil || f2 == nil || !f2.Valid() {
		t.Fatalf("系统解码跳转取帧失败: %v", err)
	}
	src.Close()
	time.Sleep(300 * time.Millisecond)

	// ---- 音频 ----
	m, err := OpenMusic(audioPath)
	if err != nil {
		t.Fatalf("系统解码器打开音频失败: %v", err)
	}
	defer m.Close()
	if got := m.Info().Backend; got != "system" {
		t.Fatalf("音频 Backend = %q, want system", got)
	}
	if ai := m.Info(); !ai.HasAudio || ai.SampleRate <= 0 {
		t.Fatalf("音频探测信息异常: %+v", ai)
	}
	m.Play()
	deadline = time.Now().Add(3 * time.Second)
	var bands []float32
	for time.Now().Before(deadline) {
		time.Sleep(60 * time.Millisecond)
		m.Pos()
		bands = m.Spectrum().Bands()
		if m.Pos() > 100*time.Millisecond && maxBand(bands) > 0.2 {
			break
		}
	}
	if p := m.Pos(); p <= 0 {
		t.Fatalf("系统解码播放位置未推进: %v", p)
	}
	if maxBand(bands) <= 0.2 {
		t.Fatalf("系统解码频谱未点亮（max=%.3f）", maxBand(bands))
	}
	m.Seek(800 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)
	if p := m.Pos(); p < 700*time.Millisecond {
		t.Fatalf("系统解码 Seek 后位置 = %v, want ≈0.8s", p)
	}
}
