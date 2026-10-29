//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// GoGUI 浏览器版 gallery（与 main.go 同一目录、同一入口，靠 build tag 分流）：
//
//   GOOS=js GOARCH=wasm go build ./cmd/gallery   -> Web 版
//   go build ./cmd/gallery                        -> 桌面版
//
// 与桌面版共用 gogui/ui（同一套 Node 树、布局、样式、图表、表单与命中/滚动逻辑），
// 只把窗口层换成 <canvas> + WebGL 2。
package main

import (
	"fmt"
	"math"
	"runtime/debug"
	"syscall/js"

	"gogui/ui"
)

var (
	page           = "overview"
	input          = ""
	chk            = true
	slider float32 = 0.4
)

// stage 记录启动阶段（浏览器冒烟测试用）。
func stage(s string) { js.Global().Set("__goguiStage", s) }

// main 取回字体 -> 注册 -> 运行同一套 UI。
// 注意：ui.Run 必须在 main goroutine 上跑（内部会跑消息/帧循环），
// 所以字体回调只负责投递结果，真正的运行等 <-done 之后再开始。
func main() {
	stage("start")
	done := make(chan bool, 1)
	loadFont("font.ttf", "GoGUIWeb", false, func(ok bool) { done <- ok })
	if !<-done {
		showError("字体加载失败：需要同目录 font.ttf")
	}
	stage("font")

	// 可选粗体：同目录放 font-bold.ttf 即自动启用（缺省时退回常规字重）
	boldDone := make(chan bool, 1)
	loadFont("font-bold.ttf", "GoGUIWeb", true, func(ok bool) { boldDone <- ok })
	<-boldDone

	defer func() {
		if r := recover(); r != nil {
			showError(fmt.Sprintf("panic: %v\n%s", r, debug.Stack()))
			stage("panic")
		}
	}()
	if err := ui.Run(ui.Options{
		Title:      "GoGUI · Web",
		Width:      1200,
		Height:     800,
		FontFamily: "GoGUIWeb",
	}, view); err != nil {
		showError(err.Error())
	}
	stage("running")
}

// loadFont 用 fetch 取字体后注册（bold=true 注册为 "<name>/Bold" 字重）。
func loadFont(url, name string, bold bool, done func(bool)) {
	onBody := js.FuncOf(func(this js.Value, args []js.Value) any {
		u8 := js.Global().Get("Uint8Array").New(args[0])
		buf := make([]byte, u8.Length())
		js.CopyBytesToGo(buf, u8)
		if bold {
			ui.RegisterFontBold(name, buf)
		} else {
			ui.RegisterFont(name, buf)
		}
		done(true)
		return nil
	})
	onResp := js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			done(false)
			return nil
		}
		resp.Call("arrayBuffer").Call("then", onBody)
		return nil
	})
	onErr := js.FuncOf(func(this js.Value, args []js.Value) any {
		done(false)
		return nil
	})
	js.Global().Call("fetch", url).Call("then", onResp).Call("catch", onErr)
}

func showError(msg string) {
	doc := js.Global().Get("document")
	if el := doc.Call("getElementById", "err"); !el.IsNull() && !el.IsUndefined() {
		el.Set("textContent", msg)
	}
	js.Global().Get("console").Call("error", msg)
}

// view 整棵界面树（每帧重建，与桌面端完全一致的写法）。
func view() *ui.Node {
	nav := []*ui.Node{
		ui.Text("GoGUI").St(ui.St().Text(20, 700, "#e8ecff").PadOnly(0, 0, 4, 0)),
		ui.Text("一套 UI 代码 · 桌面 / Web 同源").St(ui.St().Text(12, 400, "#5b6480").PadOnly(0, 0, 24, 0)),
	}
	for _, p := range [][2]string{{"overview", "概览"}, {"widgets", "组件"}, {"charts", "图表"}, {"form", "表单"}} {
		id, label := p[0], p[1]
		nav = append(nav, ui.Button(label, func() {
			page = id
			ui.RequestRedraw()
		}).K("nav-" + id).St(navSt(page == id)))
	}
	nav = append(nav, ui.Spacer(), ui.Text("WebGL 2 · wasm").St(ui.St().Text(11, 400, "#3d4459")))

	body := ui.Row(
		ui.Column(nav...).St(ui.St().W(210).Hp(100).Padding(20).BgC(ui.ColBgSoft).GapOf(6)),
		ui.Scroll(content()).St(ui.St().GrowTo(1)).K("main-scroll"),
	).St(ui.St().Hp(100).BgC(ui.ColBg))

	return ui.Stack(body, ui.ToastLayer()).St(ui.St().Hp(100))
}

func header(title, sub string) *ui.Node {
	return ui.Column(
		ui.Text(title).St(ui.St().Text(26, 700, "#eef2ff")),
		ui.Text(sub).St(ui.St().Text(13, 400, "#6b7694").PadOnly(6, 0, 0, 0)),
	)
}

