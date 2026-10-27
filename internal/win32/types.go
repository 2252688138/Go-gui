package win32

type POINT struct{ X, Y int32 }

// MONITORINFO 显示器信息（GetMonitorInfoW 使用，CbSize 需先赋值）。
type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

type RECT struct{ Left, Top, Right, Bottom int32 }

type MSG struct {
	Hwnd     uintptr
	Message  uint32
	_        uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type PIXELFORMATDESCRIPTOR struct {
	NSize           uint16
	NVersion        uint16
	DwFlags         uint32
	IPixelType      uint8
	CColorBits      uint8
	CRedBits        uint8
	CRedShift       uint8
	CGreenBits      uint8
	CGreenShift     uint8
	CBlueBits       uint8
	CBlueShift      uint8
	CAlphaBits      uint8
	CAlphaShift     uint8
	CAccumBits      uint8
	CAccumRedBits   uint8
	CAccumGreenBits uint8
	CAccumBlueBits  uint8
	CAccumAlphaBits uint8
	CDepthBits      uint8
	CStencilBits    uint8
	CAuxBuffers     uint8
	ILayerType      uint8
	BReserved       uint8
	DwLayerMask     uint32
	DwVisibleMask   uint32
	DwDamageMask    uint32
}

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_POPUP            = 0x80000000
	WS_VISIBLE          = 0x10000000
	WS_CLIPCHILDREN     = 0x02000000

	// 全屏相关：去标题栏 + 覆盖当前显示器
	GWL_STYLE                = -16
	MONITOR_DEFAULTTONEAREST = 2
	MONITOR_DEFAULTTOPRIMARY = 1
	SWP_NOSIZE               = 0x0001
	SWP_NOMOVE               = 0x0002
	SWP_NOZORDER             = 0x0004
	SWP_NOACTIVATE           = 0x0010
	SWP_FRAMECHANGED         = 0x0020
	SWP_SHOWWINDOW           = 0x0040

	// 窗口圆角：DwmSetWindowAttribute 的属性号与取值
	DWMWA_WINDOW_CORNER_PREFERENCE = 33
	DWMWCP_DONOTROUND              = 1
	DWMWCP_ROUND                   = 2
	DWMWCP_ROUNDSMALL              = 3

	// 无边框窗口：去掉 DWM 绘制的 1px 系统边框（Win11 上表现为四周白色细线）
	DWMWA_NCRENDERING_POLICY = 2
	DWMWA_BORDER_COLOR       = 34
	DWMWA_COLOR_NONE         = 0xFFFFFFFE
	DWMNCRP_DISABLED         = 1

	CS_VREDRAW = 0x0001
	CS_HREDRAW = 0x0002
	CS_DBLCLKS = 0x0008
	CS_OWNDC   = 0x0020

	CW_USEDEFAULT = -2147483648
	SW_SHOW       = 5
	SW_MAXIMIZE   = 3

	WM_NULL          = 0x0000
	WM_CREATE        = 0x0001
	WM_DESTROY       = 0x0002
	WM_SIZE          = 0x0005
	// 系统拖动/缩放窗口的模态循环起点与终点
	WM_ENTERSIZEMOVE = 0x0231
	WM_EXITSIZEMOVE  = 0x0232
	WM_ACTIVATE      = 0x0006
	WM_SETFOCUS      = 0x0007
	WM_KILLFOCUS     = 0x0008
	WM_PAINT         = 0x000F
	WM_CLOSE         = 0x0010
	WM_QUIT          = 0x0012
	WM_ERASEBKGND    = 0x0014
	WM_SETCURSOR     = 0x0020
	WM_GETMINMAXINFO = 0x0024
	WM_NCHITTEST     = 0x0084
	WM_NCACTIVATE    = 0x0086
	WM_NCCALCSIZE    = 0x0083
	// 非客户区命中测试返回值（自绘边框时用于保留窗口缩放能力）
	HTLEFT           = 10
	HTRIGHT          = 11
	HTTOP            = 12
	HTTOPLEFT        = 13
	HTTOPRIGHT       = 14
	HTBOTTOM         = 15
	HTBOTTOMLEFT     = 16
	HTBOTTOMRIGHT    = 17
	// 客户区（自绘标题栏）拖动的命中值
	HTCAPTION        = 2
	WM_NCLBUTTONDOWN = 0x00A1
	WM_KEYDOWN       = 0x0100
	WM_KEYUP         = 0x0101
	WM_CHAR          = 0x0102
	WM_SYSKEYDOWN    = 0x0104
	WM_SYSKEYUP      = 0x0105
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONDOWN   = 0x0204
	WM_RBUTTONUP     = 0x0205
	WM_MOUSEWHEEL    = 0x020A
	WM_MOUSELEAVE    = 0x02A3
	WM_DPICHANGED    = 0x02E0

	PM_REMOVE   = 0x0001
	PM_NOREMOVE = 0x0000

	IDC_ARROW = 32512

	PFD_DOUBLEBUFFER   = 0x00000001
	PFD_DRAW_TO_WINDOW = 0x00000004
	PFD_SUPPORT_OPENGL = 0x00000020
	PFD_TYPE_RGBA      = 0
	PFD_MAIN_PLANE     = 0

	GGO_BITMAP       = 1
	GGO_NATIVE       = 2
	GGO_GRAY8_BITMAP = 6

	ANTIALIASED_QUALITY = 4
	CLEARTYPE_QUALITY   = 5
	DEFAULT_CHARSET     = 1
	DEFAULT_PITCH       = 0
	FW_NORMAL           = 400
	FW_MEDIUM           = 500
	FW_SEMIBOLD         = 600
	FW_BOLD             = 700
	TRANSPARENT         = 1

	VK_ESCAPE  = 0x1B
	VK_RETURN  = 0x0D
	VK_SPACE   = 0x20
	VK_BACK    = 0x08
	VK_TAB     = 0x09
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_MENU    = 0x12
)

