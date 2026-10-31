// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package text 提供字形光栅化与测量。
//
// 光栅化方式参考 gio：不使用 GDI 的提示化位图，而是用 golang.org/x/image/font/opentype
// 直接把 TrueType 轮廓按目标像素尺寸做覆盖率抗锯齿（HintingNone），
// 笔画粗细均匀、边缘平滑，观感与浏览器（DirectWrite 灰度抗锯齿）一致。
//
// 图集为 RGBA8：普通字形写入 (v,v,v,255)，彩色 Emoji 写入真实颜色，
// 渲染时由着色器按 mode 区分（见 gpu/pushText）。
package text

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Glyph 字形在图集中的位置（位图尺寸/偏移均为物理像素）。
type Glyph struct {
	U0, V0, U1, V1 float32
	W, H           float32
	Left, Top      float32
	Advance        float32
	Valid          bool
	// Color 为 true 时图集里存的是彩色像素（Emoji），渲染时不做颜色乘算。
	Color bool
}

// Font 基于 opentype 矢量光栅化的字体。
type Font struct {
	Face   string
	Size   float32 // 逻辑字号
	Weight int32

	scale float32

	Ascent, Descent, LineHeight float32 // 逻辑像素

	mu    sync.RWMutex
	cache map[glyphKey]*Glyph
	// measure 行宽缓存（逻辑像素）：UI 每帧都会重复测量同一批字符串，
	// 逐字查字形 + 反复加锁在长列表/富文本页会吃掉大量时间，命中后只剩一次哈希查找。
	measure map[string]float32
	face    font.Face
	// fallback 主字体缺字形时的后备字体（Symbol / Emoji 单色轮廓）
	fallback     font.Face
	fallbackName string
	// emoji 彩色光栅器（GDI，懒创建）
	emoji *gdiFont

	// 图集（RGBA8），由渲染器上传。
	// 上传采用「脏矩形」策略：只把本次新光栅化的字形区域传给 GL，
	// 而不是每新增一个字形就把整张图集重传一遍。
	Atlas     []byte
	AtlasSize int // 当前图集边长（0 = 尚未分配）
	// AtlasDirty 与旧版兼容：有任何未上传区域时为 true。
	AtlasDirty bool
	// AtlasReset 图集被整体重建/清空/扩容（SetScale、写满），渲染器需重建纹理。
	AtlasReset bool
	// 脏矩形（纹理坐标：x 向右、y 自底向上，x1/y1 为开区间）
	dirtyX0, dirtyY0, dirtyX1, dirtyY1 int
	gridX                              int
	gridY                              int
	rowHeight                          int
}

type glyphKey struct {
	r  rune
	px int32
}

// 字形图集尺寸：从 minAtlasDim 起步，装不下时逐级翻倍，最大 maxAtlasDim。
// 单字体图集 = dim² × 4 字节（256→256KB、512→1MB、1024→4MB）。
// 以前每个字体一律预分配 2048²（16MB），一个页面用到十几种字号/字重就直接吃掉上百 MB。
const (
	minAtlasDim = 256
	maxAtlasDim = 1024
	// AtlasDim 向后兼容：图集最大边长。
	AtlasDim = maxAtlasDim
)

// glyphPad 字形四周留白（纹理像素），供双线性采样使用，避免渗入相邻字形。
const glyphPad = 1

// NewFont 创建字体（纯 Go 光栅化 + GDI Emoji 彩色）。
func NewFont(faceName string, size float32, weight int32, scale float32) *Font {
	if size <= 0 {
		size = 14
	}
	if scale <= 0 {
		scale = 1
	}
	if weight == 0 {
		weight = 400
	}
	f := &Font{
		Face:      faceName,
		Size:      size,
		Weight:    weight,
		scale:     scale,
		cache:     make(map[glyphKey]*Glyph, 512),
		measure:   make(map[string]float32, 256),
		AtlasSize: 0, // 懒分配：真正光栅化时才建图集
	}
	f.buildFace()
	return f
}

// atlasDim 返回当前图集边长（未分配时为最小尺寸）。
func (f *Font) atlasDim() int {
	if f.AtlasSize <= 0 {
		return minAtlasDim
	}
	return f.AtlasSize
}

// AtlasDimNow 返回当前图集边长（供渲染器创建/更新纹理）。
func (f *Font) AtlasDimNow() int { return f.atlasDim() }

