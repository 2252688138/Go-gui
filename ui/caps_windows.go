//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// platformCaps Windows 端能力表：全功能（除触摸手势外）。
func platformCaps() []Cap {
	return []Cap{
		CapVideo,
		CapAudio,
		CapWebView,
		CapTray,
		CapClipboard,
		CapDialogs,
		CapHotkey,
		CapDragDrop,
		CapSystemDark,
		CapKeepAwake,
		CapRecycleBin,
		CapPrint,
		CapScreenCapture,
		CapClipboardWatch,
		CapMultiMonitor,
		CapSingleInstance,
		CapModernNotify,
	}
}
