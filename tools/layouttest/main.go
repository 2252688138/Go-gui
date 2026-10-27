//go:build windows

package main

import (
	"gogui/ui"
)

func statCard(title, value, unit string) *ui.Node {
	return ui.Column(
		ui.Text(title).St(ui.St().Text(12, 500, "#6b7694")),
		ui.Row(
			ui.Text(value).St(ui.St().Text(30, 700, "#f1f5ff")),
			ui.Text(unit).St(ui.St().Text(12, 500, "#22d3ee").PadOnly(0, 0, 6, 6)),
		).St(ui.St().PadOnly(4, 0, 0, 0)),
	).St(ui.St().GrowTo(1).Padding(20).R(16).BgC(ui.ColBg).Stroke(1, ui.ColStroke))
}

func view() *ui.Node {
	return ui.Row(
		ui.Column(
			ui.Text("GoGUI").St(ui.St().Text(20, 700, "#e8ecff").PadOnly(0, 0, 4, 0)),
			ui.Text("自绘 GUI 框架").St(ui.St().Text(12, 400, "#5b6480").PadOnly(0, 0, 24, 0)),
			ui.Button("仪表盘", nil).St(ui.St().Text(13, 600, "#ffffff").PadXY(14, 10).R(10).W(180)),
			ui.Spacer(),
			ui.Text("v0.1").St(ui.St().Text(11, 400, "#3d4459")),
		).St(ui.St().W(220).Hp(100).Padding(20).BgC(ui.ColBgSoft).GapOf(6)),

		ui.Scroll(
			ui.Column(
				ui.Row(
					ui.Column(
						ui.Text("控制台").St(ui.St().Text(26, 700, "#eef2ff")),
						ui.Text("零 cgo 纯 Go 自绘 SDF 抗锯齿").St(ui.St().Text(13, 400, "#6b7694").PadOnly(6, 0, 0, 0)),
					).Grow(1),
					ui.Button("新建", nil).St(ui.St().Text(13, 600, "#ffffff").PadXY(18, 9).R(10)),
				).St(ui.St().PadOnly(0, 0, 24, 0)),

				ui.Row(
					statCard("渲染帧率", "120", "fps"),
					statCard("绘制指令", "128", "draws"),
					statCard("内存占用", "12.4", "MB"),
				).St(ui.St().GapOf(16).PadOnly(0, 0, 24, 0)),

				ui.Column(
					ui.Text("样式能力").St(ui.St().Text(16, 600, "#dfe5ff").PadOnly(0, 0, 14, 0)),
				).St(ui.St().Padding(24).R(16).BgC(ui.ColBg)),
			).St(ui.St().Padding(28).GapOf(20)),
		).St(ui.St().GrowTo(1)).K("main-scroll"),
	).St(ui.St().Hp(100))
}

func main() {
	root := view()
	ui.LayoutForTest(root, 1180, 760, "Microsoft YaHei UI", 1)
	ui.DebugDump(root)
}
