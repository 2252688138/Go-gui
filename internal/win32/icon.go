//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"encoding/binary"
	"sync"
	"unsafe"
)

// 窗口 / 任务栏图标：直接从 .ico 字节设置，无需落地临时文件。
//
// ICO 里每个尺寸都是一段独立的图标资源：这里解析目录，挑最大的做 ICON_BIG、
// 最小的做 ICON_SMALL，再同步窗口类图标（GCLP_HICON / GCLP_HICONSM），
// 这样标题栏、Alt+Tab、任务栏三处都拿到自绘图标。

const (
	wmSetIcon   = 0x0080
	iconSmall   = 0
	iconBig     = 1
	gclpHicon   = -14
	gclpHiconSm = -34
	// ICON_RES_LANG_NEUTRAL + PNG 资源版本（Vista+）
	iconResVersion = 0x00030000
)

var (
	iconOnce                 sync.Once
	createIconFromResourceEx func(presbits uintptr, presbytes uint32, fIcon int32, dwVer uint32, cx, cy int32, flags uint32) uintptr
	setClassLongPtrW         func(hwnd uintptr, index int32, value uintptr) uintptr
)

func loadIconAPI() {
	iconOnce.Do(func() {
		bind(libUser32, &createIconFromResourceEx, "CreateIconFromResourceEx")
		bind(libUser32, &setClassLongPtrW, "SetClassLongPtrW")
	})
}

// icoImage 一个尺寸的图标资源。
type icoImage struct {
	size int
	data []byte
}

// parseICO 解析 ICO 目录（支持 PNG 压缩条目与 BMP 条目）。
func parseICO(ico []byte) []icoImage {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[0:]) != 0 || binary.LittleEndian.Uint16(ico[2:]) != 1 {
		return nil
	}
	n := int(binary.LittleEndian.Uint16(ico[4:]))
	if n <= 0 || len(ico) < 6+16*n {
		return nil
	}
	out := make([]icoImage, 0, n)
	for i := 0; i < n; i++ {
		e := ico[6+16*i:]
		w := int(e[0])
		if w == 0 {
			w = 256
		}
		sz := int(binary.LittleEndian.Uint32(e[8:]))
		off := int(binary.LittleEndian.Uint32(e[12:]))
		if sz <= 0 || off <= 0 || off+sz > len(ico) {
			continue
		}
		out = append(out, icoImage{size: w, data: ico[off : off+sz]})
	}
	return out
}

// iconFromImage 把单个图标资源转成 HICON（失败返回 0）。
func iconFromImage(img icoImage) uintptr {
	if len(img.data) == 0 || createIconFromResourceEx == nil {
		return 0
	}
	return createIconFromResourceEx(
		uintptr(unsafe.Pointer(&img.data[0])),
		uint32(len(img.data)),
		1, // fIcon
		iconResVersion,
		0, 0, 0,
	)
}

// SetWindowIcon 用 .ico 字节给窗口设置图标（大小图标 + 窗口类图标）。
// 返回是否至少设置成功一项。
func SetWindowIcon(hwnd uintptr, ico []byte) bool {
	loadIconAPI()
	imgs := parseICO(ico)
	if hwnd == 0 || len(imgs) == 0 {
		return false
	}
	big := imgs[0]
	small := imgs[0]
	for _, im := range imgs {
		if im.size > big.size {
			big = im
		}
		if im.size < small.size {
			small = im
		}
	}
	ok := false
	if h := iconFromImage(big); h != 0 {
		SendMessageW(hwnd, wmSetIcon, iconBig, h)
		if setClassLongPtrW != nil {
			setClassLongPtrW(hwnd, gclpHicon, h)
		}
		ok = true
	}
	if h := iconFromImage(small); h != 0 {
		SendMessageW(hwnd, wmSetIcon, iconSmall, h)
		if setClassLongPtrW != nil {
			setClassLongPtrW(hwnd, gclpHiconSm, h)
		}
		ok = true
	}
	return ok
}

// SetIcon 给当前窗口设置图标（.ico 字节）。
func (w *Window) SetIcon(ico []byte) bool {
	if w == nil || w.Hwnd == 0 {
		return false
	}
	return SetWindowIcon(w.Hwnd, ico)
}
