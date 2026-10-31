//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// 桌面增强能力：多显示器枚举、屏幕捕获、剪贴板监听、单实例锁、图片打印。
// 全部走 syscall 直调，无 cgo。

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shcore   = syscall.NewLazyDLL("shcore.dll")

	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procOpenClipboard       = user32.NewProc("OpenClipboard")
	procCloseClipboard      = user32.NewProc("CloseClipboard")
	procGetClipboardData    = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAv = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardSeq     = user32.NewProc("GetClipboardSequenceNumber")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procSetProcessDPIAware  = user32.NewProc("SetProcessDPIAware")

	procGetDpiForMonitor = shcore.NewProc("GetDpiForMonitor")

	procGetPixel         = gdi32.NewProc("GetPixel")
	procGetCursorPosW    = user32.NewProc("GetCursorPos")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")

	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procGetLastError = kernel32.NewProc("GetLastError")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalSize   = kernel32.NewProc("GlobalSize")
)

const (
	smCxScreen        = 0
	smCyScreen        = 1
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCxVirtualScreen = 78
	smCyVirtualScreen = 79

	cfText  = 1
	cfDIB   = 8
	cfHDROP = 15

	biRGB        = 0
	dibRGBColors = 0
	srcCopy      = 0x00CC0020
	captureBlt   = 0x40000000

	errorAlreadyExists = 183
)

// MonitorRect 显示器工作区（物理像素）。
type MonitorRect struct {
	X, Y, W, H int
}

// MonitorInfo 显示器信息。
type MonitorInfo struct {
	Index                      int
	Name                       string
	X, Y, W, H                 int
	WorkX, WorkY, WorkW, WorkH int
	Scale                      float32
	Primary                    bool
}

type rectS struct{ Left, Top, Right, Bottom int32 }

type monitorInfoExW struct {
	CbSize    uint32
	RcMonitor rectS
	RcWork    rectS
	DwFlags   uint32
	SzDevice  [32]uint16
}

// Monitors 枚举所有显示器（顺序与系统一致，主屏优先标记）。
func Monitors() []MonitorInfo {
	out := make([]MonitorInfo, 0, 2)
	cb := syscall.NewCallback(func(hMonitor, hdc, lprc uintptr, data uintptr) uintptr {
		var mi monitorInfoExW
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		r, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
		if r == 0 {
			return 1
		}
		scale := float32(1)
		if procGetDpiForMonitor.Find() == nil {
			var dx, dy uint32
			if hr, _, _ := procGetDpiForMonitor.Call(hMonitor, 0, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy))); hr == 0 && dx > 0 {
				scale = float32(dx) / 96
			}
		}
		name := ""
		for i := 0; i < len(mi.SzDevice); i++ {
			if mi.SzDevice[i] == 0 {
				break
			}
			name += string(rune(mi.SzDevice[i]))
		}
		out = append(out, MonitorInfo{
			Index: len(out), Name: name,
			X: int(mi.RcMonitor.Left), Y: int(mi.RcMonitor.Top),
			W:     int(mi.RcMonitor.Right - mi.RcMonitor.Left),
			H:     int(mi.RcMonitor.Bottom - mi.RcMonitor.Top),
			WorkX: int(mi.RcWork.Left), WorkY: int(mi.RcWork.Top),
			WorkW:   int(mi.RcWork.Right - mi.RcWork.Left),
			WorkH:   int(mi.RcWork.Bottom - mi.RcWork.Top),
			Scale:   scale,
			Primary: mi.DwFlags&1 != 0,
		})
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, cb, 0)
	// 主屏排到最前（界面里通常先展示主屏）
	for i := 1; i < len(out); i++ {
		if out[i].Primary {
			out[0], out[i] = out[i], out[0]
			for j := range out {
				out[j].Index = j
			}
			break
		}
	}
	return out
}

// VirtualScreen 返回所有显示器的并集区域。
func VirtualScreen() (x, y, w, h int) {
	gx, _, _ := procGetSystemMetrics.Call(smXVirtualScreen)
	gy, _, _ := procGetSystemMetrics.Call(smYVirtualScreen)
	gw, _, _ := procGetSystemMetrics.Call(smCxVirtualScreen)
	gh, _, _ := procGetSystemMetrics.Call(smCyVirtualScreen)
	return int(int32(gx)), int(int32(gy)), int(gw), int(gh)
}

