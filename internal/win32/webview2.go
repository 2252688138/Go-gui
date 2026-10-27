//go:build windows

package win32

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

// 系统内置 WebView2（Edge 内核）最小化托管层：纯 syscall / purego，无 cgo、无 SDK 依赖。
//
// 只封装三层接口：
//   - ICoreWebView2Environment  创建 / 复用浏览器进程环境
//   - ICoreWebView2Controller   节点矩形（bounds）、显隐、焦点
//   - ICoreWebView2             导航、前后退、刷新、标题 / 地址 / 历史状态
//
// 当前为 **HWND 托管**：网页由 Chromium GPU 进程渲染进自己的子窗口，
// 因此像素无法被 OpenGL 批次采样（进纹理需要 DirectComposition 合成或 CEF 离屏）。
// 子窗口挂在宿主窗口客户区里，位置 = 节点矩形（物理像素）。
//
// 注意：接口中带 double 参数的方法（put_ZoomFactor / put_RasterizationScale）
// 不能经 SyscallN 调用（XMM 寄存器传参），缩放改走 ExecuteScript 注入 CSS zoom。

// ---- 接口 IID（取自 WebView2.h）----

var wvIIDCtrl2 = parseGUID("c979903e-d4ca-4228-92eb-47ee3fa96eab")

// 虚表槽位（0=QueryInterface 1=AddRef 2=Release，其后按接口声明顺序）。
const (
	wvEnvCreateController = 3 // ICoreWebView2Environment::CreateCoreWebView2Controller

	wvCtrlPutVisible = 4  // ICoreWebView2Controller::put_IsVisible
	wvCtrlPutBounds  = 6  // ...put_Bounds
	wvCtrlMoveFocus  = 12 // ...MoveFocus
	wvCtrlNotifyPos  = 23 // ...NotifyParentWindowPositionChanged
	wvCtrlClose      = 24 // ...Close
	wvCtrlGetCore    = 25 // ...get_CoreWebView2

	wvCtrl2PutBackdrop = 4 // ICoreWebView2Controller2::put_DefaultBackgroundColor

	wvCoreGetSettings   = 3  // ICoreWebView2::get_Settings
	wvCoreGetSource     = 4  // ...get_Source
	wvCoreNavigate      = 5  // ...Navigate
	wvCoreNavigateHTML  = 6  // ...NavigateToString
	wvCoreAddNavStart   = 7  // ...add_NavigationStarting
	wvCoreAddSourceChg  = 11 // ...add_SourceChanged
	wvCoreAddHistoryChg = 13 // ...add_HistoryChanged
	wvCoreAddNavDone    = 15 // ...add_NavigationCompleted
	wvCoreExecuteScript = 29 // ...ExecuteScript
	wvCoreReload        = 31 // ...Reload
	wvCoreCanGoBack     = 38 // ...get_CanGoBack
	wvCoreCanGoFwd      = 39 // ...get_CanGoForward
	wvCoreGoBack        = 40 // ...GoBack
	wvCoreGoForward     = 41 // ...GoForward
	wvCoreStop          = 43 // ...Stop
	wvCoreAddTitleChg   = 46 // ...add_DocumentTitleChanged
	wvCoreGetTitle      = 48 // ...get_DocumentTitle

	wvSetPutStatusBar = 10 // ICoreWebView2Settings::put_IsStatusBarEnabled
	wvSetPutCtxMenu   = 14 // ...put_AreDefaultContextMenusEnabled
)

// COINIT_APARTMENTTHREADED WebView2 要求 UI 线程为 STA。
const coinitApartmentThreaded = 0x2

// ---- WebView2Loader ----

// wvLoaderDLL 内嵌官方 WebView2Loader.dll（x64，165KB）。
// 这样无论程序从哪里启动、目录里有没有 DLL，都能自己释放并加载。
//
//go:embed webview2loader/WebView2Loader.dll
var wvLoaderDLL []byte

var (
	wvLibOnce sync.Once
	wvLibErr  error

	wvCreateEnvFn func(browserFolder, userDataFolder, options, handler uintptr) uintptr
	wvCoInitFn    func(pv uintptr) uintptr

	// 合成控制器创建入口（ICoreWebView2Environment3::CreateCoreWebView2CompositionController）
	wvCompCtrlSlot = 3
)

// wvLoaderCandidates WebView2Loader.dll 候选路径（按优先级）。
func wvLoaderCandidates() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" {
			return
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		key := strings.ToLower(p)
		if seen[key] {
			return
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			seen[key] = true
			out = append(out, p)
		}
	}
	add(os.Getenv("GOGUI_WEBVIEW2_LOADER"))
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(filepath.Join(dir, "WebView2Loader.dll"))
		add(filepath.Join(dir, "third_party", "webview2", "x64", "WebView2Loader.dll"))
	}
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "WebView2Loader.dll"))
		add(filepath.Join(wd, "third_party", "webview2", "x64", "WebView2Loader.dll"))
	}
	return out
}

