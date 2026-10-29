// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"math"

	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// 图表绘制：几何计算 + 原生图元（网格/柱/文字/悬浮提示）+ 一张缓存的矢量纹理。
//
// 所有可变内容（网格、柱、文字）用原生顶点绘制，永远清晰且不吃纹理内存；
// 只有弧线 / 斜线 / 多边形（折线、面积、饼/环形、雷达）才光栅成纹理，
// 并且只在「数据 / 尺寸 / 缩放 / 主题」变化时重建。

// chartLayout 一帧的图表几何（逻辑像素，绝对坐标）。
type chartLayout struct {
	r      geom.Rect // 节点矩形
	plot   geom.Rect // 绘图区（不含轴标签）
	legend geom.Rect // 图例行
	scale  float32   // 纹理用的设备像素比

	yMin, yMax float64
	yTicks     []float64

	pieC  fpt     // 饼/环/雷达圆心
	pieR  float32 // 半径
	donut float32 // 环形内径比例
}

// chartMetrics 图表的字体与尺寸常量（统一一处方便调）。
const (
	chartTickSize   float32 = 10.5
	chartLabelSize  float32 = 11
	chartLegendSize float32 = 11
	chartLegendH    float32 = 18
	chartBarGap     float32 = 0.28 // 柱间距占单组宽度的比例
)

// layout 计算图表几何。scale 为纹理设备像素比（= Renderer.Scale()）。
func (c *Chart) layout(a *App, r geom.Rect, scale float32) chartLayout {
	pad := c.Padding
	lg := chartLayout{r: r, scale: scale}

	// 外围：内边距 → 图例（顶部）→ 轴标签
	inner := geom.Rect{X: r.X + pad, Y: r.Y + pad, W: r.W - pad*2, H: r.H - pad*2}
	if inner.W < 4 || inner.H < 4 {
		lg.plot = geom.Rect{X: r.X, Y: r.Y, W: maxf(r.W, 0), H: maxf(r.H, 0)}
		lg.pieC, lg.pieR = fpt{r.X + r.W/2, r.Y + r.H/2}, minf(r.W, r.H)/2-2
		return lg
	}
	if c.legendVisible() {
		lg.legend = geom.Rect{X: inner.X, Y: inner.Y, W: inner.W, H: chartLegendH}
		inner.Y += chartLegendH + 2
		inner.H -= chartLegendH + 2
	}

	switch c.Type {
	case ChartPie, ChartDonut, ChartRadar:
		side := minf(inner.W, inner.H)
		lg.pieC = fpt{inner.X + inner.W/2, inner.Y + inner.H/2}
		lg.pieR = side/2 - 2
		lg.donut = c.Donut
		lg.plot = geom.Rect{X: lg.pieC.X - lg.pieR, Y: lg.pieC.Y - lg.pieR, W: lg.pieR * 2, H: lg.pieR * 2}
		if c.Type == ChartRadar {
			// 雷达图留出外圈轴名空间（约 18px）
			lg.pieR -= 18
			if lg.pieR < 8 {
				lg.pieR = 8
			}
		}
		return lg
	}

	// 直角坐标系：左轴刻度 + 底部 x 标签
	f := a.fonts.get(chartTickSize, 400)
	lo, hi := c.axisRange()
	lg.yMin, lg.yMax = lo, hi
	lg.yTicks = c.tickValues(lg.yMin, lg.yMax)
	yLabelW := float32(0)
	if c.ShowAxis && f != nil {
		for _, v := range lg.yTicks {
			if w := f.Measure(fmtNum(v)); w > yLabelW {
				yLabelW = w
			}
		}
		yLabelW += 10
	}
	xLabelH := float32(0)
	if len(c.Labels) > 0 && f != nil {
		xLabelH = f.LineHeight + 8
	}
	lg.plot = geom.Rect{
		X: inner.X + yLabelW,
		Y: inner.Y,
		W: inner.W - yLabelW,
		H: inner.H - xLabelH,
	}
	if lg.plot.W < 2 || lg.plot.H < 2 {
		lg.plot = geom.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
	}
	return lg
}

