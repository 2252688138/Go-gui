//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// platformCaps 浏览器端能力表。
//
// 与桌面端的差别：没有系统托盘 / 全局热键 / 文件拖入路径（DOM 只给文件名）、
// 剪贴板需要用户手势授权。视频与音频交给浏览器的 <video> 原生解码，
// 文件选择用 <input type=file> 实现（异步），内嵌浏览器用 <iframe> 实现。
func platformCaps() []Cap {
	return []Cap{
		CapVideo,
		CapAudio,
		CapWebView,
		CapClipboard,
		CapDialogs,
		CapTouch,
		CapSystemDark,
		CapKeepAwake,
		CapModernNotify,
	}
}