// growAtlas 扩容并清空图集（已有字形需重新光栅化，UV 会随之重算）。
// 调用方需持有 f.mu。
func (f *Font) growAtlas(dim int) {
	f.AtlasSize = dim
	f.Atlas = make([]byte, dim*dim*4)
	f.cache = make(map[glyphKey]*Glyph, 1024)
	f.gridX, f.gridY, f.rowHeight = 0, 0, 0
	f.AtlasDirty = false
	f.AtlasReset = true
	f.dirtyX0, f.dirtyY0, f.dirtyX1, f.dirtyY1 = 0, 0, 0, 0
}

// warnCJKOnce 只提示一次：字体加载路径每帧都可能触发，日志不能刷屏。
var warnCJKOnce sync.Once

// warnMissingCJK 主字体不含中日韩字形时提示一次。
//
// 典型场景：容器 / WSL 的 Ubuntu 默认不装 CJK 字体，表现为「英文正常、中文全空白」。
// 这里探测 '中' 的字形索引，给出可执行的修复方式，避免误判为渲染 bug。
func warnMissingCJK(sf *sfnt.Font) {
	if sf == nil {
		return
	}
	var buf sfnt.Buffer
	if idx, err := sf.GlyphIndex(&buf, '中'); err == nil && idx != 0 {
		return
	}
	warnCJKOnce.Do(func() {
		fmt.Fprint(os.Stderr,
			"[gogui] 警告：当前主字体缺少中日韩字形，中文会显示为空白。\n"+
				"  修复：安装中文字体（Debian/Ubuntu: apt install fonts-noto-cjk），\n"+
				"        或用环境变量指定字体文件：GOGUI_FONT_FILE=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc\n")
	})
}

// buildFace 按当前字号与 DPI 重建 opentype face。
func (f *Font) buildFace() {
	px := f.Size * f.scale
	if px < 1 {
		px = 1
	}
	if sf, err := loadSystemFont(f.Face, f.Weight); err == nil {
		warnMissingCJK(sf)
		if face, err := opentype.NewFace(sf, &opentype.FaceOptions{
			Size:    float64(px),
			DPI:     72,
			Hinting: font.HintingNone,
		}); err == nil {
			f.face = face
			m := face.Metrics()
			f.Ascent = float32(m.Ascent) / 64 / f.scale
			f.Descent = float32(m.Descent) / 64 / f.scale
			f.LineHeight = float32(m.Height) / 64 / f.scale
		}
	}
	if f.face == nil {
		f.face = nil
		f.Ascent = f.Size * 0.8
		f.Descent = f.Size * 0.22
		f.LineHeight = f.Size * 1.4
	}

	// 后备字体：平台符号字体（覆盖符号/箭头/部分 emoji 轮廓）。
	// Windows 是 Segoe UI Symbol，Linux 是 DejaVu Sans，macOS 是 Apple Symbols——
	// 以前硬编码成 "Segoe UI Symbol"，非 Windows 平台永远加载失败，后备字体形同虚设。
	if f.fallbackName == "" {
		f.fallbackName = platformSymbolFamily()
	}
	if f.fallbackName != "" {
		if sf, err := loadSystemFont(f.fallbackName, f.Weight); err == nil {
			if face, err := opentype.NewFace(sf, &opentype.FaceOptions{
				Size:    float64(px),
				DPI:     72,
				Hinting: font.HintingNone,
			}); err == nil {
				f.fallback = face
			}
		}
	}
	if f.emoji != nil {
		f.emoji.ensureFont(int32(px + 0.5))
	}
}

// SetScale 在 DPI 变化时重建（图集清空）。
func (f *Font) SetScale(scale float32) {
	if scale <= 0 || scale == f.scale {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scale = scale
	f.buildFace()
	f.cache = make(map[glyphKey]*Glyph, 512)
	// 字号变了，行宽缓存全部失效
	f.measure = make(map[string]float32, 256)
	// 释放旧图集（懒分配：下次光栅化时按最小尺寸重建），并标记纹理需重建
	f.Atlas = nil
	f.AtlasSize = 0
	f.AtlasDirty = false
	f.AtlasReset = true
	f.dirtyX0, f.dirtyY0, f.dirtyX1, f.dirtyY1 = 0, 0, 0, 0
	f.gridX, f.gridY, f.rowHeight = 0, 0, 0
}

// AtlasBytes 返回当前字形图集占用的字节数（懒分配，未使用时为 0）。
func (f *Font) AtlasBytes() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.Atlas)
}

