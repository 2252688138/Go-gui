// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package gpu 提供与平台/UI 解耦的 2D GPU 渲染后端接口及 OpenGL 3.3 实现。
package gpu

import (
	"gogui/internal/geom"
	"gogui/internal/text"
)

// Paint 颜色（RGBA，0~1）。
type Paint struct {
	R, G, B, A float32
}

func RGBA(r, g, b, a uint8) Paint {
	return Paint{float32(r) / 255, float32(g) / 255, float32(b) / 255, float32(a) / 255}
}

func Hex(v uint32) Paint {
	if v <= 0xFFFFFF {
		return RGBA(uint8(v>>16), uint8(v>>8), uint8(v), 255)
	}
	return RGBA(uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v))
}

func (p Paint) WithAlpha(a float32) Paint { p.A = a; return p }

func (p Paint) MulAlpha(a float32) Paint { p.A *= a; return p }

// Gradient 线性渐变（两色）。
type Gradient struct {
	On       bool
	Angle    float32 // CSS 角度：0 = 向上，90 = 向右，180 = 向下，270 = 向左（From 在起始侧）
	From, To Paint
}

// Shape 一个矩形/圆角矩形绘制指令。
type Shape struct {
	Rect        geom.Rect
	Radius      [4]float32 // 左上、右上、右下、左下
	Fill        Paint
	Grad        Gradient
	Stroke      float32
	StrokeColor Paint
	// Shadow
	ShadowBlur    float32
	ShadowSpread  float32
	ShadowOffsetX float32
	ShadowOffsetY float32
	ShadowColor   Paint
}

// TextRun 一段文字的绘制指令。
type TextRun struct {
	X, Y  float32 // 逻辑像素，Y 为行顶部
	Text  string
	Font  *text.Font
	Color Paint
}

// Image 图片绘制指令。
type Image struct {
	Rect   geom.Rect
	Tex    *Texture
	Tint   Paint
	Radius [4]float32
	// Rot 顺时针旋转的 90° 份数（0~3）：仅旋转采样坐标，Rect 保持不变
	Rot int
}

// Texture 一张 GPU 纹理。
type Texture struct {
	ID   uint32
	W, H int
	R8   bool
}

// Backend 渲染后端接口（可插拔：OpenGL / Skia / D3D11 / Metal）。
type Backend interface {
	Init() error
	Resize(pw, ph int32, scale float32)
	Begin(fill Paint)
	Draw(instrs []Instr)
	End()
	NewTextureR8(w, h int, data []byte) *Texture
	UpdateTextureR8(t *Texture, x, y, w, h int, data []byte)
	NewTextureRGBA(w, h int, data []byte) *Texture
	Destroy()
}

// InstrKind 绘制指令类型。
type InstrKind uint8

const (
	InstrShape InstrKind = iota
	InstrText
	InstrImage
	InstrClipPush
	InstrClipPop
	InstrLayerPush
	InstrLayerPop
	InstrBlurBackdrop
)

// Instr 一条绘制指令（渲染器按顺序消费）。
type Instr struct {
	Kind  InstrKind
	Shape Shape
	Text  TextRun
	Image Image
	Clip  geom.Rect
	Blur  float32
}
