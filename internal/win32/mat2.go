//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

// IdentityMAT2 返回单位变换矩阵（不可变，供字体光栅化使用）。
func IdentityMAT2() MAT2 { return identityMAT2() }
