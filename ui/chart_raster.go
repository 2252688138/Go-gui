// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"math"

	"gogui/internal/gpu"
)

// 图表的软件光栅层：GPU 后端只有「圆角矩形 / 字形 / 图片」三类图元，
// 弧线、斜线、多边形无法直接用顶点画出，因此图表把「矢量几何」
// 用 CPU 光栅成一张 RGBA 纹理（直通道 alpha），再一次性 DrawImage 上传。
//
// 超采样：画布内部按 chartSS 倍分辨率光栅（默认 2×，即每逻辑像素 4 个样本），
// 最后 box 降采样回来，边缘达到抗锯齿效果。中间缓冲是临时的，
// 上传完成后由 GC 回收，常驻显存只有一张成品纹理。

// chartSS 画布超采样倍数（2 = 每逻辑像素 4 样本）。
const chartSS = 2

// fpt 画布坐标点（逻辑像素，y 向下）。
type fpt struct {
	X, Y float32
}

// canvas 预乘 alpha 的 RGBA 画布（直通道 alpha 在输出时还原）。
type canvas struct {
	w, h int // 逻辑尺寸
	sw   int // 设备宽度 = w*ss
	sh   int // 设备高度 = h*ss
	pix  []byte
}

// newCanvas 创建画布。
func newCanvas(w, h int) *canvas {
	if w <= 0 || h <= 0 {
		return nil
	}
	sw, sh := w*chartSS, h*chartSS
	return &canvas{w: w, h: h, sw: sw, sh: sh, pix: make([]byte, sw*sh*4)}
}

// px 把逻辑坐标换到设备坐标。
func (c *canvas) px(v float32) float32 { return v * chartSS }

// blend 在设备像素 (x,y) 上叠加一个覆盖率为 cov 的预乘颜色。
func (c *canvas) blend(x, y int, p gpu.Paint, cov float32) {
	if x < 0 || y < 0 || x >= c.sw || y >= c.sh || cov <= 0 {
		return
	}
	sa := p.A * cov
	if sa <= 0 {
		return
	}
	if sa > 1 {
		sa = 1
	}
	i := (y*c.sw + x) * 4
	da := float32(c.pix[i+3]) / 255
	k := 1 - sa
	c.pix[i] = clamp8(float32(c.pix[i]) + p.R*255*sa)
	c.pix[i+1] = clamp8(float32(c.pix[i+1]) + p.G*255*sa)
	c.pix[i+2] = clamp8(float32(c.pix[i+2]) + p.B*255*sa)
	c.pix[i+3] = clamp8((sa + da*k) * 255)
}

func clamp8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// hspan 水平填充一段（端点按亚像素覆盖率渐变，横向也带抗锯齿）。
func (c *canvas) hspan(y int, x0, x1 float32, p gpu.Paint) {
	if y < 0 || y >= c.sh || x1 <= x0 {
		return
	}
	if x0 < 0 {
		x0 = 0
	}
	if x1 > float32(c.sw) {
		x1 = float32(c.sw)
	}
	if x1 <= x0 {
		return
	}
	ia, ib := int(x0), int(x1)
	for x := ia; x <= ib && x < c.sw; x++ {
		cov := minf(x1, float32(x+1)) - maxf(x0, float32(x))
		c.blend(x, y, p, cov)
	}
}

