//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"fmt"
	"syscall/js"
)

// 浏览器端的彩色 Emoji 光栅：用 canvas 2D 的 fillText 让系统 Emoji 字体（Segoe UI Emoji /
// Apple Color Emoji / Noto Color Emoji）直接画出彩色字形，再取 ImageData 裁剪成位图。
// 这样与桌面端走 GDI 的效果对齐：同一套 Font.rasterizeEmoji 逻辑，两端都能出彩色 Emoji。

type emojiBitmap struct {
	W, H int
	Pix  []byte
	Left int
	Top  int
}

type gdiFont struct {
	name string
	px   int32
	cv   js.Value
	ctx  js.Value
}

func newGdiFont(name string) *gdiFont { return &gdiFont{name: name} }

// ensureFont 按像素字号准备离屏 canvas 与字体串。
func (g *gdiFont) ensureFont(px int32) {
	if px < 8 {
		px = 8
	}
	if px == g.px && !g.ctx.IsUndefined() && !g.ctx.IsNull() {
		return
	}
	g.px = px
	if g.cv.IsUndefined() || g.cv.IsNull() {
		g.cv = js.Global().Get("document").Call("createElement", "canvas")
		g.cv.Set("width", int(px*2+16))
		g.cv.Set("height", int(px*2+16))
		g.ctx = g.cv.Call("getContext", "2d", map[string]any{"willReadFrequently": true})
		if g.ctx.IsNull() || g.ctx.IsUndefined() {
			return
		}
	}
	family := g.name
	if family == "" {
		family = `"Segoe UI Emoji","Apple Color Emoji","Noto Color Emoji",sans-serif`
	}
	g.ctx.Set("font", fmt.Sprintf("%dpx %s", px, family))
	g.ctx.Set("textBaseline", "alphabetic")
	g.ctx.Set("textAlign", "left")
}

// render 渲染一个字符，返回裁剪后的 RGBA 位图、前进宽度与是否成功。
func (g *gdiFont) render(s string, px int32) (*emojiBitmap, int, bool) {
	g.ensureFont(px)
	if g.ctx.IsUndefined() || g.ctx.IsNull() {
		return nil, 0, false
	}
	size := int(g.px*2 + 16)
	adv := int(g.ctx.Call("measureText", s).Get("width").Float())
	if adv <= 0 {
		adv = int(g.px)
	}

	g.ctx.Call("clearRect", 0, 0, size, size)
	baseY := int(g.px) * 5 / 4
	g.ctx.Call("fillText", s, 0, baseY)
	g.ctx.Set("globalCompositeOperation", "source-over")

	data := g.ctx.Call("getImageData", 0, 0, size, size).Get("data")
	n := data.Length()
	u8 := js.Global().Get("Uint8Array").New(data.Get("buffer"), data.Get("byteOffset"), n)
	buf := make([]byte, n)
	js.CopyBytesToGo(buf, u8)

	minX, minY, maxX, maxY := size, size, -1, -1
	for y := 0; y < size; y++ {
		row := y * size * 4
		for x := 0; x < size; x++ {
			if buf[row+x*4+3] == 0 {
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
		return nil, adv, false
	}

	w, h := maxX-minX+1, maxY-minY+1
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		src := (minY+y)*size*4 + minX*4
		copy(out[y*w*4:(y+1)*w*4], buf[src:src+w*4])
	}
	// Top 与桌面端一致：字形顶部相对基线的偏移
	return &emojiBitmap{W: w, H: h, Pix: out, Left: minX, Top: minY - baseY}, adv, true
}

func (g *gdiFont) close() {}

func max3(a, b, c uint8) uint8 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
