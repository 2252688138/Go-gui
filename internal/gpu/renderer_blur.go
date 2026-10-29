package gpu

import (
	"math"

	"gogui/internal/geom"
)

// fboRT 离屏渲染目标（纹理 + 帧缓冲）。
type fboRT struct {
	fbo uint32
	tex *Texture
	w   int32
	h   int32
}

// initBlur 初始化毛玻璃（backdrop-filter）所需的着色器与全屏四边形。
func (r *Renderer) initBlur() error {
	p, err := linkProgram(vertShaderBlur, fragShaderBlur)
	if err != nil {
		return err
	}
	r.blurProg = p
	r.uBlurTex = glGetUniformLocation(p, "uTex")
	r.uBlurTexel = glGetUniformLocation(p, "uTexel")
	r.uBlurRadius = glGetUniformLocation(p, "uRadius")

	glGenVertexArrays(1, &r.blurVAO)
	glGenBuffers(1, &r.blurVBO)
	glBindVertexArray(r.blurVAO)
	glBindBuffer(GL_ARRAY_BUFFER, r.blurVBO)
	quad := []float32{
		-1, -1, 0, 0,
		1, -1, 1, 0,
		1, 1, 1, 1,
		-1, -1, 0, 0,
		1, 1, 1, 1,
		-1, 1, 0, 1,
	}
	glBufferData(GL_ARRAY_BUFFER, quad, GL_STATIC_DRAW)
	setAttr(0, 2, 16, 0)
	setAttr(1, 2, 16, 8)
	glBindVertexArray(0)
	return nil
}

// ensureRT 按目标尺寸准备离屏目标（尺寸不变则复用）。
func (r *Renderer) ensureRT(rt **fboRT, w, h int32) *fboRT {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if *rt != nil && (*rt).w == w && (*rt).h == h {
		return *rt
	}
	if *rt != nil {
		r.freeRT(*rt)
	}
	t := r.NewTextureRGBA(int(w), int(h), nil)
	var fbo uint32
	glGenFramebuffers(1, &fbo)
	glBindFramebuffer(GL_FRAMEBUFFER, fbo)
	glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, t.ID, 0)
	glBindFramebuffer(GL_FRAMEBUFFER, 0)
	*rt = &fboRT{fbo: fbo, tex: t, w: w, h: h}
	return *rt
}

func (r *Renderer) freeRT(rt *fboRT) {
	if rt == nil {
		return
	}
	if rt.fbo != 0 {
		glDeleteFramebuffers(1, &rt.fbo)
	}
	if rt.tex != nil && rt.tex.ID != 0 {
		id := rt.tex.ID
		glDeleteTextures(1, &id)
	}
}

// blurPass 把 src 纹理以双 Kawase 采样写入 dst（同时完成降采样）。
func (r *Renderer) blurPass(src uint32, dst *fboRT, radius float32) {
	glBindFramebuffer(GL_FRAMEBUFFER, dst.fbo)
	glViewport(0, 0, dst.w, dst.h)
	glDisable(GL_SCISSOR_TEST)
	r.clipOn = false
	glUseProgram(r.blurProg)
	glUniform1i(r.uBlurTex, 0)
	glActiveTexture(GL_TEXTURE0)
	glBindTexture(GL_TEXTURE_2D, src)
	glUniform2f(r.uBlurTexel, 1/float32(dst.w), 1/float32(dst.h))
	glUniform1f(r.uBlurRadius, radius)
	glBindVertexArray(r.blurVAO)
	glDrawArrays(GL_TRIANGLES, 0, 6)
	glBindVertexArray(0)
}

// Backdrop 实现 backdrop-filter 毛玻璃：捕获当前帧已绘制内容，
// 逐级降采样做 Kawase 模糊，再按圆角矩形绘制回来，最后叠加 tint 色。
func (r *Renderer) Backdrop(rect geom.Rect, radius [4]float32, blur float32, tint Paint) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if blur <= 0 {
		blur = 16
	}
	r.flush()

	s := r.scale
	x0 := int32(math.Floor(float64(rect.X * s)))
	y0 := int32(math.Floor(float64(float32(r.ph) - rect.Bottom()*s)))
	w := int32(math.Ceil(float64(rect.W * s)))
	h := int32(math.Ceil(float64(rect.H * s)))
	if x0 < 0 {
		w += x0
		x0 = 0
	}
	if y0 < 0 {
		h += y0
		y0 = 0
	}
	if x0+w > r.pw {
		w = r.pw - x0
	}
	if y0+h > r.ph {
		h = r.ph - y0
	}
	if w < 2 || h < 2 {
		return
	}

	// 捕获已绘制背景到纹理
	src := r.ensureRT(&r.srcTex, w, h)
	glActiveTexture(GL_TEXTURE0)
	glBindTexture(GL_TEXTURE_2D, src.tex.ID)
	glCopyTexSubImage2D(GL_TEXTURE_2D, 0, 0, 0, x0, y0, w, h)

	// 逐级降采样（模糊半径随层级自然放大）
	a := r.ensureRT(&r.halfA, w/2+1, h/2+1)
	r.blurPass(src.tex.ID, a, 1.0)
	b := r.ensureRT(&r.quarter, w/4+1, h/4+1)
	r.blurPass(a.tex.ID, b, 1.0)
	// 1/8 层：更接近 CSS blur 的大半径观感（blur 越大层级越深）
	deep := blur >= 14
	if deep {
		a2 := r.ensureRT(&r.halfB, w/8+1, h/8+1)
		r.blurPass(b.tex.ID, a2, 1.0)
		// 回采样到 1/4（双线性放大，平滑过渡）
		r.blurPass(a2.tex.ID, b, 1.0)
	}

	// 回到当前绘制目标（开启超采样时是离屏 RT，否则是呈现目标）
	glBindFramebuffer(GL_FRAMEBUFFER, r.drawFBO)
	glViewport(0, 0, r.pw, r.ph)
	if n := len(r.clipSaved); n > 0 {
		r.setClip(r.clipSaved[n-1])
	} else {
		glDisable(GL_SCISSOR_TEST)
		r.clipOn = false
	}

	r.DrawImage(Image{
		Rect:   rect,
		Tex:    b.tex,
		Tint:   Paint{R: 1, G: 1, B: 1, A: 1},
		Radius: radius,
	})
	if tint.A > 0 {
		r.Rect(Shape{Rect: rect, Radius: radius, Fill: tint})
	}
}

// DestroyBlur 释放离屏目标（Destroy 内部调用）。
func (r *Renderer) DestroyBlur() {
	r.freeRT(r.srcTex)
	r.freeRT(r.halfA)
	r.freeRT(r.halfB)
	r.freeRT(r.quarter)
	r.srcTex, r.halfA, r.halfB, r.quarter = nil, nil, nil, nil
}
