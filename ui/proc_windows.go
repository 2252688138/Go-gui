//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"os/exec"
	"syscall"
)

// hideConsole 隐藏子进程控制台窗口（发布构建下不闪黑框）。
// 非 Windows 平台见 proc_unix.go（空实现：Unix 没有「控制台窗口」概念）。
func hideConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
