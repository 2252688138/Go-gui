//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// Segoe UI Emoji 的 COLR/CPAL 彩色表解析与分层光栅化。
//
// 为什么不用 GDI：GDI（DrawText/TextOut）不支持 COLR 彩色字形，Windows 10/11 上
// 用 Segoe UI Emoji 画出来的是单色轮廓（黑/白），这就是 exe 里 Emoji 没有颜色的原因。
// 彩色渲染只有 DirectWrite/D2D 支持，或自己按 COLR 分层合成。
//
// 这里走后者：纯 Go 解析 COLR/CPAL（v0 足够，Segoe UI Emoji 就是 v0），
// 用 x/image/font/sfnt 取每层字形轮廓、vector.Rasterizer 做抗锯齿填充，
// 再按调色板颜色逐层合成 RGBA。零 cgo、零 COM，和现有 SDF/矢量光栅化同一条路径。

// colrBase COLR 的 BaseGlyphRecord：一个 Emoji 由哪几层组成。
type colrBase struct {
	first  uint16
	layers uint16
}

// colrLayer COLR 的 LayerRecord：层字形 + 调色板索引（0xFFFF 表示用前景色）。
type colrLayer struct {
	gid    uint16
	palIdx uint16
}

// colorFont 彩色 Emoji 字体（懒加载一次，全局共享）。
type colorFont struct {
	mu    sync.Mutex
	fnt   *sfnt.Font
	upem  int32
	base  map[uint16]colrBase
	layer []colrLayer
	pal   [][4]uint8 // 选定的调色板（RGBA）
	tryed bool
	fail  bool
}

var (
	colorEmojiOnce sync.Once
	colorEmojiInst *colorFont
)

// colorEmoji 取全局彩色 Emoji 光栅器（不可用时返回 nil）。
func colorEmoji() *colorFont {
	colorEmojiOnce.Do(func() {
		colorEmojiInst = loadColorEmoji()
	})
	if colorEmojiInst == nil || colorEmojiInst.fail {
		return nil
	}
	return colorEmojiInst
}

// loadColorEmoji 找到并解析系统的彩色 Emoji 字体（Segoe UI Emoji）。
func loadColorEmoji() *colorFont {
	cands := []string{}
	if wd := os.Getenv("WINDIR"); wd != "" {
		cands = append(cands, filepath.Join(wd, "Fonts", "seguiemj.ttf"))
	}
	cands = append(cands, `C:\Windows\Fonts\seguiemj.ttf`)
	var data []byte
	for _, p := range cands {
		if b, err := os.ReadFile(p); err == nil && len(b) > 1024 {
			data = b
			break
		}
	}
	if data == nil {
		return nil
	}
	fnt, err := sfnt.Parse(data)
	if err != nil {
		return nil
	}
	tables := sfntTables(data)
	colr, cpal := tables["COLR"], tables["CPAL"]
	if len(colr) < 14 || len(cpal) < 12 {
		return nil
	}
	cf := &colorFont{fnt: fnt, base: map[uint16]colrBase{}}
	cf.upem = int32(fnt.UnitsPerEm())

	// COLR v0 头部：version(2) numBaseGlyphRecords(2) baseOffset(4) layerOffset(4) numLayerRecords(2)
	nBase := int(be16(colr[2:]))
	baseOff := int(be32(colr[4:]))
	layerOff := int(be32(colr[8:]))
	nLayer := int(be16(colr[12:]))
	if baseOff+6*nBase > len(colr) || layerOff+4*nLayer > len(colr) {
		return nil
	}
	for i := 0; i < nBase; i++ {
		p := baseOff + i*6
		cf.base[be16(colr[p:])] = colrBase{first: be16(colr[p+2:]), layers: be16(colr[p+4:])}
	}
	cf.layer = make([]colrLayer, 0, nLayer)
	for i := 0; i < nLayer; i++ {
		p := layerOff + i*4
		cf.layer = append(cf.layer, colrLayer{gid: be16(colr[p:]), palIdx: be16(colr[p+2:])})
	}

	// CPAL v0 头部：version(2) numPaletteEntries(2) numPalettes(2) numColorRecords(2)
	//              colorRecordsArrayOffset(4) colorRecordIndices[numPalettes](2*n)
	entries := int(be16(cpal[2:]))
	palettes := int(be16(cpal[4:]))
	colorOff := int(be32(cpal[8:]))
	if entries <= 0 || palettes <= 0 || colorOff+4*entries*palettes > len(cpal) {
		return nil
	}
	// 选「平均亮度更高」的调色板：深色 UI 下 Emoji 更清晰
	best, bestLum := 0, -1.0
	for p := 0; p < palettes; p++ {
		start := int(be16(cpal[12+p*2:]))
		lum := 0.0
		for i := 0; i < entries; i++ {
			q := colorOff + (start+i)*4
			if q+3 >= len(cpal) {
				continue
			}
			lum += float64(cpal[q+2])*0.299 + float64(cpal[q+1])*0.587 + float64(cpal[q])*0.114
		}
		if lum > bestLum {
			best, bestLum = p, lum
		}
	}
	start := int(be16(cpal[12+best*2:]))
	cf.pal = make([][4]uint8, entries)
	for i := 0; i < entries; i++ {
		q := colorOff + (start+i)*4
		if q+3 >= len(cpal) {
			break
		}
		cf.pal[i] = [4]uint8{cpal[q+2], cpal[q+1], cpal[q], cpal[q+3]} // BGRA -> RGBA
	}
	if len(cf.base) == 0 || len(cf.pal) == 0 {
		return nil
	}
	return cf
}

