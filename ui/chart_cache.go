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

// 图表的纹理缓存与悬浮交互。
//
// 缓存键 = 节点几何 + 设备比 + 主题 + 数据指纹；值 = 一张 RGBA 纹理。
// 每个 App 维护自己的缓存列表（纹理归属各自 GL 上下文），
// 总字节超过 chartTexBudget 时按最近使用（stamp）淘汰。

// chartTexBudget 所有图表纹理的总预算（24MB，约够 10~14 张 600×300）。
const chartTexBudget int64 = 24 << 20

// chartTexture 取得（必要时重建）图表的矢量层纹理。
func (a *App) chartTexture(c *Chart, lg chartLayout) *gpu.Texture {
	w := int(math.Round(float64(lg.plot.W * lg.scale)))
	h := int(math.Round(float64(lg.plot.H * lg.scale)))
	if w <= 0 || h <= 0 || lg.plot.W <= 1 || lg.plot.H <= 1 {
		return nil
	}
	if w > 4096 {
		w = 4096
	}
	if h > 4096 {
		h = 4096
	}

	key := c.fingerprint(w, h)
	a.chartStamp++
	if c.cache.tex != nil && c.cache.key == key {
		c.cache.stamp = a.chartStamp
		return c.cache.tex
	}

	shift := lg.shifted(lg.plot.X, lg.plot.Y)
	pix := c.rasterPlot(w, h, shift)
	if pix == nil {
		return nil
	}

	if c.cache.tex != nil {
		a.r.DeleteTexture(c.cache.tex)
		a.chartBytes -= c.cache.bytes
		c.cache.tex = nil
	}
	a.chartEvict(int64(len(pix)))
	t := a.r.NewTextureRGBA(w, h, pix)
	if t == nil {
		return nil
	}
	c.cache.tex, c.cache.key, c.cache.w, c.cache.h = t, key, w, h
	c.cache.bytes = int64(len(pix))
	c.cache.stamp = a.chartStamp
	a.chartBytes += c.cache.bytes
	a.chartList = append(a.chartList, &c.cache)
	return t
}

// chartEvict 腾出至少 need 字节（按最近使用淘汰）。
func (a *App) chartEvict(need int64) {
	for a.chartBytes+need > chartTexBudget && len(a.chartList) > 0 {
		victim := 0
		for i := 1; i < len(a.chartList); i++ {
			if a.chartList[i].stamp < a.chartList[victim].stamp {
				victim = i
			}
		}
		v := a.chartList[victim]
		a.chartList = append(a.chartList[:victim], a.chartList[victim+1:]...)
		if v.tex != nil {
			a.r.DeleteTexture(v.tex)
			a.chartBytes -= v.bytes
		}
		v.tex, v.key, v.bytes = nil, 0, 0
	}
}

// chartReleaseAll 释放全部图表纹理，下次绘制按需重建。
//
// 空闲 / 最小化时调用：图表页一次会同时驻留 7 张高分辨率纹理
// （按 superSample 倍率的平方增长，2× 时单张 600×300 就是 ~2.9MB），
// 页面切走或窗口闲置时这些纹理没有保留价值。
func (a *App) chartReleaseAll() {
	if a == nil || a.r == nil {
		return
	}
	for _, c := range a.chartList {
		if c.tex != nil {
			a.r.DeleteTexture(c.tex)
		}
		c.tex, c.key, c.bytes, c.w, c.h = nil, 0, 0, 0, 0
	}
	a.chartList = a.chartList[:0]
	a.chartBytes = 0
}

// chartResetCache 关闭窗口时清空缓存记账（GL 资源由 Renderer.Destroy 统一释放）。
func (a *App) chartResetCache() {
	a.chartBytes = 0
	a.chartList = nil
}