// wvLoad 加载 WebView2Loader.dll 并绑定导出函数（只做一次）。
func wvLoad() error {
	wvLibOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				wvLibErr = fmt.Errorf("加载 WebView2Loader 失败: %v", r)
			}
		}()
		var lib uintptr
		for _, p := range wvLoaderCandidates() {
			if h, err := syscall.LoadLibrary(p); err == nil {
				lib = uintptr(h)
				dbg("WebView2Loader = %s", p)
				break
			}
		}
		if lib == 0 {
			// 兵底：把内嵌的 Loader 释放到 %LOCALAPPDATA%\GoGUI\WebView2Loader 再加载
			if p, err := extractLoaderDLL(); err == nil {
				if h, err2 := syscall.LoadLibrary(p); err2 == nil {
					lib = uintptr(h)
					dbg("WebView2Loader（内嵌释放）= %s", p)
				}
			}
		}
		if lib == 0 {
			wvLibErr = errors.New("WebView2Loader.dll 加载失败（可设置 GOGUI_WEBVIEW2_LOADER 指定路径）")
			return
		}
		purego.RegisterLibFunc(&wvCreateEnvFn, lib, "CreateCoreWebView2EnvironmentWithOptions")
		if libOle32 == 0 {
			libOle32 = loadLib("ole32.dll")
		}
		// WebView2 要求 UI 线程处于 STA，且需要 OLE（剪贴板/拖放）
		purego.RegisterLibFunc(&wvCoInitFn, libOle32, "OleInitialize")
	})
	return wvLibErr
}

