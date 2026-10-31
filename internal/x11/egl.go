//go:build (linux || freebsd) && !android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package x11

import (
	"errors"
	"sync"

	"github.com/ebitengine/purego"
)

// EGL 上下文（purego 加载 libEGL，零 cgo）。
//
// 为什么用 EGL 而不是 GLX：EGL 只需要「X11 Display* + Window id」就能建 surface，
// 且同一份代码后续能复用到 Wayland / Android（换 platform 参数即可），
// 而 GLX 需要额外绑定 glXCreateContextAttribsARB 等一堆扩展函数。

const (
	eglNone                  = 0x3038
	eglSurfaceType           = 0x3033
	eglWindowBit             = 0x0004
	eglRenderableType        = 0x3040
	eglOpenGLBit             = 0x0008
	eglRedSize               = 0x3024
	eglGreenSize             = 0x3023
	eglBlueSize              = 0x3022
	eglAlphaSize             = 0x3021
	eglDepthSize             = 0x3025
	eglContextMajorVersion   = 0x3098
	eglContextMinorVersion   = 0x30FB
	eglContextOpenGLProfile  = 0x30FD
	eglContextCoreProfileBit = 0x0001
	eglPlatformX11KHR        = 0x31D5
)

// 已解析的 EGL 函数。
var (
	eGLDisplay             uintptr
	eglGetDisplay          func(uintptr) uintptr
	eglGetPlatformDisplay  func(uintptr, uintptr, *int32) uintptr
	eglInitialize          func(uintptr, *int32, *int32) uint32
	eglTerminate           func(uintptr) uint32
	eglChooseConfig        func(uintptr, *int32, *uintptr, int32, *int32) uint32
	eglCreateWindowSurface func(uintptr, uintptr, uintptr, *int32) uintptr
	eglCreateContext       func(uintptr, uintptr, uintptr, *int32) uintptr
	eglMakeCurrent         func(uintptr, uintptr, uintptr, uintptr) uint32
	eglSwapBuffers         func(uintptr, uintptr) uint32
	eglGetProcAddress      func(string) uintptr
	eglBindAPI             func(uint32) uint32
	eglSwapInterval        func(uintptr, int32) uint32

	eglOnce sync.Once
	eglErr  error
)

const eglOpenGLAPI = 0x30A2

// LoadEGL 动态加载 libEGL（失败后重复返回同一错误）。
func LoadEGL() error {
	eglOnce.Do(func() {
		var h uintptr
		for _, name := range []string{"libEGL.so.1", "libEGL.so", "/usr/lib/libEGL.so.1", "/usr/lib/x86_64-linux-gnu/libEGL.so.1"} {
			if lib, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
				h = lib
				break
			}
		}
		if h == 0 {
			eglErr = errors.New("egl: 找不到 libEGL（请安装 libegl1 / libglvnd）")
			return
		}
		bind := func(dst any, name string) {
			if addr, err := purego.Dlsym(h, name); err == nil && addr != 0 {
				purego.RegisterFunc(dst, addr)
			}
		}
		bind(&eglGetDisplay, "eglGetDisplay")
		bind(&eglGetPlatformDisplay, "eglGetPlatformDisplay")
		bind(&eglInitialize, "eglInitialize")
		bind(&eglTerminate, "eglTerminate")
		bind(&eglChooseConfig, "eglChooseConfig")
		bind(&eglCreateWindowSurface, "eglCreateWindowSurface")
		bind(&eglCreateContext, "eglCreateContext")
		bind(&eglMakeCurrent, "eglMakeCurrent")
		bind(&eglSwapBuffers, "eglSwapBuffers")
		bind(&eglGetProcAddress, "eglGetProcAddress")
		bind(&eglBindAPI, "eglBindAPI")
		bind(&eglSwapInterval, "eglSwapInterval")
		if eglGetDisplay == nil && eglGetPlatformDisplay == nil {
			eglErr = errors.New("egl: libEGL 缺少 eglGetDisplay 符号")
		}
	})
	return eglErr
}

// Context EGL 上下文 + surface。
type Context struct {
	Display uintptr
	Surface uintptr
	Context uintptr
	Config  uintptr
}