// fingerprint 数据 + 几何 + 主题 的 64 位指纹。
//
// 旧实现每帧为每张图表 new 一个 fnv 哈希器、逐系列把颜色串转成 []byte、
// 最后 strconv.FormatUint 生成字符串比较：图表页一开每帧就是几十次分配。
// 现在用零分配的内联 FNV-1a，直接对字符串字节与 float64 位模式滚动哈希。
func (c *Chart) fingerprint(w, h int) uint64 {
	const (
		fnvOffset = 14695981039346656037
		fnvPrime  = 1099511628211
	)
	hh := uint64(fnvOffset)
	mix := func(v uint64) {
		hh ^= v
		hh *= fnvPrime
	}
	mixf := func(v float64) { mix(math.Float64bits(v)) }
	mixs := func(s string) {
		for i := 0; i < len(s); i++ {
			hh ^= uint64(s[i])
			hh *= fnvPrime
		}
		mix(0xFF) // 字段分隔，避免 "ab"+"c" 与 "a"+"bc" 撞车
	}

	mixf(float64(w))
	mixf(float64(h))
	mixf(float64(c.Type))
	mixf(float64(c.Padding))
	mixf(float64(c.YTicks))
	mixf(float64(c.Donut))
	mixf(c.Min)
	mixf(c.Max)
	f := 0.0
	if c.Stacked {
		f += 1
	}
	if c.Smooth {
		f += 2
	}
	if c.ShowAxis {
		f += 4
	}
	mixf(f)
	for _, s := range c.Series {
		mixs(s.Color)
		mixf(float64(len(s.Values)))
		for _, v := range s.Values {
			mixf(v)
		}
		for _, v := range s.X {
			mixf(v)
		}
	}
	mixs(curTheme.Name)
	mixs(ColBgDeep)
	mixs(ColLine)
	mixs(ColTextMuted)
	return hh
}

// ---- 悬浮与命中 ----

// barRect 柱子矩形（系列 j、数据点 i）。
func barRect(c *Chart, lg chartLayout, j, i int) geom.Rect {
	n := c.points()
	ns := len(c.Series)
	if n <= 0 || ns <= 0 || j < 0 || j >= ns || i < 0 || i >= n {
		return geom.Rect{}
	}
	band := lg.bandW(n)
	groupW := band * (1 - chartBarGap)
	cx := lg.xAt(i, n)
	zeroY := lg.mapY(math.Max(c.Min, 0))

	if c.Stacked {
		acc := 0.0
		for k := 0; k <= j && k < ns; k++ {
			v := 0.0
			if i < len(c.Series[k].Values) {
				v = c.Series[k].Values[i]
			}
			top, bot := lg.mapY(acc), lg.mapY(acc+v)
			acc += v
			if k == j {
				// 注意：y 轴向下，mapY 结果里较小的值才是矩形上边。
				// 旧实现按「y1 更小就交换」的写法得到 Y=下边、H=负 → 被 max(…,1)
				// 夹成 1px，于是只有贴着基线的 1 像素能命中（用户看到的是
				// 「三根柱子都提示第一条系列」且柱体本身无法悬停）。
				if top > bot {
					top, bot = bot, top
				}
				h := bot - top
				if h < 1 {
					h = 1
				}
				return geom.Rect{X: cx - groupW/2, Y: top, W: groupW, H: h}
			}
		}
		return geom.Rect{}
	}

	barW := groupW / float32(ns)
	gap := minf(barW*0.18, 3)
	s := c.Series[j]
	if i >= len(s.Values) {
		return geom.Rect{}
	}
	top, bot := zeroY, lg.mapY(s.Values[i])
	if top > bot {
		top, bot = bot, top
	}
	h := bot - top
	if h < 1 {
		h = 1
	}
	return geom.Rect{
		X: cx - groupW/2 + float32(j)*barW + gap/2,
		Y: top,
		W: barW - gap,
		H: h,
	}
}

// seriesBase 某系列在第 i 个数据点的堆叠基准值（未堆叠恒为 0）。
// 堆叠的柱状 / 面积在图上画的是「基准 + 本系列值」，
// 命中与悬浮标点必须用同一个高度，否则会落在两条线之间。
func (c *Chart) seriesBase(j, i int) float64 {
	if !c.Stacked || (c.Type != ChartBar && c.Type != ChartArea) {
		return 0
	}
	base := 0.0
	for k := 0; k < j && k < len(c.Series); k++ {
		if i < len(c.Series[k].Values) {
			base += c.Series[k].Values[i]
		}
	}
	return base
}