type bitmapInfoHeader struct {
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

// CaptureScreen 截取屏幕区域（物理像素，x/y 可为负，用于多屏）。cursor 参数保留：
// 光标绘制需要额外合成，当前实现不含光标（返回的图里没有鼠标指针）。
func CaptureScreen(x, y, w, h int, cursor bool) (*image.NRGBA, error) {
	_ = cursor
	if w <= 0 || h <= 0 {
		return nil, errors.New("win32: 截屏区域为空")
	}
	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return nil, errors.New("win32: GetDC 失败")
	}
	defer procReleaseDC.Call(0, screen)
	memDC, _, _ := procCreateCompatibleDC.Call(screen)
	if memDC == 0 {
		return nil, errors.New("win32: CreateCompatibleDC 失败")
	}
	defer procDeleteDC.Call(memDC)

	var bi struct {
		Hdr    bitmapInfoHeader
		Colors [3]uint32
	}
	bi.Hdr.BiSize = uint32(unsafe.Sizeof(bi.Hdr))
	bi.Hdr.BiWidth = int32(w)
	bi.Hdr.BiHeight = -int32(h) // top-down
	bi.Hdr.BiPlanes = 1
	bi.Hdr.BiBitCount = 32
	bi.Hdr.BiCompression = biRGB
	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bi)), dibRGBColors,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return nil, errors.New("win32: CreateDIBSection 失败")
	}
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(memDC, bmp)
	defer procSelectObject.Call(memDC, old)

	r, _, err := procBitBlt.Call(memDC, 0, 0, uintptr(w), uintptr(h),
		screen, uintptr(int32(x)), uintptr(int32(y)), srcCopy|captureBlt)
	if r == 0 {
		return nil, fmt.Errorf("win32: BitBlt 失败: %v", err)
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(bits), w*h*4)
	for i := 0; i < len(src); i += 4 {
		img.Pix[i+0] = src[i+2]
		img.Pix[i+1] = src[i+1]
		img.Pix[i+2] = src[i+0]
		img.Pix[i+3] = 255
	}
	return img, nil
}

// ClipboardInfo 剪贴板快照。
type ClipboardInfo struct {
	Seq      uint32
	Text     string
	HasText  bool
	HasImage bool
	HasFiles bool
}

// ClipboardSequence 系统剪贴板序号（变化即表示内容被修改）。
func ClipboardSequence() uint32 {
	r, _, _ := procGetClipboardSeq.Call()
	return uint32(r)
}

// ClipboardSnapshot 读取当前剪贴板摘要。
func ClipboardSnapshot() ClipboardInfo {
	info := ClipboardInfo{Seq: ClipboardSequence()}
	info.HasText = clipboardFormat(cfText)
	info.HasImage = clipboardFormat(cfDIB)
	info.HasFiles = clipboardFormat(cfHDROP)
	if info.HasText {
		info.Text = GetClipboardText()
	}
	return info
}

func clipboardFormat(f uint32) bool {
	r, _, _ := procIsClipboardFormatAv.Call(uintptr(f))
	return r != 0
}

// ClipboardImage 读取剪贴板位图（CF_DIB，支持 24/32 位）。
func ClipboardImage() (*image.NRGBA, error) {
	if !clipboardFormat(cfDIB) {
		return nil, errors.New("win32: 剪贴板没有图片")
	}
	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return nil, errors.New("win32: OpenClipboard 失败")
	}
	defer procCloseClipboard.Call()
	h, _, _ := procGetClipboardData.Call(cfDIB)
	if h == 0 {
		return nil, errors.New("win32: GetClipboardData 失败")
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return nil, errors.New("win32: GlobalLock 失败")
	}
	defer procGlobalUnlock.Call(h)
	size, _, _ := procGlobalSize.Call(h)
	if size < 40 {
		return nil, errors.New("win32: DIB 数据过短")
	}
	hdr := (*bitmapInfoHeader)(unsafe.Pointer(p))
	w, h2 := int(hdr.BiWidth), int(hdr.BiHeight)
	topDown := h2 < 0
	if topDown {
		h2 = -h2
	}
	if w <= 0 || h2 <= 0 {
		return nil, errors.New("win32: DIB 尺寸非法")
	}
	bpp := int(hdr.BiBitCount)
	if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("win32: 暂不支持 %d 位 DIB", bpp)
	}
	stride := ((w*bpp + 31) / 32) * 4
	need := unsafe.Sizeof(*hdr) + uintptr(stride*h2)
	if need > size {
		return nil, errors.New("win32: DIB 数据不完整")
	}
	pix := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(p)+unsafe.Sizeof(*hdr))), stride*h2)
	img := image.NewNRGBA(image.Rect(0, 0, w, h2))
	for row := 0; row < h2; row++ {
		srcRow := row
		if !topDown {
			srcRow = h2 - 1 - row
		}
		for col := 0; col < w; col++ {
			o := srcRow*stride + col*bpp/8
			d := img.PixOffset(col, row)
			img.Pix[d+0] = pix[o+2]
			img.Pix[d+1] = pix[o+1]
			img.Pix[d+2] = pix[o+0]
			img.Pix[d+3] = 255
		}
	}
	return img, nil
}

