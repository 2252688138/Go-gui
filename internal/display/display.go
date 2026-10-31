// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package display 原生窗口层：把「创建窗口 / 消息泵 / GL 上下文 / 事件翻译」的平台差异
// 收敛成一份契约，上层的 ui.Backend 只做展示层适配，ui/app.go 则完全平台无关。
//
// 分层（与 gio 的 app 层同构）：
//
//	ui/app.go     布局 / 绘制 / 命中 / 滚动 / 动画（唯一一份，无 build tag）
//	ui/backend_*  把 display.Window 适配成 ui.Backend（每平台一份）
//	display/*     原生窗口 + 消息泵 + GL 上下文（win32 / x11 / cocoa / android / ios）
//	内部基础设施  internal/gpu（渲染器）、internal/gl（GL 函数表）、internal/text（字体）
//
// 每个平台提供：
//
//	func New(cfg Config) (Window, error) // 建窗口并创建/绑定 GL 上下文
//	func Poll() bool                     // 处理一轮消息；false = 已无窗口存活
//
// 约定：不支持的能力空实现或返回零值，绝不 panic。
package display

// EventType 事件类型。
//
// 数值与 ui.EventType **逐项对应**（ui 端直接做类型转换，见 ui/backend_windows.go）。
// 新增事件时两边必须同步追加在末尾，不要插入中间。
type EventType int

const (
	EvNone EventType = iota
	EvResize
	EvClose
	EvFocusGained
	EvFocusLost
	EvMouseMove
	EvMouseDown
	EvMouseUp
	EvMouseWheel
	EvMouseLeave
	EvKeyDown
	EvKeyUp
	EvChar
	// IME 组合输入
	EvCompositionStart
	EvComposition
	EvCompositionEnd
	// 系统拖动/缩放窗口开始与结束
	EvSizeMoveStart
	EvSizeMoveEnd
	// 文件拖入窗口（Event.Paths 为路径列表）
	EvDropFiles
	// 全局热键命中（Event.HotkeyID 为注册时给的 id）
	EvHotkey
	// 以下为多端新增：移动端触摸 / 软键盘整段输入 / 返回键
	// 触摸事件同时会把坐标合成为鼠标事件，业务代码可忽略 Pointer 字段。
	EvTouchDown
	EvTouchMove
	EvTouchUp
	// EvTextInput 软键盘一次上屏的多字符文本（Event.Text）
	EvTextInput
	// EvBack 返回键或返回手势（Android 实体键 / iOS 边缘手势）
	EvBack
)

// Event 平台无关的输入/窗口事件（坐标均为逻辑像素）。
type Event struct {
	Type   EventType
	X, Y   float32
	DX, DY float32
	Button int
	// Pointer 指针类型：0 = 鼠标，1 = 触摸/手写笔。
	Pointer int
	Key     uint32
	Char    rune
	Runes   []rune
	Width   float32
	Height  float32
	Scale   float32

	// Text / Caret 用于 IME 组合串与多字符上屏。
	Text  string
	Caret int

	// Paths 为拖入的文件/文件夹路径，HotkeyID 为全局热键 id。
	Paths    []string
	HotkeyID int
}

// Config 窗口配置（与各平台字段一一对应，缺项由后端忽略）。
type Config struct {
	Title        string
	Width        int
	Height       int
	MinWidth     int
	MinHeight    int
	Resizable    bool
	Decorated    bool
	Transparent  bool
	CornerRadius float32
	NoDrops      bool
}

// Window 原生窗口 + GL 上下文。
//
// 这是「显示层」的完整能力面：ui.Backend 在此基础上再加合成（DirectComposition）
// 与平台句柄，业务层（ui 包）只认识自己那份接口。
type Window interface {
	// NativeHandle 原生句柄：Win32 HWND / X11 Window(xid) / Cocoa NSView* / Android ANativeWindow*。
	NativeHandle() uintptr

	// ---- 呈现 ----
	MakeCurrent()              // 绑定本窗口的 GL 上下文
	Swap()                     // 交换缓冲
	PixelSize() (int32, int32) // 绘制目标物理像素尺寸
	Scale() float32            // 物理像素 / 逻辑像素

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
	SetEventHandler(fn func(Event))
	SetAcceptDrops(on bool)
	AcceptDrops() bool
	SetImePosition(x, y int32)

	// ---- 可选能力（不支持时空实现）----
	RegisterHotKey(mods, vk uint32, fn func()) int32
	UnregisterHotkey(id int32)

	// ---- 销毁 ----
	Close()
	Destroy()
}
