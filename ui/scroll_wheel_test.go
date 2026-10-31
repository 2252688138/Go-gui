// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestWheelFirstScroll 滚轮首次滚动：偏移表 scrolls 是懒建的，首次滚轮时还没有
// 该 key 的条目。若 stepScrollAnim 把「条目不存在」当成无效目标丢弃，滚轮就完全
// 不响应，直到拖动一次滚动条（会直接写 scrolls）后才恢复正常。
func TestWheelFirstScroll(t *testing.T) {
	root := Column(
		Scroll(Box().St(St().H(1000))).St(St().W(100).H(100)).K("page"),
	)
	Layout(root, 100, 100, newTestCache())

	a := &App{
		root:          root,
		scrolls:       map[string]float32{},
		scrollTargets: map[string]float32{},
	}
	a.animDt = 1.0 / 60
	a.handleWheel(50, 50, -1) // 向下滚一格

	if _, ok := a.scrollTargets["page"]; !ok {
		t.Fatalf("滚轮未设置滚动目标: %+v", a.scrollTargets)
	}
	if _, ok := a.scrolls["page"]; !ok {
		t.Fatal("setScrollTarget 应惰性建出 scrolls 条目")
	}

	for i := 0; i < 90; i++ {
		a.stepScrollAnim()
	}
	if got := a.scrolls["page"]; got < wheelStepSize-1 {
		t.Fatalf("缓动后偏移 = %.1f, want ≈%.0f", got, float32(wheelStepSize))
	}
}