// extractLoaderDLL 把内嵌的 WebView2Loader.dll 释放到 %LOCALAPPDATA%\GoGUI\WebView2Loader\，
// 文件名带大小，已存在同尺寸文件时直接复用（不需要管理员权限）。
func extractLoaderDLL() (string, error) {
	if len(wvLoaderDLL) == 0 {
		return "", errors.New("未内嵌 WebView2Loader.dll")
	}
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "GoGUI", "WebView2Loader")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, fmt.Sprintf("WebView2Loader_%d.dll", len(wvLoaderDLL)))
	if st, err := os.Stat(p); err == nil && st.Size() == int64(len(wvLoaderDLL)) {
		return p, nil
	}
	if err := os.WriteFile(p, wvLoaderDLL, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// ---- 自实现 COM 回调对象 ----

// wvCallback 一个 Go 实现的 COM 对象（只用于 WebView2 的完成回调与事件回调）。
//
// 布局要点：COM 接口指针必须指向「首字段是 vtable 指针」的对象，
// 不能直接指向 vtable 数组（否则 DLL 会把回调函数地址当成 vtable 去读，访问违例）。
type wvCallback struct {
	vtable [4]uintptr
	// objRef 就是 COM 对象本体：第一个（也是唯一）字段指向 vtable。
	objRef uintptr
	refs   int64
	invoke func(this, a, b uintptr) uintptr
	obj    uintptr // &objRef，传给 WebView2
}

var (
	wvCbOnce       sync.Once
	wvCbQueryPtr   uintptr
	wvCbAddRefPtr  uintptr
	wvCbReleasePtr uintptr
	wvCbInvokePtr  uintptr
	wvCbMu         sync.Mutex
	wvCbObjects    = map[uintptr]*wvCallback{}
)

// wvCbFind 按接口指针找回调对象。
func wvCbFind(this uintptr) *wvCallback {
	wvCbMu.Lock()
	c := wvCbObjects[this]
	wvCbMu.Unlock()
	return c
}

func wvInitCallbackVtables() {
	wvCbOnce.Do(func() {
		wvCbQueryPtr = purego.NewCallback(func(this, riid, ppv uintptr) uintptr {
			// 回调对象只服务于单一接口：任意 IID 都返回自身
			if ppv != 0 {
				*(*uintptr)(unsafe.Pointer(ppv)) = this
			}
			if c := wvCbFind(this); c != nil {
				atomic.AddInt64(&c.refs, 1)
			}
			return 0
		})
		wvCbAddRefPtr = purego.NewCallback(func(this uintptr) uintptr {
			if c := wvCbFind(this); c != nil {
				return uintptr(atomic.AddInt64(&c.refs, 1))
			}
			return 1
		})
		wvCbReleasePtr = purego.NewCallback(func(this uintptr) uintptr {
			if c := wvCbFind(this); c != nil {
				n := atomic.AddInt64(&c.refs, -1)
				if n <= 0 {
					return 0
				}
				return uintptr(n)
			}
			return 0
		})
		wvCbInvokePtr = purego.NewCallback(func(this, a, b uintptr) uintptr {
			if c := wvCbFind(this); c != nil && c.invoke != nil {
				return c.invoke(this, a, b)
			}
			return 0
		})
	})
}

// newWVCallback 创建回调对象，返回 COM 接口指针（vtable 首地址）。
func newWVCallback(invoke func(this, a, b uintptr) uintptr) uintptr {
	wvInitCallbackVtables()
	c := &wvCallback{invoke: invoke, refs: 1}
	c.vtable[0] = wvCbQueryPtr
	c.vtable[1] = wvCbAddRefPtr
	c.vtable[2] = wvCbReleasePtr
	c.vtable[3] = wvCbInvokePtr
	c.objRef = uintptr(unsafe.Pointer(&c.vtable[0]))
	c.obj = uintptr(unsafe.Pointer(&c.objRef))
	wvCbMu.Lock()
	wvCbObjects[c.obj] = c
	wvCbMu.Unlock()
	return c.obj
}

func dropWVCallback(obj uintptr) {
	if obj == 0 {
		return
	}
	wvCbMu.Lock()
	delete(wvCbObjects, obj)
	wvCbMu.Unlock()
}

// ---- 环境（整个进程共用一个浏览器进程）----

// WebView2Options 创建一个内嵌浏览器实例的参数。
type WebView2Options struct {
	// Parent 宿主窗口句柄（网页子窗口挂在它的客户区里）。
	Parent uintptr
	// URL 初始地址（与 HTML 二选一，HTML 优先）。
	URL string
	// HTML 直接加载一段 HTML（离线页面、模板）。
	HTML string
	// UserDataDir 用户数据目录（cookie / 缓存）；空 = %LOCALAPPDATA%\GoGUI\WebView2。
	UserDataDir string
	// Zoom 初始缩放（1 = 100%，通过注入 CSS zoom 实现）。
	Zoom float64
	// Transparent 背景透明（网页自身透明处显示下层自绘内容）。
	Transparent bool
	// CompositionRoot 非 0 时走「合成控制器」（B 阶段）：网页作为 DComp 图层渲染，
	// 传入一个 IDCompositionVisual（由 Compositor.NewVisual 创建）。
	// 置 0 则走 HWND 托管（A 阶段）。
	CompositionRoot uintptr
	// OnChange 状态变化（加载开始/结束、地址、标题、历史）时回调；可能在 UI 线程重入。
	OnChange func()
}

// WebView2 一个内嵌浏览器实例。
type WebView2 struct {
	parent uintptr
	opts   WebView2Options

	ctrl    uintptr
	ctrl2   uintptr
	core    uintptr
	settng  uintptr
	ready   bool
	closed  bool
	lastErr error

	// 待应用的布局（控制器还没就绪时先记下）
	wantBounds  RECT
	wantVisible bool
	hasBounds   bool

	// 状态（事件置脏，Refresh 时从接口读回）
	dirty      atomic.Bool
	loading    atomic.Bool
	canBack    atomic.Bool
	canForward atomic.Bool
	href       atomic.Value // string
	title      atomic.Value // string

	mu        sync.Mutex
	callbacks []uintptr
	// inner 网页绘图子窗口（Chromium 窗口），inner* 记录当前区域参数
	inner       uintptr
	comp        uintptr // ICoreWebView2CompositionController（B 阶段）
	compRoot    uintptr
	innerW, innerH int32
	innerRadius int32
}

var (
	wvEnvMu    sync.Mutex
	wvEnvObj   uintptr
	wvEnvErr   error
	wvEnvStart bool
	wvEnvQueue []*WebView2
)

// WebView2RuntimeAvailable 运行时是否可用（Loader 存在 + 运行时目录存在）。
func WebView2RuntimeAvailable() bool {
	if err := wvLoad(); err != nil {
		return false
	}
	return wvRuntimeInstalled()
}

// wvRuntimeInstalled 检查 WebView2 运行时目录。
func wvRuntimeInstalled() bool {
	roots := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "EdgeWebView", "Application"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "EdgeWebView", "Application"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "EdgeWebView", "Application"),
	}
	for _, r := range roots {
		if r == "" || strings.HasPrefix(r, "Microsoft") {
			continue
		}
		if ents, err := os.ReadDir(r); err == nil && len(ents) > 0 {
			return true
		}
	}
	return false
}

// defaultUserDataDir 默认用户数据目录。
func defaultUserDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "GoGUI", "WebView2")
}

