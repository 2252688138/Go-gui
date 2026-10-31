//go:build android && cgo

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Android 应用入口：编译成 libgogui.so 后由 NativeActivity 加载。
//
// 构建（Windows + NDK）：
//
//	$env:CC = "$env:ANDROID_NDK\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android21-clang.cmd"
//	$env:CGO_ENABLED = "1"; $env:GOOS = "android"; $env:GOARCH = "arm64"
//	go build -buildmode=c-shared -o app/src/main/jniLibs/arm64-v8a/libgogui.so ./mobile/android
package main

import (
	"gogui/internal/android"
	"gogui/ui"
)

func main() {
	// 系统调 ANativeActivity_onCreate 之前，把「界面入口」交给桥接层。
	android.OnCreate = func() {
		runApp()
	}
	// main 必须常驻：真正的 UI 循环在 OnCreate 派生的 goroutine 里跑，
	// 因为窗口要等 NativeActivity 回调交付之后才存在。
	select {}
}

// runApp 启动 UI。与桌面端同一份业务代码，只换窗口后端。
func runApp() {
	_ = ui.Run(ui.Options{
		Title:  "GoGUI",
		Width:  1080,
		Height: 1920,
	}, view)
}

// view 演示界面：一个可滚动的卡片列表 + 触摸反馈。
func view() *ui.Node {
	return ui.Column(
		ui.Row(
			ui.Text("GoGUI").St(ui.St().Text(22, 700, ui.ColText)),
			ui.Spacer(),
			ui.Tag("Android", ui.ColPrimary),
		).St(ui.St().PadXY(18, 16)),
		ui.Scroll(
			ui.Column(
				ui.Card(
					ui.Title("同一份 UI 代码", "布局 / 命中 / 滚动全部复用桌面端实现"),
					ui.Text("触摸事件在后端合成为鼠标事件，组件无需感知手指还是鼠标。").
						St(ui.St().Text(13, 400, ui.ColTextDim).PadXY(0, 6)),
				).St(ui.St().PadXY(16, 16)),
				ui.Card(
					ui.Text("能力探测：HasCap(CapTouch)").St(ui.St().Text(13, 500, ui.ColText)),
					ui.Text(capLine()).St(ui.St().Text(12, 400, ui.ColTextMuted)),
				).St(ui.St().PadXY(16, 16)),
			).St(ui.St().GapOf(12).PadXY(16, 12)),
		).St(ui.St().GrowTo(1)),
	).St(ui.St().GrowTo(1).BgC(ui.ColBg))
}

// capLine 展示当前平台能力（移动端通常只有剪贴板 / 触摸）。
func capLine() string {
	var out string
	for _, c := range ui.Capabilities() {
		if out != "" {
			out += " · "
		}
		out += ui.CapName(c)
	}
	if out == "" {
		return "无扩展能力"
	}
	return out
}
