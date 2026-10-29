//go:build windows

package win32

import (
	"syscall"
)

// 文件拖入（shell32）：DragAcceptFiles 把窗口注册为拖放目标，松手时系统
// 投递 WM_DROPFILES，消息携带 HDROP 句柄（拖入的文件/文件夹路径列表）。
// 相对 OLE IDropTarget 方案，这条链路简单可靠，且不需要初始化 OLE。
var (
	DragAcceptFiles func(hwnd uintptr, accept int32)
	DragQueryFileW  func(hdrop uintptr, index uint32, buf *uint16, bufLen uint32) uint32
	DragQueryPoint  func(hdrop uintptr, pt *POINT) int32
	DragFinish      func(hdrop uintptr)
)

// SetAcceptDrops 开启/关闭窗口接收文件拖入（NewWindow 默认开启）。
func (w *Window) SetAcceptDrops(on bool) {
	if w == nil || w.Hwnd == 0 || DragAcceptFiles == nil {
		return
	}
	v := int32(0)
	if on {
		v = 1
	}
	w.acceptDrops = on
	DragAcceptFiles(w.Hwnd, v)
}

// AcceptDrops 当前窗口是否接收文件拖入。
func (w *Window) AcceptDrops() bool { return w != nil && w.acceptDrops }

// DragPaths 从 HDROP 句柄取出全部路径（拖入几个文件/文件夹就几项）。
// 取完后不会调用 DragFinish —— 由调用方在处理结束时释放。
func DragPaths(hdrop uintptr) []string {
	if hdrop == 0 || DragQueryFileW == nil {
		return nil
	}
	const all = 0xFFFFFFFF
	n := DragQueryFileW(hdrop, all, nil, 0)
	if n == 0 {
		return nil
	}
	buf := make([]uint16, 32768) // 长路径上限 32767 + NUL
	out := make([]string, 0, n)
	for i := uint32(0); i < n; i++ {
		c := DragQueryFileW(hdrop, i, &buf[0], uint32(len(buf)))
		if c == 0 || int(c) >= len(buf) {
			continue
		}
		out = append(out, syscall.UTF16ToString(buf[:c]))
	}
	return out
}

// DragPoint 拖放落点（客户区物理像素）。系统不允许时返回 0,0 并把 ok 置 false。
func DragPoint(hdrop uintptr) (x, y int32, ok bool) {
	if hdrop == 0 || DragQueryPoint == nil {
		return 0, 0, false
	}
	var pt POINT
	if DragQueryPoint(hdrop, &pt) == 0 {
		return pt.X, pt.Y, false
	}
	return pt.X, pt.Y, true
}

// dropEvent 把 WM_DROPFILES 转成上层事件（路径 + 逻辑坐标落点）。
func (w *Window) dropEvent(hdrop uintptr) Event {
	paths := DragPaths(hdrop)
	x, y, ok := DragPoint(hdrop)
	if !ok {
		// 落点不可用（例如拖到标题栏/非客户区）时退回当前鼠标位置
		var pt POINT
		if GetCursorPos != nil && ScreenToClient != nil {
			GetCursorPos(&pt)
			ScreenToClient(w.Hwnd, &pt)
			x, y = pt.X, pt.Y
			ok = true
		}
	}
	if DragFinish != nil && hdrop != 0 {
		DragFinish(hdrop)
	}
	if !ok {
		x, y = 0, 0
	}
	return Event{
		Type:  EvDropFiles,
		Paths: paths,
		X:     float32(x) / w.scale,
		Y:     float32(y) / w.scale,
	}
}
