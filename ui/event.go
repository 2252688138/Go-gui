package ui

// Event 平台无关的输入/窗口事件。桌面端由 Win32 消息翻译而来，浏览器端由 DOM 事件翻译，
// 两边共用同一套分发逻辑（App.handleEvent）。
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
	// IME 组合输入（中文输入法）
	EvCompositionStart
	EvComposition
	EvCompositionEnd
	// 系统拖动/缩放窗口开始与结束
	EvSizeMoveStart
	EvSizeMoveEnd
	// 文件拖入窗口，Paths 为路径列表，X/Y 为松手落点
	EvDropFiles
	// 全局热键命中，HotkeyID 为注册时给的 id
	EvHotkey
)

// Event 一个输入事件（坐标均为逻辑像素）。
type Event struct {
	Type   EventType
	X, Y   float32
	DX, DY float32
	Button int
	Key    uint32
	Char   rune
	Runes  []rune
	Width  float32
	Height float32
	Scale  float32

	Text  string
	Caret int

	Paths    []string
	HotkeyID int
}

// 常用虚拟键码（与 Windows VK_* 一致）。
const (
	VK_BACK   = 0x08
	VK_TAB    = 0x09
	VK_RETURN = 0x0D
	VK_SHIFT  = 0x10
	VK_ESCAPE = 0x1B
	VK_SPACE  = 0x20
	VK_LEFT   = 0x25
	VK_UP     = 0x26
	VK_RIGHT  = 0x27
	VK_DOWN   = 0x28
	VK_DELETE = 0x2E
)