// markDirty 记录一个刚写入图集的字形矩形（纹理坐标，y 自底向上）。
// 调用方需持有 f.mu。
func (f *Font) markDirty(ox, oy, bw, bh int) {
	dim := f.atlasDim()
	y0 := dim - 1 - (oy + bh - 1)
	y1 := dim - oy
	x0, x1 := ox, ox+bw
	if y0 < 0 {
		y0 = 0
	}
	if y1 > dim {
		y1 = dim
	}
	if x0 < 0 {
		x0 = 0
	}
	if x1 > dim {
		x1 = dim
	}
	if y0 >= y1 || x0 >= x1 {
		return
	}
	if !f.AtlasDirty {
		f.dirtyX0, f.dirtyY0, f.dirtyX1, f.dirtyY1 = x0, y0, x1, y1
		f.AtlasDirty = true
		return
	}
	if x0 < f.dirtyX0 {
		f.dirtyX0 = x0
	}
	if y0 < f.dirtyY0 {
		f.dirtyY0 = y0
	}
	if x1 > f.dirtyX1 {
		f.dirtyX1 = x1
	}
	if y1 > f.dirtyY1 {
		f.dirtyY1 = y1
	}
}

// TakeDirty 取出并清空脏矩形（供渲染器做局部 glTexSubImage2D）。
func (f *Font) TakeDirty() (x0, y0, x1, y1 int, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.AtlasDirty {
		return 0, 0, 0, 0, false
	}
	x0, y0, x1, y1 = f.dirtyX0, f.dirtyY0, f.dirtyX1, f.dirtyY1
	f.AtlasDirty = false
	f.dirtyX0, f.dirtyY0, f.dirtyX1, f.dirtyY1 = 0, 0, 0, 0
	return x0, y0, x1, y1, true
}

// TakeReset 取出并清空「图集已整体重建」标记。
func (f *Font) TakeReset() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.AtlasReset {
		return false
	}
	f.AtlasReset = false
	return true
}