// ---- GDI 位图（Emoji 彩色光栅化）----

// BITMAPINFOHEADER GDI 位图头。
type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// BITMAPINFO 位图信息（单色表）。
type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [1]uint32
}

// SIZE 尺寸。
type SIZE struct{ Cx, Cy int32 }

const (
	DIB_RGB_COLORS = 0
	BI_RGB         = 0

	DT_LEFT       = 0x00000000
	DT_TOP        = 0x00000000
	DT_NOCLIP     = 0x00000100
	DT_SINGLELINE = 0x00000020
	DT_NOPREFIX   = 0x00000800
)

// LOWORD / HIWORD 提取|lParam 坐标。
func loword(v uintptr) int32 { return int32(int16(v & 0xFFFF)) }
func hiword(v uintptr) int32 { return int32(int16((v >> 16) & 0xFFFF)) }

// ---- IME ----

// COMPOSITIONFORM 输入法组合窗口位置（ImmSetCompositionWindow）。
type COMPOSITIONFORM struct {
	DwStyle      uint32
	PtCurrentPos POINT
	RcArea       RECT
}

// CANDIDATEFORM 输入法候选窗口位置（ImmSetCandidateWindow）。
type CANDIDATEFORM struct {
	DwIndex      uint32
	DwStyle      uint32
	PtCurrentPos POINT
	RcArea       RECT
}

// ---- 文件对话框 ----

// OPENFILENAMEW GetOpenFileNameW / GetSaveFileNameW 参数。
type OPENFILENAMEW struct {
	LStructSize       uint32
	HwndOwner         uintptr
	HInstance         uintptr
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        uintptr
	DwReserved        uint32
	FlagsEx           uint32
}

// BROWSEINFOW 选择文件夹对话框参数。
type BROWSEINFOW struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

// ---- 托盘 ----

// NOTIFYICONDATAW 任务栏托盘图标数据（V4）。
type NOTIFYICONDATAW struct {
	CbSize           uint32
	Hwnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

const (
	WM_IME_STARTCOMPOSITION = 0x010D
	WM_IME_ENDCOMPOSITION   = 0x010E
	WM_IME_COMPOSITION      = 0x010F
	WM_IME_SETCONTEXT       = 0x0281
	WM_IME_NOTIFY           = 0x0282
	WM_IME_CHAR             = 0x0286
	WM_COMMAND              = 0x0111
	WM_USER                 = 0x0400
	WM_TRAYICON             = WM_USER + 1
	WM_APP                  = 0x8000

	GCS_COMPSTR    = 0x0008
	GCS_COMPATTR   = 0x0010
	GCS_CURSORPOS  = 0x0080
	GCS_RESULTSTR  = 0x0800
	CFS_POINT      = 0x0002
	CFS_RECT       = 0x0001
	WA_INACTIVE    = 0
	ISC_SHOWUICOMP = 0x80000000

	NIM_ADD        = 0
	NIM_MODIFY     = 1
	NIM_DELETE     = 2
	NIM_SETVERSION = 4

	NIF_MESSAGE          = 0x00000001
	NIF_ICON             = 0x00000002
	NIF_TIP              = 0x00000004
	NIF_SHOWTIP          = 0x00000080
	NIF_INFO             = 0x00000010
	NIIF_INFO            = 0x00000001
	NIIF_USER            = 0x00000004
	NOTIFYICON_VERSION_4 = 4

	OFN_READONLY        = 0x00000001
	OFN_OVERWRITEPROMPT = 0x00000002
	OFN_HIDEREADONLY    = 0x00000004
	OFN_NOCHANGEDIR     = 0x00000008
	OFN_PATHMUSTEXIST   = 0x00000800
	OFN_FILEMUSTEXIST   = 0x00001000
	OFN_EXPLORER        = 0x00080000

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800
	MF_GRAYED    = 0x00000001
	MF_CHECKED   = 0x00000008
	MF_POPUP     = 0x00000010

	TPM_LEFTALIGN   = 0x0000
	TPM_CENTERALIGN = 0x0004
	TPM_RIGHTALIGN  = 0x0008
	TPM_BOTTOMALIGN = 0x0020
	TPM_LEFTBUTTON  = 0x0000
	TPM_RIGHTBUTTON = 0x0002
	TPM_NONOTIFY    = 0x0080
	TPM_RETURNCMD   = 0x0100

	BIF_RETURNONLYFSDIRS = 0x00000001
	BIF_EDITBOX          = 0x00000010
	BIF_NEWDIALOGSTYLE   = 0x00000040

	IDI_APPLICATION = 32512
	IDC_HAND        = 32649
	IDC_IBEAM       = 32513

	SW_HIDE     = 0
	SW_MINIMIZE = 6
	SW_RESTORE  = 9
)
