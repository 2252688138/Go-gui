//go:build android && cgo

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package android

/*
#include <android/native_activity.h>
#include <android/native_window.h>
#include <android/input.h>

#include "bridge.h"
*/
import "C"

import (
	"sync"
	"unsafe"
)

// ANativeActivity 生命周期桥接。
//
// Android 的 NativeActivity 会在 lib 加载后调用导出符号 `ANativeActivity_onCreate`。
// 我们在这里把系统回调装好（见 bridge.c），并把三件事交给 Go 侧：
//
//	窗口创建     → SetWindow（随后建立 EGL 表面与 GLES3 上下文）
//	输入队列就绪 → SetInputQueue（轮询取触摸/按键）
//	窗口销毁     → 释放 EGL 资源
//
// UI 主循环由调用方通过 OnCreate 注入（见 mobile/android/main.go），
// 因为「启动哪个界面」属于应用层的事。

var (
	createOnce sync.Once
	createMu   sync.Mutex
	// OnCreate 应用主循环入口（由 mobile/android 的 main 设置）。
	OnCreate func()
)

//export ANativeActivity_onCreate
func ANativeActivity_onCreate(act *C.ANativeActivity, savedState unsafe.Pointer, savedSize C.size_t) {
	if act == nil {
		return
	}
	SetActivity(uintptr(unsafe.Pointer(act)))
	// 装窗口 / 输入 / 销毁回调
	C.gg_install_callbacks(act)
	createMu.Lock()
	fn := OnCreate
	createMu.Unlock()
	if fn != nil {
		createOnce.Do(func() { go fn() })
	}
}

//export ggGoWindowCreated
func ggGoWindowCreated(w *C.ANativeWindow) {
	if w == nil {
		return
	}
	_ = SetWindow(uintptr(unsafe.Pointer(w)))
}

//export ggGoWindowDestroyed
func ggGoWindowDestroyed() {
	_ = SetWindow(0)
}

//export ggGoInputQueueCreated
func ggGoInputQueueCreated(q *C.AInputQueue) {
	SetInputQueue(uintptr(unsafe.Pointer(q)))
}

//export ggGoInputQueueDestroyed
func ggGoInputQueueDestroyed() {
	SetInputQueue(0)
}

//export ggGoAppDestroyed
func ggGoAppDestroyed() {
	Destroy()
}
