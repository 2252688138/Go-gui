// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// gallery 桌面版与 Web 版共用**同一份界面代码**（本文件只是壳层入口）：
//
//	go run ./cmd/gallery                       → 桌面（Win32 / WGL）
//	GOOS=js GOARCH=wasm go build ./cmd/gallery → Web（canvas + WebGL 2）
//
// 各界面案例已拆分到 ./pages 子包（package pages），并按组件类型聚合为 8 个分类页：
//
//	pages/state.go          共享状态、页面路由（Content）、全局动作（SwitchCase / SetTheme）
//	pages/case_groups.go    分类页组合（基础 / 表单 / 列表 / 媒体 / 布局 / 图表 / 文本 / 系统）
//	pages/case_controls.go  基础控件          pages/case_settings.go  开关 / 单选 / 滑块
//	pages/case_form.go      表单与弹窗        pages/case_suite.go     高级组件卡片（按类型组合）
//	pages/case_table.go     数据表格          pages/case_chat.go      聊天列表
//	pages/case_lists.go     列表与表格页（页内页签：文件树 / 图文卡片 / 列表 / 社区帖子 / 分页）
//	pages/case_video.go     视频播放          pages/case_music.go     音频播放
//	pages/case_layouts.go   布局实验室        pages/case_dashboard.go 仪表盘
//	pages/case_charts.go    图表              pages/case_markdown.go  Markdown 渲染
//	pages/case_system.go    系统能力          pages/case_webview.go   WebView 浏览器
//
// 登录 / 注册等业务页面已移除：本程序只做组件演示，不再模拟登录流程。
//
// 没有单独的 web.go：Web 端的字体自举下沉在 ui 包（ui/boot_wasm.go），
// 因此 main() / Options / view() 在所有平台完全一致，改一处两端同步生效。
//
// 运行：go run ./cmd/gallery
// 布局调试：go run ./cmd/gallery --dump dump.txt --case table
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gogui/cmd/gallery/pages"
	"gogui/internal/version"
	"gogui/ui"
)

// bindGlobalKeysOnce 全局快捷键只需绑定一次（view 每帧都会调用）。
var bindGlobalKeysOnce sync.Once

// wantFullscreen 启动参数 --fullscreen：首帧后进全屏。
var (
	wantFullscreen bool
	fsApplied      bool
)

func main() {
	// 页面内的轻提示统一走组件库 Toast（pages 包通过钩子回调，不直接依赖 ui.ToastShow）。
	pages.Toast = ui.ToastShow

	// 多实例：不再做单实例互斥，连点运行 / 多次 go run 都会开出独立窗口，
	// 各自拥有独立的页面状态与滚动位置。


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
	applyArgs(os.Args[1:], &opts)
	// Web 端用 URL 参数指定初始页面：?case=video / ?case=webview
	// （桌面端没有 URL，StartupQuery 恒返回空串）
	if c := ui.StartupQuery("case"); c != "" {
		pages.Current = c
	}
	// Web 端可直接指定要播放的媒体直链：?media=https://…/demo.mp4
	if u := ui.StartupQuery("media"); u != "" {
		pages.SetVideoURL(u)
		pages.Current = "video"
	}
	// 浏览器不支持 HLS（.m3u8 需要 hls.js），Web 端默认换成 mp4 示例
	if ui.BackendName() == "browser" && strings.HasSuffix(strings.ToLower(pages.VideoURL()), ".m3u8") {
		pages.SetVideoURL("https://www.w3schools.com/html/mov_bbb.mp4")
	}
	if err := ui.Run(opts, view); err != nil {
		fmt.Println("fatal:", err)
	}
}