// wvEnsureEnv 启动（或复用）浏览器环境；完成后自动处理排队的创建请求。
func wvEnsureEnv(dataDir string) {
	wvEnvMu.Lock()
	if wvEnvObj != 0 || wvEnvStart {
		wvEnvMu.Unlock()
		return
	}
	wvEnvStart = true
	if dataDir == "" {
		dataDir = defaultUserDataDir()
	}
	wvEnvMu.Unlock()

	if err := wvLoad(); err != nil {
		wvFailEnv(err)
		return
	}
	// UI 线程必须为 STA（OleInitialize）；重复初始化返回 S_FALSE 可忽略。
	if wvCoInitFn != nil {
		if hr := wvCoInitFn(0); hr != 0 {
			dbg("OleInitialize hr=0x%08x", uint32(hr))
		}
	}

	dirPtr, err := syscall.UTF16PtrFromString(dataDir)
	if err != nil {
		wvFailEnv(err)
		return
	}
	handler := newWVCallback(func(this, hr, env uintptr) uintptr {
		if comFail(uint32(hr)) || env == 0 {
			wvFailEnv(fmt.Errorf("WebView2 环境创建失败（0x%08x）：请确认已安装 WebView2 运行时", uint32(hr)))
			return 0
		}
		wvEnvMu.Lock()
		wvEnvObj = env
		queue := wvEnvQueue
		wvEnvQueue = nil
		wvEnvMu.Unlock()
		dbg("WebView2 环境就绪 env=%#x 待创建=%d", env, len(queue))
		for _, wv := range queue {
			wv.createController()
		}
		return 0
	})
	hr := wvCreateEnvFn(0, uintptr(unsafe.Pointer(dirPtr)), 0, handler)
	if Debug {
		dbg("CreateCoreWebView2EnvironmentWithOptions hr=0x%08x", uint32(hr))
	}
	runtime.KeepAlive(dirPtr)
	if comFail(uint32(hr)) {
		dropWVCallback(handler)
		wvFailEnv(fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions 失败（0x%08x）", uint32(hr)))
	}
}

// wvFailEnv 记录环境创建失败并通知所有排队实例。
func wvFailEnv(err error) {
	wvEnvMu.Lock()
	wvEnvErr = err
	queue := wvEnvQueue
	wvEnvQueue = nil
	wvEnvStart = false
	wvEnvMu.Unlock()
	for _, wv := range queue {
		wv.setErr(err)
	}
	dbg("WebView2 不可用: %v", err)
}

// NewWebView2 创建一个内嵌浏览器（异步就绪：用 Ready() 轮询状态）。
func NewWebView2(opts WebView2Options) (*WebView2, error) {
	if opts.Parent == 0 {
		return nil, errors.New("WebView2: 缺少宿主窗口句柄")
	}
	if err := wvLoad(); err != nil {
		return nil, err
	}
	wv := &WebView2{parent: opts.Parent, opts: opts}
	wv.href.Store("")
	wv.title.Store("")

	wvEnvMu.Lock()
	envReady := wvEnvObj != 0
	envErr := wvEnvErr
	wvEnvMu.Unlock()

	if envErr != nil {
		return nil, envErr
	}
	if envReady {
		wv.createController()
		return wv, nil
	}
	wvEnvMu.Lock()
	wvEnvQueue = append(wvEnvQueue, wv)
	wvEnvMu.Unlock()
	wvEnsureEnv(opts.UserDataDir)
	return wv, nil
}

// setErr 记录失败原因。
func (wv *WebView2) setErr(err error) {
	wv.mu.Lock()
	wv.lastErr = err
	wv.mu.Unlock()
	dbg("WebView2: %v", err)
	if wv.opts.OnChange != nil {
		wv.opts.OnChange()
	}
}

// createController 在环境就绪后创建控制器（异步）。
func (wv *WebView2) createController() {
	wvEnvMu.Lock()
	env := wvEnvObj
	wvEnvMu.Unlock()
	if env == 0 {
		return
	}
	wv.mu.Lock()
	closed := wv.closed
	wv.mu.Unlock()
	if closed {
		return
	}
	if wv.opts.CompositionRoot != 0 && wv.startComposition(env) {
		return
	}
	handler := newWVCallback(func(this, hr, ctrl uintptr) uintptr {
		if comFail(uint32(hr)) || ctrl == 0 {
			wv.setErr(fmt.Errorf("创建 WebView2 控制器失败（0x%08x）", uint32(hr)))
			return 0
		}
		wv.attach(ctrl)
		return 0
	})
	wv.mu.Lock()
	wv.callbacks = append(wv.callbacks, handler)
	wv.mu.Unlock()
	if hr := comCall(env, wvEnvCreateController, wv.parent, handler); comFail(hr) {
		wv.setErr(fmt.Errorf("CreateCoreWebView2Controller 调用失败（0x%08x）", hr))
	}
}

// attach 控制器创建完成后接入：背景 / 设置 / 事件 / 初始页面 / 待应用布局。
func (wv *WebView2) attach(ctrl uintptr) {
	wv.mu.Lock()
	if wv.closed {
		wv.mu.Unlock()
		comRelease(ctrl)
		return
	}
	wv.ctrl = ctrl
	wantBounds, hasBounds, wantVisible := wv.wantBounds, wv.hasBounds, wv.wantVisible
	opts := wv.opts
	wv.mu.Unlock()

	wv.ctrl2 = wv.queryInterface(ctrl, wvIIDCtrl2)

	var core uintptr
	if hr := comCall(ctrl, wvCtrlGetCore, uintptr(unsafe.Pointer(&core))); comFail(hr) || core == 0 {
		wv.setErr(fmt.Errorf("取 ICoreWebView2 失败（0x%08x）", hr))
		return
	}
	wv.mu.Lock()
	wv.core = core
	wv.mu.Unlock()

	// 透明背景：COREWEBVIEW2_COLOR{A=0,R=255,G=255,B=255} 按 4 字节结构体传参
	if wv.ctrl2 != 0 && opts.Transparent {
		comCall(wv.ctrl2, wvCtrl2PutBackdrop, uintptr(0xFFFFFF00))
	}
	// 设置项：关掉内置状态栏，保留右键菜单（方便调试）
	if hr := comCall(core, wvCoreGetSettings, uintptr(unsafe.Pointer(&wv.settng))); !comFail(hr) && wv.settng != 0 {
		comCall(wv.settng, wvSetPutStatusBar, 0)
		comCall(wv.settng, wvSetPutCtxMenu, 1)
	}

	// 事件：加载开始 / 完成 / 地址 / 标题 / 历史
	wv.addEvent(wvCoreAddNavStart)
	wv.addEvent(wvCoreAddNavDone)
	wv.addEvent(wvCoreAddSourceChg)
	wv.addEvent(wvCoreAddTitleChg)
	wv.addEvent(wvCoreAddHistoryChg)

	wv.mu.Lock()
	wv.ready = true
	wv.mu.Unlock()

	comCall(ctrl, wvCtrlPutVisible, boolPtr(wantVisible))
	if hasBounds {
		wv.applyBounds(wantBounds)
	}
	// 缩放（double 参数不能走 SyscallN，改用 CSS zoom）
	if opts.Zoom > 0 && opts.Zoom != 1 {
		wv.SetZoom(opts.Zoom)
	}

	switch {
	case opts.HTML != "":
		wv.LoadHTML(opts.HTML)
	case opts.URL != "":
		wv.Navigate(opts.URL)
	}
	wv.dirty.Store(true)
	if opts.OnChange != nil {
		opts.OnChange()
	}
}

// queryInterface 取可选接口（失败返回 0）。
func (wv *WebView2) queryInterface(obj uintptr, iid GUID) uintptr {
	var out uintptr
	if hr := comCall(obj, 0, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&out))); comFail(hr) {
		return 0
	}
	return out
}

