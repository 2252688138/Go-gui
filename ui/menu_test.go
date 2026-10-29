package ui

import (
	"testing"
	"time"

	"gogui/internal/geom"
)

// 鼠标离开菜单后超时自动隐藏；指针仍停在菜单内时不隐藏。
func TestMenuAutoHide(t *testing.T) {
	st := &MenuBarState{Open: 0, AutoHide: 20 * time.Millisecond}
	st.lastHover = time.Now().Add(-time.Second)
	if !st.sweepAutoHide() {
		t.Fatal("离开超时后应关闭菜单")
	}
	if st.Open != -1 || st.Sub != nil {
		t.Fatalf("关闭后状态未复位: open=%d sub=%v", st.Open, st.Sub)
	}

	hold := &MenuBarState{Open: 0, AutoHide: 20 * time.Millisecond, inside: true}
	hold.lastHover = time.Now().Add(-time.Second)
	if hold.sweepAutoHide() {
		t.Fatal("指针仍在菜单内时不应关闭")
	}

	// 负数表示不自动隐藏
	off := &MenuBarState{Open: 0, AutoHide: -1}
	off.lastHover = time.Now().Add(-time.Hour)
	if off.sweepAutoHide() {
		t.Fatal("AutoHide<0 时不应自动关闭")
	}
}

// 下拉浮层随 Open / Sub 渲染，关闭后返回 nil；子菜单可多层嵌套。
func TestMenuLayerNested(t *testing.T) {
	menus := []MenuDef{{Label: "文件", Items: []MenuItem{
		{Label: "打开"},
		{Label: "导出", Children: []MenuItem{
			{Label: "PNG"},
			{Label: "PDF", Children: []MenuItem{{Label: "高清"}}},
		}},
	}}}
	st := NewMenuBar()
	defer func() { st.Close() }()

	if MenuLayer(st, menus, 10, 40) != nil {
		t.Fatal("未展开时不应渲染浮层")
	}

	st.Open = 0
	st.lastHover = time.Now()
	layer := MenuLayer(st, menus, 10, 40)
	if layer == nil {
		t.Fatal("展开后应渲染浮层")
	}
	if got := countKeyPrefix(layer, "menu-"); got != 2 {
		t.Fatalf("一级菜单项数量 = %d, want 2", got)
	}

	st.Sub = []int{1, 1}
	deep := MenuLayer(st, menus, 10, 40)
	if got := countKeyPrefix(deep, "menu-"); got != 5 {
		t.Fatalf("多级展开后菜单项数量 = %d, want 5", got)
	}

	st.Close()
	if MenuLayer(st, menus, 10, 40) != nil {
		t.Fatal("关闭后不应渲染浮层")
	}
}

// TestMenuSwitchWhileOpen 已展开菜单时，其它顶级菜单必须仍然可命中：
// 浮层不能盖住菜单栏，否则命中会停在浮层上（它没有任何交互回调），
// 表现为「必须先关掉当前菜单，才能点其它菜单」。
func TestMenuSwitchWhileOpen(t *testing.T) {
	menus := []MenuDef{
		{Label: "文件", Items: []MenuItem{{Label: "打开"}, {Label: "退出"}}},
		{Label: "编辑", Items: []MenuItem{{Label: "复制"}}},
	}
	st := NewMenuBar()
	defer st.Close()

	// 第一项菜单保持展开，菜单栏位于标题栏内（高度 40，浮层锚点即菜单栏底部）
	st.Open = 0
	title := Column(Row(MenuBar(st, menus)).St(St().H(40))).St(St().AbsTL(0, 0).Wp(100))
	root := Stack(title, MenuLayer(st, menus, 10, 40))
	Layout(root, 800, 600, newTestCache())
	root.Frame = geom.Rect{X: 0, Y: 0, W: 800, H: 600}

	item := findKey(root, "menubar-编辑")
	if item == nil {
		t.Fatal("找不到菜单栏按钮")
	}
	if item.Frame.Y >= 40 {
		t.Fatalf("菜单栏按钮应位于浮层上方, y=%v", item.Frame.Y)
	}
	x, y := item.Frame.X+item.Frame.W/2, item.Frame.Y+item.Frame.H/2
	hit := hitInteractive(root, x, y, 0, &App{})
	if hit == nil || hit.Key != "menubar-编辑" {
		t.Fatalf("菜单展开时点菜单栏应命中菜单按钮，实际命中 %v", hit)
	}

	// 点另一个顶级菜单：旧菜单立即隐藏，直接切到被点的菜单
	hit.OnClick()
	if st.Open != 1 {
		t.Fatalf("点击后应切到第 2 个菜单, Open=%d", st.Open)
	}
	if len(st.Sub) != 0 {
		t.Fatalf("切换后子菜单路径应清空, Sub=%v", st.Sub)
	}

	// 悬停切换：重新展开第 1 项后，指针滑到第 2 项也应立即切换
	st.Open = 0
	st.Sub = []int{0}
	hit.OnHover(true)
	if st.Open != 1 || len(st.Sub) != 0 {
		t.Fatalf("悬停其它菜单应立即切换, Open=%d Sub=%v", st.Open, st.Sub)
	}
}

func countKeyPrefix(n *Node, prefix string) int {
	if n == nil {
		return 0
	}
	c := 0
	if len(n.Key) >= len(prefix) && n.Key[:len(prefix)] == prefix {
		c++
	}
	for _, ch := range n.Children {
		c += countKeyPrefix(ch, prefix)
	}
	return c
}