// ClipboardPixel 取屏幕某点的颜色（吸管）。
func ClipboardPixel(x, y int) (color.RGBA, bool) { return ScreenPixelAt(x, y) }

// ScreenPixelAt 读取屏幕坐标处像素的颜色（吸管取色）。
func ScreenPixelAt(x, y int) (color.RGBA, bool) {
	dc, _, _ := procGetDC.Call(0)
	if dc == 0 {
		return color.RGBA{}, false
	}
	defer procReleaseDC.Call(0, dc)
	r, _, _ := procGetPixel.Call(dc, uintptr(int32(x)), uintptr(int32(y)))
	if uint32(r) == 0xFFFFFFFF {
		return color.RGBA{}, false
	}
	v := uint32(r)
	return color.RGBA{R: byte(v), G: byte(v >> 8), B: byte(v >> 16), A: 255}, true
}

// CursorScreenPos 鼠标指针的屏幕坐标（吸管取色跨窗口拾取用）。
func CursorScreenPos() (int, int, bool) {
	var pt POINT
	if r, _, _ := procGetCursorPosW.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		return 0, 0, false
	}
	return int(pt.X), int(pt.Y), true
}

// LeftButtonDown 鼠标左键当前是否按下（GetAsyncKeyState 的 0x8000 位）。
func LeftButtonDown() bool { return keyDown(0x01) } // VK_LBUTTON

// RightButtonDown 鼠标右键当前是否按下。
func RightButtonDown() bool { return keyDown(0x02) } // VK_RBUTTON

// KeyDown 查询某个虚拟键当前是否按下（Esc 取消拾色用）。
func KeyDown(vk int) bool { return keyDown(uintptr(vk)) }

func keyDown(vk uintptr) bool {
	r, _, _ := procGetAsyncKeyState.Call(vk)
	return int16(uint32(r)&0xFFFF) < 0
}

// ---- 单实例锁 ----

// SingleInstance 创建命名互斥体；返回 true 表示本进程是首个实例。
func SingleInstance(key string) bool {
	name, err := syscall.UTF16PtrFromString(`Local\gogui-` + key)
	if err != nil {
		return true
	}
	h, _, lerr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return true // 创建失败：放行，不阻止启动
	}
	// 必须用 CreateMutexW 的第三个返回值（Go 已代为读取 GetLastError）判断；
	// 之后再单独调 GetLastError 会被期间其它 syscall 覆盖恒为 0，导致永远判成“首个实例”。
	if e, ok := lerr.(syscall.Errno); ok && uint32(e) == errorAlreadyExists {
		return false
	}
	return true
}

// SecondInstancePing 通知已存在的实例（广播 WM_NULL 唤醒主窗口消息循环），
// 参数通过 %TEMP%/gogui-<key>.args 传递，首个实例用 PollSecondInstanceArgs 读取。
func SecondInstancePing(key string) { SecondInstancePingArgs(key, nil) }

// SecondInstancePingArgs 带参数通知首个实例。
func SecondInstancePingArgs(key string, args []string) {
	if len(args) > 0 {
		writeSecondArgs(key, args)
	}
	procPostMessageW.Call(0xFFFF, 0, 0, 0) // HWND_BROADCAST, WM_NULL
}

// secondArgsPath 二次启动参数临时文件路径。
func secondArgsPath(key string) string {
	dir := os.TempDir()
	return filepath.Join(dir, "gogui-"+key+".args")
}

func writeSecondArgs(key string, args []string) {
	_ = os.WriteFile(secondArgsPath(key), []byte(strings.Join(args, "\n")), 0o644)
}

// PollSecondInstanceArgs 读取并清除二次启动参数（首次实例每帧调用）。
func PollSecondInstanceArgs(key string) []string {
	p := secondArgsPath(key)
	raw, err := os.ReadFile(p)
	if err != nil || len(raw) == 0 {
		return nil
	}
	_ = os.Remove(p)
	var out []string
	for _, ln := range strings.Split(string(raw), "\n") {
		if ln != "" {
			out = append(out, ln)
		}
	}
	return out
}