// applyArgs 解析命令行参数：首个实例启动时用，二次启动转发参数时也用（opts 传 nil）。
func applyArgs(args []string, opts *ui.Options) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dump":
			if opts != nil && i+1 < len(args) {
				i++
				opts.DumpLayout = args[i]
			}
		case "--md-tab":
			// 调试用：Markdown 示例页默认页签（gfm / ext / code / long）
			if i+1 < len(args) {
				i++
				pages.MdTab = args[i]
			}
		case "--case":
			if i+1 < len(args) {
				i++
				pages.Current = args[i]
			}
		case "--fullscreen":
			// 启动即全屏（调试 / 工控机 / 展台场景，也用于验证全屏渲染）
			wantFullscreen = true
		case "--menu":
			// 调试用：启动即展开指定顶级菜单（索引从 0 开始）
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					pages.TitleMenu.Open = n
				}
			}
		case "--sub":
			// 调试用：展开子菜单路径，例如 --sub 0,1
			if i+1 < len(args) {
				i++
				for _, p := range strings.Split(args[i], ",") {
					if n, err := strconv.Atoi(p); err == nil {
						pages.TitleMenu.Sub = append(pages.TitleMenu.Sub, n)
					}
				}
			}
		}
	}
}

// GOGUI_FPS=1 时输出 view() 各段构建耗时（定位每帧 50ms 的构建热点）。
var (
	profView = os.Getenv("GOGUI_FPS") != ""
	profT    = time.Now()
)

func viewMark(name string) {
	if !profView {
		return
	}
	now := time.Now()
	fmt.Fprintf(os.Stderr, "[view] %-10s %6.2fms\n", name, float64(now.Sub(profT).Microseconds())/1000)
	profT = now
}

func view() *ui.Node {
	profT = time.Now()
	// 全局快捷键只绑定一次：菜单里的「全屏切换」提示为 F11，这里真正接上
	bindGlobalKeysOnce.Do(func() {
		ui.OnKeyGlobal(func(k uint32) {
			if k == ui.VK_F11 {
				toggleFullscreen()
			}
		})
	})
	// --fullscreen：首帧后请求全屏（必须在窗口建好后调用）
	if wantFullscreen && !fsApplied {
		fsApplied = true
		if a := ui.Current(); a != nil {
			a.SetFullscreen(true)
			// 播放器是懒创建的，可能还是 nil；写之前判空（旧代码直接写 → panic 黑屏）。
			pages.SetVideoFullscreen(true)
			ui.RequestRedraw()
		}
	}
	nav := []*ui.Node{
		ui.Text("界面案例").St(ui.St().Text(19, 700, "#e8ecff")),
		ui.Text(fmt.Sprintf("%d 个界面骨架 + 布局实验室", len(pages.Cases))).St(ui.St().Text(11.5, 400, ui.ColTextMuted).PadOnly(0, 0, 18, 0)),
	}
	for i, c := range pages.Cases {
		id := c.ID
		labelColor := ui.ColTextDim
		if pages.Current == id {
			labelColor = ui.ColText
		}
		// 用色块代替字符图标：避免字体缺字形导致列表观感异常
		dotColor := [...]string{"#818cf8", "#22d3ee", "#f472b6", "#34d399", "#fbbf24"}[i%5]
		row := ui.Row(
			ui.Box().St(ui.St().W(6).H(6).R(2).BgC(dotColor)),
			ui.Text(c.Label).St(ui.St().Text(13, 500, labelColor)),
		).St(ui.St().GapOf(10).PadXY(12, 9).R(9).W(196))
		if pages.Current == id {
			row.St(ui.St().BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke))
		}
		nav = append(nav, row.K("case-"+id).Tap(func() {
			pages.SwitchCase(id)
		}).AlignTo(ui.AlignCenter))
	}
	nav = append(nav, ui.Spacer(),
		ui.Text(pages.CaseDesc()).St(ui.St().Text(11, 400, ui.ColTextMuted).WrapOn()))

	body := ui.Row(
		ui.Column(nav...).St(ui.St().W(232).Hp(100).Padding(18).BgC(ui.ColBgSoft).GapOf(4)),
		pages.Content(),
	).St(ui.St().GrowTo(1).BgC(ui.ColBg))

	viewMark("body")
	// 标题栏（菜单栏）单独作为最上层浮层：
	// 先用等高占位撑出内容区，再把标题栏放到 Stack 靠后的位置绘制，
	// 这样无论内容怎么滚动/溢出，顶部菜单栏永远压在它们之上（不会被组件遮挡）。
	shell := ui.Column(
		ui.Box().St(ui.St().H(titleBarH+1)), // 占位：标题栏 + 下方分隔线
		body,
	).St(ui.St().Hp(100).BgC(ui.ColBg))

	titleLayer := titleBar().St(ui.St().AbsTL(0, 0).Wp(100))
	viewMark("titleBar")

	// 播放器全屏：在同一窗口根 Stack 里铺满，不改系统窗口（实现在 pages/case_video.go）
	videoLayer := pages.VideoLayer()
	menuPanelLayer := ui.If(pages.CtrlMenu, func() *ui.Node {
		return ui.MenuPanel(690, 150, []ui.MenuItem{
			{Label: "新建文件", Hint: "Ctrl+N", OnPick: func() { pages.CtrlMenu = false; pages.Toast("新建文件") }},
			{Label: "打开项目", Hint: "Ctrl+O", OnPick: func() { pages.CtrlMenu = false; pages.Toast("打开项目") }},
			{Label: "导出为", Children: []ui.MenuItem{
				{Label: "PNG 图片", OnPick: func() { pages.CtrlMenu = false; pages.Toast("已导出 PNG") }},
				{Label: "PDF 文档", Hint: "Beta", OnPick: func() { pages.CtrlMenu = false; pages.Toast("已导出 PDF") }},
			}},
			{Separator: true},
			{Label: "更多设置…", Disabled: true},
		}, 180)
	})
	viewMark("panels")

	// 顶部标题栏菜单的下拉浮层（含多级子菜单）
	menuLayer := ui.MenuLayer(pages.TitleMenu, titleMenus(), titleMenuX(), titleBarH)
	viewMark("menuLayer")
	tipLayer := ui.TooltipLayer()
	selectLayer := ui.SelectLayers()
	dateLayer := ui.DateLayers()
	// 表单案例的弹窗：Alert / Confirm / Prompt / 自定义 Modal / Drawer
	formLayer := pages.FormOverlays()
	// Markdown 图片放大层（点击 markdown 中的图片后全屏预览）
	mdLayer := ui.MarkdownLightbox()
	// 组件套件页浮层（命令面板 / 自动完成 / 时间 / 级联 / 取色 / 引导 / 通知）
	suiteLayer := pages.SuiteLayers()
	toastLayer := ui.ToastLayer()
	viewMark("layers")

	return ui.Stack(shell, titleLayer, videoLayer, menuPanelLayer,
		menuLayer, tipLayer,
		selectLayer, dateLayer, formLayer, mdLayer, suiteLayer, toastLayer,
	).St(ui.St().Hp(100))
}

