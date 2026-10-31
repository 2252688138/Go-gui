/*
 * Go-gui
 * Author  : Mr.Shu
 * Website : www.cq6k.com
 *
 * Android 桥接头文件：Go 侧通过 cgo 调用的 C 入口。
 */

#ifndef GOGUI_ANDROID_BRIDGE_H
#define GOGUI_ANDROID_BRIDGE_H

#include <android/native_activity.h>
#include <android/native_window.h>

#ifdef __cplusplus
extern "C" {
#endif

/* 把窗口/输入队列/销毁回调装到 ANativeActivity 上 */
void gg_install_callbacks(ANativeActivity *a);

#ifdef __cplusplus
}
#endif

#endif /* GOGUI_ANDROID_BRIDGE_H */
