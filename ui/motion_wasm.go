//go:build js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "syscall/js"

// platformReducedMotionImpl 读取浏览器 prefers-reduced-motion。
func platformReducedMotionImpl() bool {
	mm := js.Global().Get("matchMedia")
	if mm.Type() != js.TypeFunction {
		return false
	}
	q := mm.Invoke("(prefers-reduced-motion: reduce)")
	if q.IsUndefined() || q.IsNull() {
		return false
	}
	return q.Get("matches").Bool()
}
