// Go-gui — gallery 页面拆分：列表与表格页的布局冒烟测试
// 只做无窗口布局：确认两个页签的区块都能排出有效尺寸。

package pages

import (
	"testing"

	"gogui/ui"
)

// TestListsPageListTab 列表页签：文件树 / 图文卡片 / 列表 / 社区帖子 / 分页器都在。
func TestListsPageListTab(t *testing.T) {
	prev := listTab
	listTab = "list"
	defer func() { listTab = prev }()

	root := listsCase()
	if root == nil {
		t.Fatal("listsCase 返回 nil")
	}
	ui.LayoutForTest(root, 1000, 1800, "", 1)

	for _, k := range []string{
		"scroll-list-list",
		"gal-tree-root",
		"gal-tree-scroll",
		"mediacard:m1",
		"listrow:gal-rows:row-1",
		"post:post-1",
		"pager:gal-media",
	} {
		n := findKeyNode(root, k)
		if n == nil {
			t.Fatalf("列表页签缺少节点 %s", k)
		}
		if n.Frame.W <= 0 || n.Frame.H <= 0 {
			t.Fatalf("%s 尺寸无效: %.1fx%.1f", k, n.Frame.W, n.Frame.H)
		}
	}
}

// TestListsPageTableTab 表格页签：数据表格与虚拟列表仍然存在。
func TestListsPageTableTab(t *testing.T) {
	prev := listTab
	listTab = "table"
	defer func() { listTab = prev }()

	root := listsCase()
	ui.LayoutForTest(root, 1000, 1800, "", 1)

	if findKeyNode(root, "scroll-list-table") == nil {
		t.Fatal("表格页签缺少滚动容器")
	}
	// tableCase 的表头文本与虚拟列表的滚动键
	if findKeyNode(root, "suite-vlist") == nil {
		t.Fatal("表格页签缺少虚拟列表示例")
	}
}
