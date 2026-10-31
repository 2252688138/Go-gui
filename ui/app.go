// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gogui/internal/geom"
	"gogui/internal/gpu"
	"gogui/internal/text"
)

// Options 启动参数。
type Options struct {
	Title      string
	Width      int
	Height     int
	FontFamily string
	Background any
	Resizable  bool
	// DumpLayout 非空时首帧后将布局树写入该文件并退出（排查布局问题）。
	DumpLayout string
	// Hidden 创建后不显示窗口（配合 Tray 常驻托盘）。
	Hidden bool
	// NoTitleBar 去掉系统标题栏（自绘标题栏场景）。
	NoTitleBar bool
	// CornerRadius 窗口圆角半径（逻辑像素；0 = 系统默认，>0 = 圆角）。
	// Win11 走 DWM 原生圆角，低版本/无 DWM 时回退 SetWindowRgn 区域圆角。
	CornerRadius float32
	// Icon 窗口/任务栏图标（.ico 字节）；为空时用内置默认图标。
	// 需要 exe 文件图标时，用 rsrc 类工具把同一 .ico 嵌入可执行文件。
	Icon []byte
}

// App 一个窗口的应用实例（UI 线程独占）。
type App struct {
	win   Backend
	r     *gpu.Renderer
	fonts *fontCache
	opts  Options
	view  func() *Node

	root    *Node
	dirty   bool
	running bool
	closed  bool

	mouseX, mouseY float32
	mouseIn        bool

	pressX, pressY float32
	pressed        bool
	clickPending   bool
	clickX, clickY float32

	// redrawRequest 允许业务回调在任意线程请求重绘
	redrawRequest atomic.Bool
	// hoverProbe 仅指针位置变化（未确定是否需要重绘外观）时置位
	hoverProbe bool

	hoverNode *Node
	pressNode *Node
	// pressFired 记录本次按下已分发过 OnPress 的节点（每次松开重置）
	pressFired *Node
	scrolls    map[string]float32

	bg gpu.Paint

	onKey  func(uint32)
	onChar func(rune)

	frameCount int

	// 键盘焦点（输入框）与逐帧动画
	focusKey  string
	dragKey   string
	hoverKey  string
	startTime time.Time
	animating bool
	// animFast 本帧存在需要高帧率的动画节点（未标记 AnimSlow）；
	// 只有低帧率动画时不降帧会持续吃满一个核。
	animFast bool
	// slowOnlyFrame 上一帧的重绘只由装饰动画（光标闪烁 / shimmer）驱动。
	// runLoop 据此把整树重建+绘制节流到 8–12Hz，而消息泵仍跑 120Hz。
	slowOnlyFrame bool
	// animDirty 动画续帧标记（可被装饰动画节流；dirty 则必须立即绘制）。
	animDirty bool
	// dbgAnim 本帧参与持续重绘的动画节点（GOGUI_FPS=1 时输出，定位 CPU 用）。
	dbgAnim []string
	// dbgBusy 上一帧结束时 animBusy 的取值（诊断用）。
	dbgBusy bool
	// dbgViewMs / dbgLayMs / dbgPaintMs 上一帧各阶段耗时（ms，诊断用）。
	dbgViewMs, dbgLayMs, dbgPaintMs float64

	// 图片纹理缓存
	textures *imageTextures
	// 子菜单指示三角纹理（懒创建，随窗口销毁）
	arrowTex *gpu.Texture
	// 图表矢量层纹理（每张图表一张，总预算 chartTexBudget，按 LRU 淘汰）
	chartList  []*chartCache
	chartBytes int64
	chartStamp int64
	// 视频帧纹理缓存（每个播放器一份，随窗口销毁；按帧序号做 LRU 回收）
	videos   map[*Player]*videoTexture
	videoSeq int

	// 内嵌浏览器（WebView2）：本帧绘制到的矩形与已显示集合，用于同步子窗口
	webviews map[*WebView]bool
	wvRects  map[*WebView]geom.Rect
	wvShown  map[*WebView]bool

	// IME 组合串（当前窗口内的输入法预览）
	imeText  string
	imeCaret int
	imeX,
	imeY int32

	// 文件拖入：WM_DROPFILES 的路径与落点，延迟到下一帧（节点树有效时）分发
	dropPending  bool
	dropPaths    []string
	dropX, dropY float32
	// onDrop 应用级拖入回调（没有节点设置 OnDrop 时兑底）
	onDrop func(paths []string, x, y float32)

	// 滚动：当前偏移（scrolls）与平滑滚动目标偏移（scrollTargets）
	scrollTargets map[string]float32
	// 滚动条：悬停 / 拖动 / 淡出
	scrollHoverKey  string
	scrollDragKey   string
	scrollDragY     float32
	scrollDragStart float32
	// 拖动开始时缓存的滑块几何：指针拖出 12px 热区后仍能继续跟手
	scrollDragTrack geom.Rect
	scrollDragThumb float32
	scrollDragMax   float32
	// scrollIdle 距上次滚动/悬停的秒数（滚动条淡出用）
	scrollIdle float32
	// 上一帧时间戳与帧间隔（滚动缓动/淡出推进）
	lastFrame time.Time
	animDt    float32

	// lastPaint 最近一次真正绘制的时刻（空闲回收缓存用）
	lastPaint time.Time
	// trimmed 本段空闲期是否已释放过可重建的 GPU 缓存（恢复绘制时清零）
	trimmed bool
	// trimmedDeep 本段空闲期是否已做过「工作集级」回收（更激进，代价是重新调页）
	trimmedDeep bool

	// 本帧裁剪栈（用于去重：子项裁剪区与父级完全相同时不重复入栈）
	clipStack []geom.Rect
	clipN     int
	clipSkip  int

	// 窗口关闭回调（返回 false 可阻止关闭，例如收进托盘）
	onClose func() bool
	pending []func()
	// disposed 保证 GL/字体资源只释放一次
	disposed bool
	// minimized 上一帧是否处于最小化（恢复时强制重绘，并重建媒体解码窗口）
	minimized bool
	// sizeMoving 系统正在拖动/缩放窗口（WM_ENTERSIZEMOVE..WM_EXITSIZEMOVE）。
	// 模态循环期间鼠标消息不再更新，滚动条/悬停高亮暂停绘制，
	// 否则窗边会残留一条静止的「假滚动条」。
	sizeMoving bool

	// shiftDown Shift 键当前是否按下（Shift+Tab 反向遍历用）。
	shiftDown bool
	// ctrlDown / altDown 修饰键状态（命令面板 Ctrl+K、快捷键匹配用）。
	ctrlDown bool
	altDown  bool
}

// Run 阻塞运行应用（必须在 main goroutine）。
//
// 平台差异全部收敛在 prepareRun / reportPanic（见 boot_wasm.go / boot_desktop.go）：
// Web 端会在建窗口前 fetch 字体并注册，桌面端什么都不做。
// 因此业务代码在所有平台只需要写一个 main()、一份 Options。
func Run(opts Options, view func() *Node) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			rep := ReportCrash(r, stack)
			msg := fmt.Sprintf("panic: %v\n%s", r, stack)
			if rep.File != "" {
				msg += "\n崩溃日志: " + rep.File
			}
			reportPanic(msg)
		}
	}()
	prepareRun(&opts)

	a, err := newApp(opts, view)
	if err != nil {
		return err
	}
	_ = a
	err = runLoop()
	if os.Getenv("GOGUI_X11_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[ui] runLoop 返回 err=%v\n", err)
	}
	return err
}

// OpenWindow 在当前 UI 线程再开一个窗口（可选）。
func OpenWindow(opts Options, view func() *Node) (*App, error) {
	return newApp(normalizeOpts(opts), view)
}

func normalizeOpts(opts Options) Options {
	if opts.Title == "" {
		opts.Title = "GoGUI"
	}
	if opts.Width <= 0 {
		opts.Width = 1024
	}
	if opts.Height <= 0 {
		opts.Height = 700
	}
	if opts.FontFamily == "" {
		opts.FontFamily = "Microsoft YaHei UI"
	}
	return opts
}

func newApp(opts Options, view func() *Node) (*App, error) {
	opts = normalizeOpts(opts)

	bgAny := opts.Background
	if bgAny == nil {
		bgAny = "#0b0f19"
	}

	win, err := newBackend(opts)
	if err != nil {
		return nil, err
	}

	a := &App{
		win:           win,
		opts:          opts,
		view:          view,
		dirty:         true,
		running:       true,
		bg:            C(bgAny),
		scrolls:       map[string]float32{},
		scrollTargets: map[string]float32{},
		startTime:     time.Now(),
		lastFrame:     time.Now(),
		lastPaint:     time.Now(),
		textures:      newImageTextures(),
		videos:        map[*Player]*videoTexture{},
	}

	r := gpu.NewRenderer()
	if err := r.Init(); err != nil {
		return nil, err
	}
	a.r = r

	pw, ph, scale, ss := a.renderSize()
	r.SetSuperSample(ss)
	r.Resize(pw, ph, scale)

	// B 阶段（可选）：把呈现切到 DirectComposition，网页与自绘内容同层合成。
	// 环境变量 GOGUI_COMPOSE=0 可强制回退 A 阶段（子窗口叠加）。
	win.AttachCompose(pw, ph)
	a.fonts = newFontCache(opts.FontFamily, scale)
	// 字体被 LRU 回收时同步释放图集纹理，否则显存只增不减
	a.fonts.onEvict = func(f *text.Font) { r.ReleaseFont(f) }

	win.SetEventHandler(a.handleEvent)
	registerApp(a)
	if opts.Hidden {
		win.SetVisible(false)
	}
	return a, nil
}

