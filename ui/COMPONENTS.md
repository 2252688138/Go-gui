# GoGUI 组件模块化清单（一键引入）

所有组件都在 `ui` 包内，**按文件分模块**，彼此无隐式依赖：每个组件只依赖
`gogui/internal/*`（geom / gpu / text / win32）与本包基础节点（`Box/Row/Column/Stack/Text`），
把它们连同 `internal/` 一起拷进新项目即可使用（`go.mod` 里 module 名改成自己的）。

```go
import "gogui/ui"
```

---

## 1. 音乐播放器（含频谱）

| 文件 | 入口 |
| --- | --- |
| `ui/music.go` | `OpenMusic` / `OpenMusicPlayer` / `MusicSource` / `Spectrum` |
| `ui/music_player.go` | `MusicPlayer` / `MusicPlaylist` / `MusicSpectrum` / `MusicMirrorSpectrum` |

### 最小可用（一步打开 + 一步渲染）

```go
// 打开即播放；退出时释放 ffmpeg 子进程与音频设备
src, err := ui.OpenMusicPlayer("song.mp3")
if err != nil { log.Fatal(err) }
defer src.Close()

// view() 里每帧返回一个节点
return ui.MusicPlayer("mp", src, ui.MusicPlayerOptions{
    Track: ui.MusicTrack{Title: "曲名", Artist: "歌手", Album: "专辑", Cover: "cover.jpg"},
    Prev: func() { /* 上一首 */ },
    Next: func() { /* 下一首 */ },
})
```

### 只要频谱

```go
sp := ui.NewSpectrum(48)          // 48 个对数频段，独立于播放器使用
sp.PushPCM(pcm16, 2, 48000)       // 任意来源的 s16le PCM（解码线程调用）
bands := sp.Bands()               // UI 线程每帧读，0~1，已做 attack/release 平滑
peaks := sp.Peaks()               // 峰值（缓慢回落）

ui.MusicSpectrum("s1", sp, 40, "#f5c518", "#8b5cf6").St(ui.St().Wp(100).H(78))
ui.MusicMirrorSpectrum("s2", sp, 56, "#22d3ee", "#818cf8").St(ui.St().Wp(100).H(72))
```

### 播放列表

```go
ui.MusicPlaylist("pl", []ui.MusicTrack{
    {Path: "a.mp3", Title: "A"},
    {Path: "b.flac", Title: "B"},
}, active, func(i int) { play(i) })
```

支持的容器：mp3 / m4a / aac / flac / wav / ogg / opus / wma / ape。
文件对话框过滤器：`ui.MusicFilters`；扩展名判断：`ui.IsMusicPath`。

### 音源 API（`*MusicSource`）

`Play` / `Pause` / `TogglePlay` / `Seek` / `SeekRatio` / `SetSpeed` / `SetVolume` /
`ToggleMute` / `SetLoop` / `Tick` / `Pos` / `Duration` / `Progress` / `Spectrum` / `Close`。

---

## 2. 视频播放器

| 文件 | 入口 |
| --- | --- |
| `ui/video.go` | `Player` / `VideoClip` / `LoadGIF` / `LoadImageSequence` / `NewMediaPlayer` |
| `ui/video_paint.go` | `Video()` 节点的绘制层（帧纹理缓存） |
| `ui/video_player.go` | `VideoPlayer`（金色进度条 + 控制条 + 倍速 + 全屏） |
| `ui/media.go` | ffmpeg 解码源 `MediaSource`（`OpenMedia` / `OpenMediaWith`） |

```go
p, err := ui.NewMediaPlayer("movie.mp4")
if err != nil { log.Fatal(err) }
defer p.Close()
p.AutoHide = true
p.Play()

return ui.VideoPlayer("vp", p)     // 自带进度条 / 音量 / 倍速 / 循环 / 全屏
// 或只画画面：ui.Video(p).St(ui.St().Wp(100).H(360))
```

只播内存帧序列（GIF / 图片序列 / 自研解码）：

