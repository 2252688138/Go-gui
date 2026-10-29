//go:build js && wasm

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
