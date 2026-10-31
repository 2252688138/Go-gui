//go:build !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"os"
)

// prepareRun 桌面 / 移动端启动前没有额外准备：字体直接用系统字体目录（见 internal/text/fontplat_*）。
func prepareRun(opts *Options) {
	// 未捕获 panic / fatal error 写盘：否则后台 goroutine 崩了只剩“exit status 2”。
	enableCrashLog()
	startCPUProfileIfNeeded()
}

// reportPanic 输出 Run 内部捕获到的 panic。
//
// 旧实现是空实现：runLoop 里的 panic 被 recover 后既不打印也不上报，
// 进程「静默退出（exit 0）」，使用者只看到窗口消失 / 黑屏，无法定位。
// 现在统一写到 stderr（桌面端带控制台或重定向日志时可见），
// 消息里已包含 panic 值与 goroutine 堆栈（见 Run 里的 debug.Stack）。
func reportPanic(msg string) {
	fmt.Fprintf(os.Stderr, "[gogui] 运行期 panic（界面已停止）：\n%s\n", msg)
}