func navSt(active bool) *ui.Style {
	s := ui.St().Text(13, 500, "#8b93ad").PadXY(14, 10).R(10).W(170)
	if active {
		s.Text(13, 600, "#ffffff").BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke)
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

func content() *ui.Node {
	switch page {
	case "widgets":
		return widgetsPage()
	case "charts":
		return chartsPage()
	case "form":
		return formPage()
	}
	return overviewPage()
}

func overviewPage() *ui.Node {
	return ui.Column(
		header("概览", "同一份布局 / 样式 / 图表代码，桌面走 Win32 + OpenGL，浏览器走 canvas + WebGL 2"),
		ui.Row(
			statCard("渲染后端", "WebGL", "2", "#22d3ee"),
			statCard("字体光栅", "sfnt", "atlas", "#a78bfa"),
			statCard("控件", "24", "widgets", "#34d399"),
		).St(ui.St().GapOf(16)),
		ui.Card(
			ui.Title("已实现能力", "全部由 ui 包在同一套绘制指令上实现"),
			ui.Row(
				ui.Tag("圆角 SDF", "#38bdf8"),
				ui.Tag("渐变", "#a78bfa"),
				ui.Tag("投影", "#f472b6"),
				ui.Tag("滚动裁剪", "#34d399"),
				ui.Tag("图表", "#fbbf24"),
			).St(ui.St().GapOf(10)),
		),
	).St(ui.St().Padding(28).GapOf(20))
}

func widgetsPage() *ui.Node {
	form := ui.Card(
		ui.Title("表单控件", "点击即可切换；输入框获得焦点后可直接键入"),
		ui.Row(
			ui.Checkbox("chk-a", "启用毛玻璃", chk, func(v bool) { chk = v }),
			ui.Tag("wasm", "#6ee7b7"),
			ui.Tag("WebGL 2", "#c7d2fe"),
		).St(ui.St().GapOf(20)).AlignTo(ui.AlignCenter),
		ui.Divider(),
		ui.Row(
			ui.Text("缩放").St(ui.St().Text(13, 500, ui.ColText)).St(ui.St().W(60)),
			ui.Slider("sl", slider, func(v float32) { slider = v }),
		).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
		ui.Input("in-name", input, "请输入名称，直接键入即可…", func(v string) { input = v }),
	)

	show := ui.Card(
		ui.Title("展示组件", "标签、头像、角标、骨架屏与加载动画"),
		ui.Row(
			ui.Stack(ui.Avatar("极", "#6366f1"), ui.Badge(3)).St(ui.St().W(40).H(40)),
			ui.Column(
				ui.Text("极码 AI").St(ui.St().Text(13, 600, ui.ColText)),
				ui.Row(ui.Dot("#34d399", 7), ui.Text("在线").St(ui.St().Text(11.5, 400, "#6ee7b7"))).St(ui.St().GapOf(6)),
			).St(ui.St().GapOf(4)),
			ui.Spacer(),
			ui.Spinner(22, ui.ColPrimary),
		).St(ui.St().GapOf(16)).AlignTo(ui.AlignCenter),
		ui.Divider(),
		ui.Row(
			ui.Skeleton(44, 44),
			ui.Column(ui.Skeleton(180, 12), ui.Skeleton(120, 12)).St(ui.St().GapOf(8)),
		).St(ui.St().GapOf(12)),
	)

	feed := ui.Card(
		ui.Title("交互组件", "页签、折叠面板与轻提示"),
		ui.Tabs([]string{"all", "ui"}, []string{"全部", "界面"}, "all", func(k string) {}),
		ui.Row(
			ui.Button("显示 Toast", func() {
				ui.ToastShow("操作成功 · Web 版")
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 600, "#ffffff").PadXY(16, 8).R(9).BgGrad(ui.ColPrimary, ui.ColPrimary2, 90)),
			ui.Button("清空提示", func() {
				ui.ToastHide()
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 500, ui.ColTextMuted).PadXY(16, 8).R(9)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
	)

	return ui.Column(header("组件", "表单 / 展示 / 交互三类组件，同一套自绘实现"), form, show, feed).
		St(ui.St().Padding(28).GapOf(18))
}

func chartsPage() *ui.Node {
	labels := []string{"一", "二", "三", "四", "五", "六", "七", "八"}
	bars := ui.Series{Name: "帧耗时", Values: []float64{42, 63, 51, 78, 58, 88, 71, 94}}
	mem := ui.Series{Name: "内存", Values: wave(30, 12, 8, 0.6)}
	return ui.Column(
		header("图表", "柱 / 网格 / 文字走顶点，弧线斜线多边形走纹理"),
		ui.Card(ui.BarChart("bar", labels, bars)),
		ui.Card(ui.AreaChart("area", labels, mem)),
		ui.Card(ui.LineChart("line", labels, mem, bars)),
	).St(ui.St().Padding(28).GapOf(18))
}

func formPage() *ui.Node {
	return ui.Column(
		header("表单", "输入 / 校验 / 提交都在同一棵节点树里"),
		ui.Card(
			ui.Input("acc", "", "账号", func(v string) {}),
			ui.PasswordInput("pwd", "", "密码", func(v string) {}),
			ui.Row(
				ui.Button("提交", func() {
					ui.ToastShow("已提交")
					ui.RequestRedraw()
				}).St(ui.St().Text(13, 600, "#ffffff").PadXY(16, 8).R(9).BgGrad(ui.ColPrimary, ui.ColPrimary2, 90)),
			),
		).St(ui.St().GapOf(14)),
	).St(ui.St().Padding(28).GapOf(18))
}

// wave 生成一条正弦波形数据（与桌面 gallery 造数方式一致）。
func wave(base, amp float64, n int, phase float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = base + amp*math.Sin(float64(i)*0.9+phase)
	}
	return out
}
