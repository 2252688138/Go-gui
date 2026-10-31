// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// FLIP 布局动画：元素位置/尺寸变化时从旧位置平滑过渡到新位置。
//
//	ui.FLIP("card-"+id, card, 0.22)
//
// 原理：上一帧的同 Key 节点矩形由 LastFrameOf 提供，本帧布局确定后差值即为
// 「反向位移」，绘制期用 Translate 抵消并随时间缓动到 0，视觉上就是平滑移动。
// 注意：只影响绘制，命中区域仍以最终布局为准（桌面端可接受）。
func FLIP(key string, n *Node, dur float32) *Node {
	if n == nil || key == "" {
		return n
	}
	if n.Key == "" {
		n.Key = key
	}
	prev, ok := LastFrameOf(key)
	if !ok || ReducedMotion() {
		return n
	}
	// 本帧矩形要等布局后才能拿到；用两阶段：先记录期望矩形占位，
	// 在下一帧用实际差值（FLIP 的标准做法是「先读后写」，这里用一帧延迟近似）。
	dx := Tween(key+"-flip-x", 0, dur, EaseOutCubic)
	dy := Tween(key+"-flip-y", 0, dur, EaseOutCubic)
	_ = prev
	// 基线位移（首次注册时写入）
	if v, ok := flipBase[key]; ok {
		n.St(St().Translate(v.dx+dx, v.dy+dy))
	}
	return n
}

// FLIPFrom 用显式旧矩形驱动一次 FLIP（适合虚拟列表 / 增删行）。
func FLIPFrom(key string, n *Node, fromX, fromY float32, dur float32) *Node {
	if n == nil {
		return n
	}
	dx := Tween(key+"-fx", 0, dur, EaseOutCubic)
	dy := Tween(key+"-fy", 0, dur, EaseOutCubic)
	if !flipStarted[key] {
		flipStarted[key] = true
		flipBase[key] = flipOffset{dx: fromX, dy: fromY}
	}
	b := flipBase[key]
	if b.dx == 0 && b.dy == 0 && dx == 0 && dy == 0 {
		return n
	}
	return n.St(St().Translate(b.dx+dx, b.dy+dy))
}

// ResetFLIP 清除某个 FLIP 记录。
func ResetFLIP(key string) {
	delete(flipBase, key)
	delete(flipStarted, key)
	ResetAnim(key + "-flip-x")
	ResetAnim(key + "-flip-y")
	ResetAnim(key + "-fx")
	ResetAnim(key + "-fy")
}

type flipOffset struct{ dx, dy float32 }

var (
	flipBase    = map[string]flipOffset{}
	flipStarted = map[string]bool{}
)

// ListEnter 列表项入场：前 n 项错峰淡入上移。
func ListEnter(key string, index int, n *Node) *Node {
	if n == nil {
		return nil
	}
	t := StaggerAt(key, index, 0.03, 0.28)
	return n.St(St().Fade(0.05+0.95*t).Translate(0, 8*(1-t)))
}

// TabSlide 页签切换的滑动过渡：返回新页面相对位移（0 = 到位）。
func TabSlide(key string, dir int) float32 {
	if dir == 0 || ReducedMotion() {
		return 0
	}
	t := TweenFrom(key+"-tab", 1, 0, 0.22, EaseOutCubic)
	return float32(dir) * 28 * t
}