// addEvent 注册一个事件回调（回调里只置脏标记 + 通知，具体状态在 Refresh 里读）。
func (wv *WebView2) addEvent(slot int) {
	cb := newWVCallback(func(this, sender, args uintptr) uintptr {
		switch slot {
		case wvCoreAddNavStart:
			wv.loading.Store(true)
		case wvCoreAddNavDone:
			wv.loading.Store(false)
		}
		wv.dirty.Store(true)
		if wv.opts.OnChange != nil {
			wv.opts.OnChange()
		}
		return 0
	})
	wv.mu.Lock()
	wv.callbacks = append(wv.callbacks, cb)
	core := wv.core
	wv.mu.Unlock()
	if core == 0 {
		return
	}
	var token int64
	_ = comCall(core, slot, cb, uintptr(unsafe.Pointer(&token)))
}

// Ready 控制器是否已就绪。
func (wv *WebView2) Ready() bool {
	if wv == nil {
		return false
	}
	wv.mu.Lock()
	defer wv.mu.Unlock()
	return wv.ready && !wv.closed
}

// Err 创建过程中的错误（无错返回 nil）。
func (wv *WebView2) Err() error {
	if wv == nil {
		return nil
	}
	wv.mu.Lock()
	defer wv.mu.Unlock()
	return wv.lastErr
}

// SetBounds 设置网页窗口矩形（**物理像素**，相对宿主客户区左上角）。
func (wv *WebView2) SetBounds(x, y, w, h int32) {
	if wv == nil {
		return
	}
	if w <= 0 || h <= 0 {
		wv.SetVisible(false)
		return
	}
	r := RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
	wv.mu.Lock()
	ctrl := wv.ctrl
	changed := !wv.hasBounds || wv.wantBounds != r
	wv.wantBounds = r
	wv.hasBounds = true
	wv.mu.Unlock()
	if ctrl == 0 || !changed {
		return
	}
	wv.applyBounds(r)
}

