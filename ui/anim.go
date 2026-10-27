package ui

import (
	"math"
	"sync"
	"sync/atomic"
)

// Ease 缓动函数：输入 0..1 输出 0..1（可越界，如回弹）。
type Ease func(t float32) float32

var (
	EaseLinear Ease = func(t float32) float32 { return t }
	EaseInQuad Ease = func(t float32) float32 { return t * t }
	EaseOutQuad Ease = func(t float32) float32 {
		return 1 - (1-t)*(1-t)
	}
	EaseInOutQuad Ease = func(t float32) float32 {
		if t < 0.5 {
			return 2 * t * t
		}
		return 1 - 2*(1-t)*(1-t)
	}
	EaseInCubic  Ease = func(t float32) float32 { return t * t * t }
	EaseOutCubic Ease = func(t float32) float32 {
		u := 1 - t
		return 1 - u*u*u
	}
	EaseInOutCubic Ease = func(t float32) float32 {
		if t < 0.5 {
			return 4 * t * t * t
		}
		u := -2*t + 2
		return 1 - u*u*u/2
	}
	EaseOutQuart Ease = func(t float32) float32 {
		u := 1 - t
		return 1 - u*u*u*u
	}
	// EaseOutBack 轻微回弹（弹出面板、按钮反馈）
	EaseOutBack Ease = func(t float32) float32 {
		const c1 = 1.70158
		const c3 = c1 + 1
		u := t - 1
		return 1 + c3*u*u*u + c1*u*u
	}
	// EaseOutElastic 弹簧回弹
	EaseOutElastic Ease = func(t float32) float32 {
		if t <= 0 {
			return 0
		}
		if t >= 1 {
			return 1
		}
		const c4 = float32(2 * math.Pi / 3)
		return float32(math.Pow(2, -10*float64(t)))*float32(math.Sin((float64(t)*10-0.75)*float64(c4))) + 1
	}
	EaseOutBounce Ease = func(t float32) float32 {
		const n1 = 7.5625
		const d1 = 2.75
		switch {
		case t < 1/d1:
			return n1 * t * t
		case t < 2/d1:
			t -= 1.5 / d1
			return n1*t*t + 0.75
		case t < 2.5/d1:
			t -= 2.25 / d1
			return n1*t*t + 0.9375
		default:
			t -= 2.625 / d1
			return n1*t*t + 0.984375
		}
	}
)

// animBusy 标记本帧存在活动动画（绘制结束后据此持续重绘）。
var animBusy atomic.Bool

func markAnim() { animBusy.Store(true) }

// Tween 按 key 做补间动画，返回当前值。
// target 变化时从当前值平滑过渡到新目标；dur 单位为秒。
func Tween(key string, target float32, dur float32, ease Ease) float32 {
	if dur <= 0 {
		dur = 0.2
	}
	if ease == nil {
		ease = EaseOutCubic
	}
	now := AnimTime()
	animMu.Lock()
	defer animMu.Unlock()
	if tweenStates == nil {
		tweenStates = map[string]*tweenState{}
	}
	st := tweenStates[key]
	if st == nil {
		st = &tweenState{from: target, cur: target, target: target, start: now, dur: dur, ease: ease}
		tweenStates[key] = st
		return target
	}
	if st.target != target {
		st.from = st.cur
		st.target = target
		st.start = now
		st.dur = dur
		st.ease = ease
	}
	p := (now - st.start) / st.dur
	if p >= 1 {
		st.cur = st.target
		st.from = st.target
		return st.cur
	}
	if p < 0 {
		p = 0
	}
	st.cur = st.from + (st.target-st.from)*st.ease(p)
	markAnim()
	return st.cur
}

