# GoGUI 多端渲染

一套 UI 逻辑（`ui` 包 + 业务代码）编译到 **Windows / Linux / macOS / Web**，未来扩展到 Android / iOS。
分层方式与 [gio](https://gioui.org) 同构：**平台差异只存在于「窗口后端 + GL 端口 + 字体来源」三处**，
`ui/app.go` 里 1700 行的布局、绘制、命中测试、滚动、动画、事件分发**没有任何平台条件编译**。

```
┌──────────────────────────────────────────────────────────────┐
│  业务 UI 代码（cmd/gallery, cmd/demo, 你的工程）  ← 一行不改   │
└───────────────────────────┬──────────────────────────────────┘
                            │ ui.Run(Options, view)
┌───────────────────────────▼──────────────────────────────────┐
│ ui/app.go   布局 / 绘制 / 命中 / 滚动 / 动画 / 事件分发        │
│ ui/backend.go   Backend 接口（平台无关契约）                   │
└───────────────────────────┬──────────────────────────────────┘
        ┌──────────────┬────┴─────┬──────────────┬─────────────┐
  backend_windows  backend_x11  backend_wasm  backend_null  backend_android/ios
        │              │            │             │            （待实现）
┌───────▼──────────────▼────────────▼─────────────▼────────────┐
│ internal/display   原生窗口契约（Config / Event / Window）     │
│ internal/x11       Xlib + EGL 绑定（purego，零 cgo）           │
│ internal/win32     Win32 / WGL 绑定（purego）                  │
│ internal/gpu       glport_win32.go / glport_unix.go / _wasm    │
│ internal/text      fontplat_{windows,linux,darwin,android,ios} │
└──────────────────────────────────────────────────────────────┘
```

## 支持矩阵

| 能力 | Windows | Linux (X11) | macOS | Web | Android | iOS |
|---|---|---|---|---|---|---|
| 窗口 / 输入 / 绘制 | ✅ | ✅ | ✅ Cocoa | ✅ | ✅ NativeActivity | ⚠️ 待验证 |
| OpenGL 上下文 | WGL | EGL（回退兼容配置） | ✅ NSOpenGL 3.2 Core | WebGL2 | ✅ EGL/GLES3 | ⚠️ EAGL/GLES3 |
| 中文字体 | 微软雅黑（GDI 枚举） | 扫 `NotoSansCJK` 等目录 | PingFang / Hiragino | 需 `SetWebFont` 注入 | `/system/fonts` | `/System/Library/Fonts` |
| 彩色 Emoji | ✅ GDI 位图 | ⏳ 单色回退 | ⏳ 单色回退 | ✅ canvas | ⏳ | ⏳ |
| 文件对话框 | ✅ IFileDialog | ✅ zenity/kdialog | ✅ osascript | ❌ | ⏳ 需 JNI | ⏳ |
| 剪贴板 | ✅ | ✅ xclip/wl-paste | ✅ pbpaste | ✅（需授权） | ⏳ | ⏳ |
| 系统托盘 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| 全局热键 | ✅ | ❌（需 XGrabKey） | ❌ | ❌ | ❌ | ❌ |
| 文件拖入 | ✅ WM_DROPFILES | ❌（需 XDND） | ❌ | ⚠️ 仅文件名 | ❌ | ❌ |
| 内嵌浏览器 | ✅ WebView2 | ❌ | ❌ | ⚠️ 本身即网页 | ⏳ | ⏳ |
| 视频 / 音频 | ✅ MF + ffmpeg | ✅ ffmpeg + paplay/aplay | ❌ 占位 | ⏳ | ⏳ | ⏳ |
| 深色模式检测 | ✅ 注册表 | ✅ gsettings/defaults | ✅ | ✅ 主题值 | ⏳ | ⏳ |
| 禁止休眠 | ✅ | ✅ systemd-inhibit / caffeinate | ✅ | ❌ | ❌ | ❌ |
| 触摸输入 | ❌ | ❌ | ❌ | ⚠️ DOM 触摸 | ✅ AInputQueue | ⚠️ touchesBegan |
| 零 cgo | ✅ | ✅ | ✅ | ✅ | ⚠️ 需 cgo | ⚠️ 需 cgo |

图例：✅ 已实现 ｜ ⏳ 骨架已就位（能编译、调用安全降级）｜ ❌ 不计划/暂不支持 ｜ ⚠️ 有条件

> 运行时可用 `ui.HasCap(ui.CapVideo)` / `ui.Capabilities()` 查询真实能力，
> 详见下方「能力查询」。

## 构建

```powershell
./build-multi.ps1                 # 全矩阵：windows/linux(amd64,arm64)/darwin(amd64,arm64)/wasm
./build-multi.ps1 -Only linux     # 只构建 Linux
./build-multi.ps1 -Only web       # 只构建 wasm（自动拷贝 wasm_exec.js）
./build-android.ps1               # Android：libgogui.so（arm64-v8a + armeabi-v7a，需 NDK）
./build.ps1                       # Windows 发布构建（图标资源 + NSIS 安装器）
```

手工交叉编译同样可用（桌面端一律 `CGO_ENABLED=0`）：

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o gogui-linux   ./cmd/gallery
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o gogui-macos   ./cmd/gallery
CGO_ENABLED=0 GOOS=js      GOARCH=wasm  go build -o gogui.wasm    ./cmd/gallery
```

### 各端运行依赖

| 平台 | 需要的系统库 / 工具 |
|---|---|
| Windows | 无（只用到系统自带 opengl32 / dwmapi / mfplat 等） |
| Linux | `libX11.so.6`、`libEGL.so.1`、`libGL.so.1`（Debian/Ubuntu：`libx11-6 libegl1 libgl1`）；中文字体 `fonts-noto-cjk`；**媒体必需** `ffmpeg`（提供 ffmpeg/ffprobe）+ 音频后端 `pulseaudio-utils`（paplay）或 `alsa-utils`（aplay）；可选 `zenity`/`kdialog`（文件对话框）、`xclip` 或 `wl-clipboard`（剪贴板） |
| macOS | 系统自带 OpenGL.framework（3.3 Core） |
| Web | 浏览器支持 WebGL2；页面需加载 `wasm_exec.js` |

## 能力查询与优雅降级

符号在所有平台都存在（没有实现就给空实现），能力用 `HasCap` 判断：

```go
ui.Run(ui.Options{Title: "播放器"}, func() *ui.Node {
    if ui.HasCap(ui.CapVideo) {
        p, _ := ui.NewMediaPlayer("movie.mp4")
        return ui.Column(ui.VideoPlayer("v", p))
    }
    return ui.Card(ui.Empty("当前平台不支持视频", "Windows 安装 ffmpeg 或启用 Media Foundation"))
})
```

| 常量 | 含义 |
|---|---|
| `CapVideo` / `CapAudio` | 音视频解码 |
| `CapWebView` | 内嵌浏览器 |
| `CapTray` | 系统托盘 |
| `CapClipboard` | 剪贴板 |
| `CapDialogs` | 文件/文件夹对话框 |
| `CapHotkey` | 全局热键 |
| `CapDragDrop` | 文件拖入 |
| `CapCompose` | 窗口级 GPU 合成（DirectComposition） |
| `CapSystemDark` | 深色模式检测 |
| `CapKeepAwake` | 禁止休眠 |
| `CapRecycleBin` | 回收站 |
| `CapTouch` | 触摸输入 |

## 新增一端要写哪些文件

以「新增 FooOS 后端」为例，**不需要改 `app.go`**：

| 序号 | 文件 | 内容 |
|---|---|---|
| 1 | `ui/backend_fooos.go` | `//go:build fooos`；实现 `platformWin` 的 `Backend` 方法集 + `newBackend(Options)` + `pollMessages()` |
| 2 | `internal/display/display_fooos.go` | 原生窗口适配（`display.Window`，可选：也可以直接在 backend 里调系统 API） |
| 3 | `internal/gpu/glport_fooos.go` | 同一组 `glXxx` 薄包装（参照 `glport_win32.go` / `glport_unix.go`） |
| 4 | `internal/text/fontplat_fooos.go` | `platformFontCandidates` / `platformDefaultFontCandidates` / `platformMapFontFile` / `fontDirs` |
| 5 | `ui/caps_fooos.go` | `platformCaps()` 能力表 |
| 6 | （可选）`ui/platform_fooos.go` | 对话框 / 剪贴板 / 托盘 / ShellOpen |

`Backend` 契约见 `ui/backend.go`，逐条注释了每个方法语义；缺失能力一律「空操作或返回零值，绝不 panic」。

### 事件契约

`ui.Event` / `ui.EventType` 与 `internal/display.Event` **数值逐项对应**（直接做类型转换）。
新增事件必须追加在枚举末尾，不能插入中间。移动端可用：

| 事件 | 用途 |
|---|---|
| `EvTouchDown/Move/Up` | 触摸/手写笔（后端同时合成鼠标事件，业务可忽略 `Pointer` 字段） |
| `EvTextInput` | 软键盘一次上屏的多字符文本（`Event.Text`） |
| `EvBack` | 返回键 / 返回手势 |

## Linux 后端实现要点

- **窗口**：purego 直接调用 `libX11`（`XOpenDisplay` / `XCreateSimpleWindow` / `XNextEvent`…），
  零 cgo；`XEvent` 按 64 位 Xlib 结构体**偏移量**解析（`internal/x11/event.go`，有单元测试）。
- **上下文**：`libEGL` 的 `eglGetPlatformDisplay(EGL_PLATFORM_X11_KHR, Display*)` +
  `eglCreateWindowSurface(xid)` + OpenGL 3.3 Core；Core Profile 创建失败会自动回退兼容配置。
- **GL 函数**：`eglGetProcAddress` 解析后交给 `gpu.SetGLProcLoader`，渲染器通过
  `internal/gpu/glport_unix.go` 的函数表调用（缺符号安全空操作）。
- **DPI**：读 X 资源 `Xft.dpi`（未设置按 96dpi = 1.0）。
- **窗口管理**：全屏 / 置顶走 EWMH（`_NET_WM_STATE_FULLSCREEN` / `_NET_WM_STATE_ABOVE`），
  关闭走 `WM_DELETE_WINDOW`，中文标题写 `_NET_WM_NAME`（UTF8_STRING）。
- **事件循环**：`XPending` 有事件就处理，没有就小睡 4ms（空闲不烧 CPU）。
- **视频/音频**：与 Windows 共用 `ui/media.go` / `ui/music.go`（ffmpeg 子进程解 rawvideo RGBA / s16le PCM），
  音频输出走 `internal/audio`：Windows 是 waveOut，Linux 把 PCM 管道写进 `paplay` / `aplay` / `pw-play` 的 stdin
  （天然背压 = 播放节奏限流，播放时钟按墙钟减 250ms 缓冲延迟估算）；
  Linux 没有系统解码器（`system_decoder_other.go` 返回明确错误），所以**必须装 ffmpeg**。
- **软件渲染**：`glGetString(GL_RENDERER)` 含 llvmpipe / softpipe / swrast 时，
  `ui/scale.go` 的自动档直接关闭整帧超采样（WSLg 无 GPU 加速，全屏超采样会慢到像黑屏）；
  `GOGUI_SS=1.5` 可手动覆盖。
- **全屏**：EWMH `_NET_WM_STATE_FULLSCREEN`；WSLg 下可用 `--fullscreen` 启动或 F11。
  `Expose` 事件会回读 `XGetGeometry` 同步尺寸（有些 WM 全屏时不发 ConfigureNotify）。

### Linux 已知限制（后续阶段）

1. **中文输入法**：一期用 `XLookupString`（Latin-1），中文需接 XIM/ibus（`XOpenIM`/`XCreateIC`）；
2. **文件拖入**：需实现 XDND 协议；
3. **全局热键**：需 `XGrabKey`；
4. **窗口图标**：需写 `_NET_WM_ICON`（逐像素 ARGB）；
5. **圆角窗口**：需 XShape 扩展；
6. **自绘标题栏拖动**：需 `_NET_WM_MOVERESIZE`（当前用系统标题栏）；
7. **Wayland**：通过 XWayland 运行；原生 Wayland 后端未实现。

## macOS 后端实现要点

- **objc 桥（零 cgo）**：`internal/objc` 用 purego 动态加载 `libobjc`，把 `objc_msgSend` 按参数形状注册成类型化函数
  （`SendRect3` = NSRect + 3 参数、`RectOf` = 返回 NSRect、`PointOf` = 返回 NSPoint…）。
  purego 在 darwin/amd64 支持栈传参、在 darwin/arm64 支持结构体打包，因此 32 字节的 `NSRect` 能正确传递。
- **窗口**：`NSWindow initWithContentRect:styleMask:backing:defer:` + 自定义 `NSView` 子类 `GoguiView`；
  事件用 `class_addMethod` + `purego.NewCallback` 注入 `mouseDown:` / `mouseDragged:` / `scrollWheel:` / `keyDown:`…
- **上下文**：`NSOpenGLPixelFormat`（Accelerated + DoubleBuffer + 24 位色 + OpenGL 3.2 Core）+ `NSOpenGLContext`，
  GL 函数经 `CGLGetProcAddress` 解析后交给 `gpu.SetGLProcLoader`。
- **事件循环**：`nextEventMatchingMask:untilDate:[NSDate distantPast]:inMode:dequeue:` 非阻塞泵一轮，
  再 `sendEvent:` 派发；单轮上限 256 个事件。
- **为什么尺寸靠轮询**：`resizeWithOldSuperviewSize:` / `viewDidChangeBackingProperties` 会带结构体参数，
  而 purego 的 `NewCallback` **不支持结构体参数**，因此改为每轮比较 `bounds`。
- **限制**：应用图标只能在 .app 包里（运行期改不了）、全局热键需 CGEventTap、托盘需 NSStatusBar（均未接）。
- **验证**：`darwin/amd64` 与 `darwin/arm64` 交叉编译通过；实机运行需要 macOS。

## Android 后端实现要点

- **cgo（移动端允许）**：`internal/android` 的 `android.go` 负责 EGL/GLES3 与输入，`bridge.c/h` 负责
  NativeActivity 回调装配，`activity.go` 用 `//export` 导出 `ANativeActivity_onCreate`。
- **窗口**：系统 NativeActivity 加载 `libgogui.so` 并调用导出符号；窗口就绪后
  `eglCreateWindowSurface(ANativeWindow*)` + `eglCreateContext(GLES3)`。
- **输入**：直接读 `AInputQueue`（不经过 Java），触摸 `ACTION_DOWN/MOVE/UP` 合成为鼠标事件，
  同时额外派发 `EvTouchDown/Move/Up`（`Event.Pointer` 标识手指）。
- **尺寸**：旋转/分屏通过轮询 `ANativeWindow_getWidth/Height` 检测。
- **构建**：`./build-android.ps1`（自动探测 NDK），产物 `dist/android/<abi>/libgogui.so`；
  清单用 `android:hasCode="false"` + `android.app.lib_name=gogui`，不需要 Java 代码。
- **限制**：软键盘 / 剪贴板 / 文件选择器需要 JNI 桥（一期未接，能力表标为不可用）。
- **验证**：NDK r26 实测编译通过（`arm64-v8a` 与 `armeabi-v7a`），导出符号 `ANativeActivity_onCreate` 已确认；
  实机运行需要 Android 设备。

## iOS 状态（待 macOS 验证）

- 实现：`internal/uikit`（cgo + `bridge_ios.m`：UIApplication/UIWindow/`GoguiEAGLView`/EAGLContext/UIKeyInput）
  + `ui/backend_ios.go`（触摸合成鼠标事件）。
- **验证状态**：iOS 强制 cgo + 外部链接，**无法在 Windows 上编译**；代码按 UIKit 文档编写，
  结构与 Android 的 EGL 路径一一对应，但尚未在 Xcode 里跑通，属于「已实现、待验证」。
- 构建（macOS）：`GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go build -buildmode=c-archive -o ios/libgogui.a ./mobile/ios`，
  Xcode 侧调用 Go 导出的 `StartGoUI()`。

## 自测清单

### Linux

```bash
ldd ./dist/gogui-linux-amd64      # 期望：not a dynamic executable（或仅 libc）
./dist/gogui-linux-amd64          # 期望：出现深色界面
```

1. 逐页切换 gallery 的 8 个案例，检查布局、中文、图表
2. 鼠标移动/点击/滚轮、窗口缩放、全屏（设置页）、关闭按钮
3. 输入框输入英文（中文需输入法支持，见上）
4. 高分屏：`Xft.dpi=192` 下字号与布局同步放大
5. 缺库场景：临时改名 `libEGL.so.1`，应打印安装提示而不是崩溃

### macOS / Web

```bash
./dist/gogui-darwin-arm64                      # 首次需解除隔离：xattr -d com.apple.quarantine
python -m http.server -d dist 8000             # 然后浏览器打开 /gogui.html（需 wasm_exec.js）
```

## 设计取舍（与 gio 的异同）

| 点 | gio | GoGUI |
|---|---|---|
| 平台窗口 | `app/*` 各端 `window` | `ui/backend_*.go` 各端 `Backend` |
| X11 | 纯 Go 实现 X 二进制协议 | purego 调 `libX11`（省 1.5 万行协议代码，仍零 cgo） |
| GL 加载 | 各端 `gpu/gl` 分别绑定 | `glport_{win32,unix,wasm}.go` 各自一张函数表 |
| 字体 | `font/opentype` + 各端 face | 保留现有光栅化，只换「字体来源」 |
| 触摸 | 原生 Pointer 事件 | 后端合成鼠标事件 + `Pointer` 字段（命中测试零改动） |
| 合成 | 无（全部由自己呈现） | Windows 可选 DirectComposition（网页与自绘同层） |

## 常见报错

| 报错 | 原因 / 处理 |
|---|---|
| `x11: 无法连接 X 服务器` | 无 `DISPLAY`（SSH / headless）；用 `ssh -X` 或 `Xvfb :99 & export DISPLAY=:99` |
| `egl: 找不到 libEGL` | 装 `libegl1`（Debian）或 `mesa-libEGL`（Fedora） |
| `gl: 只解析到 N 个 GL 函数` | 显卡驱动不支持 OpenGL 3.3；装 `mesa-utils` 用 `glxinfo` 确认 |
| 中文显示为方块 | 缺 CJK 字体；装 `fonts-noto-cjk`，或用 `ui.Options{FontFamily:"Noto Sans CJK SC"}` |
| `ui: 当前平台尚未实现窗口后端` | 目标平台后端未实现（macOS/Android/iOS 属后续阶段） |
