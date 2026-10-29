package ui

// 轻提示（Toast）：显示一段时间后自动隐藏，默认 2 秒。
//
// 用法（ToastLayer 放在 Stack 里，AbsBottom 定位）：
//
//	ui.ToastShow("已保存")          // 默认 ToastDefaultSeconds 秒后自动隐藏
//	ui.ToastShowFor("导出中…", 3)   // 自定义秒数
//	ui.ToastHide()                  // 立即隐藏
//
//	ui.Stack(body, ui.ToastLayer())
//
// Toast(msg) 是纯渲染组件（不计时），需要自动隐藏时用 ToastLayer。

// ToastDefaultSeconds 轻提示默认显示时长（秒）。
const ToastDefaultSeconds float32 = 2

// toastState 当前轻提示状态（UI 线程独占，无需加锁）。
var toastState struct {
	msg string
	at  float32 // 本次显示的开始时刻（AnimTime 秒）
	dur float32 // 本次显示时长（秒）
}

// ToastShow 显示一条轻提示，默认 ToastDefaultSeconds 秒后自动隐藏；msg 为空等同 ToastHide。
func ToastShow(msg string) { ToastShowFor(msg, ToastDefaultSeconds) }

// ToastShowFor 按指定秒数显示轻提示（seconds <= 0 时用默认时长）。
// 重复调用会重置计时，因此同一文案可连续触发。
func ToastShowFor(msg string, seconds float32) {
	if msg == "" {
		ToastHide()
		return
	}
	if seconds <= 0 {
		seconds = ToastDefaultSeconds
	}
	toastState.msg = msg
	toastState.at = AnimTime()
	toastState.dur = seconds
	RequestRedraw()
}

// ToastHide 立即隐藏轻提示。
func ToastHide() {
	if toastState.msg == "" {
		return
	}
	toastState.msg = ""
	RequestRedraw()
}

// ToastVisible 当前是否有轻提示在显示（含尚未被下一帧绘制成空的过期态）。
func ToastVisible() bool {
	return toastState.msg != "" && AnimTime()-toastState.at < toastState.dur
}

// ToastLayer 渲染当前轻提示；到点后自动隐藏并返回 nil。
// 显示期间每帧请求重绘，保证计时到点即消失（与 Markdown 复制提示同一套做法）。
func ToastLayer() *Node {
	msg := toastState.msg
	if msg == "" {
		return nil
	}
	if AnimTime()-toastState.at >= toastState.dur {
		toastState.msg = ""
		return nil
	}
	RequestRedraw()
	return Toast(msg)
}