// runLoop 统一消息泵与多窗口渲染循环。
// fpsDebug：GOGUI_FPS=1 时每秒输出真实帧率与重绘原因，用于定位常驻重绘源。
var fpsDebug = os.Getenv("GOGUI_FPS") != ""

// dbgRedrawFrom 最近一次 RequestRedraw 的调用栈（诊断用）。
var dbgRedrawFrom atomic.Value

// fpsOut 诊断输出：不指定 GOGUI_FPS_FILE 时用 stderr；
// 指定后追加到文件（GUI 子进程重定向 stderr 时不会卡住调用方）。
func fpsOut(s string) {
	if p := os.Getenv("GOGUI_FPS_FILE"); p != "" {
		if f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			_, _ = f.WriteString(s)
			_ = f.Close()
			return
		}
	}
	fmt.Fprint(os.Stderr, s)
}

// msSince 返回自 t 起的毫秒数（诊断用）。
func msSince(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000 }

func runLoop() error {
	last := time.Now()
	// 复用窗口列表切片：旧实现每轮循环（即使完全空闲，约 8ms 一次）都用 snapshotApps()
	// 复制一份，持续产生垃圾；这里改成原地复用，容量不足时才扩容。
	list := make([]*App, 0, 4)
	// GOGUI_FPS=1 时每秒打印帧率/重绘原因（见 fpsDebug）；这里只维护计数器。
	dbgLast := time.Now()
	dbgPainted, dbgSkipped := 0, 0
	var dbgApp *App
	var dbgGate string
	for {
		if !pollMessages() {
			break
		}
		list = appsInto(list[:0])
		if len(list) == 0 {
			break
		}
		// 上一帧存在动画时本帧继续重绘所有窗口
		needAnim := animBusy.Swap(false)
		painted := false
		// slowOnlyFrame 由 frame() 根据本帧绘制内容标记：只有装饰动画时按 8–12Hz 节流。
		for _, a := range list {
			if a == nil {
				continue
			}
			if a.closed {
				a.dispose()
				continue
			}
			// 关闭/单帧导出（DumpLayout）后释放该窗口
			if !a.running {
				a.Dispose()
				continue
			}
			// 帧间隔（滚动缓动/淡出推进用）
			now := time.Now()
			dt := float32(now.Sub(a.lastFrame).Seconds())
			a.lastFrame = now
			if dt <= 0 || dt > 0.25 {
				dt = 1.0 / 120
			}
			a.animDt = dt
			// 从最小化恢复：WM_SIZE 在尺寸未变时不会带来 EvResize，这里强制重绘一帧；
			// 并让媒体源重建视频解码窗口——最小化期间 UI 不再取帧，解码器早已被
			// 帧环/管道阻塞，不重建就会一直画着最小化前那一帧不动。
			if a.minimized {
				a.minimized = false
				a.lastFrame = now
				a.dirty = true
				ResumeMediaAll()
			}
			// 窗口最小化时跳过渲染（不可见的帧不消耗 GPU/CPU）
			if a.win != nil && a.win.IsMinimized() {
				a.minimized = true
				a.dirty = false
				a.redrawRequest.Store(false)
				// 最小化期间先把可重建的显存与物理页都让出来
				if !a.trimmed {
					a.trimCaches(true)
					a.trimmed, a.trimmedDeep = true, true
				}
				continue
			}
			// 只有装饰动画在驱动时按时间间隔节流：消息泵仍跑 120Hz（输入不掉响应），
			// 但整树重建+排版+绘制降到 forbid 8–12Hz。
			// 先无条件取走“外部请求重绘”标志：
			// 旧写法把它放在 `||` 链里，前面的 animDirty 为真时会被短路，
			// 标志永远清不掉 → 闸门每帧都打开（节流形同虚设）。
			reqRedraw := a.redrawRequest.Swap(false)
			due := true
			if a.slowOnlyFrame && !a.dirty && !needAnim && !a.hoverProbe && !reqRedraw {
				gap := time.Second / 8
				if a.mouseIn {
					gap = time.Second / 12
				}
				due = time.Since(a.lastPaint) >= gap
			}
			if fpsDebug {
				dbgGate = fmt.Sprintf("dirty=%v anim=%v need=%v probe=%v req=%v", a.dirty, a.animDirty, needAnim, a.hoverProbe, reqRedraw)
			}
			if due && (a.dirty || a.animDirty || reqRedraw || needAnim || a.hoverProbe) {
				forced := a.dirty || needAnim || a.hoverProbe
				a.dirty = false
				a.animDirty = false
				painted = true
				dbgPainted++
				dbgApp = a
				a.frame(forced)
			} else if !due {
				dbgSkipped++
			}
			// 空闲回收（两级）：
			//   1) 3 秒无绘制：释放可重建的显存（超采样离屏目标 / 毛玻璃中间层 /
			//      图表与图片纹理）+ 一次 Go 堆回收。这些资源下一帧自动重建，观感不变。
			//   2) 30 秒无绘制且窗口不在前台（或被最小化）：再把驻留的物理页交给系统。
			//      用户正看着窗口时不做，避免重新调页的顿挫。
			idle := time.Since(a.lastPaint)
			if !a.trimmed && idle >= idleTrimAfter {
				a.trimCaches(false)
				a.trimmed = true
			}
			if !a.trimmedDeep && idle >= idleTrimDeepAfter && (a.win == nil || a.win.IsMinimized() || !windowInForeground(a.HWND())) {
				a.trimCaches(true)
				a.trimmedDeep = true
			}
			a.runPending()
		}
		// 自适应休眠：有绘制/动画时维持 120Hz 节奏；整轮没有任何窗口可画时降到 60Hz，
		// 空闲窗口不再以约 125 次/秒的频率空转 PeekMessage + time.Now（降低常驻 CPU）。
		want := time.Second / 120
		if !painted && !needAnim {
			want = time.Second / 60
		}
		// GOGUI_FPS=1：每秒输出一次真实帧率与“为什么在重绘”，新增常驻重绘源时直接定位。
		if fpsDebug && time.Since(dbgLast) >= time.Second {
			var anim []string
			var slowFrame, fastFlag bool
			var dbgView, dbgLay, dbgPaint float64
			var dbgScrollAnim, dbgPressed, dbgClickPend, dbgBusyFlag bool
			var dbgDrag string
			var dbgFonts, dbgAtlasKB int
			if dbgApp != nil {
				if dbgApp.fonts != nil {
					dbgFonts = len(dbgApp.fonts.fonts)
					dbgAtlasKB = dbgApp.fonts.atlasTotal() / 1024
				}
				anim = append([]string(nil), dbgApp.dbgAnim...)
				slowFrame, fastFlag = dbgApp.slowOnlyFrame, dbgApp.animFast
				dbgBusyFlag = dbgApp.dbgBusy
				dbgView, dbgLay, dbgPaint = dbgApp.dbgViewMs, dbgApp.dbgLayMs, dbgApp.dbgPaintMs
				dbgScrollAnim, dbgPressed, dbgClickPend = dbgApp.isScrollAnimating(), dbgApp.pressed, dbgApp.clickPending
				dbgDrag = dbgApp.dragKey + "|" + dbgApp.scrollDragKey
			}
			fpsOut(fmt.Sprintf("[gogui] fps=%.0f painted=%d skipped=%d slowOnly=%v animFast=%v busy=%v view=%.1fms layout=%.1fms paint=%.1fms fonts=%d/%dKB scrollAnim=%v pressed=%v drag=%q clickPend=%v tweens=%v anim=%v\n",
				float64(dbgPainted)/time.Since(dbgLast).Seconds(), dbgPainted, dbgSkipped,
				slowFrame, fastFlag, dbgBusyFlag, dbgView, dbgLay, dbgPaint, dbgFonts, dbgAtlasKB,
				dbgScrollAnim, dbgPressed, dbgDrag, dbgClickPend, activeTweenKeys(), anim))
			fpsOut(fmt.Sprintf("[gogui] gate %s from=%v\n", dbgGate, dbgRedrawFrom.Load()))
			dbgPainted, dbgSkipped, dbgLast = 0, 0, time.Now()
		}
		el := time.Since(last)
		if el < want {
			time.Sleep(want - el)
		}
		last = time.Now()
	}
	for _, a := range snapshotApps() {
		a.dispose()
	}
	appsMu.Lock()
	apps = nil
	appsMu.Unlock()
	currentApp.Store(nil)
	return nil
}

