//go:build !windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "os/exec"

// hideConsole 非 Windows 平台无需隐藏控制台窗口（Unix 进程没有独立控制台），
// 保留同名空实现，让媒体层代码保持单一来源、无平台分支。
func hideConsole(cmd *exec.Cmd) {}
