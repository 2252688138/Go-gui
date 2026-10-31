//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

// DirectComposition 合成层（B 阶段呈现层）：
//   GL → D3D11 纹理（wglDXRegisterObjectNV）→ IDCompositionSurface → DWM 合成
//
// 纯 Go 构建下若 float 接口不可用，调用方（ui 层）会自动回退到 A 阶段。

var (
	composeOnce sync.Once
	composeErr  error

	d3d11CreateDevice     func(adapter, driverType, software, flags, levels, numLevels, sdkVer, ppDev, pLevel, ppCtx uintptr) uintptr
	dcompCreateDeviceFunc func(dxgiDevice, riid, ppDevice uintptr) uintptr

	pDXOpen   uintptr
	pDXClose  uintptr
	pDXReg    uintptr
	pDXUnreg  uintptr
	pDXLock   uintptr
	pDXUnlock uintptr
)

const (
	dcIIDDevice = "c37ea93a-e7aa-450d-b16f-9746cb0407f3"
	dcIIDTex2D  = "6f15aaf2-d208-4e89-9ab4-489535d34f9c"
)

const (
	dcDevCommit        = 3
	dcDevCreateTarget  = 6
	dcDevCreateVisual  = 7
	dcDevCreateSurface = 8
	dcDevCreateClip    = 24

	dcTargetSetRoot = 3

	// 注意：运行时虚表的 SetOffsetX/SetOffsetY 是「动画版在前、float 版在后」，
	// 与 SDK 头文件的声明顺序相反（实测 float 版在 4 / 6，槽 3 / 5 会按动画指针崩溃）。
	dcVisualSetOffsetX = 4
	dcVisualSetOffsetY = 6
	dcVisualSetClip    = 14 // 运行时：13=SetClipAnimation，14=SetClip
	dcVisualSetContent = 15
	dcVisualAddVisual  = 16

	dcSurfaceBeginDraw = 3
	dcSurfaceEndDraw   = 4

	dxgiFormatBGRA8 = 87
	dxgiAlphaPremul = 1
	d3d11FlagsBGRA  = 0x20
	d3d11SDKVersion = 7
	d3dDriverTypeHW = 1

	glAccessWriteNV = 0x0002
	glTex2D         = 0x0DE1
	glFramebuffer   = 0x8D40
	glColorAttach0  = 0x8CE0
	glFBComplete    = 0x8CD5
)

// ComposeSupported B 阶段是否可用（D3D11 + dcomp + WGL 互操作齐备）。
func ComposeSupported() bool {
	if os.Getenv("GOGUI_COMPOSE") != "1" {
		return false
	}
	return loadCompose() == nil
}

// ComposeError 返回合成层不可用的原因。
func ComposeError() error { _ = loadCompose(); return composeErr }

func loadCompose() error {
	composeOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				composeErr = fmt.Errorf("加载合成层失败: %v", r)
			}
		}()
		libD3D := loadLibOptional("d3d11.dll")
		libDC := loadLibOptional("dcomp.dll")
		if libD3D == 0 || libDC == 0 {
			composeErr = errors.New("缺少 d3d11.dll / dcomp.dll")
			return
		}
		purego.RegisterLibFunc(&d3d11CreateDevice, libD3D, "D3D11CreateDevice")
		purego.RegisterLibFunc(&dcompCreateDeviceFunc, libDC, "DCompositionCreateDevice")

		pDXOpen = wglProc("wglDXOpenDeviceNV")
		if pDXOpen == 0 {
			composeErr = errors.New("显卡驱动不支持 WGL_NV_DX_interop2")
			return
		}
		pDXClose = wglProc("wglDXCloseDeviceNV")
		pDXReg = wglProc("wglDXRegisterObjectNV")
		pDXUnreg = wglProc("wglDXUnregisterObjectNV")
		pDXLock = wglProc("wglDXLockObjectsNV")
		pDXUnlock = wglProc("wglDXUnlockObjectsNV")
	})
	return composeErr
}

