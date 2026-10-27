package ui

import (
	"testing"
	"time"
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
