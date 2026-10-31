// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// treeSample 测试用小型目录树：root 展开后应有 3 行（root / a.go / sub）。
func treeSample() []TreeNode {
	return []TreeNode{
		{Key: "root", Label: "root", Kind: TreeFolder, Children: []TreeNode{
			{Key: "a", Label: "a.go", Kind: TreeFile},
			{Key: "sub", Label: "sub", Kind: TreeFolder, Children: []TreeNode{
				{Key: "b", Label: "b.go", Kind: TreeFile},
			}},
		}},
	}
}

// buildTree 构建并布局一棵文件树（每轮重新构建，验证跨帧状态记忆）。
func buildTree(items []TreeNode, key string) *Node {
	root := Column(FileTree(items, TreeOptions{
		Key: key, Expanded: []string{"root"}, Selected: "a",
	})).St(St().W(320))
	Layout(root, 320, 600, newTestCache())
	return root
}

// rowLabelX 取行内指定文本节点的左边缘 x（测试用）。
func rowLabelX(row *Node, text string) float32 {
	for _, c := range row.Children {
		if c.Kind == KText && c.Text == text {
			return c.Frame.X
		}
	}
	return -1
}

// TestFileTreeVisibleRows 折叠的子树不构建行；展开 / 折叠后行数随之变化。
func TestFileTreeVisibleRows(t *testing.T) {
	delete(treeOpen, "tv-rows")
	delete(treeSel, "tv-rows")
	items := treeSample()

	root := buildTree(items, "tv-rows")
	tree := findKey(root, "tv-rows-root")
	if tree == nil {
		t.Fatal("找不到文件树节点")
	}
	if len(tree.Children) != 3 {
		t.Fatalf("可见行 = %d, 期望 3（root / a.go / sub）", len(tree.Children))
	}
	for i, row := range tree.Children {
		if !near(row.Frame.H, 30) {
			t.Fatalf("第 %d 行高 = %.1f, 期望 30", i, row.Frame.H)
		}
	}

	// 子节点必须比父节点缩进更深（比同名文本节点左边缘）
	parentX := rowLabelX(tree.Children[0], "root")
	childX := rowLabelX(tree.Children[1], "a.go")
	if childX <= parentX {
		t.Fatalf("子节点缩进 %.1f 应大于父节点 %.1f", childX, parentX)
	}

	// 点击 sub 行展开
	sub := tree.Children[2]
	if sub.OnClick == nil {
		t.Fatal("目录行应注册点击回调")
	}
	sub.OnClick()
	root = buildTree(items, "tv-rows")
	tree = findKey(root, "tv-rows-root")
	if len(tree.Children) != 4 {
		t.Fatalf("展开后可见行 = %d, 期望 4", len(tree.Children))
	}

	// 再点一次折叠
	tree.Children[2].OnClick()
	root = buildTree(items, "tv-rows")
	tree = findKey(root, "tv-rows-root")
	if len(tree.Children) != 3 {
		t.Fatalf("折叠后可见行 = %d, 期望 3", len(tree.Children))
	}
}

// TestFileTreeSelectAndExpandAll 点击叶子选中；TreeSetAllOpen 展开整棵树。
func TestFileTreeSelectAndExpandAll(t *testing.T) {
	delete(treeOpen, "tv-all")
	delete(treeSel, "tv-all")
	items := treeSample()
	root := buildTree(items, "tv-all")
	tree := findKey(root, "tv-all-root")

	leaf := tree.Children[1]
	leaf.OnClick()
	if got := TreeSelected("tv-all"); got != "a" {
		t.Fatalf("选中节点 = %q, 期望 a", got)
	}

	TreeSetAllOpen("tv-all", items, true)
	root = buildTree(items, "tv-all")
	tree = findKey(root, "tv-all-root")
	if len(tree.Children) != 4 {
		t.Fatalf("全部展开后可见行 = %d, 期望 4", len(tree.Children))
	}

	TreeSetAllOpen("tv-all", items, false)
	root = buildTree(items, "tv-all")
	tree = findKey(root, "tv-all-root")
	if len(tree.Children) != 1 {
		t.Fatalf("全部收起后可见行 = %d, 期望 1（只剩 root）", len(tree.Children))
	}
}