// wglProc 取 WGL 扩展入口地址（需 GL 上下文 current）。
func wglProc(name string) uintptr {
	pb, err := syscall.BytePtrFromString(name)
	if err != nil {
		return 0
	}
	p := wglGetProcAddress(pb)
	if p == 0 || p == 1 || p == 2 || p == 3 || p == ^uintptr(0) {
		return 0
	}
	return p
}

// Compositor 一个窗口的 DComp 合成器。
type Compositor struct {
	hwnd uintptr
	w, h int32

	d3dDev, d3dCtx, dxgiDev uintptr
	dcomp, target, root     uintptr
	surf                    uintptr

	glDev  uintptr
	glTex  uint32
	glFBO  uint32
	glObj  uintptr
	curTex uintptr
	locked bool
	inUse  bool
}

// NewCompositor 创建合成器（hwnd 目标窗口，w/h 物理像素）。
func NewCompositor(hwnd uintptr, w, h int32) (*Compositor, error) {
	if err := loadCompose(); err != nil {
		return nil, err
	}
	if hwnd == 0 || w <= 0 || h <= 0 {
		return nil, errors.New("Compositor: 参数无效")
	}
	c := &Compositor{hwnd: hwnd, w: w, h: h}

	var dev, ctx, level uintptr
	if hr := d3d11CreateDevice(0, d3dDriverTypeHW, 0, d3d11FlagsBGRA, 0, 0, d3d11SDKVersion,
		uintptr(unsafe.Pointer(&dev)), uintptr(unsafe.Pointer(&level)), uintptr(unsafe.Pointer(&ctx))); comFail(uint32(hr)) || dev == 0 {
		return nil, fmt.Errorf("D3D11CreateDevice 失败（0x%08x）", uint32(hr))
	}
	c.d3dDev, c.d3dCtx = dev, ctx

	iidDXGI := parseGUID("54ec77fa-1377-44e6-8c32-88fd5f44c84c")
	if hr := comCall(dev, 0, uintptr(unsafe.Pointer(&iidDXGI)), uintptr(unsafe.Pointer(&c.dxgiDev))); comFail(hr) || c.dxgiDev == 0 {
		return nil, fmt.Errorf("取 IDXGIDevice 失败（0x%08x）", hr)
	}
	iidDC := parseGUID(dcIIDDevice)
	if hr := dcompCreateDeviceFunc(c.dxgiDev, uintptr(unsafe.Pointer(&iidDC)), uintptr(unsafe.Pointer(&c.dcomp))); comFail(uint32(hr)) || c.dcomp == 0 {
		return nil, fmt.Errorf("DCompositionCreateDevice 失败（0x%08x）", uint32(hr))
	}
	if hr := comCall(c.dcomp, dcDevCreateTarget, hwnd, 1, uintptr(unsafe.Pointer(&c.target))); comFail(hr) || c.target == 0 {
		return nil, fmt.Errorf("CreateTargetForHwnd 失败（0x%08x）", hr)
	}
	if hr := comCall(c.dcomp, dcDevCreateVisual, uintptr(unsafe.Pointer(&c.root))); comFail(hr) || c.root == 0 {
		return nil, fmt.Errorf("CreateVisual 失败（0x%08x）", hr)
	}
	if hr := comCall(c.target, dcTargetSetRoot, c.root); comFail(hr) {
		return nil, fmt.Errorf("SetRoot 失败（0x%08x）", hr)
	}
	if err := c.createSurface(w, h); err != nil {
		return nil, err
	}
	c.glDev = glDXOpen(c.d3dDev)
	if c.glDev == 0 {
		return nil, errors.New("wglDXOpenDeviceNV 失败")
	}
	GlGenTextures(1, &c.glTex)
	GlGenFramebuffers(1, &c.glFBO)
	dbg("Compositor 就绪 %dx%d", w, h)
	return c, nil
}

// glDXOpen 包装 wglDXOpenDeviceNV。
func glDXOpen(dxDev uintptr) uintptr {
	r, _, _ := purego.SyscallN(pDXOpen, dxDev)
	return r
}

