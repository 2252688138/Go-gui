//go:build js && wasm

package text

import "unicode/utf16"

// utf16Ptr 浏览器端没有 syscall 的 UTF16 辅助函数，这里手工转一份 NUL 结尾的 UTF-16。
// 目前只有 GDI Emoji 渲染用到（wasm 已降级），保留实现以免调用点需要区分平台。
func utf16Ptr(s string) (*uint16, error) {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	return &u[0], nil
}
