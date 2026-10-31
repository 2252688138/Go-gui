//go:build !windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// applyUpdatePlatform 非 Windows 直接交给通用逻辑（Rename 覆盖）。
func applyUpdatePlatform(self, newExe string) (bool, error) { return false, nil }