```go
clip, _ := ui.LoadGIF("anim.gif")
p := ui.NewPlayer(clip)
ui.VideoPlayer("vp", p)
```

---

## 2.5 WebView 内嵌浏览器（系统 WebView2）

| 文件 | 入口 |
| --- | --- |
| `ui/webview.go` | `NewWebView` / `NewWebViewIn` / `WebViewNode` / `Browser` / `WebViewBar` / `WebViewStatus` / `WebViewAvailable` |
| `internal/win32/webview2.go` | 纯 syscall/purego 的 COM 绑定（Environment / Controller / CoreWebView2） |
| `internal/win32/webview2loader/` | `//go:embed` 的 WebView2Loader.dll（自动释放，无需外部文件） |

```go
wv, err := ui.NewWebView(ui.WebViewOptions{HTML: demoHTML}) // 或 {URL: "https://…"}
if err != nil { /* 没有 WebView2 运行时：降级为占位卡片 */ }
defer wv.Close()

return ui.Column(
    ui.WebViewBar(wv, ui.WebViewBarOptions{
        Address:   addr,
        Key:       "addr",
        OnAddress: func(v string) { addr = v },
        OnGo:      func() { wv.Navigate(addr) },
        ShowTitle: true,
    }),
    ui.WebViewStatus(wv),
    ui.WebViewNode(wv).St(ui.St().GrowTo(1)), // 网页区域，尺寸由布局决定
)
```

- 组件按节点矩形同步浏览器子窗口的位置（`App.frame` 绘制时收集矩形 → `syncWebViews` 按 DPI 换算成物理像素），滚出视口 / 切页自动隐藏。
- 工具栏与状态行是自绘控件；地址栏回车提交用 `ui.InputSubmit`。
- 圆角/裁剪：`wv.SetRadius(12)` 对网页子窗口做 `SetWindowRgn` 区域裁剪（真裁掉，可随 DPI/尺寸重建）。
- 限制：HWND 托管的网页永远在 GL 内容之上，**无法被自绘浮层遮挡、也不能做 3D 变换**；示例页在菜单展开时会临时隐藏网页。
- 示例：`go run ./cmd/gallery --case webview`。

### 升级路线（B 阶段 · DirectComposition 合成）

当前进展：已用窗口区域实现**圆角裁剪**；DComp 合成层进行中（`internal/win32/abi_amd64.s` 已提供 float 参数调用桥，D3D11/DXGI/DComp 的 COM 槽位已核对到 SDK 头）。

目标：把 GL 输出改走 DXGI swapchain（`WGL_NV_DX_interop2`）+ `IDCompositionVisual::SetContent`，
网页改由 `ICoreWebView2Environment3::CreateCoreWebView2CompositionController` 托管，即可获得：
同树 z-order（菜单可压上去）、`SendMouseInput` 精确转发鼠标键盘、任意形状裁剪（`IDCompositionRectangleClip`）。
`internal/win32/webview2.go` 已按 WebView2.h 记录 CompositionController 的 vtable 槽位（`put_RootVisualTarget` / `SendMouseInput` / `SendPointerInput`）。

---

## 3. 基础与表单组件

| 文件 | 组件 |
| --- | --- |
| `ui/node.go` | `Box` / `Row` / `Column` / `Stack` / `Grid` / `Wrap` / `Masonry` / `EqualRow` / `Pick` / `If` / `Each` |
| `ui/widgets.go` | `Btn` / `Input` / `Switch` / `Checkbox` / `Radio` / `Slider` / `Progress` / `Ring` / `Table` / `Tabs` / `TabsBar` / `Accordion` / `Card` / `Tag` / `Toast` / `Modal` / `TooltipOf` / `MenuItem` 菜单 |
| `ui/image.go` | `Image` / `ImageSized`（PNG/JPEG/GIF/BMP，后台解码） |
| `ui/anim.go` | `Tween` / `Spring` / `Keyframes` |
| `ui/style.go` | `St()` 链式样式（尺寸 / 间距 / 圆角 / 渐变 / 阴影 / 绝对定位 / 毛玻璃） |