// chartHit 根据鼠标位置求出命中的（系列，数据点）。mx/my 为相对节点左上角的坐标。
func chartHit(c *Chart, lg chartLayout, mx, my float32) (int, int, bool) {
	if c == nil {
		return 0, 0, false
	}
	switch c.Type {
	case ChartPie, ChartDonut:
		total := 0.0
		for _, s := range c.Series {
			for _, v := range s.Values {
				if v > 0 {
					total += v
				}
			}
		}
		if total <= 0 {
			return 0, 0, false
		}
		dx := mx - lg.pieC.X
		dy := my - lg.pieC.Y
		d := float32(math.Hypot(float64(dx), float64(dy)))
		if d > lg.pieR {
			return 0, 0, false
		}
		if c.Type == ChartDonut && d < lg.pieR*c.Donut {
			return 0, 0, false
		}
		a := float32(math.Atan2(float64(dy), float64(dx)))
		for a < -math.Pi/2 {
			a += 2 * math.Pi
		}
		acc := float32(-math.Pi / 2)
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
			sweep := float32(2 * math.Pi * sum / total)
			if a >= acc && a < acc+sweep {
				return j, 0, true
			}
			acc += sweep
		}
		return 0, 0, false

	case ChartRadar:
		n := c.points()
		if n < 3 {
			return 0, 0, false
		}
		dx := mx - lg.pieC.X
		dy := my - lg.pieC.Y
		dist := float32(math.Hypot(float64(dx), float64(dy)))
		if dist > lg.pieR+16 {
			return 0, 0, false
		}
		// 轴下标：绘制时角度从 -90° 起按轴序递增
		a := math.Atan2(float64(dy), float64(dx)) + math.Pi/2
		for a < 0 {
			a += 2 * math.Pi
		}
		i := int(math.Round(a / (2 * math.Pi) * float64(n)))
		if i >= n {
			i = 0
		}
		// 系列：取在该轴上「半径离指针最近」的那条多边形。
		// 旧实现固定返回系列 0，所以指在「上个版本」上也永远提示「当前版本」。
		maxV := c.dataMax()
		if maxV <= 0 {
			maxV = 1
		}
		if c.Max > 0 {
			maxV = c.Max
		}
		best, bestD := 0, float32(math.MaxFloat32)
		for j, s := range c.Series {
			if i >= len(s.Values) {
				continue
			}
			r := float32(s.Values[i]/maxV) * lg.pieR
			if r < 0 {
				r = 0
			}
			d := float32(math.Abs(float64(dist - r)))
			if d < bestD {
				best, bestD = j, d
			}
		}
		return best, i, true

	case ChartBar:
		n := c.points()
		ns := len(c.Series)
		if n <= 0 || ns == 0 || !lg.plot.Contains(geom.Point{X: mx, Y: my}) {
			return 0, 0, false
		}
		i := int((mx - lg.plot.X) / lg.plot.W * float32(n))
		if i < 0 {
			i = 0
		}
		if i >= n {
			i = n - 1
		}
		if c.Stacked {
			// 堆叠柱：各段共用同一水平位置，按 Y 落在哪一段判断（自上而下）
			for j := ns - 1; j >= 0; j-- {
				if r := barRect(c, lg, j, i); r.H > 0 && my >= r.Y && my <= r.Bottom() {
					return j, i, true
				}
			}
			return 0, i, true
		}
		// 分组柱：先按 X 定位到具体那一根柱子。
		// 旧实现只用 Y 判断，同一组里较高的柱子会把下面的矮柱整个吞掉，
		// 于是鼠标移到组内任意一根上都提示第一条系列（"新增"）的数值。
		best, bestD := 0, float32(math.MaxFloat32)
		for j := 0; j < ns; j++ {
			r := barRect(c, lg, j, i)
			if r.W <= 0 {
				continue
			}
			d := float32(math.Abs(float64(mx - (r.X + r.W/2))))
			if d < bestD {
				best, bestD = j, d
			}
		}
		return best, i, true
	}

	n := c.points()
	if n <= 0 || !lg.plot.Contains(geom.Point{X: mx, Y: my}) {
		return 0, 0, false
	}
	i := int((mx - lg.plot.X) / lg.plot.W * float32(n))
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	best, bestD := 0, math.MaxFloat64
	for j, s := range c.Series {
		if i >= len(s.Values) {
			continue
		}
		// 堆叠面积/柱画的是累计高度，命中也要按累计值取 y
		y := lg.mapY(c.seriesBase(j, i) + s.Values[i])
		d := math.Abs(float64(y - my))
		if d < bestD {
			best, bestD = j, d
		}
	}
	return best, i, true
}

// chartHover 本帧悬浮命中（同时记下命中结果供点击回调使用）。
// 坐标统一用绝对（屏幕逻辑）坐标，与 chartLayout 一致。
func (a *App) chartHover(c *Chart, lg chartLayout, r geom.Rect) (int, int, bool) {
	c.cache.hitS, c.cache.hitI, c.cache.hitOK = 0, 0, false
	if !a.mouseIn || !r.Contains(geom.Point{X: a.mouseX, Y: a.mouseY}) {
		return 0, 0, false
	}
	s, i, ok := chartHit(c, lg, a.mouseX, a.mouseY)
	c.cache.hitS, c.cache.hitI, c.cache.hitOK = s, i, ok
	return s, i, ok
}

