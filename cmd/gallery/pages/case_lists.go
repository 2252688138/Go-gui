// Go-gui — gallery 页面拆分：列表与表格（页内分页：列表组件 / 表格示例）
// 列表页签演示文件树 / 图文卡片 / 列表 / 社区帖子 / 分页器，表格页签沿用数据表格等既有示例。

package pages

import (
	"fmt"
	"strings"

	"gogui/ui"
)

// ---- 页面状态 ----

var (
	// listTab 页内分页：list（列表组件）/ table（表格示例），定义在 state.go。
	treeTip     = "点击箭头展开 / 折叠，点击行选中"
	mediaFilter = "all"
	mediaPage   = 1
	rowSel      = "row-2"
	rowChecked  = map[string]bool{"row-1": true, "row-4": true}
	postLiked   = map[string]bool{"post-2": true}
	postPage    = 1
)

// listsTabs 列表与表格页的内部页签（下划线风格，与系统页一致）。
func listsTabs() *ui.Node {
	return ui.Row(
		ui.TabsBar([]string{"list", "table"}, []string{"列表", "表格"}, listTab, func(k string) {
			listTab = k
			ui.RequestRedraw()
		}),
		ui.Spacer(),
		ui.Tag("文件树 / 图文卡片 / 列表 / 社区帖子 / 分页", "#a5b4fc"),
	).St(ui.St().GapOf(12).PadOnly(26, 26, 0, 26)).AlignTo(ui.AlignCenter)
}

// listsCase 列表与表格页：页签切换「列表组件」与「表格示例」。
func listsCase() *ui.Node {
	body := listPage()
	if listTab == "table" {
		body = tablePage()
	}
	return ui.Column(
		listsTabs(),
		ui.Scroll(body).St(ui.St().GrowTo(1)).K("scroll-list-"+listTab),
	).St(ui.St().GrowTo(1).Hp(100))
}

// listPage 列表页签内容。
func listPage() *ui.Node {
	return ui.Column(
		treeSection(),
		mediaSection(),
		rowsSection(),
		postSection(),
	).St(ui.St().GapOf(0))
}

// tablePage 表格页签内容（原有列表 / 表格类示例）。
func tablePage() *ui.Node {
	return ui.Column(
		tableCase(),
		chatSection(),
		padSection(suiteVListCard()),
		padSection(suiteTableCard()),
		padSection(suiteStatesCard()),
	).St(ui.St().GapOf(0))
}

// ---- 文件树 ----

func treeSection() *ui.Node {
	tree := ui.FileTree(galleryTree, ui.TreeOptions{
		Key:      "gal-tree",
		Expanded: []string{"gogui", "gogui/cmd", "gogui/cmd/gallery", "gogui/ui"},
		Selected: "gogui/ui/layout.go",
		OnPick: func(n ui.TreeNode) {
			treeTip = n.Key
			ui.RequestRedraw()
		},
	})
	sel := ui.TreeSelected("gal-tree")
	if sel == "" {
		sel = "gogui/ui/layout.go"
	}
	kind := "文件"
	if n := treeFind(galleryTree, sel); n != nil && n.Kind == ui.TreeFolder {
		kind = "目录"
	}

	return padSection(ui.Card(
		ui.Row(
			ui.Title("文件树", "缩进层级 / 展开折叠 / 选中高亮 / 右侧标签"),
			ui.Spacer(),
			ui.Btn("全部展开", ui.BtnSecondary, func() {
				ui.TreeSetAllOpen("gal-tree", galleryTree, true)
			}),
			ui.Btn("全部收起", ui.BtnGhost, func() {
				ui.TreeSetAllOpen("gal-tree", galleryTree, false)
			}),
		).St(ui.St().GapOf(8)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Scroll(tree).St(ui.St().W(400).H(340)).K("gal-tree-scroll"),
			ui.Column(
				ui.Text(baseName(sel)).St(ui.St().Text(15, 600, ui.ColText)),
				ui.Text(sel).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
				ui.Divider(),
				ui.KeyValue("节点类型", kind),
				ui.KeyValue("父目录", parentPath(sel)),
				ui.KeyValue("展开状态", "点击箭头可折叠 / 展开子项"),
				ui.Divider(),
				ui.Text(treeTip).St(ui.St().Text(11, 400, ui.ColTextMuted).WrapOn()),
			).Grow(1).St(ui.St().GapOf(8).Padding(14).R(12).
				BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke)),
		).St(ui.St().GapOf(16)),
	))
}

// ---- 图文卡片 ----

