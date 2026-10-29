package gpu

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"gogui/internal/geom"
	"gogui/internal/text"
)

const (
	maxVerts        = 60000
	floatsShapeVert = 22
	floatsTextVert  = 9
	// 顶点数据切片的初始容量（顶点数）：按典型一屏预留，避免开窗即占 ~7.5MB，
	// 同时保证常规页面不会反复扩容（append 在超过 maxVerts 时会先 flush）。
	maxVertsShapeInitial = 8192
	maxVertsTextInitial  = 20480
)

type batchKind uint8

const (
	batchNone batchKind = iota
	batchShape
	batchText
	batchImage
)

// Renderer OpenGL 3.3 批处理 2D 渲染器，实现 Backend。
type Renderer struct {
	width, height float32
	pw, ph        int32
	scale         float32

	shapeProg, textProg uint32
	shapeVAO, shapeVBO  uint32
	textVAO, textVBO    uint32

	shapeData []float32
	textData  []float32
	shapeN    int
	textN     int

	uShapeVP, uTextVP, uTextAtlas, uTextGamma int32
	// textGamma 字形覆盖率 gamma（<1 加粗变实，=1 关闭），见 fragShaderText。
	textGamma float32

	cur       batchKind
	curFont   *text.Font
	curTex    *Texture
	fonts     map[*text.Font]*Texture
	clipSaved []geom.Rect

	White *Texture

	// 图片批次
	imgProg, imgVAO, imgVBO uint32
	imgData                 []float32
	imgN                    int
	uImgVP, uImgTex         int32

	// 离屏模糊（backdrop-filter）
	blurProg    uint32
	blurVAO     uint32
	blurVBO     uint32
	uBlurTex    int32
	uBlurTexel  int32
	uBlurRadius int32
	srcTex      *fboRT
	halfA       *fboRT
	halfB       *fboRT
	quarter     *fboRT

	// 顶点缓冲容量（字节）：仅在超过时重新分配，其余帧用孤儿化+SubData
	shapeBufCap, textBufCap, imgBufCap int
	// 字形图集上传用的中间缓冲（避免每帧分配）
	atlasBuf []byte
	// 当前 scissor 区域（避免重复设置触发无意义的批次切分）
	clipCur geom.Rect
	clipOn  bool
	// targetFBO 呈现目标（0 = 窗口默认帧缓冲；>0 = 离屏目标，DComp 合成用）
	targetFBO uint32
	// drawFBO 本帧实际绑定的绘制目标：开启超采样时指向离屏 RT，否则同 targetFBO
	drawFBO uint32

	// 超采样（SSAA）：整帧先按 ssaa 倍分辨率画进离屏 RT，再由 Resolve 降采样呈现。
	// 圆角/SDF 边缘、描边、阴影与字形都因此得到 ssaa² 倍的子像素覆盖率。
	ssaa          float32
	ssaaRT        *fboRT
	resolveProg   uint32
	resolveVAO    uint32
	resolveVBO    uint32
	uResolveTex   int32
	uResolveTexel int32
	uResolveSpan  int32
	uResolveSharp int32
}

// NewRenderer 创建渲染器。
func NewRenderer() *Renderer {
	return &Renderer{scale: 1, ssaa: 1, fonts: make(map[*text.Font]*Texture), textGamma: defaultTextGamma()}
}

// defaultTextGamma 字形覆盖率 gamma 默认值（仅对亮字生效，见 fragShaderText）。
//
// 浏览器（Skia/DirectWrite）在 sRGB 空间混合前会对覆盖率做 gamma 校正，等效于让中间
// 覆盖率变大：亮度 0.5 的像素在浏览器里看起来接近 0.7。当前渲染器是 sRGB 空间直混，
// 同样的覆盖率就显得偏淡偏虚，尤其小字号中文的横竖笔。实测取 1/1.8 ≈ 0.55
// 与浏览器观感最接近；GOGUI_TEXT_GAMMA 可覆盖（1 = 完全关闭）。
func defaultTextGamma() float32 {
	if v := os.Getenv("GOGUI_TEXT_GAMMA"); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil && f > 0 && f <= 4 {
			return float32(f)
		}
	}
	return 0.55
}

