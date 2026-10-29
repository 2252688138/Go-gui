// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"math"
	"testing"
)

// TestRadarHitPerSeries 雷达图：指针落在某条系列的顶点上时必须命中该系列。
// 旧实现无论指在哪条多边形上都固定返回系列 0（提示里永远是「当前版本」）。
func TestRadarHitPerSeries(t *testing.T) {
	labels := []string{"性能", "稳定", "安全", "成本", "体验"}
	c := NewChart(ChartRadar, labels,
		Series{Name: "当前版本", Values: []float64{88, 92, 78, 70, 86}},
		Series{Name: "上个版本", Values: []float64{72, 80, 74, 64, 78}},
	)
	lg := testLayout()
	maxV := c.dataMax()
	n := len(labels)

	for j := range c.Series {
		for i := 0; i < n; i++ {
			ang := -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
			r := c.Series[j].Values[i] / maxV * float64(lg.pieR)
			mx := lg.pieC.X + float32(r*math.Cos(ang))
			my := lg.pieC.Y + float32(r*math.Sin(ang))
			hs, hi, ok := chartHit(c, lg, mx, my)
			if !ok || hs != j || hi != i {
				t.Fatalf("系列 %d（%s）轴 %d 顶点应命中自身，实际 hs=%d hi=%d ok=%v",
					j, c.Series[j].Name, i, hs, hi, ok)
			}
		}
	}
}
