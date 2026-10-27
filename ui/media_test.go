//go:build windows

package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIsMediaPath 扩展名识别（含查询串与大小写）。
func TestIsMediaPath(t *testing.T) {
	paths := []string{
		"a.mp4", "A.MKV", `D:\videos\电影.mov`, "x.avi", "x.webm", "x.flv",
		"x.ts", "x.m2ts", "x.m3u8",
		"https://cdn.example.com/a/b.m3u8?token=1",
		"https://cdn.example.com/a/b.mp4#t=10",
	}
	for _, p := range paths {
		if !IsMediaPath(p) {
			t.Fatalf("IsMediaPath(%q) = false, want true", p)
		}
	}
	notMedia := []string{"a.txt", "a.mp3", "", "noext", "a.mp4.txt"}
	for _, p := range notMedia {
		if IsMediaPath(p) {
			t.Fatalf("IsMediaPath(%q) = true, want false", p)
		}
	}
}

// TestParseProbeJSON 解析 ffprobe 输出（含旋转元数据与音频流）。
func TestParseProbeJSON(t *testing.T) {
	raw := []byte(`{
	  "streams": [
	    {"codec_type":"video","codec_name":"h264","width":1920,"height":1080,
	     "avg_frame_rate":"30000/1001","r_frame_rate":"30000/1001",
	     "disposition":{"attached_pic":0},
	     "side_data_list":[{"side_data_type":"Display Matrix","rotation":-90}]},
	    {"codec_type":"audio","codec_name":"aac","sample_rate":"44100","channels":2}
	  ],
	  "format": {"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"62.500000","bit_rate":"1200000"}
	}`)
	info, err := parseProbeJSON("a.mp4", raw)
	if err != nil {
		t.Fatalf("parseProbeJSON: %v", err)
	}
	if !info.HasVideo || !info.HasAudio {
		t.Fatalf("streams 识别错误: %+v", info)
	}
	if info.VideoCodec != "h264" || info.AudioCodec != "aac" {
		t.Fatalf("编码 = %s/%s", info.VideoCodec, info.AudioCodec)
	}
	// 旋转 90°：显示尺寸宽高互换
	if info.Width != 1080 || info.Height != 1920 {
		t.Fatalf("旋转后尺寸 = %dx%d, want 1080x1920", info.Width, info.Height)
	}
	if info.Duration != 62500*time.Millisecond {
		t.Fatalf("时长 = %v, want 62.5s", info.Duration)
	}
	if info.SampleRate != 44100 || info.Channels != 2 {
		t.Fatalf("音频参数 = %d/%d", info.SampleRate, info.Channels)
	}
	if f := info.FPS; f < 29.9 || f > 30.0 {
		t.Fatalf("帧率 = %v, want 29.97", f)
	}
}

// TestParseProbeJSONNoStream 没有音视频流 / 非法 JSON 时报错。
func TestParseProbeJSONNoStream(t *testing.T) {
	if _, err := parseProbeJSON("x", []byte(`{"streams":[],"format":{}}`)); err == nil {
		t.Fatal("空流应当报错")
	}
	if _, err := parseProbeJSON("x", []byte("not json")); err == nil {
		t.Fatal("非法 JSON 应当报错")
	}
}