func (a *App) handleEvent(ev Event) {
	currentApp.Store(a)
	switch ev.Type {
	case EvClose:
		if a.onClose != nil && !a.onClose() {
			return
		}
		a.running = false
		a.closed = true
		unregisterApp(a)
	case EvResize:
		a.win.MakeCurrent()
		// 尺寸变化：重建渲染目标（含超采样）、字形缓存与合成表面
		a.applyRenderScale()
	case EvSizeMoveStart:
		// 开始拖动/缩放窗口：隐藏滚动条与悬停高亮
		a.sizeMoving = true
		a.scrollHoverKey = ""
		a.dirty = true
	case EvSizeMoveEnd:
		a.sizeMoving = false
		a.dirty = true
	case EvMouseMove:
		a.mouseX, a.mouseY = ev.X/uiScale, ev.Y/uiScale
		a.mouseIn = true
		// 不直接全帧重绘：先在下一帧算 hover 命中，
		// 只有悬停目标/滚动条悬停真的变化时才重新绘制。
		a.hoverProbe = true
	case EvMouseLeave:
		a.mouseIn = false
		a.dirty = true
	case EvMouseDown:
		a.mouseX, a.mouseY = ev.X/uiScale, ev.Y/uiScale
		a.pressX, a.pressY = a.mouseX, a.mouseY
		a.pressed = true
		a.dirty = true
	case EvMouseUp:
		a.mouseX, a.mouseY = ev.X/uiScale, ev.Y/uiScale
		if a.pressed && a.scrollDragKey == "" {
			// 命中与回调节奏放在下一帧，基于最新的组件树
			a.clickPending = true
			a.clickX, a.clickY = a.mouseX, a.mouseY
		}
		a.pressed = false
		a.scrollDragKey = ""
		a.pressFired = nil
		a.dirty = true
	case EvMouseWheel:
		a.handleWheel(ev.X/uiScale, ev.Y/uiScale, ev.DY)
		a.scrollIdle = 0
		a.dirty = true
	case EvKeyDown:
		if ev.Key == VK_SHIFT {
			a.shiftDown = true
		}
		if ev.Key == VK_CONTROL {
			a.ctrlDown = true
		}
		if ev.Key == VK_MENU {
			a.altDown = true
		}
		// Ctrl+K 命令面板（输入焦点在编辑类控件时也生效，与 VS Code 一致）
		if a.ctrlDown && !a.altDown && (ev.Key == 'K' || ev.Key == 'k') {
			if ToggleCommandPalette() {
				a.dirty = true
				return
			}
		}
		// 应用内快捷键（带修饰键或功能键）：优先于焦点控件，Ctrl+S 这类不会先被输入框吞掉
		if (a.ctrlDown || a.altDown || ev.Key >= 0x70 && ev.Key <= 0x7B) && DispatchShortcuts(ev.Key) {
			a.dirty = true
			return
		}
		// Tab / Shift+Tab 在可聚焦控件间移动
		if ev.Key == VK_TAB {
			if a.shiftDown {
				FocusPrev()
			} else {
				FocusNext()
			}
			a.dirty = true
			return
		}
		// 菜单展开时 ESC 只关闭菜单
		if ev.Key == VK_ESCAPE && closeOpenMenus() {
			a.dirty = true
			return
		}
		// 其它浮层（下拉/日历/命令面板/弹窗）按登记顺序逐层关闭
		if ev.Key == VK_ESCAPE && closeTopOverlay() {
			a.dirty = true
			return
		}
		// 全屏时 ESC 先退出全屏；再有焦点时不直接关窗口（避免输入时误关）
		if ev.Key == VK_ESCAPE && a.win.IsFullscreen() {
			a.SetFullscreen(false)
			return
		}
		if ev.Key == VK_ESCAPE && a.focusKey != "" {
			SetFocus("")
			a.dirty = true
			return
		}
		if ev.Key == VK_ESCAPE && a.focusKey == "" {
			a.running = false
		}
		if n := a.focusedNode(); n != nil && n.OnKey != nil {
			n.OnKey(ev.Key)
			a.dirty = true
		}
		// Enter / Space 激活焦点控件（输入框会自行消费 Enter 提交）
		if ev.Key == VK_RETURN || ev.Key == VK_SPACE {
			if a.activateFocused() {
				a.dirty = true
			}
		}
		if a.onKey != nil {
			a.onKey(ev.Key)
			a.dirty = true
		}
	case EvKeyUp:
		if ev.Key == VK_SHIFT {
			a.shiftDown = false
		}
		if ev.Key == VK_CONTROL {
			a.ctrlDown = false
		}
		if ev.Key == VK_MENU {
			a.altDown = false
		}
	case EvChar:
		a.imeText, a.imeCaret = "", 0
		if n := a.focusedNode(); n != nil && n.OnChar != nil {
			n.OnChar(ev.Char)
			a.dirty = true
		}
		if a.onChar != nil {
			a.onChar(ev.Char)
			a.dirty = true
		}
	case EvCompositionStart:
		a.imeText, a.imeCaret = "", 0
		a.dirty = true
	case EvComposition:
		a.imeText, a.imeCaret = ev.Text, ev.Caret
		a.dirty = true
	case EvCompositionEnd:
		a.imeText, a.imeCaret = "", 0
		a.dirty = true
	case EvDropFiles:
		// 文件拖入：节点树每帧重建，回调必须等到 frame() 里指针有效时再分发
		if len(ev.Paths) == 0 {
			return
		}
		a.dropPending = true
		a.dropPaths = ev.Paths
		a.dropX, a.dropY = ev.X/uiScale, ev.Y/uiScale
		a.dirty = true
	default:
		a.dirty = true
	}
}

// runPending 执行在 UI 线程排队的回调（跨线程打开窗口等）。
func (a *App) runPending() {
	appsMu.Lock()
	if len(a.pending) == 0 {
		appsMu.Unlock()
		return
	}
	list := a.pending
	a.pending = nil
	appsMu.Unlock()
	for _, fn := range list {
		fn()
	}
}

