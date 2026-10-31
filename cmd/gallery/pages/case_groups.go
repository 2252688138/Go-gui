// Go-gui — gallery 页面拆分：按组件类型聚合的分类页
// 本文件把原来分散的单页示例重新组合成 8 个类型页：
// 基础组件 / 表单弹窗 / 列表表格 / 媒体播放 / 布局仪表 / 图表 / 文本富文本 / 系统操作。
// 具体示例实现仍在各自的 case_*.go 里，这里只负责组合与页签。

package pages

import (
	"gogui/ui"
)

// padSection 给没有外边距的套件卡片补上与其它案例一致的页面内边距。
func padSection(n *ui.Node) *ui.Node {
	return ui.Column(n).St(ui.St().Padding(26))
}

// ---- 基础组件 ----

// basicCase 基础组件：按钮 / 输入 / 滑块 / 开关 / 下拉框 / 多行输入 / 导航 / 提示 / 数据展示。
func basicCase() *ui.Node {
	return ui.Column(
		controlsCase(),
		basicInputsSection(),
		settingsCase(),
		padSection(suiteDisplayCard()),
	).St(ui.St().GapOf(0))
}

// ---- 表单与弹窗 ----

// formPage 表单与弹窗：基础表单 + 高级输入类组件 + 四类弹窗 / 抽屉。
func formPage() *ui.Node {
	return ui.Column(
		formCase(),
		padSection(suiteInputCard()),
	).St(ui.St().GapOf(0))
}

// ---- 列表与表格 ----

// chatSection 聊天消息列表（内部自带滚动，给固定高度避免与外层滚动冲突）。
func chatSection() *ui.Node {
	return ui.Column(
		pageHead("聊天消息列表", "气泡消息 / 输入发送 / 列表内部滚动"),
		ui.Box(chatCase()).St(ui.St().H(520)),
	).St(ui.St().Padding(26).GapOf(12))
}

// 列表类页面（listsCase / listPage / tablePage）在 case_lists.go：
// 页内用下划线页签切换「列表组件」与「表格示例」。

// ---- 布局与仪表盘 ----

// layoutCase 布局能力 + 统计卡 / 进度 / 活动列表组合示例。
func layoutCase() *ui.Node {
	return ui.Column(
		layoutsCase(),
		dashboardCase(),
	).St(ui.St().GapOf(0))
}

// ---- 文本与富文本 ----

// textCase 文本类组件：Markdown 渲染 + 富文本显示 / 编辑器。
func textCase() *ui.Node {
	return ui.Column(
		markdownCase(),
		padSection(suiteRichCard()),
	).St(ui.St().GapOf(0))
}

// ---- 媒体播放 ----

// mediaTabs 媒体页页签：视频 / 音频。
func mediaTabs() *ui.Node {
	return ui.Row(
		ui.TabsBar([]string{"video", "music"}, []string{"视频播放", "音频播放"}, mediaTab, func(k string) {
			mediaTab = k
			ui.RequestRedraw()
		}),
		ui.Spacer(),
		ui.Tag("媒体操作", "#a5b4fc"),
	).St(ui.St().GapOf(12).PadOnly(26, 26, 0, 26)).AlignTo(ui.AlignCenter)
}

// mediaCase 媒体操作：视频与音频播放合并在同一页，用页签切换。
func mediaCase() *ui.Node {
	body := musicCase()
	if mediaTab == "video" {
		body = videoCase()
	}
	return ui.Column(
		mediaTabs(),
		ui.Scroll(body).St(ui.St().GrowTo(1)).K("scroll-media-"+mediaTab),
	).St(ui.St().GrowTo(1).Hp(100))
}

// ---- 系统操作 ----

// sysTabs 系统页页签：系统能力 / WebView（网页子窗口不能放进 Scroll）。
func sysTabs() *ui.Node {
	return ui.Row(
		ui.TabsBar([]string{"cap", "webview"}, []string{"系统能力", "WebView 浏览器"}, sysTab, func(k string) {
			sysTab = k
			ui.RequestRedraw()
		}),
		ui.Spacer(),
		ui.Tag("系统级操作", "#a5b4fc"),
	).St(ui.St().GapOf(12).PadOnly(26, 26, 0, 26)).AlignTo(ui.AlignCenter)
}

// systemCase 系统级操作：系统能力 + 桌面能力 + 命令面板 + WebView。
func systemCase() *ui.Node {
	if sysTab == "webview" {
		return ui.Column(
			sysTabs(),
			// 注意：不要套 Box + 子节点 Hp(100)。父级主轴是无上限测量的，
			// 百分比高度会解析成 0 再回退成内容高，WebView 节点会被排成 0 高
			// （子窗口得不到 bounds，网页一片空白）。用 Grow 撑满剩余空间。
			webviewCase().Grow(1),
		).St(ui.St().GrowTo(1).Hp(100))
	}
	return ui.Column(
		sysTabs(),
		ui.Scroll(ui.Column(
			sysCase(),
			padSection(suiteDeskCard()),
			padSection(suiteCmdCard()),
		).St(ui.St().GapOf(0))).St(ui.St().GrowTo(1)).K("scroll-sys"),
	).St(ui.St().GrowTo(1).Hp(100))
}
