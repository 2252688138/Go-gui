//go:build windows

package main

import (
	"fmt"
	"os"
	"strconv"

	"gogui/ui"
)

// current 当前选中页面（点击后通过 ui.RequestRedraw 触发重绘）。
var current = "dashboard"

// 组件库交互状态（声明式重建，状态由业务持有）。
var (
	chkA, chkB         = true, false
	radioPick          = "dark"
	switchOn           = true
	sliderV    float32 = 0.42
	progressV  float32 = 0.66
	inputVal           = ""
	tabPick            = "all"
	accOpen            = true
	modalOpen          = false
	toastMsg           = ""
)

type page struct {
	id, label string
}

var pages = []page{
	{"dashboard", "仪表盘"},
	{"components", "组件库"},
	{"theme", "主题"},
	{"settings", "设置"},
}

func main() {
	opts := ui.Options{
		Title:      "GoGUI · 自绘现代 UI",
		Width:      1180,
		Height:     760,
		FontFamily: "Microsoft YaHei UI",
		Background: ui.ColBg,
	}
	if len(os.Args) > 2 && os.Args[1] == "--dump" {
		opts.DumpLayout = os.Args[2]
	}
	// --page components 指定首屏页面（调试布局用）
	if len(os.Args) > 4 && os.Args[3] == "--page" {
		current = os.Args[4]
	}
	if err := ui.Run(opts, view); err != nil {
		fmt.Println("fatal:", err)
	}
}

func view() *ui.Node {
	nav := make([]*ui.Node, 0, len(pages)+3)
	nav = append(nav,
		ui.Text("GoGUI").St(ui.St().Text(20, 700, "#e8ecff").PadOnly(0, 0, 4, 0)),
		ui.Text("自绘 GUI 框架").St(ui.St().Text(12, 400, "#5b6480").PadOnly(0, 0, 24, 0)),
	)
	for _, p := range pages {
		id := p.id
		nav = append(nav, ui.Button(p.label, func() {
			current = id
			ui.RequestRedraw()
		}).St(navSt(current == id)).K("nav-"+id))
	}
	nav = append(nav, ui.Spacer(),
		ui.Text("v0.2 · 组件库").St(ui.St().Text(11, 400, "#3d4459")))

	body := ui.Row(
		ui.Column(nav...).St(ui.St().W(220).Hp(100).Padding(20).BgC(ui.ColBgSoft).GapOf(6)),
		ui.Scroll(mainContent()).St(ui.St().GrowTo(1)).K("main-scroll"),
	).St(ui.St().Hp(100).BgC(ui.ColBg))

	// 叠层：模态框与 Toast 覆盖在内容之上
	return ui.Stack(body, ui.Modal(modalOpen, "确认操作", "模态框基于 Stack + 绝对定位实现，遮罩层拦截点击，卡片在容器内居中。", func() {
		modalOpen = false
		toastMsg = "已取消"
		ui.RequestRedraw()
	}), ui.Toast(toastMsg)).St(ui.St().Hp(100))
}