func (a *App) frame(forced bool) {
	if a.closed {
		return
	}
	a.win.MakeCurrent()
	currentApp.Store(a)
	probe := a.hoverProbe
	frameT0 := time.Now()
	a.hoverProbe = false
	// pw/ph 为渲染目标像素（含超高清超采样），es 为含超采样的渲染缩放；
	// 逻辑尺寸 w/h 与超采样无关，布局与命中测试不受影响。
	pw, ph, es, ss := a.renderSize()
	a.r.SetSuperSample(ss)
	a.r.Resize(pw, ph, es)
	w, h := float32(pw)/es, float32(ph)/es

	// composeFBO 在真正开始绘制前才赋值（见下），提前声明以便「无树」分支安全返回
	var composeFBO uint32
	root := a.view()
	if root == nil {
		return
	}
	a.root = root

	// 滚动条淡出计时（悬停/滚动时会在 updateScrollbar 中归零）
	if a.scrollIdle < 1.5 {
		a.scrollIdle += a.animDt
	}
	Layout(root, w, h, a.fonts)
	root.Frame = geom.Rect{X: 0, Y: 0, W: w, H: h}
	if fpsDebug {
		a.dbgViewMs = msSince(frameT0)
		frameT0 = time.Now()
	}

	// 计算 hover 与处理待执行的点击（同一帧内，树指针有效）
	prevHover := a.hoverNode
	a.hoverNode = nil
	a.pressNode = nil
	if a.mouseIn {
		a.hoverNode = hitInteractive(root, a.mouseX, a.mouseY, 0, a)
	}
	// hover 回调按 Key 去重，避免每帧重复触发
	hk := ""
	if a.hoverNode != nil {
		a.hoverNode.Hover = true
		hk = a.hoverNode.Key
	}
	hoverChanged := hk != a.hoverKey
	if hoverChanged {
		if prevHover != nil && prevHover.OnHover != nil {
			prevHover.OnHover(false)
		}
		if a.hoverNode != nil && a.hoverNode.OnHover != nil {
			a.hoverNode.OnHover(true)
		}
		a.hoverKey = hk
	}

	// 滚动条悬停/拖动
	scrollBefore := a.scrollHoverKey
	a.updateScrollbar(root)
	scrollHoverChanged := scrollBefore != a.scrollHoverKey

	if a.pressed {
		a.pressNode = hitInteractive(root, a.pressX, a.pressY, 0, a)
		if a.pressNode != nil {
			a.pressNode.Pressed = true
			// OnPress：按下后的第一帧只分发一次
			if a.pressNode.OnPress != nil && a.pressNode != a.pressFired {
				a.pressFired = a.pressNode
				a.pressNode.OnPress()
			}
		}
		if a.dragKey == "" && a.pressNode != nil && a.pressNode.OnDrag != nil && a.pressNode.Key != "" && a.scrollDragKey == "" {
			a.dragKey = a.pressNode.Key
		}
	} else if a.dragKey != "" {
		// 松手：先通知拖动节点提交（gio 里对应 Clickable.Pressed() 变 false 的那一帧），再清空 dragKey
		if n := findKey(root, a.dragKey); n != nil && n.OnDragEnd != nil {
			n.OnDragEnd()
		}
		a.dragKey = ""
		// 松手后指针下的节点可能已不是拖动节点，避免它的 Pressed 残留
		a.dirty = true
	}
	if a.dragKey != "" {
		if n := findKey(root, a.dragKey); n != nil && n.OnDrag != nil {
			ax, ay := a.visualOrigin(n, 0, 0)
			n.OnDrag(a.mouseX-ax, a.mouseY-ay)
			a.dirty = true
		}
	}

	// 平滑滚动推进（滚轮/轨道点击产生的目标位）
	a.stepScrollAnim()

	if a.clickPending {
		a.clickPending = false
		if n := hitInteractive(root, a.clickX, a.clickY, 0, a); n != nil {
			// 点击决定焦点：可聚焦节点取得焦点，否则清除焦点
			if n.Style.Focusable && n.Key != "" {
				a.focusKey = n.Key
			} else {
				a.focusKey = ""
			}
			focusVisible = false
			ax, ay := a.visualOrigin(n, 0, 0)
			// 富文本：先分发行内片段点击（链接）
			if n.Kind == KRichText && n.OnSpanClick != nil {
				if sp, ok := richAt(n, a.clickX-ax-n.Style.Pad.Left-n.Style.DrawX, a.clickY-ay-n.Style.Pad.Top-n.Style.DrawY); ok {
					n.OnSpanClick(sp)
					a.dirty = true
				}
			}
			if n.OnClickAt != nil || n.OnClick != nil {
				if n.OnClickAt != nil {
					n.OnClickAt(a.clickX-ax, a.clickY-ay)
				}
				if n.OnClick != nil {
					n.OnClick()
				}
				a.dirty = true
			}
		}
	}

	// 文件拖入：落点命中的节点优先（OnDrop），否则给应用级回调
	if a.dropPending {
		a.dropPending = false
		paths := a.dropPaths
		a.dropPaths = nil
		if n := hitDropTarget(root, a.dropX, a.dropY, 0, a); n != nil && n.OnDrop != nil {
			n.OnDrop(paths)
		} else if a.onDrop != nil {
			a.onDrop(paths, a.dropX, a.dropY)
		}
		a.dirty = true
	}

	// 纯指针移动且悬停目标/滚动条状态未变：跳过重绘。
	// 鼠标在空白区域移动、或悬停在同一控件上时无需重画整帧，
	// 毛玻璃/大列表页面的流畅度提升很明显。
	if !forced && probe && !hoverChanged && !scrollHoverChanged && !a.pressed && a.dragKey == "" && !a.clickPending && !a.animating {
		a.updateImePosition()
		return
	}

	// B 阶段：先拿离屏目标再让渲染器画进去；A 阶段用窗口默认帧缓冲。
	// 合成表面尺寸固定为物理像素，超采样只作用于中间渲染目标。
	//
	// 注意：BeginFrame 必须在确认「本帧确实要绘制」之后调用。
	// 旧实现放在函数开头，遇到上面的悬停探测提前返回时不会执行 EndFrame，
	// DComp 会停在一个未提交的帧上（表现为偶发卡顿 / 画面不刷新）。
	if f := a.win.ComposeBegin(); f != 0 {
		composeFBO = f
		a.r.SetTarget(f, pw, ph)
	} else {
		a.r.SetTarget(0, pw, ph)
	}

	if fpsDebug {
		a.dbgLayMs = msSince(frameT0)
		frameT0 = time.Now()
	}
	a.r.Begin(a.bg)
	a.clipN, a.clipSkip = 0, 0
	a.animating = false
	a.animFast = false
	if fpsDebug {
		a.dbgAnim = a.dbgAnim[:0]
	}
	// 收集 WebView 矩形前先重置图片限流标记（否则滚出视口的图片会让 waiting 永久为 1）
	if a.textures != nil {
		a.textures.beginFrame()
	}
	// 每帧重新收集 WebView 矩形：没画到的（滚出视口 / 切页）会在同步时被隐藏
	clear(a.wvRects)
	a.paint(root, 0)
	a.r.End()
	// 超高清：把超采样离屏结果降采样写回呈现目标（未开启时为空操作）
	a.r.Resolve()
	// 网页是独立的子窗口：布局确定后立即同步位置/显隐（紧贴 Swap，避免拖影）
	a.syncWebViews()
	if composeFBO != 0 {
		a.win.ComposeEnd()
	} else {
		a.win.Swap()
	}
	if fpsDebug {
		a.dbgPaintMs = msSince(frameT0)
	}

	// 含动画节点时持续重绘（骨架屏 / 加载动画 / Tween）。
	// 注意用 animDirty 而不是 dirty：dirty 代表“有真实事件待处理”（输入/点击/悬停），
	// 必须立即出帧；animDirty 只是“动画还需要下一帧”，允许 runLoop 按 8–12Hz 节流。
	if a.animating || animBusy.Load() {
		a.animDirty = true
	}
	// 记录：本帧是否只由装饰动画（光标闪烁 / shimmer）驱动。
	// 这种情况下一帧不必整树重建，交给 runLoop 按 8–12Hz 节流，
	// 否则常驻骨架屏会让整个套件页一直满帧重排（CPU 吃满一个核）。
	busy := animBusy.Load()
	a.dbgBusy = busy
	a.slowOnlyFrame = a.animating && !a.animFast && !busy &&
		!a.pressed && a.dragKey == "" && a.scrollDragKey == "" &&
		!a.clickPending && !a.isScrollAnimating()
	// 平滑滚动未到位：继续请求帧（不依赖 dirty，避免鼠标不动时停在半路）
	if a.isScrollAnimating() {
		a.animDirty = true
	}
	// 滚动条淡出：隐停后还有一段淡出过渡，需继续重绘
	if a.scrollbarFading() {
		a.animDirty = true
	}
	// 图片仍在解码/被限流：继续刷帧推进加载
	if a.textures != nil && (imagePending() > 0 || a.textures.waiting.Load() > 0) {
		a.animDirty = true
	}

	a.updateImePosition()

	a.frameCount++
	a.lastPaint = time.Now()
	a.trimmed, a.trimmedDeep = false, false
	a.logMemStats()
	if a.opts.DumpLayout != "" && a.frameCount == 1 {
		var sb strings.Builder
		dumpNode(&sb, root, 0)
		_ = os.WriteFile(a.opts.DumpLayout, []byte(sb.String()), 0644)
		a.running = false
	}
}

// paint 递归绘制。sy 为祖先滚动容器累积的垂直偏移（只影响内容，不影响自身框）。
func (a *App) paint(n *Node, sy float32) {
	if n == nil || n.Style.Opacity <= 0 {
		return
	}
	st := &n.Style
	if st.Animated {
		a.animating = true
		if !st.AnimSlow {
			a.animFast = true
		}
		if fpsDebug && len(a.dbgAnim) < 8 {
			tag := "fast"
			if st.AnimSlow {
				tag = "slow"
			}
			a.dbgAnim = append(a.dbgAnim, fmt.Sprintf("%s(%s)", n.Key, tag))
		}
	}

	r := n.Frame.Offset(st.DrawX, sy+st.DrawY)
	if r.W <= 0 || r.H <= 0 {
		return
	}

	// 滚动容器：内容变短时把偏移钳制回有效范围（否则会停在空白处）
	if n.Kind == KScroll && n.Key != "" {
		maxScroll := n.ContentH - n.Frame.H
		if maxScroll < 0 {
			maxScroll = 0
		}
		a.clampScroll(n.Key, maxScroll)
	}

	clipped := st.Clip || n.Kind == KScroll
	if clipped {
		a.pushClip(r)
	}

	// 毛玻璃：捕获已绘制背景并模糊（必须在绘制自身背景之前）
	if st.BackdropBlur > 0 {
		a.r.Backdrop(r, st.Radius, st.BackdropBlur, st.BackdropTint)
	}

	// 不在绘制层自动叠色：悬停/按下外观完全由业务样式决定，保持背景透明无色
	bg := st.Bg
	bgTo := st.BgTo
	shadow := st.ShadowColor
	// 渐变端色常为全透明（rgba(...,0)），不能只看 alpha 判断是否画渐变——
	// 否则白色→透明的遮罩会退化成实心白，把底下的色相层整个盖住。
	gradOn := st.BgToSet || bgTo.A > 0

	if bg.A > 0 || gradOn {
		sh := gpu.Shape{Rect: r, Radius: st.Radius, ShadowBlur: st.ShadowBlur, ShadowOffsetY: st.ShadowY, ShadowSpread: st.ShadowSpread, ShadowColor: shadow}
		if gradOn {
			sh.Grad = gpu.Gradient{On: true, Angle: st.BgAngle, From: bg, To: bgTo}
		} else {
			sh.Fill = bg
		}
		if st.Border > 0 {
			sh.Stroke = st.Border
			sh.StrokeColor = st.BorderColor
		}
		a.r.Rect(sh)
	} else if st.Border > 0 && st.BorderColor.A > 0 {
		a.r.Rect(gpu.Shape{Rect: r, Radius: st.Radius, Stroke: st.Border, StrokeColor: st.BorderColor})
	}

	switch n.Kind {
	case KText, KButton:
		a.paintText(n, r, st)
	case KImage:
		if tex := a.textures.get(a.r, a, n.Src); tex != nil {
			a.r.DrawImage(gpu.Image{Rect: r, Tex: tex, Tint: gpu.Paint{R: 1, G: 1, B: 1, A: 1}, Radius: st.Radius})
		}
	case KVideo:
		a.paintVideo(n, r, st)
	case KWebView:
		a.paintWebView(n, r, st)
	case KArrow:
		a.paintArrow(r, st.Color)
	case KChart:
		a.paintChart(n, r)
	case KRichText:
		a.paintRichText(n, r, st)
	}

	childSy := sy
	contentClip := r
	if n.Kind == KScroll {
		childSy = sy - a.scrolls[n.Key]
		// 视口区间必须用「未减偏移的 sy」计算（scrollViewport 内部自己减 offset），
		// 在这里换算成窗口坐标后随裁剪区传下去。paintChildren 收到的 sy 已减过
		// offset，若在那里再算一次会把可见区间整体平移一个 offset——offset=0 时
		// 恰好相等所以第一页正常，滚动后所有行都被剔除 → 只见第一页、后续全空白。
		if vy0, vy1, ok := scrollViewport(n, sy, a); ok {
			contentClip = geom.Rect{X: r.X, Y: vy0, W: r.W, H: vy1 - vy0}
		} else {
			contentClip = geom.Rect{}
		}
	}
	a.paintChildren(n, n.Children, childSy, clipped, contentClip)

	// 滚动条（gio 风格：内容溢出时在右侧绘制细条，悬停/拖动加宽）
	if n.Kind == KScroll {
		a.paintScrollbar(n, r)
	}

	// 统一焦点环（键盘导航期间绘制在内容之上）
	a.paintFocusRing(n, r, st)

	if clipped {
		a.popClip()
	}
}

