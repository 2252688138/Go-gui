//go:build windows && !amd64 && !(cgo && gogui_cgo)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

// 非 amd64 且无 cgo：没有 float 调用能力，只提供占位实现（B 阶段不可用）。

// ComSlot 取 COM 对象指定虚表槽位的函数地址。
func ComSlot(obj uintptr, slot int) uintptr { return 0 }

// ComCallF 调用 fn(self, float)（不可用）。
func ComCallF(obj uintptr, slot int, v float32) uint32 { return 0x80004005 }

// ComCallFF 调用 fn(self, float, float)（不可用）。
func ComCallFF(obj uintptr, slot int, a, b float32) uint32 { return 0x80004005 }

// ComCallF1P 调用 fn(self, float, void*)（不可用）。
func ComCallF1P(obj uintptr, slot int, a float32, p uintptr) uint32 { return 0x80004005 }

// ComCallP1F 调用 fn(self, void*, float)（不可用）。
func ComCallP1F(obj uintptr, slot int, p uintptr, a float32) uint32 { return 0x80004005 }
