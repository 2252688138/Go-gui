package gpu

import (
	"gogui/internal/geom"
)

const floatsImgVert = 16

// initImage 初始化图片批次（顶点：pos2 uv2 color4 local2 half2 radius4）。
func (r *Renderer) initImage() error {
	p, err := linkProgram(vertShaderImg, fragShaderImg)
	if err != nil {
		return err
	}
	r.imgProg = p
	r.uImgVP = glGetUniformLocation(p, "uViewport")
	r.uImgTex = glGetUniformLocation(p, "uTex")

	glGenVertexArrays(1, &r.imgVAO)
	glGenBuffers(1, &r.imgVBO)
	glBindVertexArray(r.imgVAO)
	glBindBuffer(GL_ARRAY_BUFFER, r.imgVBO)
	sd := int32(floatsImgVert * 4)
	setAttr(0, 2, sd, 0)
	setAttr(1, 2, sd, 8)
	setAttr(2, 4, sd, 16)
	setAttr(3, 2, sd, 32)
	setAttr(4, 2, sd, 40)
	setAttr(5, 4, sd, 48)
	glBindVertexArray(0)

	r.imgData = make([]float32, 0, 4096*floatsImgVert)
	return nil
}

// DrawImage 绘制一张纹理（可圆角、着色）。
func (r *Renderer) DrawImage(im Image) {
	if im.Tex == nil || im.Rect.W <= 0 || im.Rect.H <= 0 {
		return
	}
	if r.cur != batchImage || r.curTex != im.Tex {
		r.flush()
		r.cur = batchImage
		r.curFont = nil
		r.curTex = im.Tex
	}
	if r.imgN+6 > maxVerts {
		r.flush()
		r.cur = batchImage
		r.curTex = im.Tex
	}

	tint := im.Tint
	if tint.A <= 0 {
		tint.A = 1
	}
	if tint.R == 0 && tint.G == 0 && tint.B == 0 {
		tint.R, tint.G, tint.B = 1, 1, 1
	}

	rs := im.Rect
	hx, hy := rs.W/2, rs.H/2
	cx, cy := rs.X+hx, rs.Y+hy
	// 纹理坐标：图像数据首行为顶部，GL v=0 为底部，故上下翻转。
	// 纹理坐标：图像数据首行为顶部（GL v=0 为底部），屏幕 y 向下，
	// 所以屏幕顶部取 v=0 才是正立；Rot 再按 90° 顺时针旋转 UV。
	cs := [6][2]float32{{-hx, -hy}, {hx, -hy}, {hx, hy}, {-hx, -hy}, {hx, hy}, {-hx, hy}}
	uv := [6][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 0}, {1, 1}, {0, 1}}
	for i := 0; i < 6; i++ {
		u, v := uv[i][0], uv[i][1]
		switch im.Rot & 3 {
		case 1: // 90° 顺时针
			u, v = v, 1-u
		case 2: // 180°
			u, v = 1-u, 1-v
		case 3: // 270°（90° 逆时针）
			u, v = 1-v, u
		}
		r.imgData = append(r.imgData,
			cx+cs[i][0], cy+cs[i][1],
			u, v,
			tint.R, tint.G, tint.B, tint.A,
			cs[i][0], cs[i][1],
			hx, hy,
			im.Radius[0], im.Radius[1], im.Radius[2], im.Radius[3],
		)
	}
	r.imgN += 6
	_ = geom.Rect{}
}

// NewTextureRGBAFull 上传整张 RGBA 纹理（NewTextureRGBA 的别名，便于 ui 层调用）。
func (r *Renderer) NewTextureRGBAFull(w, h int, data []byte) *Texture {
	return r.NewTextureRGBA(w, h, data)
}
