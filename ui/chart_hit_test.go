package ui

import (
	"math"
	"strings"
	"testing"
)

// TestBarHitPerSeries 分组柱状图：鼠标落在同一组的不同柱子上，
// 必须命中各自对应的系列（旧实现只比较 Y，较高的柱会把矮柱整个吞掉，
// 于是三根柱都提示第一条系列「新增」）。
func TestBarHitPerSeries(t *testing.T) {
	c := NewChart(ChartBar, []string{"W1"},
		Series{Name: "新增", Values: []float64{90}},
		Series{Name: "续费", Values: []float64{50}},
		Series{Name: "退款", Values: []float64{10}},
	)
	lg := testLayout()

	seen := map[string]bool{}
	for j := 0; j < len(c.Series); j++ {
		r := barRect(c, lg, j, 0)
		if r.W <= 0 || r.H <= 0 {
			t.Fatalf("第 %d 根柱几何异常: %+v", j, r)
		}
		hs, hi, ok := chartHit(c, lg, r.X+r.W/2, r.Y+r.H/2)
		if !ok || hs != j || hi != 0 {
			t.Fatalf("鼠标在第 %d 根柱上应命中系列 %d，实际 hs=%d hi=%d ok=%v（r=%+v plot=%+v 点=(%.1f,%.1f) n=%d）",
				j, j, hs, hi, ok, r, lg.plot, r.X+r.W/2, r.Y+r.H/2, c.points())
		}
		tip := c.tipText(hs, hi)
		if tip == "" {
			t.Fatalf("第 %d 根柱提示为空", j)
		}
		if seen[tip] {
			t.Fatalf("不同柱子出现相同提示 %q", tip)
		}
		seen[tip] = true
		if !strings.Contains(tip, c.Series[j].Name) {
			t.Fatalf("第 %d 根柱提示应包含系列名 %q，实际 %q", j, c.Series[j].Name, tip)
		}
	}
}

// TestBarHitStacked 堆叠柱：按 Y 落在哪一段判断系列。
func TestBarHitStacked(t *testing.T) {
	c := NewChart(ChartBar, []string{"W1"},
		Series{Name: "A", Values: []float64{20}},
		Series{Name: "B", Values: []float64{30}},
		Series{Name: "C", Values: []float64{40}},
	)
	c.Stacked = true
	lg := testLayout()
	for j := 0; j < len(c.Series); j++ {
		r := barRect(c, lg, j, 0)
		hs, hi, ok := chartHit(c, lg, r.X+r.W/2, r.Y+r.H/2)
		if !ok || hs != j || hi != 0 {
			t.Fatalf("堆叠段 %d 应命中系列 %d，实际 hs=%d hi=%d ok=%v", j, j, hs, hi, ok)
		}
	}
}

// TestPieTipTextPerSector 饼图/环形图提示只显示当前扇区名，
// 不能再混入「第一个扇区名」（截图 bug：华东 · 华南 61）。
func TestPieTipTextPerSector(t *testing.T) {
	labels := []string{"华东", "华南", "华北", "西部"}
	for _, typ := range []ChartType{ChartPie, ChartDonut} {
		c := NewChart(typ, labels, PieSeries([]float64{50, 30, 15, 5}, labels)...)
		for j := range labels {
			tip := c.tipText(j, 0)
			if !strings.Contains(tip, labels[j]) {
				t.Fatalf("类型 %d 扇区 %d 提示应含 %q，实际 %q", typ, j, labels[j], tip)
			}
			for k, l := range labels {
				if k != j && strings.Contains(tip, l) {
					t.Fatalf("类型 %d 扇区 %d 提示混入其他扇区名 %q：%q", typ, j, l, tip)
				}
			}
		}
	}
}

// TestPieHitPerSector 扇形命中必须与绘制起点（-90°）和比例一致。
func TestPieHitPerSector(t *testing.T) {
	vals := []float64{50, 30, 15, 5}
	labels := []string{"A", "B", "C", "D"}
	total := 100.0
	for _, typ := range []ChartType{ChartPie, ChartDonut} {
		c := NewChart(typ, labels, PieSeries(vals, labels)...)
		lg := testLayout()
		acc := -math.Pi / 2
		for j, v := range vals {
			sweep := 2 * math.Pi * v / total
			a := acc + sweep/2
			rad := float64(lg.pieR) * 0.65
			mx := lg.pieC.X + float32(rad*math.Cos(a))
			my := lg.pieC.Y + float32(rad*math.Sin(a))
			hs, hi, ok := chartHit(c, lg, mx, my)
			if !ok || hs != j || hi != 0 {
				t.Fatalf("类型 %d 扇区 %d 中心应命中自身，实际 hs=%d hi=%d ok=%v", typ, j, hs, hi, ok)
			}
			acc += sweep
		}
	}
}
