package gpu

import "gogui/internal/geom"

// PushClip 开启一个裁剪矩形（与已有裁剪取交集）。
func (r *Renderer) PushClip(c geom.Rect) {
	// 与已有裁剪取交集：嵌套裁剪（Scroll 里的 Clip 卡片等）不能把裁剪区「放大」，
	// 否则内层比外层大时内容会溢出到父级（典型现象：滚动内容盖住标题栏/菜单栏）。
	if n := len(r.clipSaved); n > 0 {
		c = c.Intersect(r.clipSaved[n-1])
	}
	r.clipSaved = append(r.clipSaved, c)
	r.setClip(c) // setClip 内部会先 flush
}

// PopClip 关闭上一层裁剪。
func (r *Renderer) PopClip() {
	if n := len(r.clipSaved); n > 0 {
		r.clipSaved = r.clipSaved[:n-1]
	}
	if n := len(r.clipSaved); n > 0 {
		r.setClip(r.clipSaved[n-1])
	} else {
		r.clearClip()
	}
}
