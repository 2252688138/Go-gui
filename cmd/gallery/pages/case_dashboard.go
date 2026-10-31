// Go-gui — gallery 页面拆分：仪表盘
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
	"strconv"
)

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
