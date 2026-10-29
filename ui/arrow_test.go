// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestArrowCoverage 三角纹理必须是真正的右向三角：中心实、四角空、尖端覆盖率低。
func TestArrowCoverage(t *testing.T) {
	const n = arrowTexDim
	px := arrowCoverage(n)
	alpha := func(x, y int) byte { return px[(y*n+x)*4+3] }
	if a := alpha(n/2, n/2); a < 250 {
		t.Fatalf("三角中心应完全覆盖，实际 alpha=%d", a)
	}
	for _, p := range [][2]int{{0, 0}, {n - 1, 0}, {0, n - 1}, {n - 1, n - 1}, {1, n / 2}} {
		if a := alpha(p[0], p[1]); a != 0 {
			t.Fatalf("(%d,%d) 不应有像素，实际 alpha=%d", p[0], p[1], a)
		}
	}
	if alpha(n-4, n/2) >= alpha(n/2, n/2) {
		t.Fatal("尖端覆盖率应低于主体")
	}
}

// TestArrowNode 指示器节点尺寸/颜色就绪。
func TestArrowNode(t *testing.T) {
	n := Arrow("#5b6480")
	if n == nil || n.Kind != KArrow {
		t.Fatal("Arrow 应返回 KArrow 节点")
	}
	if n.Style.Width <= 0 || n.Style.Height <= 0 {
		t.Fatalf("尺寸非法: %vx%v", n.Style.Width, n.Style.Height)
	}
	if n.Style.Color.A <= 0 {
		t.Fatal("颜色不应透明")
	}
}