func mainContent() *ui.Node {
	switch current {
	case "components":
		return componentsPage()

	case "theme":
		return ui.Column(
			header("主题", "深色 / 浅色配色 token（演示：颜色与圆角全部数据驱动）"),
			ui.Row(
				themeCard("深色", "#0b0f19", ui.ColBg, "#6366f1"),
				themeCard("浅色", "#f8fafc", "#ffffff", "#4f46e5"),
				themeCard("森林", "#0c1a14", "#132a20", "#10b981"),
			).St(ui.St().GapOf(16)),
			ui.Card(
				ui.Title("组件色板", "统一 token 便于主题切换"),
				ui.Row(
					ui.Tag("primary", "#a5b4fc"),
					ui.Tag("success", "#6ee7b7"),
					ui.Tag("warning", "#fcd34d"),
					ui.Tag("danger", "#fca5a5"),
				).St(ui.St().GapOf(8)),
				ui.Progress(progressV, ui.ColPrimary),
			),
		).St(ui.St().Padding(28).GapOf(20))

	case "settings":
		rows := []struct{ t, s, c string }{
			{"常规设置", "语言 / 启动项 / 默认目录", "#38bdf8"},
			{"外观", "主题、圆角与密度", "#a78bfa"},
			{"快捷键", "自定义键盘映射", "#34d399"},
			{"关于", "版本 0.2.0 · OpenGL 3.3", "#fbbf24"},
		}
		items := make([]*ui.Node, 0, len(rows))
		for _, r := range rows {
			items = append(items, ui.ListItem("set-"+r.t, r.t, r.s, r.c, func() {
				toastMsg = r.t + " 已打开"
				ui.RequestRedraw()
			}))
		}
		return ui.Column(append([]*ui.Node{header("设置", "列表项可点击，滚动内容超出时滚轮生效")}, items...)...).
			St(ui.St().Padding(28).GapOf(12))
	}

	return ui.Column(
		ui.Row(
			ui.Column(
				ui.Text("控制台").St(ui.St().Text(26, 700, "#eef2ff")),
				ui.Text("零 cgo · 纯 Go 自绘 · SDF 抗锯齿 · 跨平台抽象").St(ui.St().Text(13, 400, "#6b7694").PadOnly(6, 0, 0, 0)),
			).Grow(1),
			ui.Button("打开组件库", func() {
				current = "components"
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 600, "#ffffff").PadXY(18, 9).R(10).
				BgGrad("#6366f1", "#8b5cf6", 90).Shadow(18, 6, 0, "rgba(99,102,241,0.45)")),
		).St(ui.St().PadOnly(0, 0, 24, 0)),

		ui.Row(
			statCard("渲染帧率", "120", "fps", "#22d3ee"),
			statCard("绘制指令", "128", "draws", "#a78bfa"),
			statCard("内存占用", "12.4", "MB", "#34d399"),
		).St(ui.St().GapOf(16)),

		ui.Card(
			ui.Title("样式能力", "全部由片元着色器与批处理实现"),
			ui.Row(
				ui.Tag("圆角 SDF", "#38bdf8"),
				ui.Tag("线性渐变", "#a78bfa"),
				ui.Tag("投影", "#f472b6"),
				ui.Tag("Flexbox", "#34d399"),
				ui.Tag("Stack 叠层", "#fbbf24"),
			).St(ui.St().GapOf(10)),
		),
	).St(ui.St().Padding(28).GapOf(20))
}

func componentsPage() *ui.Node {
	// 表单控件
	form := ui.Card(
		ui.Title("表单控件", "点击即可切换；输入框点击获取焦点后可直接键入"),
		ui.Row(
			ui.Checkbox("chk-a", "启用自动更新", chkA, func(v bool) { chkA = v }),
			ui.Checkbox("chk-b", "发送匿名统计", chkB, func(v bool) { chkB = v }),
		).St(ui.St().GapOf(24)),
		ui.Divider(),
		ui.Row(
			ui.Text("主题").St(ui.St().Text(12.5, 500, ui.ColTextMuted).W(48)),
			ui.Radio("r-dark", "深色", radioPick == "dark", func() { radioPick = "dark" }),
			ui.Radio("r-light", "浅色", radioPick == "light", func() { radioPick = "light" }),
			ui.Radio("r-auto", "跟随系统", radioPick == "auto", func() { radioPick = "auto" }),
		).St(ui.St().GapOf(20)).AlignTo(ui.AlignCenter),
		ui.Divider(),
		ui.Row(
			ui.Text("毛玻璃效果").St(ui.St().Text(13, 500, ui.ColText)).Grow(1),
			ui.Switch("sw-glass", switchOn, func(v bool) { switchOn = v }),
		).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("缩放比例").St(ui.St().Text(13, 500, ui.ColText)).St(ui.St().W(90)),
			ui.Slider("sl-scale", sliderV, func(v float32) { sliderV = v }),
			ui.Text(strconv.Itoa(int(sliderV*100))+"%").St(ui.St().Text(12, 500, "#a5b4fc").W(44)),
		).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("渲染进度").St(ui.St().Text(13, 500, ui.ColText)).St(ui.St().W(90)),
			ui.Progress(progressV, ui.ColPrimary).Grow(1),
			ui.Text(strconv.Itoa(int(progressV*100))+"%").St(ui.St().Text(12, 500, "#a5b4fc").W(44)),
		).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
		ui.Input("in-name", inputVal, "请输入名称，回车无效，直接键入即可…", func(v string) { inputVal = v }),
	)

	// 展示组件
	show := ui.Card(
		ui.Title("展示组件", "标签、头像、角标、骨架屏与加载动画"),
		ui.Row(
			ui.Tag("默认", "#c7d2fe"),
			ui.Tag("成功", "#6ee7b7"),
			ui.Tag("警告", "#fcd34d"),
			ui.Tag("危险", "#fca5a5"),
		).St(ui.St().GapOf(8)),
		ui.Row(
			ui.Stack(ui.Avatar("极", "#6366f1"), ui.Badge(3)).St(ui.St().W(40).H(40)),
			ui.Stack(ui.Avatar("码", "#0ea5e9"), ui.Badge(12)).St(ui.St().W(40).H(40)),
			ui.Column(
				ui.Text("极码 AI").St(ui.St().Text(13, 600, ui.ColText)),
				ui.Row(ui.Dot("#34d399", 7), ui.Text("在线").St(ui.St().Text(11.5, 400, "#6ee7b7"))).St(ui.St().GapOf(6)),
			).St(ui.St().GapOf(4)),
			ui.Spacer(),
			ui.Spinner(22, ui.ColPrimary),
		).St(ui.St().GapOf(16)).AlignTo(ui.AlignCenter),
		ui.Divider(),
		ui.Text("骨架屏（逐帧呼吸动画）").St(ui.St().Text(12, 500, ui.ColTextMuted)),
		ui.Row(
			ui.Skeleton(44, 44),
			ui.Column(
				ui.Skeleton(180, 12),
				ui.Skeleton(120, 12),
			).St(ui.St().GapOf(8)),
		).St(ui.St().GapOf(12)),
	)

	// 反馈与浮层
	feed := ui.Card(
		ui.Title("交互组件", "页签、折叠面板、模态框与轻提示"),
		ui.Tabs([]string{"all", "ui", "dev"}, []string{"全部", "界面", "开发"}, tabPick, func(k string) { tabPick = k }),
		ui.Accordion("acc-1", "什么是 Stack 叠层？", "Stack 将子节点叠放在同一区域，配合 AbsFill / AbsTL / AbsCenter 等绝对定位样式，可实现开关旋钮、进度条、模态框与 Toast。", accOpen, func() { accOpen = !accOpen }),
		ui.Accordion("acc-2", "如何接入动画？", "给节点设置 .St(St().AnimOn()) 后，界面会在有动画节点时逐帧重绘，配合 ui.AnimTime() 做插值。", false, func() {}),
		ui.Row(
			ui.Button("打开模态框", func() {
				modalOpen = true
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 600, "#ffffff").PadXY(16, 8).R(9).BgGrad(ui.ColPrimary, ui.ColPrimary2, 90)),
			ui.Button("显示 Toast", func() {
				toastMsg = "操作成功 · " + strconv.Itoa(int(sliderV*100)) + "%"
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 500, "#c7d2fe").PadXY(16, 8).R(9).BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke)),
			ui.Button("清空提示", func() {
				toastMsg = ""
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 500, ui.ColTextMuted).PadXY(16, 8).R(9)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
	)

	return ui.Column(
		header("组件库", "表单 / 展示 / 交互三类组件，全部零依赖、纯 Go 自绘"),
		form, show, feed,
	).St(ui.St().Padding(28).GapOf(18))
}

