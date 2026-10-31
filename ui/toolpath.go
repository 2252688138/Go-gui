// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"os"
	"os/exec"
)

// toolPath 定位外部工具（ffmpeg / ffprobe 等）：显式路径 > PATH。
//
// 媒体层（ui/media.go）与各平台能力探测共用这一份实现，
// 以前 Windows 与类 Unix 各有一份，Linux 接入 ffmpeg 路径后会重复声明。
func toolPath(explicit, name string) string {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
		return ""
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	// Windows 下 PATH 里通常是不带扩展名的名字，LookPath 已处理 PATHEXT；
	// 这里额外兜一层，兼容个别被裁剪的环境。
	if p, err := exec.LookPath(name + ".exe"); err == nil {
		return p
	}
	return ""
}
