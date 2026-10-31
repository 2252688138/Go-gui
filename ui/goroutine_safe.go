// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"runtime/debug"
)

// goSafe 启动一个受保护的后台 goroutine。
//
// Go 里任何一个 goroutine panic 都会杀掉整个进程（runtime 打印全部 goroutine
// 后以 exit status 2 退出，且跑不进 Run 的 recover）。库内部起的轮询 / 解码 /
// 定时器 goroutine 因此必须自带 recover，否则一个偶发 nil 就让整个界面消失。
func goSafe(tag string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				rep := ReportCrash(fmt.Sprintf("[%s] %v", tag, r), stack)
				reportPanic(fmt.Sprintf("后台 goroutine panic（%s）: %v\n%s\n日志: %s", tag, r, stack, rep.File))
			}
		}()
		fn()
	}()
}

// recoverReport 供 goroutine 首行 defer 使用：
//
//	go func() { defer recoverReport("clipboard-watch"); ... }()
func recoverReport(tag string) {
	if r := recover(); r != nil {
		stack := debug.Stack()
		rep := ReportCrash(fmt.Sprintf("[%s] %v", tag, r), stack)
		reportPanic(fmt.Sprintf("后台 goroutine panic（%s）: %v\n%s\n日志: %s", tag, r, stack, rep.File))
	}
}

// safeCall 同步执行回调并兜住 panic（用于定时器等非自建 goroutine 的入口）。
func safeCall(tag string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			rep := ReportCrash(fmt.Sprintf("[%s] %v", tag, r), stack)
			reportPanic(fmt.Sprintf("后台回调 panic（%s）: %v\n%s\n日志: %s", tag, r, stack, rep.File))
		}
	}()
	fn()
}
