// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"runtime"
	"testing"
)

// BenchmarkTableVOldPrealloc 旧实现的每帧开销：按 Count 预分配并逐行填充 entries。
func BenchmarkTableVOldPrealloc(b *testing.B) {
	type entry struct {
		row, group int
		isGroup    bool
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		entries := make([]entry, 0, 100000)
		for r := 0; r < 100000; r++ {
			entries = append(entries, entry{row: r, group: -1})
		}
		_ = len(entries)
	}
}

// BenchmarkTableVWindowBuild 新实现：只构建可见窗口（十万行表格）。
func BenchmarkTableVWindowBuild(b *testing.B) {
	a := testApp()
	currentApp.Store(a)
	defer currentApp.Store(nil)
	cfg := TableVConfig{
		Key: "bench-table", Headers: []string{"#", "名称", "状态"},
		RowH: 34, Height: 240, Count: 100000,
		Row: func(i int) []*Node { return []*Node{Text("1"), Text("row"), Text("ok")} },
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TableV(cfg)
	}
}

// TestAnimSlowFlag 低帧率动画标记：AnimSlowOn 降频，普通 AnimOn 仍走高帧率。
func TestAnimSlowFlag(t *testing.T) {
	s := St().AnimSlowOn()
	if !s.Animated || !s.AnimSlow {
		t.Fatalf("AnimSlowOn 应同时置 Animated/AnimSlow，实际 %+v", s)
	}
	if fast := St().AnimOn(); fast.AnimSlow {
		t.Fatal("AnimOn 不应降帧")
	}
	merged := St().merge(*St().AnimSlowOn())
	if !merged.Animated || !merged.AnimSlow {
		t.Fatalf("merge 后丢失 AnimSlow：%+v", merged)
	}
}

// TestHitTestPassthroughOverlay 全空的 AbsFill 叠层不能吃掉点击。
// 回归：套件页 suiteLayers() 在所有浮层都关闭时仍返回空 Stack，
// 导致 hitTest 停在该叠层上，全窗口点击（含标题栏按钮）全部失效。
func TestHitTestPassthroughOverlay(t *testing.T) {
	a := testApp()
	btn := Btn("确定", BtnPrimary, func() {}).K("ok")
	root := Stack(
		Column(btn).St(St().W(200).H(60)),
		Stack(nil, nil).K("overlay").St(St().AbsFill()), // 无任何浮层的空叠层
	).St(St().W(400).H(300))
	Layout(root, 400, 300, newTestCache())

	n := findKey(root, "ok")
	if n == nil {
		t.Fatal("未找到按钮")
	}
	cx, cy := n.Frame.X+n.Frame.W/2, n.Frame.Y+n.Frame.H/2
	got := hitInteractive(root, cx, cy, 0, a)
	if got == nil || got.Key != "ok" {
		t.Fatalf("点击按钮应命中 ok，实际 %+v", got)
	}

	// 空白处：不能被空叠层挡住，应穿透到下层容器
	if hit := hitTest(root, 380, 280, 0, a); hit != nil && hit.Key == "overlay" {
		t.Fatalf("空叠层不应参与命中，实际命中 %+v", hit)
	}
}

// TestNilNodeTapIsSafe 可选节点（如全透明 Scrim 返回 nil）上链式 Tap 不能 panic。
// 回归：CommandPaletteLayer 首帧 t=0 → Scrim 返回 nil → 直接 .Tap() 解引用崩溃。
func TestNilNodeTapIsSafe(t *testing.T) {
	var n *Node
	if got := n.Tap(func() {}); got != nil {
		t.Fatalf("nil 节点 Tap 应返回 nil，实际 %+v", got)
	}
	if got := n.TapAt(func(x, y float32) {}); got != nil {
		t.Fatalf("nil 节点 TapAt 应返回 nil，实际 %+v", got)
	}
	if Scrim(0, "x") != nil {
		t.Fatal("透明度为 0 的 Scrim 应返回 nil")
	}
}

// TestCommandPaletteLayerClosedNoPanic 命令面板关闭时构建浮层不应崩溃。
func TestCommandPaletteLayerClosedNoPanic(t *testing.T) {
	a := testApp()
	currentApp.Store(a)
	defer currentApp.Store(nil)
	paletteOpen = false
	for i := 0; i < 3; i++ {
		CommandPaletteLayer()
	}
}

// tableBodyOf 取表格节点的滚动体（行窗口在这里）。
func tableBodyOf(t *testing.T, key string, n *Node) *Node {
	t.Helper()
	var find func(*Node) *Node
	find = func(x *Node) *Node {
		if x == nil {
			return nil
		}
		if x.Kind == KScroll && x.Key == key {
			return x
		}
		for _, c := range x.Children {
			if r := find(c); r != nil {
				return r
			}
		}
		return nil
	}
	if r := find(n); r != nil {
		return r
	}
	t.Fatalf("未找到表格滚动体 %q", key)
	return nil
}

