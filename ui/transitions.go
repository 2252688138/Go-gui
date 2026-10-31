// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"math"
	"strconv"
	"strings"
)

// 过渡与动画编排工具。
//
// 现有 anim.go 提供 Tween / Spring / Keyframes；本文件补上「浮层进出场」需要的
// KeepAlive（关闭时先播完再移除）、统一的 Scrim / Pop 造型、序列 / 并发 / 错峰编排、
// 数字滚动，以及 reduced-motion 支持。

// keepState 单个浮层的进出场进度。
type keepState struct {
	open bool
	t    float32
	last float32
}

var (
	keepStates     = map[string]*keepState{}
	staggerStates  = map[string]float32{}
	animStartTimes = map[string]float32{}
)

// KeepAlive 浮层进出场状态：
//
//	visible, t := ui.KeepAlive("modal-x", open, 0.18)
//	if !visible { return nil }   // 关闭动画播完，可以不再渲染
//	opacity, dy := ui.PopOffset(t) // 用于 Style.Fade / Translate
//
// 关闭时 t 从 1 平滑降到 0，期间仍需渲染；t 到 0 后返回 visible=false。
func KeepAlive(key string, open bool, dur float32) (bool, float32) {
	if dur <= 0 || ReducedMotion() {
		if open {
			return true, 1
		}
		delete(keepStates, key)
		return false, 0
	}
	now := AnimTime()
	st := keepStates[key]
	if st == nil {
		st = &keepState{open: open, t: 0, last: now}
		if open {
			st.t = 0
		}
		keepStates[key] = st
	}
	if st.open != open {
		st.open = open
	}
	dt := now - st.last
	if dt < 0 {
		dt = 0
	}
	if dt > 0.1 {
		dt = 0.1
	}
	st.last = now
	target := float32(0)
	if open {
		target = 1
	}
	step := dt / dur
	if st.t < target {
		st.t += step
		if st.t > target {
			st.t = target
		}
	} else if st.t > target {
		st.t -= step
		if st.t < target {
			st.t = target
		}
	}
	if st.t > 0.0001 || open {
		markAnim()
	}
	if !open && st.t <= 0.0001 {
		delete(keepStates, key)
		return false, 0
	}
	return true, EaseOutCubic(st.t)
}

// ResetKeepAlive 清除某个浮层的过渡状态（强制下次重新入场）。
func ResetKeepAlive(key string) { delete(keepStates, key) }

// PopOffset 浮层统一造型：返回不透明度与纵向位移（0.96→1 缩放用 ScaleOf）。
func PopOffset(t float32) (opacity, dy float32) {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return 0.55 + 0.45*t, 10 * (1 - t)
}

// ScaleOf 浮层缩放（配合 Translate 做「弹入」观感，绘制层不直接支持缩放时用于尺寸计算）。
func ScaleOf(t float32) float32 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return 0.96 + 0.04*t
}

// Scrim 遮罩层（opacity 0..1），放在 Stack 中铺满父容器。
func Scrim(t float32, key string) *Node {
	if t <= 0 {
		return nil
	}
	if t > 1 {
		t = 1
	}
	col := C(ColScrim)
	col.A *= t
	return Box().St(St().AbsFill().BgC(col).AnimSlowOn()).K(key)
}

// Duration 按 reduced-motion 设置调整过渡时长（开启时归零）。
func Duration(d float32) float32 {
	if ReducedMotion() {
		return 0
	}
	return d
}

// ReducedMotion 系统是否要求「减弱动态效果」。
func ReducedMotion() bool { return platformReducedMotion() }

// ---- 编排 ----

// animStart 返回 key 首次调用的时刻（用于手动编排时间轴）。
func animStart(key string) float32 {
	if t, ok := animStartTimes[key]; ok {
		return t
	}
	t := AnimTime()
	animStartTimes[key] = t
	return t
}

// ResetTimeline 重置某个编排时间轴。
func ResetTimeline(key string) { delete(animStartTimes, key) }

// StaggerAt 错峰动画进度：第 i 项在 start 之后延迟 i*step 开始，持续 dur。
func StaggerAt(key string, i int, step, dur float32) float32 {
	if dur <= 0 || ReducedMotion() {
		return 1
	}
	el := AnimTime() - animStart(key) - float32(i)*step
	if el <= 0 {
		markAnim()
		return 0
	}
	if el >= dur {
		return 1
	}
	markAnim()
	return EaseOutCubic(el / dur)
}

// SequenceAt 串行动画：第 i 段从 start + i*dur 开始，返回该段自身进度。
// 上一段未结束时返回 0，全部结束后返回 1。
func SequenceAt(key string, i int, dur float32) float32 {
	if dur <= 0 || ReducedMotion() {
		return 1
	}
	el := AnimTime() - animStart(key) - float32(i)*dur
	if el <= 0 {
		markAnim()
		return 0
	}
	if el >= dur {
		return 1
	}
	markAnim()
	return EaseOutCubic(el / dur)
}

// ParallelAt 并行动画：所有段同时开始，返回当前进度。
func ParallelAt(key string, dur float32) float32 {
	if dur <= 0 || ReducedMotion() {
		return 1
	}
	if p := (AnimTime() - animStart(key)) / dur; p < 1 {
		markAnim()
		if p < 0 {
			return 0
		}
		return EaseOutCubic(p)
	}
	return 1
}

// ---- 数字滚动 ----

// AnimatedNumber 数值缓动到目标值（statistic / KPI 用）。
// 小数位按 step 的小数位数保留。
func AnimatedNumber(key string, target float64, dur float32, decimals int) float64 {
	v := Tween(key, float32(target), Duration(dur), EaseOutCubic)
	if math.Abs(float64(v)-target) < 0.5*math.Pow(10, float64(-decimals)) {
		return target
	}
	return float64(v)
}

// FormatNumber 千分位格式化（decimals 为小数位）。
func FormatNumber(v float64, decimals int) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if frac != "" {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// ShimmerPhase 骨架屏流光相位 0..1（循环）。
//
// 注意：这里不能调 markAnim()。骨架屏是「装饰性慢动画」，重绘由节点的
// Style.AnimSlow 驱动（Skeleton 内部已标记）；一旦在这里点亮全局 animBusy，
// runLoop 会把它当成高帧率动画，整页回到满帧重建（实测 CPU 吃满一个核）。
func ShimmerPhase(speed float32) float32 {
	if speed <= 0 {
		speed = 1.4
	}
	t := AnimTime() * speed
	return t - float32(math.Floor(float64(t)))
}