```go
ui.Btn("保存", ui.BtnPrimary, save)
ui.Input("name", value, "请输入", func(v string) { value = v })
ui.Switch("sw", on, func(v bool) { on = v })
```

---

## 3.5 标题栏与菜单（多级子菜单）

| 文件 | 组件 |
| --- | --- |
| `ui/menu.go` | `MenuItem`（`Children` 子菜单 / `Separator` / `Checked` / `Disabled`）/ `MenuDef` / `NewMenuBar` / `MenuBar` / `MenuLayer` / `MenuPanel` / `WindowButtons` |
| `ui/app.go` | `App.BeginDragMove`（自绘标题栏拖动）/ `App.ToggleMaximize` / `App.IsMaximized` / `Node.OnPressEvt` |

```go
mb := ui.NewMenuBar()
menus := []ui.MenuDef{{
    Label: "文件",
    Items: []ui.MenuItem{
        {Label: "打开", Icon: "▸", Children: []ui.MenuItem{
            {Label: "图片…", OnPick: openImage},
            {Label: "视频…", Hint: "Ctrl+O", OnPick: openVideo},
        }},
        {Separator: true},
        {Label: "退出", OnPick: quit},
    },
}}

// 每帧：标题栏 + 内容；菜单浮层必须放在根 Stack 里
root := ui.Stack(
    ui.Column(
        ui.Row(ui.MenuBar(mb, menus), ui.Spacer(), ui.WindowButtons()).
            OnPressEvt(func() { ui.Current().BeginDragMove() }), // 按在标题栏空白处拖动窗口
        body,
    ),
    ui.MenuLayer(mb, menus, anchorX, anchorY), // anchorX/Y = 菜单栏左边缘与底部
)
```

要点：`MenuLayer` 自带点击外部关闭与 ESC 关闭；顶部菜单展开时鼠标划过其它菜单自动切换；子菜单悬停展开，可无限嵌套。
鼠标离开菜单（菜单栏 + 浮层）后默认 3s 自动收起：`mb.AutoHide = 5*time.Second` 改时长，`mb.AutoHide = -1` 关闭自动隐藏；指针仍停在菜单内时不会隐藏。

---

## 3.6 Markdown 渲染

| 文件 | 入口 |
| --- | --- |
| `ui/markdown.go` | `Markdown` / `MarkdownOptions` / `MarkdownLightbox` / `MarkdownOpenImage` |
| `ui/markdown_parse.go` | `MarkdownParse`（块级 + 行内解析） |
| `ui/markdown_highlight.go` | 轻量语法着色（go / js / ts / py / json / bash / sql / html / css / yaml） |
| `ui/markdown_keywords.go` | 各语言关键字表 |
| `ui/markdown_state.go` | 页签 / 折叠 / 图片预览状态 |
| `ui/textwrap.go` | 与布局共用的折行算法（`WrapOn` 文本按测量宽度真实折行） |

```go
// 每帧渲染：宽度 760，自动折行 + 代码复制 + 图片预览
return ui.Scroll(ui.Markdown("md", src, ui.MarkdownOptions{Width: 760})).K("scroll-md")

// 图片放大层放在根 Stack 最后一层（点击 markdown 图片后全屏预览）
ui.Stack(page, ui.MarkdownLightbox())
```

支持：标题 / 段落 / 粗体 / 斜体 / 删除线 / 行内代码 / 链接 / 图片 / 有序无序列表 / 任务列表 /
引用与 GitHub 提示块（`[!TIP]` 等）/ 代码块（行号 + 着色 + 复制 + 超 22 行折叠）/ 表格 / 分隔线 /
`[toc]` / `[tabs]` / `[details]` / `[card]` / `[badge]` / `[progress]` / `[gallery]`，
`[video]` `[audio]` 显示为占位卡（真实播放用 `VideoPlayer` / `MusicPlayer`）。

内置限制：图片仅支持本地文件（远程 URL 降级占位卡）；不支持 Mermaid / 数学公式。