// GlyphFor 取（必要时光栅化）字形。
//
// 热路径：命中的字形只在读锁下查一次 map（旧实现每字符都抢一次互斥锁，
// 一屏文本每帧上千次加锁/解锁，是绘制阶段最贵的开销之一）。
// 未命中才升级为写锁光栅化，并在写锁内复查一次避免重复光栅化。
func (f *Font) GlyphFor(r rune) *Glyph {
	f.mu.RLock()
	px := int32(f.Size*f.scale + 0.5)
	if px < 1 {
		px = 1
	}
	key := glyphKey{r, px}
	if g, ok := f.cache[key]; ok {
		f.mu.RUnlock()
		return g
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()
	if g, ok := f.cache[key]; ok {
		return g
	}
	g := f.rasterize(r)
	f.cache[key] = g
	return g
}

func (f *Font) rasterize(r rune) *Glyph {
	// Emoji 优先走彩色位图路径
	if isEmojiRune(r) {
		if g, ok := f.rasterizeEmoji(r); ok {
			return g
		}
	}
	if g, ok := f.rasterizeFace(f.face, r); ok {
		return g
	}
	// 主字体缺字形：尝试后备字体（符号/箭头），最后再试 Emoji 单色轮廓
	if g, ok := f.rasterizeFace(f.fallback, r); ok {
		return g
	}
	if g, ok := f.rasterizeEmoji(r); ok {
		return g
	}
	g := &Glyph{}
	if unicode.IsSpace(r) {
		g.Advance = f.Size * f.scale * 0.5
	} else {
		g.Advance = f.Size * f.scale * 0.9
	}
	return g
}

// rasterizeFace 用 opentype face 光栅化灰度字形并写入 RGBA 图集。
func (f *Font) rasterizeFace(face font.Face, r rune) (*Glyph, bool) {
	if face == nil {
		return nil, false
	}
	dr, mask, maskp, advance, ok := face.Glyph(fixed.Point26_6{}, r)
	if !ok || dr.Empty() {
		return nil, false
	}
	bw, bh := dr.Dx(), dr.Dy()
	if bw <= 0 || bh <= 0 {
		return nil, false
	}
	ox, oy, ok := f.allocCell(bw, bh)
	if !ok {
		return nil, false
	}
	gray := color.GrayModel
	dim := f.AtlasSize
	for y := 0; y < bh; y++ {
		// GL 纹理第 0 行对应 v=0（底部），位图首行是顶部，写入时垂直翻转
		dstRow := dim - 1 - (oy + y)
		if dstRow < 0 || dstRow >= dim {
			continue
		}
		dst := f.Atlas[dstRow*dim*4+ox*4:]
		srcY := maskp.Y + y
		for x := 0; x < bw; x++ {
			v := gray.Convert(mask.At(maskp.X+x, srcY)).(color.Gray).Y
			i := x * 4
			dst[i], dst[i+1], dst[i+2], dst[i+3] = v, v, v, 255
		}
	}
	f.markDirty(ox, oy, bw, bh)
	g := f.finishGlyph(ox, oy, bw, bh, advance, dr.Min.X, dr.Min.Y)
	return g, true
}

// rasterizeEmoji 用 GDI 渲染彩色 Emoji 位图。
func (f *Font) rasterizeEmoji(r rune) (*Glyph, bool) {
	if f.emoji == nil {
		f.emoji = newGdiFont("Segoe UI Emoji")
	}
	px := int32(f.Size*f.scale + 0.5)
	if px < 8 {
		return nil, false
	}
	f.emoji.ensureFont(px)
	img, adv, ok := f.emoji.render(string(r), px)
	if !ok || img == nil {
		return nil, false
	}
	bw, bh := img.W, img.H
	ox, oy, ok2 := f.allocCell(bw, bh)
	if !ok2 {
		return nil, false
	}
	for y := 0; y < bh; y++ {
		dstRow := f.AtlasSize - 1 - (oy + y)
		if dstRow < 0 || dstRow >= f.AtlasSize {
			continue
		}
		dst := f.Atlas[dstRow*f.AtlasSize*4+ox*4:]
		src := img.Pix[y*img.W*4:]
		copy(dst[:bw*4], src[:bw*4])
	}
	f.markDirty(ox, oy, bw, bh)
	g := f.finishGlyph(ox, oy, bw, bh, fixed.Int26_6(adv*64), img.Left, img.Top)
	g.Color = true
	return g, true
}

// allocCell 在图集中分配一块 bw×bh 区域（含 padding）。
// 图集从 256² 起步，装不下逐级翻倍到 1024²；到顶后写满才清空重用。
func (f *Font) allocCell(bw, bh int) (int, int, bool) {
	if f.Atlas == nil {
		f.growAtlas(minAtlasDim)
	}
	need := bw + glyphPad*2
	dim := f.AtlasSize
	if f.gridX+need+1 > dim {
		f.gridX = 0
		f.gridY += f.rowHeight + 1
		f.rowHeight = 0
	}
	if f.gridY+bh+glyphPad*2+1 > dim {
		// 装不下：先尝试扩容（已有字形会重新光栅化），已是最大尺寸才清空重用
		if !f.growOneStep(need, bh+glyphPad*2+2) {
			f.gridX, f.gridY, f.rowHeight = 0, 0, 0
			f.cache = map[glyphKey]*Glyph{}
			for i := range f.Atlas {
				f.Atlas[i] = 0
			}
			// 显存里的旧字形必须一并清除，交给渲染器重建纹理
			f.AtlasReset = true
			f.AtlasDirty = false
			f.dirtyX0, f.dirtyY0, f.dirtyX1, f.dirtyY1 = 0, 0, 0, 0
		}
	}
	ox, oy := f.gridX+glyphPad, f.gridY+glyphPad
	f.gridX += need + 1
	if bh+glyphPad*2 > f.rowHeight {
		f.rowHeight = bh + glyphPad*2
	}
	// 注意：这里不能再置 AtlasDirty——脏区统一由 markDirty 在写入像素后叠加，
	// 否则会退化成「整张图集」级别的脏区（=每次新字形仍全量上传）。
	return ox, oy, true
}

// maxDim 图集最大边长：超高清渲染（scale ≥ 1.75）时字形位图成倍变大，1024²
// 很容易被写满而触发「清空重用」（会让同帧已生成的 UV 失效、出现缺字），故放宽到 2048²。
// 常规 1× 渲染仍维持 1024²（单字体 4MB），避免无谓的显存/堆占用。
func (f *Font) maxDim() int {
	if f.scale >= 1.75 {
		return 2048
	}
	return maxAtlasDim
}

// growOneStep 把图集放大到能容纳 needW×needH 的下一档；已最大或仍放不下返回 false。
func (f *Font) growOneStep(needW, needH int) bool {
	maxDim := f.maxDim()
	want := f.AtlasSize * 2
	if want < minAtlasDim*2 {
		want = minAtlasDim * 2
	}
	for want < needW || want < needH {
		if want >= maxDim {
			return false
		}
		want *= 2
	}
	if want > maxDim {
		want = maxDim
	}
	if want <= f.AtlasSize {
		return false
	}
	f.growAtlas(want)
	return true
}

// finishGlyph 生成 UV（纹素边界）与度量。
//
// UV 必须取「纹素边界」而不是「纹素中心」：绘制时 quad 的宽高就是 bw×bh 个物理像素，
// 且左上角已对齐到整数物理像素（见 gpu.pushText 的 math.Round）。按边界取值时，
// 第 k 个 fragment（中心 x0+k+0.5）映射到的纹素坐标恰好是 ox+k+0.5，正落在纹素中心，
// 双线性采样权重为 1，等价于 1:1 直接拷贝，字形保持光栅化时的锐度。
//
// 旧实现用 (ox+0.5)…(ox+bw-0.5) 的纹素中心对齐，但 quad 并没有内缩半像素，
// 相当于把整块字形横向/纵向各压缩 1 个纹素；每个 fragment 都落在纹素之间，
// 双线性把相邻纹素一混，笔画就被重采样糊掉——这正是「GPU 文字发虚」的主因。
func (f *Font) finishGlyph(ox, oy, bw, bh int, advance fixed.Int26_6, left, top int) *Glyph {
	dim := float32(f.AtlasSize)
	inv := 1.0 / dim
	g := &Glyph{}
	g.U0 = float32(ox) * inv
	g.U1 = float32(ox+bw) * inv
	// 位图首行写在图集顶部：oy 向下增长，纹理 v 轴向上增长，故上下边界要倒过来算。
	g.V1 = float32(f.AtlasSize-oy) * inv
	g.V0 = float32(f.AtlasSize-oy-bh) * inv
	g.W = float32(bw)
	g.H = float32(bh)
	g.Left = float32(left)
	g.Top = float32(top)
	g.Advance = float32(advance) / 64
	if g.Advance == 0 {
		g.Advance = float32(bw)
	}
	g.Valid = true
	return g
}

// measureCacheMax 行宽缓存条数上限：超出后整体丢弃重来，
// 避免长文本/动态数字（计时器、滚动日志）把缓存撑成内存泄漏。
const measureCacheMax = 8192

// Measure 计算字符串行宽（逻辑像素）。
//
// 布局阶段（measureNode）与绘制阶段（paintText）每帧都会测量同一批字符串，
// 这里按字符串整体缓存结果：命中即返回，避免逐字 GlyphFor + 除法。
func (f *Font) Measure(s string) float32 {
	if s == "" {
		return 0
	}
	f.mu.RLock()
	if w, ok := f.measure[s]; ok {
		f.mu.RUnlock()
		return w
	}
	sc := f.scale
	f.mu.RUnlock()

	var w float32
	for _, r := range s {
		w += f.GlyphFor(r).Advance
	}
	if sc <= 0 {
		sc = 1
	}
	out := w / sc

	f.mu.Lock()
	if f.measure == nil || len(f.measure) >= measureCacheMax {
		f.measure = make(map[string]float32, 256)
	}
	f.measure[s] = out
	f.mu.Unlock()
	return out
}

// MeasureRunes 仅测量部分。
func (f *Font) MeasureRunes(rs []rune) float32 {
	var w float32
	for _, r := range rs {
		w += f.GlyphFor(r).Advance
	}
	return w / f.scale
}

// Truncate 按宽度截断并加省略号（fit 为逻辑像素）。
func (f *Font) Truncate(s string, fit float32) string {
	if f.Measure(s) <= fit || fit <= 0 {
		return s
	}
	rs := []rune(s)
	ell := f.Measure("…")
	limit := fit - ell
	var w float32
	n := 0
	for _, r := range rs {
		w += f.GlyphFor(r).Advance / f.scale
		if w > limit {
			break
		}
		n++
	}
	if n <= 0 {
		return "…"
	}
	return string(rs[:n]) + "…"
}

// Scale 返回当前 DPI 缩放。
func (f *Font) Scale() float32 { return f.scale }

// Close 释放资源（opentype face 无需显式释放）。
func (f *Font) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.face = nil
	f.fallback = nil
	f.cache = map[glyphKey]*Glyph{}
	f.measure = nil
	// 释放字形图集（4MB）：字体被缓存回收后不应继续占内存
	f.Atlas = nil
	f.AtlasDirty = false
	f.AtlasReset = true
	f.AtlasSize = 0
	f.gridX, f.gridY, f.rowHeight = 0, 0, 0
	if f.emoji != nil {
		f.emoji.close()
		f.emoji = nil
	}
}

