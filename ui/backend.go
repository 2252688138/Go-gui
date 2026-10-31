// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// Backend 平台窗口后端：把「窗口 + 消息泵 + GL 上下文」的平台差异收敛成一组方法，
// 让 app.go 里的布局、绘制、命中测试、滚动与事件分发完全不认识任何平台细节。
//
// 每个平台一份实现（文件名 backend_<os>.go），结构体统一叫 platformWin，
// 并提供两个包级入口：
//
//	func newBackend(opts Options) (*platformWin, error) // 创建窗口与 GL 上下文
//	func pollMessages() bool                            // 处理一轮消息，false = 已无窗口
//
// 这与 gio 的 app.Window 分层同构：新增平台只加文件，不改 app.go。
//
// 约定：**不支持的能力一律空操作或返回零值，绝不 panic、绝不 nil 解引用**，
// 业务代码可通过 Capabilities() / HasCap() 查询后再决定是否使用。
type Backend interface {
	// ---- 生命周期与呈现 ----
	SetEventHandler(fn func(Event)) // 注册事件回调（App.handleEvent）
	MakeCurrent()                   // 绑定本窗口的 GL 上下文
	Swap()                          // 交换缓冲，呈现一帧
	PixelSize() (int32, int32)      // 绘制目标的物理像素尺寸
	Scale() float32                 // 物理像素 / 逻辑像素

	// ---- 窗口状态 ----
	IsMinimized() bool
	SetTitle(s string)
	SetIcon(ico []byte) bool
	SetVisible(v bool)
	Minimize()
	Restore()
	Center()
	BeginDragMove()
	ToggleMaximize()
	IsMaximized() bool
	SetFullscreen(v bool)
	IsFullscreen() bool
	SetAlwaysOnTop(on bool)
	AlwaysOnTop() bool
	SetCornerRadius(v float32)
	CornerRadius() float32

	// ---- 输入 ----
	SetAcceptDrops(on bool)
	AcceptDrops() bool
	SetImePosition(x, y int32)

	// ---- 可选能力（不支持时空实现）----
	RegisterHotKey(mods, vk uint32, fn func()) int32
	UnregisterHotkey(id int32)

	// ---- 合成（仅 Windows 的 DirectComposition 有实际实现）----
	AttachCompose(pw, ph int32)
	HasCompose() bool
	ResizeCompose(pw, ph int32)
	ComposeBegin() uint32
	ComposeEnd()

	// ---- 原生句柄 ----
	// Windows: HWND；X11: Window(xid)；Cocoa: NSView*；Android: ANativeWindow*；Web: 0。
	NativeHandle() uintptr

	// ---- 销毁 ----
	Close()
	Destroy()
}

// composerBackend 可选接口：仅 Windows 的 DirectComposition 合成器实现，
// webview.go（windows 专用）通过类型断言取回合成器做图层偏移与圆角裁剪。
type composerBackend interface {
	Compositor() any
}
