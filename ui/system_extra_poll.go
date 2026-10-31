// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "time"

// watchClipboardPolling 跨平台兜底：轮询剪贴板文本变化。
func watchClipboardPolling(fn func(ClipboardInfo)) (stop func()) {
	if fn == nil {
		return func() {}
	}
	quit := make(chan struct{})
	go func() {
		defer recoverReport("clipboard-poll")
		last := ClipboardText()
		tk := time.NewTicker(700 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-quit:
				return
			case <-tk.C:
				cur := ClipboardText()
				if cur == last {
					continue
				}
				last = cur
				info := ClipboardInfo{Text: cur, HasText: cur != ""}
				postToUI(func() {
					fn(info)
					RequestRedraw()
				})
			}
		}
	}()
	return func() { close(quit) }
}
