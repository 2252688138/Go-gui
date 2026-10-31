// Go-gui — gallery 页面拆分：共享状态 / 页面路由 / 全局动作
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
)

// Current 当前分类页（组件已按类型聚合，登录/注册等业务页不再演示）。
var Current = "basic"

// CaseItem 一个组件分类页的元信息（ID 即 --case / ?case= 的参数值）。
type CaseItem struct {
	ID, Label, Icon, Desc string
}

// Cases 左侧导航：按组件类型分组，每个分类一页。
//
//	基础组件  ← 控件总览 / 设置（开关、单选、滑块、折叠）/ 数据展示
//	表单弹窗  ← 表单校验 / 高级输入 / 四类弹窗 / 抽屉
//	列表表格  ← 页签：列表（文件树 / 图文卡片 / 列表 / 社区帖子 / 分页）/ 表格（数据表格 / 聊天 / 虚拟列表 / 拖拽重排）
//	媒体播放  ← 视频播放 / 音频播放（页签切换）
//	布局仪表  ← 8 种布局 / 统计卡与活动列表
//	图表      ← 折线 / 面积 / 柱状 / 饼 / 环形 / 雷达 / 散点
//	文本富文本 ← Markdown / 富文本 / 代码高亮
//	系统操作  ← 系统能力 / 桌面能力 / 命令面板 / WebView（页签切换）
var Cases = []CaseItem{
	{"basic", "基础组件", "▩", "按钮 / 输入 / 下拉框 / 多行输入 / 滑块 / 开关 / 导航 / 提示"},
	{"form", "表单与弹窗", "▦", "表单校验 / 下拉 / 日期 / 上传 / 四类弹窗 / 抽屉"},
	{"list", "列表与表格", "▤", "文件树 / 图文卡片 / 列表 / 社区帖子 / 分页 / 数据表格（页内页签切换）"},
	{"media", "媒体播放", "◉", "视频 / 音频 / 频谱 / 播放列表"},
	{"layout", "布局与仪表盘", "▨", "Flex / 网格 / 换行 / 瀑布流 / 叠层 / 响应式 / 统计卡"},
	{"charts", "图表", "◐", "折线 / 面积 / 柱状 / 饼 / 环形 / 雷达 / 散点"},
	{"text", "文本与富文本", "✎", "Markdown 渲染 / 富文本 / 代码高亮"},
	{"system", "系统操作", "◈", "文件与拖入 / 剪贴板 / 热键 / 托盘 / WebView / 打印"},
}

// ---- 案例状态 ----
var (
	volume       float32 = 0.6
	autoSave             = true
	syncOn               = false
	Theme                = "深色"
	accOpen              = true
	settingsText         = ""

	msgInput = ""
	messages = []chatMsg{
		{false, "你好，GoGUI 组件库更新了吗？"},
		{true, "已经更新：新增 Stack 叠层、绝对定位与表单控件。"},
		{false, "那聊天界面能做吗？"},
		{true, "当然，下面这个案例就是用它拼的。"},
	}

	// 控件总览页状态（CtrlMenu 供壳层的浮动菜单面板使用）
	CtrlMenu    bool
	ctrlPwd     string
	ctrlBanner          = true
	ctrlSlider  float32 = 0.35
	ctrlTab             = "ui"
	ctrlAccOpen         = true
	ctrlRow             = -1
	ctrlTip             = "悬停图标按钮可看到提示"

	// listTab 列表与表格页的内分页：list（列表组件）/ table（表格示例）。
	listTab = "list"
	// mediaTab 媒体播放页页签：video / music。
	mediaTab = "video"
	// sysTab 系统操作页页签：cap（系统能力）/ webview（内嵌浏览器）。
	sysTab = "cap"
)

// toastShow 弹出一条轻提示：默认 2 秒后自动隐藏（ui.ToastDefaultSeconds 可调），
// 传空串等同立即隐藏。
func toastShow(msg string) { Toast(msg) }

type chatMsg struct {
	me   bool
	text string
}

