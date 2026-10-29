//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "syscall/js"

// platformPrefersDark 浏览器端用 prefers-color-scheme 媒体查询。
func platformPrefersDark() bool {
	mq := js.Global().Call("matchMedia", "(prefers-color-scheme: dark)")
	if mq.IsNull() || mq.IsUndefined() {
		return true
	}
	return mq.Get("matches").Bool()
}
