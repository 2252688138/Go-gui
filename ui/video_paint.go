//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// videoTexture 一个播放器的帧纹理缓存（帧号或尺寸变化时更新纹理）。
type videoTexture struct {
	tex  *gpu.Texture
	sig  uint64
	w, h int
	// used 最近一次绘制的帧序号（LRU 回收用）
	used int
}

// maxVideoTextures 同时保留的视频帧纹理数量上限。
// 每张 1080p 纹理约 8MB（显存 + 驱动侧，上传时还有同尺寸拷贝），旧实现按
// *Player 建缓存却从不回收：只要业务每帧新建一个 Player（示例页的三种填充
// 方式演示就是这样），纹理就会无上限增长，几分钟就能吃掉 1GB 以上内存。
// 现在按 LRU 只保留最近绘制的若干张，其余连纹理一起删除。
const maxVideoTextures = 8

// videoTextureFor 取播放器当前帧的纹理（首次上传，后续按帧增量更新）。
func (a *App) videoTextureFor(p *Player, f *VideoFrame) *gpu.Texture {
	if a == nil || a.r == nil || p == nil || !f.Valid() {
		return nil
	}
	if a.videos == nil {
		a.videos = map[*Player]*videoTexture{}
	}
	a.videoSeq++
	e := a.videos[p]
	if e == nil || e.tex == nil || e.w != f.W || e.h != f.H {
		if e != nil && e.tex != nil {
			a.r.DeleteTexture(e.tex)
		}
		tex := a.r.NewTextureRGBA(f.W, f.H, f.Pix)
		a.videos[p] = &videoTexture{tex: tex, sig: p.frameSig, w: f.W, h: f.H, used: a.videoSeq}
		a.trimVideoTextures()
		return tex
	}
	if e.sig != p.frameSig {
		a.r.UpdateTextureRGBA(e.tex, 0, 0, f.W, f.H, f.Pix)
		e.sig = p.frameSig
	}
	e.used = a.videoSeq
	return e.tex
}

// trimVideoTextures 回收最久未绘制（used 最小）的视频纹理，把缓存压回上限内。
// 当前帧刚取过纹理的播放器 used 最大，不会被选为回收对象。
func (a *App) trimVideoTextures() {
	for len(a.videos) > maxVideoTextures {
		var oldP *Player
		oldest := 0
		for pl, en := range a.videos {
			if oldP == nil || en.used < oldest {
				oldP, oldest = pl, en.used
			}
		}
		if oldP == nil {
			return
		}
		if en := a.videos[oldP]; en != nil && en.tex != nil {
			a.r.DeleteTexture(en.tex)
		}
		delete(a.videos, oldP)
	}
}

// paintVideo 绘制视频画面：按 Player.Fit 等比填充，返回实际绘制区域。
func (a *App) paintVideo(n *Node, r geom.Rect, st *Style) {
	p := n.Video
	if p == nil {
		a.paintVideoPlaceholder(n, r, st)
		return
	}
	// 记录画面矩形，供控制条自动隐藏与点击判定
	p.rect = rectBox{X: r.X, Y: r.Y, W: r.W, H: r.H}

	f := p.Frame()
	tex := a.videoTextureFor(p, f)
	if tex == nil || f == nil {
		a.paintVideoPlaceholder(n, r, st)
		return
	}
	// 旋转 90°/270° 时画面宽高互换（Fit 按交换后的比例计算）
	fw, fh := float32(f.W), float32(f.H)
	q := p.Rotation & 3
	if q == 1 || q == 3 {
		fw, fh = fh, fw
	}
	a.r.DrawImage(gpu.Image{
		Rect:   fitVideoRect(r, fw, fh, p.Fit),
		Tex:    tex,
		Tint:   gpu.Paint{R: 1, G: 1, B: 1, A: 1},
		Radius: st.Radius,
		Rot:    q,
	})
}

// paintVideoPlaceholder 无画面/取帧失败时的占位（暗底 + 提示文字）。
func (a *App) paintVideoPlaceholder(n *Node, r geom.Rect, st *Style) {
	a.r.Rect(gpu.Shape{
		Rect:   r,
		Radius: st.Radius,
		Fill:   paintOf(ColText, 0.06),
	})
	label := "无画面"
	if p := n.Video; p != nil {
		switch {
		case p.Err() != nil:
			label = "取帧失败"
		case p.Source != nil:
			label = "解码中…"
		}
	}
	tmp := &Node{
		Kind:  KText,
		Text:  label,
		Style: Style{Opacity: 1, FontSize: 12, FontWeight: 500, Color: C("#6b7694"), CenterY: true, TextAlign: AlignCenter},
	}
	a.paintText(tmp, r, &tmp.Style)
}

// fitVideoRect 按填充方式计算画面实际矩形（Contain 留黑边 / Cover 溢出裁切 / Fill 拉伸）。
func fitVideoRect(r geom.Rect, sw, sh float32, fit VideoFit) geom.Rect {
	if sw <= 0 || sh <= 0 || r.W <= 0 || r.H <= 0 {
		return r
	}
	if fit == FitFill {
		return r
	}
	scale := r.W / sw
	if fit == FitCover {
		if v := r.H / sh; v > scale {
			scale = v
		}
	} else if v := r.H / sh; v < scale {
		scale = v
	}
	dw, dh := sw*scale, sh*scale
	return geom.Rect{X: r.X + (r.W-dw)/2, Y: r.Y + (r.H-dh)/2, W: dw, H: dh}
}
