//go:build (linux || freebsd || openbsd || netbsd) && !android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

// platformSymbolFamily 后备（符号）字体家族名：Linux 系用 DejaVu Sans。
// DejaVu 几乎在所有发行版上随桌面环境预装，符号/箭头覆盖率足够；
// 找不到时 loadSystemFont 返回错误，主字体照常工作（只是没有后备字形）。
func platformSymbolFamily() string { return "DejaVu Sans" }
