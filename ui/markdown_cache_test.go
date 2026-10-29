// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestMarkdownTreeCacheReuse 相同输入必须复用上一次的渲染树（同一批 *Node），
// 而根节点仍然每帧新建（否则外层的 Key/Style 会被上一帧污染）。
func TestMarkdownTreeCacheReuse(t *testing.T) {
	MarkdownResetState()
	src := mdTestDoc
	a := Markdown("mdk", src, MarkdownOptions{Width: 700})
	b := Markdown("mdk", src, MarkdownOptions{Width: 700})
	if a == b {
		t.Fatal("根节点必须每帧新建")
	}
	if len(a.Children) == 0 || len(a.Children) != len(b.Children) {
		t.Fatalf("子节点数量不一致：%d / %d", len(a.Children), len(b.Children))
	}
	for i := range a.Children {
		if a.Children[i] != b.Children[i] {
			t.Fatalf("第 %d 个子节点未命中缓存（应复用同一指针）", i)
		}
	}
}

// TestMarkdownTreeCacheInvalidation 影响排版的输入发生变化时必须重建。
func TestMarkdownTreeCacheInvalidation(t *testing.T) {
	sameTree := func(x, y *Node) bool {
		if len(x.Children) != len(y.Children) {
			return false
		}
		for i := range x.Children {
			if x.Children[i] != y.Children[i] {
				return false
			}
		}
		return len(x.Children) > 0
	}

	MarkdownResetState()
	base := Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 700})

	// 1) 宽度变化
	if sameTree(base, Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 520})) {
		t.Fatal("宽度变化后不应复用缓存")
	}
	// 2) 交互状态变化（MarkdownResetState 会 +1）
	MarkdownResetState()
	if sameTree(base, Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 700})) {
		t.Fatal("交互状态版本变化后不应复用缓存")
	}
	// 3) 正文字号变化
	if sameTree(base, Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 700, BaseSize: 16})) {
		t.Fatal("字号变化后不应复用缓存")
	}
	// 4) 节点 Key 变化（页签/折叠状态按 Key 区分）
	if sameTree(base, Markdown("other", mdTestDoc, MarkdownOptions{Width: 700})) {
		t.Fatal("Key 变化后不应复用缓存")
	}
	// 5) 干扰项：命中缓存时不能误判为重建
	MarkdownResetState()
	x := Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 700})
	y := Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 700})
	if !sameTree(x, y) {
		t.Fatal("相同输入应命中缓存")
	}
}

// BenchmarkMarkdownCached 命中渲染树缓存（当前热路径）。
func BenchmarkMarkdownCached(b *testing.B) {
	MarkdownResetState()
	_ = Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 720}) // 预热
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 720})
	}
}

// BenchmarkMarkdownRebuild 每帧完整重建渲染树（旧行为）。
func BenchmarkMarkdownRebuild(b *testing.B) {
	MarkdownResetState()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mdTreeCache = map[mdTreeKey][]*Node{}
		_ = Markdown("mdk", mdTestDoc, MarkdownOptions{Width: 720})
	}
}
