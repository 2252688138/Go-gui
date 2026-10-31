// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"
)

// 崩溃报告与日志面板。
//
//	ui.SetCrashHandler(func(rep ui.CrashReport) { upload(rep.Summary) })
//
// panic 会被捕获并写入 %AppData%/gogui/crash-*.log（含堆栈），
// CrashPanel 在界面里展示最近一次崩溃。

// CrashReport 一次崩溃的信息。
type CrashReport struct {
	Time    time.Time
	Message string
	Stack   string
	File    string
}

var (
	crashMu     sync.Mutex
	crashLast   CrashReport
	crashHook   func(CrashReport)
	crashDirSet string
)

// SetCrashHandler 注册崩溃回调（在 panic 恢复后、写盘后调用）。
func SetCrashHandler(fn func(CrashReport)) {
	crashMu.Lock()
	crashHook = fn
	crashMu.Unlock()
}

// SetCrashDir 指定崩溃日志目录（默认用户配置目录/gogui/crashes）。
func SetCrashDir(dir string) {
	crashMu.Lock()
	crashDirSet = dir
	crashMu.Unlock()
}

// CrashLast 最近一次崩溃（无则 ok=false）。
func CrashLast() (CrashReport, bool) {
	crashMu.Lock()
	defer crashMu.Unlock()
	if crashLast.Time.IsZero() {
		return CrashReport{}, false
	}
	return crashLast, true
}

// ReportCrash 记录一次崩溃（Run 的 recover 与业务自捕获都可调用）。
func ReportCrash(v any, stack []byte) CrashReport {
	now := time.Now()
	msg := fmt.Sprint(v)
	rep := CrashReport{Time: now, Message: msg, Stack: string(stack)}
	dir := crashDir()
	if err := os.MkdirAll(dir, 0o755); err == nil {
		name := filepath.Join(dir, fmt.Sprintf("crash-%s.log", now.Format("20060102-150405")))
		body := fmt.Sprintf("time: %s\npanic: %s\n\n%s", now.Format(time.RFC3339), msg, stack)
		if os.WriteFile(name, []byte(body), 0o644) == nil {
			rep.File = name
		}
	}
	crashMu.Lock()
	crashLast = rep
	hook := crashHook
	crashMu.Unlock()
	if hook != nil {
		hook(rep)
	}
	return rep
}

func crashDir() string {
	crashMu.Lock()
	d := crashDirSet
	crashMu.Unlock()
	if d != "" {
		return d
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "gogui", "crashes")
}

// enableCrashLog 把未捕获 panic / fatal error 完整写入崩溃日志文件。
//
// Go 的 fatal error（后台 goroutine panic、concurrent map write、栈溢出、死锁）
// 不经过 recover，只打到 stderr；控制台滚屏后就只剩「exit status 2」，无从定位。
// SetCrashOutput 会让运行时把完整堆栈（含 goroutine 1 与 panic 头部）写到文件。
func enableCrashLog() {
	dir := crashDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("crash-%s.log", time.Now().Format("20060102-150405"))))
	if err != nil {
		return
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		_ = f.Close()
		return
	}
	// SetCrashOutput 会复制文件描述符，句柄可以立即关闭。
	_ = f.Close()
}

// startCPUProfileIfNeeded GOGUI_PPROF=<文件路径> 时采集 N 秒 CPU profile。
//
//	GOGUI_PPROF=cpu.prof  →  运行 8 秒后自动落盘
//	go tool pprof -top dist\gogui-gallery.exe cpu.prof
func startCPUProfileIfNeeded() {
	path := os.Getenv("GOGUI_PPROF")
	if path == "" {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		return
	}
	time.AfterFunc(8*time.Second, func() {
		pprof.StopCPUProfile()
		_ = f.Close()
	})
}

// RecoverToReport 供业务在 goroutine 里 defer 使用：
//
//	defer ui.RecoverToReport("worker")
func RecoverToReport(tag string) {
	if r := recover(); r != nil {
		ReportCrash(fmt.Sprintf("[%s] %v", tag, r), debug.Stack())
	}
}

// CrashPanel 崩溃日志面板（最近一次崩溃摘要 + 文件路径）。
func CrashPanel() *Node {
	rep, ok := CrashLast()
	if !ok {
		return EmptyState("", "运行正常", "还没有捕获到崩溃", "", nil)
	}
	col := Column(
		Row(
			Dot(ColDanger, 8),
			Text("最近一次崩溃").St(St().Text(12.5, 600, ColDanger)).Grow(1),
			Text(rep.Time.Format("2006-01-02 15:04:05")).
				St(St().Text(10.5, 400, ColTextMuted)),
		).St(St().GapOf(8)).AlignTo(AlignCenter),
		Text(rep.Message).St(St().Text(12, 500, ColText).WrapOn()),
	)
	stack := rep.Stack
	if len(stack) > 1600 {
		stack = stack[:1600] + "…"
	}
	col.Add(Text(stack).St(St().Text(10.5, 400, ColTextDim).WrapOn()).
		St(St().Padding(10).R(8).BgC(ColBgDeep)))
	if rep.File != "" {
		col.Add(Row(
			Text(rep.File).St(St().Text(10.5, 400, ColTextMuted).GrowTo(1).Clipped()),
			Text("打开目录").St(St().Text(11, 600, ColPrimary)).Tap(func() {
				RevealInExplorer(rep.File)
			}),
		).St(St().GapOf(8)).AlignTo(AlignCenter))
	}
	return col.St(St().GapOf(10).Padding(14).R(12).BgC(ColPanel).Stroke(1, ColStroke))
}
