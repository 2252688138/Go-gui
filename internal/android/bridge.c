/*
 * Go-gui
 * Author  : Mr.Shu
 * Website : www.cq6k.com
 *
 * Android NativeActivity 与 Go 之间的桥接。
 *
 * 为什么要有这个文件：cgo 规定「含 //export 的 Go 文件，其 preamble 只能放声明、
 * 不能放定义」。因此把所有 C 侧的静态函数与回调装配放在这里，
 * Go 侧只保留 //export 的导出函数。
 *
 * 注意 ANativeActivity 结构体本身**没有** inputQueue 字段（那是
 * android_native_app_glue 的 app 结构体），输入队列由 Go 侧自己保存。
 */

#include <android/native_activity.h>
#include <android/native_window.h>
#include <android/input.h>
#include <stdlib.h>

#include "bridge.h"

/* Go 侧导出的函数（在 activity.go 中 //export） */
extern void ggGoWindowCreated(ANativeWindow *w);
extern void ggGoWindowDestroyed(void);
extern void ggGoInputQueueCreated(AInputQueue *q);
extern void ggGoInputQueueDestroyed(void);
extern void ggGoAppDestroyed(void);

static void gg_cb_window_created(ANativeActivity *a, ANativeWindow *w) {
    ggGoWindowCreated(w);
}

static void gg_cb_window_resized(ANativeActivity *a, ANativeWindow *w) {
    /* 尺寸变化由 Go 侧轮询 ANativeWindow_getWidth/Height 检测 */
    (void)a;
    (void)w;
}

static void gg_cb_window_redraw(ANativeActivity *a, ANativeWindow *w) {
    /* 自绘框架按帧刷新，系统重绘请求无需处理 */
    (void)a;
    (void)w;
}

static void gg_cb_window_destroyed(ANativeActivity *a, ANativeWindow *w) {
    (void)a;
    (void)w;
    ggGoWindowDestroyed();
}

static void gg_cb_input_created(ANativeActivity *a, AInputQueue *q) {
    (void)a;
    ggGoInputQueueCreated(q);
}

static void gg_cb_input_destroyed(ANativeActivity *a, AInputQueue *q) {
    (void)a;
    (void)q;
    ggGoInputQueueDestroyed();
}

static void gg_cb_destroy(ANativeActivity *a) {
    (void)a;
    ggGoAppDestroyed();
}

void gg_install_callbacks(ANativeActivity *a) {
    if (a == NULL || a->callbacks == NULL) {
        return;
    }
    a->callbacks->onNativeWindowCreated = gg_cb_window_created;
    a->callbacks->onNativeWindowResized = gg_cb_window_resized;
    a->callbacks->onNativeWindowRedrawNeeded = gg_cb_window_redraw;
    a->callbacks->onNativeWindowDestroyed = gg_cb_window_destroyed;
    a->callbacks->onInputQueueCreated = gg_cb_input_created;
    a->callbacks->onInputQueueDestroyed = gg_cb_input_destroyed;
    a->callbacks->onDestroy = gg_cb_destroy;
}
