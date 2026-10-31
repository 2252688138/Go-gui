// Go-gui — gallery 页面拆分：基础页输入控件的布局冒烟测试
// 只做无窗口布局：确认新增的下拉框 / 多行输入等控件都能排布出有效尺寸。

package pages

import (
	"testing"

	"gogui/ui"
)

// findKeyNode 按 Key 深度查找节点（测试用）。
func findKeyNode(n *ui.Node, key string) *ui.Node {
	if n == nil {
		return nil
	}
	if n.Key == key {
		return n
	}
	for _, c := range n.Children {
		if got := findKeyNode(c, key); got != nil {
			return got
		}
	}
	return nil
}

// TestBasicPageInputControls 基础页新增控件：节点存在且尺寸有效。
func TestBasicPageInputControls(t *testing.T) {
	root := basicCase()
	if root == nil {
		t.Fatal("basicCase 返回 nil")
	}
	ui.LayoutForTest(root, 1000, 1600, "", 1)

	for _, k := range []string{
		"basic-role", "basic-team", "basic-bio",
		"basic-num", "basic-date", "basic-search", "basic-agree",
		"basic-sw-def", "basic-sw-sm", "basic-sw-lg", "basic-sw-dis",
		"basic-sw-sparkle", "basic-sw-icon", "basic-sw-label", "basic-sw-grad",
		"basic-sw-iconnode", "basic-sw-glyph", "basic-sw-text", "basic-sw-node",
	} {
		n := findKeyNode(root, k)
		if n == nil {
			t.Fatalf("基础页缺少节点 %s", k)
		}
		if n.Frame.W <= 0 || n.Frame.H <= 0 {
			t.Fatalf("%s 尺寸无效: %.1fx%.1f", k, n.Frame.W, n.Frame.H)
		}
	}
}