// vspan 垂直填充一段。
func (c *canvas) vspan(x int, y0, y1 float32, p gpu.Paint) {
	if x < 0 || x >= c.sw || y1 <= y0 {
		return
	}
	if y0 < 0 {
		y0 = 0
	}
	if y1 > float32(c.sh) {
		y1 = float32(c.sh)
	}
	if y1 <= y0 {
		return
	}
	for y := int(y0); y <= int(y1) && y < c.sh; y++ {
		cov := minf(y1, float32(y+1)) - maxf(y0, float32(y))
		c.blend(x, y, p, cov)
	}
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// fillPoly 填充多边形（扫描线，每设备行 4 个子样本 + 端点覆盖率）。
func (c *canvas) fillPoly(pts []fpt, p gpu.Paint) {
	n := len(pts)
	if n < 3 || p.A <= 0 {
		return
	}
	dp := make([]fpt, n)
	minY, maxY := float32(c.sh), float32(0)
	for i, q := range pts {
		dp[i] = fpt{c.px(q.X), c.px(q.Y)}
		minY = minf(minY, dp[i].Y)
		maxY = maxf(maxY, dp[i].Y)
	}
	y0, y1 := int(math.Floor(float64(minY))), int(math.Ceil(float64(maxY)))
	if y0 < 0 {
		y0 = 0
	}
	if y1 > c.sh {
		y1 = c.sh
	}
	const sub = 4
	xs := make([]float32, 0, 16)
	for y := y0; y < y1; y++ {
		for s := 0; s < sub; s++ {
			sy := float32(y) + (float32(s)+0.5)/sub
			xs = xs[:0]
			for i := 0; i < n; i++ {
				a := dp[i]
				b := dp[(i+1)%n]
				if (a.Y <= sy) == (b.Y <= sy) {
					continue
				}
				xs = append(xs, a.X+(sy-a.Y)/(b.Y-a.Y)*(b.X-a.X))
			}
			if len(xs) < 2 {
				continue
			}
			sortF32(xs)
			for k := 0; k+1 < len(xs); k += 2 {
				c.hspan(y, xs[k], xs[k+1], p.MulAlpha(1.0/sub))
			}
		}
	}
}

// sortF32 插入排序（交点数量很少，避免额外的 sort 依赖与分配）。
func sortF32(v []float32) {
	for i := 1; i < len(v); i++ {
		x := v[i]
		j := i - 1
		for j >= 0 && v[j] > x {
			v[j+1] = v[j]
			j--
		}
		v[j+1] = x
	}
}

// strokePoly 描边折线（按长轴扫描，线宽按法线投影换算）。
func (c *canvas) strokePoly(pts []fpt, width float32, p gpu.Paint, closed bool) {
	if len(pts) < 2 || p.A <= 0 || width <= 0 {
		return
	}
	hw := width / 2
	last := len(pts) - 1
	for i := 0; i < len(pts); i++ {
		j := i + 1
		if j > last {
			if !closed {
				break
			}
			j = 0
		}
		c.line(pts[i], pts[j], hw, p)
	}
}

// line 一段抗锯齿粗线段。
func (c *canvas) line(a, b fpt, hw float32, p gpu.Paint) {
	x0, y0 := c.px(a.X), c.px(a.Y)
	x1, y1 := c.px(b.X), c.px(b.Y)
	wh := c.px(hw)
	dx, dy := x1-x0, y1-y0
	if math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
		if dx == 0 {
			return
		}
		if dx < 0 {
			x0, x1, y0, y1 = x1, x0, y1, y0
			dx, dy = -dx, -dy
		}
		len := math.Hypot(float64(dx), float64(dy))
		h := float32(float64(wh) * math.Abs(float64(dx)) / len)
		for x := int(x0); x <= int(x1) && x < c.sw; x++ {
			if x < 0 {
				continue
			}
			t := (float32(x) + 0.5 - x0) / dx
			if t < 0 || t > 1 {
				continue
			}
			cy := y0 + dy*t
			c.vspan(x, cy-h, cy+h, p)
		}
		return
	}
	if dy < 0 {
		x0, x1, y0, y1 = x1, x0, y1, y0
		dx, dy = -dx, -dy
	}
	len := math.Hypot(float64(dx), float64(dy))
	w := float32(float64(wh) * math.Abs(float64(dy)) / len)
	for y := int(y0); y <= int(y1) && y < c.sh; y++ {
		if y < 0 {
			continue
		}
		t := (float32(y) + 0.5 - y0) / dy
		if t < 0 || t > 1 {
			continue
		}
		cx := x0 + dx*t
		c.hspan(y, cx-w, cx+w, p)
	}
}

// disc 实心圆（2×2 子样本）。
func (c *canvas) disc(cx, cy, r float32, p gpu.Paint) {
	if r <= 0 || p.A <= 0 {
		return
	}
	dx, dy := c.px(cx), c.px(cy)
	dr := c.px(r)
	x0, x1 := int(dx-dr-1), int(dx+dr+1)
	y0, y1 := int(dy-dr-1), int(dy+dr+1)
	const sub = 2
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			hit := 0
			for sy := 0; sy < sub; sy++ {
				for sx := 0; sx < sub; sx++ {
					px := float32(x) + (float32(sx)+0.5)/sub
					py := float32(y) + (float32(sy)+0.5)/sub
					if (px-dx)*(px-dx)+(py-dy)*(py-dy) <= dr*dr {
						hit++
					}
				}
			}
			if hit > 0 {
				c.blend(x, y, p, float32(hit)/(sub*sub))
			}
		}
	}
}

// result 降采样成直通道 alpha 的 RGBA（可直接上传纹理）。
func (c *canvas) result() []byte {
	if c == nil {
		return nil
	}
	out := make([]byte, c.w*c.h*4)
	ss := chartSS
	inv := 1.0 / float32(ss*ss)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			var r, g, b, a float32
			for sy := 0; sy < ss; sy++ {
				i := ((y*ss+sy)*c.sw + x*ss) * 4
				for sx := 0; sx < ss; sx++ {
					r += float32(c.pix[i])
					g += float32(c.pix[i+1])
					b += float32(c.pix[i+2])
					a += float32(c.pix[i+3])
					i += 4
				}
			}
			r *= inv
			g *= inv
			b *= inv
			a *= inv
			o := (y*c.w + x) * 4
			if a <= 0.5 {
				continue
			}
			// 预乘 → 直通道
			k := 255 / a
			out[o] = clamp8(r * k)
			out[o+1] = clamp8(g * k)
			out[o+2] = clamp8(b * k)
			out[o+3] = clamp8(a)
		}
	}
	return out
}