// sfntTables 解析 TrueType 表目录，返回 tag -> 表数据。
func sfntTables(data []byte) map[string][]byte {
	out := map[string][]byte{}
	if len(data) < 12 {
		return out
	}
	n := int(be16(data[4:]))
	for i := 0; i < n; i++ {
		p := 12 + i*16
		if p+16 > len(data) {
			break
		}
		tag := string(data[p : p+4])
		off := int(be32(data[p+8:]))
		ln := int(be32(data[p+12:]))
		if off < 0 || ln < 0 || off+ln > len(data) {
			continue
		}
		out[tag] = data[off : off+ln]
	}
	return out
}

func be16(b []byte) uint16 { return binary.BigEndian.Uint16(b) }
func be32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }

// render 渲染一个 Emoji 字符，返回裁剪后的 RGBA 位图、前进宽度与是否成功。
func (c *colorFont) render(r rune, px int32) (*emojiBitmap, int, bool) {
	if c == nil || c.fnt == nil || px < 8 {
		return nil, 0, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	var buf sfnt.Buffer
	gid, err := c.fnt.GlyphIndex(&buf, r)
	if err != nil || gid == 0 {
		return nil, 0, false
	}
	ppem := fixed.Int26_6(px * 64)
	adv, err := c.fnt.GlyphAdvance(&buf, gid, ppem, font.HintingNone)
	if err != nil || adv <= 0 {
		adv = ppem
	}
	b, ok := c.base[uint16(gid)]
	if !ok || b.layers == 0 {
		return nil, 0, false
	}

	size := int(px)*2 + 32
	// 与 GDI 版保持同一基线：顶对齐后基线在 3/4 字号处（Top 语义为「字形顶部相对基线」）
	baseY := int(px) * 3 / 4
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	var rast vector.Rasterizer

	for i := 0; i < int(b.layers); i++ {
		li := int(b.first) + i
		if li < 0 || li >= len(c.layer) {
			break
		}
		lay := c.layer[li]
		if lay.gid == 0 {
			continue
		}
		segs, err := c.fnt.LoadGlyph(&buf, sfnt.GlyphIndex(lay.gid), ppem, nil)
		if err != nil || len(segs) == 0 {
			continue
		}
		rast.Reset(size, size)
		started := false
		for _, s := range segs {
			// sfnt 的 Y 轴向上，画布向下，这里翻转到基线坐标系
			pt := func(q int) (float32, float32) {
				x := float32(s.Args[q].X) / 64
				y := float32(baseY) - float32(s.Args[q].Y)/64
				return x, y
			}
			switch s.Op {
			case sfnt.SegmentOpMoveTo:
				if started {
					rast.ClosePath()
				}
				x, y := pt(0)
				rast.MoveTo(x, y)
				started = true
			case sfnt.SegmentOpLineTo:
				x, y := pt(0)
				rast.LineTo(x, y)
			case sfnt.SegmentOpQuadTo:
				cx, cy := pt(0)
				x, y := pt(1)
				rast.QuadTo(cx, cy, x, y)
			case sfnt.SegmentOpCubeTo:
				cx0, cy0 := pt(0)
				cx1, cy1 := pt(1)
				x, y := pt(2)
				rast.CubeTo(cx0, cy0, cx1, cy1, x, y)
			}
		}
		if started {
			rast.ClosePath()
		}
		col := c.layerColor(lay.palIdx)
		rast.Draw(img, img.Bounds(), image.NewUniform(col), image.Point{})
	}

	// 裁剪非透明区域，并换算成相对基线的偏移（与 GDI 版语义一致）
	minX, minY, maxX, maxY := size, size, -1, -1
	for y := 0; y < size; y++ {
		row := y * img.Stride
		for x := 0; x < size; x++ {
			if img.Pix[row+x*4+3] == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return nil, int(adv >> 6), false
	}
	w, h := maxX-minX+1, maxY-minY+1
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		src := (minY+y)*img.Stride + minX*4
		copy(out[y*w*4:(y+1)*w*4], img.Pix[src:src+w*4])
	}
	return &emojiBitmap{W: w, H: h, Pix: out, Left: minX, Top: minY - baseY}, int(adv >> 6), true
}

// layerColor 层颜色：0xFFFF 表示使用前景色（深色 UI 用白色）。
func (c *colorFont) layerColor(idx uint16) color.RGBA {
	if idx == 0xFFFF || int(idx) >= len(c.pal) {
		return color.RGBA{255, 255, 255, 255}
	}
	p := c.pal[idx]
	a := p[3]
	if a == 0 {
		a = 255
	}
	return color.RGBA{p[0], p[1], p[2], a}
}
