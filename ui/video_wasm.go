//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "gogui/internal/geom"

// 浏览器端的视频/音频解码尚未接入（桌面端走 ffmpeg / Media Foundation），
// 这里保留同名的公开类型与绘制钩子，让业务代码与组件树原样编译，
// 播放相关调用安全地什么都不做。

// VideoFit 画面填充方式。
type VideoFit uint8

const (
	VideoFitContain VideoFit = iota
	VideoFitCover
	VideoFitFill
	VideoFitNone
)

// Player 视频播放器（浏览器端占位）。
type Player struct {
	Fit    VideoFit
	Volume float32
	Loop   bool
}

// Size 视频画面像素尺寸（浏览器端无解码器，返回 0）。
func (p *Player) Size() (int, int) { return 0, 0 }

// videoTexture 视频帧纹理（浏览器端不使用）。
type videoTexture struct{}

// VideoClip 内置演示片（浏览器端不使用）。
type VideoClip struct {
	Name   string
	Width  int
	Height int
	FPS    float64
}

// VideoSource 视频源接口（浏览器端占位）。
type VideoSource interface {
	Duration() float64
}

// paintVideo 视频画面绘制：浏览器端跳过。
func (a *App) paintVideo(n *Node, r geom.Rect, st *Style) {}
