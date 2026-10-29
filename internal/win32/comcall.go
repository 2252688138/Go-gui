//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

// 供诊断程序 / 外部包使用的 COM 调用导出（内部实现见 mf.go 的 comCall）。

// ComCallHR 调用 obj 的第 slot 个虚表方法，参数按整数/指针传入，返回 HRESULT。
func ComCallHR(obj uintptr, slot int, args ...uintptr) uint32 {
	return comCall(obj, slot, args...)
}

// ComRelease 释放一个 COM 引用（对应 IUnknown::Release）。
func ComRelease(obj uintptr) { comRelease(obj) }