// ---- 顶部标题栏（自绘：品牌 + 菜单栏 + 窗口按钮 + 拖动）----

const titleBarH float32 = 40

// brandWidth 品牌区宽度（与 Row 内实际布局一致，用于菜单浮层定位）。
func brandWidth() float32 { return 18 + 8 + ui.TextWidth("GoGUI", 12.5, 600) }

// titleMenuX 菜单栏左边缘的窗口坐标（标题栏左内边距 10 + 品牌区 + 间距 14）。
func titleMenuX() float32 { return 10 + brandWidth() + 14 }

func titleBar() *ui.Node {
	// dragTitle 按下标题栏空白/品牌区时交给系统拖动窗口
	dragTitle := func() {
		pages.TitleMenu.Close()
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
		ui.MenuBar(pages.TitleMenu, titleMenus()),
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
				{Label: "图片…", OnPick: func() { pages.Toast("打开图片（示例）") }},
				{Label: "视频…", OnPick: func() { pages.SwitchCase("video") }},
				{Label: "音频…", OnPick: func() { pages.SwitchCase("music") }},
			}},
			{Label: "新建窗口", Hint: "Ctrl+N", OnPick: openChildWindow},
			{Label: "关闭当前窗口", Hint: "Alt+F4", OnPick: closeWindow},
			{Separator: true},
			{Label: "退出", OnPick: closeWindow},
		}},
		{Label: "编辑", Items: []ui.MenuItem{
			{Label: "复制当前页面名", Hint: "Ctrl+C", OnPick: func() { pages.Toast("已复制：" + pages.Current) }},
			{Label: "粘贴", Hint: "Ctrl+V", Disabled: true},
			{Separator: true},
			{Label: "查找", Children: []ui.MenuItem{
				{Label: "当前页面内查找…", Hint: "Ctrl+F", OnPick: func() { pages.Toast("查找（示例）") }},
				{Label: "全局搜索…", Hint: "Ctrl+Shift+F", OnPick: func() { pages.Toast("全局搜索（示例）") }},
			}},
		}},
		{Label: "视图", Items: []ui.MenuItem{
			{Label: "界面案例", Children: caseMenuItems()},
			{Separator: true},
			{Label: "全屏切换", Hint: "F11", OnPick: toggleFullscreen},
			{Label: "主题", Children: []ui.MenuItem{
				{Label: "深色", Checked: pages.Theme == "深色", OnPick: func() { pages.SetTheme("深色") }},
				{Label: "浅色", Checked: pages.Theme == "浅色", OnPick: func() { pages.SetTheme("浅色") }},
				{Label: "跟随系统", Checked: pages.Theme == "跟随系统", OnPick: func() { pages.SetTheme("跟随系统") }},
			}},
		}},
		{Label: "自定义", Items: []ui.MenuItem{
			{Label: "二级菜单演示", Children: []ui.MenuItem{
				{Label: "直接执行项", Hint: "Enter", OnPick: func() { pages.Toast("二级菜单项") }},
				{Label: "三级菜单演示", Children: []ui.MenuItem{
					{Label: "三级项 A", OnPick: func() { pages.Toast("三级菜单项 A") }},
					{Label: "三级项 B", OnPick: func() { pages.Toast("三级菜单项 B") }},
					{Separator: true},
					{Label: "禁用项", Disabled: true},
				}},
				{Separator: true},
				{Label: "带快捷键的项", Hint: "Ctrl+K", OnPick: func() { pages.Toast("自定义菜单项") }},
			}},
			{Separator: true},
			{Label: "当前页面：" + pages.Current, Disabled: true},
		}},
		{Label: "帮助", Items: []ui.MenuItem{
			{Label: "菜单用法", Children: []ui.MenuItem{
				{Label: "MenuBar / MenuLayer", OnPick: func() { pages.SwitchCase("controls") }},
				{Label: "MenuItem.Children 嵌套", OnPick: func() { pages.Toast("Children 非空即为子菜单，可多层嵌套") }},
			}},
			{Label: "关于 GoGUI", OnPick: func() { pages.Toast(version.Full() + " · 自绘标题栏与多级菜单示例") }},
		}},
	}
}