func header(title, sub string) *ui.Node {
	return ui.Column(
		ui.Text(title).St(ui.St().Text(26, 700, "#eef2ff")),
		ui.Text(sub).St(ui.St().Text(13, 400, "#6b7694").PadOnly(6, 0, 0, 0)),
	)
}

func navSt(active bool) *ui.Style {
	s := ui.St().Text(13, 500, "#8b93ad").PadXY(14, 10).R(10).W(180)
	if active {
		s.Text(13, 600, "#ffffff").BgGrad(ui.ColBgSoft, ui.ColBgSoft, 0).Stroke(1, ui.ColStroke)
	}
	return s
}

func statCard(title, value, unit, accent string) *ui.Node {
	return ui.Column(
		ui.Text(title).St(ui.St().Text(12, 500, "#6b7694")),
		ui.Row(
			ui.Text(value).St(ui.St().Text(30, 700, "#f1f5ff")),
			ui.Text(unit).St(ui.St().Text(12, 500, accent).PadOnly(0, 0, 6, 6)),
		).St(ui.St().PadOnly(4, 0, 0, 0)),
	).St(ui.St().GrowTo(1).FlexBasis(1).Padding(20).R(16).BgC(ui.ColBg).Stroke(1, ui.ColStroke).
		Shadow(24, 8, 0, "rgba(0,0,0,0.35)"))
}

func themeCard(name, bg, card, accent string) *ui.Node {
	return ui.Column(
		ui.Text(name).St(ui.St().Text(14, 600, "#dfe5ff")),
		ui.Box().St(ui.St().Wp(100).H(72).R(10).BgC(bg).Stroke(1, "#334155")),
		ui.Box().St(ui.St().Wp(100).H(40).R(10).BgC(card).Stroke(1, "#334155")),
		ui.Box().St(ui.St().W(48).H(20).R(6).BgC(accent)),
	).St(ui.St().GrowTo(1).FlexBasis(1).Padding(16).R(16).BgC(ui.ColBgDeep).Stroke(1, ui.ColStroke).GapOf(10))
}