示例页：`go run ./cmd/gallery --case markdown`。

---
## 4. 系统能力


---

## 8. 文件拖入与系统操作能力（ui/system.go + internal/win32/system.go）

### 8.1 文件拖入（WM_DROPFILES）

链路：`NewWindow` 默认 `DragAcceptFiles(hwnd, 1)` → 松手时系统投递 `WM_DROPFILES`
→ `win32.dropEvent` 用 `DragQueryFileW` 取路径、`DragQueryPoint` 取落点（客户区物理像素）
→ `EvDropFiles` 事件 → `App` 缓存到 `dropPending` → 下一帧 `frame()` 里用落点撞树（`hitDropTarget`）
→ 最近的设了 `OnDrop` 的节点优先，否则给 `App` 级回调。

| API | 说明 |
| --- | --- |
| `Node.DropEvt(fn func(paths []string))` | 节点级拖入回调（沿祖先冒泡，需在区域内松手） |
| `ui.DropZone(key, hint, height, onDrop)` | 现成的拖入区域（图标 + 提示 + 边框） |
| `App.SetOnDropFiles(fn)` / `ui.SetOnDropFiles(fn)` | 应用级兜底回调（x/y 为落点逻辑坐标） |
| `App.SetAcceptDrops(bool)` / `ui.SetAcceptDrops(bool)` | 开关当前窗口的拖入接收 |
| `ui.PickFiles(title, filter)` | 多选文件对话框（`OFN_ALLOWMULTISELECT`） |
| `ui.FileRow(path, onClick)` / `ui.FormatBytes(n)` | 拖入结果行（名称 / 大小 / 目录）与字节格式化 |

注意事项：

- `WM_DROPFILES` 只在松手时到达，没有 dragenter/dragover，因此**没有悬停高亮**；
  需要悬停高亮必须改用 OLE `IDropTarget`，本框架未提供。
- 回调里 node 指针只在当前帧有效，因此分发统一放在 `frame()` 内（与 `OnClick` 同一时机）；
  不要在事件回调里直接改界面状态以外的树结构。
- 拖入路径不会自动过滤，需自行用 `ui.ImageSize` / 后缀判断；文件夹路径同样会出现在 `paths` 里。

### 8.2 系统操作能力

| API | 实现 | 说明 |
| --- | --- | --- |
| `ui.SetAlwaysOnTop(bool)` / `App.SetAlwaysOnTop` | `SetWindowPos(HWND_TOPMOST/NOTOPMOST)` | 窗口置顶 |
| `ui.SetClipboardText` / `ui.ClipboardText` | CF_UNICODETEXT | 文本剪贴板读写 |
| `ui.ClipboardSetFiles(paths)` | CF_HDROP（DROPFILES 头 + 双 NUL 宽字符表） | 复制文件引用到资源管理器 |
| `ui.RegisterHotkey(mods, vk, fn)` / `UnregisterHotkey(id)` | `RegisterHotKey` + `WM_HOTKEY` | 全局热键（失焦/最小化也触发） |
| `ui.SetKeepAwake(bool)` | `SetThreadExecutionState` | 阻止休眠与关屏 |
| `ui.RevealInExplorer(path)` | `explorer.exe /select,"path"` | 定位文件 |
| `ui.MoveToRecycleBin(path)` | `SHFileOperationW(FO_DELETE\|FOF_ALLOWUNDO)` | 移入回收站（可还原） |
| `ui.SystemDarkMode()` | 读注册表 `AppsUseLightTheme` | 跟随系统深浅色 |
| `(*Tray).Notify(title, msg)` | `Shell_NotifyIconW(NIF_INFO)` | 托盘气泡通知 |