// legendVisible 是否需要图例。
func (c *Chart) legendVisible() bool {
	if !c.ShowLegend {
		return false
	}
	if c.Type == ChartPie || c.Type == ChartDonut {
		return len(c.Series) > 0
	}
	return len(c.Series) > 1
}

// axisRange 纵轴范围。
func (c *Chart) axisRange() (float64, float64) {
	lo := c.Min
	hi := c.Max
	if hi <= 0 {
		hi = niceMax(c.dataMax(), c.YTicks)
	} else {
		hi = niceMax(hi, c.YTicks)
	}
	if hi <= lo {
		hi = lo + 1
	}
	return lo, hi
}

// tickValues 均匀刻度值（从下到上）。
func (c *Chart) tickValues(lo, hi float64) []float64 {
	n := c.YTicks
	if n < 1 {
		n = 4
	}
	out := make([]float64, 0, n+1)
	for i := 0; i <= n; i++ {
		out = append(out, lo+(hi-lo)*float64(i)/float64(n))
	}
	return out
}

// mapY 数值 → 绘图区 y 坐标。
func (lg *chartLayout) mapY(v float64) float32 {
	span := lg.yMax - lg.yMin
	if span <= 0 {
		return lg.plot.Bottom()
	}
	r := (v - lg.yMin) / span
	return lg.plot.Bottom() - float32(r)*lg.plot.H
}

// xAt 第 i 个数据点在绘图区的横坐标（组中心）。
func (lg *chartLayout) xAt(i, n int) float32 {
	if n <= 1 {
		return lg.plot.X + lg.plot.W/2
	}
	return lg.plot.X + lg.plot.W*(float32(i)+0.5)/float32(n)
}

// bandW 单组数据在绘图区占的宽度。
func (lg *chartLayout) bandW(n int) float32 {
	if n <= 0 {
		return lg.plot.W
	}
	return lg.plot.W / float32(n)
}

// ---- 绘制 ----

// paintChart 绘制图表节点。
func (a *App) paintChart(n *Node, r geom.Rect) {
	c := n.Chart
	if c == nil {
		return
	}
	c.applyDefaults()
	// 图表纹理的设备像素比 = effScale × 超采样倍数。
	// 曲线/弧线/多边形是光栅进纹理再贴上去的：只按设备像素生成的话，2× 档会把
	// 一张 1 倍图放大 2 倍再降采样，曲线边缘反而比 1:1 更糊（用户直观感受就是
	// 「超高清还不如标准清晰」）。纹理尺寸由 chart_cache 限到 4096 并受 LRU 预算约束。
	lg := c.layout(a, r, a.effScale()*a.textureSS())
	c.cache.plot = lg.plot
	c.cache.pieC, c.cache.pieR = lg.pieC, lg.pieR
	// 几何缓存：点击回调（OnPick）直接复用，不必重新跑布局
	c.cache.lg, c.cache.lgOK = lg, true
	c.cache.nodeX, c.cache.nodeY = r.X, r.Y

	// 1. 网格与坐标轴（原生图元）
	a.paintChartGrid(c, lg)

	// 2. 柱状图（原生圆角矩形，零纹理内存）
	if c.Type == ChartBar {
		a.paintChartBars(c, lg)
	}

	// 3. 需要曲线的类型：一张缓存纹理
	if c.needsTexture() {
		if tex := a.chartTexture(c, lg); tex != nil {
			a.r.DrawImage(gpu.Image{Rect: lg.plot, Tex: tex, Tint: gpu.Paint{R: 1, G: 1, B: 1, A: 1}})
		}
	}

	// 4. 悬浮态（原生：高亮柱 / 竖线 / 提示框）
	hs, hi, hov := a.chartHover(c, lg, r)
	a.paintChartHover(c, lg, hs, hi, hov)

	// 5. 文字：刻度 / 标签 / 图例 / 标题
	a.paintChartText(c, lg)
}