func (wv *WebView2) applyBounds(r RECT) {
	wv.mu.Lock()
	ctrl := wv.ctrl
	wv.mu.Unlock()
	if ctrl == 0 {
		return
	}
	comCall(ctrl, wvCtrlPutBounds, uintptr(unsafe.Pointer(&r)))
	// 宿主窗口移动 / DPI 变化时通知（下拉框、输入法弹层依赖它）
	comCall(ctrl, wvCtrlNotifyPos)
}

// SetVisible 显示 / 隐藏网页窗口（隐藏后不再绘制，页面保持存活）。
func (wv *WebView2) SetVisible(v bool) {
	if wv == nil {
		return
	}
	wv.mu.Lock()
	changed := wv.wantVisible != v
	wv.wantVisible = v
	ctrl := wv.ctrl
	wv.mu.Unlock()
	if ctrl == 0 || !changed {
		return
	}
	comCall(ctrl, wvCtrlPutVisible, boolPtr(v))
}

// SetZoom 缩放（1 = 100%）：注入 CSS zoom（接口的 put_ZoomFactor 是 double 参数，SyscallN 传不了）。
func (wv *WebView2) SetZoom(z float64) {
	if wv == nil || z <= 0 {
		return
	}
	script := fmt.Sprintf("document.documentElement.style.zoom='%g'", z)
	wv.ExecuteScript(script)
}

// ExecuteScript 在页面里执行一段 JS（异步返回，不等待结果）。
func (wv *WebView2) ExecuteScript(js string) {
	if wv == nil || js == "" {
		return
	}
	s, err := syscall.UTF16PtrFromString(js)
	if err != nil {
		return
	}
	wv.mu.Lock()
	core := wv.core
	wv.mu.Unlock()
	if core == 0 {
		return
	}
	comCall(core, wvCoreExecuteScript, uintptr(unsafe.Pointer(s)), 0)
	runtime.KeepAlive(s)
}

// Focus 把焦点交给网页（点完地址栏回到网页时用）。
func (wv *WebView2) Focus() {
	wv.mu.Lock()
	ctrl := wv.ctrl
	wv.mu.Unlock()
	if ctrl != 0 {
		comCall(ctrl, wvCtrlMoveFocus, 1) // COREWEBVIEW2_MOVE_FOCUS_REASON_PROGRAMMATIC
	}
}

// Navigate 打开地址。
func (wv *WebView2) Navigate(url string) {
	if wv == nil || strings.TrimSpace(url) == "" {
		return
	}
	u, err := syscall.UTF16PtrFromString(strings.TrimSpace(url))
	if err != nil {
		return
	}
	wv.mu.Lock()
	core := wv.core
	wv.mu.Unlock()
	if core == 0 {
		return
	}
	wv.loading.Store(true)
	wv.dirty.Store(true)
	comCall(core, wvCoreNavigate, uintptr(unsafe.Pointer(u)))
	runtime.KeepAlive(u)
}

// LoadHTML 直接加载一段 HTML（不需要网络）。
func (wv *WebView2) LoadHTML(html string) {
	if wv == nil || html == "" {
		return
	}
	h, err := syscall.UTF16PtrFromString(html)
	if err != nil {
		return
	}
	wv.mu.Lock()
	core := wv.core
	wv.mu.Unlock()
	if core == 0 {
		return
	}
	wv.loading.Store(true)
	wv.dirty.Store(true)
	comCall(core, wvCoreNavigateHTML, uintptr(unsafe.Pointer(h)))
	runtime.KeepAlive(h)
}

// Back / Forward / Reload / StopLoad 导航控制。
func (wv *WebView2) Back() { wv.callCore(wvCoreGoBack) }

func (wv *WebView2) Forward() { wv.callCore(wvCoreGoForward) }

func (wv *WebView2) Reload() { wv.callCore(wvCoreReload) }

func (wv *WebView2) StopLoad() { wv.callCore(wvCoreStop) }

func (wv *WebView2) callCore(slot int) {
	if wv == nil {
		return
	}
	wv.mu.Lock()
	core := wv.core
	wv.mu.Unlock()
	if core != 0 {
		comCall(core, slot)
	}
}

// Refresh 把事件置脏的状态从接口读回（每帧最多一次）。
func (wv *WebView2) Refresh() {
	if wv == nil || !wv.dirty.Swap(false) {
		return
	}
	wv.mu.Lock()
	core := wv.core
	settng := wv.settng
	wv.mu.Unlock()
	if core == 0 {
		return
	}
	_ = settng
	if s, ok := wv.comString(core, wvCoreGetSource); ok {
		wv.href.Store(s)
	}
	if s, ok := wv.comString(core, wvCoreGetTitle); ok {
		wv.title.Store(s)
	}
	wv.canBack.Store(comBool(core, wvCoreCanGoBack))
	wv.canForward.Store(comBool(core, wvCoreCanGoFwd))
}