// textGammaFor 当前超采样下的字形覆盖率 gamma。
//
// 超采样的降采样是「子样本线性平均」，gamma 补偿作用在平均之后并不会失真，
// 因此不需要（也不应该）按 1/ss 收敛。旧实现让 2× 下的补偿从 0.55 弱化到 0.775，
// 细笔画的覆盖率被压低，文字明显发灰发虚——观感反而比 1:1 更「不清晰」。
// 现在所有倍率统一使用同一补偿值：SSAA 只提升几何/边缘精度，不改变笔画实度。
func (r *Renderer) textGammaFor() float32 {
	return r.textGamma
}

func (r *Renderer) Init() error {
	p1, err := linkProgram(vertShaderShape, fragShaderShape)
	if err != nil {
		return fmt.Errorf("shape shader: %w", err)
	}
	p2, err := linkProgram(vertShaderText, fragShaderText)
	if err != nil {
		return fmt.Errorf("text shader: %w", err)
	}
	r.shapeProg, r.textProg = p1, p2
	r.uShapeVP = glGetUniformLocation(p1, "uViewport")
	r.uTextVP = glGetUniformLocation(p2, "uViewport")
	r.uTextAtlas = glGetUniformLocation(p2, "uAtlas")
	r.uTextGamma = glGetUniformLocation(p2, "uTextGamma")

	glGenVertexArrays(1, &r.shapeVAO)
	glGenBuffers(1, &r.shapeVBO)
	glBindVertexArray(r.shapeVAO)
	glBindBuffer(GL_ARRAY_BUFFER, r.shapeVBO)
	sd := int32(floatsShapeVert * 4)
	setAttr(0, 2, sd, 0)
	setAttr(1, 2, sd, 8)
	setAttr(2, 4, sd, 16)
	setAttr(3, 4, sd, 32)
	setAttr(4, 2, sd, 48)
	setAttr(5, 4, sd, 56)
	setAttr(6, 4, sd, 72)

	glGenVertexArrays(1, &r.textVAO)
	glGenBuffers(1, &r.textVBO)
	glBindVertexArray(r.textVAO)
	glBindBuffer(GL_ARRAY_BUFFER, r.textVBO)
	td := int32(floatsTextVert * 4)
	setAttr(0, 2, td, 0)
	setAttr(1, 2, td, 8)
	setAttr(2, 4, td, 16)
	setAttr(3, 1, td, 32)
	glBindVertexArray(0)

	glDisable(GL_DEPTH_TEST)
	glDisable(GL_CULL_FACE)
	glEnable(GL_BLEND)
	glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA)

	white := make([]byte, 4*4)
	for i := range white {
		white[i] = 255
	}
	r.White = r.NewTextureRGBA(2, 2, white)

	// 顶点缓冲初始容量按「典型一屏」预留，而不是按硬上限 maxVerts 预留：
	// 旧实现一开窗就分配 60000×22×4B ≈ 5.3MB + 60000×9×4B ≈ 2.2MB 的 Go 切片，
	// 而实际一屏通常只有几百到几千个顶点。缓冲区会按需扩容（上限仍为 maxVerts）。
	r.shapeData = make([]float32, 0, maxVertsShapeInitial*floatsShapeVert)
	r.textData = make([]float32, 0, maxVertsTextInitial*floatsTextVert)
	if err := r.initImage(); err != nil {
		return err
	}
	if err := r.initBlur(); err != nil {
		return err
	}
	if err := r.initResolve(); err != nil {
		return err
	}
	return nil
}

func setAttr(loc uint32, size int32, stride, offset int32) {
	glEnableVertexAttribArray(loc)
	glVertexAttribPointer(loc, size, GL_FLOAT, 0, stride, offset)
}

