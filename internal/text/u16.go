//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import "syscall"

func utf16Ptr(s string) (*uint16, error) { return syscall.UTF16PtrFromString(s) }
