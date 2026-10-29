// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestSliderThumbCentered 滑块（thumb）与轨道在容器内垂直居中：
// 旧实现 thumb 用 AbsTL(x, 0) 贴顶，视觉上比 4px 轨道高 2px，看起来"没对齐"。
func TestSliderThumbCentered(t *testing.T) {
	root := Column(Slider("sl", 0.5, nil)).St(St().W(200))
	Layout(root, 200, 18, newTestCache())

	s := findKey(root, "sl")
	if s == nil || len(s.Children) != 3 {
		t.Fatalf("slider 结构异常: %v", s)
	}
	track := s.Children[0]
	thumb := s.Children[2]
	trackMid := track.Frame.Y + track.Frame.H/2
	thumbMid := thumb.Frame.Y + thumb.Frame.H/2
	if !near(trackMid, thumbMid) {
		t.Fatalf("滑块中心 %.1f 应与轨道中心 %.1f 对齐", thumbMid, trackMid)
	}
	if !near(thumbMid, s.Frame.H/2) {
		t.Fatalf("滑块中心 %.1f 应位于容器中线 %.1f", thumbMid, s.Frame.H/2)
	}
}

// TestSliderHitReturnsSlider 命中测试应返回滑块节点本身：
// 滑块只注册 OnClickAt/OnDrag，旧 hitInteractive 只认 OnClick，
// 导致按下时命中内部 Box、dragKey 建立不起来 —— 拖动无实时反馈。
func TestSliderHitReturnsSlider(t *testing.T) {
	root := Column(Slider("sl", 0.5, nil)).St(St().W(200))
	Layout(root, 200, 18, newTestCache())

	s := findKey(root, "sl")
	a := &App{scrolls: map[string]float32{}}
	h := hitInteractive(root, s.Frame.X+40, s.Frame.Y+s.Frame.H/2, 0, a)
	if h == nil || h.Key != "sl" {
		t.Fatalf("命中应是滑块 sl，实际 %+v", h)
	}
	if h.OnDrag == nil || h.OnClickAt == nil {
		t.Fatal("滑块应同时具备 OnDrag 与 OnClickAt")
	}
}
