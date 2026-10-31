//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"os"
	"path/filepath"
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

// platformFontCandidates Windows 端：按已识别的家族名映射到 %WINDIR%\Fonts 下的字体文件；
// 未识别的家族返回 nil，由调用方回退 platformDefaultFontCandidates。
func platformFontCandidates(faceName string, weight int32) []string {
	return platformKnownFamily(faceName, weight)
}

// platformDefaultFontCandidates 未指定 FontFamily（或未命中已知家族）时的候选：
// 微软雅黑优先，回退 Segoe UI；粗体优先 msyhbd。
func platformDefaultFontCandidates(weight int32) []string {
	bold := weight >= 600
	dir := os.Getenv("WINDIR")
	if dir == "" {
		dir = `C:\Windows`
	}
	fonts := filepath.Join(dir, "Fonts")
	switch {
	case bold:
		return []string{filepath.Join(fonts, "msyhbd.ttc"), filepath.Join(fonts, "msyh.ttc")}
	}
	return []string{filepath.Join(fonts, "msyh.ttc"), filepath.Join(fonts, "segoeui.ttf")}
}

// platformKnownFamily 已识别的家族名 → 候选文件（未识别返回 nil，由调用方回退默认）。
//
// 保留在字体层而非 font.go，是为了让「哪些文件名叫什么名字」这类平台知识
// 不往平台无关代码里泄漏。
func platformKnownFamily(faceName string, weight int32) []string {
	bold := weight >= 600
	dir := os.Getenv("WINDIR")
	if dir == "" {
		dir = `C:\Windows`
	}
	fonts := filepath.Join(dir, "Fonts")
	name := faceName
	switch {
	case name == "" || containsAny(name, "雅黑", "YaHei", "Microsoft YaHei"):
		if bold {
			return []string{filepath.Join(fonts, "msyhbd.ttc"), filepath.Join(fonts, "msyh.ttc"), filepath.Join(fonts, "segoeuib.ttf")}
		}
		return []string{filepath.Join(fonts, "msyh.ttc"), filepath.Join(fonts, "segoeui.ttf")}
	case containsAny(name, "宋体", "SimSun"):
		return []string{filepath.Join(fonts, "simsun.ttc"), filepath.Join(fonts, "msyh.ttc")}
	case containsAny(name, "黑体", "SimHei"):
		return []string{filepath.Join(fonts, "simhei.ttf"), filepath.Join(fonts, "msyh.ttc")}
	case containsAny(name, "Symbol", "symbol"):
		return []string{filepath.Join(fonts, "seguisym.ttf"), filepath.Join(fonts, "segoeui.ttf")}
	case containsAny(name, "Emoji", "emoji"):
		return []string{filepath.Join(fonts, "seguiemj.ttf"), filepath.Join(fonts, "seguisym.ttf")}
	case containsAny(name, "Consolas", "Mono", "mono"):
		if bold {
			return []string{filepath.Join(fonts, "consolab.ttf"), filepath.Join(fonts, "consola.ttf")}
		}
		return []string{filepath.Join(fonts, "consola.ttf"), filepath.Join(fonts, "msyh.ttc")}
	case containsAny(name, "Segoe UI", "Segoe"):
		if bold {
			return []string{filepath.Join(fonts, "segoeuib.ttf")}
		}
		return []string{filepath.Join(fonts, "segoeui.ttf")}
	}
	return nil
}
