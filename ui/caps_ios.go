//go:build ios

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// platformCaps iOS 端能力表。
//
// 一期只报告「确定可用」的能力：触摸（UIKit 手势）。
// 剪贴板 / 系统深色模式需要额外接入 UIPasteboard / traitCollection，
// 媒体与内嵌浏览器需要 AVFoundation / WKWebView，均列入后续阶段。
func platformCaps() []Cap {
	return []Cap{
		CapTouch,
	}
}