func mediaSection() *ui.Node {
	all := galleryMedia()
	items := make([]ui.MediaItem, 0, len(all))
	for _, it := range all {
		if mediaFilter == "all" || it.Badge == mediaFilter {
			items = append(items, it)
		}
	}
	cards := ui.MediaCards(items, ui.MediaCardOptions{
		Width: 186,
		OnPick: func(it ui.MediaItem) {
			toastShow("打开：" + it.Title)
		},
		OnAction: func(it ui.MediaItem) {
			toastShow("已设为壁纸：" + it.Title)
		},
	}, 12)

	return padSection(ui.Card(
		ui.Row(
			ui.Title("图文卡片", "封面 / 角标 / 评分 / 标题 / 标签 / 操作按钮，Wrap 自适应换行"),
			ui.Spacer(),
			ui.Tabs([]string{"all", "媒体", "界面", "文档"},
				[]string{"全部", "媒体", "界面", "文档"}, mediaFilter, func(k string) {
					mediaFilter = k
					ui.RequestRedraw()
				}),
			ui.Text(fmt.Sprintf("%d 项", len(items))).St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		cards,
		ui.Row(
			ui.Spacer(),
			ui.Pager(ui.PagerOptions{
				Key: "gal-media", Page: mediaPage, Total: 9, Locked: 6,
				OnPick: func(p int) {
					mediaPage = p
					toastShow(fmt.Sprintf("已切换到第 %d 页", p))
					ui.RequestRedraw()
				},
			}),
			ui.Spacer(),
		).AlignTo(ui.AlignCenter),
	))
}

// ---- 列表 ----

func rowsSection() *ui.Node {
	checked := 0
	for _, it := range galleryRows {
		if rowChecked[it.Key] {
			checked++
		}
	}
	rows := make([]ui.ListRowItem, 0, len(galleryRows))
	for _, it := range galleryRows {
		if v, ok := rowChecked[it.Key]; ok {
			it.Checked = v
		}
		rows = append(rows, it)
	}

	return padSection(ui.Card(
		ui.Row(
			ui.Title("列表", "图标 / 标题 / 副标题 / 右侧标签与说明，支持多选、分隔线与选中高亮"),
			ui.Spacer(),
			ui.Text(fmt.Sprintf("已选 %d 项", checked)).St(ui.St().Text(11.5, 500, ui.ColTextDim)),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		ui.ListRows(rows, ui.ListRowsOptions{
			Key:      "gal-rows",
			Selected: rowSel,
			Dividers: true,
			CheckBox: true,
			OnPick: func(it ui.ListRowItem, i int) {
				rowSel = it.Key
				toastShow("选中：" + it.Title)
				ui.RequestRedraw()
			},
			OnCheck: func(it ui.ListRowItem, i int, on bool) {
				rowChecked[it.Key] = on
			},
		}),
		ui.Row(
			ui.Text("点击行选中，可勾选多行；分页器支持省略号折叠与上锁页").
				St(ui.St().Text(11, 400, ui.ColTextMuted)),
			ui.Spacer(),
			ui.Pager(ui.PagerOptions{
				Key: "gal-rows", Page: 1, Total: 6, Window: 1,
				OnPick: func(p int) { toastShow(fmt.Sprintf("列表第 %d 页", p)) },
			}),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
	))
}

// ---- 社区帖子 ----

func postSection() *ui.Node {
	cards := make([]*ui.Node, 0, len(galleryPosts))
	for _, p := range galleryPosts {
		post := p
		if postLiked[post.Key] {
			post.Liked = true
		}
		cards = append(cards, ui.PostCard(post, ui.PostCardOptions{
			OnLike: func(it ui.PostItem) {
				postLiked[it.Key] = !postLiked[it.Key]
				ui.RequestRedraw()
			},
			OnOpen: func(it ui.PostItem) {
				toastShow("打开帖子：" + it.Author)
			},
		}))
	}
	return padSection(ui.Card(
		ui.Row(
			ui.Title("社区帖子", "头像 / 昵称 / 时间 / 正文 / 话题 / 配图 / 点赞评论转发"),
			ui.Spacer(),
			ui.Pager(ui.PagerOptions{
				Key: "gal-posts", Page: postPage, Total: 4,
				OnPick: func(p int) { postPage = p; ui.RequestRedraw() },
			}),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		ui.Column(cards...).St(ui.St().GapOf(12)),
	))
}

// ---- 演示数据 ----

// galleryTree 文件树示例（结构与仓库一致）。
var galleryTree = []ui.TreeNode{
	{Key: "gogui", Label: "gogui", Kind: ui.TreeFolder, Children: []ui.TreeNode{
		{Key: "gogui/.git", Label: ".git", Kind: ui.TreeFolder},
		{Key: "gogui/.workflow", Label: ".workflow", Kind: ui.TreeFolder},
		{Key: "gogui/.zym", Label: ".zym", Kind: ui.TreeFolder},
		{Key: "gogui/bin", Label: "bin", Kind: ui.TreeFolder},
		{Key: "gogui/cmd", Label: "cmd", Kind: ui.TreeFolder, Tag: "3", Children: []ui.TreeNode{
			{Key: "gogui/cmd/gallery", Label: "gallery", Kind: ui.TreeFolder, Children: []ui.TreeNode{
				{Key: "gogui/cmd/gallery/pages", Label: "pages", Kind: ui.TreeFolder, Tag: "新增", TagColor: ui.ColSuccess},
				{Key: "gogui/cmd/gallery/gallery.exe", Label: "gallery.exe", Kind: ui.TreeFile},
				{Key: "gogui/cmd/gallery/main.go", Label: "main.go", Kind: ui.TreeFile},
				{Key: "gogui/cmd/gallery/rsrc_windows_amd64.syso", Label: "rsrc_windows_amd64.syso", Kind: ui.TreeFile},
			}},
		}},
		{Key: "gogui/dist", Label: "dist", Kind: ui.TreeFolder},
		{Key: "gogui/installer", Label: "installer", Kind: ui.TreeFolder},
		{Key: "gogui/internal", Label: "internal", Kind: ui.TreeFolder},
		{Key: "gogui/ui", Label: "ui", Kind: ui.TreeFolder, Tag: "72", Children: []ui.TreeNode{
			{Key: "gogui/ui/layout.go", Label: "layout.go", Kind: ui.TreeFile, Tag: "M", TagColor: ui.ColWarn},
			{Key: "gogui/ui/tree.go", Label: "tree.go", Kind: ui.TreeFile, Tag: "新增", TagColor: ui.ColSuccess},
			{Key: "gogui/ui/cards.go", Label: "cards.go", Kind: ui.TreeFile, Tag: "新增", TagColor: ui.ColSuccess},
			{Key: "gogui/ui/pager.go", Label: "pager.go", Kind: ui.TreeFile, Tag: "新增", TagColor: ui.ColSuccess},
			{Key: "gogui/ui/widgets.go", Label: "widgets.go", Kind: ui.TreeFile},
			{Key: "gogui/ui/assets", Label: "assets", Kind: ui.TreeFolder},
		}},
		{Key: "gogui/tools", Label: "tools", Kind: ui.TreeFolder},
		{Key: "gogui/go.mod", Label: "go.mod", Kind: ui.TreeFile},
		{Key: "gogui/README.md", Label: "README.md", Kind: ui.TreeLink, Tag: "链接"},
	}},
}

// galleryMedia 图文卡片数据（封面用 ui/assets 下的示例截图，缺失时回落为渐变色块）。
func galleryMedia() []ui.MediaItem {
	return []ui.MediaItem{
		{Key: "m1", Cover: assetCover("视频播放器.png"), Title: "视频播放器", Sub: "逐帧解码 · 全屏 · 倍速",
			Badge: "媒体", Score: "8.1", Tags: []string{"播放", "全屏"}, Action: "设为壁纸", Accent: "#6366f1"},
		{Key: "m2", Cover: assetCover("音乐播放器.png"), Title: "音乐播放器", Sub: "频谱 · 播放列表",
			Badge: "媒体", Score: "7.9", Tags: []string{"音频", "频谱"}, Action: "设为壁纸", Accent: "#22d3ee"},
		{Key: "m3", Cover: assetCover("图表组件.png"), Title: "图表组件", Sub: "折线 / 饼 / 雷达",
			Badge: "界面", Score: "8.5", Tags: []string{"图表"}, Action: "设为壁纸", Accent: "#34d399"},
		{Key: "m4", Cover: assetCover("仪表盘.png"), Title: "仪表盘", Sub: "统计卡 · 活动列表",
			Badge: "界面", Score: "8.8", Tags: []string{"布局", "统计"}, Action: "设为壁纸", Accent: "#fbbf24"},
		{Key: "m5", Cover: assetCover("表格.png"), Title: "数据表格", Sub: "斑马纹 · 行内进度",
			Badge: "界面", Score: "8.3", Tags: []string{"表格"}, Action: "设为壁纸", Accent: "#38bdf8"},
		{Key: "m6", Cover: assetCover("表单.png"), Title: "表单与弹窗", Sub: "校验 · 抽屉 · 模态",
			Badge: "界面", Score: "8.4", Tags: []string{"表单"}, Action: "设为壁纸", Accent: "#a78bfa"},
		{Key: "m7", Cover: assetCover("markdown渲染.png"), Title: "Markdown 渲染", Sub: "代码高亮 · 图片预览",
			Badge: "文档", Score: "9.0", Tags: []string{"文档", "富文本"}, Action: "设为壁纸", Accent: "#f472b6"},
		{Key: "m8", Cover: assetCover("webview组件.png"), Title: "WebView 浏览器", Sub: "内嵌网页子窗口",
			Badge: "文档", Score: "8.6", Tags: []string{"系统", "网页"}, Action: "设为壁纸", Accent: "#818cf8"},
		{Key: "m9", Cover: assetCover("系统操作.png"), Title: "系统操作", Sub: "拖入 · 剪贴板 · 热键",
			Badge: "界面", Score: "7.8", Tags: []string{"系统"}, Action: "设为壁纸", Accent: "#f87171"},
		{Key: "m10", Cover: assetCover("浅色主题.png"), Title: "浅色主题", Sub: "语义 token 运行时切换",
			Badge: "界面", Score: "8.0", Tags: []string{"主题"}, Action: "设为壁纸", Accent: "#0ea5e9"},
	}
}

// galleryRows 列表行数据。
var galleryRows = []ui.ListRowItem{
	{Key: "row-1", Title: "设计稿终稿.fig", Sub: "2.4 MB · 昨天 18:20 更新", Meta: "昨天",
		Tag: "设计", Color: "#6366f1", Icon: "设"},
	{Key: "row-2", Title: "render_pipeline.go", Sub: "45 KB · 渲染管线重构", Meta: "2 分钟前",
		Tag: "代码", TagColor: "#34d399", Color: "#22d3ee", Icon: "代"},
	{Key: "row-3", Title: "季度汇报.pdf", Sub: "共 32 页 · 已共享给 8 人", Meta: "周一",
		Tag: "文档", TagColor: "#fbbf24", Color: "#a78bfa", Icon: "文"},
	{Key: "row-4", Title: "布局引擎性能剖析", Sub: "拖动窗口时的测量耗时采样", Meta: "3 天前",
		Tag: "分析", TagColor: "#f472b6", Color: "#f472b6", Icon: "析"},
	{Key: "row-5", Title: "构建产物 gallery.exe", Sub: "v1.4.2 · 12.6 MB", Meta: "上周",
		Tag: "产物", TagColor: "#38bdf8", Color: "#38bdf8", Glyph: "✓"},
}

// galleryPosts 社区帖子数据。
var galleryPosts = []ui.PostItem{
	{Key: "post-1", Author: "林深", AvatarBg: "#6366f1", Role: "楼主", Time: "2 小时前 · 来自组件讨论区",
		Body:   "把列表页重做了一遍：文件树、图文卡片、列表、社区帖子、分页器全部抽成可复用组件，页内再用下划线页签切「列表 / 表格」，切换时滚动位置互不干扰。",
		Images: []string{assetCover("表格.png"), assetCover("仪表盘.png")},
		Tags:   []string{"组件", "界面框架"}, Likes: 128, Comments: 24, Shares: 6},
	{Key: "post-2", Author: "阿澄", AvatarBg: "#22d3ee", Role: "开发者", Time: "昨天 21:40",
		Body:   "文件树的展开状态按 Key 记忆，折叠的子树不构建节点，长目录也不会掉帧；分页器的上锁页直接禁用点击，悬停有提示。",
		Images: []string{assetCover("视频播放器.png"), assetCover("音乐播放器.png"), assetCover("图表组件.png")},
		Tags:   []string{"性能", "交互"}, Likes: 96, Comments: 13, Shares: 2, Liked: true},
}

// ---- 小工具 ----

// assetCover 把 ui/assets 下的示例截图当作封面；找不到（工作目录不同）时返回空串，
// 由组件回落成渐变色块，避免出现空图占位。
func assetCover(name string) string {
	for _, p := range []string{
		"ui/assets/" + name,
		"../ui/assets/" + name,
		"../../ui/assets/" + name,
	} {
		if w, _ := ui.ImageSize(p); w > 0 {
			return p
		}
	}
	return ""
}

// baseName 取路径最后一段。
func baseName(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// parentPath 取父目录路径。
func parentPath(p string) string {
	if i := strings.LastIndex(p, "/"); i > 0 {
		return p[:i]
	}
	return "—"
}

// treeFind 按 Key 查找树节点。
func treeFind(list []ui.TreeNode, key string) *ui.TreeNode {
	for i := range list {
		if list[i].Key == key {
			return &list[i]
		}
		if n := treeFind(list[i].Children, key); n != nil {
			return n
		}
	}
	return nil
}
