// Go-gui — gallery 页面拆分：数据表格
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
	"strconv"
)

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
