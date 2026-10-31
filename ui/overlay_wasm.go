//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"syscall/js"

	"gogui/internal/geom"
)

// 浏览器端「DOM 叠加层」：把 <video> / <iframe> 这类原生元素按 UI 逻辑矩形
// 盖在 WebGL 画布上。
//
// 为什么需要它：整页界面由 canvas 绘制，浏览器不允许把 <video>/<iframe> 直接
// 画进 WebGL（除非逐帧 drawImage 上传纹理，代价高且 iframe 无法这么做）。因此
// 用一层 position:fixed 的透明容器承载这些元素，每帧（Swap 时）按绘制阶段记录
// 的逻辑矩形重新定位，并对滚动/裁剪区做 clip-path 裁切。
//
// 坐标系：UI 逻辑坐标与 CSS 像素 1:1（backend_wasm.go 中 Scale() = dpr，
// dpr 只放大画布像素，不改 CSS 尺寸），所以矩形可直接当 CSS 值使用，
// 只需扣掉画布本身在页面中的偏移。

var (
	overlayRootEl js.Value
	overlayEls    = map[string]js.Value{}
	overlayUsed   = map[string]bool{}
	overlayPlaces []overlayPlacement
	overlaySeq    int
)

// overlayPlacement 一帧内某个叠加元素的期望位置与裁剪区。
type overlayPlacement struct {
	ID    string
	Rect  geom.Rect
	Clip  geom.Rect // 零尺寸表示不裁剪
	Apply func(el js.Value)
}

// overlayRoot 取/建叠加容器（覆盖整个视口，指针事件默认穿透）。
func overlayRoot() js.Value {
	if !overlayRootEl.IsUndefined() && !overlayRootEl.IsNull() {
		return overlayRootEl
	}
	doc := js.Global().Get("document")
	root := doc.Call("createElement", "div")
	root.Set("id", "gogui-overlay")
	st := root.Get("style")
	st.Set("position", "fixed")
	st.Set("left", "0")
	st.Set("top", "0")
	st.Set("width", "100%")
	st.Set("height", "100%")
	st.Set("pointerEvents", "none")
	st.Set("overflow", "hidden")
	st.Set("zIndex", "5")
	root.Set("style", st)
	doc.Get("body").Call("appendChild", root)
	overlayRootEl = root
	return root
}

// overlayID 生成一个稳定的叠加元素 id。
func overlayID(prefix string) string {
	overlaySeq++
	return fmt.Sprintf("%s-%d", prefix, overlaySeq)
}

// overlayEl 取/建叠加元素（同 id 复用，避免每帧新建 DOM）。
func overlayEl(id, tag string) js.Value {
	if el, ok := overlayEls[id]; ok && !el.IsUndefined() && !el.IsNull() {
		return el
	}
	doc := js.Global().Get("document")
	el := doc.Call("createElement", tag)
	st := el.Get("style")
	st.Set("position", "absolute")
	st.Set("left", "0")
	st.Set("top", "0")
	st.Set("width", "0")
	st.Set("height", "0")
	st.Set("display", "none")
	st.Set("margin", "0")
	st.Set("border", "0")
	st.Set("pointerEvents", "none")
	el.Set("style", st)
	overlayRoot().Call("appendChild", el)
	overlayEls[id] = el
	return el
}

// dropOverlayEl 删除叠加元素（媒体/浏览器关闭时调用）。
func dropOverlayEl(id string) {
	el, ok := overlayEls[id]
	if !ok {
		return
	}
	if !el.IsUndefined() && !el.IsNull() {
		if p := el.Get("parentNode"); !p.IsNull() && !p.IsUndefined() {
			p.Call("removeChild", el)
		}
	}
	delete(overlayEls, id)
	delete(overlayUsed, id)
}

// placeOverlay 记录本帧的位置（绘制阶段调用，Swap 时统一写入 DOM）。
func placeOverlay(id string, r geom.Rect, clip geom.Rect, apply func(el js.Value)) {
	overlayUsed[id] = true
	overlayPlaces = append(overlayPlaces, overlayPlacement{ID: id, Rect: r, Clip: clip, Apply: apply})
}

// currentClip 取当前绘制裁剪栈与给定矩形的交集（无裁剪时返回零矩形）。
func (a *App) currentClip(r geom.Rect) geom.Rect {
	n := a.clipN
	if n > len(a.clipStack) {
		n = len(a.clipStack)
	}
	if n <= 0 {
		return geom.Rect{}
	}
	out := r
	for i := 0; i < n; i++ {
		out = intersectRect(out, a.clipStack[i])
	}
	if out.W <= 0 || out.H <= 0 {
		return geom.Rect{W: 0, H: 0}
	}
	return out
}

// syncOverlays 把本帧记录的位置写入 DOM（由 platformWin.Swap 每帧调用一次）。
func syncOverlays() {
	if len(overlayEls) == 0 && len(overlayPlaces) == 0 {
		return
	}
	// 画布在页面中的偏移（通常为 0，但页面可能给 canvas 留边距）
	ox, oy := float32(0), float32(0)
	if winOnce != nil && !winOnce.canvas.IsUndefined() && !winOnce.canvas.IsNull() {
		rc := winOnce.canvas.Call("getBoundingClientRect")
		ox = float32(rc.Get("left").Float())
		oy = float32(rc.Get("top").Float())
	}
	for _, pl := range overlayPlaces {
		el := overlayEls[pl.ID]
		if el.IsUndefined() || el.IsNull() {
			continue
		}
		st := el.Get("style")
		st.Set("display", "block")
		st.Set("left", fmt.Sprintf("%.2fpx", pl.Rect.X+ox))
		st.Set("top", fmt.Sprintf("%.2fpx", pl.Rect.Y+oy))
		st.Set("width", fmt.Sprintf("%.2fpx", pl.Rect.W))
		st.Set("height", fmt.Sprintf("%.2fpx", pl.Rect.H))
		if pl.Clip.W > 0 && pl.Clip.H > 0 {
			x0 := clampf(pl.Clip.X-pl.Rect.X, 0, pl.Rect.W)
			y0 := clampf(pl.Clip.Y-pl.Rect.Y, 0, pl.Rect.H)
			x1 := clampf(pl.Clip.X+pl.Clip.W-pl.Rect.X, 0, pl.Rect.W)
			y1 := clampf(pl.Clip.Y+pl.Clip.H-pl.Rect.Y, 0, pl.Rect.H)
			st.Set("clipPath", fmt.Sprintf("inset(%.2fpx %.2fpx %.2fpx %.2fpx)",
				y0, pl.Rect.W-x1, pl.Rect.H-y1, x0))
		} else {
			st.Set("clipPath", "none")
		}
		if pl.Apply != nil {
			pl.Apply(el)
		}
	}
	// 本帧没有出现的元素先隐藏（节点可能已切走或滚动到视口外）
	for id, el := range overlayEls {
		if overlayUsed[id] {
			continue
		}
		if el.IsUndefined() || el.IsNull() {
			continue
		}
		el.Get("style").Set("display", "none")
	}
	overlayUsed = map[string]bool{}
	overlayPlaces = overlayPlaces[:0]
}
