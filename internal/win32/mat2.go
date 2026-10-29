//go:build windows

package win32

// IdentityMAT2 返回单位变换矩阵（不可变，供字体光栅化使用）。
func IdentityMAT2() MAT2 { return identityMAT2() }
