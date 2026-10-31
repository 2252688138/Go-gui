//go:build !windows && (!(linux || freebsd) || android)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// ResumeMediaAll 最小化恢复后重建解码器。
// 非 Windows 端暂无解码后端（见 media_stub.go），这里保持同名空实现，
// 让 app.go 的恢复逻辑与 Windows 端共用同一份代码。
func ResumeMediaAll() {}
