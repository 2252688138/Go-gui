// Go-gui — gallery 页面拆分：图表
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"fmt"
	"gogui/ui"
	"math"
)

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
