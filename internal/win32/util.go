//go:build windows

package win32

import (
	"unsafe"
)

// unsafePointer 统一别名，避免到处 import unsafe。
type unsafePointer = unsafe.Pointer

func cstr(s string) *byte {
	// 短命字符串：GL 名称，用全局池缓存避免 GC 回收导致悬垂指针
	if p, ok := nameCache[s]; ok {
		return p
	}
	buf := make([]byte, len(s)+1)
	copy(buf, s)
	p := &buf[0]
	nameCache[s] = p
	keepAlive = append(keepAlive, buf)
	return p
}

var (
	nameCache = make(map[string]*byte, 64)
	keepAlive [][]byte
)
