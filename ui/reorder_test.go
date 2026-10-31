// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestReorderListRowEvents 拖拽重排的行必须注册 OnDrag。
// App 只在 pressNode.OnDrag != nil 时才建立 dragKey，漏了 OnDrag 会导致
// 松手时 OnDragEnd 永不分发，表现为「拖拽重排完全没作用」。
func TestReorderListRowEvents(t *testing.T) {
	rows := []string{"a", "b", "c", "d"}
	n := ReorderList(ReorderListOptions{
		Key: "t-reorder", Count: len(rows), RowH: 30, Height: 120,
		Row: func(i int) *Node {
			return Row(Text(rows[i])).St(St().H(30))
		},
		OnMove: func(from, to int) { rows = MoveSlice(rows, from, to) },
	})
	var item *Node
	for _, c := range n.Children {
		if c.Key == "t-reorder-item-1" {
			item = c
		}
	}
	if item == nil {
		t.Fatal("未找到行节点 t-reorder-item-1")
	}
	if item.OnPress == nil {
		t.Fatal("行节点应注册 OnPress")
	}
	if item.OnDrag == nil {
		t.Fatal("行节点必须注册 OnDrag，否则 dragKey 不会建立、OnDragEnd 永不触发")
	}
	if item.OnDragEnd == nil {
		t.Fatal("行节点应注册 OnDragEnd")
	}

	// 模拟拖动：按下第 1 行 → 目标第 3 行 → 松手。
	item.OnPress()
	reorderTarget["t-reorder"] = 3
	item.OnDragEnd()
	if got := rows[3]; got != "b" {
		t.Fatalf("松手后 b 应落在第 4 位，实际 %v", rows)
	}
	delete(reorderDrag, "t-reorder")
	delete(reorderTarget, "t-reorder")
	delete(reorderStale, "t-reorder")
}

// TestReorderListLayout 行高与容器高度按参数布局（拖动行的半透明/指示线不改变布局）。
func TestReorderListLayout(t *testing.T) {
	n := ReorderList(ReorderListOptions{
		Key: "t-reorder-layout", Count: 20, RowH: 30, Height: 220,
		Row:    func(i int) *Node { return Text("r").St(St().H(30)) },
		OnMove: func(from, to int) {},
	})
	LayoutForTest(n, 300, 220, "", 1)
	if n.Frame.H < 219.5 || n.Frame.H > 220.5 {
		t.Fatalf("容器高度应为 220，实际 %.1f", n.Frame.H)
	}
	for _, c := range n.Children {
		if c.Key == "t-reorder-layout-item-2" {
			if c.Frame.H < 29.5 || c.Frame.H > 30.5 {
				t.Fatalf("行高应为 30，实际 %.1f", c.Frame.H)
			}
			return
		}
	}
	t.Fatal("未找到行节点 t-reorder-layout-item-2")
}

// TestReorderListNoDragKeepsState 未拖动时不应残留半透明/指示线状态。
func TestReorderListNoDragKeepsState(t *testing.T) {
	_ = ReorderList(ReorderListOptions{
		Key: "t-reorder2", Count: 5, RowH: 30, Height: 120,
		Row:    func(i int) *Node { return Text("r") },
		OnMove: func(from, to int) {},
	})
	if _, ok := reorderDrag["t-reorder2"]; ok {
		t.Fatal("未拖动时不应存在拖动状态")
	}
	n := ReorderList(ReorderListOptions{
		Key: "t-reorder3", Count: 5, RowH: 30, Height: 120,
		Row:    func(i int) *Node { return Text("r") },
		OnMove: func(from, to int) {},
	})
	if got := len(n.Children); got != 6 { // 5 行 + 底部占位 0（无占位时即 5）
		if got != 5 && got != 6 {
			t.Fatalf("未拖动时不应额外插入指示线，实际子节点 %d", got)
		}
	}
}
