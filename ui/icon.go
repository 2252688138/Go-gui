package ui

import _ "embed"

// 内置品牌图标（圆角渐变方块 + 白色 G，多尺寸 ICO，由 .zym/icongen 生成）。
// 应用既想让窗口/任务栏带图标又不想带外部文件时用它；业务可通过
// Options.Icon 传入自己的 .ico 字节覆盖。
//
//go:embed assets/icon.ico
var defaultIcon []byte

// DefaultIcon 返回内置图标的 ICO 字节（只读，勿修改）。
func DefaultIcon() []byte { return defaultIcon }

// SetWindowIcon 用 .ico 字节设置当前窗口图标（大小图标 + 窗口类图标）。
func SetWindowIcon(ico []byte) bool {
	a := currentApp.Load()
	if a == nil || a.win == nil {
		return false
	}
	return a.win.SetIcon(ico)
}