func (r *Renderer) NewTextureR8(w, h int, data []byte) *Texture {
	var id uint32
	glGenTextures(1, &id)
	glBindTexture(GL_TEXTURE_2D, id)
	glPixelStorei(GL_UNPACK_ALIGNMENT, 1)
	glTexImage2D(GL_TEXTURE_2D, 0, GL_R8, int32(w), int32(h), GL_RED, GL_UNSIGNED_BYTE, data)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_TEXTURE_WRAP_CLAMP_TO_EDGE)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_TEXTURE_WRAP_CLAMP_TO_EDGE)
	return &Texture{ID: id, W: w, H: h, R8: true}
}

func (r *Renderer) UpdateTextureR8(t *Texture, x, y, w, h int, data []byte) {
	if t == nil || len(data) == 0 {
		return
	}
	glBindTexture(GL_TEXTURE_2D, t.ID)
	glPixelStorei(GL_UNPACK_ALIGNMENT, 1)
	glTexSubImage2D(GL_TEXTURE_2D, 0, int32(x), int32(y), int32(w), int32(h), GL_RED, GL_UNSIGNED_BYTE, data)
}

func (r *Renderer) NewTextureRGBA(w, h int, data []byte) *Texture {
	var id uint32
	glGenTextures(1, &id)
	glBindTexture(GL_TEXTURE_2D, id)
	glPixelStorei(GL_UNPACK_ALIGNMENT, 1)
	glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA8, int32(w), int32(h), GL_RGBA, GL_UNSIGNED_BYTE, data)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_TEXTURE_WRAP_CLAMP_TO_EDGE)
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_TEXTURE_WRAP_CLAMP_TO_EDGE)
	return &Texture{ID: id, W: w, H: h}
}

func (r *Renderer) Resize(pw, ph int32, scale float32) {
	if pw < 1 {
		pw = 1
	}
	if ph < 1 {
		ph = 1
	}
	r.pw, r.ph = pw, ph
	if scale <= 0 {
		scale = 1
	}
	r.scale = scale
	r.width = float32(pw) / scale
	r.height = float32(ph) / scale
	glViewport(0, 0, pw, ph)
}

// Bounds 逻辑尺寸。
func (r *Renderer) Bounds() geom.Rect { return geom.Rect{0, 0, r.width, r.height} }

// SetTarget 切换绘制目标（0 = 窗口默认帧缓冲，>0 = 离屏 FBO，用于 DComp 合成）。
func (r *Renderer) SetTarget(fbo uint32, pw, ph int32) {
	if pw <= 0 || ph <= 0 {
		return
	}
	r.targetFBO = fbo
	r.pw, r.ph = pw, ph
	r.width = float32(pw) / r.scale
	r.height = float32(ph) / r.scale
}

// Scale DPI 缩放。
func (r *Renderer) Scale() float32 { return r.scale }

// SetSuperSample 设置整帧超采样倍数（1 = 关闭，推荐 1.5~2）。
// 调用后下一帧生效：绘制目标重建为 pw*ssaa × ph*ssaa，绘制结束时由 Resolve 降采样呈现。
// 倍率变化会重建离屏目标，因此不要在每帧内调整。
func (r *Renderer) SetSuperSample(n float32) {
	if n < 1 {
		n = 1
	}
	if n > 2 {
		n = 2
	}
	if n == r.ssaa {
		return
	}
	r.ssaa = n
	r.dropSSAART()
}

// SuperSample 返回当前超采样倍数。
func (r *Renderer) SuperSample() float32 { return r.ssaa }

// dropSSAART 丢弃离屏超采样目标（倍率/尺寸变化时调用）。
func (r *Renderer) dropSSAART() {
	if r.ssaaRT != nil {
		r.freeRT(r.ssaaRT)
		r.ssaaRT = nil
	}
}

