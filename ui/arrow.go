// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// 子菜单指示三角：用一张预生成的抗锯齿三角形纹理绘制。
//
// 旧实现用「›」字符画箭头，问题有三：
//  1. 字形来自系统字体，粗细/高度/基线随字体与字号漂移，和 12.5px 菜单文字混排会显得偏小、偏上；
//  2. 它是「单角引号」而不是三角，放大后看就是一道细斜线（用户看到「三角图标不对」）；
//  3. 语义上它还是文字，会参与文本测量与折行，窄面板里可能被挤掉。
//
// 这里改用 4×4 超采样生成的三角形纹理（白色 + alpha），绘制时按 Style.Color 着色，
// 尺寸、对齐、颜色完全由布局控制。
const arrowTexDim = 16

// arrowCoverage 返回 size×size 的三角形覆盖率纹理（白色像素 + alpha）。
func arrowCoverage(size int) []byte {
	px := make([]byte, size*size*4)
	// 顶点（右向三角形，垂直居中）：留 1px 边距避免纹理边缘采样到 0
	ax, ay := 3.0, 2.0
	bx, by := 3.0, float64(size-2)
	cx, cy := float64(size-3), float64(size)/2
	const ss = 4
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			hit := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					pxx := float64(x) + (float64(sx)+0.5)/ss
					pyy := float64(y) + (float64(sy)+0.5)/ss
					if insideTri(pxx, pyy, ax, ay, bx, by, cx, cy) {
						hit++
					}
				}
			}
			if hit == 0 {
				continue
			}
			i := (y*size + x) * 4
			px[i], px[i+1], px[i+2] = 255, 255, 255
			px[i+3] = uint8(255 * hit / (ss * ss))
		}
	}
	return px
}

// insideTri 判断点是否在三角形内（同向符号法）。
func insideTri(px, py, ax, ay, bx, by, cx, cy float64) bool {
	d1 := (px-bx)*(ay-by) - (ax-bx)*(py-by)
	d2 := (px-cx)*(by-cy) - (bx-cx)*(py-cy)
	d3 := (px-ax)*(cy-ay) - (cx-ax)*(py-ay)
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

// arrowTexture 懒创建并缓存三角形纹理（GL 资源随窗口销毁）。
func (a *App) arrowTexture() *gpu.Texture {
	if a.arrowTex != nil && a.arrowTex.ID != 0 {
		return a.arrowTex
	}
	a.arrowTex = a.r.NewTextureRGBA(arrowTexDim, arrowTexDim, arrowCoverage(arrowTexDim))
	return a.arrowTex
}

// paintArrow 绘制子菜单指示三角。
func (a *App) paintArrow(r geom.Rect, c gpu.Paint) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	tex := a.arrowTexture()
	if tex == nil {
		return
	}
	a.r.DrawImage(gpu.Image{Rect: r, Tex: tex, Tint: c})
}

// Arrow 子菜单指示三角节点（右向）。
func Arrow(color any) *Node {
	n := &Node{Kind: KArrow, Style: Style{Opacity: 1}}
	n.Style.Width, n.Style.Height = 5, 8
	n.Style.Color = C(color)
	return n
}
