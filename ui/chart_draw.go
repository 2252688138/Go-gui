package ui

import (
	"math"

	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// 矢量层光栅：折线 / 面积 / 饼图 / 环形 / 雷达 / 散点。
// 坐标统一用「相对绘图区的逻辑像素」，由 chartDraw.pt 乘上设备比 k。

type chartDraw struct {
	cv *canvas
	lg chartLayout
	k  float32 // 逻辑像素 → 画布像素
}

func (d *chartDraw) pt(x, y float32) fpt { return fpt{x * d.k, y * d.k} }

// pts 把逻辑点阵列转成画布坐标。
func (d *chartDraw) pts(src []fpt) []fpt {
	out := make([]fpt, len(src))
	for i, p := range src {
		out[i] = d.pt(p.X, p.Y)
	}
	return out
}

// shifted 把布局平移到绘图区左上角，并把 plot 归零。
func (lg chartLayout) shifted(ox, oy float32) chartLayout {
	out := lg
	out.plot.X -= ox
	out.plot.Y -= oy
	out.pieC = fpt{lg.pieC.X - ox, lg.pieC.Y - oy}
	out.r = geom.Rect{}
	return out
}

// rasterPlot 把矢量层光栅成 w×h 的 RGBA（直通道 alpha）。
func (c *Chart) rasterPlot(w, h int, lg chartLayout) []byte {
	cv := newCanvas(w, h)
	if cv == nil {
		return nil
	}
	d := &chartDraw{cv: cv, lg: lg, k: lg.scale}
	switch c.Type {
	case ChartLine:
		d.lines(c, false)
	case ChartArea:
		d.lines(c, true)
	case ChartPie, ChartDonut:
		d.pie(c)
	case ChartRadar:
		d.radar(c)
	case ChartScatter:
		d.scatter(c)
	}
	return cv.result()
}

// 逻辑线宽（设备像素上至少 2px，避免高 DPI 下发丝）。
func (d *chartDraw) lineWidth() float32 { return maxf(2, 2*d.k) }

// seriesPoints 第 j 条系列的指纹点列（相对绘图区）。
func (d *chartDraw) seriesPoints(c *Chart, j int) []fpt {
	s := c.Series[j]
	n := c.points()
	if n <= 0 {
		return nil
	}
	out := make([]fpt, 0, n)
	for i := 0; i < n && i < len(s.Values); i++ {
		x := d.lg.xAt(i, n) - d.lg.plot.X
		out = append(out, fpt{x, d.lg.mapY(s.Values[i]) - d.lg.plot.Y})
	}
	return out
}

// smoothPoints Catmull-Rom 插值（每段 sub 个采样点）。
func smoothPoints(src []fpt, sub int) []fpt {
	n := len(src)
	if n < 3 || sub < 2 {
		return src
	}
	out := make([]fpt, 0, n*sub)
	at := func(i int) fpt {
		if i < 0 {
			return src[0]
		}
		if i >= n {
			return src[n-1]
		}
		return src[i]
	}
	for i := 0; i < n-1; i++ {
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		for s := 0; s < sub; s++ {
			t := float32(s) / float32(sub)
			t2, t3 := t*t, t*t*t
			f := func(a0, a1, a2, a3 float32) float32 {
				return 0.5 * ((2*a1) + (-a0+a2)*t + (2*a0-5*a1+4*a2-a3)*t2 + (-a0+3*a1-3*a2+a3)*t3)
			}
			out = append(out, fpt{f(p0.X, p1.X, p2.X, p3.X), f(p0.Y, p1.Y, p2.Y, p3.Y)})
		}
	}
	out = append(out, src[n-1])
	return out
}

// lines 折线 / 面积。
func (d *chartDraw) lines(c *Chart, area bool) {
	n := c.points()
	if n <= 0 {
		return
	}
	baseY := d.lg.mapY(math.Max(c.Min, 0)) - d.lg.plot.Y
	if c.Min > 0 {
		baseY = d.lg.plot.H
	}

	if c.Stacked && area {
		// 堆叠面积：自下而上累加，每层向下填充到上一层
		acc := make([]float64, n)
		for j := range c.Series {
			top := make([]fpt, 0, n)
			bot := make([]fpt, 0, n)
			for i := 0; i < n; i++ {
				v := 0.0
				if i < len(c.Series[j].Values) {
					v = c.Series[j].Values[i]
				}
				x := d.lg.xAt(i, n) - d.lg.plot.X
				top = append(top, fpt{x, d.lg.mapY(acc[i]+v) - d.lg.plot.Y})
				bot = append(bot, fpt{x, d.lg.mapY(acc[i]) - d.lg.plot.Y})
				acc[i] += v
			}
			poly := append([]fpt{}, top...)
			for i := len(bot) - 1; i >= 0; i-- {
				poly = append(poly, bot[i])
			}
			fill := c.colorOf(j)
			fill.A *= 0.32
			d.cv.fillPoly(d.pts(poly), fill)
			d.cv.strokePoly(d.pts(top), d.lineWidth(), c.colorOf(j), false)
		}
		return
	}

	for j := range c.Series {
		pts := d.seriesPoints(c, j)
		if len(pts) < 2 {
			if len(pts) == 1 {
				d.cv.disc(pts[0].X*d.k, pts[0].Y*d.k, d.lineWidth()*0.9, c.colorOf(j))
			}
			continue
		}
		drawPts := pts
		if c.Smooth {
			drawPts = smoothPoints(pts, 8)
		}
		col := c.colorOf(j)
		if area {
			poly := append([]fpt{}, drawPts...)
			poly = append(poly, fpt{drawPts[len(drawPts)-1].X, baseY})
			poly = append(poly, fpt{drawPts[0].X, baseY})
			fill := col
			fill.A *= 0.24
			d.cv.fillPoly(d.pts(poly), fill)
		}
		d.cv.strokePoly(d.pts(drawPts), d.lineWidth(), col, false)
		// 数据点：点数不多时画小圆，方便瞄准 / 悬浮
		if n <= 40 {
			r := maxf(2.2, 2.6*d.k)
			bg := C(ColBgDeep)
			for _, p := range pts {
				d.cv.disc(p.X*d.k, p.Y*d.k, r+maxf(1, d.k), bg)
				d.cv.disc(p.X*d.k, p.Y*d.k, r, col)
			}
		}
	}
}

// pie 饼图 / 环形图（扇区间留 1° 间隙，悬浮扇区直接高亮）。
func (d *chartDraw) pie(c *Chart) {
	total := 0.0
	for _, s := range c.Series {
		for _, v := range s.Values {
			if v > 0 {
				total += v
			}
		}
	}
	if total <= 0 {
		return
	}
	cx, cy := d.lg.pieC.X*d.k, d.lg.pieC.Y*d.k
	r1 := d.lg.pieR * d.k
	r0 := float32(0)
	if c.Type == ChartDonut {
		r0 = r1 * c.Donut
	}
	gap := float32(0.012) // 扇形间隙（弧度）
	ang := float32(-math.Pi / 2)
	for j, s := range c.Series {
		sum := 0.0
		for _, v := range s.Values {
			if v > 0 {
				sum += v
			}
		}
		if sum <= 0 {
			continue
		}
		sweep := float32(2*math.Pi*sum/total) - gap
		if sweep < 0.004 {
			sweep = 0.004
		}
		d.sector(cx, cy, r0, r1, ang+gap/2, ang+gap/2+sweep, c.colorOf(j))
		ang += float32(2*math.Pi*sum/total)
	}
}

// sector 画一个圆环扇区（画布像素坐标，点在内部即亮，抗锯齿由画布超采样完成）。
func (d *chartDraw) sector(cx, cy, r0, r1, a0, a1 float32, p gpu.Paint) {
	if r1 <= 0 || a1 <= a0 || p.A <= 0 {
		return
	}
	sweep := a1 - a0
	if sweep >= 2*math.Pi-1e-4 {
		sweep = 2 * math.Pi
	}
	root := func(v float32) float32 { return v * chartSS }
	cx, cy, r0, r1 = root(cx), root(cy), root(r0), root(r1)
	x0, x1 := int(cx-r1), int(cx+r1+1)
	y0, y1 := int(cy-r1), int(cy+r1+1)
	r0sq, r1sq := r0*r0, r1*r1
	if r0sq >= r1sq {
		return
	}
	for y := y0; y <= y1; y++ {
		if y < 0 || y >= d.cv.sh {
			continue
		}
		dy := float32(y) + 0.5 - cy
		for x := x0; x <= x1; x++ {
			if x < 0 || x >= d.cv.sw {
				continue
			}
			dx := float32(x) + 0.5 - cx
			dd := dx*dx + dy*dy
			if dd < r0sq || dd > r1sq {
				continue
			}
			if sweep < 2*math.Pi {
				a := float32(math.Atan2(float64(dy), float64(dx)))
				for a < a0 {
					a += 2 * math.Pi
				}
				if a > a0+sweep {
					continue
				}
			}
			d.cv.blend(x, y, p, 1)
		}
	}
}

// radar 雷达图：蛛网底 + 各系列多边形。
func (d *chartDraw) radar(c *Chart) {
	cx, cy := d.lg.pieC.X*d.k, d.lg.pieC.Y*d.k
	R := d.lg.pieR * d.k
	if R <= 0 {
		return
	}
	n := c.points()
	if n < 3 {
		return
	}
	angAt := func(i int) float64 { return -math.Pi/2 + 2*math.Pi*float64(i)/float64(n) }
	grid := C(ColLine)
	for l := 1; l <= 4; l++ {
		rr := R * float32(l) / 4
		pts := make([]fpt, n)
		for i := 0; i < n; i++ {
			pts[i] = fpt{cx + float32(math.Cos(angAt(i)))*rr, cy + float32(math.Sin(angAt(i)))*rr}
		}
		d.cv.strokePoly(pts, maxf(1, d.k), grid, true)
	}
	for i := 0; i < n; i++ {
		d.cv.line(fpt{cx, cy}, fpt{cx + float32(math.Cos(angAt(i)))*R, cy + float32(math.Sin(angAt(i)))*R}, maxf(0.6, 0.7*d.k), grid)
	}

	maxV := c.dataMax()
	if maxV <= 0 {
		maxV = 1
	}
	if c.Max > 0 {
		maxV = c.Max
	}
	for j := range c.Series {
		s := c.Series[j]
		if len(s.Values) < n {
			continue
		}
		col := c.colorOf(j)
		pts := make([]fpt, n)
		for i := 0; i < n; i++ {
			r := float32(s.Values[i]/maxV) * R
			if r < 0 {
				r = 0
			}
			pts[i] = fpt{cx + float32(math.Cos(angAt(i)))*r, cy + float32(math.Sin(angAt(i)))*r}
		}
		fill := col
		fill.A *= 0.22
		d.cv.fillPoly(pts, fill)
		d.cv.strokePoly(pts, d.lineWidth()*0.9, col, true)
	}
}

// scatter 散点图。
func (d *chartDraw) scatter(c *Chart) {
	n := c.points()
	if n <= 0 {
		return
	}
	for j := range c.Series {
		s := c.Series[j]
		col := c.colorOf(j)
		for i, v := range s.Values {
			x := d.lg.xAt(i, n) - d.lg.plot.X
			if i < len(s.X) {
				x = float32((s.X[i] - d.lg.yMin) / (d.lg.yMax - d.lg.yMin) * float64(d.lg.plot.W))
			}
			y := d.lg.mapY(v) - d.lg.plot.Y
			d.cv.disc(x*d.k, y*d.k, maxf(2.6, 3.4*d.k), col)
		}
	}
}