// ensureSSAA 准备超采样离屏目标并校验完整性；不可用时返回 false，绘制退回呈现目标。
func (r *Renderer) ensureSSAA() bool {
	if r.ssaa <= 1.01 || r.pw <= 0 || r.ph <= 0 {
		return false
	}
	rt := r.ensureRT(&r.ssaaRT, r.pw, r.ph)
	if rt == nil {
		return false
	}
	glBindFramebuffer(GL_FRAMEBUFFER, rt.fbo)
	if glCheckFramebufferStatus(GL_FRAMEBUFFER) != GL_FRAMEBUFFER_COMPLETE {
		glBindFramebuffer(GL_FRAMEBUFFER, r.targetFBO)
		r.dropSSAART()
		return false
	}
	return true
}

// Resolve 把超采样离屏内容降采样写入呈现目标（targetFBO，0 = 窗口帧缓冲）。
// 未开启超采样时是空操作。必须在 End() 之后、Swap/EndFrame 之前调用。
func (r *Renderer) Resolve() {
	if r.ssaa <= 1.01 || r.ssaaRT == nil || r.ssaaRT.tex == nil {
		return
	}
	ss := r.ssaa
	dw := int32(float32(r.pw) / ss)
	dh := int32(float32(r.ph) / ss)
	if dw < 1 || dh < 1 {
		return
	}
	glBindFramebuffer(GL_FRAMEBUFFER, r.targetFBO)
	glViewport(0, 0, dw, dh)
	glDisable(GL_SCISSOR_TEST)
	// 降采样是整屏覆盖写：关掉混合，避免半透明背景与目标里的旧内容再次混合
	glDisable(GL_BLEND)
	r.clipOn = false
	glUseProgram(r.resolveProg)
	glUniform1i(r.uResolveTex, 0)
	glUniform2f(r.uResolveTexel, 1/float32(r.ssaaRT.w), 1/float32(r.ssaaRT.h))
	glUniform1f(r.uResolveSpan, float32(r.ssaaRT.w)/float32(dw))
	glUniform1f(r.uResolveSharp, ssaaSharpen())
	glActiveTexture(GL_TEXTURE0)
	glBindTexture(GL_TEXTURE_2D, r.ssaaRT.tex.ID)
	glBindVertexArray(r.resolveVAO)
	glDrawArrays(GL_TRIANGLES, 0, 6)
	glBindVertexArray(0)
	// 恢复混合（后续帧的常规绘制依赖它）
	glEnable(GL_BLEND)
}

// ssaaSharpen 降采样后的锐化强度（默认 0.2）。
//
// 整帧 box 降采样是「数学上正确」的，但 1px 级别的横竖笔与 1px 描边被平均后对比度下降，
// 人眼主观上会觉得「超高清还不如 1:1 清晰」。这里补一道轻度 unsharp 把被平滑掉的
// 高频成分找回一部分：0 = 关闭，0.15~0.3 为推荐区间，GOGUI_SS_SHARPEN 可覆盖。
func ssaaSharpen() float32 {
	if v := os.Getenv("GOGUI_SS_SHARPEN"); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil && f >= 0 && f <= 1 {
			return float32(f)
		}
	}
	return 0.4
}

