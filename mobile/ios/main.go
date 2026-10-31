//go:build ios && cgo

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// iOS 应用入口：编译成 libgogui.a（c-archive），由 Xcode 工程的 AppDelegate 调用。
//
// 构建（仅在 macOS + Xcode 上可用）：
//
//	GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
//	  go build -buildmode=c-archive -o ios/libgogui.a ./mobile/ios
//
// Xcode 侧需要在 main.m 里调用 Go 导出的 `StartGoUI()`（见 ios/README.md）。
package main

import (
	"gogui/ui"
)

// view 演示界面（与 android 示例同构）。
func view() *ui.Node {
	return ui.Column(
		ui.Row(
			ui.Text("GoGUI").St(ui.St().Text(22, 700, ui.ColText)),
			ui.Spacer(),
			ui.Tag("iOS", ui.ColPrimary),
		).St(ui.St().PadXY(18, 16)),
		ui.Scroll(
			ui.Column(
				ui.Card(
					ui.Title("同一份 UI 代码", "触摸事件已合成为鼠标事件"),
					ui.Text("iOS 端使用 UIKit + EAGL + GLES3，与 Android 的 EGL 实现同构。").
						St(ui.St().Text(13, 400, ui.ColTextDim).PadXY(0, 6)),
				).St(ui.St().PadXY(16, 16)),
			).St(ui.St().GapOf(12).PadXY(16, 12)),
		).St(ui.St().GrowTo(1)),
	).St(ui.St().GrowTo(1).BgC(ui.ColBg))
}

// main 由 Go 运行时在 c-archive 初始化时启动；
// 真正的界面在 Xcode 侧调用 StartGoUI 之后才创建窗口。
func main() {
	select {}
}

// StartGoUI 由 Objective-C 侧调用（cgo 会导出为同名符号）。
//
//export StartGoUI
func StartGoUI() {
	go func() {
		_ = ui.Run(ui.Options{Title: "GoGUI", Width: 390, Height: 844}, view)
	}()
}
