//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"syscall"
	"testing"
	"unsafe"
)

// makeHDROP 构造一份 CF_HDROP 内存块（DROPFILES 头 + 双 NUL 结尾的宽字符路径表），
// 与拖放时系统交给 WM_DROPFILES 的 HDROP 布局一致。
func makeHDROP(t *testing.T, paths ...string) uintptr {
	t.Helper()
	var list []uint16
	for _, p := range paths {
		list = append(list, syscall.StringToUTF16(p)...)
	}
	list = append(list, 0)
	hdr := uintptr(unsafe.Sizeof(dropFiles{}))
	h := GlobalAlloc(GMEMoveable, hdr+uintptr(len(list))*2)
	if h == 0 {
		t.Fatal("GlobalAlloc 失败")
	}
	p := GlobalLock(h)
	if p == 0 {
		t.Fatal("GlobalLock 失败")
	}
	df := (*dropFiles)(unsafe.Pointer(p))
	df.PFiles = uint32(hdr)
	df.FWide = 1
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p+hdr)), len(list))
	copy(dst, list)
	GlobalUnlock(h)
	return h
}

// TestDragPaths 校验 WM_DROPFILES → 路径列表的解析（多选 / 中文 / 长路径）。
func TestDragPaths(t *testing.T) {
	want := []string{
		`C:\Windows\win.ini`,
		`D:\素材\截图 2026.png`,
		`E:\a\b\c\d\e\f\一个很长很长很长很长很长很长很长很长的文件夹名.txt`,
	}
	h := makeHDROP(t, want...)
	defer GlobalFree(h)

	got := DragPaths(h)
	if len(got) != len(want) {
		t.Fatalf("路径数量 = %d，期望 %d（got=%v）", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 项 = %q，期望 %q", i, got[i], want[i])
		}
	}
}

// TestDragPathsEmpty 空句柄不应 panic。
func TestDragPathsEmpty(t *testing.T) {
	if got := DragPaths(0); got != nil {
		t.Fatalf("DragPaths(0) = %v，期望 nil", got)
	}
	if _, _, ok := DragPoint(0); ok {
		t.Fatal("DragPoint(0) 不应返回 ok")
	}
}