// caseMenuItems 把示例列表映射成菜单项（演示动态生成的子菜单）。
func caseMenuItems() []ui.MenuItem {
	out := make([]ui.MenuItem, 0, len(pages.Cases))
	for _, c := range pages.Cases {
		id := c.ID
		out = append(out, ui.MenuItem{
			Label:   c.Label,
			Checked: pages.Current == id,
			OnPick:  func() { pages.SwitchCase(id) },
		})
	}
	return out
}

func toggleFullscreen() {
	a := ui.Current()
	if a == nil {
		return
	}
	on := !a.IsFullscreen()
	// 播放器懒创建：未打开过视频时为 nil，直接赋值会 panic（曾导致「全屏即黑屏」）
	pages.SetVideoFullscreen(on)
	a.SetFullscreen(on)
	ui.RequestRedraw()
}

func closeWindow() {
	if a := ui.Current(); a != nil {
		a.Close()
	}
}

func openChildWindow() {
	if _, err := ui.NewWindow(ui.Options{Title: "子窗口 · GoGUI", Width: 520, Height: 360}, simpleChildView); err != nil {
		pages.Toast("打开子窗口失败：" + err.Error())
		return
	}
	pages.Toast("已打开子窗口")
}

func simpleChildView() *ui.Node {
	return ui.Center(ui.Column(
		ui.Text("子窗口").St(ui.St().Text(20, 700, ui.ColText)),
		ui.Text("从顶部标题栏菜单「文件 → 新建窗口」打开。").St(ui.St().Text(12, 400, ui.ColTextMuted)),
	).St(ui.St().GapOf(8)))
}
