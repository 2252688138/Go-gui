//go:build windows

package win32

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	libUser32   uintptr
	libGdi32    uintptr
	libKernel32 uintptr
	libOpengl32 uintptr
	libImm32    uintptr
	libComdlg32 uintptr
	libShell32  uintptr
	libOle32    uintptr
	libDwmapi   uintptr
	libAdvapi32 uintptr
)

func loadLib(name string) uintptr {
	h, err := syscall.LoadLibrary(name)
	if err != nil {
		panic("win32: cannot load " + name + ": " + err.Error())
	}
	return uintptr(h)
}

// loadLibOptional 加载可选 DLL，加载失败返回 0（老系统上可能没有该库）。
func loadLibOptional(name string) uintptr {
	h, err := syscall.LoadLibrary(name)
	if err != nil {
		return 0
	}
	return uintptr(h)
}

func bind(lib uintptr, fn any, name string) {
	purego.RegisterLibFunc(fn, lib, name)
}

func procAddr(lib uintptr, name string) (uintptr, error) {
	return syscall.GetProcAddress(syscall.Handle(lib), name)
}

// Debug 开启后向 stderr 输出诊断日志。
var Debug = true

func dbg(format string, args ...any) {
	if Debug {
		fmt.Fprintf(os.Stderr, "[win32] "+format+"\n", args...)
	}
}

