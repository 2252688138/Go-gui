//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"errors"
	"syscall"
	"unsafe"
)

// GDI 打印：PrintDlgW 选打印机 → StartDocW/StartPage → 回调里用 GDI 绘制 → EndDoc。
// 回调拿到的 dc 是打印机设备上下文，dpiX/dpiY 为打印机分辨率，
// 页面尺寸 pageW/pageH 单位为 0.01 毫米（与 DEVMODE 一致）。

var (
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")

	procPrintDlgW        = comdlg32.NewProc("PrintDlgW")
	procStartDocW        = gdi32.NewProc("StartDocW")
	procStartPage        = gdi32.NewProc("StartPage")
	procEndPage          = gdi32.NewProc("EndPage")
	procEndDoc           = gdi32.NewProc("EndDoc")
	procAbortDoc         = gdi32.NewProc("AbortDoc")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procTextOutW         = gdi32.NewProc("TextOutW")
	procMoveToEx         = gdi32.NewProc("MoveToEx")
	procLineTo           = gdi32.NewProc("LineTo")
	procRectangle        = gdi32.NewProc("Rectangle")
	procEllipse          = gdi32.NewProc("Ellipse")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procSetTextAlign     = gdi32.NewProc("SetTextAlign")
	procGetDeviceCaps    = gdi32.NewProc("GetDeviceCaps")
)

const (
	pdReturnDC         = 0x00000100
	pdNoWarning        = 0x00000080
	pdReturnDefault    = 0x00000400
	pdUseDevModeCopies = 0x00040000
	psSolid            = 0
	transparentBk      = 1
	logPixelsX         = 88
	logPixelsY         = 90
	horzRes            = 8
	vertRes            = 10
	physWidth          = 110
	physHeight         = 111
)

type docInfoW struct {
	CbSize       int32
	LpszDocName  *uint16
	LpszOutput   *uint16
	LpszDatatype *uint16
	FwType       uint32
}

type printDlgW struct {
	LStructSize    uint32
	HwndOwner      uintptr
	HDevMode       uintptr
	HDevNames      uintptr
	HDC            uintptr
	Flags          uint32
	NFromPage      uint16
	NToPage        uint16
	NMinPage       uint16
	NMaxPage       uint16
	NCopies        uint16
	HInstance      uintptr
	LCustData      uintptr
	LpfnHook       uintptr
	LpTemplateName *uint16
	DwResultAction uint32
}

var nullBrush = ^uintptr(0)

// PrintInfo 一次打印任务的输出信息。
type PrintInfo struct {
	DPIx, DPIy   int
	PageW, PageH int // 0.01mm
}

// PrintDialog 弹出打印对话框并打印 pages 页。
// draw 返回 false 可中止后续页。用户取消返回 (false, nil)。
func PrintDialog(owner uintptr, title string, pages int, draw func(dc uintptr, page int, info PrintInfo) bool) (bool, error) {
	pd := printDlgW{
		LStructSize: uint32(unsafe.Sizeof(printDlgW{})),
		HwndOwner:   owner,
		Flags:       pdReturnDC | pdNoWarning | pdUseDevModeCopies,
		NMinPage:    1,
		NMaxPage:    uint16(maxInt(pages, 1)),
		NFromPage:   1,
		NToPage:     uint16(maxInt(pages, 1)),
	}
	r, _, _ := procPrintDlgW.Call(uintptr(unsafe.Pointer(&pd)))
	if r == 0 {
		return false, nil
	}
	if pd.HDC == 0 {
		return false, errors.New("win32: 打印对话框未返回 DC")
	}
	defer procDeleteDC.Call(pd.HDC)

	info := PrintInfo{}
	if v, _, _ := procGetDeviceCaps.Call(pd.HDC, logPixelsX); v != 0 {
		info.DPIx = int(v)
	}
	if v, _, _ := procGetDeviceCaps.Call(pd.HDC, logPixelsY); v != 0 {
		info.DPIy = int(v)
	}
	if v, _, _ := procGetDeviceCaps.Call(pd.HDC, physWidth); v != 0 {
		info.PageW = int(v)
	}
	if v, _, _ := procGetDeviceCaps.Call(pd.HDC, physHeight); v != 0 {
		info.PageH = int(v)
	}
	if info.DPIx <= 0 {
		info.DPIx = 300
	}
	if info.DPIy <= 0 {
		info.DPIy = 300
	}

	t, _ := syscall.UTF16PtrFromString(title)
	di := docInfoW{CbSize: int32(unsafe.Sizeof(docInfoW{})), LpszDocName: t}
	if r, _, err := procStartDocW.Call(pd.HDC, uintptr(unsafe.Pointer(&di))); int32(r) <= 0 {
		return false, errors.New("win32: StartDoc 失败: " + errText(err))
	}
	defer procEndDoc.Call(pd.HDC)
	for p := 0; p < pages; p++ {
		if r, _, _ := procStartPage.Call(pd.HDC); int32(r) <= 0 {
			procAbortDoc.Call(pd.HDC)
			return false, errors.New("win32: StartPage 失败")
		}
		ok := true
		if draw != nil {
			ok = draw(pd.HDC, p, info)
		}
		procEndPage.Call(pd.HDC)
		if !ok {
			procAbortDoc.Call(pd.HDC)
			return false, nil
		}
	}
	return true, nil
}

