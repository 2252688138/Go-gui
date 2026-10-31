//go:build !windows && !js && !wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

// 类 Unix 端（Linux / macOS / Android / iOS）的 Emoji。
//
// 一期策略：不做彩色位图光栅。彩色 Emoji 在 Windows 走 GDI（系统帮你合成彩色、
// 带位图表的字形），浏览器端走 canvas fillText；类 Unix 端要实现同样效果需要自己
// 解码 CBDT / sbix 彩色位表，属于后续增强项。
//
// 这里的 render 一律返回 false，font.go 会回退到字体自身的**单色字形**：
// 结果是 Emoji 能显示（可能为黑白轮廓），不会出现豆腐块或 panic，
// 与「不支持的能力空实现」的跨平台约定一致。

// emojiBitmap 一张彩色字形位图（RGBA，首行为顶部）。字段与 Windows/Web 端保持一致。
type emojiBitmap struct {
	W, H int
	Pix  []byte
	Left int
	Top  int
}

// gdiFont 类 Unix 端的占位光栅器（不持有任何原生资源）。
type gdiFont struct {
	name string
}

func newGdiFont(name string) *gdiFont { return &gdiFont{name: name} }

// ensureFont 空实现：无需准备 DC/位图/字体。
func (g *gdiFont) ensureFont(px int32) {}

// render 始终返回 false：交给调用方回退到单色字形。
func (g *gdiFont) render(s string, px int32) (*emojiBitmap, int, bool) { return nil, 0, false }

func (g *gdiFont) close() {}

// max3 三值取最大（与 Windows/Web 端同名辅助函数保持一致）。
func max3(a, b, c uint8) uint8 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}
