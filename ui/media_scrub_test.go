//go:build windows

package ui

import (
	"testing"
	"time"

	"gogui/internal/win32"
)

// TestMediaScrubSeekCommits 拖动中不阻塞且能拿到预览帧，松手后一次性定位到最终位置。
func TestMediaScrubSeekCommits(t *testing.T) {
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
	// 拖动：每次取帧都必须立即返回（不等待解码）
	for _, pos := range []time.Duration{300 * time.Millisecond, 900 * time.Millisecond, 1200 * time.Millisecond} {
		start := time.Now()
		if _, err := src.Frame(pos); err != nil {
			t.Fatalf("拖动取帧(%v): %v", pos, err)
		}
		if d := time.Since(start); d > 60*time.Millisecond {
			t.Fatalf("拖动取帧(%v)耗时 %v，应 <60ms（不阻塞 UI）", pos, d)
		}
	}

	// 拖动中必须记录目标位置（即使目标恰好在缓冲内，松手也要定位）
	src.mu.Lock()
	pending, hasPending := src.pendingSeek, src.hasPending
	src.mu.Unlock()
	if !hasPending || pending != 1200*time.Millisecond {
		t.Fatalf("拖动目标未记录: pending=%v has=%v", pending, hasPending)
	}

	// 拖动中要有单帧预览（画面跟手，而不是冻在旧帧）
	deadline := time.Now().Add(5 * time.Second)
	for {
		src.mu.Lock()
		prev := src.previewFrame
		src.mu.Unlock()
		if prev != nil && prev.Valid() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("拖动中始终没有预览帧（画面不会更新）")
		}
		src.Frame(1200 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)
	}

	// 松手：解码窗口立刻定位到目标位置，且取帧拿到有效画面
	src.MediaScrub(false)
	src.mu.Lock()
	start, seekStart := src.scrubbing, src.start
	src.mu.Unlock()
	if start {
		t.Fatal("松手后 scrubbing 仍为 true")
	}
	if seekStart != 1200*time.Millisecond {
		t.Fatalf("松手后窗口起点 = %v, want 1.2s", seekStart)
	}
	dl := time.Now().Add(3 * time.Second)
	for {
		f, ferr := src.Frame(1200 * time.Millisecond)
		if ferr == nil && f != nil && f.Valid() {
			break
		}
		if time.Now().After(dl) {
			t.Fatalf("松手后定位超时: %v", ferr)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestVideoScrubRelease 松手提交：拖动状态清理并把拖动状态同步给播放器。
func TestVideoScrubRelease(t *testing.T) {
	const key = "test-scrub-stream"
	p := NewPlayer(nil)
	videoScrub[key] = true
	videoScrubAt[key] = AnimTime()
	p.scrub = true

	videoReleaseScrub(key, p)

	if videoScrub[key] {
		t.Fatal("videoReleaseScrub 后仍处于拖动状态")
	}
	if _, ok := videoScrubAt[key]; ok {
		t.Fatal("videoReleaseScrub 后应清理拖动时间戳")
	}
	if p.scrub {
		t.Fatal("videoReleaseScrub 后 Player.scrub 应为 false")
	}
}

// TestDragEndEvent 拖动结束回调：OnDragEnd 在 Node 上可注册并由 App 分发（接口自检）。
func TestDragEndEvent(t *testing.T) {
	called := false
	n := Box().K("drag-node").DragEvt(func(x, y float32) {}).
		DragEndEvt(func() { called = true })
	if n.OnDragEnd == nil {
		t.Fatal("DragEndEvt 未注册回调")
	}
	n.OnDragEnd()
	if !called {
		t.Fatal("OnDragEnd 未执行")
	}
}

// TestMediaScrubAudioFollows 拖动时声音暂停，松手后声音从目标位置继续（音频时钟跟着跳）。
func TestMediaScrubAudioFollows(t *testing.T) {
	if win32.AudioDevices() == 0 {
		t.Skip("没有音频输出设备，跳过声音跟随测试")
	}
	path := makeTestVideo(t)
	src, err := OpenMediaWith(path, MediaConfig{})
	if err != nil {
		t.Fatalf("OpenMediaWith: %v", err)
	}
	defer src.Close()
	if !src.HasAudio() {
		t.Skip("未能打开音频设备，跳过")
	}
	// 先正常播放一小段，让声音真正跑起来
	for i := 0; i < 30; i++ {
		if _, ferr := src.Frame(time.Duration(i*40) * time.Millisecond); ferr != nil {
			t.Fatalf("播放取帧: %v", ferr)
		}
		time.Sleep(15 * time.Millisecond)
	}
	dl0 := time.Now().Add(2 * time.Second)
	for {
		if _, ok := src.MediaClock(); ok {
			break
		}
		if time.Now().After(dl0) {
			t.Fatal("播放中音频时钟无效")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 开拖：声音必须停（否则声音会继续从旧位置播下去）
	src.MediaScrub(true)
	for i := 0; i < 5; i++ {
		src.Frame(1200 * time.Millisecond)
		time.Sleep(15 * time.Millisecond)
	}
	if _, ok := src.MediaClock(); ok {
		t.Fatal("拖动中音频时钟应无效（声音已暂停）")
	}

	// 松手：声音从 1.2s 重新开始，时钟落在目标附近并继续前进
	src.MediaScrub(false)
	dl := time.Now().Add(5 * time.Second)
	var first time.Duration
	for {
		if cp, ok := src.MediaClock(); ok && cp >= 1200*time.Millisecond {
			first = cp
			break
		}
		src.Frame(1200 * time.Millisecond)
		if time.Now().After(dl) {
			t.Fatalf("松手后音频时钟未回到目标位置（当前 %v）", first)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if first > 2200*time.Millisecond {
		t.Fatalf("松手后音频位置 = %v, want ≈1.2s（声音没跟着跳）", first)
	}
}

// TestPlayerScrubEndToEnd 端到端模拟 VideoPlayer 的逐帧驱动：
// 拖动中位置跟手且不推进时钟，松手后位置/画面一起跳到目标。
func TestPlayerScrubEndToEnd(t *testing.T) {
	path := makeTestVideo(t)
	p, err := NewMediaPlayerWith(path, MediaConfig{NoAudio: true})
	if err != nil {
		t.Fatalf("NewMediaPlayerWith: %v", err)
	}
	defer p.Close()
	const key = "e2e-stream"

	// 模拟播放 0.3s（每帧：同步拖动状态 → 推进 → 取帧）
	p.Play()
	for i := 0; i < 15; i++ {
		p.scrub = videoScrub[key]
		p.mediaScrub()
		p.advance(AnimTime())
		time.Sleep(16 * time.Millisecond)
	}

	total := p.Duration()
	if total <= 0 {
		t.Fatalf("时长 = %v", total)
	}
	target := time.Duration(float64(total) * 0.7)

	// 按下拖动：拖动开始 → 进入拖动模式 → 位置跟手
	videoScrub[key] = true
	videoScrubAt[key] = AnimTime()
	p.scrub = true
	p.mediaScrub()
	p.SeekRatio(0.7)
	for i := 0; i < 12; i++ {
		p.scrub = videoScrub[key]
		p.mediaScrub()
		p.advance(AnimTime())
		time.Sleep(16 * time.Millisecond)
	}
	if d := p.Pos() - target; d > 60*time.Millisecond || d < -60*time.Millisecond {
		t.Fatalf("拖动中位置 = %v, want ≈%v（应跟手）", p.Pos(), target)
	}

	// 松手：事件驱动提交
	videoReleaseScrub(key, p)
	dl := time.Now().Add(5 * time.Second)
	var got *VideoFrame
	var at time.Duration
	for {
		p.scrub = videoScrub[key]
		p.mediaScrub()
		p.advance(AnimTime())
		got, at = p.Frame(), p.Pos()
		if got != nil && got.Valid() && at >= target-200*time.Millisecond && at <= target+600*time.Millisecond {
			break
		}
		if time.Now().After(dl) {
			t.Fatalf("松手后画面未更新到目标位置：pos=%v frame=%v", at, got != nil)
		}
		time.Sleep(15 * time.Millisecond)
	}
	delete(videoScrub, key)
}
