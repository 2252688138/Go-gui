package text

import "syscall"

func utf16Ptr(s string) (*uint16, error) { return syscall.UTF16PtrFromString(s) }