// TestMediaCardLayout 图文卡片：固定宽度、封面高度、操作按钮可点。
func TestMediaCardLayout(t *testing.T) {
	root := Column(MediaCards([]MediaItem{{
		Key: "a", Title: "视频播放器", Sub: "逐帧解码", Badge: "媒体", Score: "8.1",
		Tags: []string{"播放"}, Action: "设为壁纸",
	}}, MediaCardOptions{Width: 186, OnAction: func(MediaItem) {}}, 12)).St(St().W(420))
	Layout(root, 420, 400, newTestCache())

	card := findKey(root, "mediacard:a")
	if card == nil {
		t.Fatal("找不到卡片节点")
	}
	if !near(card.Frame.W, 186) {
		t.Fatalf("卡片宽度 = %.1f, 期望 186", card.Frame.W)
	}
	if card.Frame.H <= 118 {
		t.Fatalf("卡片高度 = %.1f, 应大于封面高度 118", card.Frame.H)
	}
	btn := findKey(root, "mediacard:a:action")
	if btn == nil || btn.Frame.H <= 0 || btn.OnClick == nil {
		t.Fatalf("操作按钮异常: %+v", btn)
	}
}

// TestListRowsLayout 列表行：行高、分隔线、多选框与选中高亮。
func TestListRowsLayout(t *testing.T) {
	items := []ListRowItem{
		{Key: "r1", Title: "设计稿", Sub: "2.4 MB", Meta: "昨天", Tag: "设计", Icon: "设", Color: "#6366f1"},
		{Key: "r2", Title: "渲染管线", Sub: "45 KB", Meta: "2 分钟前", Tag: "代码", Glyph: "✓"},
	}
	root := Column(ListRows(items, ListRowsOptions{
		Key: "lv", Selected: "r2", Dividers: true, CheckBox: true,
		OnPick: func(ListRowItem, int) {},
	})).St(St().W(560))
	Layout(root, 560, 300, newTestCache())

	for _, k := range []string{"listrow:lv:r1", "listrow:lv:r2", "listrow:lv:r1:cb"} {
		n := findKey(root, k)
		if n == nil || n.Frame.W <= 0 || n.Frame.H <= 0 {
			t.Fatalf("列表缺少节点或尺寸无效: %s", k)
		}
	}
	row := findKey(root, "listrow:lv:r2")
	if !near(row.Frame.H, 50) {
		t.Fatalf("行高 = %.1f, 期望 50", row.Frame.H)
	}
	if row.OnClick == nil {
		t.Fatal("列表行应注册点击回调")
	}
}

// TestPostCardLayout 社区帖子：头像 / 正文 / 配图 / 操作按钮尺寸有效。
func TestPostCardLayout(t *testing.T) {
	root := Column(PostCard(PostItem{
		Key: "p1", Author: "林深", Time: "2 小时前", Role: "楼主",
		Body: "正文内容", Tags: []string{"组件"},
		Likes: 12, Comments: 3, Shares: 1,
	}, PostCardOptions{Width: 640})).St(St().W(700))
	Layout(root, 700, 400, newTestCache())

	card := findKey(root, "post:p1")
	if card == nil || !near(card.Frame.W, 640) {
		t.Fatalf("帖子卡片异常: %+v", card)
	}
	like := findKey(root, "post:p1:12")
	if like == nil || like.OnClick == nil {
		t.Fatalf("点赞按钮异常: %+v", like)
	}
}

// TestPagerLayout 分页器：当前页 / 省略号 / 上锁页禁用。
func TestPagerLayout(t *testing.T) {
	root := Column(Pager(PagerOptions{Key: "pg", Page: 1, Total: 9, Locked: 6})).
		St(St().W(400))
	Layout(root, 400, 100, newTestCache())

	first := findKey(root, "pager:pg:1")
	if first == nil || !near(first.Frame.W, 34) {
		t.Fatalf("第 1 页按钮异常: %+v", first)
	}
	if findKey(root, "pager:pg:9") == nil {
		t.Fatal("最后一页应始终显示")
	}
	// 第 9 页始终显示，且在 Locked=6 时上锁
	locked := findKey(root, "pager:pg:9")
	if locked == nil {
		t.Fatal("缺少第 9 页按钮")
	}
	if !locked.Disabled {
		t.Fatal("上锁页应禁用点击")
	}
	if locked.OnClick != nil {
		t.Fatal("上锁页不应注册点击回调")
	}
	// 第 8 页被省略号折叠（当前页 1、窗口 2）
	if findKey(root, "pager:pg:8") != nil {
		t.Fatal("第 8 页应被省略号折叠")
	}
}
