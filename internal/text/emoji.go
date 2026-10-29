//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"gogui/internal/win32"
)

// emojiBitmap 一张彩色字形位图（RGBA，首行为顶部）。
type emojiBitmap struct {
	W, H int
	Pix  []byte
	Left int
	Top  int
}

// gdiFont 用 GDI 把 Emoji 渲染到 32bpp DIB（Windows 会输出彩色字形）。
type gdiFont struct {
	mu    sync.Mutex
	name  string
	hdc   uintptr
	bmp   uintptr
	bits  unsafe.Pointer
	old   uintptr
	hfont uintptr
	px    int32
	size  int32
}

func newGdiFont(name string) *gdiFont { return &gdiFont{name: name} }

// ensureFont 按像素字号准备 DC/位图/字体。
func (g *gdiFont) ensureFont(px int32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if px < 8 {
		px = 8
	}
	if g.hdc == 0 {
		g.hdc = win32.CreateCompatibleDC(0)
		if g.hdc == 0 {
			return
		}
	}
	size := px*2 + 32
	if size > 512 {
		size = 512
	}
	if g.bmp == 0 || size != g.size {
		if g.bmp != 0 {
			if g.old != 0 {
				win32.SelectObject(g.hdc, g.old)
			}
			win32.DeleteObject(g.bmp)
			g.bmp = 0
		}
		bi := win32.BITMAPINFO{}
		bi.Header.BiSize = uint32(unsafe.Sizeof(bi.Header))
		bi.Header.BiWidth = size
		bi.Header.BiHeight = -size // 负数 = top-down，避免二次翻转
		bi.Header.BiPlanes = 1
		bi.Header.BiBitCount = 32
		bi.Header.BiCompression = win32.BI_RGB
		g.bmp = win32.CreateDIBSection(g.hdc, &bi, win32.DIB_RGB_COLORS, &g.bits, 0, 0)
		if g.bmp == 0 {
			return
		}
		g.old = win32.SelectObject(g.hdc, g.bmp)
		g.size = size
	}
	if g.hfont != 0 && g.px == px {
		return
	}
	if g.hfont != 0 {
		win32.DeleteObject(g.hfont)
		g.hfont = 0
	}
	face, _ := syscall.UTF16PtrFromString("Segoe UI Emoji")
	g.hfont = win32.CreateFontW(-px, 0, 0, 0, win32.FW_NORMAL, 0, 0, 0,
		win32.DEFAULT_CHARSET, 0, 0, win32.ANTIALIASED_QUALITY, win32.DEFAULT_PITCH, face)
	g.px = px
}

// render 渲染一个字符，返回裁剪后的 RGBA 位图、前进宽度与是否成功。
func (g *gdiFont) render(s string, px int32) (*emojiBitmap, int, bool) {
	g.ensureFont(px)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.hdc == 0 || g.bmp == 0 || g.hfont == 0 || g.bits == nil {
		return nil, 0, false
	}
	u, err := syscall.UTF16FromString(s)
	if err != nil || len(u) <= 1 {
		return nil, 0, false
	}
	n := int32(len(u) - 1)
	win32.SelectObject(g.hdc, g.hfont)
	win32.SetBkMode(g.hdc, win32.TRANSPARENT)

	var sz win32.SIZE
	win32.GetTextExtentPoint32W(g.hdc, &u[0], n, &sz)
	adv := int(sz.Cx)
	if adv <= 0 {
		adv = int(px)
	}

	buf := unsafe.Slice((*byte)(g.bits), int(g.size)*int(g.size)*4)
	rect := win32.RECT{Left: 0, Top: 0, Right: g.size, Bottom: g.size}
	const flags = win32.DT_LEFT | win32.DT_TOP | win32.DT_SINGLELINE | win32.DT_NOPREFIX | win32.DT_NOCLIP

	// pass1：彩色字形
	for i := range buf {
		buf[i] = 0
	}
	win32.SetTextColor(g.hdc, 0x00FFFFFF)
	win32.DrawTextW(g.hdc, &u[0], n, &rect, flags)
	win32.GdiFlush()
	colorPix := make([]byte, len(buf))
	copy(colorPix, buf)

	// pass2：白色蒙版（保留深色 emoji 的轮廓 alpha）
	for i := range buf {
		buf[i] = 0
	}
	win32.DrawTextW(g.hdc, &u[0], n, &rect, flags)
	win32.GdiFlush()
	maskPix := buf

	size := int(g.size)
	minX, minY, maxX, maxY := size, size, -1, -1
	for y := 0; y < size; y++ {
		row := y * size * 4
		for x := 0; x < size; x++ {
			i := row + x*4
			if colorPix[i]|colorPix[i+1]|colorPix[i+2]|maskPix[i]|maskPix[i+1]|maskPix[i+2] == 0 {
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
	w := maxX - minX + 1
	h := maxY - minY + 1
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		srcRow := (minY + y) * size * 4
		dstRow := y * w * 4
		for x := 0; x < w; x++ {
			si := srcRow + (minX+x)*4
			di := dstRow + x*4
			b, gg, r := colorPix[si], colorPix[si+1], colorPix[si+2]
			mb, mg, mr := maskPix[si], maskPix[si+1], maskPix[si+2]
			cmax := max3(r, gg, b)
			mmax := max3(mr, mg, mb)
			a := cmax
			if mmax > a {
				a = mmax
			}
			if cmax <= 16 {
				// 纯深色：回退成蒙版色（白色），避免不可见
				r, gg, b = mmax, mmax, mmax
			}
			out[di], out[di+1], out[di+2], out[di+3] = r, gg, b, a
		}
	}
	return &emojiBitmap{W: w, H: h, Pix: out, Left: minX, Top: minY - int(px)*3/4}, adv, true
}

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

// close 释放 GDI 资源。
func (g *gdiFont) close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.hfont != 0 {
		win32.DeleteObject(g.hfont)
		g.hfont = 0
	}
	if g.bmp != 0 {
		if g.old != 0 && g.hdc != 0 {
			win32.SelectObject(g.hdc, g.old)
		}
		win32.DeleteObject(g.bmp)
		g.bmp = 0
	}
	if g.hdc != 0 {
		win32.DeleteDC(g.hdc)
		g.hdc = 0
	}
}

var _ = strings.TrimSpace
