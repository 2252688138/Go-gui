/*
 * Go-gui
 * Author  : Mr.Shu
 * Website : www.cq6k.com
 *
 * iOS 桥接头文件：Go(cgo) → Objective-C。
 *
 * 生命周期与事件方向相反：Objective-C 侧通过 extern 的 ggIOS* 函数回调 Go。
 */

#ifndef GOGUI_UIKIT_BRIDGE_H
#define GOGUI_UIKIT_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

/* Go 侧导出的回调（uikit_ios.go 中 //export） */
extern void ggIOSTouch(int phase, double x, double y);
extern void ggIOSResize(int w, int h, double scale);
extern void ggIOSKey(int keyCode, const char *chars, int down);

/* 初始化 UIApplication + UIWindow + EAGL 视图，返回窗口句柄（失败为 NULL） */
void *gg_ios_init(void);

/* 建立/重建 EAGL 的 renderbuffer（尺寸变化后调用），返回 1 成功 */
int gg_ios_setup_gl(void);

/* 绑定上下文 / 提交帧 */
int gg_ios_make_current(void);
void gg_ios_present(void);

/* 当前绘制缓冲的物理像素尺寸与缩放 */
void gg_ios_size(int *w, int *h, double *scale);

/* 窗口标题 / 可见性 / 全屏（移动端恒全屏） */
void gg_ios_set_title(void *win, const char *utf8);
void gg_ios_set_touch_events(int enabled);

/* 取 GL 函数地址（EAGL 只需 dlsym，返回非 0 即可用） */
void *gg_ios_gl_proc(const char *name);

/* 泵一轮事件（UIKit 的事件由 RunLoop 推动，这里只做空转 + 尺寸检测） */
void gg_ios_poll(void);

#ifdef __cplusplus
}
#endif

#endif /* GOGUI_UIKIT_BRIDGE_H */
