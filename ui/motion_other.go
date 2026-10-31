//go:build !windows && !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// platformReducedMotionImpl 非 Windows 桌面端暂不探测系统动画设置，默认保留动画。
func platformReducedMotionImpl() bool { return false }
