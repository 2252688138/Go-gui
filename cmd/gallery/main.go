//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// gallery 桌面版：含视频/音乐/WebView/系统能力等 Windows 专属案例。
// Web 版是同一目录下的 web.go（js && wasm），入口同为 go build ./cmd/gallery。
// 运行：go run ./cmd/gallery
// 布局调试：go run ./cmd/gallery --dump dump.txt --case table
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gogui/internal/version"
	"gogui/ui"
)

var current = "login"

type caseItem struct {
	id, label, icon, desc string
}

var cases = []caseItem{
	{"login", "登录页", "→", "表单校验 / 输入框 / 记住我"},
	{"dashboard", "仪表盘", "▤", "统计卡 / 进度 / 活动列表"},
	{"settings", "设置页", "⚙", "开关 / 单选 / 滑块 / 折叠"},
	{"chat", "聊天", "✉", "气泡消息 / 输入发送"},
	{"table", "数据表格", "▦", "表头 / 斑马纹 / 状态标签"},
	{"layouts", "布局方式", "▨", "Flex / 网格 / 换行 / 瀑布流 / 叠层 / 分布 / 响应式"},
	{"controls", "控件总览", "▩", "按钮变体 / 输入 / 滑块 / 表格 / 菜单 / 提示"},
	{"form", "表单与弹窗", "▦", "下拉 / 多行 / 数字 / 日期 / 四类弹窗 / 抽屉"},
	{"system", "系统能力", "◈", "图片 / 拖入文件 / 毛玻璃 / Emoji / 动画 / 输入法 / 系统操作"},
	{"music", "音乐播放器", "♫", "封面 / 频谱特效 / 进度 / 播放列表"},
	{"video", "视频播放器", "◉", "mp4/mkv/mov/avi/webm/flv/ts/m3u8 + 控制条"},
	{"markdown", "Markdown 渲染", "✎", "标题 / 列表 / 引用 / 代码 / 表格 / 扩展块"},
	{"webview", "WebView 浏览器", "◎", "系统 WebView2 真网页 / 自绘地址栏 / 前进后退"},
	{"charts", "图表", "◐", "折线 / 面积 / 柱状 / 饼 / 环形 / 雷达 / 散点"},
}

// ---- 案例状态 ----
var (
	userName = ""
	password = ""
	remember = true
	loginTip = ""

	volume       float32 = 0.6
	autoSave             = true
	syncOn               = false
	theme                = "深色"
	accOpen              = true
	settingsText         = ""

	msgInput = ""
	messages = []chatMsg{
		{false, "你好，GoGUI 组件库更新了吗？"},
		{true, "已经更新：新增 Stack 叠层、绝对定位与表单控件。"},
		{false, "那聊天界面能做吗？"},
		{true, "当然，下面这个案例就是用它拼的。"},
	}

	// 控件总览页状态
	ctrlMenu    bool
	ctrlPwd     string
	ctrlBanner          = true
	ctrlSlider  float32 = 0.35
	ctrlTab             = "ui"
	ctrlAccOpen         = true
	ctrlRow             = -1
	ctrlTip             = "悬停图标按钮可看到提示"
)

// toastShow 弹出一条轻提示：默认 2 秒后自动隐藏（ui.ToastDefaultSeconds 可调），
// 传空串等同立即隐藏。
func toastShow(msg string) { ui.ToastShow(msg) }

type chatMsg struct {
	me   bool
	text string
}

func main() {
	opts := ui.Options{
		Title:      "GoGUI 组件演示 " + version.String(),
		Width:      1180,
		Height:     760,
		NoTitleBar: true,
		// 窗口圆角：Win11 走 DWM 原生圆角（平滑），低版本回退 SetWindowRgn 区域圆角
		CornerRadius: 5,
		// FontFamily 预设值（按子串匹配、区分大小写；未命中的任意名称一律回落微软雅黑）：
		//   空 / "Microsoft YaHei" → msyh.ttc（微软雅黑，粗体 msyhbd.ttc）  默认
		//   "SimSun" / 宋体 → simsun.ttc          "SimHei" / 黑体 → simhei.ttf
		//   "Consolas" / Mono → consola.ttf       "Segoe UI" / Segoe → segoeui.ttf
		//   "Symbol" → seguisym.ttf               "Emoji" → seguiemj.ttf
		FontFamily: "Mtrueicrosoft YaHei UI", // 备注：含 "YaHei" 即命中雅黑（多了 true 也不影响匹配）
		Background: "#070a1215",
	}
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--dump":
			if i+1 < len(os.Args) {
				i++
				opts.DumpLayout = os.Args[i]
			}
		case "--md-tab":
			// 调试用：Markdown 示例页默认页签（gfm / ext / code / long）
			if i+1 < len(os.Args) {
				i++
				mdDemoTab = os.Args[i]
			}
		case "--case":
			if i+1 < len(os.Args) {
				i++
				current = os.Args[i]
			}
		case "--menu":
			// 调试用：启动即展开指定顶级菜单（索引从 0 开始）
			if i+1 < len(os.Args) {
				i++
				if n, err := strconv.Atoi(os.Args[i]); err == nil {
					titleMenu.Open = n
				}
			}
		case "--sub":
			// 调试用：展开子菜单路径，例如 --sub 0,1
			if i+1 < len(os.Args) {
				i++
				for _, p := range strings.Split(os.Args[i], ",") {
					if n, err := strconv.Atoi(p); err == nil {
						titleMenu.Sub = append(titleMenu.Sub, n)
					}
				}
			}
		}
	}
	if err := ui.Run(opts, view); err != nil {
		fmt.Println("fatal:", err)
	}
}

func view() *ui.Node {
	nav := []*ui.Node{
		ui.Text("界面案例").St(ui.St().Text(19, 700, "#e8ecff")),
		ui.Text(fmt.Sprintf("%d 个界面骨架 + 布局实验室", len(cases))).St(ui.St().Text(11.5, 400, ui.ColTextMuted).PadOnly(0, 0, 18, 0)),
	}
	for i, c := range cases {
		id := c.id
		labelColor := ui.ColTextDim
		if current == id {
			labelColor = ui.ColText
		}
		// 用色块代替字符图标：避免字体缺字形导致列表观感异常
		dotColor := [...]string{"#818cf8", "#22d3ee", "#f472b6", "#34d399", "#fbbf24"}[i%5]
		row := ui.Row(
			ui.Box().St(ui.St().W(6).H(6).R(2).BgC(dotColor)),
			ui.Text(c.label).St(ui.St().Text(13, 500, labelColor)),
		).St(ui.St().GapOf(10).PadXY(12, 9).R(9).W(196))
		if current == id {
			row.St(ui.St().BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke))
		}
		nav = append(nav, row.K("case-"+id).Tap(func() {
			current = id
			loginTip = ""
			ui.RequestRedraw()
		}).AlignTo(ui.AlignCenter))
	}
	nav = append(nav, ui.Spacer(),
		ui.Text(caseDesc()).St(ui.St().Text(11, 400, ui.ColTextMuted).WrapOn()))

	body := ui.Row(
		ui.Column(nav...).St(ui.St().W(232).Hp(100).Padding(18).BgC(ui.ColBgSoft).GapOf(4)),
		content(),
	).St(ui.St().GrowTo(1).BgC(ui.ColBg))

	// 标题栏（菜单栏）单独作为最上层浮层：
	// 先用等高占位撑出内容区，再把标题栏放到 Stack 靠后的位置绘制，
	// 这样无论内容怎么滚动/溢出，顶部菜单栏永远压在它们之上（不会被组件遮挡）。
	shell := ui.Column(
		ui.Box().St(ui.St().H(titleBarH+1)), // 占位：标题栏 + 下方分隔线
		body,
	).St(ui.St().Hp(100).BgC(ui.ColBg))

	titleLayer := titleBar().St(ui.St().AbsTL(0, 0).Wp(100))

	return ui.Stack(shell, titleLayer,
		// 播放器全屏：在同一窗口根 Stack 里铺满，不改系统窗口
		ui.If(current == "video" && videoPl != nil && videoPl.Fullscreen, func() *ui.Node {
			return ui.Stack(
				ui.VideoPlayer("gvf", videoPl),
				ui.Row(
					ui.Text("退出全屏（Esc）").St(ui.St().Text(11.5, 600, "#ffffff").
						PadXY(12, 6).R(7).BgC("rgba(8,12,20,0.72)").
						Stroke(1, "rgba(255,255,255,0.18)")),
				).St(ui.St().AbsTL(16, 14)).Tap(func() {
					videoPl.Fullscreen = false
					ui.SetFocus("")
					if cur := ui.Current(); cur != nil {
						cur.SetFullscreen(false)
					}
					videoMsg = "已退出全屏"
					ui.RequestRedraw()
				}),
			).St(ui.St().AbsFill())
		}),
		ui.If(ctrlMenu, func() *ui.Node {
			return ui.MenuPanel(690, 150, []ui.MenuItem{
				{Label: "新建文件", Hint: "Ctrl+N", OnPick: func() { ctrlMenu = false; toastShow("新建文件") }},
				{Label: "打开项目", Hint: "Ctrl+O", OnPick: func() { ctrlMenu = false; toastShow("打开项目") }},
				{Label: "导出为", Children: []ui.MenuItem{
					{Label: "PNG 图片", OnPick: func() { ctrlMenu = false; toastShow("已导出 PNG") }},
					{Label: "PDF 文档", Hint: "Beta", OnPick: func() { ctrlMenu = false; toastShow("已导出 PDF") }},
				}},
				{Separator: true},
				{Label: "更多设置…", Disabled: true},
			}, 180)
		}),
		// 顶部标题栏菜单的下拉浮层（含多级子菜单）
		ui.MenuLayer(titleMenu, titleMenus(), titleMenuX(), titleBarH),
		ui.TooltipLayer(),
		// 表单浮层：下拉框选项面板 / 日历面板（锚点取控件上一帧矩形）
		ui.SelectLayers(),
		ui.DateLayers(),
		// 表单案例的弹窗：Alert / Confirm / Prompt / 自定义 Modal / Drawer
		formOverlays(),
		// Markdown 图片放大层（点击 markdown 中的图片后全屏预览）
		ui.MarkdownLightbox(),
		ui.ToastLayer(),
	).St(ui.St().Hp(100))
}

// ---- 顶部标题栏（自绘：品牌 + 菜单栏 + 窗口按钮 + 拖动）----

const titleBarH float32 = 40

// titleMenu 标题栏菜单栏状态（跨帧保持）。
var titleMenu = ui.NewMenuBar()

// brandWidth 品牌区宽度（与 Row 内实际布局一致，用于菜单浮层定位）。
func brandWidth() float32 { return 18 + 8 + ui.TextWidth("GoGUI", 12.5, 600) }

// titleMenuX 菜单栏左边缘的窗口坐标（标题栏左内边距 10 + 品牌区 + 间距 14）。
func titleMenuX() float32 { return 10 + brandWidth() + 14 }

func titleBar() *ui.Node {
	// dragTitle 按下标题栏空白/品牌区时交给系统拖动窗口
	dragTitle := func() {
		titleMenu.Close()
		if a := ui.Current(); a != nil {
			a.BeginDragMove()
		}
	}

	brand := ui.Row(
		ui.Box().St(ui.St().W(18).H(18).R(5).BgGrad("#6366f1", "#8b5cf6", 120)),
		ui.Text("GoGUI").St(ui.St().Text(12.5, 600, ui.ColText)),
	).St(ui.St().GapOf(8)).AlignTo(ui.AlignCenter)
	// 品牌区也可以拖窗口（里面没有可交互控件）
	brand.OnPressEvt(dragTitle)

	bar := ui.Row(
		brand,
		ui.MenuBar(titleMenu, titleMenus()),
		// 拖拽区：只有标题栏空白处才交给系统拖动窗口。
		// 以前是把 OnPress 挂在整条 Column 上，按菜单/按钮时一旦命中的是空白或间隙
		// 就会进入系统移动模态循环（SC_MOVE），Windows 会在窗口四周画一圈拖动边标，
		// 按住不放就一直存在，松开才消失——这正是“点顶部菜单冒出一圈边标”的来源。
		// 注意：无子节点的 Box 高度会塔陷为 0，必须显式给高，否则命中不到、拖不动窗口。
		ui.Box().St(ui.St().GrowTo(1).H(titleBarH)).OnPressEvt(dragTitle),
		ui.WindowButtons(),
	).St(ui.St().H(titleBarH).PadOnly(0, 10, 0, 8).GapOf(14).BgC(ui.ColBgSoft)).
		AlignTo(ui.AlignCenter)

	return ui.Column(bar, ui.Divider()).K("titlebar")
}

func titleMenus() []ui.MenuDef {
	return []ui.MenuDef{
		{Label: "文件", Items: []ui.MenuItem{
			{Label: "打开", Children: []ui.MenuItem{
				{Label: "图片…", OnPick: func() { toastShow("打开图片（示例）") }},
				{Label: "视频…", OnPick: func() { switchCase("video") }},
				{Label: "音频…", OnPick: func() { switchCase("music") }},
			}},
			{Label: "新建窗口", Hint: "Ctrl+N", OnPick: openChildWindow},
			{Label: "关闭当前窗口", Hint: "Alt+F4", OnPick: closeWindow},
			{Separator: true},
			{Label: "退出", OnPick: closeWindow},
		}},
		{Label: "编辑", Items: []ui.MenuItem{
			{Label: "复制当前页面名", Hint: "Ctrl+C", OnPick: func() { toastShow("已复制：" + current) }},
			{Label: "粘贴", Hint: "Ctrl+V", Disabled: true},
			{Separator: true},
			{Label: "查找", Children: []ui.MenuItem{
				{Label: "当前页面内查找…", Hint: "Ctrl+F", OnPick: func() { toastShow("查找（示例）") }},
				{Label: "全局搜索…", Hint: "Ctrl+Shift+F", OnPick: func() { toastShow("全局搜索（示例）") }},
			}},
		}},
		{Label: "视图", Items: []ui.MenuItem{
			{Label: "界面案例", Children: caseMenuItems()},
			{Separator: true},
			{Label: "全屏切换", Hint: "F11", OnPick: toggleFullscreen},
			{Label: "主题", Children: []ui.MenuItem{
				{Label: "深色", Checked: theme == "深色", OnPick: func() { setTheme("深色") }},
				{Label: "浅色", Checked: theme == "浅色", OnPick: func() { setTheme("浅色") }},
				{Label: "跟随系统", Checked: theme == "跟随系统", OnPick: func() { setTheme("跟随系统") }},
			}},
		}},
		{Label: "自定义", Items: []ui.MenuItem{
			{Label: "二级菜单演示", Children: []ui.MenuItem{
				{Label: "直接执行项", Hint: "Enter", OnPick: func() { toastShow("二级菜单项") }},
				{Label: "三级菜单演示", Children: []ui.MenuItem{
					{Label: "三级项 A", OnPick: func() { toastShow("三级菜单项 A") }},
					{Label: "三级项 B", OnPick: func() { toastShow("三级菜单项 B") }},
					{Separator: true},
					{Label: "禁用项", Disabled: true},
				}},
				{Separator: true},
				{Label: "带快捷键的项", Hint: "Ctrl+K", OnPick: func() { toastShow("自定义菜单项") }},
			}},
			{Separator: true},
			{Label: "当前页面：" + current, Disabled: true},
		}},
		{Label: "帮助", Items: []ui.MenuItem{
			{Label: "菜单用法", Children: []ui.MenuItem{
				{Label: "MenuBar / MenuLayer", OnPick: func() { switchCase("controls") }},
				{Label: "MenuItem.Children 嵌套", OnPick: func() { toastShow("Children 非空即为子菜单，可多层嵌套") }},
			}},
			{Label: "关于 GoGUI", OnPick: func() { toastShow(version.Full() + " · 自绘标题栏与多级菜单示例") }},
		}},
	}
}

// caseMenuItems 把示例列表映射成菜单项（演示动态生成的子菜单）。
func caseMenuItems() []ui.MenuItem {
	out := make([]ui.MenuItem, 0, len(cases))
	for _, c := range cases {
		id := c.id
		out = append(out, ui.MenuItem{
			Label:   c.label,
			Checked: current == id,
			OnPick:  func() { switchCase(id) },
		})
	}
	return out
}

func switchCase(id string) {
	current = id
	loginTip = ""
	ui.RequestRedraw()
}

// setTheme 切换主题：深色 / 浅色 / 跟随系统（真实应用到 ui 组件库）。
func setTheme(name string) {
	switch name {
	case "浅色":
		ui.UseTheme(ui.ThemeLight)
	case "跟随系统", "系统":
		ui.UseTheme(ui.ThemeSystem)
	default:
		ui.UseTheme(ui.ThemeDark)
	}
	theme = name
	toastShow("主题已切换：" + name)
	ui.RequestRedraw()
}

// uiScaleRatio 当前缩放对应的滑块位置（0~1）。
func uiScaleRatio() float32 {
	return (ui.UIScale() - ui.UIScaleMin) / (ui.UIScaleMax - ui.UIScaleMin)
}

// setUIScaleRatio 滑块位置 → 实际缩放（60%~150%）。
func setUIScaleRatio(v float32) {
	ui.SetUIScale(ui.UIScaleMin + v*(ui.UIScaleMax-ui.UIScaleMin))
}

// ssLabel 超采样倍数的可读文案。
func ssLabel(v float32) string {
	if v <= 1.01 {
		return "标准 1:1"
	}
	return strconv.FormatFloat(float64(v), 'g', 3, 32) + "× 超采样"
}

// setSuperSample 切换清晰度档位（自动 / 标准 / 1.25× / 1.5× / 2×），立即热生效。
func setSuperSample(mode float32) {
	ui.SetSuperSample(mode)
	toastShow("清晰度：" + ssLabel(ui.SuperSample()))
	ui.RequestRedraw()
}

func toggleFullscreen() {
	a := ui.Current()
	if a == nil {
		return
	}
	videoPl.Fullscreen = !a.IsFullscreen()
	a.SetFullscreen(!a.IsFullscreen())
	ui.RequestRedraw()
}