// comString 读取一个返回 LPWSTR 的属性并释放字符串内存。
func (wv *WebView2) comString(obj uintptr, slot int) (string, bool) {
	var p uintptr
	if hr := comCall(obj, slot, uintptr(unsafe.Pointer(&p))); comFail(hr) || p == 0 {
		return "", false
	}
	defer CoTaskMemFree(p)
	return utf16PtrToString(p), true
}

func comBool(obj uintptr, slot int) bool {
	var v int32
	if hr := comCall(obj, slot, uintptr(unsafe.Pointer(&v))); comFail(hr) {
		return false
	}
	return v != 0
}

func utf16PtrToString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for n < 1<<20 {
		if *(*uint16)(unsafe.Pointer(p + uintptr(n)*2)) == 0 {
			break
		}
		n++
	}
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(p)), n))
}

// URL 当前地址。
func (wv *WebView2) URL() string {
	if wv == nil {
		return ""
	}
	if s, ok := wv.href.Load().(string); ok {
		return s
	}
	return ""
}

// Title 当前页面标题。
func (wv *WebView2) Title() string {
	if wv == nil {
		return ""
	}
	if s, ok := wv.title.Load().(string); ok {
		return s
	}
	return ""
}

// Loading 是否正在加载。
func (wv *WebView2) Loading() bool { return wv != nil && wv.loading.Load() }

// CanGoBack 历史状态。
func (wv *WebView2) CanGoBack() bool { return wv != nil && wv.canBack.Load() }

// CanGoForward 历史状态。
func (wv *WebView2) CanGoForward() bool { return wv != nil && wv.canForward.Load() }

// Close 关闭网页并释放接口（必须在 UI 线程调用）。
func (wv *WebView2) Close() {
	if wv == nil {
		return
	}
	wv.mu.Lock()
	if wv.closed {
		wv.mu.Unlock()
		return
	}
	wv.closed = true
	ctrl, core, settng := wv.ctrl, wv.core, wv.settng
	callbacks := wv.callbacks
	wv.ctrl, wv.ctrl2, wv.core, wv.settng = 0, 0, 0, 0
	wv.callbacks = nil
	wv.ready = false
	wv.mu.Unlock()

	// 释放顺序与去重同样重要：
	//   - QueryInterface 取到的 Controller2 与控制器是同一个对象（COM 恒等性），
	//     重复 Release 会二次释放导致崩溃，因此同地址只释放一次；
	//   - 最后才 Close + Release 控制器（Close 后内部对象失效）。
	for _, p := range []uintptr{settng, core} {
		if p != 0 && p != ctrl {
			comRelease(p)
		}
	}
	if ctrl != 0 {
		comCall(ctrl, wvCtrlClose)
		comRelease(ctrl)
	}
	for _, cb := range callbacks {
		dropWVCallback(cb)
	}
}

// ---- 内层窗口（Chrome_WidgetWin_*）----
//
// WebView2 的 HWND 托管会在宿主里生成一个 Chromium 子窗口，网页就画在它上面。
// 对它 SetWindowRgn 可以得到真正的圆角裁剪（不是盖一层背景遮出来的伪圆角）。
type wvInner struct {
	hwnd uintptr
	w, h int32
	radius int32
}

var (
	user32Lazy   = syscall.NewLazyDLL("user32.dll")
	procEnumChild = user32Lazy.NewProc("EnumChildWindows")
	procGetClass   = user32Lazy.NewProc("GetClassNameW")
	procGetClient  = user32Lazy.NewProc("GetClientRect")
	procSetRgn     = user32Lazy.NewProc("SetWindowRgn")
	procDeleteObj  = user32Lazy.NewProc("DeleteObject")
)

// findInnerWindow 在宿主窗口里找 Chromium 的绘图子窗口（类名含 Chrome_WidgetWin）。
func findInnerWindow(parent uintptr) uintptr {
	var found uintptr
	cb := purego.NewCallback(func(hwnd, lparam uintptr) uintptr {
		var buf [128]uint16
		n, _, _ := procGetClass.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n > 0 {
			name := syscall.UTF16ToString(buf[:n])
			if strings.Contains(strings.ToLower(name), "chrome_widgetwin") {
				found = hwnd
				return 0 // 停止枚举
			}
		}
		return 1
	})
	procEnumChild.Call(parent, cb, 0)
	return found
}

