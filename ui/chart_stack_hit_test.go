package ui

import (
	"strings"
	"testing"
)

// TestStackedAreaHoverUsesCumulative 堆叠面积图：命中与悬浮标点都必须按
// 「累计高度」（即实际绘制高度）计算。旧实现用原始值，
// 悬浮点会落在两条线之间、还会误判成相邻系列。
func TestStackedAreaHoverUsesCumulative(t *testing.T) {
	c := NewChart(ChartArea, []string{"1", "2"},
		Series{Name: "桌面端", Values: []float64{40, 40}},
		Series{Name: "移动端", Values: []float64{60, 60}},
	)
	c.Stacked = true
	lg := testLayout()
	x := lg.xAt(0, 2)

	// 系列 1（移动端）绘制顶边 = mapY(40+60)（y 轴向下，顶边往上偏 1px 更稳）
	if hs, hi, ok := chartHit(c, lg, x, lg.mapY(100)+1); !ok || hs != 1 || hi != 0 {
		t.Fatalf("累计 100 处应命中系列 1（移动端），实际 hs=%d hi=%d ok=%v", hs, hi, ok)
	}
	// 系列 0（桌面端）顶边 = mapY(40)
	if hs, hi, ok := chartHit(c, lg, x, lg.mapY(40)+1); !ok || hs != 0 || hi != 0 {
		t.Fatalf("累计 40 处应命中系列 0（桌面端），实际 hs=%d hi=%d ok=%v", hs, hi, ok)
	}
	// 提示里仍是该系列自身数值，而不是累计值
	if tip := c.tipText(1, 0); !strings.Contains(tip, "60") || strings.Contains(tip, "100") {
		t.Fatalf("移动端提示应为自身数值 60，实际 %q", tip)
	}
}

// TestSeriesBase 堆叠基准值只在堆叠的柱/面积上生效。
func TestSeriesBase(t *testing.T) {
	c := NewChart(ChartArea, nil,
		Series{Name: "A", Values: []float64{10, 20}},
		Series{Name: "B", Values: []float64{5, 6}},
	)
	if b := c.seriesBase(1, 0); b != 0 {
		t.Fatalf("未堆叠时基准应为 0，实际 %v", b)
	}
	c.Stacked = true
	if b := c.seriesBase(1, 1); b != 20 {
		t.Fatalf("堆叠时第二系列第 1 点基准应为 20，实际 %v", b)
	}
	if b := c.seriesBase(0, 0); b != 0 {
		t.Fatalf("首个系列基准应为 0，实际 %v", b)
	}
}