热键修饰键常量：`ui.HotAlt / HotCtrl / HotShift / HotWin / HotNoRepeat`；
`RegisterHotkey` 内部已带上 `MOD_NOREPEAT`，返回 id（0 表示被其它程序占用）。
热键回调在 UI 线程执行，可直接改状态并 `ui.RequestRedraw()`。
| 文件 | 能力 |
| --- | --- |
| `ui/app.go` | `Run` / `Options` / `App`（`Minimize` / `SetFullscreen` / `Post`） |
| `ui/platform.go` | `OpenFileDialog` / `SaveFileDialog` / `PickFolderDialog` / `NewTray` / `NewWindow` |
| `ui/debug.go` | `LayoutForTest` / `DebugDump`（布局单测与 dump） |

`Options.DumpLayout` 可导出首帧布局树：`go run ./cmd/gallery --dump dump.txt --case music`。

---

## 5. 在 gallery 里看效果

```bash
go run ./cmd/gallery               # 全部示例（推荐）
go run ./cmd/gallery --case music   # 音乐播放器
go run ./cmd/gallery --case video   # 视频播放器
```

> 在 `cmd/gallery` 目录下可以用 `go run .`；不要用 `go run main.go`，
> 那样只会编译单个文件（已把所有示例页合并在 `main.go` 里，两者均可，
> 但后续新增文件时只用 `go run .` 才不会漏文件）。

示例页函数各自成段（`musicCase` / `videoCase` / `layoutsCase` 等），可以整段拷走。

---

## 6. 解码器回退（没装 ffmpeg 也能播）

| 后端 | 实现 | 入口 |
| --- | --- | --- |
| ffmpeg（优先） | 子进程解 rawvideo / s16le | `ui/media.go` |
| 系统解码（回退） | Windows Media Foundation（`IMFSourceReader`，纯 syscall，无 cgo） | `ui/system_decoder.go` + `internal/win32/mf.go` |

行为：`ui.OpenMedia` / `ui.OpenMusic` 先看 `ffmpeg` / `ffprobe` 是否在 PATH；
缺失时自动走系统解码器（视频 RGB32 → NV12 → YUY2 依次尝试后转 RGBA，音频 PCM → waveOut）。
`MediaInfo.Backend` 返回实际后端（`"ffmpeg"` / `"system"`），`ui.BackendName()` / `ui.FormatBackend()`
可直接用于界面展示。

```go
// 强制走系统解码器（验证回退路径 / 系统无 ffmpeg 时的默认行为）
os.Setenv("GOGUI_DECODER", "system")
src, err := ui.NewMediaPlayer("movie.mp4")
fmt.Println(src.Info().Backend) // system
```

能力对比：

| | ffmpeg 后端 | 系统解码后端 |
| --- | --- | --- |
| 容器 | mp4/mkv/mov/avi/webm/flv/ts/m3u8 | 系统支持的容器（mp4/mov/wav/mp3/m4a 等） |
| 编码 | 全部（含 HEVC/AV1 滤镜） | 系统已安装的 MFT（H.264 / AAC / MP3 / WMA…） |
| 最大分辨率限制 | 支持 `MediaConfig.MaxWidth/MaxHeight` | 跟随源尺寸 |
| 倍速 | 支持（atempo） | 1x（不重采样） |
| 拖动预览帧 | 单帧抓取 | 保留上一帧（无额外抓帧进程） |

> Media Foundation 的 `IMFSourceReader` 对跨线程调度敏感：所有调用（创建 / ReadSample /
> Seek / Release）都通过 `win32.MFRun` 串行在一个锁定线程上执行，否则会在创建第二个
> reader 时访问违例。

---

## 7. 注意事项

- 音频 / 视频解码都依赖外部 `ffmpeg` / `ffprobe`（PATH 中需存在，或通过
  `MediaConfig.FFmpeg` / `FFprobe` 指定路径）。
- 音源、视频源用完务必 `Close()`，否则 ffmpeg 子进程与音频设备不会释放。
- `MusicPlayer` / `VideoPlayer` 的 `key` 在同一界面内必须唯一（交互状态按 key 保存）。
- 窗口最小化会自动暂停渲染，恢复时 `ui` 会强制重绘并让媒体源重建解码窗口，
  画面不会停在最小化前的那一帧。