// TestParseFrameRate 帧率文本解析。
func TestParseFrameRate(t *testing.T) {
	rates := map[string]float64{
		"25":         25,
		"30000/1001": 29.97002997002997,
		"0/0":        0,
		"":           0,
		"abc":        0,
		"1/0":        0,
	}
	for in, want := range rates {
		got := parseFrameRate(in)
		if want == 0 {
			if got != 0 {
				t.Fatalf("parseFrameRate(%q) = %v, want 0", in, got)
			}
			continue
		}
		if got < want-0.001 || got > want+0.001 {
			t.Fatalf("parseFrameRate(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestAtempoChain 倍速拆分为 atempo 组合。
func TestAtempoChain(t *testing.T) {
	type tc struct {
		v    float64
		want string
	}
	cases := []tc{
		{0.25, "atempo=0.5,atempo=0.500"},
		{0.5, "atempo=0.500"},
		{1, "atempo=1.000"},
		{1.5, "atempo=1.500"},
		{4, "atempo=2,atempo=2.000"},
	}
	for _, c := range cases {
		got := strings.Join(atempoChain(c.v), ",")
		if got != c.want {
			t.Fatalf("atempoChain(%v) = %q, want %q", c.v, got, c.want)
		}
	}
}

// TestMediaSourceErr 空路径 / 非媒体文件应当报错。
func TestMediaSourceErr(t *testing.T) {
	if _, err := OpenMedia(""); err == nil {
		t.Fatal("空路径应当报错")
	}
	dir := t.TempDir()
	txt := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(txt, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenMedia(txt); err == nil {
		t.Fatal("非媒体文件应当报错")
	}
}

// makeTestVideo 用 ffmpeg 生成 2 秒带声测试视频（无 ffmpeg 时跳过）。
func makeTestVideo(t *testing.T) string {
	t.Helper()
	ff := toolPath("", "ffmpeg")
	if ff == "" {
		t.Skip("未安装 ffmpeg，跳过解码集成测试")
	}
	path := filepath.Join(t.TempDir(), "clip.mp4")
	cmd := exec.Command(ff, "-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x180:rate=25:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", path)
	hideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("生成测试视频失败（%v）: %s", err, out)
	}
	return path
}

// TestMediaSourceDecode 端到端：ffmpeg 生成 2 秒测试视频，再解码按位置取帧。
func TestMediaSourceDecode(t *testing.T) {
	path := makeTestVideo(t)
	src, err := OpenMediaWith(path, MediaConfig{NoAudio: true})
	if err != nil {
		t.Fatalf("OpenMedia: %v", err)
	}
	defer src.Close()

	if info := src.Info(); !info.HasVideo || info.Width != 320 || info.Height != 180 {
		t.Fatalf("探测信息错误: %+v", info)
	}
	if d := src.Duration(); d < 1900*time.Millisecond || d > 2100*time.Millisecond {
		t.Fatalf("时长 = %v, want ~2s", d)
	}
	// 默认跟随源尺寸，不缩放
	if w, h := src.Size(); w != 320 || h != 180 {
		t.Fatalf("输出尺寸 = %dx%d, want 320x180", w, h)
	}

	var last []byte
	for i := 0; i < 20; i++ {
		pos := time.Duration(i) * 100 * time.Millisecond
		f, err := src.Frame(pos)
		if err != nil {
			t.Fatalf("Frame(%v): %v", pos, err)
		}
		if f == nil || !f.Valid() {
			t.Fatalf("Frame(%v) 无画面", pos)
		}
		if f.W != 320 || f.H != 180 {
			t.Fatalf("帧尺寸 = %dx%d", f.W, f.H)
		}
		if i >= 2 && last != nil {
			same := len(f.Pix) == len(last)
			if same {
				for j := range f.Pix {
					if f.Pix[j] != last[j] {
						same = false
						break
					}
				}
			}
			if same {
				t.Fatalf("Frame(%v) 与前帧完全相同，解码没有推进", pos)
			}
		}
		last = f.Pix
	}

	// 跳转（超出已缓冲范围）后仍能取到画面
	if f, err := src.Frame(1500 * time.Millisecond); err != nil || f == nil || !f.Valid() {
		t.Fatalf("跳转取帧失败: %v", err)
	}
}

// TestMediaSourceHLS 真实 HLS 直链播放（需网络）。运行方式：
// 设置环境变量 GOGUI_HLS_TEST=<m3u8 地址> 后 go test ./ui/ -run TestMediaSourceHLS -v
func TestMediaSourceHLS(t *testing.T) {
	u := strings.TrimSpace(os.Getenv("GOGUI_HLS_TEST"))
	if u == "" {
		t.Skip("未设置 GOGUI_HLS_TEST，跳过 HLS 联播测试")
	}
	src, err := OpenMediaWith(u, MediaConfig{NoAudio: true})
	if err != nil {
		t.Fatalf("OpenMediaWith(%s): %v", u, err)
	}
	defer src.Close()
	if info := src.Info(); !info.HasVideo || info.Width <= 0 {
		t.Fatalf("探测信息异常: %+v", info)
	}
	// 首帧可能要等建连 / 拉分片：轮询重试（不假定单次调用就能拿到）
	deadline := time.Now().Add(20 * time.Second)
	for i := 0; i < 30; i++ {
		pos := time.Duration(i) * 100 * time.Millisecond
		var f *VideoFrame
		for {
			var err error
			f, err = src.Frame(pos)
			if err != nil {
				t.Fatalf("Frame(%v): %v", pos, err)
			}
			if f != nil && f.Valid() {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("Frame(%v) 超时仍未取到画面", pos)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// TestMediaPlayerPlayback 播放器推进：位置前进、画面可用、能暂停/变速/调音量。
func TestMediaPlayerPlayback(t *testing.T) {
	path := makeTestVideo(t)
	p, err := NewMediaPlayerWith(path, MediaConfig{})
	if err != nil {
		t.Fatalf("NewMediaPlayerWith: %v", err)
	}
	defer p.Close()
	if p.Duration() <= 0 {
		t.Fatal("时长未知")
	}
	if info := p.Info(); !info.HasAudio {
		t.Fatalf("探测信息缺音频: %+v", info)
	}

	p.Play()
	now := float32(0)
	deadline := time.Now().Add(700 * time.Millisecond)
	var first *VideoFrame
	for time.Now().Before(deadline) {
		now += 0.02
		p.advance(now)
		if f := p.Frame(); f != nil && f.Valid() && first == nil {
			first = f
		}
		time.Sleep(20 * time.Millisecond)
	}
	if first == nil {
		t.Fatal("没有取到画面")
	}
	if pos := p.Pos(); pos < 300*time.Millisecond {
		t.Fatalf("播放位置未推进: %v", pos)
	}

	p.SetSpeed(1.5)
	p.SetVolume(0.4)
	p.ToggleMute()
	p.Pause()
	if p.Playing {
		t.Fatal("暂停后仍在播放")
	}
}

// TestPlayerRotationAndLock 旋转归一化与锁定开关。
func TestPlayerRotationAndLock(t *testing.T) {
	p := NewPlayer(NewVideoClip([]VideoFrame{testFrame(2, 2, 10, 20, 30, 255, 0)}, 24, false))
	if p.RotationDeg() != 0 {
		t.Fatalf("初始角度 = %d, want 0", p.RotationDeg())
	}
	p.RotateBy(1)
	if p.RotationDeg() != 90 {
		t.Fatalf("旋转 90° 后 = %d", p.RotationDeg())
	}
	p.RotateBy(3)
	if p.RotationDeg() != 0 {
		t.Fatalf("再转 270° 应回到 0，got %d", p.RotationDeg())
	}
	p.SetRotation(-1)
	if p.RotationDeg() != 270 {
		t.Fatalf("SetRotation(-1) = %d, want 270", p.RotationDeg())
	}
	p.SetRotation(2)
	if p.RotationDeg() != 180 {
		t.Fatalf("SetRotation(2) = %d, want 180", p.RotationDeg())
	}
	if p.Locked {
		t.Fatal("默认不应锁定")
	}
	p.ToggleLock()
	if !p.Locked {
		t.Fatal("ToggleLock 应进入锁定")
	}
	p.SetLocked(false)
	if p.Locked {
		t.Fatal("SetLocked(false) 失败")
	}
}

// TestMediaScrubFast 拖动中取帧不阻塞，松手后定位到目标。
func TestMediaScrubFast(t *testing.T) {
	path := makeTestVideo(t)
	src, err := OpenMediaWith(path, MediaConfig{NoAudio: true})
	if err != nil {
		t.Fatalf("OpenMediaWith: %v", err)
	}
	defer src.Close()
	if _, err := src.Frame(0); err != nil {
		t.Fatalf("首帧: %v", err)
	}

	src.MediaScrub(true)
	start := time.Now()
	for i := 0; i < 10; i++ {
		if _, err := src.Frame(time.Duration(i) * 150 * time.Millisecond); err != nil {
			t.Fatalf("拖动取帧: %v", err)
		}
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("拖动中 10 次取帧耗时 %v，应快速返回（不重启解码）", d)
	}

	src.MediaScrub(false)
	deadline := time.Now().Add(3 * time.Second)
	for {
		f, ferr := src.Frame(1500 * time.Millisecond)
		if ferr == nil && f != nil && f.Valid() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("松手后定位超时: %v", ferr)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