// initResolve 初始化降采样着色器与全屏四边形。
func (r *Renderer) initResolve() error {
	p, err := linkProgram(vertShaderBlur, fragShaderResolve)
	if err != nil {
		return err
	}
	r.resolveProg = p
	r.uResolveTex = glGetUniformLocation(p, "uTex")
	r.uResolveTexel = glGetUniformLocation(p, "uTexel")
	r.uResolveSpan = glGetUniformLocation(p, "uSpan")
	r.uResolveSharp = glGetUniformLocation(p, "uSharpen")

	glGenVertexArrays(1, &r.resolveVAO)
	glGenBuffers(1, &r.resolveVBO)
	glBindVertexArray(r.resolveVAO)
	glBindBuffer(GL_ARRAY_BUFFER, r.resolveVBO)
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

func (r *Renderer) Begin(fill Paint) {
	r.shapeData = r.shapeData[:0]
	r.textData = r.textData[:0]
	r.shapeN, r.textN = 0, 0
	r.imgN = 0
	r.cur = batchNone
	r.curFont = nil
	r.curTex = nil
	r.clipSaved = r.clipSaved[:0]
	r.clipOn = false
	r.clipCur = geom.Rect{}
	// 开启超采样时整帧画进高分辨率离屏目标，End 之后由 Resolve 降采样呈现
	r.drawFBO = r.targetFBO
	if r.ensureSSAA() {
		r.drawFBO = r.ssaaRT.fbo
	}
	glBindFramebuffer(GL_FRAMEBUFFER, r.drawFBO)
	glViewport(0, 0, r.pw, r.ph)
	glDisable(GL_SCISSOR_TEST)
	glClearColor(fill.R, fill.G, fill.B, fill.A)
	glClear(GL_COLOR_BUFFER_BIT)
}

func (r *Renderer) End() { r.flush() }

// Destroy 释放渲染资源。
func (r *Renderer) Destroy() {
	r.DestroyBlur()
	r.dropSSAART()
	if r.resolveVBO != 0 {
		glDeleteBuffers(1, &r.resolveVBO)
	}
	if r.resolveVAO != 0 {
		glDeleteVertexArrays(1, &r.resolveVAO)
	}
	if r.resolveProg != 0 {
		glDeleteProgram(r.resolveProg)
	}
	if r.imgVBO != 0 {
		glDeleteBuffers(1, &r.imgVBO)
	}
	if r.imgVAO != 0 {
		glDeleteVertexArrays(1, &r.imgVAO)
	}
	if r.imgProg != 0 {
		glDeleteProgram(r.imgProg)
	}
	if r.blurVBO != 0 {
		glDeleteBuffers(1, &r.blurVBO)
	}
	if r.blurVAO != 0 {
		glDeleteVertexArrays(1, &r.blurVAO)
	}
	if r.blurProg != 0 {
		glDeleteProgram(r.blurProg)
	}
	if r.shapeVBO != 0 {
		glDeleteBuffers(1, &r.shapeVBO)
	}
	if r.textVBO != 0 {
		glDeleteBuffers(1, &r.textVBO)
	}
	if r.shapeVAO != 0 {
		glDeleteVertexArrays(1, &r.shapeVAO)
	}
	if r.textVAO != 0 {
		glDeleteVertexArrays(1, &r.textVAO)
	}
	if r.shapeProg != 0 {
		glDeleteProgram(r.shapeProg)
	}
	if r.textProg != 0 {
		glDeleteProgram(r.textProg)
	}
}

func (r *Renderer) ensureKind(k batchKind, f *text.Font) {
	if r.cur != k || (k == batchText && f != r.curFont) || (k == batchImage && r.curTex == nil) {
		r.flush()
		r.cur = k
		r.curFont = f
	}
}

func (r *Renderer) flush() {
	switch r.cur {
	case batchShape:
		if r.shapeN > 0 {
			glUseProgram(r.shapeProg)
			glUniform2f(r.uShapeVP, r.width, r.height)
			glBindVertexArray(r.shapeVAO)
			r.uploadVerts(&r.shapeVBO, &r.shapeBufCap, r.shapeData)
			glDrawArrays(GL_TRIANGLES, 0, int32(r.shapeN))
		}
		r.shapeData = r.shapeData[:0]
		r.shapeN = 0
	case batchText:
		if r.textN > 0 && r.curFont != nil {
			tex := r.fontTexture(r.curFont)
			glUseProgram(r.textProg)
			glUniform2f(r.uTextVP, r.width, r.height)
			glUniform1i(r.uTextAtlas, 0)
			glUniform1f(r.uTextGamma, r.textGammaFor())
			glActiveTexture(GL_TEXTURE0)
			glBindTexture(GL_TEXTURE_2D, tex.ID)
			glBindVertexArray(r.textVAO)
			r.uploadVerts(&r.textVBO, &r.textBufCap, r.textData)
			glDrawArrays(GL_TRIANGLES, 0, int32(r.textN))
		}
		r.textData = r.textData[:0]
		r.textN = 0
	case batchImage:
		if r.imgN > 0 && r.curTex != nil {
			glUseProgram(r.imgProg)
			glUniform2f(r.uImgVP, r.width, r.height)
			glUniform1i(r.uImgTex, 0)
			glActiveTexture(GL_TEXTURE0)
			glBindTexture(GL_TEXTURE_2D, r.curTex.ID)
			glBindVertexArray(r.imgVAO)
			r.uploadVerts(&r.imgVBO, &r.imgBufCap, r.imgData)
			glDrawArrays(GL_TRIANGLES, 0, int32(r.imgN))
		}
		r.imgData = r.imgData[:0]
		r.imgN = 0
		r.curTex = nil
	}
}

// uploadVerts 上传顶点数据。
// 旧实现每次都 GlBufferData(数据) 重新分配整块缓冲，会反复触发驱动分配 +
// 与上一帧绘制同步；现在只在容量不足时扩容，其余帧孤儿化后 GlBufferSubData 覆盖，
// 大幅减少滚动/动画时的驱动开销。
func (r *Renderer) uploadVerts(vbo *uint32, capBytes *int, data []float32) {
	if len(data) == 0 {
		return
	}
	n := len(data) * 4
	glBindBuffer(GL_ARRAY_BUFFER, *vbo)
	if *capBytes < n {
		*capBytes = n + n/2 + 4096
	}
	// 孤儿化：解除与上一帧的依赖，避免驱动等待上一帧用完这块缓冲
	glBufferAlloc(GL_ARRAY_BUFFER, *capBytes, GL_STREAM_DRAW)
	glBufferSubData(GL_ARRAY_BUFFER, 0, data)
}

func (r *Renderer) fontTexture(f *text.Font) *Texture {
	t, ok := r.fonts[f]
	if !ok {
		dim := f.AtlasDimNow()
		t = r.NewTextureRGBA(dim, dim, f.Atlas)
		f.TakeDirty()
		f.TakeReset()
		r.fonts[f] = t
		return t
	}
	// 图集被重建/扩容/清空（DPI 变化、写满）：丢弃旧纹理重新上传
	if f.TakeReset() {
		r.DeleteTexture(t)
		dim := f.AtlasDimNow()
		t = r.NewTextureRGBA(dim, dim, f.Atlas)
		f.TakeDirty()
		r.fonts[f] = t
		return t
	}
	if x0, y0, x1, y1, dirty := f.TakeDirty(); dirty {
		r.uploadAtlasRegion(f, t, x0, y0, x1, y1)
	}
	return t
}

// uploadAtlasRegion 只上传图集的脏矩形区域。
// 以前每光栅化一个新字形就把 2048²（16MB）的整张图集重传，滚动时新文字不断出现
// 会连续触发全量上传，是明显卡顿的主因；现在按行拼接后一次 glTexSubImage2D 提交。
func (r *Renderer) uploadAtlasRegion(f *text.Font, t *Texture, x0, y0, x1, y1 int) {
	dim := f.AtlasDimNow()
	if t == nil || t.ID == 0 || dim <= 0 || len(f.Atlas) < dim*dim*4 {
		return
	}
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > dim {
		x1 = dim
	}
	if y1 > dim {
		y1 = dim
	}
	w, rows := x1-x0, y1-y0
	if w <= 0 || rows <= 0 {
		return
	}

	glBindTexture(GL_TEXTURE_2D, t.ID)
	glPixelStorei(GL_UNPACK_ALIGNMENT, 1)

	// 整行范围：内存连续，直接上传
	if x0 == 0 && x1 == dim {
		off := y0 * dim * 4
		glTexSubImage2D(GL_TEXTURE_2D, 0, 0, int32(y0), int32(dim), int32(rows), GL_RGBA, GL_UNSIGNED_BYTE, f.Atlas[off:])
		return
	}

	n := rows * w * 4
	if cap(r.atlasBuf) < n {
		r.atlasBuf = make([]byte, n)
	}
	buf := r.atlasBuf[:n]
	stride := dim * 4
	rowBytes := w * 4
	for i := 0; i < rows; i++ {
		src := (y0+i)*stride + x0*4
		copy(buf[i*rowBytes:(i+1)*rowBytes], f.Atlas[src:src+rowBytes])
	}
	glTexSubImage2D(GL_TEXTURE_2D, 0, int32(x0), int32(y0), int32(w), int32(rows), GL_RGBA, GL_UNSIGNED_BYTE, buf)
}

// UpdateTextureRGBA 局部更新 RGBA 纹理。
func (r *Renderer) UpdateTextureRGBA(t *Texture, x, y, w, h int, data []byte) {
	if t == nil || len(data) == 0 {
		return
	}
	glBindTexture(GL_TEXTURE_2D, t.ID)
	glPixelStorei(GL_UNPACK_ALIGNMENT, 1)
	glTexSubImage2D(GL_TEXTURE_2D, 0, int32(x), int32(y), int32(w), int32(h), GL_RGBA, GL_UNSIGNED_BYTE, data)
}

// DeleteTexture 释放一张纹理（画面尺寸变化、片源切换时调用）。
func (r *Renderer) DeleteTexture(t *Texture) {
	if t == nil || t.ID == 0 {
		return
	}
	id := t.ID
	glDeleteTextures(1, &id)
	t.ID = 0
}

// ReleaseFont 释放字体对应的图集纹理（字体实例被缓存回收时调用）。
func (r *Renderer) ReleaseFont(f *text.Font) {
	if f == nil {
		return
	}
	if t, ok := r.fonts[f]; ok {
		r.DeleteTexture(t)
		delete(r.fonts, f)
	}
}

func (r *Renderer) setClip(c geom.Rect) {
	// 同一矩形重复设置（嵌套滚动容器常见）不打断批次
	if r.clipOn && r.clipCur == c {
		return
	}
	r.flush()
	s := r.scale
	x := int32(c.X * s)
	y := int32(float32(r.ph) - c.Bottom()*s)
	w := int32(c.W * s)
	h := int32(c.H * s)
	glEnable(GL_SCISSOR_TEST)
	glScissor(x, y, w, h)
	r.clipOn, r.clipCur = true, c
}

func (r *Renderer) clearClip() {
	if !r.clipOn {
		return
	}
	r.flush()
	glDisable(GL_SCISSOR_TEST)
	r.clipOn = false
}

// Rect 立即绘制一个 Shape。
func (r *Renderer) Rect(sh Shape) { r.pushShape(sh) }

// TextRun 立即绘制一段文字。
func (r *Renderer) TextRun(t TextRun) { r.pushText(t) }

func (r *Renderer) pushShape(sh Shape) {
	rs := sh.Rect
	if rs.W <= 0 || rs.H <= 0 {
		return
	}
	r.ensureKind(batchShape, nil)
	s := r.scale

	if sh.ShadowColor.A > 0 {
		sp := sh.ShadowSpread
		blur := sh.ShadowBlur
		if blur <= 0 {
			blur = 1
		}
		base := rs.Offset(sh.ShadowOffsetX, sh.ShadowOffsetY)
		outer := geom.Rect{base.X - sp - blur, base.Y - sp - blur, base.W + (sp+blur)*2, base.H + (sp+blur)*2}
		var rad [4]float32
		for i := 0; i < 4; i++ {
			rad[i] = sh.Radius[i] + sp
		}
		r.appendQuad(outer, rad, Paint{}, Paint{A: -1}, blur*1.6, 2, 0)
	}

	if sh.Grad.On {
		r.appendQuad(rs, sh.Radius, sh.Grad.From, sh.Grad.To, 0.7, 0, 0)
	} else if sh.Fill.A > 0 {
		r.appendQuad(rs, sh.Radius, sh.Fill, Paint{A: -1}, 0.7, 0, 0)
	}

	if sh.Stroke > 0 && sh.StrokeColor.A > 0 {
		r.appendQuad(rs.InsetAll(sh.Stroke*0.5), sh.Radius, sh.StrokeColor, Paint{A: -1}, 0.7, 1, sh.Stroke)
	}
	_ = s
}

func (r *Renderer) appendQuad(rs geom.Rect, rad [4]float32, c1, c2 Paint, feather, mode, stroke float32) {
	// 填充/描边的 SDF 边缘过渡带宽度按「设备像素」恒定：超采样时画布分辨率是 ss 倍，
	// 若仍按逻辑像素给 0.7，过渡带在设备空间就变成 0.7*ss 像素，降采样后边缘反而更软，
	// 白白吃掉超采样带来的锐度。这里除以 ss，让 1.25×/1.5×/2× 的边缘都比 1:1 更干净。
	if r.ssaa > 1.01 && mode < 1.5 && feather > 0 {
		feather /= r.ssaa
	}
	if r.shapeN+6 > maxVerts {
		r.flush()
		r.cur = batchShape
	}
	hx, hy := rs.W/2, rs.H/2
	cx, cy := rs.X+hx, rs.Y+hy
	s := r.scale
	cs := [6][2]float32{{-hx, -hy}, {hx, -hy}, {hx, hy}, {-hx, -hy}, {hx, hy}, {-hx, hy}}
	for _, c := range cs {
		r.shapeData = append(r.shapeData,
			(cx+c[0])*1, (cy+c[1])*1,
			c[0], c[1],
			c1.R, c1.G, c1.B, c1.A,
			c2.R, c2.G, c2.B, c2.A,
			hx, hy,
			rad[0], rad[1], rad[2], rad[3],
			feather, mode, stroke, 0,
		)
	}
	r.shapeN += 6
	_ = s
}

func (r *Renderer) pushText(t TextRun) {
	f := t.Font
	if f == nil || t.Text == "" {
		return
	}
	r.ensureKind(batchText, f)

	sc := f.Scale()
	penX := t.X * sc
	baseY := t.Y*sc + f.Ascent*sc
	c := t.Color

	for _, ch := range t.Text {
		g := f.GlyphFor(ch)
		if !g.Valid {
			penX += g.Advance
			continue
		}
		x0 := float32(math.Round(float64(penX + g.Left)))
		y0 := float32(math.Round(float64(baseY + g.Top)))
		x1 := x0 + g.W
		y1 := y0 + g.H

		if r.textN+6 > maxVerts {
			r.flush()
			r.cur = batchText
			r.curFont = f
		}
		v := [6][2]float32{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y0}, {x1, y1}, {x0, y1}}
		uv := [6][2]float32{{g.U0, g.V1}, {g.U1, g.V1}, {g.U1, g.V0}, {g.U0, g.V1}, {g.U1, g.V0}, {g.U0, g.V0}}
		mode := float32(0)
		if g.Color {
			mode = 1
		}
		for i := 0; i < 6; i++ {
			r.textData = append(r.textData, v[i][0]/sc, v[i][1]/sc, uv[i][0], uv[i][1], c.R, c.G, c.B, c.A, mode)
		}
		r.textN += 6
		penX += g.Advance
	}
}

// Draw 实现 Backend 接口。
func (r *Renderer) Draw(instrs []Instr) {
	for _, in := range instrs {
		switch in.Kind {
		case InstrShape:
			r.pushShape(in.Shape)
		case InstrText:
			r.pushText(in.Text)
		case InstrClipPush:
			r.clipSaved = append(r.clipSaved, in.Clip)
			r.setClip(in.Clip)
		case InstrClipPop:
			if n := len(r.clipSaved); n > 0 {
				r.clipSaved = r.clipSaved[:n-1]
			}
			if n := len(r.clipSaved); n > 0 {
				r.setClip(r.clipSaved[n-1])
			} else {
				r.clearClip()
			}
		}
	}
}