// aliasCase 兼容旧页面 id（--case video / 菜单里的 controls 等）：
// 统一折叠到新的分类页，并顺带切换对应页签。
func aliasCase(id string) string {
	switch id {
	case "video":
		mediaTab = "video"
		return "media"
	case "music":
		mediaTab = "music"
		return "media"
	case "webview":
		sysTab = "webview"
		return "system"
	case "login", "controls", "settings", "suite":
		return "basic"
	case "dashboard":
		return "layout"
	case "table":
		listTab = "table"
		return "list"
	case "chat":
		listTab = "table"
		return "list"
	case "markdown":
		return "text"
	}
	return id
}

// normalizeCurrent 把外部直接写入的 Current（启动参数 / URL 参数）规范化。
func normalizeCurrent() { Current = aliasCase(Current) }

// SwitchCase 切换当前页面（左侧导航 / 菜单 / 快捷入口统一走这里）。
func SwitchCase(id string) {
	Current = aliasCase(id)
	ui.RequestRedraw()
}

// SetTheme 切换主题：深色 / 浅色 / 跟随系统（真实应用到 ui 组件库）。
func SetTheme(name string) {
	switch name {
	case "浅色":
		ui.UseTheme(ui.ThemeLight)
	case "跟随系统", "系统":
		ui.UseTheme(ui.ThemeSystem)
	default:
		ui.UseTheme(ui.ThemeDark)
	}
	Theme = name
	toastShow("主题已切换：" + name)
	ui.RequestRedraw()
}

// CaseDesc 当前页面的描述文案（左侧导航底部）。
func CaseDesc() string {
	for _, c := range Cases {
		if c.ID == Current {
			return c.Desc
		}
	}
	return ""
}

// Content 按当前页面 id 构建右侧内容区。
func Content() *ui.Node {
	normalizeCurrent()
	switch Current {
	case "basic":
		return ui.Scroll(basicCase()).St(ui.St().GrowTo(1)).K("scroll-basic")
	case "form":
		return ui.Scroll(formPage()).St(ui.St().GrowTo(1)).K("scroll-form")
	case "list":
		// 列表页内部自带页签 + 滚动区（同 media / system 页）
		return listsCase()
	case "media":
		// 媒体页内部自带页签 + 滚动区
		return mediaCase()
	case "layout":
		return ui.Scroll(layoutCase()).St(ui.St().GrowTo(1)).K("scroll-layout")
	case "charts":
		return ui.Scroll(chartsCase()).St(ui.St().GrowTo(1)).K("scroll-charts")
	case "text":
		return ui.Scroll(textCase()).St(ui.St().GrowTo(1)).K("scroll-text")
	case "system":
		// 系统页内部自带页签；WebView 是独立子窗口，不能放进 Scroll
		return systemCase()
	}
	return ui.Scroll(basicCase()).St(ui.St().GrowTo(1)).K("scroll-basic")
}

func pageHead(title, sub string) *ui.Node {
	return ui.Column(
		ui.Text(title).St(ui.St().Text(22, 700, ui.ColText)),
		ui.Text(sub).St(ui.St().Text(12, 400, ui.ColTextMuted).PadOnly(4, 0, 0, 0)),
	).St(ui.St().PadOnly(0, 0, 6, 0))
}

// ---- pages 包的对外 API（供 main.go 的壳层使用）----

// Toast 轻提示钩子，由 main 注入为 ui.ToastShow。
var Toast = func(msg string) {}

// TitleMenu 标题栏菜单栏状态（跨帧保持，壳层与页面共用）。
var TitleMenu = ui.NewMenuBar()

// SetVideoFullscreen 切换视频播放器全屏（播放器懒创建，可能为 nil）。
func SetVideoFullscreen(on bool) {
	if videoPl != nil {
		videoPl.Fullscreen = on
	}
}

// VideoFullscreen 播放器当前是否处于全屏。
func VideoFullscreen() bool { return videoPl != nil && videoPl.Fullscreen }

// SetVideoURL 设置播放直链（Web 端 ?media= 参数用）。
func SetVideoURL(u string) { videoURL = u }

// VideoURL 当前播放直链。
func VideoURL() string { return videoURL }
