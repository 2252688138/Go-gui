// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"testing"

	"gogui/internal/geom"
)

// testApp 构造不含窗口的 App（滚动逻辑与窗口无关，只依赖滚动状态）。
func testApp() *App {
	return &App{
		scrolls:       map[string]float32{},
		scrollTargets: map[string]float32{},
		textures:      newImageTextures(),
		mouseIn:       true,
		animDt:        1.0 / 120,
	}
}

// testScrollTree 构造 root(800x600) → outer(800x400,内容1000) → inner(300x200,内容600)。
func testScrollTree() (*Node, *Node, *Node) {
	root := &Node{Kind: KBox, Style: Style{Opacity: 1}, Frame: geom.Rect{X: 0, Y: 0, W: 800, H: 600}}
	outer := &Node{Kind: KScroll, Key: "outer", Style: Style{Opacity: 1},
		Frame: geom.Rect{X: 0, Y: 0, W: 800, H: 400}, ContentH: 1000, Parent: root}
	inner := &Node{Kind: KScroll, Key: "inner", Style: Style{Opacity: 1},
		Frame: geom.Rect{X: 10, Y: 10, W: 300, H: 200}, ContentH: 600, Parent: outer}
	root.Children = []*Node{outer}
	outer.Children = []*Node{inner}
	return root, outer, inner
}

// TestWheelBubblesToOuterScroll 内层滚动到底后，滚轮剩余量继续作用于外层容器。
// 旧实现命中内层就 return，外层页面完全无法继续滚动（表现为“滚不动”）。
func TestWheelBubblesToOuterScroll(t *testing.T) {
	a := testApp()
	root, _, _ := testScrollTree()
	a.root = root
	a.scrolls["inner"] = 400 // 内层已到底（600-200）

	a.handleWheel(20, 20, -1) // 向下滚一格

	if got := a.scrollTargets["outer"]; !near(got, wheelStepSize) {
		t.Fatalf("内层到底后应把 %v 传给外层，实际 outer target=%v", float32(wheelStepSize), got)
	}
	if got := a.scrollTargets["inner"]; got != 0 {
		t.Fatalf("内层已到底不应再变化，inner target=%v", got)
	}

	// 反过来：内层还能滚时不冒泡到外层
	a2 := testApp()
	a2.root = root
	a2.scrolls["inner"] = 0
	a2.handleWheel(20, 20, -1)
	if got := a2.scrollTargets["outer"]; got != 0 {
		t.Fatalf("内层还能滚动时不应带动外层，outer target=%v", got)
	}
	if got := a2.scrollTargets["inner"]; !near(got, wheelStepSize) {
		t.Fatalf("滚轮应作用于内层，inner target=%v", got)
	}
}

// TestScrollTargetClamped 目标偏移不超过内容范围。
func TestScrollTargetClamped(t *testing.T) {
	a := testApp()
	a.setScrollTarget("k", 9999, 300)
	if got := a.scrollTarget("k"); got != 300 {
		t.Fatalf("目标偏移应夹取到 300，实际 %v", got)
	}
	a.setScrollTarget("k", -50, 300)
	if got := a.scrollTarget("k"); got != 0 {
		t.Fatalf("目标偏移不应为负，实际 %v", got)
	}
}

// TestSmoothScrollConverges 平滑滚动应收敛到目标并清理状态。
func TestSmoothScrollConverges(t *testing.T) {
	a := testApp()
	a.scrolls["k"] = 0
	a.setScrollTarget("k", 300, 1000)
	if !a.isScrollAnimating() {
		t.Fatal("设定目标后应处于滚动动画中")
	}
	for i := 0; i < 400; i++ {
		a.stepScrollAnim()
	}
	if got := a.scrolls["k"]; got != 300 {
		t.Fatalf("滚动应收敛到 300，实际 %v", got)
	}
	if _, ok := a.scrollTargets["k"]; ok {
		t.Fatal("到位后应清理目标位，避免无休止重绘")
	}
	if a.isScrollAnimating() {
		t.Fatal("到位后不应再有滚动动画")
	}
}

// TestScrollbarDragOutsideZone 指针拖出 12px 热区后仍继续跟手。
// 旧实现在鼠标离开热区时直接 return，滑块会“粘”在原地。
func TestScrollbarDragOutsideZone(t *testing.T) {
	a := testApp()
	root, outer, _ := testScrollTree()
	a.root = root
	a.pressed = true

	// 按在滑块上（轨道右侧 12px 内、滑块范围内）
	a.mouseX, a.mouseY = outer.Frame.Right()-6, 20
	a.updateScrollbar(root)
	if a.scrollDragKey != "outer" {
		t.Fatalf("应进入滚动条拖拽状态，实际 %q", a.scrollDragKey)
	}

	a.mouseY = 120
	a.updateScrollbar(root)
	v1 := a.scrolls["outer"]
	if v1 <= 0 {
		t.Fatalf("向下拖拽应增加偏移，实际 %v", v1)
	}

	// 关键：指针横向移出热区，仍处于按下状态
	a.mouseX = 100
	a.mouseY = 160
	a.updateScrollbar(root)
	v2 := a.scrolls["outer"]
	if v2 <= v1 {
		t.Fatalf("拖出热区后应继续跟手：v1=%v v2=%v", v1, v2)
	}
	if v2 > 600 {
		t.Fatalf("偏移不应超过最大滚动量 600，实际 %v", v2)
	}

	// 松手后清除拖拽状态，不应再改变偏移
	a.pressed = false
	a.mouseX, a.mouseY = 100, 300
	a.updateScrollbar(root)
	if a.scrollDragKey != "" {
		t.Fatal("松手后应清除滚动条拖拽状态")
	}
	if a.scrolls["outer"] != v2 {
		t.Fatalf("松手后偏移不应变化：%v → %v", v2, a.scrolls["outer"])
	}
}

// TestScrollTrackClickJumps 点击轨道空白处直接定位（滑块中心对齐指针）。
func TestScrollTrackClickJumps(t *testing.T) {
	a := testApp()
	root, outer, _ := testScrollTree()
	a.root = root
	a.pressed = true
	// 点轨道底部（滑块只占上部 40%）
	a.mouseX, a.mouseY = outer.Frame.Right()-6, 380
	a.updateScrollbar(root)
	if v := a.scrolls["outer"]; v < 500 {
		t.Fatalf("点击轨道底部应跳到接近底部，实际 %v", v)
	}
}

// TestScrollClampedWhenContentShrinks 内容变短后偏移被夹回合法范围。
func TestScrollClampedWhenContentShrinks(t *testing.T) {
	a := testApp()
	a.scrolls["k"] = 500
	a.scrollTargets["k"] = 500
	a.clampScroll("k", 120)
	if a.scrolls["k"] != 120 || a.scrollTargets["k"] != 120 {
		t.Fatalf("应夹取到 120，实际 scroll=%v target=%v", a.scrolls["k"], a.scrollTargets["k"])
	}
}
