//go:build windows && !(cgo && gogui_cgo)

package win32

// 纯 Go 构建：没有 cgo，无法可靠地调用带 float 参数的 COM 方法
// （DirectComposition 的几何接口全是 float，Win64 走 XMM 寄存器）。
// B 阶段（同层合成）因此只在 cgo 构建下启用；纯 Go 构建下 WebView 组件走 A 阶段。

// CgoEnabled 是否以 cgo 构建（决定 WebView 走 A 还是 B 阶段）。
func CgoEnabled() bool { return false }