var (
	RegisterClassExW              func(wc *WNDCLASSEXW) uint16
	UnregisterClassW              func(name *uint16, hinst uintptr) int32
	CreateWindowExW               func(exStyle uint32, class, title *uint16, style uint32, x, y, w, h int32, parent, menu, inst, param uintptr) uintptr
	DefWindowProcW                func(hwnd, msg, wp, lp uintptr) uintptr
	DestroyWindow                 func(hwnd uintptr) int32
	ShowWindow                    func(hwnd uintptr, cmd int32) int32
	UpdateWindow                  func(hwnd uintptr) int32
	PeekMessageW                  func(msg *MSG, hwnd uintptr, min, max uint32, remove uint32) int32
	TranslateMessage              func(msg *MSG) int32
	DispatchMessageW              func(msg *MSG) uintptr
	PostQuitMessage               func(code int32)
	PostMessageW                  func(hwnd uintptr, msg uint32, wp, lp uintptr) int32
	GetClientRect                 func(hwnd uintptr, r *RECT) int32
	GetWindowRect                 func(hwnd uintptr, r *RECT) int32
	ScreenToClient                func(hwnd uintptr, p *POINT) int32
	AdjustWindowRectEx            func(r *RECT, style uint32, menu int32, exStyle uint32) int32
	LoadCursorW                   func(inst uintptr, name uintptr) uintptr
	SetCursor                     func(cursor uintptr) uintptr
	GetDC                         func(hwnd uintptr) uintptr
	ReleaseDC                     func(hwnd uintptr, hdc uintptr) int32
	InvalidateRect                func(hwnd uintptr, r *RECT, erase int32) int32
	ValidateRect                  func(hwnd uintptr, r *RECT) int32
	SetCapture                    func(hwnd uintptr) uintptr
	ReleaseCapture                func() int32
	SetWindowTextW                func(hwnd uintptr, text *uint16) int32
	GetDpiForWindow               func(hwnd uintptr) uint32
	SetProcessDpiAwarenessContext func(ctx uintptr) int32
	SetProcessDPIAware            func() int32
	SetFocus                      func(hwnd uintptr) uintptr
	CreateCompatibleDC            func(hdc uintptr) uintptr
	DeleteDC                      func(hdc uintptr) int32
	GetDeviceCaps                 func(hdc uintptr, index int32) int32
	GetForegroundWindow           func() uintptr
	TrackMouseEvent               func(tme *TRACKMOUSEEVENT) int32
	// 自绘标题栏：SendMessage(WM_NCLBUTTONDOWN, HTCAPTION) 交给系统拖动窗口
	SendMessageW func(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr
	IsZoomed     func(hwnd uintptr) int32

	// 多窗口 / 托盘 / 菜单 / 光标
	SetForegroundWindow    func(hwnd uintptr) int32
	SetWindowPos           func(hwnd, after uintptr, x, y, cx, cy int32, flags uint32) int32
	GetCursorPos           func(p *POINT) int32
	LoadIconW              func(inst uintptr, name uintptr) uintptr
	CreatePopupMenu        func() uintptr
	AppendMenuW            func(menu uintptr, flags uint32, id uintptr, item *uint16) int32
	TrackPopupMenu         func(menu uintptr, flags uint32, x, y, reserved int32, hwnd uintptr, rect uintptr) int32
	DestroyMenu            func(menu uintptr) int32
	RegisterWindowMessageW func(name *uint16) uint32
	IsWindow               func(hwnd uintptr) int32
	IsWindowVisible        func(hwnd uintptr) int32
	IsIconic               func(hwnd uintptr) int32
	SetActiveWindow        func(hwnd uintptr) uintptr

	// 全屏（无边框覆盖当前显示器）
	GetWindowLongPtrW func(hwnd uintptr, index int32) uintptr
	SetWindowLongPtrW func(hwnd uintptr, index int32, v uintptr) uintptr
	MonitorFromWindow func(hwnd uintptr, flags uint32) uintptr
	GetMonitorInfoW   func(hmon uintptr, mi *MONITORINFO) int32
)

// 窗口圆角（DWM 原生圆角优先；dwmapi 缺失时回退 SetWindowRgn 区域圆角）
var (
	DwmSetWindowAttribute func(hwnd uintptr, attr uint32, pv unsafe.Pointer, cb uint32) int32
	// DwmExtendFrameIntoClientArea：把 DWM 边框向外扩展，用于彻底去掉 1px 边框线
	DwmExtendFrameIntoClientArea func(hwnd uintptr, m *MARGINS) int32
	SetWindowRgn                 func(hwnd, hrgn uintptr, redraw int32) int32
	CreateRoundRectRgn           func(x1, y1, x2, y2, w, h int32) uintptr
)

// MARGINS DWM 边框扩展尺寸。
type MARGINS struct {
	Left, Right, Top, Bottom int32
}

// IME（imm32）
var (
	ImmGetContext            func(hwnd uintptr) uintptr
	ImmReleaseContext        func(hwnd uintptr, himc uintptr) int32
	ImmGetCompositionStringW func(himc uintptr, index uint32, buf uintptr, bufLen int32) int32
	ImmSetCompositionWindow  func(himc uintptr, cf *COMPOSITIONFORM) int32
	ImmSetCandidateWindow    func(himc uintptr, cf *CANDIDATEFORM) int32
)

// 文件对话框 / 选择文件夹（comdlg32、shell32）
var (
	GetOpenFileNameW     func(ofn *OPENFILENAMEW) int32
	GetSaveFileNameW     func(ofn *OPENFILENAMEW) int32
	Shell_NotifyIconW    func(msg uint32, data *NOTIFYICONDATAW) int32
	SHBrowseForFolderW   func(bi *BROWSEINFOW) uintptr
	SHGetPathFromIDListW func(pidl uintptr, path *uint16) int32
	ShellExecuteW        func(hwnd uintptr, op, file, params, dir *uint16, show int32) uintptr
	CoTaskMemFree        func(p uintptr)
)

var (
	GetModuleHandleW func(name *uint16) uintptr
	GetLastError     func() uint32
)

var (
	ChoosePixelFormat   func(hdc uintptr, pfd *PIXELFORMATDESCRIPTOR) int32
	SetPixelFormat      func(hdc uintptr, format int32, pfd *PIXELFORMATDESCRIPTOR) int32
	DescribePixelFormat func(hdc uintptr, format int32, size uint32, pfd *PIXELFORMATDESCRIPTOR) int32
	SwapBuffers         func(hdc uintptr) int32
	CreateFontW         func(h, w, esc, orient, weight int32, italic, underline, strike uint32, charset, outPrec, clipPrec, quality, pitch int32, face *uint16) uintptr
	SelectObject        func(hdc uintptr, obj uintptr) uintptr
	DeleteObject        func(obj uintptr) int32
	CreateSolidBrush    func(color uint32) uintptr
	SetBkMode           func(hdc uintptr, mode int32) int32
	SetTextColor        func(hdc uintptr, color uint32) uint32
	GetGlyphOutlineW    func(hdc uintptr, ch uint32, format uint32, gm *GLYPHMETRICS, bufSize uint32, buf uintptr, mat *MAT2) uint32
	GetTextMetricsW     func(hdc uintptr, tm *TEXTMETRICW) int32

	// Emoji 彩色字形（DIB 光栅化）
	CreateDIBSection      func(hdc uintptr, pbmi *BITMAPINFO, usage uint32, bits *unsafe.Pointer, section uintptr, offset uint32) uintptr
	DrawTextW             func(hdc uintptr, text *uint16, count int32, rect *RECT, format uint32) int32
	SetBkColor            func(hdc uintptr, color uint32) uint32
	GdiFlush              func() int32
	GetTextExtentPoint32W func(hdc uintptr, s *uint16, n int32, size *SIZE) int32
)

var (
	wglCreateContext     func(hdc uintptr) uintptr
	wglMakeCurrent       func(hdc, rc uintptr) int32
	wglDeleteContext     func(rc uintptr) int32
	wglGetProcAddress    func(name *byte) uintptr
	wglGetCurrentContext func() uintptr
)

type TRACKMOUSEEVENT struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

type FIXED struct {
	Fract uint16
	Value int16
}

type MAT2 [4]FIXED

func identityMAT2() MAT2 {
	var m MAT2
	m[0].Value = 1
	m[3].Value = 1
	return m
}

type GLYPHMETRICS struct {
	BlackBoxX uint32
	BlackBoxY uint32
	OriginX   int32
	OriginY   int32
	CellIncX  int16
	CellIncY  int16
}

type TEXTMETRICW struct {
	Height           int32
	Ascent           int32
	Descent          int32
	InternalLeading  int32
	ExternalLeading  int32
	AveCharWidth     int32
	MaxCharWidth     int32
	Weight           int32
	Overhang         int32
	DigitizedAspectX int32
	DigitizedAspectY int32
	FirstChar        uint16
	LastChar         uint16
	DefaultChar      uint16
	BreakChar        uint16
	Italic           uint8
	Underlined       uint8
	StruckOut        uint8
	PitchAndFamily   uint8
	FaceName         [32]uint16
}

func init() {
	libKernel32 = loadLib("kernel32.dll")
	libUser32 = loadLib("user32.dll")
	libGdi32 = loadLib("gdi32.dll")
	libOpengl32 = loadLib("opengl32.dll")
	libImm32 = loadLib("imm32.dll")
	libComdlg32 = loadLib("comdlg32.dll")
	libShell32 = loadLib("shell32.dll")
	libOle32 = loadLib("ole32.dll")
	libDwmapi = loadLibOptional("dwmapi.dll")
	libAdvapi32 = loadLib("advapi32.dll")

	bind(libUser32, &RegisterClassExW, "RegisterClassExW")
	bind(libUser32, &UnregisterClassW, "UnregisterClassW")
	bind(libUser32, &CreateWindowExW, "CreateWindowExW")
	bind(libUser32, &DefWindowProcW, "DefWindowProcW")
	bind(libUser32, &DestroyWindow, "DestroyWindow")
	bind(libUser32, &ShowWindow, "ShowWindow")
	bind(libUser32, &UpdateWindow, "UpdateWindow")
	bind(libUser32, &PeekMessageW, "PeekMessageW")
	bind(libUser32, &TranslateMessage, "TranslateMessage")
	bind(libUser32, &DispatchMessageW, "DispatchMessageW")
	bind(libUser32, &PostQuitMessage, "PostQuitMessage")
	bind(libUser32, &PostMessageW, "PostMessageW")
	bind(libUser32, &GetClientRect, "GetClientRect")
	bind(libUser32, &GetWindowRect, "GetWindowRect")
	bind(libUser32, &ScreenToClient, "ScreenToClient")
	bind(libUser32, &AdjustWindowRectEx, "AdjustWindowRectEx")
	bind(libUser32, &LoadCursorW, "LoadCursorW")
	bind(libUser32, &SetCursor, "SetCursor")
	bind(libUser32, &GetDC, "GetDC")
	bind(libUser32, &ReleaseDC, "ReleaseDC")
	bind(libUser32, &InvalidateRect, "InvalidateRect")
	bind(libUser32, &ValidateRect, "ValidateRect")
	bind(libUser32, &SetCapture, "SetCapture")
	bind(libUser32, &ReleaseCapture, "ReleaseCapture")
	bind(libUser32, &SetWindowTextW, "SetWindowTextW")
	bind(libUser32, &SetFocus, "SetFocus")
	bind(libUser32, &GetForegroundWindow, "GetForegroundWindow")
	bind(libUser32, &TrackMouseEvent, "TrackMouseEvent")
	bind(libUser32, &SendMessageW, "SendMessageW")
	bind(libUser32, &IsZoomed, "IsZoomed")
	bind(libUser32, &SetForegroundWindow, "SetForegroundWindow")
	bind(libUser32, &SetWindowPos, "SetWindowPos")
	bind(libUser32, &GetCursorPos, "GetCursorPos")
	bind(libUser32, &LoadIconW, "LoadIconW")
	bind(libUser32, &CreatePopupMenu, "CreatePopupMenu")
	bind(libUser32, &AppendMenuW, "AppendMenuW")
	bind(libUser32, &TrackPopupMenu, "TrackPopupMenu")
	bind(libUser32, &DestroyMenu, "DestroyMenu")
	bind(libUser32, &RegisterWindowMessageW, "RegisterWindowMessageW")
	bind(libUser32, &IsWindow, "IsWindow")
	bind(libUser32, &IsWindowVisible, "IsWindowVisible")
	bind(libUser32, &IsIconic, "IsIconic")
	bind(libUser32, &SetActiveWindow, "SetActiveWindow")
	bindOptional(libUser32, &GetWindowLongPtrW, "GetWindowLongPtrW")
	bindOptional(libUser32, &SetWindowLongPtrW, "SetWindowLongPtrW")
	bindOptional(libUser32, &MonitorFromWindow, "MonitorFromWindow")
	bindOptional(libUser32, &GetMonitorInfoW, "GetMonitorInfoW")
	// DrawTextW 属于 user32（不是 gdi32）
	bind(libUser32, &DrawTextW, "DrawTextW")
	bindOptional(libUser32, &SetProcessDpiAwarenessContext, "SetProcessDpiAwarenessContext")
	bindOptional(libUser32, &SetProcessDPIAware, "SetProcessDPIAware")
	bindOptional(libUser32, &GetDpiForWindow, "GetDpiForWindow")

	bind(libKernel32, &GetModuleHandleW, "GetModuleHandleW")
	bind(libKernel32, &GetLastError, "GetLastError")
	// 剪贴板（clipboard.go）
	bind(libUser32, &OpenClipboard, "OpenClipboard")
	bind(libUser32, &CloseClipboard, "CloseClipboard")
	bind(libUser32, &EmptyClipboard, "EmptyClipboard")
	bind(libUser32, &SetClipboardData, "SetClipboardData")
	bind(libUser32, &GetClipboardData, "GetClipboardData")
	bind(libUser32, &IsClipboardFormatAvailable, "IsClipboardFormatAvailable")
	bind(libKernel32, &GlobalAlloc, "GlobalAlloc")
	bind(libKernel32, &GlobalLock, "GlobalLock")
	bind(libKernel32, &GlobalUnlock, "GlobalUnlock")
	bind(libKernel32, &GlobalFree, "GlobalFree")

	bind(libGdi32, &ChoosePixelFormat, "ChoosePixelFormat")
	bind(libGdi32, &SetPixelFormat, "SetPixelFormat")
	bind(libGdi32, &DescribePixelFormat, "DescribePixelFormat")
	bind(libGdi32, &SwapBuffers, "SwapBuffers")
	bind(libGdi32, &CreateFontW, "CreateFontW")
	bind(libGdi32, &SelectObject, "SelectObject")
	bind(libGdi32, &DeleteObject, "DeleteObject")
	bind(libGdi32, &CreateSolidBrush, "CreateSolidBrush")
	bind(libGdi32, &SetBkMode, "SetBkMode")
	bind(libGdi32, &SetTextColor, "SetTextColor")
	bind(libGdi32, &GetGlyphOutlineW, "GetGlyphOutlineW")
	bind(libGdi32, &GetTextMetricsW, "GetTextMetricsW")
	bind(libGdi32, &CreateCompatibleDC, "CreateCompatibleDC")
	bind(libGdi32, &DeleteDC, "DeleteDC")
	bind(libGdi32, &GetDeviceCaps, "GetDeviceCaps")
	bind(libGdi32, &CreateDIBSection, "CreateDIBSection")
	bind(libGdi32, &SetBkColor, "SetBkColor")
	bind(libGdi32, &GdiFlush, "GdiFlush")
	bind(libGdi32, &GetTextExtentPoint32W, "GetTextExtentPoint32W")

	bind(libOpengl32, &wglCreateContext, "wglCreateContext")
	bind(libOpengl32, &wglMakeCurrent, "wglMakeCurrent")
	bind(libOpengl32, &wglDeleteContext, "wglDeleteContext")
	bind(libOpengl32, &wglGetProcAddress, "wglGetProcAddress")
	bind(libOpengl32, &wglGetCurrentContext, "wglGetCurrentContext")

	bind(libImm32, &ImmGetContext, "ImmGetContext")
	bind(libImm32, &ImmReleaseContext, "ImmReleaseContext")
	bind(libImm32, &ImmGetCompositionStringW, "ImmGetCompositionStringW")
	bind(libImm32, &ImmSetCompositionWindow, "ImmSetCompositionWindow")
	bind(libImm32, &ImmSetCandidateWindow, "ImmSetCandidateWindow")

	bind(libComdlg32, &GetOpenFileNameW, "GetOpenFileNameW")
	bind(libComdlg32, &GetSaveFileNameW, "GetSaveFileNameW")

	bind(libShell32, &Shell_NotifyIconW, "Shell_NotifyIconW")
	bind(libShell32, &SHBrowseForFolderW, "SHBrowseForFolderW")
	bind(libShell32, &SHGetPathFromIDListW, "SHGetPathFromIDListW")
	bind(libShell32, &ShellExecuteW, "ShellExecuteW")
	// 文件拖入（drop.go）
	bind(libShell32, &DragAcceptFiles, "DragAcceptFiles")
	bind(libShell32, &DragQueryFileW, "DragQueryFileW")
	bind(libShell32, &DragQueryPoint, "DragQueryPoint")
	bind(libShell32, &DragFinish, "DragFinish")
	// 回收站删除（system.go）
	bindOptional(libShell32, &SHFileOperationW, "SHFileOperationW")

	// 全局热键（system.go）
	bind(libUser32, &RegisterHotKey, "RegisterHotKey")
	bind(libUser32, &UnregisterHotKey, "UnregisterHotKey")
	// 阻止休眠 / 屏幕常亮（system.go）
	bind(libKernel32, &SetThreadExecutionState, "SetThreadExecutionState")
	// 系统深浅色检测（system.go）
	bind(libAdvapi32, &RegOpenKeyExW, "RegOpenKeyExW")
	bind(libAdvapi32, &RegQueryValueExW, "RegQueryValueExW")
	bind(libAdvapi32, &RegCloseKey, "RegCloseKey")

	bind(libOle32, &CoTaskMemFree, "CoTaskMemFree")

	bind(libUser32, &SetWindowRgn, "SetWindowRgn")
	bind(libGdi32, &CreateRoundRectRgn, "CreateRoundRectRgn")
	if libDwmapi != 0 {
		bind(libDwmapi, &DwmSetWindowAttribute, "DwmSetWindowAttribute")
		bind(libDwmapi, &DwmExtendFrameIntoClientArea, "DwmExtendFrameIntoClientArea")
	}
}

func bindOptional(lib uintptr, fn any, name string) {
	_ = name
	purego.RegisterLibFunc(fn, lib, name)
}
