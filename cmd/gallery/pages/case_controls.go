// Go-gui — gallery 页面拆分：控件总览
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
	"strconv"
)

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
					CtrlMenu = !CtrlMenu
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
