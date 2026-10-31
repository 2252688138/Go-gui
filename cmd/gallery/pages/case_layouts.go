// Go-gui — gallery 页面拆分：布局实验室
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"fmt"
	"gogui/ui"
	"strconv"
)

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