func (c *Compositor) createSurface(w, h int32) error {
	if c.surf != 0 {
		comRelease(c.surf)
		c.surf = 0
	}
	if hr := comCall(c.dcomp, dcDevCreateSurface, uintptr(w), uintptr(h),
		dxgiFormatBGRA8, dxgiAlphaPremul, uintptr(unsafe.Pointer(&c.surf))); comFail(hr) || c.surf == 0 {
		return fmt.Errorf("CreateSurface 失败（0x%08x）", hr)
	}
	if hr := comCall(c.root, dcVisualSetContent, c.surf); comFail(hr) {
		return fmt.Errorf("SetContent 失败（0x%08x）", hr)
	}
	comCall(c.dcomp, dcDevCommit)
	c.w, c.h = w, h
	return nil
}

// RootVisual 根视觉（网页图层挂到它下面）。
func (c *Compositor) RootVisual() uintptr {
	if c == nil {
		return 0
	}
	return c.root
}

// NewVisual 在根视觉下创建子视觉（零偏移）。
func (c *Compositor) NewVisual() (uintptr, error) {
	if c == nil || c.dcomp == 0 {
		return 0, errors.New("Compositor 未初始化")
	}
	var v uintptr
	if hr := comCall(c.dcomp, dcDevCreateVisual, uintptr(unsafe.Pointer(&v))); comFail(hr) || v == 0 {
		return 0, fmt.Errorf("CreateVisual 失败（0x%08x）", hr)
	}
	if hr := comCall(c.root, dcVisualAddVisual, v, 0); comFail(hr) {
		return 0, fmt.Errorf("AddVisual 失败（0x%08x）", hr)
	}
	comCall(c.dcomp, dcDevCommit)
	return v, nil
}

// SetRoundedClip 给视觉设置圆角裁剪（float 接口；纯 Go 下可能不可用）。
func (c *Compositor) SetRoundedClip(visual uintptr, x, y, w, h, radius float32) error {
	if c == nil || c.dcomp == 0 || visual == 0 {
		return errors.New("Compositor 未初始化")
	}
	var clip uintptr
	if hr := comCall(c.dcomp, dcDevCreateClip, uintptr(unsafe.Pointer(&clip))); comFail(hr) || clip == 0 {
		return fmt.Errorf("CreateRectangleClip 失败（0x%08x）", hr)
	}
	defer comRelease(clip)
	ComCallF(clip, 4, x)
	ComCallF(clip, 6, y)
	ComCallF(clip, 8, x+w)
	ComCallF(clip, 10, y+h)
	ComCallF(clip, 12, radius)
	ComCallF(clip, 14, radius)
	ComCallF(clip, 16, radius)
	ComCallF(clip, 18, radius)
	ComCallF(clip, 20, radius)
	ComCallF(clip, 22, radius)
	ComCallF(clip, 24, radius)
	ComCallF(clip, 26, radius)
	if hr := comCall(visual, dcVisualSetClip, clip); comFail(hr) {
		return fmt.Errorf("SetClip 失败（0x%08x）", hr)
	}
	comCall(c.dcomp, dcDevCommit)
	return nil
}

