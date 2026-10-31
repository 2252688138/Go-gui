// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package x11

import "unsafe"

// X11 事件解码（平台无关，纯 Go，可单元测试）。
//
// 为什么不用 Go 结构体映射：Xlib 的 XEvent 是 C 联合体，字段类型混合 int 与
// unsigned long，在 LP64 上按 8 字节对齐；用 Go 结构体映射极易因 padding 差异
// 读错字段。这里统一放进固定长度的字节缓冲，按**偏移量**读取，简单且可验证。
//
// 偏移量均按 64 位 Xlib 结构体计算：
//
//	type       0    (int)
//	serial     8    (unsigned long)
//	send_event 16   (Bool = int)
//	display    24   (*Display)
//	window     32   (Window = unsigned long)
//	…
//	x          64   (int)   —— XKeyEvent / XButtonEvent / XMotionEvent 共同偏移
//	y          68   (int)

// 事件类型（X.h）。
const (
	KeyPress        = 2
	KeyRelease      = 3
	ButtonPress     = 4
	ButtonRelease   = 5
	MotionNotify    = 6
	EnterNotify     = 7
	LeaveNotify     = 8
	FocusIn         = 9
	FocusOut        = 10
	Expose          = 12
	DestroyNotify   = 17
	UnmapNotify     = 18
	MapNotify       = 19
	ConfigureNotify = 22
	ClientMessage   = 33
)

// 事件掩码（X.h）。
const (
	KeyPressMask        = 1 << 0
	KeyReleaseMask      = 1 << 1
	ButtonPressMask     = 1 << 2
	ButtonReleaseMask   = 1 << 3
	EnterWindowMask     = 1 << 4
	LeaveWindowMask     = 1 << 5
	PointerMotionMask   = 1 << 6
	ExposureMask        = 1 << 15
	StructureNotifyMask = 1 << 17
	FocusChangeMask     = 1 << 21
	PropertyChangeMask  = 1 << 22
)

// Event 一次读取出来的原始事件（196 字节，XEvent 实际 192，留余量）。
type Event struct {
	Raw [196]byte
}

// Type 事件类型。
func (e *Event) Type() int { return int(at(e.Raw[:], 0)) }

// X / Y 指针坐标（ButtonPress / MotionNotify 等）。
func (e *Event) X() int { return int(int32(at(e.Raw[:], 64))) }
func (e *Event) Y() int { return int(int32(at(e.Raw[:], 68))) }

// State 修饰键状态。
func (e *Event) State() uint32 { return uint32(at(e.Raw[:], 80)) }

// Button 鼠标键号（1 左 / 2 中 / 3 右 / 4 上滚 / 5 下滚）。
func (e *Event) Button() int { return int(at(e.Raw[:], 84)) }

// Keycode 硬件键码。
func (e *Event) Keycode() byte { return byte(at(e.Raw[:], 84)) }

// Width / Height 新尺寸（ConfigureNotify）。
func (e *Event) Width() int  { return int(int32(at(e.Raw[:], 56))) }
func (e *Event) Height() int { return int(int32(at(e.Raw[:], 60))) }

// Data 取 ClientMessage 的第 i 个 long。
func (e *Event) Data(i int) int64 { return int64(at(e.Raw[:], 56+8*i)) }

// MessageType ClientMessage 的 message_type（Atom）。
func (e *Event) MessageType() uintptr { return uintptr(at(e.Raw[:], 40)) }

// SetField 写入一个机器字（单测构造事件用；Xlib 已做字节序转换）。
func (e *Event) SetField(off int, v uint64) { setLong(e.Raw[:], off, v) }

// at 读取 [off, off+8) 的 little-endian 机器字（Xlib 已做字节序转换）。
// 越界或负偏移返回 0：事件类型未知时字段偏移没有意义，读飞不能崩进程。
func at(b []byte, off int) uint64 {
	if off < 0 || off+8 > len(b) {
		return 0
	}
	return *(*uint64)(unsafe.Pointer(&b[off]))
}

// setLong 写入 little-endian 机器字（同样做边界校验）。
func setLong(b []byte, off int, v uint64) {
	if off < 0 || off+8 > len(b) {
		return
	}
	*(*uint64)(unsafe.Pointer(&b[off])) = v
}