// isEmojiRune 判断是否彩色 Emoji（常见区段）。
func isEmojiRune(r rune) bool {
	switch {
	case r == 0x200D || r == 0xFE0F || r == 0x20E3: // ZWJ / VS16 / 键帽
		return false
	case r >= 0x1F000 && r <= 0x1FAFF: // 表情/交通/补充符号
		return true
	case r >= 0x2600 && r <= 0x27BF: // 杂项符号与装饰
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF: // 国旗区域指示符
		return true
	}
	return false
}

// ---- 系统字体解析（按家族名映射到字体文件）----

var (
	sysFontsMu sync.Mutex
	sysFonts   = map[string]*sfnt.Font{}
)

// fontCandidates 返回候选字体文件（路径有序，前面的优先）。
//
// GOGUI_FONT_FILE 可直接指定主字体文件，优先级最高：
// 容器/WSL 里系统没装中文字体时（表现为「英文正常、中文全空白」），
// 用它指向任意 ttc/ttf（如 WSLg 下的 /mnt/c/Windows/Fonts/msyh.ttc）即可，无需改系统。
//
// 平台差异全部收敛在 platformFontCandidates / platformDefaultFontCandidates 里：
//   - Windows：按文件名猜 %WINDIR%\Fonts 下的字体；
//   - Linux/macOS/Android/iOS：扫描系统字体目录并按关键词匹配；
//   - 浏览器：查 Web 运行时注册进来的字体表。
func fontCandidates(faceName string, weight int32) []string {
	if p := strings.TrimSpace(os.Getenv("GOGUI_FONT_FILE")); p != "" {
		return []string{p}
	}
	if p := platformFontCandidates(faceName, weight); len(p) > 0 {
		return p
	}
	return platformDefaultFontCandidates(weight)
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && indexOf(s, sub) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	n, m := len(s), len(sub)
	if m == 0 || m > n {
		return -1
	}
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}