// rowStart 跳过顶部占位（有 placeholder 时子项 0 的 Key 为空）。
func rowStart(children []*Node) int {
	if len(children) > 0 && children[0] != nil && children[0].Key == "" {
		return 1
	}
	return 0
}

// TestTableVGroupWindow 分组表格的条目映射与窗口化：滚动到任意位置时
// 第一条可见条目应与 entries 下标一致（分组头 / 数据行不串位）。
func TestTableVGroupWindow(t *testing.T) {
	a := testApp()
	currentApp.Store(a)
	defer currentApp.Store(nil)
	delete(tableCollaps, "perf-groups")

	cfg := TableVConfig{
		Key: "perf-groups", Headers: []string{"#", "名称"},
		RowH: 34, Height: 240, Count: 500,
		Groups: []TableGroup{
			{Label: "A", Start: 0, Count: 100},
			{Label: "B", Start: 100, Count: 50},
		},
		Row:   func(i int) []*Node { return []*Node{Text(fmt.Sprint(i)), Text("x")} },
		OnRow: func(int) {},
	}

	body := tableBodyOf(t, cfg.Key, TableV(cfg))
	if len(body.Children) < 2 {
		t.Fatalf("可见行过少：%d", len(body.Children))
	}
	if body.Children[0].Key != "perf-groups-g-0" || body.Children[1].Key != "perf-groups-r-0" {
		t.Fatalf("顶部应为 A 组头 + 第 0 行，实际 %q / %q",
			body.Children[0].Key, body.Children[1].Key)
	}

	// 条目下标 101 = B 组头（A 组 1 个头 + 100 行）；
	// 窗口带 4 行 overscan，所以首个子项是条目 97（第 96 行），B 组头在其后第 4 个。
	a.scrolls[cfg.Key] = 101 * 34
	body = tableBodyOf(t, cfg.Key, TableV(cfg))
	i := rowStart(body.Children)
	if i+5 >= len(body.Children) {
		t.Fatalf("偏移 101 行后可见行不足：%d", len(body.Children))
	}
	if body.Children[i].Key != "perf-groups-r-96" ||
		body.Children[i+4].Key != "perf-groups-g-1" ||
		body.Children[i+5].Key != "perf-groups-r-100" {
		t.Fatalf("滚动后应为 r-96 (overscan) / B 组头 / 第 100 行，实际 %q / %q / %q",
			body.Children[i].Key, body.Children[i+4].Key, body.Children[i+5].Key)
	}

	// 折叠 A 组后条目总数 = 1 + 1 + 50，偏移 0 时先看到 A 头再 B 头
	tableCollaps[cfg.Key][0] = true
	a.scrolls[cfg.Key] = 0
	body = tableBodyOf(t, cfg.Key, TableV(cfg))
	j := rowStart(body.Children)
	if j+2 >= len(body.Children) {
		t.Fatalf("折叠后可见行不足：%d", len(body.Children))
	}
	if body.Children[j].Key != "perf-groups-g-0" || body.Children[j+1].Key != "perf-groups-g-1" ||
		body.Children[j+2].Key != "perf-groups-r-100" {
		t.Fatalf("折叠后应为 A 头 / B 头 / 第 100 行，实际 %q / %q / %q",
			body.Children[j].Key, body.Children[j+1].Key, body.Children[j+2].Key)
	}
}

// TestTableVAllocWindowed 表格构建的分配量不能随 Count 线性增长。
// 旧实现每帧 make([]entry, Count) 并逐行填充，十万行时单帧分配远超 MB 级。
func TestTableVAllocWindowed(t *testing.T) {
	a := testApp()
	currentApp.Store(a)
	defer currentApp.Store(nil)

	cfg := TableVConfig{
		Key: "perf-alloc", Headers: []string{"#", "名称", "状态"},
		RowH: 34, Height: 240, Count: 200000,
		Row: func(i int) []*Node {
			return []*Node{Text("1"), Text("row"), Text("ok")}
		},
	}
	for i := 0; i < 3; i++ {
		TableV(cfg)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	const builds = 20
	for i := 0; i < builds; i++ {
		TableV(cfg)
	}
	runtime.ReadMemStats(&after)
	per := float64(after.TotalAlloc-before.TotalAlloc) / builds
	if per > 1<<20 {
		t.Fatalf("单次构建分配 %.0f KB，疑似仍按行数预分配（应只构建可见行）", per/1024)
	}
}
