//go:build windows

package text

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// platformMapFontFile 以内存映射方式读取字体文件。
// 字体文件（如 msyh.ttc 约 20MB）用 os.ReadFile 读入会全部计入 Go 堆并长期常驻；
// 映射后数据由系统页缓存支撑，按需调入、可回收，也不占 Go 堆。
// 注意：返回的切片要在进程生命周期内保持有效（字体对象一直被缓存），故不解除映射。
func platformMapFontFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() <= 0 {
		return os.ReadFile(path)
	}
	h, err := windows.CreateFileMapping(windows.Handle(f.Fd()), nil, windows.PAGE_READONLY, 0, 0, nil)
	if err != nil || h == 0 {
		return os.ReadFile(path)
	}
	defer windows.CloseHandle(h)
	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil || addr == 0 {
		return os.ReadFile(path)
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(addr)), int(st.Size())), nil
}

// platformFontCandidates 桌面端直接用系统字体目录，无需额外候选。
func platformFontCandidates(faceName string, weight int32) []string { return nil }
