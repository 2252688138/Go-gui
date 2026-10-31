//go:build linux || freebsd || openbsd || netbsd || darwin || android || ios

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"os"

	"golang.org/x/sys/unix"
)

// 类 Unix 端的字体文件读取（平台无关的挑选逻辑在 fontpick.go）。

// mmapFontFile 以只读方式映射字体文件：数据由系统页缓存支撑，按需调入、可回收，
// 不占 Go 堆（CJK 字体动辄 20MB）。失败时回退整读。
//
// 返回的切片在进程生命周期内保持有效（字体对象被缓存，不主动解除映射）。
func mmapFontFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() <= 0 {
		return os.ReadFile(path)
	}
	data, err := unix.Mmap(int(f.Fd()), 0, int(st.Size()), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return os.ReadFile(path)
	}
	return data, nil
}