// mapFontFile 读取字体文件：桌面端走内存映射，浏览器端查已注册字体表。
func mapFontFile(path string) ([]byte, error) { return platformMapFontFile(path) }

// loadSystemFont 解析系统字体文件，优先返回文件名向量中的第一个可用字体。
func loadSystemFont(faceName string, weight int32) (*sfnt.Font, error) {
	paths := fontCandidates(faceName, weight)
	for _, p := range paths {
		key := p
		sysFontsMu.Lock()
		if f, ok := sysFonts[key]; ok {
			sysFontsMu.Unlock()
			if f != nil {
				return f, nil
			}
			continue
		}
		data, err := mapFontFile(p)
		var picked *sfnt.Font
		if err == nil {
			if col, cerr := opentype.ParseCollection(data); cerr == nil && col.NumFonts() > 0 {
				// 优先取族名匹配 "UI" 的那个（Microsoft YaHei UI）
				var best *sfnt.Font
				for i := 0; i < col.NumFonts(); i++ {
					if fnt, ferr := col.Font(i); ferr == nil {
						if best == nil {
							best = fnt
						}
						var buf sfnt.Buffer
						if fam, ferr := fnt.Name(&buf, sfnt.NameIDFamily); ferr == nil && containsAny(fam, "UI") {
							best = fnt
							break
						}
					}
				}
				picked = best
			} else if fnt, ferr := opentype.Parse(data); ferr == nil {
				picked = fnt
			}
		}
		sysFonts[key] = picked
		sysFontsMu.Unlock()
		if picked != nil {
			return picked, nil
		}
	}
	return nil, os.ErrNotExist
}
