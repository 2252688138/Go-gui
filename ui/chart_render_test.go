package ui

import (
	"testing"

	"gogui/internal/geom"
)

// chartCase 光栅测试用例。
type chartCase struct {
	name string
	c    *Chart
}

// testLayout 造一个不依赖 App / 字体缓存的图表几何。
func testLayout() chartLayout {
	lg := chartLayout{}
	lg.r = geom.Rect{X: 0, Y: 0, W: 260, H: 150}
	lg.plot = geom.Rect{X: 40, Y: 10, W: 200, H: 110}
	lg.scale = 1
	lg.yMin, lg.yMax = 0, 100
	lg.yTicks = []float64{0, 25, 50, 75, 100}
	lg.pieC = fpt{130, 75}
	lg.pieR = 56
	return lg
}

// opaque 统计非透明像素数量（纹理是否真的有内容）。
func opaque(px []byte) int {
	n := 0
	for i := 3; i < len(px); i += 4 {
		if px[i] > 8 {
			n++
		}
	}
	return n
}

func TestCanvasPrimitives(t *testing.T) {
	cv := newCanvas(60, 40)
	if cv == nil {
		t.Fatal("画布创建失败")
	}
	cv.fillPoly([]fpt{{5, 5}, {55, 10}, {30, 35}}, C("#ff0000"))
	if n := opaque(cv.result()); n < 200 {
		t.Fatalf("多边形填充像素过少: %d", n)
	}

	cv2 := newCanvas(60, 40)
	cv2.strokePoly([]fpt{{2, 30}, {30, 2}, {58, 30}}, 2, C("#00ff00"), false)
	if n := opaque(cv2.result()); n < 60 {
		t.Fatalf("折线描边像素过少: %d", n)
	}

	cv3 := newCanvas(40, 40)
	cv3.disc(20, 20, 12, C("#38bdf8"))
	if n := opaque(cv3.result()); n < 300 {
		t.Fatalf("圆点像素过少: %d", n)
	}
}

// TestChartRasterAllTypes 所有需要纹理的类型都必须真的画出东西。
func TestChartRasterAllTypes(t *testing.T) {
	series := []Series{
		{Name: "A", Values: []float64{10, 40, 25, 80, 55, 95}},
		{Name: "B", Values: []float64{30, 20, 60, 45, 70, 35}},
	}
	labels := []string{"a", "b", "c", "d", "e", "f"}
	pieVals := []float64{30, 25, 20, 15, 10}

	cases := []chartCase{
		{"line", NewChart(ChartLine, labels, series...)},
		{"area", NewChart(ChartArea, labels, series...)},
		{"pie", NewChart(ChartPie, labels, PieSeries(pieVals, labels)...)},
		{"donut", NewChart(ChartDonut, labels, PieSeries(pieVals, labels)...)},
		{"radar", NewChart(ChartRadar, []string{"x", "y", "z", "w", "v"}, series...)},
		{"scatter", NewChart(ChartScatter, nil, series...)},
	}
	lg0 := testLayout()
	lg := lg0.shifted(lg0.plot.X, lg0.plot.Y)
	for _, tc := range cases {
		px := tc.c.rasterPlot(200, 110, lg)
		if len(px) != 200*110*4 {
			t.Fatalf("%s: 像素尺寸不对 (%d)", tc.name, len(px))
		}
		if n := opaque(px); n < 80 {
			t.Fatalf("%s: 矢量层几乎是空的（%d 像素）", tc.name, n)
		}
	}
}

// TestChartHit 悬浮 / 点击命中。
func TestChartHit(t *testing.T) {
	lg := testLayout()
	labels := []string{"a", "b", "c", "d"}
	bar := NewChart(ChartBar, labels, Series{Name: "A", Values: []float64{10, 20, 30, 40}})
	x := lg.xAt(2, 4)
	if s, i, ok := chartHit(bar, lg, x, lg.mapY(30)); !ok || i != 2 || s != 0 {
		t.Fatalf("柱状图命中错误: s=%d i=%d ok=%v", s, i, ok)
	}

	pie := NewChart(ChartPie, []string{"p", "q"}, PieSeries([]float64{50, 50}, []string{"p", "q"})...)
	if _, i, ok := chartHit(pie, lg, lg.pieC.X+10, lg.pieC.Y-lg.pieR*0.5); !ok || i != 0 {
		t.Fatalf("饼图命中错误: i=%d ok=%v", i, ok)
	}
}

// TestNiceMaxAndFormat 轴上限与数字格式化。
func TestNiceMaxAndFormat(t *testing.T) {
	if got := niceMax(97, 4); got != 100 {
		t.Fatalf("niceMax(97,4)=%v, want 100", got)
	}
	if got := niceMax(0, 4); got != 1 {
		t.Fatalf("niceMax(0,4)=%v, want 1", got)
	}
	if got := fmtNum(12345); got != "1.2万" {
		t.Fatalf("fmtNum(12345)=%q", got)
	}
	if got := fmtNum(48); got != "48" {
		t.Fatalf("fmtNum(48)=%q", got)
	}
}