// needsTexture 该类型是否需要矢量纹理。
func (c *Chart) needsTexture() bool {
	switch c.Type {
	case ChartLine, ChartArea, ChartPie, ChartDonut, ChartRadar, ChartScatter:
		return true
	}
	return false
}

// paintChartGrid 网格线 + 坐标轴。
func (a *App) paintChartGrid(c *Chart, lg chartLayout) {
	if lg.plot.W <= 0 || lg.plot.H <= 0 {
		return
	}
	line := C(ColLine)
	line.A *= 0.9
	if c.ShowGrid {
		for _, v := range lg.yTicks {
			y := float32(math.Round(float64(lg.mapY(v))))
			a.r.Rect(gpu.Shape{Rect: geom.Rect{X: lg.plot.X, Y: y, W: lg.plot.W, H: 1}, Fill: line})
		}
		// 竖向网格：跟随 x 组中心
	cn := c.points()
	if c.Type != ChartRadar && cn > 1 && cn <= 24 {
		for i := 0; i < cn; i++ {
			x := float32(math.Round(float64(lg.xAt(i, cn))))
			a.r.Rect(gpu.Shape{Rect: geom.Rect{X: x, Y: lg.plot.Y, W: 1, H: lg.plot.H}, Fill: line})
		}
	}
	}
	if c.ShowAxis {
		axis := C(ColStroke)
		axis.A *= 0.85
		a.r.Rect(gpu.Shape{Rect: geom.Rect{X: lg.plot.X, Y: lg.plot.Bottom() - 0.5, W: lg.plot.W, H: 1}, Fill: axis})
		a.r.Rect(gpu.Shape{Rect: geom.Rect{X: lg.plot.X - 0.5, Y: lg.plot.Y, W: 1, H: lg.plot.H}, Fill: axis})
	}
}

// paintChartBars 柱状图（支持分组 / 堆叠 / 负值）。
func (a *App) paintChartBars(c *Chart, lg chartLayout) {
	n := c.points()
	if n <= 0 || len(c.Series) == 0 {
		return
	}
	ns := len(c.Series)
	band := lg.bandW(n)
	groupW := band * (1 - chartBarGap)
	baseY := lg.mapY(0)
	if c.Min > 0 {
		baseY = lg.plot.Bottom()
	}
	zeroY := lg.mapY(math.Max(c.Min, 0))

	var barW, gap float32
	if c.Stacked {
		barW = groupW
	} else {
		barW = groupW / float32(ns)
		gap = minf(barW*0.18, 3)
		if gap < 0 {
			gap = 0
		}
	}
	rad := [4]float32{}
	if r := minf(barW, 10) * 0.22; r > 1 {
		rad = [4]float32{r, r, r, r}
	}

	for i := 0; i < n; i++ {
		cx := lg.xAt(i, n)
		stackAcc := 0.0
		for j := 0; j < ns; j++ {
			s := c.Series[j]
			if i >= len(s.Values) {
				continue
			}
			v := s.Values[i]
			p := c.colorOf(j)
			var y0, y1 float32
			var x0, w float32
			if c.Stacked {
				y0 = lg.mapY(stackAcc)
				y1 = lg.mapY(stackAcc + v)
				stackAcc += v
				x0, w = cx-groupW/2, barW
			} else {
				x0 = cx - groupW/2 + float32(j)*barW + gap/2
				w = barW - gap
				y0 = lg.mapY(v)
				y1 = zeroY
			}
			if y1 < y0 {
				y0, y1 = y1, y0
			}
			h := y1 - y0
			if h < 1 {
				h = 1
			}
			// 渐变：顶部亮一点，底部稍沉，层次更清晰
			from := p
			to := gpu.Paint{R: p.R * 0.72, G: p.G * 0.72, B: p.B * 0.72, A: p.A}
			a.r.Rect(gpu.Shape{
				Rect:   geom.Rect{X: x0, Y: y0, W: w, H: h},
				Radius: rad,
				Grad:   gpu.Gradient{On: true, Angle: 90, From: from, To: to},
			})
		}
	}
	_ = baseY
}