// SetRoundedRegion 让网页窗口按圆角区域裁剪（radius <= 0 取消裁剪）。
// 区域尺寸取自网页子窗口的客户区，窗口缩放后重调一次即可。
func (wv *WebView2) SetRoundedRegion(radius int32) {
	if wv == nil {
		return
	}
	wv.mu.Lock()
	inner := wv.inner
	wv.mu.Unlock()
	if inner == 0 {
		inner = findInnerWindow(wv.parent)
		if inner == 0 {
			return
		}
		wv.mu.Lock()
		wv.inner = inner
		wv.mu.Unlock()
	}
	var r RECT
	procGetClient.Call(inner, uintptr(unsafe.Pointer(&r)))
	cw, ch := r.Right-r.Left, r.Bottom-r.Top
	if cw <= 0 || ch <= 0 {
		return
	}
	wv.mu.Lock()
	same := wv.innerW == cw && wv.innerH == ch && wv.innerRadius == radius
	if !same {
		wv.innerW, wv.innerH, wv.innerRadius = cw, ch, radius
	}
	wv.mu.Unlock()
	if same {
		return
	}
	// 仅当页面需要圆角时才去查子窗口；全透明背景时 Chromium 子窗口可能不存在
	if radius <= 0 {
		procSetRgn.Call(inner, 0, 1)
		return
	}
	rgn := CreateRoundRectRgn(0, 0, cw+1, ch+1, radius*2, radius*2)
	if rgn != 0 {
		procSetRgn.Call(inner, rgn, 1)
	}
}

// InnerHwnd 网页绘图子窗口句柄（Chromium 窗口；未就绪时为 0）。
func (wv *WebView2) InnerHwnd() uintptr {
	if wv == nil {
		return 0
	}
	wv.mu.Lock()
	h := wv.inner
	wv.mu.Unlock()
	if h == 0 {
		h = findInnerWindow(wv.parent)
		if h != 0 {
			wv.mu.Lock()
			wv.inner = h
			wv.mu.Unlock()
		}
	}
	return h
}

// boolPtr 布尔转 COM BOOL。
func boolPtr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

// startComposition 创建「合成控制器」并把网页挂到指定 DComp 视觉（B 阶段）。
// 失败时返回 false，调用方回退 HWND 托管（A 阶段）。
func (wv *WebView2) startComposition(env uintptr) bool {
iidEnv3 := parseGUID("80a22ae3-be7c-4ce2-afe1-5a50056cdeeb")
var env3 uintptr
if hr := comCall(env, 0, uintptr(unsafe.Pointer(&iidEnv3)), uintptr(unsafe.Pointer(&env3))); comFail(hr) || env3 == 0 {
return false
}
handler := newWVCallback(func(this, hr, comp uintptr) uintptr {
if comFail(uint32(hr)) || comp == 0 {
wv.setErr(fmt.Errorf("创建合成控制器失败（0x%08x），回退 A 阶段", uint32(hr)))
return 0
}
wv.attachComposition(comp)
return 0
})
wv.mu.Lock()
wv.callbacks = append(wv.callbacks, handler)
wv.mu.Unlock()
if hr := comCall(env3, wvEnvCompCreate, wv.parent, handler); comFail(hr) {
return false
}
return true
}

// attachComposition 合成控制器就绪：设置 RootVisualTarget，取 UiController 后复用通用接入流程。
func (wv *WebView2) attachComposition(comp uintptr) {
root := wv.opts.CompositionRoot
if hr := comCall(comp, wvCompPutRoot, root); comFail(hr) {
wv.setErr(fmt.Errorf("put_RootVisualTarget 失败（0x%08x）", hr))
return
}
wv.mu.Lock()
wv.comp, wv.compRoot = comp, root
wv.mu.Unlock()

// ICoreWebView2CompositionController2::get_UiController
iidCC2 := parseGUID("0b6a3d24-49cb-4806-ba20-b5e0734a7b26")
var cc2 uintptr
if hr := comCall(comp, 0, uintptr(unsafe.Pointer(&iidCC2)), uintptr(unsafe.Pointer(&cc2))); comFail(hr) || cc2 == 0 {
wv.setErr(fmt.Errorf("取 CompositionController2 失败（0x%08x）", hr))
return
}
var ui uintptr
if hr := comCall(cc2, wvCompGetUI, uintptr(unsafe.Pointer(&ui))); comFail(hr) || ui == 0 {
wv.setErr(fmt.Errorf("取 UiController 失败（0x%08x）", hr))
return
}
comRelease(cc2)
wv.attach(ui)
}

// SendMouseInput 把鼠标事件转发给网页（B 阶段没有子窗口，必须自己转）。
// kind: 0=HOVER 1=LEAVE 2=LEFTDOWN 3=LEFTUP 4=RIGHTDOWN 5=RIGHTUP 6=WHEEL
func (wv *WebView2) SendMouseInput(kind int, vkeys uint32, wheel int32, x, y int32) {
if wv == nil {
return
}
wv.mu.Lock()
comp := wv.comp
wv.mu.Unlock()
if comp == 0 {
return
}
type pt32 struct{ X, Y int32 }
	p := pt32{X: x, Y: y}
comCall(comp, wvCompSendMouse, uintptr(kind), uintptr(vkeys), uintptr(uint32(wheel)), uintptr(unsafe.Pointer(&p)))
}