// paintChartHover 悬浮高亮 + 数值提示（全部原生图元，不重建纹理）。
func (a *App) paintChartHover(c *Chart, lg chartLayout, hs, hi int, ok bool) {
	if !ok {
		return
	}
	f := a.fonts.get(chartLabelSize, 500)
	if f == nil {
		return
	}

	var tip string
	var color gpu.Paint
	var anchorX, anchorY float32

	switch c.Type {
	case ChartBar:
		r := barRect(c, lg, hs, hi)
		if r.H <= 0 {
			return
		}
		if !c.Stacked {
			a.r.Rect(gpu.Shape{Rect: r, Radius: [4]float32{3, 3, 3, 3}, Fill: gpu.Paint{R: 1, G: 1, B: 1, A: 0.14}})
		}
		color = c.colorOf(hs)
		tip = c.tipText(hs, hi)
		anchorX, anchorY = r.X+r.W/2, r.Y

	case ChartLine, ChartArea, ChartScatter:
		n := c.points()
		if hi < 0 || hi >= n {
			return
		}
		x := lg.xAt(hi, n)
		guide := C(ColText)
		guide.A = 0.16
		a.r.Rect(gpu.Shape{Rect: geom.Rect{X: x, Y: lg.plot.Y, W: 1, H: lg.plot.H}, Fill: guide})
		color = c.colorOf(hs)
		if hs < len(c.Series) && hi < len(c.Series[hs].Values) {
			// 标点也要落在该系列的绘制高度上（堆叠时是累计值）
			y := lg.mapY(c.seriesBase(hs, hi) + c.Series[hs].Values[hi])
			a.r.Rect(gpu.Shape{Rect: geom.Rect{X: x - 6, Y: y - 6, W: 12, H: 12}, Radius: [4]float32{6, 6, 6, 6}, Fill: C(ColBgDeep)})
			a.r.Rect(gpu.Shape{Rect: geom.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}, Radius: [4]float32{4, 4, 4, 4}, Fill: color})
			anchorX, anchorY = x, y
		}
		tip = c.tipText(hs, hi)

	default:
		if hs >= len(c.Series) {
			return
		}
		color = c.colorOf(hs)
		tip = c.tipText(hs, hi)
		anchorX, anchorY = lg.pieC.X, lg.pieC.Y-lg.pieR*0.35
	}

	if tip == "" {
		return
	}
	tipW := f.Measure(tip) + 20
	tipH := f.LineHeight + 12
	tx := anchorX + 14
	ty := anchorY - tipH - 8
	if tx+tipW > lg.r.Right() {
		tx = anchorX - tipW - 14
	}
	if tx < lg.r.X {
		tx = lg.r.X
	}
	if ty < lg.r.Y {
		ty = anchorY + 14
	}
	if ty+tipH > lg.r.Bottom() {
		ty = lg.r.Bottom() - tipH
	}
	a.r.Rect(gpu.Shape{
		Rect:          geom.Rect{X: tx, Y: ty, W: tipW, H: tipH},
		Radius:        [4]float32{8, 8, 8, 8},
		Fill:          C(ColPanel),
		Stroke:        1,
		StrokeColor:   paintOf(ColStroke, 0.9),
		ShadowBlur:    18,
		ShadowOffsetY: 6,
		ShadowColor:   gpu.Paint{A: 0.35},
	})
	a.r.Rect(gpu.Shape{Rect: geom.Rect{X: tx + 8, Y: ty + tipH/2 - 3, W: 6, H: 6}, Radius: [4]float32{3, 3, 3, 3}, Fill: color})
	a.r.TextRun(gpu.TextRun{X: tx + 18, Y: ty + 6, Text: tip, Font: f, Color: C(ColText)})
}

// tipText 悬浮提示文案。
//
// 直角坐标（折线/面积/柱/散点/雷达）：`X 轴标签 · 系列名  值单位`；
// 饼图 / 环形图：每个 Series 就是一个扇区，只显示扇区名 + 数值。
// 旧实现给饼图也拼了 c.Labels[i]，而命中扇区时 i 恒为 0，
// 结果永远显示「第一个扇区名 · 当前扇区名 值」（截图里的「华东 · 华南 61」）。
func (c *Chart) tipText(s, i int) string {
	if s < 0 || s >= len(c.Series) {
		return ""
	}
	ser := c.Series[s]
	var out string
	var v float64

	if c.Type == ChartPie || c.Type == ChartDonut {
		for _, x := range ser.Values {
			v += x
		}
		name := ser.Name
		if name == "" && s < len(c.Labels) {
			name = c.Labels[s]
		}
		out = name
	} else {
		if i >= 0 && i < len(c.Labels) && c.Labels[i] != "" {
			out = c.Labels[i] + " · "
		}
		if i >= 0 && i < len(ser.Values) {
			v = ser.Values[i]
		}
		out += ser.Name
	}

	val := fmtNum(v) + c.Unit
	if out == "" {
		return val
	}
	return out + "  " + val
}