// NewContext 为给定 X11 Display* 与 Window id 创建 OpenGL 3.3 Core 上下文。
func NewContext(dpy uintptr, win Window) (*Context, error) {
	if err := LoadEGL(); err != nil {
		return nil, err
	}
	edpy := uintptr(0)
	if eglGetPlatformDisplay != nil {
		edpy = eglGetPlatformDisplay(eglPlatformX11KHR, dpy, nil)
	}
	if edpy == 0 && eglGetDisplay != nil {
		edpy = eglGetDisplay(dpy)
	}
	if edpy == 0 {
		return nil, errors.New("egl: eglGetDisplay 失败（无法为 X11 显示建立 EGL）")
	}
	var major, minor int32
	if eglInitialize(edpy, &major, &minor) == 0 {
		return nil, errors.New("egl: eglInitialize 失败（显卡驱动缺 EGL 支持）")
	}
	if eglBindAPI != nil {
		eglBindAPI(eglOpenGLAPI)
	}
	cfgAttribs := []int32{
		eglSurfaceType, eglWindowBit,
		eglRenderableType, eglOpenGLBit,
		eglRedSize, 8,
		eglGreenSize, 8,
		eglBlueSize, 8,
		eglAlphaSize, 8,
		eglDepthSize, 0,
		eglNone,
	}
	var cfg uintptr
	var num int32
	if eglChooseConfig(edpy, &cfgAttribs[0], &cfg, 1, &num) == 0 || num < 1 {
		return nil, errors.New("egl: 找不到可用的 OpenGL 配置（eglChooseConfig）")
	}
	surf := eglCreateWindowSurface(edpy, cfg, uintptr(win), nil)
	if surf == 0 {
		return nil, errors.New("egl: eglCreateWindowSurface 失败（窗口不可渲染）")
	}
	ctxAttribs := []int32{
		eglContextMajorVersion, 3,
		eglContextMinorVersion, 3,
		eglContextOpenGLProfile, eglContextCoreProfileBit,
		eglNone,
	}
	ctx := eglCreateContext(edpy, cfg, 0, &ctxAttribs[0])
	if ctx == 0 {
		// 回退：不要 core profile（部分驱动只给到兼容模式）
		ctxAttribs = []int32{eglContextMajorVersion, 3, eglContextMinorVersion, 3, eglNone}
		ctx = eglCreateContext(edpy, cfg, 0, &ctxAttribs[0])
	}
	if ctx == 0 {
		return nil, errors.New("egl: eglCreateContext 失败（驱动不支持 OpenGL 3.3）")
	}
	c := &Context{Display: edpy, Surface: surf, Context: ctx, Config: cfg}
	if err := c.MakeCurrent(); err != nil {
		return nil, err
	}
	if eglSwapInterval != nil {
		eglSwapInterval(edpy, 1)
	}
	return c, nil
}

// MakeCurrent 绑定上下文到当前线程。
func (c *Context) MakeCurrent() error {
	if c == nil || eglMakeCurrent == nil {
		return errors.New("egl: 上下文未初始化")
	}
	if eglMakeCurrent(c.Display, c.Surface, c.Surface, c.Context) == 0 {
		return errors.New("egl: eglMakeCurrent 失败")
	}
	return nil
}

// Swap 提交一帧（等价于 wglSwapBuffers）。返回 false 表示交换失败（上下文/表面失效）。
func (c *Context) Swap() bool {
	if c == nil || eglSwapBuffers == nil {
		return false
	}
	return eglSwapBuffers(c.Display, c.Surface) != 0
}

// Proc 按名字取 GL 函数地址（交给 internal/gpu 的函数表）。
func (c *Context) Proc(name string) uintptr {
	if eglGetProcAddress != nil {
		if p := eglGetProcAddress(name); p != 0 {
			return p
		}
	}
	return 0
}

// Destroy 释放上下文与 surface。
func (c *Context) Destroy() {
	if c == nil || c.Display == 0 {
		return
	}
	if eglMakeCurrent != nil {
		eglMakeCurrent(c.Display, 0, 0, 0)
	}
	if eglTerminate != nil {
		eglTerminate(c.Display)
	}
	c.Display, c.Surface, c.Context = 0, 0, 0
}