func closeWindow() {
	if a := ui.Current(); a != nil {
		a.Close()
	}
}

func openChildWindow() {
	if _, err := ui.NewWindow(ui.Options{Title: "子窗口 · GoGUI", Width: 520, Height: 360}, simpleChildView); err != nil {
		toastShow("打开子窗口失败：" + err.Error())
		return
	}
	toastShow("已打开子窗口")
}

func simpleChildView() *ui.Node {
	return ui.Center(ui.Column(
		ui.Text("子窗口").St(ui.St().Text(20, 700, ui.ColText)),
		ui.Text("从顶部标题栏菜单「文件 → 新建窗口」打开。").St(ui.St().Text(12, 400, ui.ColTextMuted)),
	).St(ui.St().GapOf(8)))
}

func caseDesc() string {
	for _, c := range cases {
		if c.id == current {
			return c.desc
		}
	}
	return ""
}

func content() *ui.Node {
	switch current {
	case "dashboard":
		return ui.Scroll(dashboardCase()).St(ui.St().GrowTo(1)).K("scroll-dash")
	case "settings":
		return ui.Scroll(settingsCase()).St(ui.St().GrowTo(1)).K("scroll-set")
	case "chat":
		return ui.Column(chatCase()).St(ui.St().GrowTo(1).Hp(100))
	case "table":
		return ui.Scroll(tableCase()).St(ui.St().GrowTo(1)).K("scroll-tab")
	case "layouts":
		return ui.Scroll(layoutsCase()).St(ui.St().GrowTo(1)).K("scroll-layouts")
	case "controls":
		return ui.Scroll(controlsCase()).St(ui.St().GrowTo(1)).K("scroll-ctrl")
	case "form":
		return ui.Scroll(formCase()).St(ui.St().GrowTo(1)).K("scroll-form")
	case "system":
		return ui.Scroll(sysCase()).St(ui.St().GrowTo(1)).K("scroll-sys")
	case "music":
		return ui.Scroll(musicCase()).St(ui.St().GrowTo(1)).K("scroll-music")
	case "video":
		return ui.Scroll(videoCase()).St(ui.St().GrowTo(1)).K("scroll-video")
	case "markdown":
		return ui.Scroll(markdownCase()).St(ui.St().GrowTo(1)).K("scroll-md")
	case "webview":
		// 网页是独立子窗口，不能放进 Scroll（无法跟随滚动裁剪）
		return ui.Column(webviewCase()).St(ui.St().GrowTo(1).Hp(100))
	case "charts":
		return ui.Scroll(chartsCase()).St(ui.St().GrowTo(1)).K("scroll-charts")
	}
	return ui.Center(loginCase()).St(ui.St().GrowTo(1).Hp(100))
}

// ---- 案例一：登录页 ----
func loginCase() *ui.Node {
	hint := loginTip
	if hint == "" {
		hint = "演示：输入任意账号密码（英文/数字），点击登录"
	}
	return ui.Card(
		ui.Column(
			ui.Stack(
				ui.Box().St(ui.St().W(44).H(44).R(12).BgGrad("#6366f1", "#8b5cf6", 120)),
				ui.Text("G").St(ui.St().AbsFill().Text(20, 700, "#ffffff").CenterYOn().TAlign(ui.AlignCenter)),
			).St(ui.St().W(44).H(44)),
			ui.Text("欢迎回来").St(ui.St().Text(22, 700, ui.ColText).PadOnly(14, 0, 4, 0)),
			ui.Text("登录 GoGUI 控制台").St(ui.St().Text(12.5, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(0)),
		ui.Column(
			fieldLabel("账号"),
			ui.Input("login-user", userName, "请输入账号", func(v string) { userName = v }),
			fieldLabel("密码"),
			ui.Input("login-pass", password, "请输入密码", func(v string) { password = v }),
		).St(ui.St().GapOf(6)),
		ui.Row(
			ui.Checkbox("login-remember", "记住我", remember, func(v bool) { remember = v }),
			ui.Spacer(),
			ui.Text("忘记密码？").St(ui.St().Text(12, 500, "#8b93ff")),
		).AlignTo(ui.AlignCenter),
		ui.Button("登 录", func() {
			if userName == "" || password == "" {
				loginTip = "请先填写账号和密码"
			} else {
				loginTip = "登录成功：" + userName
				toastShow("欢迎，" + userName)
			}
			ui.RequestRedraw()
		}).St(ui.St().Wp(100).H(40).Text(14, 600, "#ffffff").R(10).
			BgGrad("#6366f1", "#8b5cf6", 90).Shadow(20, 8, 0, "rgba(99,102,241,0.42)")),
		ui.Text(hint).St(ui.St().Text(11.5, 400, ui.ColTextMuted).TAlign(ui.AlignCenter).WrapOn()),
		ui.Divider(),
		ui.Row(
			ui.Text("还没有账号？").St(ui.St().Text(12, 400, ui.ColTextMuted)),
			ui.Text("立即注册").St(ui.St().Text(12, 600, "#a5b4fc")),
		).St(ui.St().GapOf(6)).AlignTo(ui.AlignCenter),
	).St(ui.St().W(380).Padding(26).GapOf(18).R(20).BgC(ui.ColPanelDeep).
		Stroke(1, ui.ColStroke).Shadow(48, 20, 0, "rgba(0,0,0,0.55)"))
}

func fieldLabel(s string) *ui.Node {
	return ui.Text(s).St(ui.St().Text(12, 500, ui.ColTextDim).PadOnly(6, 0, 2, 0))
}

// ---- 案例二：仪表盘 ----
func dashboardCase() *ui.Node {
	return ui.Column(
		pageHead("仪表盘", "统计卡、进度条与活动列表的组合布局"),
		ui.Row(
			kpi("今日请求", "12 845", "+8.2%", "#22d3ee"),
			kpi("平均延迟", "38", "ms", "#a78bfa"),
			kpi("错误率", "0.24", "%", "#f472b6"),
			kpi("在线节点", "6", "/ 8", "#34d399"),
		).St(ui.St().GapOf(14)),
		ui.Row(
			ui.Card(
				ui.Title("资源占用", "滑块可调整演示值"),
				progressRow("CPU", volume, "#6366f1"),
				progressRow("内存", 0.48, "#22d3ee"),
				progressRow("磁盘", 0.72, "#34d399"),
			).St(ui.St().GrowTo(2).FlexBasis(2)),
			ui.Card(
				ui.Title("最近活动", "点击条目查看"),
				ui.ListItem("act-1", "部署成功", "prod · 2 分钟前", "#34d399", func() {
					toastShow("prod 部署成功")
					ui.RequestRedraw()
				}),
				ui.ListItem("act-2", "新增节点", "cn-east-3 · 14 分钟前", "#38bdf8", func() {
					toastShow("已新增节点 cn-east-3")
					ui.RequestRedraw()
				}),
				ui.ListItem("act-3", "告警恢复", "api-latency · 1 小时前", "#fbbf24", func() {
					toastShow("告警已恢复")
					ui.RequestRedraw()
				}),
			).St(ui.St().GrowTo(1).FlexBasis(1)),
		).St(ui.St().GapOf(16)).AlignTo(ui.AlignStart),
	).St(ui.St().Padding(26).GapOf(16))
}

func kpi(title, value, sub, accent string) *ui.Node {
	return ui.Column(
		ui.Text(title).St(ui.St().Text(11.5, 500, ui.ColTextMuted)),
		ui.Row(
			ui.Text(value).St(ui.St().Text(26, 700, "#f1f5ff")),
			ui.Text(sub).St(ui.St().Text(11.5, 600, accent).PadOnly(0, 0, 5, 6)),
		).AlignTo(ui.AlignEnd),
	).St(ui.St().GrowTo(1).FlexBasis(1).Padding(18).GapOf(4).R(14).
		BgC(ui.ColBg).Stroke(1, ui.ColStroke).Shadow(20, 8, 0, "rgba(0,0,0,0.32)"))
}

func progressRow(name string, v float32, color string) *ui.Node {
	return ui.Row(
		ui.Text(name).St(ui.St().Text(12, 500, ui.ColTextDim).W(48)),
		ui.Progress(v, color).Grow(1),
		ui.Text(strconv.Itoa(int(v*100))+"%").St(ui.St().Text(11.5, 500, "#a5b4fc").W(40)),
	).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter)
}

// ---- 案例三：设置页 ----
func settingsCase() *ui.Node {
	return ui.Column(
		pageHead("设置", "开关 / 单选 / 滑块 / 折叠面板 / 输入框"),
		ui.Card(
			ui.Title("通用", ""),
			ui.Row(
				ui.Column(
					ui.Text("自动保存").St(ui.St().Text(13, 500, ui.ColText)),
					ui.Text("编辑内容 2 秒后自动落盘").St(ui.St().Text(11.5, 400, ui.ColTextMuted).PadOnly(3, 0, 0, 0)),
				).Grow(1),
				ui.Switch("set-save", autoSave, func(v bool) { autoSave = v }),
			).AlignTo(ui.AlignCenter),
			ui.Row(
				ui.Column(
					ui.Text("云端同步").St(ui.St().Text(13, 500, ui.ColText)),
					ui.Text("多设备保持一致").St(ui.St().Text(11.5, 400, ui.ColTextMuted).PadOnly(3, 0, 0, 0)),
				).Grow(1),
				ui.Switch("set-sync", syncOn, func(v bool) { syncOn = v }),
			).AlignTo(ui.AlignCenter),
			ui.Divider(),
			ui.Text("主题").St(ui.St().Text(12.5, 500, ui.ColTextDim)),
			ui.Row(
				ui.Radio("th-dark", "深色", theme == "深色", func() { setTheme("深色") }),
				ui.Radio("th-light", "浅色", theme == "浅色", func() { setTheme("浅色") }),
				ui.Radio("th-auto", "跟随系统", theme == "跟随系统", func() { setTheme("跟随系统") }),
			).St(ui.St().GapOf(22)),
			ui.Divider(),
			ui.Row(
				ui.Text("界面缩放").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(70)),
				ui.Slider("set-scale", uiScaleRatio(), setUIScaleRatio),
				ui.Text(strconv.Itoa(int(ui.UIScale()*100+0.5))+"%").St(ui.St().Text(11.5, 500, "#a5b4fc").W(44)),
				ui.Btn("重置", ui.BtnGhost, func() {
					ui.SetUIScale(1)
					ui.RequestRedraw()
				}),
			).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
			ui.Divider(),
			ui.Row(
				ui.Text("清晰度").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(70)),
				ui.Radio("ss-auto", "自动", ui.SuperSampleMode() == ui.SSAuto, func() { setSuperSample(ui.SSAuto) }),
				ui.Radio("ss-off", "标准", ui.SuperSampleMode() == ui.SSOff, func() { setSuperSample(ui.SSOff) }),
				ui.Radio("ss-125", "1.25×", ui.SuperSampleMode() == ui.SS125, func() { setSuperSample(ui.SS125) }),
				ui.Radio("ss-150", "1.5×", ui.SuperSampleMode() == ui.SS150, func() { setSuperSample(ui.SS150) }),
				ui.Radio("ss-200", "2× 超高清", ui.SuperSampleMode() == ui.SS200, func() { setSuperSample(ui.SS200) }),
				ui.Text(ssLabel(ui.SuperSample())).St(ui.St().Text(11.5, 500, "#a5b4fc").W(96)),
			).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
			ui.Text("超高清 = 整帧按更高分辨率渲染再降采样，并对降采样后的边缘做轻度锐化补偿，圆角、描边与文字更锐（自动档在 DPI<150% 时走 2×）。").
				St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
			ui.Divider(),
			ui.Input("set-name", settingsText, "工作区名称", func(v string) { settingsText = v }),
			ui.Accordion("set-acc", "高级选项", "渲染后端 OpenGL 3.3，字形由 opentype 矢量覆盖率抗锯齿，DPI 采用 per-monitor v2 感知。", accOpen, func() { accOpen = !accOpen }),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

// ---- 案例四：聊天 ----
func chatCase() *ui.Node {
	items := make([]*ui.Node, 0, len(messages))
	for _, m := range messages {
		items = append(items, bubble(m))
	}
	items = append(items, ui.Spacer())

	return ui.Column(
		ui.Row(
			ui.Avatar("极", "#6366f1"),
			ui.Column(
				ui.Text("极码 AI").St(ui.St().Text(13.5, 600, ui.ColText)),
				ui.Row(ui.Dot("#34d399", 7), ui.Text("在线").St(ui.St().Text(11, 400, "#6ee7b7"))).St(ui.St().GapOf(6)),
			).Grow(1).St(ui.St().GapOf(3)),
			ui.Tag("示例会话", "#a5b4fc"),
		).St(ui.St().GapOf(12).Padding(16).BgC(ui.ColBgSoft).
			Stroke(1, ui.ColStroke)).AlignTo(ui.AlignCenter),

		ui.Scroll(append(items, ui.Box().St(ui.St().H(1)))...).St(ui.St().GrowTo(1).Padding(18)).K("scroll-chat"),

		ui.Row(
			ui.Box(
				ui.Input("chat-input", msgInput, "输入消息后回车无效，点发送…", func(v string) { msgInput = v }),
			).Grow(1),
			ui.Button("发送", func() {
				if msgInput == "" {
					return
				}
				messages = append(messages, chatMsg{true, msgInput})
				msgInput = ""
				toastShow("已发送")
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 600, "#ffffff").PadXY(18, 9).R(10).
				BgGrad("#6366f1", "#8b5cf6", 90)),
		).St(ui.St().GapOf(12).Padding(16)).AlignTo(ui.AlignCenter),
	).St(ui.St().Hp(100))
}

func bubble(m chatMsg) *ui.Node {
	st := ui.St().Text(13, 400, "#e6ecff").Padding(12).R(14).WrapOn()
	var row *ui.Node
	if m.me {
		st.BgGrad("#4f46e5", "#6366f1", 90)
		row = ui.Row(
			ui.Spacer(),
			ui.Text(m.text).St(st),
			ui.Avatar("我", "#0ea5e9"),
		)
	} else {
		st.BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke)
		row = ui.Row(
			ui.Avatar("极", "#6366f1"),
			ui.Text(m.text).St(st),
			ui.Spacer(),
		)
	}
	return row.St(ui.St().GapOf(10).PadOnly(0, 0, 10, 0)).AlignTo(ui.AlignStart)
}

// ---- 案例五：数据表格 ----
type row struct {
	name, status, owner string
	progress            float32
}

var rows = []row{
	{"渲染管线重构", "进行中", "李雷", 0.72},
	{"布局引擎优化", "已完成", "韩梅梅", 1},
	{"输入法支持", "待开始", "张三", 0.12},
	{"离屏毛玻璃", "进行中", "李雷", 0.45},
	{"虚拟列表", "已完成", "王五", 1},
}

func tableCase() *ui.Node {
	head := tableRow(
		ui.Text("任务").St(ui.St().Text(12, 600, ui.ColTextDim)).Grow(1),
		ui.Text("状态").St(ui.St().Text(12, 600, ui.ColTextDim).W(96)),
		ui.Text("进度").St(ui.St().Text(12, 600, ui.ColTextDim).W(180)),
		ui.Text("负责人").St(ui.St().Text(12, 600, ui.ColTextDim).W(90)),
	)
	body := make([]*ui.Node, 0, len(rows))
	for i, r := range rows {
		bg := "#0e1626"
		if i%2 == 1 {
			bg = ui.ColBgDeep
		}
		color := "#34d399"
		switch r.status {
		case "进行中":
			color = "#38bdf8"
		case "待开始":
			color = "#fbbf24"
		}
		name := r.name
		body = append(body, tableRow(
			ui.Text(name).St(ui.St().Text(12.5, 500, ui.ColText)).Grow(1),
			ui.Text(r.status).St(ui.St().Text(11, 600, color).PadXY(8, 3).R(7).
				BgC("rgba(255,255,255,0.05)").Stroke(1, "rgba(255,255,255,0.08)")).St(ui.St().W(96)),
			ui.Row(ui.Progress(r.progress, color).Grow(1)).St(ui.St().W(180)),
			ui.Text(r.owner).St(ui.St().Text(12, 400, "#98a3c0")).St(ui.St().W(90)),
		).St(ui.St().BgC(bg)).K("row-"+strconv.Itoa(i)).Tap(func() {
			toastShow("选中：" + name)
			ui.RequestRedraw()
		}))
	}

	return ui.Column(
		pageHead("数据表格", "斑马纹行 + 状态标签 + 行内进度，点击行可选中"),
		ui.Row(
			ui.Tag("全部 5", "#a5b4fc"),
			ui.Tag("进行中 2", "#38bdf8"),
			ui.Tag("已完成 2", "#34d399"),
			ui.Tag("待开始 1", "#fbbf24"),
		).St(ui.St().GapOf(8)),
		ui.Column(append([]*ui.Node{head}, body...)...).St(ui.St().R(14).Stroke(1, ui.ColStroke).Clipped()),
	).St(ui.St().Padding(26).GapOf(16))
}

func tableRow(cells ...*ui.Node) *ui.Node {
	return ui.Row(cells...).St(ui.St().GapOf(12).PadXY(16, 12).BgC(ui.ColBg)).AlignTo(ui.AlignCenter)
}

// ---- 案例：布局方式（布局实验室） ----

// layTile 等宽/弹性色块（高度固定，宽度由布局决定）。
func layTile(label, color string, h float32) *ui.Node {
	return ui.Box(
		ui.Text(label).St(ui.St().Text(12, 600, "#ffffff").TAlign(ui.AlignCenter).CenterYOn()),
	).St(ui.St().H(h).MinWidth(56).PadXY(10, 0).R(10).Clipped().
		BgGrad(color, "rgba(255,255,255,0.10)", 110))
}

// layJustify 主轴分布演示行。
func layJustify(label string, a ui.Align) *ui.Node {
	chips := make([]*ui.Node, 0, 3)
	for i := 0; i < 3; i++ {
		chips = append(chips, ui.Text(strconv.Itoa(i+1)).St(ui.St().W(30).H(24).
			Text(11.5, 600, ui.ColTextDim).TAlign(ui.AlignCenter).CenterYOn().R(7).
			BgC("rgba(255,255,255,0.06)").Stroke(1, ui.ColStroke)))
	}
	return ui.Row(
		ui.Text(label).St(ui.St().Text(11.5, 500, ui.ColTextMuted).W(112)),
		ui.Row(chips...).St(ui.St().GrowTo(1).PadXY(8, 6).R(8).
			BgC("rgba(255,255,255,0.03)").Stroke(1, ui.ColStroke)).JustifyTo(a),
	).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter)
}

