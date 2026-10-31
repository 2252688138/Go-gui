//go:build js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "syscall/js"

// prefsStorageKey localStorage 键名。
const prefsStorageKey = "gogui.prefs"

// prefsLoadRaw 从 localStorage 读取偏好（Web 端没有文件路径）。
func prefsLoadRaw() ([]byte, string, error) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return nil, "", nil
	}
	v := ls.Call("getItem", prefsStorageKey)
	if v.IsNull() || v.IsUndefined() {
		return nil, "", nil
	}
	return []byte(v.String()), "", nil
}

// prefsSaveRaw 写入 localStorage。
func prefsSaveRaw(buf []byte) error {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return nil
	}
	ls.Call("setItem", prefsStorageKey, string(buf))
	return nil
}