// paintChildren 绘制子节点。
// 裁剪区内的子项按可见区间先做一次粗剔除：滚动长列表时不再为屏幕外的节点
// 生成顶点（以前每个子节点无论可见与否都要走一遍绘制与顶点拼装）。
// cr 由调用方准备好：滚动容器是「窗口坐标的可见区间」（paint 里已换算），
// 普通裁剪容器是自身矩形；sy 已含祖先滚动偏移。
func (a *App) paintChildren(n *Node, children []*Node, sy float32, clipped bool, cr geom.Rect) {
	if !clipped {
		for _, c := range children {
			a.paint(c, sy)
		}
		return
	}
	for _, c := range children {
		if c == nil {
			continue
		}
		// PosAbs 子项可能被摆放到底部等位置，不做提前剔除
		if !c.Style.PosAbs {
			fr := c.Frame.Offset(0, sy)
			if fr.Bottom()+c.Style.Margin.Bottom < cr.Y || fr.Y-c.Style.Margin.Top > cr.Bottom() {
				continue
			}
		}
		a.paint(c, sy)
	}
}

// pushClip 开启一层裁剪；与当前裁剪区完全相同时跳过（Scroll 内再套 Clip 很常见），
// 避免重复入栈并在 Pop 时引发多余的批次切分。
func (a *App) pushClip(r geom.Rect) {
	if a.clipN < len(a.clipStack) {
		a.clipStack[a.clipN] = r
	} else {
		a.clipStack = append(a.clipStack, r)
	}
	if a.clipN > 0 && a.clipStack[a.clipN-1] == r {
		a.clipSkip++
	} else {
		a.r.PushClip(r)
	}
	a.clipN++
}

// popClip 关闭一层裁剪。
func (a *App) popClip() {
	if a.clipN <= 0 {
		return
	}
	a.clipN--
	if a.clipSkip > 0 {
		a.clipSkip--
		return
	}
	a.r.PopClip()
}

// scrollbarFading 滚动条淡出过渡是否仍在进行（需继续重绘推进）。
func (a *App) scrollbarFading() bool {
	return a.scrollIdle < 1.5
}

// scrollbarMetrics 计算滑块高度与可滚动距离（内容不溢出时 ok=false）。
func scrollbarMetrics(n *Node, r geom.Rect) (thumbH, maxScroll float32, ok bool) {
	if n == nil || r.H <= 0 || n.Frame.H <= 0 || n.ContentH <= n.Frame.H || n.ContentH <= 0 {
		return 0, 0, false
	}
	maxScroll = n.ContentH - n.Frame.H
	if maxScroll <= 0 {
		return 0, 0, false
	}
	thumbH = r.H * (n.Frame.H / n.ContentH)
	if thumbH < 24 {
		thumbH = 24
	}
	if thumbH > r.H {
		thumbH = r.H
	}
	if thumbH >= r.H {
		return 0, 0, false
	}
	return thumbH, maxScroll, true
}

// scrollbarZone 滚动条热区宽度（比视觉宽度大，方便鼠标抓取）。
const scrollbarZone = 12

// updateScrollbar 处理滚动条悬停、拖动与轨道点击。
func (a *App) updateScrollbar(root *Node) {
	a.scrollHoverKey = ""
	// 拖动/缩放窗口期间：鼠标消息已进不了窗口，滚动条悬停与拖动状态一律冻结
	if a.sizeMoving {
		return
	}
	if !a.mouseIn {
		if !a.pressed {
			a.scrollDragKey = ""
		}
		return
	}

	// 拖动中：指针即使离开热区也继续跟手（以前一离开 12px 区就“粘死”）
	if a.pressed && a.scrollDragKey != "" {
		a.applyScrollbarDrag(a.scrollDragKey)
		a.scrollIdle = 0
		return
	}

	// 命中滚动条区域（后遍历的容器在上层，取最后一个）
	var hit *Node
	var hitRect geom.Rect
	walkScrollbars(root, 0, a, func(n *Node, r geom.Rect) {
		zone := geom.Rect{X: r.Right() - scrollbarZone, Y: r.Y, W: scrollbarZone, H: r.H}
		if zone.Contains(geom.Point{X: a.mouseX, Y: a.mouseY}) {
			hit, hitRect = n, r
		}
	})
	if hit == nil {
		if !a.pressed {
			a.scrollDragKey = ""
		}
		return
	}
	a.scrollHoverKey = hit.Key
	a.scrollIdle = 0
	thumbH, maxScroll, ok := scrollbarMetrics(hit, hitRect)
	if !ok || !a.pressed {
		return
	}

	thumbY := hitRect.Y + (hitRect.H-thumbH)*a.scrolls[hit.Key]/maxScroll
	inThumb := a.mouseY >= thumbY-4 && a.mouseY <= thumbY+thumbH+4
	a.scrollDragKey = hit.Key
	if !inThumb {
		// 轨道点击：滑块中心对齐指针（与浏览器/gio 一致的手感），再接着拖动
		travel := hitRect.H - thumbH
		want := float32(0)
		if travel > 0 {
			want = (a.mouseY - hitRect.Y - thumbH/2) / travel * maxScroll
		}
		a.setScrollTarget(hit.Key, want, maxScroll)
		a.scrolls[hit.Key] = a.scrollTarget(hit.Key)
		delete(a.scrollTargets, hit.Key)
	}
	a.scrollDragTrack = hitRect
	a.scrollDragThumb = thumbH
	a.scrollDragMax = maxScroll
	a.scrollDragY = a.mouseY
	a.scrollDragStart = a.scrolls[hit.Key]
}

// applyScrollbarDrag 按鼠标位移换算滚动偏移（拖出热区依然生效）。
func (a *App) applyScrollbarDrag(key string) {
	if a.scrollDragMax <= 0 {
		return
	}
	travel := a.scrollDragTrack.H - a.scrollDragThumb
	if travel <= 0 {
		return
	}
	v := a.scrollDragStart + (a.mouseY-a.scrollDragY)/travel*a.scrollDragMax
	if v < 0 {
		v = 0
	}
	if v > a.scrollDragMax {
		v = a.scrollDragMax
	}
	delete(a.scrollTargets, key)
	if a.scrolls[key] != v {
		a.scrolls[key] = v
		a.dirty = true
	}
}

// walkScrollbars 遍历可见滚动容器的滚动条几何（视口外子树直接剪枝）。
func walkScrollbars(n *Node, sy float32, a *App, fn func(*Node, geom.Rect)) {
	if n == nil || a == nil {
		return
	}
	walkScrollbarsIn(n, sy, a, n.Frame, fn)
}

