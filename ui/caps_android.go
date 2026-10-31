//go:build android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// platformCaps Android 端能力表。
//
// 与桌面的差别：没有托盘 / 全局热键 / 文件对话框（需 Java 侧 Intent 桥）/
// 文件拖入；媒体解码与内嵌浏览器需要额外接入（MediaCodec / 系统 WebView）。
// 一期只报告「确定可用」的能力，避免业务误判。
func platformCaps() []Cap {
	return []Cap{
		CapTouch,
	}
}