// paintChartText 刻度、轴标签、图例、标题。
func (a *App) paintChartText(c *Chart, lg chartLayout) {
	tick := a.fonts.get(chartTickSize, 400)
	label := a.fonts.get(chartLabelSize, 500)
	leg := a.fonts.get(chartLegendSize, 500)
	dim := C(ColTextMuted)

	if c.ShowAxis && tick != nil && lg.plot.W > 0 {
		for _, v := range lg.yTicks {
			s := fmtNum(v)
			w := tick.Measure(s)
			a.r.TextRun(gpu.TextRun{
				X:     lg.plot.X - w - 7,
				Y:     lg.mapY(v) - tick.LineHeight/2,
				Text:  s,
				Font:  tick,
				Color: dim,
			})
		}
	}

	// X 轴标签：按可用宽度自动抽稀，避免重叠
	if label != nil && len(c.Labels) > 0 && c.Type != ChartPie && c.Type != ChartDonut {
		n := c.points()
		if n > 0 {
			stride := 1
			for stride < n && lg.bandW(n)*float32(stride) < 46 {
				stride++
			}
			ly := lg.plot.Bottom() + 5
			for i := 0; i < n; i++ {
				if i%stride != 0 {
					continue
				}
				if i >= len(c.Labels) {
					break
				}
				s := c.Labels[i]
				w := label.Measure(s)
				x := lg.xAt(i, n) - w/2
				if x < lg.r.X {
					x = lg.r.X
				}
				if x+w > lg.r.Right() {
					x = lg.r.Right() - w
				}
				a.r.TextRun(gpu.TextRun{X: x, Y: ly, Text: s, Font: label, Color: dim})
			}
		}
	}

	// 雷达图轴名
	if c.Type == ChartRadar && label != nil {
		n := len(c.Labels)
		for i := 0; i < n; i++ {
			ang := -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
			rr := float64(lg.pieR) + 14
			cx := float64(lg.pieC.X) + math.Cos(ang)*rr
			cy := float64(lg.pieC.Y) + math.Sin(ang)*rr
			w := label.Measure(c.Labels[i])
			a.r.TextRun(gpu.TextRun{
				X:     float32(cx) - w/2,
				Y:     float32(cy) - label.LineHeight/2,
				Text:  c.Labels[i],
				Font:  label,
				Color: dim,
			})
		}
	}

	// 图例：右上角起，放不下自动换行（最多 2 行）
	if leg != nil && c.legendVisible() {
		x := lg.legend.Right()
		y := lg.legend.Y + (lg.legend.H-leg.LineHeight)/2
		for i := len(c.Series) - 1; i >= 0; i-- {
			s := c.Series[i]
			name := s.Name
			if name == "" && i < len(c.Labels) {
				name = c.Labels[i]
			}
			if name == "" {
				continue
			}
			w := leg.Measure(name)
			itemW := w + 18
			if x-itemW < lg.legend.X {
				break
			}
			x -= itemW
			p := c.colorOf(i)
			a.r.Rect(gpu.Shape{
				Rect:   geom.Rect{X: x, Y: y + leg.LineHeight/2 - 4, W: 8, H: 8},
				Radius: [4]float32{2, 2, 2, 2},
				Fill:   p,
			})
			a.r.TextRun(gpu.TextRun{X: x + 12, Y: y, Text: name, Font: leg, Color: C(ColTextDim)})
		}
	}

	// 标题：左上角（与图例同一行左侧）
	if c.Title != "" && leg != nil {
		a.r.TextRun(gpu.TextRun{
			X:     lg.legend.X,
			Y:     lg.legend.Y + (lg.legend.H-leg.LineHeight)/2,
			Text:  c.Title,
			Font:  leg,
			Color: C(ColText),
		})
	}
}