// walkScrollbarsIn 带可见矩形的滚动条遍历。
// 每帧调用一次，长列表/多页文档里屏幕外的整棵子树不必再走一遍
// （旧实现无条件递归全部节点，命中测试之外的额外 O(节点数) 开销）。
func walkScrollbarsIn(n *Node, sy float32, a *App, vis geom.Rect, fn func(*Node, geom.Rect)) {
	if n == nil || n.Style.Opacity <= 0 || n.Disabled {
		return
	}
	r := n.Frame.Offset(0, sy)
	if !n.Style.PosAbs && vis.W > 0 && vis.H > 0 &&
		(r.Bottom() < vis.Y || r.Y > vis.Bottom() || r.Right() < vis.X || r.X > vis.Right()) {
		return
	}
	if n.Kind == KScroll {
		if _, _, ok := scrollbarMetrics(n, r); ok {
			fn(n, r)
		}
	}
	for _, c := range n.Children {
		childSy := sy
		childVis := vis
		if n.Kind == KScroll {
			childSy = sy - a.scrolls[n.Key]
			// 滚动内容只在自身视口内可见
			inner := r.Intersect(vis)
			if inner.W <= 0 || inner.H <= 0 {
				inner = r
			}
			childVis = inner
		}
		walkScrollbarsIn(c, childSy, a, childVis, fn)
	}
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// scrollViewport 返回滚动容器内容在视口中的可见纵向区间（ok=false 表示完全不可见）。
func scrollViewport(n *Node, sy float32, a *App) (y0, y1 float32, ok bool) {
	if n == nil || n.Frame.H <= 0 || n.ContentH <= 0 {
		return 0, 0, false
	}
	top := n.Frame.Y + sy - a.scrolls[n.Key]
	bot := top + n.ContentH
	vy0 := n.Frame.Y + sy
	vy1 := vy0 + n.Frame.H
	if top < vy0 {
		top = vy0
	}
	if bot > vy1 {
		bot = vy1
	}
	return top, bot, bot > top
}

// paintScrollbar 绘制纵向滚动条（内容溢出时显示；悬停/拖动加宽，闲置淡出）。
func (a *App) paintScrollbar(n *Node, r geom.Rect) {
	// 拖动/缩放窗口期间不画滚动条（否则窗边会闪出/残留一条静止的条）
	if a.sizeMoving {
		return
	}
	if n.Key == "" {
		return
	}
	thumbH, maxScroll, ok := scrollbarMetrics(n, r)
	if !ok {
		return
	}
	v := a.scrolls[n.Key]
	if v < 0 {
		v = 0
	}
	if v > maxScroll {
		v = maxScroll
	}
	y := r.Y + (r.H-thumbH)*v/maxScroll

	// 悬停/拖动时加宽并加深；闲置后淡出（以前 scrollLastUsed 从未递增，永远不淡出）
	w := float32(4)
	alpha := float32(0.18)
	active := a.scrollHoverKey == n.Key || a.scrollDragKey == n.Key
	if active {
		w = 8
		alpha = 0.42
	} else if a.scrollIdle > 0.7 {
		f := (a.scrollIdle - 0.7) / 0.7
		if f > 1 {
			f = 1
		}
		alpha *= 1 - f
		if alpha <= 0.01 {
			return
		}
	}
	x := r.Right() - w - 2
	a.r.Rect(gpu.Shape{
		Rect:   geom.Rect{X: x, Y: y, W: w, H: thumbH},
		Radius: [4]float32{w / 2, w / 2, w / 2, w / 2},
		Fill:   scrollbarPaint(alpha),
	})
}

// scrollbarPaint 滚动条颜色：跟随主题取文字色的半透明版（深色主题=白，浅色主题=深灰），
// 旧实现写死白色，浅色背景上永远看不见。空闲态再提亮一点保证可辨。
func scrollbarPaint(alpha float32) gpu.Paint {
	col := C(ColText)
	if !IsDark() {
		// 浅色主题下同等 alpha 对比度偏低，适当加深
		if alpha < 0.5 {
			alpha *= 1.6
		}
	}
	col.A = alpha
	return col
}

func (a *App) paintText(n *Node, r geom.Rect, st *Style) {
	if n.Text == "" {
		return
	}
	f := a.fonts.getFamily(st.FontSize, st.FontWeight, st.FontFamily)
	if f == nil {
		return
	}
	lh := st.LineHeight
	if lh <= 0 {
		lh = f.LineHeight
	}
	flh := f.LineHeight

	tx := r.X + st.Pad.Left
	// 与布局测量共用同一折行算法与缓存，行数严格一致
	wrap := st.Wrap && n.wrapW > 0
	var wlines []string
	if wrap {
		wlines = wrapLinesCached(f, n.Text, n.wrapW)
	}

	// 垂直定位：按钮等在垂直居中类节点上按整体高度居中，避开 padding 挤压；
	// 普通文本仍从顶部开始排版（此时无需先数行数，省一次遍历）。
	y := r.Y + st.Pad.Top
	if st.CenterY || n.Kind == KButton {
		lines := lineCount(n.Text)
		if wrap {
			lines = len(wlines)
		}
		totalH := float32(lines) * lh
		y = r.Y + (r.H-totalH)/2
		if y < r.Y {
			y = r.Y
		}
	}

	draw := func(ln string) {
		x := tx
		switch {
		case n.Kind == KButton:
			// 按钮内文字水平居中
			x = r.X + (r.W-f.Measure(ln))/2
		case st.TextAlign == AlignCenter:
			x = r.X + (r.W-f.Measure(ln))/2
		case st.TextAlign == AlignEnd:
			x = r.Right() - f.Measure(ln) - st.Pad.Right
		}

		// 行内微调：行高大于字体行高时居中，小于时不能为负（否则顶部被裁）
		vy := y
		if lh > flh {
			vy = y + (lh-flh)/2
		}
		a.r.TextRun(gpu.TextRun{X: x, Y: vy, Text: ln, Font: f, Color: st.Color})
		y += lh
	}
	// 逐行直接绘制：不再为每个文本节点每帧分配行切片
	if wrap {
		for _, ln := range wlines {
			draw(ln)
		}
	} else {
		forEachLine(n.Text, draw)
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

// ---- 命中测试 ----

// hitInteractive 返回命中的最近可交互节点（自身或祖先）。
// 同时把 OnPress（按下响应，例如自绘标题栏拖动）视为可交互。
// OnClickAt / OnDrag 也算：滑块这类只注册「带坐标点击 + 拖动」的控件，
// 若只认 OnClick，按下时命中的是内部 Box，dragKey 永远不会建立，
// 表现为拖动毫无反应、松手才靠点击跳一下（Slider 实时跟手失效的根因）。
func hitInteractive(root *Node, x, y, sy float32, a *App) *Node {
	n := hitTest(root, x, y, sy, a)
	for c := n; c != nil; c = c.Parent {
		if c.OnClick != nil || c.OnPress != nil || c.OnClickAt != nil || c.OnDrag != nil || c.OnSpanClick != nil {
			return c
		}
	}
	return n
}

// hitDropTarget 返回落点下最近的「接收拖入」节点（自身或祖先）。
// 拖入没有悬停态（WM_DROPFILES 只在松手时到达），所以只在落下那一刻命中。
func hitDropTarget(root *Node, x, y, sy float32, a *App) *Node {
	n := hitTest(root, x, y, sy, a)
	for c := n; c != nil; c = c.Parent {
		if c.OnDrop != nil {
			return c
		}
	}
	return n
}

func hitTest(n *Node, x, y, sy float32, a *App) *Node {
	if n == nil || n.Style.Opacity <= 0 || n.Disabled {
		return nil
	}
	r := n.Frame.Offset(0, sy)
	if !r.Contains(geom.Point{X: x, Y: y}) {
		return nil
	}
	childSy := sy
	if n.Kind == KScroll {
		childSy = sy - a.scrolls[n.Key]
	}
	for i := len(n.Children) - 1; i >= 0; i-- {
		if c := hitTest(n.Children[i], x, y, childSy, a); c != nil {
			return c
		}
	}
	// 纯装饰节点（无交互、无背景、无可见子节点）不参与命中：
	// 否则一个全空的 AbsFill 叠层会把整窗点击全吞掉（连标题栏关闭按钮都点不动），
	// 而滚轮没受影响，现象就是「只能滚、点什么都不反应」。
	if inputPassthrough(n) {
		return nil
	}
	return n
}

// inputPassthrough 判断节点是否对鼠标输入“透明”（可穿透）。
// 条件：没有事件回调、不可聚焦、没有可见背板、没有任何非空子节点，
// 且不是自带内容的节点（滚动容器 / 图片 / 视频 / 网页 / 图表）。
func inputPassthrough(n *Node) bool {
	if n.OnClick != nil || n.OnPress != nil || n.OnClickAt != nil || n.OnDrag != nil ||
		n.OnSpanClick != nil || n.OnKey != nil || n.OnChar != nil || n.OnDrop != nil ||
		n.OnHover != nil || n.Style.Focusable {
		return false
	}
	switch n.Kind {
	case KScroll, KImage, KVideo, KWebView, KChart, KRichText, KArrow:
		return false
	}
	if n.Style.Bg.A > 0 || n.Style.BgTo.A > 0 || n.Style.Border > 0 || n.Style.BackdropBlur > 0 {
		return false
	}
	for _, c := range n.Children {
		if c != nil {
			return false
		}
	}
	return true
}

// wheelStepSize 鼠标滚轮一格（WHEEL_DELTA）的滚动距离（逻辑像素）。
const wheelStepSize = 96

// handleWheel 滚轮滚动：从指针命中的最内层滚动容器开始，
// 内层到底/到顶后把剩余滚动量冒泡给外层容器（以前内层到顶就 return，
// 外层页面完全无法继续滚动，手感像卡住）。
// 滚动目标写入 scrollTargets，由 stepScrollAnim 缓动到目标位，避免一格一跳的生硬感。
func (a *App) handleWheel(x, y, dy float32) {
	if a.root == nil || dy == 0 {
		return
	}
	// dy 为轮转向（上正下负）：向下滚时偏移量增加
	rem := -dy * wheelStepSize
	if rem == 0 {
		return
	}
	n := hitTest(a.root, x, y, 0, a)
	changed := false
	for c := n; c != nil && rem != 0; c = c.Parent {
		if c.Kind != KScroll || c.Key == "" {
			continue
		}
		max := c.ContentH - c.Frame.H
		if max <= 0 {
			continue
		}
		cur := a.scrollTarget(c.Key)
		if cur > max {
			cur = max
		}
		nv := cur + rem
		if nv < 0 {
			nv = 0
		}
		if nv > max {
			nv = max
		}
		if used := nv - cur; used != 0 {
			a.setScrollTarget(c.Key, nv, max)
			rem -= used
			changed = true
		}
	}
	if changed {
		a.scrollIdle = 0
		a.dirty = true
	}
}

// scrollTarget 返回 key 的平滑滚动目标位（未设定时返回当前位）。
func (a *App) scrollTarget(key string) float32 {
	if t, ok := a.scrollTargets[key]; ok {
		return t
	}
	return a.scrolls[key]
}

// setScrollTarget 设定滚动目标位（自动夹取）。
func (a *App) setScrollTarget(key string, v, max float32) {
	if v < 0 {
		v = 0
	}
	if v > max {
		v = max
	}
	if a.scrollTargets == nil {
		a.scrollTargets = map[string]float32{}
	}
	// 偏移表是懒建的：滚轮首次滚动时 scrolls 里还没有这个 key，
	// 必须先建出 0 起点，否则 stepScrollAnim 会把目标当成“无效节点”丢掉，
	// 表现为“有滚动条但滚轮不滚，拖一下滚动条后才正常”。
	if a.scrolls == nil {
		a.scrolls = map[string]float32{}
	}
	if _, ok := a.scrolls[key]; !ok {
		a.scrolls[key] = 0
	}
	if a.scrolls[key] == v {
		delete(a.scrollTargets, key)
		return
	}
	a.scrollTargets[key] = v
}

// clampScroll 把当前偏移夹到合法范围（内容变短时避免停在空白处）。
func (a *App) clampScroll(key string, max float32) {
	if v, ok := a.scrolls[key]; ok {
		if v > max {
			a.scrolls[key] = max
		} else if v < 0 {
			a.scrolls[key] = 0
		}
	}
	if v, ok := a.scrollTargets[key]; ok {
		if v > max {
			a.scrollTargets[key] = max
		} else if v < 0 {
			a.scrollTargets[key] = 0
		}
	}
}

// Invalidate 请求重绘（可跨 goroutine 调用）。
func (a *App) Invalidate() { a.redrawRequest.Store(true) }

// requestFrame 请求一次带滚动动画的尾帧（平滑滚动进行中时保证下一帧仍会绘制）。
func (a *App) requestFrame() {
	a.redrawRequest.Store(true)
}

// isScrollAnimating 是否仍有滚动动画未到位。
func (a *App) isScrollAnimating() bool {
	for k, cur := range a.scrolls {
		if tgt, ok := a.scrollTargets[k]; ok && math.Abs(float64(tgt-cur)) > 0.05 {
			return true
		}
	}
	return false
}

// stepScrollAnim 推进平滑滚动：关键盘时间步长向目标位置缓动。
func (a *App) stepScrollAnim() {
	dt := a.animDt
	if dt <= 0 {
		dt = 1.0 / 120
	}
	if dt > 0.05 {
		dt = 0.05
	}
	k := 1 - float32(math.Exp(float64(-dt*22)))
	dirty := false
	for key, tgt := range a.scrollTargets {
		cur, ok := a.scrolls[key]
		if !ok {
			// 已建条目由 setScrollTarget 保证；这里兼容历史目标：
			// 从 0 开始缓动，而不是直接丢掉滚动目标。
			a.scrolls[key] = 0
			cur = 0
		}
		d := tgt - cur
		if d > -0.35 && d < 0.35 {
			cur = tgt
		} else {
			cur += d * k
		}
		if cur == a.scrolls[key] {
			continue
		}
		a.scrolls[key] = cur
		dirty = true
		if math.Abs(float64(tgt-cur)) < 0.2 {
			delete(a.scrollTargets, key)
		}
	}
	if dirty {
		a.dirty = true
	}
}

// focusedNode 返回当前获得键盘焦点的节点（上一帧树）。
func (a *App) focusedNode() *Node {
	if a.focusKey == "" || a.root == nil {
		return nil
	}
	return findKey(a.root, a.focusKey)
}

func findKey(n *Node, key string) *Node {
	if n == nil {
		return nil
	}
	if n.Key == key {
		return n
	}
	for _, c := range n.Children {
		if r := findKey(c, key); r != nil {
			return r
		}
	}
	return nil
}

// visualOrigin 返回节点在窗口坐标系中的左上角（已计入祖先滚动偏移）。
func (a *App) visualOrigin(n *Node, _, _ float32) (float32, float32) {
	x, y := n.Frame.X, n.Frame.Y
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Kind == KScroll {
			y -= a.scrolls[p.Key]
		}
	}
	return x, y
}

// updateImePosition 把输入法候选窗口移到焦点输入框下方。
func (a *App) updateImePosition() {
	if a.imeText == "" || a.focusKey == "" || a.root == nil {
		return
	}
	n := findKey(a.root, a.focusKey)
	if n == nil {
		return
	}
	x, y := a.visualOrigin(n, 0, 0)
	s := a.effScale()
	nx := int32((x + n.Frame.W - 8) * s)
	ny := int32((y + n.Frame.H) * s)
	if nx == a.imeX && ny == a.imeY {
		return
	}
	a.imeX, a.imeY = nx, ny
	a.win.SetImePosition(nx, ny)
}

// ---- 包级 API（供组件与业务调用）----

// SetFocus 设置键盘焦点到指定 Key（传空串取消焦点）。
func SetFocus(key string) {
	if a := currentApp.Load(); a != nil {
		a.focusKey = key
		a.redrawRequest.Store(true)
	}
}

// FocusedKey 返回当前焦点 Key。
func FocusedKey() string {
	if a := currentApp.Load(); a != nil {
		return a.focusKey
	}
	return ""
}

// IsFocused 判断节点是否持有焦点。
func IsFocused(key string) bool { return key != "" && FocusedKey() == key }

// IsHovered 判断鼠标当前是否悬停在 Key 对应的节点上（悬停外观用）。
func IsHovered(key string) bool {
	a := currentApp.Load()
	return a != nil && key != "" && a.hoverKey == key
}

// IsPressed 判断鼠标是否正按在 Key 对应的节点上（按下外观用）。
func IsPressed(key string) bool {
	a := currentApp.Load()
	return a != nil && key != "" && a.pressed && a.pressNode != nil && a.pressNode.Key == key
}

// LastFrameOf 返回上一帧中 Key 对应节点在窗口坐标系里的矩形（浮层按锚点定位用）。
// 一帧延迟对浮层不可见；首帧或节点不存在时返回 false。
func LastFrameOf(key string) (geom.Rect, bool) {
	a := currentApp.Load()
	if a == nil || a.root == nil || key == "" {
		return geom.Rect{}, false
	}
	n := findKey(a.root, key)
	if n == nil || n.Frame.W <= 0 || n.Frame.H <= 0 {
		return geom.Rect{}, false
	}
	x, y := a.visualOrigin(n, 0, 0)
	r := n.Frame
	r.X, r.Y = x, y
	return r, true
}

// ComposingText 返回当前窗口的 IME 组合串（未组合时为空）。
func ComposingText() string {
	if a := currentApp.Load(); a != nil {
		return a.imeText
	}
	return ""
}

// ComposingCaret 返回组合串内的光标位置（rune 数）。
func ComposingCaret() int {
	if a := currentApp.Load(); a != nil {
		return a.imeCaret
	}
	return 0
}

// AnimTime 返回启动至今的秒数（含 Animated 节点时每帧递增重绘）。
func AnimTime() float32 {
	a := currentApp.Load()
	if a == nil || a.startTime.IsZero() {
		return 0
	}
	return float32(time.Since(a.startTime).Seconds())
}

// MousePos 返回鼠标当前位置（逻辑像素）。
func MousePos() (float32, float32) {
	if a := currentApp.Load(); a != nil {
		return a.mouseX, a.mouseY
	}
	return 0, 0
}

// ScrollOffset 返回指定滚动容器当前的纵向偏移（虚拟列表据此计算可见区间）。
func ScrollOffset(key string) float32 {
	if a := currentApp.Load(); a != nil {
		return a.scrolls[key]
	}
	return 0
}

// ScrollTo 立即跳转到指定偏移（自动夹取到 0..max）。
// 虚拟列表 / 目录跳转 / 「回到顶部」用。
func ScrollTo(key string, y float32) {
	a := currentApp.Load()
	if a == nil || key == "" {
		return
	}
	if y < 0 {
		y = 0
	}
	if a.root != nil {
		if n := findKey(a.root, key); n != nil {
			if max := n.ContentH - n.Frame.H; max > 0 && y > max {
				y = max
			}
		}
	}
	a.scrolls[key] = y
	delete(a.scrollTargets, key)
	a.dirty = true
}

// ScrollBy 相对滚动（正数向下）。
func ScrollBy(key string, dy float32) { ScrollTo(key, ScrollOffset(key)+dy) }

// MouseDown 报告鼠标左键当前是否仍按下（拖动残留清理用）。
func MouseDown() bool {
	if a := currentApp.Load(); a != nil {
		return a.pressed
	}
	return false
}

// 悬停提示（TooltipLayer 读取）
var (
	tooltipText        string
	tooltipX, tooltipY float32
)

// SetTooltip 设置悬停提示文本与位置（空串清除），一般传 MousePos()。
func SetTooltip(text string, x, y float32) {
	if text == "" && tooltipText == "" {
		return
	}
	tooltipText, tooltipX, tooltipY = text, x, y
	RequestRedraw()
}

// TextWidth 测量单行文本宽度（逻辑像素，UI 线程调用）。
func TextWidth(s string, size float32, weight int32) float32 {
	return TextWidthOf(s, size, weight, "")
}

// TextWidthOf 按字体家族测量单行文本宽度。
func TextWidthOf(s string, size float32, weight int32, family string) float32 {
	a := currentApp.Load()
	if a == nil || a.fonts == nil {
		return 0
	}
	f := a.fonts.getFamily(size, weight, family)
	if f == nil {
		return 0
	}
	return f.Measure(s)
}

// TextGlyphBounds 返回单行文本相对「行顶」的视觉包围盒：top 为字形顶部偏移，
// h 为字形视觉高度（均为逻辑像素）。字形按实际位图包围盒计算，
// 因此可用于图标在固定高度容器里的光学垂直居中（CenterY 按行高居中会因降部留白而偏上）。
func TextGlyphBounds(s string, size float32, weight int32) (top, h float32) {
	a := currentApp.Load()
	if a == nil || a.fonts == nil {
		return 0, 0
	}
	return textGlyphBoundsOf(a.fonts.getFamily(size, weight, ""), s)
}

// textGlyphBoundsOf 按字体计算单行文本相对行顶的视觉包围盒（逻辑像素）。
func textGlyphBoundsOf(f *text.Font, s string) (top, h float32) {
	if f == nil {
		return 0, 0
	}
	sc := f.Scale()
	if sc <= 0 {
		sc = 1
	}
	var y0, y1 float32
	first := true
	for _, r := range s {
		g := f.GlyphFor(r)
		if !g.Valid || g.W <= 0 || g.H <= 0 {
			continue
		}
		// 基线 = 行顶 + Ascent；Glyph.Top 为相对基线的位图顶部（向下为正，通常为负）
		t := f.Ascent + g.Top/sc
		b := t + g.H/sc
		if first {
			y0, y1, first = t, b, false
			continue
		}
		if t < y0 {
			y0 = t
		}
		if b > y1 {
			y1 = b
		}
	}
	if first {
		return 0, 0
	}
	return y0, y1 - y0
}

// currentApp 保存当前 UI 线程的应用实例，供包级 API 使用。
var currentApp atomic.Pointer[App]

var (
	appsMu sync.Mutex
	apps   []*App
)

func registerApp(a *App) {
	appsMu.Lock()
	apps = append(apps, a)
	appsMu.Unlock()
	currentApp.Store(a)
}

func unregisterApp(a *App) {
	appsMu.Lock()
	for i, x := range apps {
		if x == a {
			apps = append(apps[:i], apps[i+1:]...)
			break
		}
	}
	appsMu.Unlock()
}

func snapshotApps() []*App {
	appsMu.Lock()
	out := make([]*App, len(apps))
	copy(out, apps)
	appsMu.Unlock()
	return out
}

// appsInto 把当前窗口列表追加到 dst 上（调用方传 dst[:0] 原位复用），
// 供渲染主循环使用：避免每轮循环都分配一次窗口列表切片。
func appsInto(dst []*App) []*App {
	appsMu.Lock()
	dst = append(dst, apps...)
	appsMu.Unlock()
	return dst
}

// WindowCount 返回当前存活窗口数。
func WindowCount() int {
	return len(snapshotApps())
}

// SetOnKey 设置全局按键回调（任意节点聚焦时都会收到，用于 ESC 等全局快捷键）。
func (a *App) SetOnKey(fn func(uint32)) { a.onKey = fn }

// SetOnDropFiles 设置本窗口的文件拖入回调（拖到没有 OnDrop 的区域时触发）。
func (a *App) SetOnDropFiles(fn func(paths []string, x, y float32)) { a.onDrop = fn }

// SetAcceptDrops 开启/关闭窗口接收文件拖入（默认开启）。
func (a *App) SetAcceptDrops(on bool) { a.win.SetAcceptDrops(on) }

// AcceptDrops 当前窗口是否接收文件拖入。
func (a *App) AcceptDrops() bool { return a.win.AcceptDrops() }

// SetAlwaysOnTop 窗口置顶 / 取消置顶。
func (a *App) SetAlwaysOnTop(on bool) { a.win.SetAlwaysOnTop(on); a.dirty = true }

// IsAlwaysOnTop 当前窗口是否置顶。
func (a *App) IsAlwaysOnTop() bool { return a.win.AlwaysOnTop() }

// RegisterHotkey 注册系统级热键（窗口失焦/最小化同样触发），返回 id（0 = 失败）。
func (a *App) RegisterHotkey(mods uint32, vk uint32, fn func()) int32 {
	return a.win.RegisterHotKey(mods, vk, fn)
}

// UnregisterHotkey 注销热键。
func (a *App) UnregisterHotkey(id int32) { a.win.UnregisterHotkey(id) }

// OnKeyGlobal 给当前活跃窗口设置全局按键回调（无窗口时忽略）。
func OnKeyGlobal(fn func(uint32)) {
	if a := currentApp.Load(); a != nil {
		a.onKey = fn
	}
}

// RequestRedraw 从任意 goroutine 请求重绘（业务回调修改状态后调用）。
func RequestRedraw() {
	for _, a := range snapshotApps() {
		a.redrawRequest.Store(true)
	}
	if fpsDebug {
		// 记录调用点（文件名:行号），定位“每帧都在请求重绘”的来源。
		var sb strings.Builder
		for i := 1; i <= 4; i++ {
			if _, f, l, ok := runtime.Caller(i); ok {
				sb.WriteString(fmt.Sprintf("%s:%d < ", f[strings.LastIndexByte(f, '/')+1:], l))
			}
		}
		dbgRedrawFrom.Store(sb.String())
	}
}

// RequestRedrawIn 只重绘指定窗口。
func RequestRedrawIn(a *App) {
	if a != nil {
		a.redrawRequest.Store(true)
	}
}

// SetOnClose 设置窗口关闭拦截：返回 false 阻止关闭（例如最小化到托盘）。
func (a *App) SetOnClose(fn func() bool) { a.onClose = fn }

// HWND 返回窗口句柄（文件对话框、托盘使用）。
func (a *App) HWND() uintptr {
	if a == nil || a.win == nil {
		return 0
	}
	return a.win.NativeHandle()
}

// ShowWindow 显示/隐藏窗口。
func (a *App) ShowWindow(v bool) {
	a.win.SetVisible(v)
	a.dirty = true
}

// Minimize 最小化窗口。
func (a *App) Minimize() { a.win.Minimize() }

// Center 把窗口居中到当前显示器工作区（水平 + 垂直）。
func (a *App) Center() {
	if a == nil {
		return
	}
	a.win.Center()
}

// BeginDragMove 把按下事件交给系统做窗口拖动（自绘标题栏用）。
func (a *App) BeginDragMove() {
	if a == nil {
		return
	}
	a.win.BeginDragMove()
	// 系统拖动期间会吞掉鼠标消息，返回后清掉按下状态，避免 Pressed 残留
	a.pressed = false
	a.pressFired = nil
	a.dragKey = ""
	a.clickPending = false
	a.dirty = true
}

// IsMaximized 窗口是否已最大化。
func (a *App) IsMaximized() bool { return a != nil && a.win.IsMaximized() }

// ToggleMaximize 最大化与还原之间切换。
func (a *App) ToggleMaximize() {
	if a == nil {
		return
	}
	a.win.ToggleMaximize()
	a.dirty = true
}

// SetFullscreen 切换窗口全屏（去掉标题栏并覆盖当前显示器；ESC 可退出）。
func (a *App) SetFullscreen(v bool) {
	a.win.SetFullscreen(v)
	a.win.MakeCurrent()
	a.applyRenderScale()
}

// IsFullscreen 窗口是否处于全屏。
func (a *App) IsFullscreen() bool { return a.win.IsFullscreen() }

// SetCornerRadius 设置窗口圆角半径（逻辑像素；<=0 取消圆角）。
// 全屏状态下不生效，退出全屏后自动恢复。
func (a *App) SetCornerRadius(v float32) {
	if a == nil || a.win == nil {
		return
	}
	a.win.SetCornerRadius(v)
	RequestRedraw()
}

// CornerRadius 返回当前窗口圆角半径（逻辑像素）。
func (a *App) CornerRadius() float32 {
	if a == nil || a.win == nil {
		return 0
	}
	return a.win.CornerRadius()
}

// Restore 还原窗口并置前。
func (a *App) Restore() { a.win.Restore(); a.dirty = true }

// SetTitle 修改窗口标题。
func (a *App) SetTitle(s string) { a.win.SetTitle(s) }

// Close 关闭窗口。
func (a *App) Close() { a.win.Close() }

// Post 在 UI 线程执行回调（可从任意 goroutine 调用）。
func (a *App) Post(fn func()) {
	if fn == nil {
		return
	}
	appsMu.Lock()
	a.pending = append(a.pending, fn)
	appsMu.Unlock()
	a.redrawRequest.Store(true)
}

// idleTrimAfter 空闲多久后释放可重建的 GPU 缓存（第一级，观感无损）。
const idleTrimAfter = 3 * time.Second

// idleTrimDeepAfter 空闲多久后再把物理页交还系统（第二级，重新调页有微小代价）。
const idleTrimDeepAfter = 30 * time.Second

// trimCaches 释放可重建的显存占用（离屏目标 / 图表纹理 / 图片纹理）。
//
// deep=true 时额外把本进程驻留的物理页交还系统（Windows 走 SetProcessWorkingSetSize），
// 只在窗口最小化或长期壑置时调用。
// 只回收「下一帧会自动重建」的资源：字体图集重建代价是整页重新光栅化字形，
// 窗口帧缓冲又属于系统合成链路，都不适合在这里动。
func (a *App) trimCaches(deep bool) {
	if a == nil || a.r == nil || a.win == nil {
		return
	}
	a.win.MakeCurrent()
	before := a.r.OffscreenBytes()
	a.chartReleaseAll()
	if a.textures != nil {
		a.textures.releaseAll(a.r)
	}
	a.r.ReleaseOffscreen()
	// Go 堆：先做一次 GC + 归还未使用的页（Go 的 scavenager 是懒回收，
	// 空闲时主动一次能把 sys 里的空 arena 真正还给系统）。
	debug.FreeOSMemory()
	if deep {
		// 深度空闲（最小化 / 长时间不在前台）：把字体图集也收到最小集。
		// 再次显示时首帧会按需重建，只多花一帧栅格化，换来几十 MB 常驻内存回落。
		if a.fonts != nil {
			a.fonts.trimTo(idleKeepFonts)
		}
		trimProcessMemory()
	}
	a.logTrim(before, a.r.OffscreenBytes(), deep)
}

func (a *App) dispose() {
	if a == nil || a.disposed {
		return
	}
	a.disposed = true
	a.closed = true
	// 先关网页（WebView2 要求在同一条 UI 线程上释放），再拆 GL 与窗口
	a.closeWebViews()
	a.win.MakeCurrent()
	a.chartResetCache()
	// Markdown 渲染树缓存按窗口区分，关窗时一并释放，避免常驻节点树
	mdDropTreeCache(a)
	if a.r != nil {
		a.r.Destroy()
	}
	if a.fonts != nil {
		a.fonts.close()
	}
	a.win.Destroy()
}

// Dispose 主动释放窗口资源。
func (a *App) Dispose() {
	unregisterApp(a)
	a.dispose()
}

var _ = fmt.Sprintf