// layCard 瀑布流卡片（高度不等）。
func layCard(title, body, color string, h float32) *ui.Node {
	return ui.Column(
		ui.Row(
			ui.Dot(color, 7),
			ui.Text(title).St(ui.St().Text(12.5, 600, ui.ColText)),
		).St(ui.St().GapOf(8)).AlignTo(ui.AlignCenter),
		ui.Text(body).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn().PadOnly(6, 0, 0, 0)),
	).St(ui.St().H(h).Padding(12).GapOf(6).R(12).BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke))
}

func layoutsCase() *ui.Node {
	code := func(s string) *ui.Node {
		return ui.Text(s).St(ui.St().Text(11, 500, "#8b93ff").PadXY(8, 4).R(6).
			BgC("rgba(99,102,241,0.12)").Stroke(1, "rgba(99,102,241,0.28)"))
	}

	// 响应式卡片：横向时按 Grow 等分，纵向时铺满整行
	rcards := func(row bool) []*ui.Node {
		items := []*ui.Node{
			layTile("卡片 A", "#6366f1", 64),
			layTile("卡片 B", "#22d3ee", 64),
			layTile("卡片 C", "#f472b6", 64),
		}
		if row {
			for _, it := range items {
				it.Grow(1)
			}
		}
		return items
	}

	wrapTags := []string{"弹性布局", "网格", "换行流式", "瀑布流", "主轴分布", "绝对定位", "叠层", "百分比", "等分栏", "响应式", "断点", "对齐"}
	tagNodes := make([]*ui.Node, 0, len(wrapTags))
	for _, t := range wrapTags {
		tagNodes = append(tagNodes, ui.Tag(t, "#a5b4fc"))
	}

	gridTiles := make([]*ui.Node, 0, 8)
	gridColors := []string{"#6366f1", "#22d3ee", "#f472b6", "#34d399", "#fbbf24", "#38bdf8", "#a78bfa", "#f87171"}
	for i := 0; i < 8; i++ {
		gridTiles = append(gridTiles, layTile("单元 "+strconv.Itoa(i+1), gridColors[i%len(gridColors)], 54))
	}

	masonryItems := []*ui.Node{
		layCard("弹性", "Grow / Basis 决定主轴尺寸分配", "#6366f1", 96),
		layCard("网格", "列宽等分，行高取该行最高子项", "#22d3ee", 128),
		layCard("换行", "放不下自动折行，行内可分布对齐", "#f472b6", 78),
		layCard("瀑布流", "每项落入当前最矮的列", "#34d399", 150),
		layCard("叠层", "Stack 子项重叠，后声明者在上", "#fbbf24", 86),
		layCard("响应式", "按窗口宽度切换列数与方向", "#38bdf8", 112),
	}

	return ui.Column(
		pageHead("布局方式", "8 种布局能力：Flex / 主轴分布 / 网格 / 换行 / 瀑布流 / 叠层 / 等分与百分比 / 响应式"),
		ui.Wrap(
			code("Row"), code("Column"), code("Grid"), code("Wrap"), code("Masonry"),
			code("Stack"), code("EqualRow"), code("JustifyTo"), code("ColsN"), code("ColWidth"), code("Pick"),
		).St(ui.St().GapOf(8)),

		ui.Card(
			ui.Title("① 弹性布局 Flex", "Row / Column + Grow 按比例分配剩余空间，定宽项不参与分配"),
			ui.Row(
				layTile("Grow 1", "#6366f1", 72).Grow(1),
				layTile("Grow 2", "#22d3ee", 72).Grow(2),
				layTile("Grow 3", "#f472b6", 72).Grow(3),
			).St(ui.St().GapOf(10)),
			ui.Row(
				layTile("定宽 120", "#0ea5e9", 34).St(ui.St().W(120)),
				ui.Spacer(),
				layTile("定宽 90", "#a78bfa", 34).St(ui.St().W(90)),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		),

		ui.Card(
			ui.Title("② 主轴分布 Justify", "同一排子项按剩余空间分布：start / center / end / between / around / evenly"),
			layJustify("space-between", ui.AlignSpaceBetween),
			layJustify("space-around", ui.AlignSpaceAround),
			layJustify("space-evenly", ui.AlignSpaceEvenly),
			layJustify("center", ui.AlignCenter),
			layJustify("end", ui.AlignEnd),
		),

		ui.Card(
			ui.Title("③ 网格 Grid", "ColsN 固定列数 / ColWidth 按最小列宽自适应列数（拖动窗口宽度会变化）"),
			ui.Text("ColsN(3)").St(ui.St().Text(11.5, 500, ui.ColTextMuted)),
			ui.Grid(gridTiles[:3]...).St(ui.St().ColsN(3).GapOf(10)),
			ui.Text("ColWidth(150)").St(ui.St().Text(11.5, 500, ui.ColTextMuted).PadOnly(6, 0, 0, 0)),
			ui.Grid(gridTiles...).St(ui.St().ColWidth(150).GapOf(10)),
		),

		ui.Card(
			ui.Title("④ 换行流式 Wrap", "一行放不下自动折行（flex-wrap: wrap），行内支持主轴分布"),
			ui.Wrap(tagNodes...).St(ui.St().GapOf(8)),
			ui.Wrap(tagNodes[:6]...).St(ui.St().GapOf(8)).JustifyTo(ui.AlignSpaceBetween),
		),

		ui.Card(
			ui.Title("⑤ 瀑布流 Masonry", "列宽等分，子项按当前最矮列落位，高度可不等"),
			ui.Masonry(masonryItems...).St(ui.St().ColsN(3).GapOf(10)),
		),

		ui.Card(
			ui.Title("⑥ 叠层与绝对定位", "Stack 子项重叠绘制；AbsTL / AbsTR / AbsCenter / AbsFill 相对父容器定位"),
			ui.Stack(
				ui.Box().St(ui.St().AbsFill().R(12).BgGrad(ui.ColBgSoft, ui.ColBgDeep, 120)),
				ui.Text("Stack 底层：渐变背景铺满").St(ui.St().AbsCenter().Text(12.5, 500, "#c7d2fe")),
				ui.Tag("AbsTL", "#34d399").St(ui.St().AbsTL(10, 10)),
				ui.Tag("AbsTR", "#38bdf8").St(ui.St().AbsTR(10, 10)),
				ui.Tag("AbsBL", "#fbbf24").St(ui.St().AbsBL(10, 10)),
				ui.Tag("AbsBR", "#f472b6").St(ui.St().AbsBR(10, 10)),
			).St(ui.St().H(132).Clipped()),
		),

		ui.Card(
			ui.Title("⑦ 等分栏与百分比", "EqualRow 横向等分；Wp / Hp 按父容器百分比定尺寸"),
			ui.EqualRow(
				layTile("25%", "#6366f1", 46),
				layTile("25%", "#22d3ee", 46),
				layTile("25%", "#f472b6", 46),
				layTile("25%", "#34d399", 46),
			).St(ui.St().GapOf(10)),
			ui.Row(
				layTile("Wp 30", "#a78bfa", 40).St(ui.St().Wp(30)),
				layTile("Wp 70", "#38bdf8", 40).St(ui.St().Wp(70)),
			).St(ui.St().GapOf(0)),
		),

		ui.Card(
			ui.Title("⑧ 响应式断点", "ui.Pick / ui.BP 按视口宽度切换布局（<640 纵向，≥640 横向三列）"),
			ui.Text(fmt.Sprintf("视口 %.0f × %.0f · 自适应列数 %d · 拖动窗口边缘观察变化",
				ui.ViewportW(), ui.ViewportH(), ui.BP(1, 2, 3, 4))).
				St(ui.St().Text(11.5, 500, ui.ColTextMuted)),
			ui.Pick(
				func() *ui.Node { return ui.Column(rcards(false)...).St(ui.St().GapOf(10)) },
				func() *ui.Node { return ui.Row(rcards(true)...).St(ui.St().GapOf(10)) },
				func() *ui.Node { return ui.Row(rcards(true)...).St(ui.St().GapOf(10)) },
				func() *ui.Node {
					return ui.Row(rcards(true)...).St(ui.St().GapOf(10))
				},
			),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

// ---- 案例：图表（折线 / 面积 / 柱状 / 饼 / 环形 / 雷达 / 散点）----
var (
	chartOnce    bool
	chartPhase   float64
	chartPicked  = "悬浮查看数值，点击柱 / 扇区 / 数据点"
	chartMonths  = []string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月"}
	chartWeeks   = []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	chartLine    *ui.Chart
	chartArea    *ui.Chart
	chartBar     *ui.Chart
	chartPie     *ui.Chart
	chartDonut   *ui.Chart
	chartRadar   *ui.Chart
	chartScatter *ui.Chart
)

// wave 造一段演示数据（相位变化时曲线形状跟着变）。
func wave(base, amp float64, n int, ph float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		v := base + amp*math.Sin(float64(i)*1.1+ph) + amp*0.35*math.Sin(float64(i)*2.7+ph*2)
		if v < 1 {
			v = 1
		}
		out[i] = math.Round(v)
	}
	return out
}

// ensureCharts 只建一次图表对象（数据与缓存跟着 Chart 走，不要每帧新建）。
func ensureCharts() {
	if chartOnce {
		return
	}
	chartOnce = true
	chartLine = ui.NewChart(ui.ChartLine, chartMonths)
	chartLine.Smooth = true
	chartLine.Unit = " 次"
	chartLine.Title = "访问趋势"
	chartLine.OnPick = func(s, i int) { pickChart("折线图", chartLine, s, i) }

	chartArea = ui.NewChart(ui.ChartArea, chartMonths)
	chartArea.Stacked = true
	chartArea.Unit = " 台"
	chartArea.Title = "终端分布"

	chartBar = ui.NewChart(ui.ChartBar, chartWeeks)
	chartBar.Unit = " 单"
	chartBar.Title = "每周下单"
	chartBar.OnPick = func(s, i int) { pickChart("柱状图", chartBar, s, i) }

	chartPie = ui.NewChart(ui.ChartPie, []string{"华东", "华南", "华北", "西部", "海外"})
	chartPie.Title = "区域占比"
	chartPie.OnPick = func(s, i int) { pickChart("饼图", chartPie, s, i) }

	chartDonut = ui.NewChart(ui.ChartDonut, []string{"直连", "代理", "缓存", "其他"})
	chartDonut.Title = "流量来源"
	chartDonut.OnPick = func(s, i int) { pickChart("环形图", chartDonut, s, i) }

	chartRadar = ui.NewChart(ui.ChartRadar, []string{"性能", "稳定", "安全", "成本", "体验"})
	chartRadar.Title = "综合评分"

	chartScatter = ui.NewChart(ui.ChartScatter, nil)
	chartScatter.Title = "延迟分布"
	chartScatter.Unit = " ms"

	applyChartData()
}

// applyChartData 重新生成所有图表的数据（指纹变化 → 纹理重建一次）。
func applyChartData() {
	n := len(chartMonths)
	chartLine.Series = []ui.Series{
		{Name: "访问量", Values: wave(1400, 620, n, chartPhase)},
		{Name: "注册量", Values: wave(760, 320, n, chartPhase+1.4)},
	}
	chartArea.Series = []ui.Series{
		{Name: "桌面端", Values: wave(520, 180, n, chartPhase+0.4)},
		{Name: "移动端", Values: wave(880, 260, n, chartPhase+1.1)},
	}
	chartBar.Series = []ui.Series{
		{Name: "新增", Color: "#6366f1", Values: wave(340, 150, len(chartWeeks), chartPhase+0.2)},
		{Name: "续费", Color: "#22d3ee", Values: wave(220, 110, len(chartWeeks), chartPhase+1.8)},
		{Name: "退款", Color: "#f472b6", Values: wave(60, 40, len(chartWeeks), chartPhase+2.6)},
	}
	pieVals := wave(38, 22, 5, chartPhase)
	for i := range pieVals {
		if pieVals[i] < 4 {
			pieVals[i] = 4
		}
	}
	chartPie.Series = ui.PieSeries(pieVals, chartPie.Labels)
	donutVals := wave(30, 18, 4, chartPhase+0.9)
	for i := range donutVals {
		if donutVals[i] < 4 {
			donutVals[i] = 4
		}
	}
	chartDonut.Series = ui.PieSeries(donutVals, chartDonut.Labels)
	chartRadar.Series = []ui.Series{
		{Name: "当前版本", Values: []float64{88, 92, 78, 70, 86}},
		{Name: "上个版本", Values: []float64{72, 80, 74, 64, 78}},
	}
	chartScatter.Series = []ui.Series{
		{Name: "集群 A", Color: "#34d399", Values: wave(42, 26, 26, chartPhase)},
		{Name: "集群 B", Color: "#a78bfa", Values: wave(64, 30, 26, chartPhase+2.2)},
	}
}

// pickChart 点击回调：只改文字状态，不动图表缓存。
func pickChart(kind string, c *ui.Chart, s, i int) {
	if s < 0 || s >= len(c.Series) {
		return
	}
	ser := c.Series[s]
	// 饼图 / 环形图：每个系列就是一个扇区，没有「X 轴标签」的概念，
	// 直接输出扇区名 + 自身数值（否则会拼上第一个标签，如「华东 · 华南」）。
	if c.Type == ui.ChartPie || c.Type == ui.ChartDonut {
		sum := 0.0
		for _, x := range ser.Values {
			sum += x
		}
		chartPicked = fmt.Sprintf("选中 %s · %s = %v%s", kind, ser.Name, sum, c.Unit)
		ui.RequestRedraw()
		return
	}
	name, label, v := kind, "", 0.0
	name = kind + " · " + ser.Name
	if i >= 0 && i < len(ser.Values) {
		v = ser.Values[i]
	}
	if i >= 0 && i < len(c.Labels) {
		label = c.Labels[i]
	}
	chartPicked = fmt.Sprintf("选中 %s：%s = %v%s", name, label, v, c.Unit)
	ui.RequestRedraw()
}

// chartCard 图表卡片（固定高度，宽度均分）。
func chartCard(title, sub string, chart *ui.Chart, h float32) *ui.Node {
	return ui.Card(
		ui.Title(title, sub),
		ui.ChartView("chart-"+title, chart).St(ui.St().GrowTo(1).H(h)),
	).St(ui.St().GrowTo(1).FlexBasis(1).W(520).Padding(16).GapOf(10).R(14).
		BgC(ui.ColBg).Stroke(1, ui.ColStroke))
}

// chartsCase 图表展示页。
func chartsCase() *ui.Node {
	ensureCharts()
	return ui.Column(
		pageHead("图表组件", "纯自绘 / 纹理缓存 / 悬浮提示 / 跟随主题，数据不变时每帧只发 1~2 个 draw call"),
		ui.Row(
			chartCard("折线图", "Smooth 平滑 + 数据点", chartLine, 200),
			chartCard("面积图", "堆叠 + 半透明填充", chartArea, 200),
		).St(ui.St().GapOf(16)),
		ui.Row(
			chartCard("柱状图", "分组柱 / 悬浮高亮 / 点击回传", chartBar, 190),
			chartCard("饼图", "扇区间隙 + 悬浮提示", chartPie, 190),
		).St(ui.St().GapOf(16)),
		ui.Row(
			chartCard("环形图", "内径 58%", chartDonut, 190),
			chartCard("雷达图", "多系列多边形叠加", chartRadar, 190),
		).St(ui.St().GapOf(16)),
		ui.Row(
			chartCard("散点图", "数值点密度展示", chartScatter, 190),
			ui.Card(
				ui.Title("实现与性能", "为什么图表不卡也不吃内存"),
				ui.Text("· 网格 / 柱 / 文字全部用原生顶点绘制，零纹理、永远清晰").St(ui.St().Text(11.5, 400, ui.ColTextDim)),
				ui.Text("· 只有弧线 / 斜线 / 多边形（折线、饼、雷达、散点）光栅成一张 RGBA 纹理").St(ui.St().Text(11.5, 400, ui.ColTextDim)),
				ui.Text("· 纹理按「尺寸 + 缩放 + 主题 + 数据指纹」缓存，不变则每帧 0 CPU 光栅化").St(ui.St().Text(11.5, 400, ui.ColTextDim)),
				ui.Text("· 总预算 24MB，超出按 LRU 淘汰，窗口销毁随 GL 上下文释放").St(ui.St().Text(11.5, 400, ui.ColTextDim)),
				ui.Text("· 悬浮高亮与提示框全用原生图元叠加，鼠标移动不重建纹理").St(ui.St().Text(11.5, 400, ui.ColTextDim)),
			).St(ui.St().GrowTo(1).FlexBasis(1).W(520).Padding(16).GapOf(8).R(14).
				BgC(ui.ColBg).Stroke(1, ui.ColStroke)),
		).St(ui.St().GapOf(16)),
		ui.Row(
			ui.Text(chartPicked).St(ui.St().Text(12, 500, ui.ColTextDim).GrowTo(1)),
			ui.Btn("刷新数据", ui.BtnPrimary, func() {
				chartPhase += 0.57
				applyChartData()
				chartPicked = "数据已更新（缓存失效并重建一次纹理）"
				ui.RequestRedraw()
			}),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
	).St(ui.St().Padding(26).GapOf(16))
}

// ---- 案例六：控件总览 ----

// act 生成一个"写提示 + 重绘"的回调。
func act(msg string) func() {
	return func() {
		toastShow(msg)
		ui.RequestRedraw()
	}
}

func cell(s string) *ui.Node {
	return ui.Text(s).St(ui.St().Text(12.5, 500, ui.ColText))
}

func controlsCase() *ui.Node {
	return ui.Column(
		pageHead("控件总览", "参照 gio widget 体系：按钮 / 输入 / 滑块 / 表格 / 菜单 / 提示"),

		ui.If(ctrlBanner, func() *ui.Node {
			return ui.Banner("info", "组件库更新", "新增 Table、Menu、Banner、Breadcrumb、Ring、Tooltip 与按钮变体；输入框支持光标闪烁与方向键编辑。", func() {
				ctrlBanner = false
				ui.RequestRedraw()
			})
		}),

		ui.Card(
			ui.Title("按钮", "五种变体 / 图标按钮 / 禁用态"),
			ui.Row(
				ui.Btn("主要", ui.BtnPrimary, act("主要按钮")),
				ui.Btn("次要", ui.BtnSecondary, act("次要按钮")),
				ui.Btn("幽灵", ui.BtnGhost, act("幽灵按钮")),
				ui.Btn("危险", ui.BtnDanger, act("危险操作")),
				ui.Btn("成功", ui.BtnSuccess, act("操作成功")),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
			ui.Row(
				ui.TooltipOf(ui.IconButton("+", act("新建")), "新建"),
				ui.TooltipOf(ui.IconButton("×", act("关闭")), "关闭"),
				ui.TooltipOf(ui.IconButton("→", act("下一步")), "下一步"),
				ui.Btn("禁用", ui.BtnSecondary, nil).DisabledOn(true).St(ui.St().Fade(0.5)),
				ui.Text(ctrlTip).St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		),

		ui.Card(
			ui.Title("输入与滑块", "输入框：光标闪烁 / 左右键 / Home-End / 退格；滑块可拖动"),
			ui.Row(
				ui.Box(ui.Input("ctrl-name", settingsText, "名称", func(v string) { settingsText = v })).Grow(1),
				ui.Box(ui.PasswordInput("ctrl-pwd", ctrlPwd, "密码", func(v string) { ctrlPwd = v })).Grow(1),
			).St(ui.St().GapOf(12)),
			ui.Row(
				ui.Text("数值").St(ui.St().Text(12, 500, ui.ColTextMuted).W(44)),
				ui.Slider("ctrl-slider", ctrlSlider, func(v float32) { ctrlSlider = v }),
				ui.Text(strconv.Itoa(int(ctrlSlider*100))).St(ui.St().Text(12, 500, "#a5b4fc").W(34)),
				ui.Progress(ctrlSlider, ui.ColPrimary).Grow(1),
			).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
			ui.Row(
				ui.Ring(ctrlSlider, 46, ui.ColInfo),
				ui.Ring(0.8, 32, ui.ColSuccess),
				ui.Spinner(20, ui.ColPrimary),
				ui.Column(
					ui.KeyValue("渲染后端", "OpenGL 3.3"),
					ui.KeyValue("字体光栅化", "opentype 覆盖率 AA"),
				).Grow(1),
			).St(ui.St().GapOf(18)).AlignTo(ui.AlignCenter),
		),

		ui.Card(
			ui.Title("导航", "TabsBar / Breadcrumb / 浮层菜单"),
			ui.TabsBar([]string{"ui", "api", "doc"}, []string{"界面", "接口", "文档"}, ctrlTab, func(k string) {
				ctrlTab = k
			}),
			ui.Breadcrumb([]string{"首页", "组件", "导航"}, func(i int) {
				toastShow("面包屑第 " + strconv.Itoa(i+1) + " 级")
				ui.RequestRedraw()
			}),
			ui.Row(
				ui.Btn("打开菜单", ui.BtnSecondary, func() {
					ctrlMenu = !ctrlMenu
					ui.RequestRedraw()
				}),
				ui.Text("菜单为 Stack + 绝对定位浮层，点击项后关闭").St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
			).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		),

		ui.Card(
			ui.Title("表格与空状态", "固定/弹性列宽、斑马纹、整行点击"),
			ui.Table([]string{"任务", "状态", "进度", "负责人"}, []float32{0, 92, 130, 84},
				[][]*ui.Node{
					{cell("渲染管线重构"), ui.Tag("进行中", "#38bdf8"), ui.Ring(0.72, 20, ui.ColInfo), cell("李雷")},
					{cell("布局引擎优化"), ui.Tag("已完成", "#34d399"), ui.Ring(1, 20, ui.ColSuccess), cell("韩梅梅")},
					{cell("输入法支持"), ui.Tag("待开始", "#fbbf24"), ui.Ring(0.15, 20, ui.ColWarn), cell("张三")},
				}, func(i int) {
					ctrlRow = i
					toastShow("选中第 " + strconv.Itoa(i+1) + " 行")
					ui.RequestRedraw()
				}),
			ui.If(ctrlRow >= 0, func() *ui.Node {
				return ui.KeyValue("当前选中", strconv.Itoa(ctrlRow+1)+" 行")
			}),
			ui.Accordion("ctrl-acc", "空状态示例（展开）", "列表为空时展示 Empty 组件。", ctrlAccOpen, func() {
				ctrlAccOpen = !ctrlAccOpen
			}),
			ui.If(ctrlAccOpen, func() *ui.Node {
				return ui.Empty("暂无数据", "调整筛选条件后重试")
			}),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

func pageHead(title, sub string) *ui.Node {
	return ui.Column(
		ui.Text(title).St(ui.St().Text(22, 700, ui.ColText)),
		ui.Text(sub).St(ui.St().Text(12, 400, ui.ColTextMuted).PadOnly(4, 0, 0, 0)),
	).St(ui.St().PadOnly(0, 0, 6, 0))
}

// ---- 系统能力演示：图片 / 毛玻璃 / Emoji / 动画 / 滚动条 / IME / 多窗口 ----

var (
	sysAnimOn    bool
	sysImgPath   string
	sysOpened    int
	sysTray      *ui.Tray
	sysMsg       = "点击下方按钮体验文件对话框 / 多窗口 / 托盘"
	sysScrollRow = -1
	sysIme       = ""

	// 拖入文件 / 系统操作演示状态
	sysDropPaths  []string
	sysDropMsg    = "把文件或文件夹拖到上面的框里松手，这里会显示结果"
	sysClipText   = ""
	sysTopMost    bool
	sysHotkeyID   int32
	sysWakeLock   bool
	sysHotkeyNote = "未注册"
)

// sysSampleImage 找一个系统自带壁纸作为图片演示素材。
func sysSampleImage() string {
	if sysImgPath != "" {
		return sysImgPath
	}
	cands := []string{
		`C:\Windows\Web\Wallpaper\Windows\img0.jpg`,
		`C:\Windows\Web\Wallpaper\Windows\img19.jpg`,
		`C:\Windows\Web\Wallpaper\Theme1\img1.jpg`,
		`C:\Windows\Web\Wallpaper\ThemeA\img1.jpg`,
		`C:\Windows\Web\4K\Wallpaper\Windows\img0_3840x2160.jpg`,
	}
	for _, p := range cands {
		if w, _ := ui.ImageSize(p); w > 0 {
			return p
		}
	}
	return ""
}

func sysSection(title, sub string, body *ui.Node) *ui.Node {
	return ui.Column(
		ui.Title(title, sub),
		body,
	).St(ui.St().GapOf(10).PadOnly(0, 0, 10, 0))
}

// sysPickImage 从拖入的路径里挑一张能解码的图片（自动填到图片预览区）。
func sysPickImage(paths []string) string {
	for _, p := range paths {
		switch strings.ToLower(filepath.Ext(p)) {
		case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp":
			if w, _ := ui.ImageSize(p); w > 0 {
				return p
			}
		}
	}
	return ""
}

// sysDropAccept 统一的拖入处理（节点级 DropZone 与窗口级兜底共用）。
func sysDropAccept(paths []string) {
	if len(paths) == 0 {
		return
	}
	sysDropPaths = paths
	if p := sysPickImage(paths); p != "" {
		sysImgPath = p
		sysDropMsg = fmt.Sprintf("收到 %d 项，已把图片送入上方预览：%s", len(paths), filepath.Base(p))
	} else if len(paths) == 1 {
		sysDropMsg = "收到 1 项：" + paths[0]
	} else {
		sysDropMsg = fmt.Sprintf("收到 %d 项：%s …", len(paths), filepath.Base(paths[0]))
	}
	ui.RequestRedraw()
}

func sysCase() *ui.Node {
	// 动画：Tween / Spring / 关键帧
	t := float32(0)
	if sysAnimOn {
		t = 1
	}
	ease := ui.Tween("sys-ease", t, 0.5, ui.EaseOutCubic)
	back := ui.Tween("sys-back", t, 0.6, ui.EaseOutBack)
	sp := ui.Spring("sys-spring", t, 210, 15)
	kf := ui.Keyframes("sys-kf", []ui.Key{
		{At: 0, Value: 0},
		{At: 0.5, Value: 1},
		{At: 1, Value: 0},
	}, 2.4, true)

	// 图片区
	imgBody := ui.Column()
	if p := sysSampleImage(); p != "" {
		w, h := ui.ImageSize(p)
		scale := float32(280) / float32(w)
		imgBody.Add(
			ui.Row(
				ui.ImageSized(p, 280, float32(h)*scale).St(ui.St().R(14).Clipped()),
				ui.Column(
					ui.KeyValue("文件", p),
					ui.KeyValue("尺寸", fmt.Sprintf("%d × %d px", w, h)),
					ui.Text("支持 PNG / JPEG / GIF / BMP，纹理上传后按圆角裁剪绘制。").St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
				).St(ui.St().GrowTo(1).GapOf(8)),
			).St(ui.St().GapOf(16)),
		)
	} else {
		imgBody.Add(ui.Empty("未找到示例图片", "点击下方「选择图片」载入本地图片"))
	}
	imgBody.Add(ui.Row(
		ui.Btn("选择图片", ui.BtnPrimary, func() {
			if p, ok := ui.OpenFileDialog("选择图片", "图片|*.png;*.jpg;*.jpeg;*.gif;*.bmp|所有文件|*.*"); ok {
				sysImgPath = p
				sysMsg = "已载入：" + p
			}
		}),
		ui.Btn("打开文件夹", ui.BtnSecondary, func() {
			if p, ok := ui.PickFolderDialog("选择目录"); ok {
				sysMsg = "已选择目录：" + p
			}
		}),
		ui.Btn("另存为", ui.BtnSecondary, func() {
			if p, ok := ui.SaveFileDialog("保存文件", "文本|*.txt|所有文件|*.*", "txt"); ok {
				sysMsg = "另存为：" + p
			}
		}),
	).St(ui.St().GapOf(10)))

	// 毛玻璃
	glass := ui.Stack(
		ui.Box().St(ui.St().AbsFill().BgGrad("#312e81", "#0891b2", 45).R(18)),
		ui.Box().St(ui.St().AbsTL(30, 24).W(130).H(130).R(65).BgC("#a78bfa")),
		ui.Box().St(ui.St().AbsTL(210, 78).W(160).H(160).R(80).BgC("#f472b6")),
		ui.Box().St(ui.St().AbsTL(420, 20).W(100).H(100).R(50).BgC("#fbbf24")),
		ui.Box().St(ui.St().AbsTL(560, 60).W(120).H(120).R(60).BgC("#34d399")),
		ui.Column(
			ui.Text("毛玻璃 backdrop-filter").St(ui.St().Text(16, 700, "#ffffff")),
			ui.Text("离屏 FBO 捕获背景 → 双 Kawase 降采样模糊 → 圆角回绘 + 色调叠加。").St(ui.St().Text(12, 400, "#dbeafe").WrapOn()),
		).St(ui.St().AbsTL(70, 60).W(340).Padding(18).R(16).GapOf(8).
			Backdrop(22, ui.ColGlass).Stroke(1, "rgba(255,255,255,0.28)"))).
		St(ui.St().H(230).R(18).Clipped())

	// 动画
	anim := ui.Column(
		ui.Row(
			ui.Text("Tween").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Box().St(ui.St().W(80+240*ease).H(14).R(7).BgGrad(ui.ColPrimary, ui.ColPrimary2, 90)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("Spring").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Stack(
				ui.Box().St(ui.St().Wp(100).H(14).R(7).BgC("rgba(255,255,255,0.06)")),
				ui.Box().St(ui.St().W(22).H(14).R(7).BgC(ui.ColSuccess).AbsTL(240*sp, 0)),
			).St(ui.St().W(280).H(14)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("Back").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Box().St(ui.St().W(26).H(26).R(13).BgC(ui.ColPrimary2).MarOnly(0, 0, 0, 240*back)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("Keyframes").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Stack(
				ui.Box().St(ui.St().Wp(100).H(32).R(8).BgC("rgba(255,255,255,0.04)")),
				ui.Box().St(ui.St().W(8).H(8).R(4).BgC(ui.ColWarn).AbsTL(8+260*kf, 12)),
			).St(ui.St().W(280).H(32)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Btn("播放 / 复位", ui.BtnSecondary, func() { sysAnimOn = !sysAnimOn }),
			ui.Text("Tween 缓动 + Spring 弹簧积分 + Keyframes 循环").St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(10)),
	).St(ui.St().GapOf(10))

	// 字体与 Emoji
	fontRow := ui.Column(
		ui.Text("😀 🚀 🔥 🎉 🌈 🐳 ✅ 🎯").St(ui.St().Text(26, 400, "#ffffff")),
		ui.Text("彩色 Emoji 走 GDI 彩色位图光栅化，普通字行走 opentype 覆盖率抗锯齿。").St(ui.St().Text(12, 400, ui.ColTextDim)),
		ui.Row(
			ui.Text("微软雅黑 UI").St(ui.St().Text(14, 400, "#cbd5e1")),
			ui.Text("Segoe UI").St(ui.St().Text(14, 400, "#cbd5e1").Font("Segoe UI")),
			ui.Text("Consolas 0123").St(ui.St().Text(14, 400, "#cbd5e1").Font("Consolas")),
			ui.Text("宋体 SimSun").St(ui.St().Text(14, 400, "#cbd5e1").Font("SimSun")),
		).St(ui.St().GapOf(18)),
	).St(ui.St().GapOf(10))

	// IME 输入
	ime := ui.Column(
		ui.Row(
			ui.Text("姓名").St(ui.St().Text(12, 500, ui.ColTextDim).W(60)),
			ui.Input("sys-ime", sysIme, "点击后用中文输入法输入…", func(v string) {
				sysIme = v
				sysMsg = "输入内容：" + v
			}),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Text("组合串会以高亮块显示在光标处；上屏由 WM_IME_COMPOSITION(GCS_RESULTSTR) 派发。").St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	)

	// 滚动条
	rows := make([]*ui.Node, 0, 16)
	for i := 0; i < 16; i++ {
		idx := i
		color := ui.ColTextMuted
		if sysScrollRow == i {
			color = ui.ColPrimary
		}
		rows = append(rows, ui.Row(
			ui.Box().St(ui.St().W(6).H(6).R(3).BgC(color)),
			ui.Text("列表项 "+strconv.Itoa(i+1)).St(ui.St().Text(12.5, 500, color)),
			ui.Spacer(),
			ui.Text("拖拽右侧滚动条或滚轮").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(10).PadXY(12, 8).R(8).BgC("rgba(255,255,255,0.02)")).
			Tap(func() { sysScrollRow = idx; ui.RequestRedraw() }))
	}
	scrollDemo := ui.Scroll(ui.Column(rows...).St(ui.St().GapOf(6))).
		St(ui.St().H(190).R(10).Padding(8).BgC("rgba(255,255,255,0.02)").Stroke(1, ui.ColStroke)).
		K("sys-scroll")

	// 多窗口 / 托盘
	platform := ui.Column(
		ui.Row(
			ui.Btn("打开子窗口", ui.BtnPrimary, func() {
				sysOpened++
				title := fmt.Sprintf("子窗口 #%d", sysOpened)
				if _, err := ui.NewWindow(ui.Options{Title: title, Width: 520, Height: 360}, sysChildView); err != nil {
					sysMsg = "打开窗口失败：" + err.Error()
				} else {
					sysMsg = "已打开 " + title
				}
			}),
			ui.Btn("创建托盘图标", ui.BtnSecondary, func() {
				if sysTray != nil {
					sysMsg = "托盘图标已存在"
					return
				}
				owner := ui.Current()
				t, err := ui.NewTray("GoGUI 系统能力演示", func() {
					if owner != nil {
						owner.ShowWindow(true)
						owner.Restore()
					}
				}, []ui.TrayItem{
					{ID: 1, Label: "显示主窗口"},
					{ID: 2, Label: "换成托盘提示", Checked: true},
					{Separator: true},
					{ID: 9, Label: "退出"},
				}, func(id int) {
					switch id {
					case 1:
						if owner != nil {
							owner.ShowWindow(true)
							owner.Restore()
						}
					case 2:
						if sysTray != nil {
							sysTray.SetTooltip("GoGUI · " + strconv.Itoa(sysOpened) + " 个窗口")
						}
					case 9:
						if owner != nil {
							owner.Close()
						}
					}
				})
				if err != nil {
					sysMsg = "创建托盘失败"
					return
				}
				sysTray = t
				sysMsg = "已创建托盘图标（右键看菜单）"
			}),
			ui.Btn("移除托盘", ui.BtnGhost, func() {
				if sysTray != nil {
					sysTray.Remove()
					sysTray = nil
					sysMsg = "托盘图标已移除"
				}
			}),
		).St(ui.St().GapOf(10)),
		ui.Text(sysMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	).St(ui.St().GapOf(10))

	// ---- 拖入文件：节点级 DropZone + 窗口级兜底回调 ----
	if a := ui.Current(); a != nil {
		a.SetOnDropFiles(func(paths []string, x, y float32) {
			sysDropMsg = fmt.Sprintf("窗口级回调：%d 项，落点 %.0f, %.0f", len(paths), x, y)
			sysDropPaths = paths
			if p := sysPickImage(paths); p != "" {
				sysImgPath = p
			}
			ui.RequestRedraw()
		})
	}
	dropBody := ui.Column(
		ui.DropZone("sys-drop", "把文件 / 文件夹拖到这里松手", 138, sysDropAccept),
		ui.Text(sysDropMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	)
	if len(sysDropPaths) > 0 {
		dropBody.Add(ui.Scroll(
			ui.Column(ui.Each(sysDropPaths, func(p string, _ int) *ui.Node {
				return ui.FileRow(p, nil)
			})...).St(ui.St().GapOf(4)),
		).St(ui.St().H(168).R(10).Padding(6).BgC("rgba(255,255,255,0.02)").Stroke(1, ui.ColStroke)).
			K("sys-drop-list"))
	}
	dropBody.Add(ui.Wrap(
		ui.Btn("选择文件（多选）", ui.BtnSecondary, func() {
			if ps, ok := ui.PickFiles("选择文件（可多选）", "所有文件|*.*"); ok {
				sysDropAccept(ps)
			}
		}),
		ui.Btn("复制路径", ui.BtnGhost, func() {
			if len(sysDropPaths) == 0 {
				sysDropMsg = "还没有拖入文件"
			} else if ui.ClipboardSetFiles(sysDropPaths) {
				sysDropMsg = fmt.Sprintf("已复制 %d 个路径到剪贴板（可粘贴到资源管理器）", len(sysDropPaths))
			} else {
				sysDropMsg = "复制到剪贴板失败"
			}
			ui.RequestRedraw()
		}),
		ui.Btn("定位文件", ui.BtnGhost, func() {
			if len(sysDropPaths) == 0 {
				sysDropMsg = "还没有拖入文件"
			} else if ui.RevealInExplorer(sysDropPaths[0]) {
				sysDropMsg = "已在资源管理器中定位：" + sysDropPaths[0]
			}
			ui.RequestRedraw()
		}),
		ui.Btn("移入回收站", ui.BtnDanger, func() {
			if len(sysDropPaths) == 0 {
				sysDropMsg = "还没有拖入文件"
				return
			}
			n := 0
			for _, p := range sysDropPaths {
				if ui.MoveToRecycleBin(p) {
					n++
				}
			}
			sysDropMsg = fmt.Sprintf("已移入回收站 %d / %d 项（可从回收站还原）", n, len(sysDropPaths))
			sysDropPaths = nil
			ui.RequestRedraw()
		}),
		ui.Btn("清空列表", ui.BtnGhost, func() {
			sysDropPaths = nil
			sysDropMsg = "已清空拖入列表"
			ui.RequestRedraw()
		}),
	).St(ui.St().GapOf(8)))

	// ---- 系统操作能力 ----
	opsBody := ui.Column(
		ui.Row(
			ui.Text("窗口置顶").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Switch("sys-top", sysTopMost, func(v bool) {
				sysTopMost = v
				ui.SetAlwaysOnTop(v)
				if v {
					sysMsg = "已置顶：切到其它窗口时本窗口仍在最上层"
				} else {
					sysMsg = "已取消置顶"
				}
				ui.RequestRedraw()
			}),
			ui.Text("HWND_TOPMOST / HWND_NOTOPMOST").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("全局热键").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Btn("注册 Ctrl+Alt+G", ui.BtnSecondary, func() {
				if sysHotkeyID != 0 {
					sysHotkeyNote = "已注册，先注销"
					ui.RequestRedraw()
					return
				}
				id := ui.RegisterHotkey(ui.HotCtrl|ui.HotAlt, 'G', func() {
					sysHotkeyNote = "热键已触发（窗口失焦也有效）"
					ui.RequestRedraw()
				})
				sysHotkeyID = id
				if id == 0 {
					sysHotkeyNote = "注册失败：组合可能被其它程序占用"
				} else {
					sysHotkeyNote = fmt.Sprintf("已注册 id=%d", id)
				}
				ui.RequestRedraw()
			}),
			ui.Btn("注销", ui.BtnGhost, func() {
				if sysHotkeyID != 0 {
					ui.UnregisterHotkey(sysHotkeyID)
					sysHotkeyID = 0
					sysHotkeyNote = "已注销"
				} else {
					sysHotkeyNote = "当前没有热键"
				}
				ui.RequestRedraw()
			}),
			ui.Text(sysHotkeyNote).St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("屏保常亮").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Switch("sys-awake", sysWakeLock, func(v bool) {
				sysWakeLock = v
				ui.SetKeepAwake(v)
				if v {
					sysMsg = "已调用 SetThreadExecutionState：不会休眠、不会关屏"
				} else {
					sysMsg = "已恢复默认电源策略"
				}
				ui.RequestRedraw()
			}),
			ui.Text("ES_CONTINUOUS | ES_DISPLAY_REQUIRED").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("剪贴板").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Input("sys-clip", sysClipText, "输入文本后点「写入剪贴板」", func(v string) {
				sysClipText = v
			}),
			ui.Btn("写入", ui.BtnSecondary, func() {
				if ui.ClipboardSetText(sysClipText) {
					sysMsg = "已写入剪贴板：" + sysClipText
				} else {
					sysMsg = "写入剪贴板失败"
				}
				ui.RequestRedraw()
			}),
			ui.Btn("读取", ui.BtnGhost, func() {
				sysClipText = ui.ClipboardText()
				sysMsg = "剪贴板内容：" + sysClipText
				ui.RequestRedraw()
			}),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("托盘通知").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Btn("弹出气泡", ui.BtnSecondary, func() {
				if sysTray == nil {
					sysMsg = "请先在上方「多窗口 / 托盘」里创建托盘图标"
				} else {
					sysTray.Notify("GoGUI 系统能力", "这是一条 NIF_INFO 气泡通知")
					sysMsg = "已弹出托盘气泡通知"
				}
				ui.RequestRedraw()
			}),
			ui.Btn("检测系统主题", ui.BtnGhost, func() {
				mode := "浅色"
				if ui.SystemDarkMode() {
					mode = "深色"
				}
				sysMsg = "注册表 AppsUseLightTheme：系统当前为" + mode + "模式"
				ui.RequestRedraw()
			}),
			ui.Text("NIF_INFO 气泡 / RegQueryValueEx").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Text(sysMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	).St(ui.St().GapOf(10))

	return ui.Column(
		pageHead("系统能力", "图片 / 拖入文件 / 毛玻璃 / Emoji / 动画 / 输入法 / 滚动条 / 系统操作"),
		sysSection("图片解码与纹理绘制", "PNG/JPEG/GIF/BMP → RGBA 纹理 → 圆角裁剪", imgBody),
		sysSection("拖入文件 WM_DROPFILES", "拖到窗口内松手 → 落点命中节点 OnDrop → 路径列表 / 图片自动预览", dropBody),
		sysSection("毛玻璃 backdrop-filter", "离屏 FBO + Kawase 模糊", glass),
		sysSection("动画系统", "Tween / Spring / Keyframes，仅在有动画时逐帧重绘", anim),
		sysSection("多字体与彩色 Emoji", "opentype 主字体 + Segoe UI Symbol fallback + GDI 彩色 Emoji", fontRow),
		sysSection("输入法（IME）", "WM_IME_COMPOSITION 组合串预览与上屏", ime),
		sysSection("滚动条", "内容溢出时显示，悬停加宽、可拖拽", scrollDemo),
		sysSection("多窗口 / 托盘 / 文件对话框", "独立 GL 上下文窗口、任务栏托盘、系统对话框", platform),
		sysSection("系统操作能力", "置顶 / 全局热键 / 托盘气泡 / 剪贴板 / 阻止休眠 / 深浅色 / 回收站", opsBody),
	).St(ui.St().Padding(26).GapOf(18))
}

// sysChildView 子窗口内容。
func sysChildView() *ui.Node {
	return ui.Column(
		ui.Text("独立窗口").St(ui.St().Text(22, 700, ui.ColText)),
		ui.Text("每个窗口拥有自己的 GL 上下文、字形图集与布局树。").St(ui.St().Text(12.5, 400, ui.ColTextDim).WrapOn()),
		ui.Spacer(),
		ui.Row(
			ui.Btn("再开一个", ui.BtnSecondary, func() {
				sysOpened++
				ui.NewWindow(ui.Options{Title: fmt.Sprintf("子窗口 #%d", sysOpened), Width: 480, Height: 320}, sysChildView)
			}),
			ui.Btn("关闭本窗口", ui.BtnDanger, func() {
				if a := ui.Current(); a != nil {
					a.Close()
				}
			}),
		).St(ui.St().GapOf(10)),
	).St(ui.St().Padding(26).GapOf(12).Hp(100).BgC(ui.ColBg))
}

// ---- 案例：视频播放器 ----

var (
	videoPl       *ui.Player
	videoEp       = 1
	videoDemoClip *ui.VideoClip
	// videoURL 默认播放地址（HLS 直链），启动后自动载入
	videoURL       = "https://v6.ppqrrs.com/wjv6/202610/25/4qUXBp64qg90/video/index.m3u8"
	videoAutoTried bool
	videoMsg       = "点击画面播放/暂停；拖动进度条跳转；聚焦画面后 ←/→ 快退快进 5 秒"
)

// demoSource 内置程序化片段（懒生成，多个演示播放器共用同一帧序列）。
func demoSource() *ui.VideoClip {
	if videoDemoClip == nil {
		videoDemoClip = demoClip()
	}
	return videoDemoClip
}

// videoHooks 给播放器挂上演示用回调。
func videoHooks(p *ui.Player) *ui.Player {
	p.OnPlay = func() { videoMsg = "开始播放" }
	p.OnPause = func() { videoMsg = "已暂停" }
	p.OnLoopChange = func(loop bool) {
		if loop {
			videoMsg = "已开启循环播放"
		} else {
			videoMsg = "已关闭循环播放"
		}
	}
	p.OnVolume = func(v float32, muted bool) {
		if muted {
			videoMsg = "已静音"
			return
		}
		videoMsg = fmt.Sprintf("音量 %.0f%%", v*100)
	}
	p.OnFullscreen = func() {
		// 播放器铺满窗口 + 窗口真全屏（覆盖桌面）
		if p.Fullscreen {
			ui.SetFocus("gvf-surface") // 让 ESC 交给播放器/窗口处理
			videoMsg = "已全屏：播放器铺满整个桌面（ESC 或再点全屏按钮退出）"
		} else {
			ui.SetFocus("")
			videoMsg = "已退出全屏"
		}
	}
	return p
}

// demoPlayer 首次访问时生成程序化片段（320×180 / 24fps / 4s）。
func demoPlayer() *ui.Player {
	if videoPl == nil {
		videoPl = videoHooks(ui.NewPlayer(demoSource()))
		videoPl.AutoHide = true
	}
	return videoPl
}

// demoClip 程序化生成一段循环动画片段（无需外部素材）。
func demoClip() *ui.VideoClip {
	const w, h, fps, secs = 320, 180, 24, 4
	total := fps * secs
	frames := make([]ui.VideoFrame, 0, total)
	dots := [][3]byte{{255, 214, 130}, {130, 226, 255}, {255, 140, 205}, {150, 245, 170}}
	for i := 0; i < total; i++ {
		t := float64(i) / fps
		pix := make([]byte, w*h*4)
		// 背景：极坐标波纹 + 中心辉光
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dx := float64(x) - w/2
				dy := float64(y) - h/2
				dist := math.Hypot(dx, dy) / (h * 0.62)
				if dist > 1 {
					dist = 1
				}
				ang := math.Atan2(dy, dx) + t*0.9
				wave := 0.5 + 0.5*math.Sin(ang*2+t*1.4)
				glow := 1 - dist
				o := (y*w + x) * 4
				pix[o] = byte(18 + 90*wave*glow)
				pix[o+1] = byte(20 + 70*glow)
				pix[o+2] = byte(46 + 130*glow)
				pix[o+3] = 255
			}
		}
		// 四个绕圈的运动圆点
		for k := 0; k < 4; k++ {
			ang := t*1.1 + float64(k)*math.Pi/2
			cx := int(float64(w)/2 + 96*math.Cos(ang))
			cy := int(float64(h)/2 + 58*math.Sin(ang))
			fillDisc(pix, w, h, cx, cy, 8, dots[k])
		}
		// 中心呼吸方块
		side := int(10 + 6*math.Sin(t*3))
		fillRect(pix, w, h, w/2-side/2, h/2-side/2, side, side, [3]byte{255, 255, 255})
		// 底部进度条（随帧增长的亮条）
		fillRect(pix, w, h, 0, h-5, w*(i+1)/total, 5, [3]byte{120, 132, 255})
		frames = append(frames, ui.VideoFrame{Pix: pix, W: w, H: h})
	}
	return ui.NewVideoClip(frames, fps, true)
}

// fillDisc 在 RGBA 缓冲里画实心圆。
func fillDisc(pix []byte, w, h, cx, cy, rad int, c [3]byte) {
	for y := cy - rad; y <= cy+rad; y++ {
		if y < 0 || y >= h {
			continue
		}
		for x := cx - rad; x <= cx+rad; x++ {
			if x < 0 || x >= w {
				continue
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy > rad*rad {
				continue
			}
			o := (y*w + x) * 4
			pix[o], pix[o+1], pix[o+2], pix[o+3] = c[0], c[1], c[2], 255
		}
	}
}

// fillRect 在 RGBA 缓冲里填充矩形（越界自动裁剪）。
func fillRect(pix []byte, w, h, x0, y0, rw, rh int, c [3]byte) {
	for y := y0; y < y0+rh; y++ {
		if y < 0 || y >= h {
			continue
		}
		for x := x0; x < x0+rw; x++ {
			if x < 0 || x >= w {
				continue
			}
			o := (y*w + x) * 4
			pix[o], pix[o+1], pix[o+2], pix[o+3] = c[0], c[1], c[2], 255
		}
	}
}

// openVideoGIF 选择本地 GIF 作为新片源。
func openVideoGIF() {
	path, ok := ui.OpenFileDialog("选择 GIF 动图", "GIF|*.gif|所有文件|*.*")
	if !ok {
		return
	}
	clip, err := ui.LoadGIF(path)
	if err != nil {
		videoMsg = "载入失败：" + err.Error()
		ui.RequestRedraw()
		return
	}
	p := demoPlayer()
	p.SetClip(clip)
	p.Play()
	videoMsg = fmt.Sprintf("已载入 %s（%d 帧）", path, clip.FrameCount())
	ui.RequestRedraw()
}

// useBuiltinClip 切回内置合成片段（网络失败或想跑离线演示时）。
func useBuiltinClip() {
	old := videoPl
	p := videoHooks(ui.NewPlayer(demoSource()))
	p.AutoHide = true
	p.Fullscreen = false
	videoPl = p
	if old != nil && old != p {
		old.Close()
	}
	p.Play()
	videoMsg = "已切换到内置片段（320×180 / 24fps）"
	ui.RequestRedraw()
}

// openVideoFile 选择本地视频（mp4 / mkv / mov / avi / webm / flv / ts / m3u8）。
func openVideoFile() {
	path, ok := ui.OpenFileDialog("选择视频", ui.MediaFilters)
	if !ok {
		return
	}
	loadMedia(path)
}

// loadMedia 用 ffmpeg 解码打开媒体：ffprobe 探测在后台 goroutine，完成后回 UI 线程换源。
func loadMedia(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	videoMsg = "正在载入 " + path + " …"
	ui.RequestRedraw()
	cur := ui.Current()
	go func() {
		p, err := ui.NewMediaPlayerWith(path, ui.MediaConfig{MaxWidth: 1280, MaxHeight: 720})
		apply := func() {
			if err != nil {
				videoMsg = "载入失败：" + err.Error()
				ui.RequestRedraw()
				return
			}
			old := videoPl
			p.Fullscreen = false
			p.AutoHide = true
			videoHooks(p)
			videoPl = p
			if old != nil && old != p {
				old.Close()
			}
			p.Play()
			info := p.Info()
			w, h := p.Size()
			audio := "无音频"
			if info.HasAudio {
				audio = info.AudioCodec
				if !p.HasAudio() {
					audio += "（无音频设备）"
				}
			}
			videoMsg = fmt.Sprintf("已载入 %s · %s %s · %d×%d · %s · %.2f fps · %s",
				filepath.Base(path), strings.ToUpper(info.Format), info.VideoCodec,
				w, h, audio, info.FPS, fmtDur(info.Duration))
			ui.RequestRedraw()
		}
		if cur != nil {
			cur.Post(apply)
		} else {
			apply()
		}
	}()
}

// playVideoURL 播放输入框里的直链（mp4 / m3u8 等）。
func playVideoURL() {
	u := strings.TrimSpace(videoURL)
	if u == "" {
		videoMsg = "请先粘贴视频直链（mp4 / m3u8 等）"
		ui.RequestRedraw()
		return
	}
	loadMedia(u)
}

// fmtDur 时长文本。
func fmtDur(d time.Duration) string {
	if d <= 0 {
		return "时长未知"
	}
	t := int(d.Seconds())
	if t >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", t/3600, t%3600/60, t%60)
	}
	return fmt.Sprintf("%d:%02d", t/60, t%60)
}

// videoFitDemo 同一片段的三种填充方式（只显示画面，不带控制条）。
func videoFitDemo(label string, fit ui.VideoFit) *ui.Node {
	q := videoFitPlayer(fit)
	q.Fit = fit
	q.HideControls = true
	q.SetPos(demoPlayer().Pos())
	return ui.Column(
		ui.Row(
			ui.Text(label).St(ui.St().Text(11.5, 600, ui.ColTextDim)),
			ui.Spacer(),
			ui.Text(map[ui.VideoFit]string{ui.FitContain: "留边", ui.FitCover: "裁切", ui.FitFill: "拉伸"}[fit]).
				St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().Wp(100)).AlignTo(ui.AlignCenter),
		ui.Video(q).St(ui.St().Wp(100).H(110).R(10).Clipped().BgC("#000000")),
	).St(ui.St().GrowTo(1).GapOf(6))
}

// videoFitPlayer 三种填充方式的演示播放器（按 Fit 缓存）。
// 旧实现每帧 ui.NewPlayer(demoSource()) 新建播放器，而 App 里每个播放器都带一张
// 帧纹理且从不回收，纹理与 Player 随帧累积——这是示例页内存涨到 GB 级的主因。
var videoFitPlays = map[ui.VideoFit]*ui.Player{}

func videoFitPlayer(fit ui.VideoFit) *ui.Player {
	if q := videoFitPlays[fit]; q != nil {
		return q
	}
	q := ui.NewPlayer(demoSource())
	q.Fit = fit
	q.HideControls = true
	videoFitPlays[fit] = q
	return q
}

// backendOf 播放器当前使用的解码后端（ffmpeg / system）。
func backendOf(p *ui.Player) string {
	if p != nil {
		if b := p.Info().Backend; b != "" {
			return b
		}
	}
	return ui.BackendName()
}

// videoMetaTags 顶部信息标签：内置片段显示帧数，媒体源显示容器 / 编码 / 音频 / 时长。
func videoMetaTags(p *ui.Player, state string) *ui.Node {
	w, h := p.Size()
	tags := []*ui.Node{ui.Tag(fmt.Sprintf("尺寸 %d×%d", w, h), "#a5b4fc")}
	info := p.Info()
	switch {
	case info.HasVideo:
		name := strings.ToUpper(info.Format)
		if name == "" {
			name = "视频"
		}
		tags = append(tags,
			ui.Tag(name+" · "+info.VideoCodec, "#22d3ee"),
			ui.Tag(fmt.Sprintf("%.2f fps", info.FPS), "#34d399"),
		)
		if info.HasAudio {
			tags = append(tags, ui.Tag("音频 "+info.AudioCodec, "#f472b6"))
		} else {
			tags = append(tags, ui.Tag("无声", "#94a3b8"))
		}
	case p.Clip != nil:
		tags = append(tags,
			ui.Tag("内置片段", "#22d3ee"),
			ui.Tag(strconv.Itoa(p.Clip.FrameCount())+" 帧", "#34d399"),
		)
	}
	tags = append(tags,
		ui.Tag("解码 "+ui.FormatBackend(backendOf(p)), "#38bdf8"),
		ui.Tag(fmtDur(p.Duration()), "#fbbf24"),
		ui.Tag(state, "#818cf8"),
	)
	return ui.Row(tags...).St(ui.St().GapOf(8))
}

func videoCase() *ui.Node {
	// 首次进入时自动载入默认播放地址（m3u8 直链），失败仍会保留内置片段
	if !videoAutoTried && strings.TrimSpace(videoURL) != "" {
		videoAutoTried = true
		loadMedia(videoURL)
	}
	p := demoPlayer()
	// 字幕与上/下一集：控制条只在回调非空时显示跳集按钮
	p.Subtitle = fmt.Sprintf("第 %d 集 · 正常人才干不出那种事", videoEp)
	p.OnPrev = func() {
		if videoEp > 1 {
			videoEp--
		}
		p.SetPos(0)
		p.Play()
		videoMsg = fmt.Sprintf("已切到第 %d 集（示例：同一个片段）", videoEp)
		ui.RequestRedraw()
	}
	p.OnNext = func() {
		videoEp++
		p.SetPos(0)
		p.Play()
		videoMsg = fmt.Sprintf("已切到第 %d 集（示例：同一个片段）", videoEp)
		ui.RequestRedraw()
	}
	state := "暂停"
	if p.PlayingNow() {
		state = "播放中"
	}
	posText := fmt.Sprintf("%.2f / %.2f s",
		float64(p.Pos().Milliseconds())/1000, float64(p.Duration().Milliseconds())/1000)
	loopText := "关闭"
	if p.Loop {
		loopText = "开"
	}

	return ui.Column(
		pageHead("视频播放器", "mp4 / mkv / mov / avi / webm / flv / ts / m3u8 均可播放（ffmpeg 解码）+ 逐帧纹理、拖动进度、倍速、音量、循环、全屏"),
		videoMetaTags(p, state),
		// 全屏时由根层铺满播放，这里不再重复绘制
		ui.If(!p.Fullscreen, func() *ui.Node {
			return ui.VideoPlayer("gv", p).St(ui.St().H(380).BgC("#000000"))
		}),
		ui.Row(
			ui.Btn("◀ 上一集", ui.BtnSecondary, func() { p.OnPrev() }),
			ui.Spacer(),
			ui.Tag(fmt.Sprintf("第 %d 集 · 实时源 · HLS 播放 · HD 国语", videoEp), "#a5b4fc"),
			ui.Spacer(),
			ui.Btn("下一集 ▶", ui.BtnSecondary, func() { p.OnNext() }),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Text(videoMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
		ui.Row(
			ui.Btn("打开本地视频", ui.BtnPrimary, openVideoFile),
			ui.Btn("打开 GIF 动图", ui.BtnSecondary, openVideoGIF),
			ui.Btn("内置片段", ui.BtnGhost, useBuiltinClip),
			ui.Btn("暂停 / 播放", ui.BtnSecondary, func() {
				p.TogglePlay()
				ui.RequestRedraw()
			}),
			ui.Btn("回到第一帧", ui.BtnGhost, func() { p.SetPos(0) }),
			ui.Btn("旋转 90°", ui.BtnGhost, func() { p.RotateBy(1) }),
			ui.Btn(map[bool]string{true: "解锁画面", false: "锁定画面"}[p.Locked], ui.BtnGhost, func() { p.ToggleLock() }),
			ui.Btn("全屏演示", ui.BtnGhost, func() {
				p.Fullscreen = true
				if cur := ui.Current(); cur != nil {
					cur.SetFullscreen(true)
				}
				ui.SetFocus("gvf-surface") // 让 ESC 走播放器
				videoMsg = "已全屏：播放器铺满整个桌面（ESC 或右上角按钮退出）"
				ui.RequestRedraw()
			}),
		).St(ui.St().GapOf(10)),
		ui.Row(
			ui.Box(ui.Input("video-url", videoURL, "粘贴视频直链（https://… .mp4 / .m3u8）", func(v string) { videoURL = v })).Grow(1),
			ui.Btn("播放链接", ui.BtnSecondary, playVideoURL),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Card(
				ui.Title("支持格式", "全部交给 ffmpeg demux / decode，不依赖系统解码器"),
				ui.Row(ui.Tag("mp4", "#a5b4fc"), ui.Tag("mkv", "#22d3ee"), ui.Tag("mov", "#f472b6"),
					ui.Tag("avi", "#34d399"), ui.Tag("webm", "#fbbf24"), ui.Tag("flv", "#fb923c")).St(ui.St().GapOf(6)),
				ui.Row(ui.Tag("ts", "#c084fc"), ui.Tag("m2ts", "#818cf8"), ui.Tag("m3u8 / HLS", "#38bdf8"),
					ui.Tag("远端直链", "#94a3b8")).St(ui.St().GapOf(6)),
				ui.KeyValue("解码", "ffmpeg 子进程 → rawvideo RGBA"),
				ui.KeyValue("声音", "waveOut 输出，以音频时钟同步画面"),
				ui.KeyValue("分辨率", "跟随源尺寸（可用 MediaConfig 限高）"),
			).St(ui.St().GrowTo(1)),
			ui.Card(
				ui.Title("操作方式", "鼠标 + 键盘（画面获焦后）"),
				ui.KeyValue("播放 / 暂停", "点击画面 / 控制条按钮 / Space"),
				ui.KeyValue("进度跳转", "点击或拖动进度条"),
				ui.KeyValue("快退 / 快进", "← / →（各 5 秒）"),
				ui.KeyValue("音量", "↑ / ↓ 或拖动音量条"),
				ui.KeyValue("倍速 / 循环 / 全屏", "控制条右侧胶囊按钮"),
			).St(ui.St().GrowTo(1)),
		).St(ui.St().GapOf(14)),
		ui.Row(
			ui.Card(
				ui.Title("实时状态", "每帧从 Player 读取"),
				ui.KeyValue("当前帧", strconv.Itoa(p.FrameIndex())),
				ui.KeyValue("播放位置", posText),
				ui.KeyValue("播放速率", fmt.Sprintf("%g×", p.SpeedValue())),
				ui.KeyValue("循环", loopText),
				ui.KeyValue("旋转", fmt.Sprintf("%d°", p.RotationDeg())),
				ui.KeyValue("锁定", map[bool]string{true: "已锁定", false: "未锁"}[p.Locked]),
				ui.KeyValue("音量", fmt.Sprintf("%.0f%%", p.Volume*100)),
			).St(ui.St().GrowTo(1)),
		).St(ui.St().GapOf(14)),
		ui.Card(
			ui.Title("画面填充方式", "Player.Fit：Contain 留边 / Cover 裁切 / Fill 拉伸"),
			ui.Row(
				videoFitDemo("Contain", ui.FitContain),
				videoFitDemo("Cover", ui.FitCover),
				videoFitDemo("Fill", ui.FitFill),
			).St(ui.St().GapOf(12)),
		),
		ui.Card(
			ui.Title("接入真实视频", "三种帧源：内存帧序列 / 外部解码器 / 内置 ffmpeg"),
			ui.Text("ui.LoadGIF(path) → VideoClip；ui.LoadImageSequence(paths, fps, loop) → VideoClip；"+
				"视频文件直接 ui.NewMediaPlayer(path)（或 ui.OpenMedia(path) + ui.NewPlayerOf），"+
				"内部用 ffprobe 探测 + ffmpeg 解 rawvideo RGBA 帧、s16le 音频经 waveOut 输出，"+
				"播放位置以音频设备时钟为基准。需 PATH 中有 ffmpeg/ffprobe；播放完调 Player.Close 释放进程与设备。").
				St(ui.St().Text(12, 400, ui.ColTextDim).WrapOn()),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

// musicDemo 内置演示曲目（程序化合成 WAV）。
type musicDemo struct {
	title, artist, album string
	root                 float64 // 基频（Hz）
	bpm                  int
	seed                 int
}

var musicDemos = []musicDemo{
	{"Sunrise Drive", "GoGUI Synth", "Demo Album · Vol.1", 220.0, 112, 1},
	{"Neon Nights", "GoGUI Synth", "Demo Album · Vol.1", 164.8, 128, 2},
	{"Midnight Rain", "GoGUI Synth", "Demo Album · Vol.2", 196.0, 92, 3},
}

var (
	musicIndex int
	musicAuto  bool
	musicSrc   *ui.MusicSource
	musicMsg   = "点播放听到的是程序化合成的示例曲目（无需外部素材）；频谱随声音实时跳动"
	musicPaths = map[int]string{}
)

// musicDemoPath 生成 / 复用内置 WAV（放在系统缓存目录）。
func musicDemoPath(i int) string {
	if p := musicPaths[i]; p != "" {
		return p
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "gogui-gallery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	d := musicDemos[i]
	p := filepath.Join(dir, fmt.Sprintf("demo-%d-%d.wav", i, d.seed))
	if st, serr := os.Stat(p); serr != nil || st.Size() < 4096 {
		if werr := writeDemoWAV(p, d); werr != nil {
			return ""
		}
	}
	musicPaths[i] = p
	return p
}

// writeDemoWAV 生成一段简单合成器循环（底鼓 + 贝斯 + 琶音 + 踩镲），
// 44.1kHz / 单声道 / 16bit，总长 16 秒，可直接被 ffmpeg 解码播放。
func writeDemoWAV(path string, d musicDemo) error {
	const rate = 44100
	const secs = 16.0
	n := int(secs * rate)
	samples := make([]int16, n)
	beat := 60.0 / float64(d.bpm)
	// 大调音阶（主音 → 八度），琶音按 1-3-5 跳进
	scale := []float64{1, 9.0 / 8, 5.0 / 4, 3.0 / 2, 5.0 / 3, 2, 5.0 / 2}
	rnd := uint32(2463534242 + d.seed*7919)
	noise := func() float64 {
		rnd ^= rnd << 13
		rnd ^= rnd >> 17
		rnd ^= rnd << 5
		return float64(rnd)/float64(1<<31) - 1
	}
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		var v float64

		// 主旋律：每个 1/8 拍换一个音，指数衰减包络
		eighth := beat / 2
		si := int(t / eighth)
		st := t - float64(si)*eighth
		note := d.root * scale[(si*3+int(d.seed)*2)%len(scale)]
		env := math.Exp(-st*6) * 0.28
		v += math.Sin(2*math.Pi*note*t) * env
		v += math.Sin(2*math.Pi*note*2*t) * env * 0.16 // 泛音，频谱更丰富

		// 贝斯：每拍一个低音（正弦 + 三次谐波）
		bi := int(t / beat)
		bt := t - float64(bi)*beat
		bf := d.root / 4 * scale[bi%4]
		benv := math.Exp(-bt*3.2) * 0.34
		v += math.Sin(2*math.Pi*bf*t) * benv
		v += math.Sin(2*math.Pi*bf*3*t) * benv * 0.12

		// 底鼓：每拍 120Hz→45Hz 扫频 + 指数衰减
		if bt < 0.18 {
			kf := 45 + 75*math.Exp(-bt*28)
			v += math.Sin(2*math.Pi*kf*bt) * math.Exp(-bt*16) * 0.55
		}
		// 踩镲：半拍处的短噪声
		hi := t - float64(int(t/(beat/2)))*(beat/2)
		if hi < 0.05 {
			v += noise() * math.Exp(-hi*90) * 0.12
		}

		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		samples[i] = int16(v * 32000)
	}

	data := make([]byte, 44+len(samples)*2)
	putU32 := func(off int, v uint32) {
		data[off], data[off+1], data[off+2], data[off+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	putU16 := func(off int, v uint16) {
		data[off], data[off+1] = byte(v), byte(v>>8)
	}
	copy(data[0:4], "RIFF")
	putU32(4, uint32(len(data)-8))
	copy(data[8:12], "WAVE")
	copy(data[12:16], "fmt ")
	putU32(16, 16)
	putU16(20, 1) // PCM
	putU16(22, 1) // 单声道
	putU32(24, rate)
	putU32(28, rate*2)
	putU16(32, 2)
	putU16(34, 16)
	copy(data[36:40], "data")
	putU32(40, uint32(len(samples)*2))
	for i, s := range samples {
		putU16(44+i*2, uint16(s))
	}
	return os.WriteFile(path, data, 0o644)
}

// currentMusicTrack 当前曲目信息。
func currentMusicTrack() ui.MusicTrack {
	if musicIndex < 0 || musicIndex >= len(musicDemos) {
		return ui.MusicTrack{}
	}
	d := musicDemos[musicIndex]
	t := ui.MusicTrack{Title: d.title, Artist: d.artist, Album: d.album}
	if musicSrc != nil {
		t.Path = musicSrc.Path()
		t.Dur = musicSrc.Duration()
	}
	return t
}

// openMusicDemo 打开指定下标的示例曲目。
func openMusicDemo(i int) {
	n := len(musicDemos)
	if i < 0 {
		i = n - 1
	}
	if i >= n {
		i = 0
	}
	p := musicDemoPath(i)
	if p == "" {
		musicMsg = "示例音频生成失败（缓存目录不可写？）"
		ui.RequestRedraw()
		return
	}
	musicIndex = i
	openMusicFile(p, true)
	musicMsg = fmt.Sprintf("正在播放示例：%s · %s（%s）",
		musicDemos[i].title, musicDemos[i].artist, fmtDur(musicSrc.Duration()))
	ui.RequestRedraw()
}

// openMusicFile 打开音频文件并（可选）自动播放，旧音源自动关闭。
func openMusicFile(path string, play bool) {
	if strings.TrimSpace(path) == "" {
		return
	}
	src, err := ui.OpenMusic(path)
	if err != nil {
		musicMsg = "打开失败：" + err.Error()
		ui.RequestRedraw()
		return
	}
	old := musicSrc
	musicSrc = src
	if play {
		src.Play()
	}
	if old != nil && old != src {
		old.Close()
	}
	info := src.Info()
	name := filepath.Base(path)
	if info.AudioCodec != "" {
		musicMsg = fmt.Sprintf("已打开 %s · %s · %d Hz · %s",
			name, info.AudioCodec, info.SampleRate, fmtDur(src.Duration()))
	} else {
		musicMsg = fmt.Sprintf("已打开 %s（%s）", name, fmtDur(src.Duration()))
	}
	ui.RequestRedraw()
}

// openMusicDialog 选择本地音乐文件。
func openMusicDialog() {
	path, ok := ui.OpenFileDialog("选择音乐", ui.MusicFilters)
	if !ok {
		return
	}
	openMusicFile(path, true)
}

// musicRow 播放列表一行（当前项高亮）。
func musicRow(i int, d musicDemo) *ui.Node {
	active := i == musicIndex
	bg := "rgba(255,255,255,0.03)"
	color := ui.ColTextDim
	idxColor := ui.ColTextMuted
	if active {
		bg = "rgba(245,197,24,0.12)"
		color = "#ffffff"
		idxColor = "#f5c518"
	}
	row := ui.Row(
		ui.Text(fmt.Sprint(i+1)).St(ui.St().W(18).Text(11, 700, idxColor)),
		ui.Column(
			ui.Text(d.title).St(ui.St().Text(12.5, 600, color)),
			ui.Text(fmt.Sprintf("%s · %d BPM", d.artist, d.bpm)).
				St(ui.St().Text(10.5, 400, ui.ColTextMuted)),
		).Grow(1).St(ui.St().GapOf(2)),
		ui.If(active, func() *ui.Node { return ui.Tag("播放中", "#f5c518") }),
	).St(ui.St().GapOf(10).PadXY(10, 8).R(9).BgC(bg))
	row.K(fmt.Sprintf("music-row-%d", i)).Tap(func() { openMusicDemo(i) })
	return row
}

// musicSpectrumCard 频谱实验室：大号柱状 + 镜像频谱与统计标签。
func musicSpectrumCard(src *ui.MusicSource) *ui.Node {
	if src == nil || src.Spectrum() == nil {
		return ui.Card(ui.Title("频谱特效", "打开音源后显示实时频谱"))
	}
	sp := src.Spectrum()
	bands := sp.Bands()
	var peak, sum float32
	for _, v := range bands {
		if v > peak {
			peak = v
		}
		sum += v
	}
	avg := float32(0)
	if len(bands) > 0 {
		avg = sum / float32(len(bands))
	}
	return ui.Card(
		ui.Title("频谱特效", "FFT（2048 点 Hann 窗）→ 40 个对数频段 → attack/release 平滑"),
		ui.MusicSpectrum("gmp-spec-big", sp, 40, "rgba(245,197,24,0.95)", "rgba(139,92,246,0.95)").
			St(ui.St().Wp(100).H(96)),
		ui.MusicMirrorSpectrum("gmp-spec-mirror", sp, 56, "rgba(34,211,238,0.92)", "rgba(129,140,248,0.92)").
			St(ui.St().Wp(100).H(72)),
		ui.Row(
			ui.Tag(fmt.Sprintf("峰值 %.2f", peak), "#fbbf24"),
			ui.Tag(fmt.Sprintf("均值 %.2f", avg), "#22d3ee"),
			ui.Tag(fmt.Sprintf("%d 段", len(bands)), "#a5b4fc"),
			ui.Tag(fmt.Sprintf("%d Hz", src.Info().SampleRate), "#34d399"),
		).St(ui.St().GapOf(8)),
		ui.Text("频谱数据来自正在播放的 PCM（与 waveOut 同一份数据流），暂停时能量自动衰减；"+
			"ui.MusicSpectrum / ui.MusicMirrorSpectrum 可嵌到任意界面，也可单独用 ui.NewSpectrum(64) 分析自己的 PCM。").
			St(ui.St().Text(12, 400, ui.ColTextDim).WrapOn()),
	)
}

func musicCase() *ui.Node {
	// 首次进入自动准备并播放第一首示例（同步生成 16s WAV，几十毫秒）
	if !musicAuto {
		musicAuto = true
		openMusicDemo(0)
	}
	src := musicSrc
	if src == nil {
		return ui.Column(
			pageHead("音乐播放器", "mp3 / flac / wav / m4a / aac / ogg / opus / wma / ape（ffmpeg 解码）+ 实时 FFT 频谱"),
			ui.Empty("暂无音源", "点下方「打开本地音乐」选择文件，或「播放示例」听程序化生成的曲目"),
			ui.Row(
				ui.Btn("打开本地音乐", ui.BtnPrimary, openMusicDialog),
				ui.Btn("播放示例", ui.BtnSecondary, func() { openMusicDemo(musicIndex) }),
			).St(ui.St().GapOf(10)),
			ui.Text(musicMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
		).St(ui.St().Padding(26).GapOf(16))
	}

	state := "暂停"
	if src.Playing() {
		state = "播放中"
	}
	rows := make([]*ui.Node, 0, len(musicDemos))
	for i, d := range musicDemos {
		rows = append(rows, musicRow(i, d))
	}
	info := src.Info()
	loopText := "循环关"
	if src.Loop() {
		loopText = "循环开"
	}

	return ui.Column(
		pageHead("音乐播放器", "mp3 / flac / wav / m4a / aac / ogg / opus / wma / ape（ffmpeg 解码）+ 实时 FFT 频谱特效"),
		ui.Row(
			ui.Tag(strings.ToUpper(info.Format), "#22d3ee"),
			ui.Tag("解码 "+ui.FormatBackend(info.Backend), "#38bdf8"),
			ui.Tag(info.AudioCodec, "#f472b6"),
			ui.Tag(fmt.Sprintf("%d Hz", info.SampleRate), "#34d399"),
			ui.Tag(fmtDur(src.Duration()), "#fbbf24"),
			ui.Tag(state, "#818cf8"),
			ui.Tag(fmt.Sprintf("倍速 %g×", src.Speed()), "#a5b4fc"),
			ui.Tag(loopText, "#94a3b8"),
		).St(ui.St().GapOf(8)),
		ui.MusicPlayer("gmp", src, ui.MusicPlayerOptions{
			Track: currentMusicTrack(),
			Bars:  40,
			Prev:  func() { openMusicDemo(musicIndex - 1) },
			Next:  func() { openMusicDemo(musicIndex + 1) },
		}),
		ui.Row(
			ui.Btn("打开本地音乐", ui.BtnPrimary, openMusicDialog),
			ui.Btn("播放 / 暂停", ui.BtnSecondary, func() {
				src.TogglePlay()
				ui.RequestRedraw()
			}),
			ui.Btn("回到开头", ui.BtnGhost, func() { src.Seek(0) }),
			ui.Btn("单曲循环", ui.BtnGhost, func() {
				src.SetLoop(!src.Loop())
				ui.RequestRedraw()
			}),
			ui.Btn("上一首", ui.BtnGhost, func() { openMusicDemo(musicIndex - 1) }),
			ui.Btn("下一首", ui.BtnGhost, func() { openMusicDemo(musicIndex + 1) }),
		).St(ui.St().GapOf(10)),
		ui.Text(musicMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
		ui.Row(
			ui.Card(
				ui.Title("播放列表", "点击切换（示例曲目为程序化合成，也可换成自己的文件）"),
				ui.Column(rows...).St(ui.St().GapOf(4)),
			).Grow(1),
			ui.Card(
				ui.Title("接入真实音乐", "OpenMusic → MusicPlayer"),
				ui.KeyValue("打开", "ui.OpenMusic(path) / ui.OpenMusicWith(path, cfg)"),
				ui.KeyValue("组件", "ui.MusicPlayer(key, src, opt)"),
				ui.KeyValue("频谱", "src.Spectrum() → ui.MusicSpectrum"),
				ui.KeyValue("解码", "ffmpeg → s16le PCM → waveOut"),
				ui.KeyValue("释放", "src.Close()（退出时释放进程与设备）"),
				ui.Text("音乐与视频互不影响：视频用 ui.NewMediaPlayer，音乐用 ui.OpenMusic。"+
					"两者都以音频设备时钟为播放位置，拖动进度条会一次性定位并重建解码窗口。").
					St(ui.St().Text(12, 400, ui.ColTextDim).WrapOn()),
			).Grow(1),
		).St(ui.St().GapOf(14)),
		musicSpectrumCard(src),
		ui.Card(
			ui.Title("操作方式", "鼠标点击即可"),
			ui.KeyValue("播放 / 暂停", "中间金色圆形按钮"),
			ui.KeyValue("上一首 / 下一首", "两侧跳曲按钮（或点播放列表）"),
			ui.KeyValue("进度", "点击或拖动进度条"),
			ui.KeyValue("音量", "喇叭图标静音 / 拖动音量条"),
			ui.KeyValue("循环 / 倍速", "右侧胶囊按钮（0.5x~2x 循环切换）"),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

// ---- 案例：Markdown 渲染 ----

// mdDemoTab 当前示例文档页签（跨帧保持）。
var mdDemoTab = "gfm"

// mdSampleGFM GFM 基础语法。
const mdSampleGFM = `# GoGUI Markdown 渲染

参照 Web 版 **Markdown.vue** 的渲染思路：*块级解析* → ~~行内样式~~ → 组件映射。
行内支持：**加粗**、*斜体*、***粗斜体***、~~删除线~~、「行内代码」、==高亮==、[链接](https://go.dev)、<kbd>Ctrl</kbd>+<kbd>S</kbd>。

![GoGUI 图标](frontend/public/favicon.ico)

## 二级标题

### 三级标题

普通段落会自动折行：GoGUI 的 KText 只支持单一样式，组件在解析阶段把行内片段切成 token，
按可用宽度做贪心折行（中文逐字可断，英文按空格断行），因此长段落、窄窗口都不会溢出容器。

> 引用块：支持多行引用。
> 第二行仍然属于同一个引用。

> [!TIP]
> GitHub 风格的提示块：识别 [!NOTE] / [!TIP] / [!WARNING] / [!IMPORTANT] / [!CAUTION]。

- 无序列表项一
- 无序列表项二
  - 嵌套子项（按缩进识别）
  - 另一个子项
- 任务列表：
  - [x] 完成块级解析
  - [ ] 待办：Mermaid / 数学公式

1. 有序列表
2. 第二项
3. 第三项

| 能力 | Web 版 | GoGUI 版 |
| --- | :---: | --- |
| 标题 / 段落 / 列表 | ✅ | ✅ |
| 代码高亮 | highlight.js | 内置轻量着色 |
| 图片预览 | 全屏 lightbox | 全屏 lightbox |
| 远程图片 | ✅ | 仅本地文件 |

---

文字结尾。
`

// mdSampleExt 扩展块。
const mdSampleExt = `# 扩展块

[toc]

## 页签

[tabs]
[tab label="概览"]
这是 **第一个页签** 的内容。

- 支持块级语法
- 支持嵌套列表

[/tab]
[tab label="代码"]
` + "```go" + `
func main() {
    fmt.Println("hello")
}
` + "```" + `
[/tab]
[tab label="表格"]
| A | B |
| --- | --- |
| 1 | 2 |
[/tab]
[/tabs]

## 折叠

[details title="点击查看实现细节"]
[details] 内部可以继续写 Markdown：

1. 解析扩展块
2. 递归渲染子块
3. 状态按 key 保存在组件内
[/details]

## 卡片

[card title="GoGUI" subtitle="纯 Go 的即时模式 GUI" url="https://go.dev"]
卡片正文支持 **行内样式** 与列表：

- 声明式节点
- 帧内重建
- 状态按 Key 继承
[/card]

## 徽章与进度

[badge text="稳定" color="success"] [badge text="Beta" color="warning"] [badge text="实验" color="error"]

[progress value="72" max="100" label="组件完成度"]

## 图片集

[gallery]
![示例 1](frontend/public/favicon.ico)
![示例 2](frontend/public/favicon.ico)
[/gallery]

## 音视频占位

[video src="movie.mp4" title="示例视频"]
[audio src="song.mp3" title="示例音频"]
`

// mdSampleCode 代码高亮样本。
const mdSampleCode = "# 代码块渲染\n\n代码块会自动识别语言做轻量着色（关键字 / 字符串 / 注释 / 数字 / 函数名），\n并带行号、复制按钮；超过 22 行自动折叠。\n\n```go\npackage main\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n)\n\n// Greet 返回问候语\nfunc Greet(name string) string {\n\tif strings.TrimSpace(name) == \"\" {\n\t\tname = \"GoGUI\"\n\t}\n\treturn fmt.Sprintf(\"hello, %s\", name)\n}\n\nfunc main() {\n\tfmt.Println(Greet(\"markdown\"))\n}\n```\n\n```typescript\ninterface User {\n  id: number\n  name: string\n}\n\nexport async function load(id: number): Promise<User> {\n  const res = await fetch(`/api/users/${id}`)\n  if (!res.ok) throw new Error('load failed')\n  return res.json()\n}\n```\n\n```python\ndef fib(n: int) -> int:\n    \"\"\"斐波那契\"\"\"\n    a, b = 0, 1\n    for _ in range(n):\n        a, b = b, a + b\n    return a\n\nprint(fib(10))\n```\n\n```json\n{\n  \"name\": \"gogui\",\n  \"markdown\": true,\n  \"version\": 1.0\n}\n```\n\n```bash\n# 运行示例\ngo run ./cmd/gallery --case markdown\n```\n"

// mdSampleLong 长文 + 目录。
const mdSampleLong = `# 长文与目录

[toc]

## 一、设计目标

组件需要在「只有单一样式文本节点」的框架上还原 Web 版 Markdown 的观感：

1. **块级结构** —— 标题、段落、列表、引用、代码、表格、分隔线、图片；
2. **行内样式** —— 粗体、斜体、删除线、行内代码、链接、高亮、kbd；
3. **扩展块** —— tabs / details / card / badge / progress / gallery；
4. **交互** —— 代码复制、折叠展开、页签切换、图片全屏预览、链接外部打开。

## 二、排版策略

### 2.1 折行

框架的 KText 绘制只按换行符分行，Wrap 只影响测量。组件在解析后把行内片段切成 token，
用字体度量做贪心折行，再把每一行生成为 Row（每个 token 一个 Text 节点），
从而保证「测量高度 = 实际绘制高度」。

### 2.2 状态

页签、折叠、代码展开、复制提示、图片预览都按「父级 key + 块序号」保存在包级状态里，
因此界面每帧重建也不丢失交互状态。

### 2.3 性能

- 解析结果按源码字符串缓存（最多 24 篇，超出清空）；
- 每帧只重建节点、不重复解析；
- 字体走窗口级字体缓存（LRU 24 个字体实例）。

## 三、已知限制

- 远程图片会下载到临时目录缓存后正常显示（浏览器版仍为占位卡）；
- 音视频块降级为占位卡，真正播放请用 MusicPlayer / VideoPlayer；
- 数学公式为 LaTeX 子集，未覆盖的写法会退化成等宽源码；
- 表格列宽按内容比例分配，列很多时会出现横向压缩。

## 四、后续计划

- [x] 数学公式（LaTeX 子集真排版）
- [x] Mermaid 代码块 → 提示卡
- [x] 目录锚点滚动
- [x] 代码高亮
- [x] 图片 lightbox

> 备注：组件文件为 ui/markdown.go（渲染）、ui/markdown_parse.go（解析）、
> ui/markdown_highlight.go（着色）、ui/markdown_state.go（状态）。
`

// mdSampleNew 本轮新增渲染项的演示文档。
const mdSampleNew = `# 新增渲染项

本页演示本轮新增能力：数学公式、Mermaid 提示卡、脚注、定义列表、链接卡、diff 着色、上标下标、任务勾选与目录锚点滚动。

## 一、数学公式

行内：$E=mc^2$，求根公式写成 $x = (-b+\sqrt{b^2-4ac})/(2a)$，欧拉恒等式 $e^{i\pi}+1=0$。

块级：

$$
f(x) = \frac{ab} + \sqrt{x^2+1}
$$

$$
\sum_{k=1}^{n} k = \frac{n(n+12} + \int_0^1 x\,dx = \frac{12}
$$

矩阵与分段：

$$
A = \begin{pmatrix} a_{11} & a_{12} \\ a_{21} & a_{22} \end{pmatrix}, \quad |x| = \begin{cases} x & x \ge 0 \\ -x & x < 0 \end{cases}
$$

## 二、Mermaid 流程图（原生绘制）

~~~mermaid
flowchart LR
  A[解析] --> B{有缓存?}
  B -->|是| C[复用渲染树]
  B -->|否| D[重建节点]
  D --> E[写回缓存]
~~~

~~~mermaid
flowchart TD
  S[请求] --> P{命中缓存?}
  P -->|是| R[返回渲染树]
  P -->|否| B[解析源码]
  B --> L[分层排版]
  L --> R
~~~

## 三、脚注与定义列表

脚注引用会自动上标并与文末条目互链[^1]，支持缩进续行[^2]。

渲染树缓存
: 相同输入复用同一批节点，滚动只做一次 map 查找。
: 交互状态（页签 / 折叠 / 勾选）参与缓存键。

性能
: 长文档命中缓存后单帧只剩节点指针复制。

## 四、链接卡与自动链接

https://go.dev/doc/

裸链接同样可点击：https://github.com/golang/go

## 五、diff 代码块

~~~diff
--- a/ui/markdown.go
+++ b/ui/markdown.go
@@ -12,3 +12,4 @@
-// 旧：每帧重建整篇文档
+// 新：渲染树缓存 + 状态版本号
+// 新：diff 行底色
~~~

## 六、任务列表可勾选

- [x] 数学公式排版
- [ ] 点击方块可切换勾选状态

## 七、目录锚点滚动

点击下面目录条目可平滑滚动到对应小节。

[toc]

## 八、远程图片

桌面端会下载并缓存到临时目录，浏览器版显示占位卡。

[^1]: 脚注内容由 doc.Footnotes 统一渲染在文末。
[^2]: 第二行缩进续写也归属于同一条脚注。
`

// markdownCase Markdown 组件示例页。
func markdownCase() *ui.Node {
	docs := []struct{ id, label, src string }{
		{"gfm", "GFM 基础", mdSampleGFM},
		{"ext", "扩展块", mdSampleExt},
		{"code", "代码高亮", mdSampleCode},
		{"long", "长文 / 目录", mdSampleLong},
		{"new", "新增渲染", mdSampleNew},
	}
	keys := make([]string, 0, len(docs))
	labels := make([]string, 0, len(docs))
	src := docs[0].src
	for _, d := range docs {
		keys = append(keys, d.id)
		labels = append(labels, d.label)
		if d.id == mdDemoTab {
			src = d.src
		}
	}
	w := ui.ViewportW() - 232 - 96
	if w < 320 {
		w = 320
	}
	return ui.Column(
		ui.Column(
			ui.Row(
				ui.Text("Markdown 渲染").St(ui.St().Text(17, 700, ui.ColText)),
				ui.Spacer(),
				ui.Tag("ui.Markdown", "#a5b4fc"),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
			ui.Text("对齐 Web 版 Markdown.vue：块级解析 + 行内样式 + 扩展块；代码块带行号 / 着色 / 复制 / 折叠，图片点击可全屏预览。").St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
			ui.Tabs(keys, labels, mdDemoTab, func(k string) {
				mdDemoTab = k
				ui.RequestRedraw()
			}),
			ui.Divider(),
			ui.Markdown("gallery-md", src, ui.MarkdownOptions{Width: w}),
		).St(ui.St().Padding(22).R(16).BgC(ui.ColBgDeep).Stroke(1, ui.ColStroke).GapOf(12)),
	).St(ui.St().Padding(24).Wp(100))
}

// ---- 案例：WebView 内嵌浏览器（系统 WebView2）----

var (
	wvInst  *ui.WebView
	wvAddr  = "https://www.bing.com"
	wvTried bool
	wvTip   string
)

// webviewDemoHTML 本机示例页（离线也能看、能交互：输入 / 按钮 / 列表滚动）。
const webviewDemoHTML = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8">
<title>GoGUI · WebView 示例</title>
<style>
  :root{color-scheme:dark}
  *{box-sizing:border-box}
  body{margin:0;font:14px/1.6 'Microsoft YaHei UI',system-ui,sans-serif;background:#0b1120;color:#e6ecff}
  .hero{padding:28px 30px;background:linear-gradient(120deg,#6366f1,#8b5cf6 55%,#22d3ee)}
  .hero h1{margin:0;font-size:24px;letter-spacing:.5px}
  .hero p{margin:6px 0 0;opacity:.9;font-size:13px}
  .wrap{padding:22px 30px 40px;display:grid;gap:16px;grid-template-columns:repeat(auto-fit,minmax(260px,1fr))}
  .card{background:#111a2c;border:1px solid #1f2a41;border-radius:14px;padding:16px}
  .card h2{margin:0 0 10px;font-size:14px;color:#a5b4fc;letter-spacing:.4px}
  input[type=text]{width:100%;padding:10px 12px;border-radius:9px;border:1px solid #2a3550;background:#0c1424;color:#e6ecff;outline:none}
  input[type=text]:focus{border-color:#6366f1;box-shadow:0 0 0 3px rgba(99,102,241,.25)}
  button{margin-top:10px;padding:9px 14px;border:0;border-radius:9px;background:linear-gradient(90deg,#6366f1,#8b5cf6);color:#fff;font-weight:600;cursor:pointer}
  button:active{transform:translateY(1px)}
  .row{display:flex;align-items:center;gap:10px;margin-top:8px;color:#93a2c4}
  .pill{display:inline-block;padding:3px 9px;border-radius:999px;background:rgba(99,102,241,.18);color:#c7d2fe;font-size:12px}
  .list{max-height:150px;overflow:auto;margin:0;padding-left:18px;color:#93a2c4}
  a{color:#7dd3fc}
  .bar{height:8px;border-radius:999px;background:#1b2436;overflow:hidden;margin-top:12px}
  .bar i{display:block;height:100%;width:38%;background:linear-gradient(90deg,#22d3ee,#8b5cf6);animation:slide 3s ease-in-out infinite}
  @keyframes slide{0%,100%{margin-left:0}50%{margin-left:62%}}
</style></head><body>
  <div class="hero">
    <h1>GoGUI · WebView2</h1>
    <p>这段页面由系统 Edge 内核渲染，滚动、输入、按钮、动画都是浏览器原生的。</p>
  </div>
  <div class="wrap">
    <div class="card">
      <h2>真实交互验证</h2>
      <input id="name" type="text" placeholder="输入点东西…">
      <button onclick="greet()">点我</button>
      <div class="row"><span class="pill" id="out">等待操作</span></div>
    </div>
    <div class="card">
      <h2>滚动与链接</h2>
      <ul class="list">
        <li>第一行：鼠标滚轮可滚动</li><li>第二行：中文输入法直接可用</li>
        <li>第三行：右键菜单、复制粘贴原生</li><li>第四行：视频 / canvas 均走 GPU</li>
        <li>第五行：链接可点击 <a href="https://www.bing.com">Bing</a></li>
        <li>第六行：重新加载、前进后退由自绘工具栏控制</li>
        <li>第七行：滚动到这里说明列表可滚</li><li>第八行：结束</li>
      </ul>
      <div class="bar"><i></i></div>
    </div>
  </div>
  <script>
    function greet(){
      var v = document.getElementById('name').value;
      document.getElementById('out').textContent = v ? ('你好，' + v + '！') : '先输入内容再点我';
    }
    document.getElementById('name').addEventListener('keydown', function(e){ if(e.key === 'Enter') greet(); });
  </script>
</body></html>`

// webviewCase WebView 组件演示页：自绘工具栏 + 真实网页。
func webviewCase() *ui.Node {
	if !wvTried {
		wvTried = true
		if !ui.WebViewAvailable() {
			wvTip = "未检测到 WebView2 运行时，先安装 Microsoft Edge WebView2 Runtime"
		} else if wv, err := ui.NewWebView(ui.WebViewOptions{HTML: webviewDemoHTML}); err != nil {
			wvTip = "创建 WebView 失败：" + err.Error()
		} else {
			wvInst = wv
			wv.SetRadius(12)
		}
	}

	header := ui.Row(
		ui.Column(
			ui.Text("WebView 内嵌浏览器").St(ui.St().Text(19, 700, "#e8ecff")),
			ui.Text("系统 WebView2（Edge 内核）· 网页由 Chromium 的 GPU 进程渲染，交互与帧率都是原生的。").
				St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
		).Grow(1).St(ui.St().GapOf(4)),
		ui.Tag("A 阶段 · HWND 托管", "#7dd3fc"),
	).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter)

	var bar *ui.Node
	if wvInst != nil {
		bar = ui.WebViewBar(wvInst, ui.WebViewBarOptions{
			Address:   wvAddr,
			Key:       "gv-addr",
			OnAddress: func(v string) { wvAddr = v },
			OnGo:      func() { navigateWebView(wvAddr) },
			ShowTitle: true,
		})
	} else {
		bar = ui.Row(
			ui.Text("浏览器不可用").St(ui.St().Text(12, 600, "#f87171")),
		).St(ui.St().Padding(10).R(10).BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke))
	}

	links := ui.Row(
		ui.Btn("本机示例", ui.BtnSecondary, func() {
			if wvInst != nil {
				wvInst.LoadHTML(webviewDemoHTML)
				wvTip = "已加载本机示例页（离线可交互）"
			}
		}),
		ui.Btn("Bing", ui.BtnSecondary, func() { navigateWebView("https://www.bing.com") }),
		ui.Btn("MDN", ui.BtnSecondary, func() { navigateWebView("https://developer.mozilla.org/zh-CN/") }),
		ui.Btn("GitHub", ui.BtnSecondary, func() { navigateWebView("https://github.com") }),
		ui.Spacer(),
		ui.Text(wvTip).St(ui.St().Text(11, 400, ui.ColTextMuted).WrapOn()),
	).St(ui.St().GapOf(8)).AlignTo(ui.AlignCenter)

	status := ui.Box().St(ui.St().H(1))
	var area *ui.Node
	if wvInst != nil {
		status = ui.WebViewStatus(wvInst)
		if titleMenu.IsOpen() {
			// 菜单是自绘浮层，会被网页子窗口盖住，展开时先把网页临时隐藏
			area = ui.Center(
				ui.Column(
					ui.Text("菜单已展开 · 网页临时隐藏").St(ui.St().Text(13, 600, "#c7d2fe")),
					ui.Text("HWND 托管的网页永远在最上层，A 阶段无法被自绘浮层遮挡；升级到 DirectComposition 合成后可解除此限制。").
						St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn().TAlign(ui.AlignCenter)),
				).St(ui.St().GapOf(6)).AlignTo(ui.AlignCenter),
			).St(ui.St().GrowTo(1).R(12).BgC(ui.ColBgDeep).Stroke(1, ui.ColStroke))
		} else {
			area = ui.WebViewNode(wvInst).St(ui.St().GrowTo(1).R(12))
		}
	} else {
		area = ui.Center(
			ui.Column(
				ui.Text("WebView2 暂不可用").St(ui.St().Text(14, 600, "#f87171")),
				ui.Text(wvTip).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn().TAlign(ui.AlignCenter)),
			).St(ui.St().GapOf(6)).AlignTo(ui.AlignCenter),
		).St(ui.St().GrowTo(1).R(12).BgC(ui.ColBgDeep).Stroke(1, ui.ColStroke))
	}

	return ui.Column(header, bar, links, status, area).
		St(ui.St().Padding(18).GapOf(12).Hp(100))
}

// navigateWebView 地址栏 / 快捷入口统一走这里（补全协议 + 提示）。
func navigateWebView(url string) {
	if wvInst == nil {
		return
	}
	wvInst.Navigate(url)
	wvTip = "正在打开 " + url
	ui.RequestRedraw()
}

// 「表单与弹窗」案例：FormItem / Select / Textarea / NumberInput / SearchInput /
// Chip / DateBox + Alert / Confirm / Prompt / Modal / Drawer。
//
// 浮层（下拉、日历、弹窗）都在根 Stack 末尾渲染：formOverlays / SelectLayers / DateLayers。

// ---- 案例状态 ----

var (
	formName   = "极码 AI"
	formMail   = ""
	formBio    = ""
	formSearch = ""
	formExp    = float64(3)
	formDate   = ""
	formRole   = "dev"
	formTeam   = "core"

	formErrName = ""
	formErrMail = ""
	formSaved   = ""

	formTags = map[string]bool{"Go": true, "渲染": true, "组件库": false, "OpenGL": false}

	dlgAlert    bool
	dlgConfirm  bool
	dlgPrompt   bool
	dlgModal    bool
	dlgDrawer   bool
	dlgPromptID = "极码 AI"
)

var (
	roleSel = ui.NewSelect("form-role")
	teamSel = ui.NewSelect("form-team")
	dateCal = ui.NewDate("form-date")
)

var roleOpts = []ui.SelectOption{
	{Value: "dev", Label: "研发工程师", Hint: "R&D"},
	{Value: "design", Label: "设计师", Hint: "Design"},
	{Value: "pm", Label: "产品经理", Hint: "PM"},
	{Value: "ops", Label: "运维 / SRE", Hint: "Ops"},
	{Value: "guest", Label: "只读访客", Hint: "禁用示例", Disabled: true},
}

var teamOpts = []ui.SelectOption{
	{Value: "core", Label: "核心引擎组"},
	{Value: "infra", Label: "基础设施组"},
	{Value: "growth", Label: "增长组"},
	{Value: "lab", Label: "图形实验室"},
}

// ---- 页面 ----

func formCase() *ui.Node {
	return ui.Column(
		pageHead("表单与弹窗", "下拉框 / 多行输入 / 数字 / 日期 / 标签 / 校验，以及四类弹窗与抽屉"),

		ui.If(formSaved != "", func() *ui.Node {
			return ui.Banner("success", "操作完成", formSaved, func() {
				formSaved = ""
				ui.RequestRedraw()
			})
		}),

		ui.Card(
			ui.Title("基础信息", "含必填校验：保存时校验名称与邮箱"),
			ui.FormItemInline("名称", true, formErrName, ui.Box(
				ui.Input("form-name", formName, "请输入名称", func(v string) {
					formName = v
					if formErrName != "" {
						formErrName = ui.ValidateRequired("名称", v)
					}
				}),
			)),
			ui.FormItemInline("邮箱", true, formErrMail, ui.Box(
				ui.Input("form-mail", formMail, "name@example.com", func(v string) {
					formMail = v
					if formErrMail != "" {
						formErrMail = ui.ValidateEmail("邮箱", v)
					}
				}),
			)),
			ui.FormItemInline("角色", false, "", ui.SelectBox(roleSel, roleOpts, formRole, "请选择角色", func(v string) {
				formRole = v
			})),
			ui.FormItemInline("团队", false, "", ui.SelectBox(teamSel, teamOpts, formTeam, "请选择团队", func(v string) {
				formTeam = v
			})),
			ui.FormItemInline("工作年限", false, "", ui.NumberInput("form-exp", formExp, 0, 40, 1, func(v float64) {
				formExp = v
			})),
			ui.FormItemInline("入职日期", false, "", ui.DateBox(dateCal, formDate, func(v string) {
				formDate = v
			})),
			ui.FormItemInline("擅长方向", false, "", ui.Wrap(
				chipCell("Go"), chipCell("渲染"), chipCell("组件库"), chipCell("OpenGL"),
			).St(ui.St().GapOf(8))),
			ui.FormItemInline("简介", false, "", ui.Box(
				ui.Textarea("form-bio", formBio, "写点自我介绍…（支持换行，Enter 换行 / Esc 失焦）", 104, func(v string) {
					formBio = v
				}),
			)),
			ui.FormFooter(
				ui.Btn("重置", ui.BtnSecondary, func() {
					formName, formMail, formBio, formSearch = "极码 AI", "", "", ""
					formExp, formDate, formRole, formTeam = 3, "", "dev", "core"
					formErrName, formErrMail, formSaved = "", "", ""
					ui.RequestRedraw()
				}),
				ui.Btn("保存", ui.BtnPrimary, func() {
					formErrName = ui.ValidateRequired("名称", formName)
					formErrMail = ui.ValidateRequired("邮箱", formMail)
					if formErrMail == "" {
						formErrMail = ui.ValidateEmail("邮箱", formMail)
					}
					if formErrName == "" && formErrMail == "" {
						formSaved = fmt.Sprintf("已保存：%s · %s · %d 年 · %s", formName, roleLabel(formRole), int(formExp), formDate)
					}
					ui.RequestRedraw()
				}),
			),
		),

		ui.Card(
			ui.Title("搜索与标签", "SearchInput 带清除按钮；Chip 可切换选中"),
			ui.FormItem("", false, "", ui.SearchInput("form-search", formSearch, "搜索组件 / 成员…", func(v string) {
				formSearch = v
			})),
			ui.Row(
				ui.Text("多选："+tagSummary()).St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
				ui.Spacer(),
				ui.ChipRemovable("清空选择", func() {
					formTags = map[string]bool{}
					ui.RequestRedraw()
				}),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		),

		ui.Card(
			ui.Title("弹窗", "Alert / Confirm / Prompt / 自定义 Modal / 右侧抽屉"),
			ui.Row(
				ui.Btn("提示 Alert", ui.BtnSecondary, func() {
					dlgAlert = true
					ui.RequestRedraw()
				}),
				ui.Btn("确认 Confirm", ui.BtnSecondary, func() {
					dlgConfirm = true
					ui.RequestRedraw()
				}),
				ui.Btn("输入 Prompt", ui.BtnSecondary, func() {
					dlgPrompt = true
					ui.RequestRedraw()
				}),
				ui.Btn("自定义 Modal", ui.BtnPrimary, func() {
					dlgModal = true
					ui.RequestRedraw()
				}),
				ui.Btn("抽屉 Drawer", ui.BtnSuccess, func() {
					dlgDrawer = true
					ui.RequestRedraw()
				}),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
			ui.KeyValue("当前角色", roleLabel(formRole)),
			ui.KeyValue("入职日期", orDash(formDate)),
			ui.KeyValue("工作年限", strconv.Itoa(int(formExp))+" 年"),
			ui.KeyValue("简介字数", strconv.Itoa(len([]rune(formBio)))),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

// chipCell 单个可选标签。
func chipCell(name string) *ui.Node {
	on := formTags[name]
	return ui.Chip(name, on, func() {
		formTags[name] = !on
	})
}

// tagSummary 已选标签汇总文案。
func tagSummary() string {
	out := ""
	for _, k := range []string{"Go", "渲染", "组件库", "OpenGL"} {
		if formTags[k] {
			if out != "" {
				out += "、"
			}
			out += k
		}
	}
	if out == "" {
		return "未选择"
	}
	return out
}

// roleLabel 角色值转显示名。
func roleLabel(v string) string {
	for _, o := range roleOpts {
		if o.Value == v {
			return o.Label
		}
	}
	return "未选择"
}

func orDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

// formOverlays 表单案例的弹窗层（放在根 Stack 末尾）。关闭时返回 nil。
func formOverlays() *ui.Node {
	alert := ui.AlertDlg("dlg-alert", dlgAlert, ui.DlgSuccess, "保存成功",
		"配置已写入本地存储，下次启动自动生效。", func() {
			dlgAlert = false
			ui.RequestRedraw()
		})
	confirm := ui.ConfirmDlg("dlg-confirm", dlgConfirm, ui.DlgWarn, "删除确认",
		"删除后不可恢复，确定继续？", "删除", func() {
			dlgConfirm = false
			ui.RequestRedraw()
		}, func() {
			dlgConfirm = false
			formSaved = "已删除（示例，不影响真实数据）"
			ui.RequestRedraw()
		})
	prompt := ui.PromptDlg("dlg-prompt", dlgPrompt, ui.DlgInfo, "重命名",
		"新名称", dlgPromptID, "请输入名称", "保存", func(v string) {
			dlgPrompt = false
			if v != "" {
				formName = v
				dlgPromptID = v
				formSaved = "已重命名为 " + v
			}
			ui.RequestRedraw()
		}, func() {
			dlgPrompt = false
			ui.RequestRedraw()
		})
	modal := ui.ModalPanel(ui.DialogOpts{
		Key:   "dlg-custom",
		Open:  dlgModal,
		Kind:  ui.DlgInfo,
		Title: "表单快照",
		BodyNode: ui.Column(
			ui.Text("确认以下信息后提交：").St(ui.St().Text(12.5, 400, ui.ColTextDim)),
			ui.Box().St(ui.St().H(6)),
			ui.KeyValue("名称", formName),
			ui.KeyValue("邮箱", orDash(formMail)),
			ui.KeyValue("角色", roleLabel(formRole)),
			ui.KeyValue("年限", strconv.Itoa(int(formExp))+" 年"),
			ui.KeyValue("日期", orDash(formDate)),
			ui.KeyValue("标签", tagSummary()),
		).St(ui.St().GapOf(6)),
		Width:    440,
		Closable: true,
		OnClose: func() {
			dlgModal = false
			ui.RequestRedraw()
		},
		CancelLabel: "再改改",
		OKLabel:     "确认提交",
		OnCancel: func() {
			dlgModal = false
			ui.RequestRedraw()
		},
		OnOK: func() {
			dlgModal = false
			formSaved = "已提交表单快照"
			ui.RequestRedraw()
		},
		Blur: 6,
	})
	drawer := ui.Drawer(ui.DrawerOpts{
		Key:   "dlg-drawer",
		Open:  dlgDrawer,
		Title: "成员详情",
		Width: 400,
		OnClose: func() {
			dlgDrawer = false
			ui.RequestRedraw()
		},
		Body: ui.Column(
			ui.KeyValue("名称", formName),
			ui.KeyValue("角色", roleLabel(formRole)),
			ui.KeyValue("团队", teamLabel(formTeam)),
			ui.KeyValue("年限", strconv.Itoa(int(formExp))+" 年"),
			ui.KeyValue("入职", orDash(formDate)),
			ui.Divider(),
			ui.Text("抽屉内容可以放任意组件，外层自动纵向滚动。").St(ui.St().Text(12, 400, ui.ColTextMuted).WrapOn()),
		).St(ui.St().GapOf(10)),
	})
	layers := make([]*ui.Node, 0, 5)
	for _, n := range []*ui.Node{alert, confirm, prompt, modal, drawer} {
		if n != nil {
			layers = append(layers, n)
		}
	}
	if len(layers) == 0 {
		return nil
	}
	return ui.Stack(layers...).St(ui.St().AbsFill())
}

// teamLabel 团队值转显示名。
func teamLabel(v string) string {
	for _, o := range teamOpts {
		if o.Value == v {
			return o.Label
		}
	}
	return "未选择"
}
