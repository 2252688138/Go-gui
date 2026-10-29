// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
	"time"

	"gogui/internal/geom"
)

// nearDur 时长比较（浮点推进存在亚毫秒误差，容忍 2ms）。
func nearDur(a, b time.Duration) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= 2*time.Millisecond
}

// testFrame 生成 w×h 的纯色帧。
func testFrame(w, h int, r, g, b, a uint8, delay time.Duration) VideoFrame {
	pix := make([]byte, w*h*4)
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+1], pix[i+2], pix[i+3] = r, g, b, a
	}
	return VideoFrame{Pix: pix, W: w, H: h, Delay: delay}
}

// TestVideoClipTimeline 时间轴：帧起点、总时长与位置到帧号的映射。
func TestVideoClipTimeline(t *testing.T) {
	clip := NewVideoClip([]VideoFrame{
		testFrame(4, 4, 255, 0, 0, 255, 100*time.Millisecond),
		testFrame(4, 4, 0, 255, 0, 255, 100*time.Millisecond),
		testFrame(4, 4, 0, 0, 255, 255, 100*time.Millisecond),
	}, 24, false)

	if d := clip.Duration(); d != 300*time.Millisecond {
		t.Fatalf("Duration = %v, want 300ms", d)
	}
	if w, h := clip.Size(); w != 4 || h != 4 {
		t.Fatalf("Size = %dx%d, want 4x4", w, h)
	}
	if got := clip.FrameStart(1); got != 100*time.Millisecond {
		t.Fatalf("FrameStart(1) = %v, want 100ms", got)
	}

	type probe struct {
		pos  time.Duration
		want int
	}
	probes := []probe{
		{0, 0},
		{99 * time.Millisecond, 0},
		{100 * time.Millisecond, 1},
		{299 * time.Millisecond, 2},
		{300 * time.Millisecond, 2},
		{-1, 0},
	}
	for _, c := range probes {
		if got := clip.FrameIndexAt(c.pos); got != c.want {
			t.Fatalf("FrameIndexAt(%v) = %d, want %d", c.pos, got, c.want)
		}
	}

	var nilClip *VideoClip
	if got := nilClip.FrameIndexAt(0); got != -1 {
		t.Fatalf("nil clip FrameIndexAt = %d, want -1", got)
	}
}

// TestVideoClipDefaultFPS Delay 全为 0 时按 FPS 折算帧时长。
func TestVideoClipDefaultFPS(t *testing.T) {
	clip := NewVideoClip([]VideoFrame{
		testFrame(2, 2, 1, 2, 3, 255, 0),
		testFrame(2, 2, 1, 2, 3, 255, 0),
	}, 10, false)
	if got := clip.FrameDelay(0); got != 100*time.Millisecond {
		t.Fatalf("FrameDelay = %v, want 100ms", got)
	}
	if got := clip.Duration(); got != 200*time.Millisecond {
		t.Fatalf("Duration = %v, want 200ms", got)
	}
}

// TestPlayerAdvance 播放推进与循环/结束行为。
func TestPlayerAdvance(t *testing.T) {
	clip := NewVideoClip([]VideoFrame{
		testFrame(2, 2, 255, 0, 0, 255, 100*time.Millisecond),
		testFrame(2, 2, 0, 255, 0, 255, 100*time.Millisecond),
		testFrame(2, 2, 0, 0, 255, 255, 100*time.Millisecond),
	}, 24, true)
	p := NewPlayer(clip)
	p.Play()

	p.advance(0)    // 首帧只记录时钟
	p.advance(0.05) // +50ms
	if !nearDur(p.Pos(), 50*time.Millisecond) {
		t.Fatalf("Pos = %v, want 50ms", p.Pos())
	}
	if idx := p.FrameIndex(); idx != 0 {
		t.Fatalf("FrameIndex = %d, want 0", idx)
	}
	if f := p.Frame(); f == nil || f.Pix[0] != 255 {
		t.Fatal("Frame() 应为第一帧")
	}

	// 继续推进到末帧
	p.advance(0.2) // dt=150ms，累计 200ms
	if !nearDur(p.Pos(), 200*time.Millisecond) {
		t.Fatalf("Pos = %v, want 200ms", p.Pos())
	}
	if idx := p.FrameIndex(); idx != 2 {
		t.Fatalf("FrameIndex = %d, want 2", idx)
	}

	// 跨过总时长后循环回绕
	p.advance(0.3) // dt=100ms，累计 300ms → 回绕到 0
	if !nearDur(p.Pos(), 0) {
		t.Fatalf("loop Pos = %v, want 0", p.Pos())
	}

	// 关闭循环后播放到末尾应自动暂停
	p.advance(0.5) // dt=200ms → 200ms
	p.SetLoop(false)
	clip.Loop = false
	p.advance(0.6) // dt=100ms → 300ms = 时长
	if !nearDur(p.Pos(), clip.Duration()) {
		t.Fatalf("end Pos = %v, want %v", p.Pos(), clip.Duration())
	}
	if p.PlayingNow() {
		t.Fatal("播放到末尾后应自动暂停")
	}
}

