/*
 * Go-gui
 * Author  : Mr.Shu
 * Website : www.cq6k.com
 *
 * iOS 窗口/上下文桥接（Objective-C）。
 *
 * 结构：
 *   GoguiEAGLView  UIView 子类，layerClass 为 CAEAGLLayer；
 *                  OpenGL ES 3 上下文 + renderbuffer（framebuffer 0 直接指向屏幕）
 *   触摸事件        touchesBegan/Moved/Ended/Cancelled → ggIOSTouch
 *   键盘（软键盘）  UIKeyInput 协议 → insertText / deleteBackward → ggIOSKey
 */

#import <UIKit/UIKit.h>
#import <QuartzCore/QuartzCore.h>
#import <OpenGLES/ES3/gl.h>
#import <OpenGLES/ES3/glext.h>
#import <OpenGLES/EAGL.h>
#import <string.h>
#import <dlfcn.h>
#import <dispatch/dispatch.h>

#include "bridge_ios.h"

@interface GoguiEAGLView : UIView <UIKeyInput>
@property (nonatomic, strong) EAGLContext *glContext;
@property (nonatomic, assign) GLuint colorRenderbuffer;
@property (nonatomic, assign) GLuint framebuffer;
@property (nonatomic, assign) int pixelW;
@property (nonatomic, assign) int pixelH;
@property (nonatomic, assign) BOOL touchEnabled;
@end

static GoguiEAGLView *gView = nil;
static UIWindow *gWindow = nil;
static UIViewController *gController = nil;

@implementation GoguiEAGLView

+ (Class)layerClass {
    return [CAEAGLLayer class];
}

- (instancetype)initWithFrame:(CGRect)frame {
    self = [super initWithFrame:frame];
    if (self) {
        CAEAGLLayer *layer = (CAEAGLLayer *)self.layer;
        layer.opaque = YES;
        layer.contentsScale = [UIScreen mainScreen].scale;
        // 保留 drawable 内容，避免切后台回来闪黑
        layer.drawableProperties = @{
            kEAGLDrawablePropertyRetainedBacking: @NO,
            kEAGLDrawablePropertyColorFormat: kEAGLColorFormatRGBA8
        };
        _glContext = [[EAGLContext alloc] initWithAPI:kEAGLRenderingAPIOpenGLES3];
        if (!_glContext) {
            _glContext = [[EAGLContext alloc] initWithAPI:kEAGLRenderingAPIOpenGLES2];
        }
        self.multipleTouchEnabled = YES;
        self.touchEnabled = YES;
        [self setupGL];
    }
    return self;
}

- (void)setupGL {
    if (!_glContext) {
        return;
    }
    [EAGLContext setCurrentContext:_glContext];
    if (_framebuffer == 0) {
        glGenFramebuffers(1, &_framebuffer);
    }
    if (_colorRenderbuffer == 0) {
        glGenRenderbuffers(1, &_colorRenderbuffer);
    }
    glBindFramebuffer(GL_FRAMEBUFFER, _framebuffer);
    glBindRenderbuffer(GL_RENDERBUFFER, _colorRenderbuffer);
    // 关键：renderbuffer 的存储直接来自 CAEAGLLayer，framebuffer 0 即屏幕
    [_glContext renderbufferStorage:GL_RENDERBUFFER fromDrawable:(CAEAGLLayer *)self.layer];
    glFramebufferRenderbuffer(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_RENDERBUFFER, _colorRenderbuffer);

    GLint w = 0, h = 0;
    glGetRenderbufferParameteriv(GL_RENDERBUFFER, GL_RENDERBUFFER_WIDTH, &w);
    glGetRenderbufferParameteriv(GL_RENDERBUFFER, GL_RENDERBUFFER_HEIGHT, &h);
    _pixelW = w;
    _pixelH = h;
}

- (void)layoutSubviews {
    [super layoutSubviews];
    int oldW = _pixelW, oldH = _pixelH;
    [self setupGL];
    if (_pixelW != oldW || _pixelH != oldH) {
        ggIOSResize(_pixelW, _pixelH, (double)[UIScreen mainScreen].scale);
    }
}

/* ---- 触摸 ---- */

- (void)emitTouches:(NSSet<UITouch *> *)touches phase:(int)phase {
    if (!self.touchEnabled) {
        return;
    }
    for (UITouch *t in touches) {
        CGPoint p = [t locationInView:self];
        // UIKit 原点在左上角，与框架一致，无需翻转
        ggIOSTouch(phase, (double)p.x, (double)p.y);
        break; // 一期只跟踪第一根手指
    }
}