// PagePixels 按打印机 DPI 换算页面像素尺寸（GDI 坐标单位）。
func PagePixels(info PrintInfo) (w, h int) {
	w = int(float32(info.PageW) / 2540 * float32(info.DPIx))
	h = int(float32(info.PageH) / 2540 * float32(info.DPIy))
	if w <= 0 {
		w = 2480
	}
	if h <= 0 {
		h = 3508
	}
	return w, h
}

// ---- GDI 绘制原语（打印回调内使用，坐标单位 = 设备像素）----

// PrintText 在 DC 上绘制文本（坐标像素，fontPx 为字体像素高）。
func PrintText(dc uintptr, x, y, fontPx float32, text, rgb string) {
	r, g, b := hexRGB(rgb)
	face := mustUTF16("Microsoft YaHei")
	font, _, _ := procCreateFontW.Call(
		uintptr(int32(-fontPx)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&face[0])))
	old, _, _ := procSelectObject.Call(dc, font)
	defer func() {
		procSelectObject.Call(dc, old)
		procDeleteObject.Call(font)
	}()
	procSetBkMode.Call(dc, transparentBk)
	procSetTextColor.Call(dc, uintptr(rgbRef(r, g, b)))
	p, _ := syscall.UTF16FromString(text)
	procTextOutW.Call(dc, uintptr(int32(x)), uintptr(int32(y)), uintptr(unsafe.Pointer(&p[0])), uintptr(len(p)-1))
}

// PrintRect 画矩形（fill 为空表示不填充，strokeW 为 0 表示不描边；单位像素）。
func PrintRect(dc uintptr, x, y, w, h, strokeW float32, strokeRGB, fillRGB string) {
	var brush, pen uintptr
	if fillRGB != "" {
		r, g, b := hexRGB(fillRGB)
		brush, _, _ = procCreateSolidBrush.Call(uintptr(rgbRef(r, g, b)))
	} else {
		brush = nullBrush
	}
	if strokeW > 0 {
		r, g, b := hexRGB(strokeRGB)
		pen, _, _ = procCreatePen.Call(psSolid, uintptr(int32(strokeW)), uintptr(rgbRef(r, g, b)))
	}
	oldB, _, _ := procSelectObject.Call(dc, brush)
	oldP, _, _ := procSelectObject.Call(dc, pen)
	procRectangle.Call(dc, uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(x+w)), uintptr(int32(y+h)))
	procSelectObject.Call(dc, oldB)
	procSelectObject.Call(dc, oldP)
	if brush != nullBrush {
		procDeleteObject.Call(brush)
	}
	if pen != 0 {
		procDeleteObject.Call(pen)
	}
}

// PrintLine 画直线（单位像素）。
func PrintLine(dc uintptr, x1, y1, x2, y2, width float32, rgb string) {
	r, g, b := hexRGB(rgb)
	pen, _, _ := procCreatePen.Call(psSolid, uintptr(int32(width)), uintptr(rgbRef(r, g, b)))
	old, _, _ := procSelectObject.Call(dc, pen)
	procMoveToEx.Call(dc, uintptr(int32(x1)), uintptr(int32(y1)), 0)
	procLineTo.Call(dc, uintptr(int32(x2)), uintptr(int32(y2)))
	procSelectObject.Call(dc, old)
	procDeleteObject.Call(pen)
}

func hexRGB(s string) (int, int, int) {
	if len(s) == 7 && s[0] == '#' {
		v := 0
		for i := 1; i < 7; i++ {
			c := s[i]
			v <<= 4
			switch {
			case c >= '0' && c <= '9':
				v |= int(c - '0')
			case c >= 'a' && c <= 'f':
				v |= int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				v |= int(c-'A') + 10
			}
		}
		return (v >> 16) & 0xff, (v >> 8) & 0xff, v & 0xff
	}
	return 0, 0, 0
}

// rgbRef 转为 Win32 COLORREF（0x00BBGGRR）。
func rgbRef(r, g, b int) uint32 {
	return uint32(b)<<16 | uint32(g)<<8 | uint32(r)
}

func mustUTF16(s string) []uint16 {
	p, _ := syscall.UTF16FromString(s)
	return p
}

func errText(err error) string {
	if err == nil {
		return "unknown"
	}
	return err.Error()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