// TestPlayerSeek 跳转、比例跳转、音量与速率夹取。
func TestPlayerSeek(t *testing.T) {
	p := NewPlayer(NewVideoClip([]VideoFrame{
		testFrame(2, 2, 0, 0, 0, 255, time.Second),
		testFrame(2, 2, 0, 0, 0, 255, time.Second),
	}, 24, true))

	p.SeekRatio(0.5)
	if got := p.Pos(); got != time.Second {
		t.Fatalf("SeekRatio(0.5) Pos = %v, want 1s", got)
	}
	p.SeekBy(-10 * time.Second)
	if got := p.Pos(); got != 0 {
		t.Fatalf("SeekBy 负向夹取 Pos = %v, want 0", got)
	}
	p.SeekBy(99 * time.Second)
	if got := p.Pos(); got != 2*time.Second {
		t.Fatalf("SeekBy 正向夹取 Pos = %v, want 2s", got)
	}

	p.SetVolume(2)
	if p.Volume != 1 || p.Muted {
		t.Fatalf("SetVolume(2) = %v muted=%v, want 1/false", p.Volume, p.Muted)
	}
	p.SetVolume(-1)
	if p.Volume != 0 || !p.Muted {
		t.Fatalf("SetVolume(-1) = %v muted=%v, want 0/true", p.Volume, p.Muted)
	}
	p.ToggleMute()
	if p.Muted {
		t.Fatal("ToggleMute 后应为非静音")
	}

	p.SetSpeed(10)
	if p.SpeedValue() != 4 {
		t.Fatalf("SetSpeed(10) = %v, want 4", p.SpeedValue())
	}
}

// TestGIFToClip GIF 多帧解码：帧数、尺寸、延迟与合成结果。
func TestGIFToClip(t *testing.T) {
	pal := color.Palette{color.RGBA{}, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}}
	frame := func(idx uint8) *image.Paletted {
		img := image.NewPaletted(image.Rect(0, 0, 4, 4), pal)
		for i := range img.Pix {
			img.Pix[i] = idx
		}
		return img
	}
	g := &gif.GIF{
		Image:  []*image.Paletted{frame(1), frame(2)},
		Delay:  []int{5, 7},
		Config: image.Config{ColorModel: pal, Width: 4, Height: 4},
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatalf("EncodeAll: %v", err)
	}
	decoded, err := gif.DecodeAll(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	clip, err := GIFToClip(decoded)
	if err != nil {
		t.Fatalf("GIFToClip: %v", err)
	}
	if clip.FrameCount() != 2 {
		t.Fatalf("FrameCount = %d, want 2", clip.FrameCount())
	}
	if w, h := clip.Size(); w != 4 || h != 4 {
		t.Fatalf("Size = %dx%d, want 4x4", w, h)
	}
	if got := clip.Duration(); got != 120*time.Millisecond {
		t.Fatalf("Duration = %v, want 120ms", got)
	}
	second := clip.Frames[1]
	if len(second.Pix) != 4*4*4 {
		t.Fatalf("frame pix len = %d, want 64", len(second.Pix))
	}
	if second.Pix[1] != 255 {
		t.Fatalf("第二帧绿色通道 = %d, want 255", second.Pix[1])
	}
	if _, err := GIFToClip(nil); err == nil {
		t.Fatal("空 GIF 应返回错误")
	}
}