- (void)touchesBegan:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
    [self emitTouches:touches phase:0];
}
- (void)touchesMoved:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
    [self emitTouches:touches phase:1];
}
- (void)touchesEnded:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
    [self emitTouches:touches phase:2];
}
- (void)touchesCancelled:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
    [self emitTouches:touches phase:2];
}

/* ---- 软键盘（UIKeyInput） ---- */

- (BOOL)canBecomeFirstResponder {
    return YES;
}
- (BOOL)hasText {
    return NO;
}
- (void)insertText:(NSString *)text {
    const char *utf8 = [text UTF8String];
    ggIOSKey(0, utf8, 1);
}
- (void)deleteBackward {
    ggIOSKey(8, NULL, 1); // 退格
}
- (UITextAutocorrectionType)autocorrectionType {
    return UITextAutocorrectionTypeNo;
}

@end

/* UIKit 初始化必须在主线程执行：Go 的 main 跑在独立线程上，
 * 因此非主线程时同步切换到主队列（主线程此时在 UIApplicationMain 的 runloop 里）。*/
static void *gg_ios_init_main(void);

void *gg_ios_init(void) {
    if (![NSThread isMainThread]) {
        __block void *res = NULL;
        dispatch_sync(dispatch_get_main_queue(), ^{
            res = gg_ios_init_main();
        });
        return res;
    }
    return gg_ios_init_main();
}

static void *gg_ios_init_main(void) {
    if (gWindow != nil) {
        return (void *)CFBridgingRetain(gWindow);
    }
    UIApplication *app = [UIApplication sharedApplication];
    CGRect bounds = [[UIScreen mainScreen] bounds]; // 逻辑点

    gWindow = [[UIWindow alloc] initWithFrame:bounds];
    gController = [[UIViewController alloc] init];
    gView = [[GoguiEAGLView alloc] initWithFrame:bounds];
    gController.view = gView;
    gWindow.rootViewController = gController;
    [gWindow makeKeyAndVisible];
    [app setStatusBarHidden:YES];

    ggIOSResize(gView.pixelW, gView.pixelH, (double)[UIScreen mainScreen].scale);
    return (void *)CFBridgingRetain(gWindow);
}

int gg_ios_setup_gl(void) {
    if (gView == nil) {
        return 0;
    }
    [gView setupGL];
    return gView.pixelW > 0 && gView.pixelH > 0 ? 1 : 0;
}

int gg_ios_make_current(void) {
    if (gView == nil || gView.glContext == nil) {
        return 0;
    }
    BOOL ok = [EAGLContext setCurrentContext:gView.glContext];
    glBindFramebuffer(GL_FRAMEBUFFER, gView.framebuffer);
    glViewport(0, 0, gView.pixelW, gView.pixelH);
    return ok ? 1 : 0;
}

void gg_ios_present(void) {
    if (gView == nil || gView.glContext == nil) {
        return;
    }
    [gView.glContext presentRenderbuffer:GL_RENDERBUFFER];
}

void gg_ios_size(int *w, int *h, double *scale) {
    if (gView != nil) {
        if (w) { *w = gView.pixelW; }
        if (h) { *h = gView.pixelH; }
    }
    if (scale) {
        *scale = (double)[UIScreen mainScreen].scale;
    }
}

void gg_ios_set_title(void *win, const char *utf8) {
    // iOS 没有窗口标题：内容显示在 App 名/导航条，一期忽略
    (void)win;
    (void)utf8;
}

void gg_ios_set_touch_events(int enabled) {
    if (gView != nil) {
        gView.touchEnabled = enabled != 0;
    }
}

void *gg_ios_gl_proc(const char *name) {
    if (name == NULL) {
        return NULL;
    }
    void *p = dlsym(RTLD_DEFAULT, name);
    if (p != NULL) {
        return p;
    }
    // OpenGL ES 用统一前缀导出，dlsym 找不到时按 ES 前缀再试一次
    char buf[128];
    snprintf(buf, sizeof(buf), "gl%s", name);
    return dlsym(RTLD_DEFAULT, buf);
}

void gg_ios_poll(void) {
    // UIKit 事件由 runloop 派发，这里只做尺寸同步
    static int lastW = 0, lastH = 0;
    if (gView == nil) {
        return;
    }
    if (gView.pixelW != lastW || gView.pixelH != lastH) {
        lastW = gView.pixelW;
        lastH = gView.pixelH;
        ggIOSResize(lastW, lastH, (double)[UIScreen mainScreen].scale);
    }
}