// TweenFrom 首次调用时以 initial 作为起点（用于入场动画）。
func TweenFrom(key string, initial, target, dur float32, ease Ease) float32 {
	animMu.Lock()
	_, ok := tweenStates[key]
	animMu.Unlock()
	if !ok {
		animMu.Lock()
		if tweenStates == nil {
			tweenStates = map[string]*tweenState{}
		}
		if _, ok2 := tweenStates[key]; !ok2 {
			tweenStates[key] = &tweenState{from: initial, cur: initial, target: initial, start: AnimTime(), dur: dur, ease: ease}
		}
		animMu.Unlock()
	}
	return Tween(key, target, dur, ease)
}

// Spring 弹簧动画：按刚度/阻尼做数值积分，产生自然的过冲与回弹。
func Spring(key string, target float32, stiffness, damping float32) float32 {
	if stiffness <= 0 {
		stiffness = 220
	}
	if damping <= 0 {
		damping = 22
	}
	now := AnimTime()
	animMu.Lock()
	defer animMu.Unlock()
	if springStates == nil {
		springStates = map[string]*springState{}
	}
	st := springStates[key]
	if st == nil {
		springStates[key] = &springState{x: target, v: 0, target: target, last: now}
		return target
	}
	dt := now - st.last
	if dt <= 0 {
		dt = 1.0 / 120
	}
	if dt > 1.0/20 {
		dt = 1.0 / 20
	}
	st.last = now
	st.target = target
	// 半隐式欧拉积分
	steps := int(dt*240) + 1
	h := dt / float32(steps)
	for i := 0; i < steps; i++ {
		acc := stiffness*(st.target-st.x) - damping*st.v
		st.v += acc * h
		st.x += st.v * h
	}
	if math.Abs(float64(st.target-st.x)) < 0.01 && math.Abs(float64(st.v)) < 0.01 {
		st.x = st.target
		st.v = 0
		return st.x
	}
	markAnim()
	return st.x
}

// Key 时间轴关键帧。
type Key struct {
	At    float32 // 0..1
	Value float32
	Ease  Ease
}

// Keyframes 按关键帧插值（自动循环或单次）。
func Keyframes(key string, keys []Key, dur float32, loop bool) float32 {
	if len(keys) == 0 {
		return 0
	}
	if dur <= 0 {
		dur = 1
	}
	now := AnimTime()
	animMu.Lock()
	defer animMu.Unlock()
	if timelineStates == nil {
		timelineStates = map[string]*timelineState{}
	}
	st := timelineStates[key]
	if st == nil {
		st = &timelineState{start: now}
		timelineStates[key] = st
	}
	p := (now - st.start) / dur
	if loop {
		p -= float32(math.Floor(float64(p)))
	} else if p >= 1 {
		p = 1
	} else {
		markAnim()
	}
	if p < 0 {
		p = 0
	}
	// 找到 p 所在区间
	prev := keys[0]
	for i := 1; i < len(keys); i++ {
		k := keys[i]
		if p <= k.At {
			span := k.At - prev.At
			if span <= 0 {
				return k.Value
			}
			t := (p - prev.At) / span
			e := k.Ease
			if e == nil {
				e = EaseLinear
			}
			return prev.Value + (k.Value-prev.Value)*e(t)
		}
		prev = k
	}
	return prev.Value
}

// ResetAnim 重置指定 key 的动画（重建入场动画时用）。
func ResetAnim(key string) {
	animMu.Lock()
	delete(tweenStates, key)
	delete(springStates, key)
	delete(timelineStates, key)
	animMu.Unlock()
}

// AnimActive 是否仍有动画在播放（业务一般不需要调用）。
func AnimActive() bool { return animBusy.Load() }

// ---- 内部状态 ----

type tweenState struct {
	from, cur, target float32
	start, dur        float32
	ease              Ease
}

type springState struct {
	x, v, target float32
	last         float32
}

type timelineState struct {
	start float32
}

var (
	animMu         sync.Mutex
	tweenStates    map[string]*tweenState
	springStates   map[string]*springState
	timelineStates map[string]*timelineState
)