// TestFitVideoRect 画面填充：Contain 留边、Cover 溢出、Fill 拉伸。
func TestFitVideoRect(t *testing.T) {
	r := geom.Rect{X: 0, Y: 0, W: 200, H: 100}

	contain := fitVideoRect(r, 100, 100, FitContain)
	if contain.W != 100 || contain.H != 100 || contain.X != 50 || contain.Y != 0 {
		t.Fatalf("contain = %+v", contain)
	}
	cover := fitVideoRect(r, 100, 100, FitCover)
	if cover.W != 200 || cover.H != 200 || cover.Y != -50 {
		t.Fatalf("cover = %+v", cover)
	}
	if fill := fitVideoRect(r, 100, 100, FitFill); fill != r {
		t.Fatalf("fill = %+v, want %+v", fill, r)
	}
}

// TestVideoPlayerLayout 播放器组件参与布局：默认 100% 宽 × 360 高，画面铺满。
func TestVideoPlayerLayout(t *testing.T) {
	p := NewPlayer(NewVideoClip([]VideoFrame{
		testFrame(16, 9, 10, 20, 30, 255, 40*time.Millisecond),
	}, 24, true))
	root := Column(VideoPlayer("demo", p)).St(St().Padding(10))
	Layout(root, 640, 480, newTestCache())

	surface := findKey(root, "demo-surface")
	if surface == nil {
		t.Fatal("未找到画面节点 demo-surface")
	}
	if !near(surface.Frame.W, 620) || !near(surface.Frame.H, 360) {
		t.Fatalf("surface = %.1fx%.1f, want 620x360", surface.Frame.W, surface.Frame.H)
	}
	if surface.Kind != KVideo || surface.Video != p {
		t.Fatal("画面节点未绑定播放器")
	}
	if findKey(root, "demo-play") == nil {
		t.Fatal("控制条缺少播放按钮")
	}
	if findKey(root, "demo-seek") == nil {
		t.Fatal("控制条缺少进度条")
	}
}

// TestPlayerLoopToggle 关掉循环后不再回绕（Clip.Loop 不得覆盖 Player 开关）。
func TestPlayerLoopToggle(t *testing.T) {
	clip := NewVideoClip([]VideoFrame{
		testFrame(2, 2, 255, 0, 0, 255, 50*time.Millisecond),
		testFrame(2, 2, 0, 255, 0, 255, 50*time.Millisecond),
	}, 24, true)
	p := NewPlayer(clip)
	if !p.Loop {
		t.Fatal("应继承片段的循环设置")
	}
	p.Play()
	p.SetLoop(false)
	p.advance(0)
	p.advance(0.08) // 80ms < 100ms
	p.advance(0.16) // 累计 160ms > 时长，未开循环应停在末尾
	if p.PlayingNow() {
		t.Fatal("关闭循环后播放到末尾应暂停")
	}
	if !nearDur(p.Pos(), clip.Duration()) {
		t.Fatalf("Pos = %v, want %v", p.Pos(), clip.Duration())
	}
}

// TestVideoVolPanelPos 音量浮层贴图标上方并水平居中。
func TestVideoVolPanelPos(t *testing.T) {
	icon := geom.Rect{X: 404, Y: 472, W: 28, H: 28}
	root := geom.Rect{X: 258, Y: 133, W: 880, H: 380}
	x, y := videoVolPanelPos(icon, root, 34, 112)
	if !near(x, 143) { // 404-258+14-17
		t.Fatalf("popup x = %.1f, want 143", x)
	}
	if !near(y, 221) { // 472-133-112-6
		t.Fatalf("popup y = %.1f, want 221", y)
	}
	// 面板底边应落在图标顶边上方 6px 处
	if !near(y+112, icon.Y-root.Y-6) {
		t.Fatalf("popup bottom = %.1f, want %.1f", y+112, icon.Y-root.Y-6)
	}
}

// TestVideoPlayerFullscreen 全屏时播放器铺满满窗口（不改系统窗口）。
func TestVideoPlayerFullscreen(t *testing.T) {
	p := NewPlayer(NewVideoClip([]VideoFrame{
		testFrame(16, 9, 10, 20, 30, 255, 40*time.Millisecond),
	}, 24, true))
	p.Fullscreen = true
	root := Stack(VideoPlayer("fs", p)).St(St().Wp(100).Hp(100))
	Layout(root, 640, 480, newTestCache())

	n := findKey(root, "fs-root")
	if n == nil {
		t.Fatal("未找到播放器根节点 fs-root")
	}
	if !near(n.Frame.W, root.Frame.W) || !near(n.Frame.H, root.Frame.H) {
		t.Fatalf("全屏尺寸 = %.0fx%.0f, want %.0fx%.0f",
			n.Frame.W, n.Frame.H, root.Frame.W, root.Frame.H)
	}
}