// BeginFrame 取一帧可写的 D3D11 纹理并绑定为 GL FBO。
func (c *Compositor) BeginFrame() (fbo uint32, w, h int32, err error) {
	if c == nil || c.surf == 0 {
		return 0, 0, 0, errors.New("Compositor 未初始化")
	}
	if c.inUse {
		return c.glFBO, c.w, c.h, nil
	}
	iidTex := parseGUID(dcIIDTex2D)
	var tex uintptr
	var off [2]int32
	if hr := comCall(c.surf, dcSurfaceBeginDraw, 0, uintptr(unsafe.Pointer(&iidTex)),
		uintptr(unsafe.Pointer(&tex)), uintptr(unsafe.Pointer(&off[0]))); comFail(hr) || tex == 0 {
		return 0, 0, 0, fmt.Errorf("BeginDraw 失败（0x%08x）", hr)
	}
	if tex != c.curTex {
		if c.glObj != 0 {
			unreg := c.glObj
			purego.SyscallN(pDXUnreg, c.glDev, unreg)
			c.glObj = 0
		}
		c.glObj, _, _ = purego.SyscallN(pDXReg, c.glDev, tex, uintptr(c.glTex), glTex2D, glAccessWriteNV)
		c.curTex = tex
		if c.glObj == 0 {
			return 0, 0, 0, errors.New("wglDXRegisterObjectNV 失败")
		}
	}
	objs := c.glObj
	if r, _, _ := purego.SyscallN(pDXLock, c.glDev, 1, uintptr(unsafe.Pointer(&objs))); r == 0 {
		return 0, 0, 0, errors.New("wglDXLockObjectsNV 失败")
	}
	c.locked = true
	GlBindTexture(glTex2D, c.glTex)
	GlBindFramebuffer(glFramebuffer, c.glFBO)
	GlFramebufferTexture2D(glFramebuffer, glColorAttach0, glTex2D, c.glTex, 0)
	if st := GlCheckFramebufferStatus(glFramebuffer); st != glFBComplete {
		GlBindFramebuffer(glFramebuffer, 0)
		return 0, 0, 0, fmt.Errorf("离屏 FBO 不完整（0x%x）", st)
	}
	c.inUse = true
	return c.glFBO, c.w, c.h, nil
}

// EndFrame 解锁并提交。
func (c *Compositor) EndFrame() {
	if c == nil || !c.inUse {
		return
	}
	GlBindFramebuffer(glFramebuffer, 0)
	if c.locked {
		objs := c.glObj
		purego.SyscallN(pDXUnlock, c.glDev, 1, uintptr(unsafe.Pointer(&objs)))
		c.locked = false
	}
	comCall(c.surf, dcSurfaceEndDraw)
	comCall(c.dcomp, dcDevCommit)
	c.inUse = false
}

// Resize 重建表面。
func (c *Compositor) Resize(w, h int32) error {
	if c == nil {
		return errors.New("Compositor 未初始化")
	}
	if w <= 0 || h <= 0 || (w == c.w && h == c.h) {
		return nil
	}
	if c.inUse {
		c.EndFrame()
	}
	if c.glObj != 0 {
		purego.SyscallN(pDXUnreg, c.glDev, c.glObj)
		c.glObj, c.curTex = 0, 0
	}
	return c.createSurface(w, h)
}

// Close 释放合成资源。
func (c *Compositor) Close() {
	if c == nil {
		return
	}
	if c.inUse {
		c.EndFrame()
	}
	if c.glObj != 0 {
		purego.SyscallN(pDXUnreg, c.glDev, c.glObj)
		c.glObj, c.curTex = 0, 0
	}
	if c.glDev != 0 {
		purego.SyscallN(pDXClose, c.glDev)
		c.glDev = 0
	}
	if c.glTex != 0 {
		GlDeleteTextures(1, &c.glTex)
	}
	if c.glFBO != 0 {
		GlDeleteFramebuffers(1, &c.glFBO)
	}
	for _, p := range []uintptr{c.surf, c.root, c.target} {
		comRelease(p)
	}
	if c.dcomp != 0 {
		comRelease(c.dcomp)
	}
	comRelease(c.dxgiDev)
	comRelease(c.d3dCtx)
	comRelease(c.d3dDev)
	c.surf, c.root, c.target, c.dcomp = 0, 0, 0, 0
}

// SetVisualOffset 设置视觉偏移（物理像素）。B 阶段用它把网页图层摆到节点矩形位置。
func (c *Compositor) SetVisualOffset(visual uintptr, x, y float32) {
	if c == nil || visual == 0 {
		return
	}
	ComCallF(visual, dcVisualSetOffsetX, x)
	ComCallF(visual, dcVisualSetOffsetY, y)
	comCall(c.dcomp, dcDevCommit)
}
