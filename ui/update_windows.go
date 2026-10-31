//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// applyUpdatePlatform 用「等待进程退出 → 改名旧文件 → 落盘新文件 → 启动」的方式替换自身。
// 返回 (handled, err)：handled=false 时由通用逻辑处理。
func applyUpdatePlatform(self, newExe string) (bool, error) {
	old := self + ".old"
	newPath := self + ".new"
	if err := copyFile(newExe, newPath); err != nil {
		return true, err
	}
	script := fmt.Sprintf(
		`timeout /t 1 /nobreak >nul & if exist "%s" del /f /q "%s" & move /y "%s" "%s" >nul & start "" "%s"`,
		old, old, self, old, self)
	cmd := exec.Command("cmd", "/c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if err := cmd.Start(); err != nil {
		return true, err
	}
	go func() {
		// 给启动脚本一点时间，然后退出当前进程
		cmd.Wait()
	}()
	os.Exit(0)
	return true, nil
}

// copyFile 复制文件（用于把下载的新版本放到自身目录）。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = out.ReadFrom(in)
	return err
}
