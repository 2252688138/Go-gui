//go:build !windows && !js && (!(linux || freebsd) || android)

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"gogui/internal/geom"
)

// 视频绘制垫片：macOS / iOS / Android 等平台尚无解码后端，
// 保留与 Windows 一致的符号（videoTexture / paintVideo），
// 让同一套 UI 代码原样编译，界面走占位分支。Linux 已接入真实实现（video_paint.go）。

// videoTexture 视频帧纹理（本平台不使用，仅占位）。
type videoTexture struct{}

// paintVideo 视频画面绘制：本平台无解码后端，跳过（style 背景照常绘制）。
func (a *App) paintVideo(n *Node, r geom.Rect, st *Style) {}
