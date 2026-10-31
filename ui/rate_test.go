// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestRateHalfStarClip 半星渲染：上层实心星按 26 宽居中排版，由 13 宽裁剪容器
// 截出左半。若裁剪把文本 Frame 收缩成 13，居中文字会被重新居中，视觉上就是
// 「实心星和空心星错位叠加」的异常。
func TestRateHalfStarClip(t *testing.T) {
	root := Column(Rate("t-rate", 3.5, 5, true, nil))
	Layout(root, 400, 80, newTestCache())

	row := root.Children[0]
	if row == nil || len(row.Children) != 5 {
		t.Fatalf("Rate 子项数异常: %+v", row)
	}
	half := row.Children[3]
	if half.Kind != KStack {
		t.Fatalf("第 4 颗星应为半星 Stack, got kind %v", half.Kind)
	}
	clip := half.Children[1]
	if !near(clip.Frame.W, 13) || !near(clip.Frame.H, 26) {
		t.Fatalf("裁剪容器 = %.1fx%.1f, want 13x26", clip.Frame.W, clip.Frame.H)
	}
	txt := clip.Children[0]
	if !near(txt.Frame.W, 26) {
		t.Fatalf("实心星文本宽 = %.1f, want 26（不能被裁剪区收缩，否则居中偏移）", txt.Frame.W)
	}
	if !near(txt.Frame.X, clip.Frame.X) {
		t.Fatalf("实心星文本 x = %.1f, want %.1f（左对齐裁切）", txt.Frame.X, clip.Frame.X)
	}
}
