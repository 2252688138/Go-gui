// Package geom 提供与渲染无关的几何基础类型（float32，直接喂 GPU）。
package geom

import "math"

type Point struct {
	X, Y float32
}

func Pt(x, y float32) Point { return Point{x, y} }

func (p Point) Add(o Point) Point { return Point{p.X + o.X, p.Y + o.Y} }
func (p Point) Sub(o Point) Point { return Point{p.X - o.X, p.Y - o.Y} }

func (p Point) Round() Point {
	return Point{float32(math.Round(float64(p.X))), float32(math.Round(float64(p.Y)))}
}

type Size struct {
	W, H float32
}

func Sz(w, h float32) Size { return Size{w, h} }

func (s Size) IsEmpty() bool { return s.W <= 0 || s.H <= 0 }

// Rect 左上角 + 尺寸，右/下边界不包含在内部。
type Rect struct {
	X, Y, W, H float32
}

func R(x, y, w, h float32) Rect { return Rect{x, y, w, h} }

func (r Rect) Right() float32  { return r.X + r.W }
func (r Rect) Bottom() float32 { return r.Y + r.H }
func (r Rect) Size() Size      { return Size{r.W, r.H} }
func (r Rect) Min() Point      { return Point{r.X, r.Y} }
func (r Rect) Center() Point   { return Point{r.X + r.W/2, r.Y + r.H/2} }
func (r Rect) Max() Point      { return Point{r.X + r.W, r.Y + r.H} }
func (r Rect) IsEmpty() bool   { return r.W <= 0 || r.H <= 0 }

func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.Right() && p.Y >= r.Y && p.Y < r.Bottom()
}

func (r Rect) Inset(i Insets) Rect {
	return Rect{r.X + i.Left, r.Y + i.Top, r.W - i.Left - i.Right, r.H - i.Top - i.Bottom}
}

func (r Rect) InsetAll(v float32) Rect { return r.Inset(Insets{v, v, v, v}) }

func (r Rect) Offset(dx, dy float32) Rect { return Rect{r.X + dx, r.Y + dy, r.W, r.H} }

func (r Rect) Intersect(o Rect) Rect {
	x := max32(r.X, o.X)
	y := max32(r.Y, o.Y)
	right := min32(r.Right(), o.Right())
	bottom := min32(r.Bottom(), o.Bottom())
	if right <= x || bottom <= y {
		return Rect{}
	}
	return Rect{x, y, right - x, bottom - y}
}

func (r Rect) Union(o Rect) Rect {
	x := min32(r.X, o.X)
	y := min32(r.Y, o.Y)
	right := max32(r.Right(), o.Right())
	bottom := max32(r.Bottom(), o.Bottom())
	return Rect{x, y, right - x, bottom - y}
}

// Insets 四边内间距。
type Insets struct {
	Top, Right, Bottom, Left float32
}

func All(v float32) Insets           { return Insets{v, v, v, v} }
func Sym(h, v float32) Insets        { return Insets{v, h, v, h} }
func Only(t, r, b, l float32) Insets { return Insets{t, r, b, l} }
func (i Insets) Horizontal() float32 { return i.Left + i.Right }
func (i Insets) Vertical() float32   { return i.Top + i.Bottom }

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
