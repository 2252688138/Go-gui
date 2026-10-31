//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"syscall/js"

	"gogui/internal/text"
)

// 浏览器端启动自举：先把字体取回来再建窗口。
//
// 为什么要放在 ui 包里：浏览器没有系统字体，必须 fetch 同目录的 font.ttf 后注册，
// 这一步是**平台细节**。下沉到这里之后，业务代码（cmd/gallery/main.go）
// 在所有平台都只有一个 main()、一份 opts，不需要为 Web 单独写启动文件。
//
// 约定：页面同目录放 font.ttf（必需）+ font-bold.ttf（可选粗体）。

// 启动阶段（写到 window.__goguiStage，供页面加载动画与冒烟测试读取）
const (
	stageStart = "start"
	stageFont  = "font"
	stagePanic = "panic"
)

func setStage(s string) {
	if g := js.Global(); !g.IsUndefined() {
		g.Set("__goguiStage", s)
	}
}

// prepareRun 在任何平台上都在 newApp 之前调用；Web 端在这里加载字体。
func prepareRun(opts *Options) {
	setStage(stageStart)

	if !fetchFont("font.ttf", "GoGUIWeb", false) {
		showPageError("字体加载失败：请把 font.ttf 放到 index.html 同目录")
		setStage("nofont")
		return
	}
	// 粗体可选：缺失时退回常规字重，不报错
	_ = fetchFont("font-bold.ttf", "GoGUIWeb", true)

	// 浏览器端只能用注册进来的字体
	opts.FontFamily = "GoGUIWeb"
	setStage(stageFont)
}

// fetchFont 同步等待 fetch 完成（阻塞在 channel 上，Go 的 wasm 运行时会把
// 控制权交回事件循环，因此回调能正常执行）。
func fetchFont(url, name string, bold bool) bool {
	done := make(chan bool, 1)

	onBody := js.FuncOf(func(this js.Value, args []js.Value) any {
		u8 := js.Global().Get("Uint8Array").New(args[0])
		buf := make([]byte, u8.Length())
		js.CopyBytesToGo(buf, u8)
		if bold {
			text.RegisterFont(name+"/Bold", buf)
		} else {
			text.RegisterFont(name, buf)
		}
		done <- true
		return nil
	})
	onResp := js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			done <- false
			return nil
		}
		resp.Call("arrayBuffer").Call("then", onBody)
		return nil
	})
	onErr := js.FuncOf(func(this js.Value, args []js.Value) any {
		done <- false
		return nil
	})

	js.Global().Call("fetch", url).Call("then", onResp).Call("catch", onErr)
	return <-done
}

// showPageError 把错误写到页面上的 #err 浮层（同时打 console）。
// 同时写到 window.__goguiPanic，页面自身脚本覆盖 #err 时仍能查到原始信息。
func showPageError(msg string) {
	doc := js.Global().Get("document")
	js.Global().Set("__goguiPanic", msg)
	if el := doc.Call("getElementById", "err"); !el.IsNull() && !el.IsUndefined() {
		el.Set("textContent", msg)
		el.Get("style").Set("display", "block")
	}
	js.Global().Get("console").Call("error", msg)
}

// reportPanic Web 端把 panic 显示到页面上，避免只剩一片黑屏。
func reportPanic(msg string) {
	setStage(stagePanic)
	showPageError(msg)
}
