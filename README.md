# GoGUI

纯 Go、零 cgo 的跨平台自绘 GUI 框架。界面主体全部由 OpenGL 自绘（不依赖 WebView），另带可选的内嵌浏览器组件（系统 WebView2）；风格对齐现代 Web UI（深色、圆角、渐变、投影、flex 布局），样式可完全自定义。

简体中文 ｜ [English](README.en.md)

```
┌──────────────────────────────────────────────┐
│  cmd/demo           示例应用                  │
├──────────────────────────────────────────────┤
│  ui                 声明式组件树 / 样式 / 布局 │
│                     事件路由 / 绘制 / 主题     │
├──────────────────────────────────────────────┤
│  internal/gpu       OpenGL 3.3 批处理渲染器   │
│  internal/text     opentype 矢量光栅化        │
│  internal/geom      几何基础类型              │
│  internal/win32     Win32/WGL 绑定（purego）  │
└──────────────────────────────────────────────┘
```

## 为什么不用 WebView 也能做到 Web 观感

| Web 能力 | 本框架的实现 |
|---|---|
| CSS 圆角 / 描边 | 片元着色器 SDF（解析式有向距离场），完美抗锯齿，任意半径 |
| 线性渐变 | 顶点插值双色渐变（`BgGrad`） |
| box-shadow | SDF 外扩 + 距离衰减近似高斯，一次 draw |
| flex 布局 | 两阶段 measure / arrange 约束求解引擎 |
| CSS 样式表 | Go 链式 DSL（类型安全、零解析开销） |
| DOM 事件 | 命中测试 + 冒泡到最近可交互祖先 |
| 文本排版 | `x/image/font/opentype` 矢量覆盖率抗锯齿 → GPU 灰度图集（与 gio 同源，中文清晰） |
| 高分屏 | per-monitor DPI v2 + 全部按逻辑像素布局 |

## 超高清渲染（整帧超采样）

渲聘器支持整帧 **SSAA**：整帧先按 N 倍分辨率画进离屏目标，再平均降采样回窗口像素；字形也按同一倍数光栅化。因此圆角、描边、阴影与文字都获得 N² 倍子像素覆盖率，边缘与小字号笔画明显更细腻。

```go
ui.SetSuperSample(ui.SSAuto) // 自动：DPI < 150% 时 2×，否则 1×（推荐）
ui.SetSuperSample(ui.SS200)  // 强制 2×
ui.SetSuperSample(ui.SSOff)  // 1:1，GPU 开销最低
// 另有 ui.SS125 / ui.SS150
```

- 逻辑布局、命中测试与鼠标坐标完全不变，只改变渲染分辨率（字体缓存、裁剪、圆角半径都自动跟随）。
- 像素预算 8192×4320：窗口很大时自动逐级降档（`2× → 1.5× → 1.25× → 1×`），避免 4K/5K 下填充率爆掉。
- 字形覆盖率 gamma 随倍率收敛（`1-(1-g)/ss`），避免超采样后文字过粗；字形图集在光栅倍率 ≥ 1.75 时自动放宽到 2048²。
- 降采样为 4×4 子带采样（box），并关掉混合整屏覆写，保证半透明背景也不叠两次。
- 运行时热切换：`SetSuperSample` 会重建渲染目标与字形图集，可放在设置页里做清晰度选项（gallery 设置页已内置）。

## 快速开始

```go
package main

import "gogui/ui"

func main() {
    ui.Run(ui.Options{
        Title:      "My App",
        Width:      1180,
        Height:     760,
        FontFamily: "Microsoft YaHei UI",
        Background: "#070a12",
    }, view)
}

func view() *ui.Node {
    return ui.Column(
        ui.Text("控制台").St(ui.St().Text(26, 700, "#eef2ff")),
        ui.Button("新建", func() { /* 点击回调 */ }).
            St(ui.St().Text(13, 600, "#fff").PadXY(18, 9).R(10).
                BgGrad("#6366f1", "#8b5cf6", 90).
                Shadow(18, 6, 0, "rgba(99,102,241,0.45)")),
    ).St(ui.St().Padding(28).GapOf(20))
}
```

```bash
go run ./cmd/demo
# 无控制台窗口的发布构建：
go build -ldflags "-H=windowsgui" -o app.exe ./cmd/demo
```

## 构建与体积优化

`build.ps1` 把发布构建参数固定下来（`-trimpath -buildvcs=false`、`-s -w -H windowsgui -buildid=`、`CGO_ENABLED=0`）：

```powershell
./build.ps1                 # gallery -> dist\gallery.exe
./build.ps1 -Target all     # gallery + demo + tools
./build.ps1 -Upx            # 再叠加 UPX -9（体积最小）
./build.ps1 -Dev            # 开发构建：保留符号 + 控制台日志
```

实测（gallery，go1.26 windows/amd64）：

| 构建方式 | 体积 | 说明 |
| --- | --- | --- |
| `go build`（默认） | 6.45 MB | 带符号表、调试信息、本机路径 |
| `build.ps1` | **4.54 MB** | −30%，去掉符号/调试信息与构建路径 |
| `build.ps1 -Upx` | **1.52 MB** | −76%，启动时自解压（常驻内存略增，偶发杀软误报） |

说明：
- 单文件、零外部依赖：全部走 `purego`/`syscall` 调系统 OpenGL 与 Win32，**不需要 CGO、不需要运行库**；
- `GOAMD64` 默认 `v1`（兼容旧 CPU），自用机器可设 `v3` 换取少量性能；
- 不想上 UPX 又要点体积：`-ldflags "-s -w"` 已是大头，剩下的只有图片解码器（gif/jpeg/png）与 `x/image` 字体光栅化，按需裁剪收益有限；
- 体积对比可用 `Get-ChildItem dist\*.exe | % { "{0:N2} MB  {1}" -f ($_.Length/1MB), $_.Name }`。

## 状态与交互

组件树是**每帧重建**的纯函数，交互状态由你自己的包级变量持有，改完状态调用 `ui.RequestRedraw()` 即可（线程安全）：

```go
var current = "dashboard"

ui.Button("组件库", func() {
    current = "components"
    ui.RequestRedraw() // 下一帧重建 → 重新布局 → 重绘
}).St(ui.St().Text(13, 500, "#8b93ad").PadXY(14, 10).R(10))
```

- 点击回调在**下一帧**基于最新组件树触发（避免旧指针失效），无需给节点设置 `Key`
- `hover` / `pressed` 高亮由框架自动叠加（`OnClick != nil` 的节点）
- 滚动偏移按节点 `Key` 保存在 App 内，滚动容器请设置 `.K("唯一名")`

## 内置组件

除了 `Box/Row/Column/Center/Spacer/Text/Button/Scroll/Stack` 等基础节点，`ui/widgets.go` 提供开箱即用的组合控件（状态由调用方持有，回调内调用 `ui.RequestRedraw()`）：

| 分类 | 组件 |
|---|---|
| 按钮 | `Btn`（5 变体）`IconButton` `Button` `Link` |
| 表单 | `Input` `PasswordInput` `Checkbox` `Radio` `Switch` `Slider` `Progress` `Ring` |
| 展示 | `Card` `Title` `Divider` `Tag` `Dot` `Avatar` `Badge` `KeyValue` `Empty` `Skeleton` `Spinner` |
| 导航 | `Tabs` `TabsBar` `Breadcrumb` `Accordion` |
| 数据 | `Table`（固定/弹性列宽、斑马纹、整行点击）`ListItem` |
| 媒体 | `Video`（画面节点）`VideoPlayer`（播放器）`Player` `VideoClip` `MediaSource`（ffmpeg 解码） |
| 浏览器 | `WebView`（系统 WebView2 内嵌真实网页）+ `WebViewNode` / `WebViewBar` / `WebViewStatus` |
| 浮层 | `Modal` `ModalEx` `MenuPanel` `TooltipOf` + `TooltipLayer` `Toast` `Banner` |

支撑这些组件的引擎能力：
- `Stack` 叠层容器：子节点重叠绘制，后声明者在上层（开关旋钮 / 进度条 / 浮层）
- 绝对定位样式：`AbsFill` `AbsTL` `AbsTR` `AbsBL` `AbsBR` `AbsCenter` `AbsBottom`
- 焦点与键盘：`St().FocusOn()` + `Node.OnCharEvt/OnKeyEvt`，配合 `ui.SetFocus/IsFocused`
- 逐帧动画：`St().AnimOn()` + `ui.AnimTime()`，存在动画节点时自动持续重绘
- 带坐标点击：`Node.TapAt(func(x, y float32))`（滑块按点击位置取值）
- 拖动：`Node.DragEvt(func(x, y float32))`，坐标相对节点（滑块、滚动容器）
- 悬停：`Node.OnHoverEvt` 按 Key 去重触发，配合 `ui.MousePos()` 做 Tooltip
- 禁用：`Node.DisabledOn(true)` 整棵子树不参与命中
- 滚动条：`Scroll` 内容溢出时自动绘制 4px 半透明圆角条
- 文本测量：`ui.TextWidth(s, size, weight)`
- 提示浮层：`ui.SetTooltip` / `ui.TooltipLayer()`（放在根 Stack 最上层）

## 界面案例

`cmd/gallery` 提供 8 个可直接复用的界面骨架与布局实验页，左侧列表切换，点击与输入均可交互：

| 案例 | 内容 | 关键能力 |
|---|---|---|
| 登录页 | 品牌头 / 账号密码 / 记住我 / 校验提示 | `Input` `Checkbox` 焦点输入 |
| 仪表盘 | 4 个 KPI 卡 / 进度条 / 活动列表 | `Card` `Progress` `ListItem` `Grow` 分栏 |
| 设置页 | 开关 / 单选 / 滑块 / 折叠 / 输入 | `Switch` `Radio` `Slider` `Accordion` |
| 聊天 | 左右气泡 / 会话头 / 发送输入 | `Scroll` + `Spacer` 对齐气泡 |
| 数据表格 | 表头 / 斑马纹 / 状态标签 / 行内进度 | `Row` 定宽列 + `Clipped` 圆角表格 |
| 布局方式 | 8 种布局能力逐项演示，可拖动窗口看响应式 | `Grid` `Wrap` `Masonry` `Stack` `EqualRow` `JustifyTo` `Pick` |
| 视频播放器 | 内置合成片段 + 打开 GIF / 本地视频（mp4/mkv/mov/avi/webm/flv/ts/m3u8）+ 直链 | `VideoPlayer` `MediaSource` 逐帧纹理 |

```bash
go run ./cmd/gallery                       # 打开案例程序
go run ./cmd/gallery --dump d.txt --case table   # 只跑指定案例并导出布局树
```

## 布局系统

`Row` / `Column` 是 flex 容器，默认 `AlignStretch`（与 CSS `align-items: stretch` 一致）。

| 方法 | 含义 |
|---|---|
| `.W(v)` `.H(v)` | 固定尺寸 |
| `.Wp(p)` `.Hp(p)` | 按父容器百分比 |
| `.MinWidth` `.MaxWidth` `.MinHeight` `.MaxHeight` | 尺寸约束 |
| `.GrowTo(n)` | 主轴剩余空间按比例分配 |
| `.Padding(v)` `.PadXY(h,v)` `.PadOnly(t,r,b,l)` | 内间距 |
| `.GapOf(v)` | 子项间距 |
| `.AlignTo(a)` `.JustifyTo(a)` | 交叉轴对齐 / 主轴分布（`AlignSpaceBetween` `AlignSpaceAround` `AlignSpaceEvenly`） |
| `.Self(a)` | 单项覆盖父容器交叉轴对齐（align-self） |
| `.Clipped()` | 裁剪溢出内容 |
| `.CenterY` (Style 字段) | 文字在自身框内垂直居中（`Button` 默认开启，普通 `Text` 顶部对齐） |

### 网格 / 换行 / 瀑布流 / 响应式

除 `Row` / `Column` / `Stack` 外，引擎还内置四种容器与两种响应式工具：

| 构造器 | 说明 | 参数 |
|---|---|---|
| `Grid(...)` | 等宽列网格，行高取该行最高子项，单元格默认填满 | `.St(St().ColsN(3))` 固定列数；`.St(St().ColWidth(180))` 按列宽自适应列数 |
| `Wrap(...)` | 流式换行（`flex-wrap: wrap`），行内支持 `JustifyTo` 分布 | 可用宽不足时自动折行 |
| `Masonry(...)` | 瀑布流：子项依次落入当前最矮的列，高度可不等 | 列参数同 `Grid` |
| `EqualRow(...)` / `EqualCol(...)` | 主轴等分栏（自动给子项设 `Grow=1`） | — |
| `Pick(xs, sm, md, lg)` | 响应式：按视口宽度（<640 / <960 / <1280 / 其余）选择构造函数 | 传 `nil` 自动回退到可用分支 |
| `BP(xs, sm, md, lg)` | 按同一断点取任意值（如网格列数） | 泛型 |
| `ViewportW()` / `ViewportH()` | 最近一次布局的视口尺寸 | — |

```go
ui.Grid(tiles...).St(ui.St().ColWidth(180).GapOf(12))      // 自适应列数的网格
ui.Wrap(tags...).St(ui.St().GapOf(8)).JustifyTo(ui.AlignSpaceBetween)
ui.Masonry(cards...).St(ui.St().ColsN(3).GapOf(12))         // 瀑布流
ui.EqualRow(a, b, c).St(ui.St().GapOf(10))                  // 三栏等分
ui.Pick(narrow, wide, wide, wide)                          // 窄屏切换布局
```

## 样式能力

```go
ui.St().
    Text(15, 600, "#e8ecff").     // 字号 / 字重 / 颜色
    BgGrad("#1e293b", "#0f172a", 90). // 线性渐变（角度）
    R(14).                        // 圆角（Rx 可逐角）
    Stroke(1, "#2c3550").         // 描边
    Shadow(24, 8, 0, "rgba(0,0,0,0.4)") // 模糊 / Y 偏移 / 扩散 / 颜色
```

颜色支持 `#rgb` `#rrggbb` `#rrggbbaa` `rgb()` / `rgba()` / `0xRRGGBB` / `gpu.Paint`。
交互态（hover / pressed）由框架自动叠加，无需手写。

## 渲染管线

1. `measure`：自底向上算尺寸（文字用 opentype 度量）
2. `arrange`：自顶向下定位 + 分配 grow
3. 绘制：按顺序产生 shape / text 顶点，遇到类型或纹理切换才 flush，合批提交 GPU
4. 裁剪：`glScissor`（嵌套取交集），滚动容器自动裁剪

顶点格式（shape）：`pos | local | color | color2 | halfSize | radius4 | params`，一个矩形 6 个顶点。

## 目录

```
gogui/
├── ui/            组件树、样式、布局、事件、主题、调试
│   ├── app.go     窗口/事件循环/绘制
│   ├── layout.go  measure + arrange
│   ├── node.go    声明式节点与构造器
│   ├── style.go   样式 DSL
│   └── debug.go   布局树打印
├── internal/
│   ├── gpu/       GLSL 着色器、批处理、纹理、裁剪
│   ├── text/      opentype 字形光栅化、图集、度量
│   ├── win32/     Win32 + WGL 绑定（purego 动态加载）
│   └── geom/      几何类型
├── cmd/demo/      示例
└── tools/         fonttest / atlasdump / layouttest 调试工具
```

## 调试

```bash
go run ./tools/layouttest      # 打印完整布局树（无需窗口）
go run ./tools/fonttest        # 打印字形度量
go run ./tools/atlasdump       # ASCII 打印字形图集
```

运行时可设置 `ui.Options.DumpLayout = "layout.txt"`，首帧后导出布局树并退出。

## 跨平台路线

当前实现为 Windows（Win32 + WGL）。`internal/gpu.Backend` 接口已抽象，新增平台只需实现：

| 平台 | 窗口层 | 渲染层 |
|---|---|---|
| Windows | Win32（已完成） | OpenGL 3.3（已完成） |
| macOS | Cocoa `NSWindow` + `CAMetalLayer` | Metal 或 OpenGL 4.1 |
| Linux | X11 / Wayland | OpenGL / Vulkan |
| Android | `ANativeWindow` + EGL | GLES 3 |
| Web | `canvas`（鼠标 / 键盘 / 滚轮事件） | WebGL 2 + WebAssembly（已完成） |

上层 `ui` 包与平台无关，换后端不需要改动业务代码。

### 浏览器端（web/）

```bash
./web/build.ps1                                   # -> web/dist/main.wasm + index.html + wasm_exec.js + font.ttf
go run ./web/server -dir web/dist -addr 127.0.0.1:8088 -open
```

- `web/gl`：WebGL2 封装与单着色器管线（圆角 SDF / 渐变 / 阴影 / 字形 / 图片，一次 drawArrays）。
- `web/draw`：2D 图元（圆角矩形、竖向渐变、投影、任意方向线段、字形四边形），逻辑坐标按 `Scale` 换算设备像素。
- `web/text`：用 `x/image/font/sfnt` 读 TTF，自写扫描线光栅器把字形压成 8bit 覆盖率写进 2048² R8 图集（对应桌面端 DirectWrite）。
- `web/main.go`：浏览器端演示界面（顶栏 / 侧栏 / 概览 / 组件 / 图表），交互（悬停、点击、开关、滑块、输入框、滚轮）全部在 Go 侧命中计算。
- 运行参数加 `?debug` 可在左下角看到帧率、顶点数与采样像素颜色。

## 系统能力（M2）

### 图片

```go
ui.Image("logo.png")                       // 尺寸取原图
ui.ImageSized("photo.jpg", 280, 175).St(ui.St().R(14).Clipped())
```

支持 PNG / JPEG / GIF / BMP。`ui.ImageSize` 只读图片头用于布局测量，`DecodeImage` 解码为 NRGBA 后上传 RGBA 纹理，绘制走独立图片批次（圆角遮罩 + tint）。

### 毛玻璃 backdrop-filter

```go
ui.Box(...).St(ui.St().Backdrop(22, "#0b102066").Stroke(1, "rgba(255,255,255,0.28)"))
```

`gpu.Backdrop` 先 flush 当前批次，把已绘制区域 `glCopyTexSubImage2D` 到离屏纹理，经 1/2 → 1/4（blur ≥ 14 时再 1/8 → 回 1/4）双 Kawase 采样模糊，最后按圆角回绘并叠加 tint；FBO 与纹理逐帧复用。

### 多字体与彩色 Emoji

```go
ui.Text("Consolas 0123").St(ui.St().Text(14, 400, "#cbd5e1").Font("Consolas"))
```

字形图集改为 RGBA8：普通字形写 `(v,v,v,255)` 走 alpha 混合；Emoji 写真实颜色并在顶点带 `mode=1` 直出。主字体缺字形时依次回退 `Segoe UI Symbol` → GDI 彩色 Emoji（`Segoe UI Emoji` 渲染到 32bpp DIB，第二遍白色蒙版补齐深色像素 alpha）。

### 动画

```go
x := ui.Tween("panel", target, 0.45, ui.EaseOutCubic)   // 补间
s := ui.Spring("knob", target, 210, 15)                 // 弹簧积分
k := ui.Keyframes("pulse", []ui.Key{{At: 0}, {At: 0.5, Value: 1}, {At: 1}}, 2.4, true)
```

缓动集合：Linear / Quad / Cubic / Quart / Back / Elastic / Bounce。只有存在活动动画时才置位 `animBusy`，主循环据此继续出帧，静止时零空转。

### 滚动条

内容溢出时在右侧绘制圆角滑块：悬停/拖动加宽到 8px 并加深，闲置 0.9s 后淡出；支持直接拖拽滑块（`updateScrollbar`），拖拽期间不触发业务点击。

### 输入法（IME）

`WM_IME_STARTCOMPOSITION / COMPOSITION / ENDCOMPOSITION` 全链路接入：

- `GCS_RESULTSTR` 上屏 → 逐字符派发 `EvChar`（不再重复产生 `WM_CHAR`）
- `GCS_COMPSTR` 组合串 → `ui.ComposingText()` / `ui.ComposingCaret()`，Input 用高亮块显示预编辑文本
- 焦点输入框位置经 `ImmSetCompositionWindow` / `ImmSetCandidateWindow` 同步候选窗

### 多窗口 / 托盘 / 文件对话框

窗口默认居中：主窗口与 `NewWindow` 子窗口创建时都按所在显示器的工作区（排除任务栏）水平 + 垂直居中，多屏按窗口所在屏计算；运行中可用 `ui.Current().Center()` 重新归位。

```go
ui.NewWindow(ui.Options{Title: "子窗口", Width: 520, Height: 360}, view)
ui.NewTray("提示", onClick, []ui.TrayItem{{ID: 1, Label: "显示主窗口"}}, onMenu)
ui.OpenFileDialog("选择图片", "图片|*.png;*.jpg|所有文件|*.*")
ui.PickFolderDialog("选择目录")
ui.SaveFileDialog("保存", "文本|*.txt", "txt")
```

多窗口共用一个 UI 线程与消息泵，每个窗口拥有独立 GL 上下文、字形图集与布局树；`ui.Current()` 返回当前活跃窗口，`App.Post` 可跨线程投递回调。托盘依附窗口的 `WM_TRAYICON` 消息，右键弹出 `TrackPopupMenu` 菜单。窗口全屏用 `App.SetFullscreen(true)` / `App.IsFullscreen()`（去标题栏 + 覆盖当前显示器，ESC 自动退出）。

窗口圆角：启动参数 `ui.Options.CornerRadius`（逻辑像素，0 = 系统默认）或运行中 `App.SetCornerRadius(14)`（`<=0` 取消）。Win11 走 DWM 原生圆角（`DWMWA_WINDOW_CORNER_PREFERENCE`，边缘平滑），DWM 不可用时自动回退 `CreateRoundRectRgn` + `SetWindowRgn` 区域圆角；窗口缩放/DPI 变化时区域自动重建，进入全屏自动取消、退出全屏恢复。无标题栏窗口（`NoTitleBar`）创建时还会把 `DWMWA_BORDER_COLOR` 置为 `DWMWA_COLOR_NONE`，去掉 DWM 默认绘制的 1px 白色系统边框（旧系统回退关闭 DWM 非客户区渲染）。

## 已知限制

- 富文本（行内混排 / 图文环绕）与竖排未实现
- 虚拟列表尚未内置（可用分页手动实现）
- 视频解码依赖外部 ffmpeg / ffprobe（子进程，大分辨率软解 CPU 占用较高）

## 视频播放器

`KVideo` 是画面节点，`Player` 保存播放状态，`VideoPlayer` 把两者拼成带控制条的组件。

### 直接播放 mp4 / mkv / mov / avi / webm / flv / ts / m3u8

框架自带 ffmpeg 解码源（子进程，无 cgo），需 PATH 里有 `ffmpeg` / `ffprobe`：

```go
p, err := ui.NewMediaPlayer("movie.mkv")        // 或 m3u8 直链
if err != nil { /* 未安装 ffmpeg / 无视频流 / 链接不可达 */ }
p.Play()
defer p.Close()                                  // 释放 ffmpeg 进程与音频设备
ui.VideoPlayer("v", p).St(ui.St().H(420))

// 需要控制解码参数时：
src, err := ui.OpenMediaWith("4k.mp4", ui.MediaConfig{MaxHeight: 720, HWAccel: true})
p = ui.NewPlayerOf(src)
```

- 探测：`ui.ProbeMedia(path)` → `MediaInfo`（时长 / 尺寸 / 帧率 / 编解码 / 是否有音频），支持旋转元数据（宽高自动互换）
- 画面：ffmpeg `-f rawvideo -pix_fmt rgba` 管道 → 帧环（默认跟随源尺寸，可用 `MaxWidth/MaxHeight` 限制），按播放位置取帧，跳转时 `-ss` 重启窗口
- 声音：第二个 ffmpeg 进程输出 s16le/48k/立体声 → WinMM `waveOut`（`internal/win32/audio.go`），倍速用 `atempo` 滤镜重建音频链路
- 同步：有音频时以 waveOut 已播样本为时钟（`MediaClock`），播放器位置跟随音频；音量 / 静音 / 倍速 / 暂停均直达解码源
- 工具函数：`ui.IsMediaPath(path)`、`ui.MediaFilters`（文件对话框过滤器）、`ui.MediaFormats()`
- 其它帧源：`ui.LoadGIF` / `ui.LoadImageSequence` / 自定义 `ui.VideoSource`（`Size/Duration/Frame`）

```go
// 内存帧序列（GIF 动图 / 图片序列 / 自研解码）
clip, err := ui.LoadGIF("a.gif")
p := ui.NewPlayer(clip)
```

`Player` 方法：`Play/Pause/TogglePlay`、`SetPos/SeekBy/SeekRatio`、`SetSpeed`、`SetVolume/ToggleMute`、`SetLoop`、`Progress/Pos/Duration/Size/Frame/FrameIndex`；回调 `OnPlay/OnPause/OnSeek/OnVolume/OnLoopChange/OnFullscreen`。

字段开关：`Fit`（`FitContain` 留边 / `FitCover` 裁切 / `FitFill` 拉伸）、`AutoHide`（播放中控制条 2.5s 后自动隐藏）、`HideControls`（纯画面）、`Subtitle`（底部字幕）、`Fullscreen`（播放器铺满窗口）、`StepMode`；回调另有 `OnPrev`/`OnNext`（上一集/下一集）。

控制条样式（流媒体站点风格：金色进度条 + 无底图标按钮 + 右侧倍速面板）：全宽进度条点击/拖动跳转，已播放段金色、末尾白色圆点，悬停加粗；图标行含播放/暂停（形状绘制，不依赖字体字形）、上一集/下一集（`OnPrev`/`OnNext` 非空时出现）、音量图标（悬停时在上方弹出竖向音量滑块，0 在底部）、`当前 / 总时长`；右侧依次是「循环」胶囊、「倍速」胶囊（展开 0.5x / 0.75x / 正常 / 1.25x / 1.5x / 2x 面板并高亮当前项）和四角形状绘制的全屏按钮——点击切换 `Player.Fullscreen`，同时调用 `App.SetFullscreen` 让窗口去标题栏覆盖整个显示器，播放器在其根 `Stack` 内 `AbsFill` 铺满桌面（ESC 或再点按钮恢复）。音量图标是形状绘制的喇叭（底座 + 渐高喇叭口 + 声波，静音时变暗去声波），与播放/暂停同构，保证同行按钮基线对齐；悬停时在其上方弹出竖向滑块（0 在底部）。暂停时画面中央显示双环播放按钮，`Player.Subtitle` 非空时在画面底部居中显示字幕，画面获焦后 Space / ← → / ↑ ↓ / Home End 快捷键。

画面左侧中间另有竖排两个圆形工具按钮：**旋转**（每次 90°，循环 0/90/180/270，`Player.RotateBy(1)` / `SetRotation(3)`，激活时金色高亮）与**锁定**（`Player.Locked` / `ToggleLock()`）：锁定后点击画面、键盘快捷键、底部菜单（播放/音量/倍速/全屏/跳集）全部失效，底部只保留一条只读进度条，中央播放按钮也不再显示，只留左侧两个工具按钮可解锁。拖动进度时不会每个事件都重启 ffmpeg：`MediaScrub` 先把目标位置记下并在拖动期间直接返回现有帧（等待预算降到 80ms），松手后一次性 `-ss` 定位，拖动因此不再卡顿。旋转通过旋转纹理采样坐标实现（`gpu.Image.Rot`），90°/270° 时画面宽高自动互换，`Fit` 按交换后的比例计算。

渲染层：每帧比对帧号，帧变化时用 `glTexSubImage2D` 局部更新纹理（尺寸变化才重建并 `DeleteTexture`），纹理按 `*Player` 缓存于窗口。

## 性能与内存

- **字形图集按需扩容**：每个字体从 256²(256KB) 起步，装不下逐级翻倍到 1024²(4MB)，字体实例上限 24 个（LRU 回收并释放纹理）。旧版每个字体固定 2048²（16MB），一个页面用十几种字号就会吃掉上百 MB。
- **图集局部上传**：只把新光栅化的字形矩形 `glTexSubImage2D` 上去，不再每出现一个新字就整张重传。
- **字体文件内存映射**：`msyh.ttc` 等用 `MapViewOfFile` 映射，数据由系统页缓存承载，不占 Go 堆。
- **顶点缓冲复用**：容量足够时孤儿化 + `glBufferSubData`，避免每帧重新分配 VBO。
- **按需重绘**：鼠标移动只做命中测试，悬停目标/滚动条状态没变就不重绘整帧；窗口最小化时不渲染；滚动容器视口外的子树直接跳过。
- **图片异步解码**：图片在后台 goroutine 解码（并发上限 4），UI 线程只做纹理上传，滚动到大量图片时不再卡顿。
- **媒体**：帧环上限 48MB / 2 秒；ffmpeg 解码线程限制为 CPU 核数一半（2~4）。

内存构成可用环境变量 `GOGUI_MEMSTATS=1` 观察（每 120 帧向 stderr 打印一次）：

```
[gogui] frames=120 heap=19.4MB inuse=20.6MB sys=27.7MB gc=2 fonts=13 atlas=3.2MB images=0
```

播放视频时的 ffmpeg 子进程：视频 1 个 + 音频 1 个（`MediaConfig.NoAudio` 或源无音轨时只有 1 个）；拖动进度时额外 1 个单帧预览进程，松手即退出。

## 滚动与滚动条

- 滚轮一格 96px，并做缓动（约 60ms 收敛），不再「一格一跳」。
- 内层滚动容器到底/到顶后，剩余滚动量冒泡给外层容器。
- 拖动滑块时指针移出 12px 热区仍继续跟手；点击轨道空白处滑块中心对齐指针。
- 滚动条闲置 0.7s 后淡出，悬停/拖动时加宽加深。


## 内嵌浏览器（WebView2）

`ui.WebView` 把系统 WebView2（Edge 内核）当普通节点用：网页由 Chromium 的 GPU 进程渲染，
滚动 / 输入法 / 视频 / canvas / 右键菜单全部是浏览器原生行为。

```go
wv, err := ui.NewWebView(ui.WebViewOptions{URL: "https://example.com"})
if err != nil { /* 没有 WebView2 运行时 */ }
defer wv.Close()

return ui.Column(
    ui.WebViewBar(wv, ui.WebViewBarOptions{
        Address:   addr,
        Key:       "addr",
        OnAddress: func(v string) { addr = v },
        OnGo:      func() { wv.Navigate(addr) },
        ShowTitle: true,
    }),
    ui.WebViewNode(wv).St(ui.St().GrowTo(1)), // 网页区域，尺寸由布局决定
)
```

- **只落在节点矩形里**：每帧绘制时记录 WebView 节点的矩形，按 DPI 换算成物理像素后同步给浏览器子窗口；滚出视口 / 切页 / 隐藏后自动隐藏。
- **离线页面**：`ui.NewWebView(ui.WebViewOptions{HTML: "<h1>hi</h1>"})` 或 `wv.LoadHTML(...)`。
- **API**：`Navigate` / `LoadHTML` / `Back` / `Forward` / `Reload` / `Stop` / `Focus` / `SetZoom` / `ExecuteScript` / `URL` / `Title` / `Loading` / `CanGoBack` / `CanGoForward` / `Close`；`ui.WebViewAvailable()` 探测运行时。
- **依赖**：系统 WebView2 运行时（Win11 自带，Win10 装 Evergreen Runtime）。`WebView2Loader.dll` 已 `//go:embed` 进二进制，无需外部文件；找不到文件时自动释放到 `%LOCALAPPDATA%\GoGUI\WebView2Loader\`。
- **圆角 / 裁剪**：`wv.SetRadius(12)` 会对网页子窗口做 `SetWindowRgn` 区域裁剪（真裁掉，不是遮罩），随尺寸/DPI 重建；`WebViewNode(...).St(St().R(12))` 只影响占位卡片外观。
- **限制（HWND 托管）**：网页像素不进入 OpenGL 批次，因此**无法被菜单/Modal 遮挡**（浮层展开时组件会临时隐藏网页），也不能做 3D 变换。想同层混合需 DirectComposition 合成（B 阶段，进行中）。
- **示例**：`go run ./cmd/gallery --case webview`。
## WebView 两种渲染模式（A / B）

| 模式 | 触发 | 网页呈现 | 能被菜单盖住 | 圆角 | 交互 |
|---|---|---|---|---|---|
| **A · HWND 托管** | 默认（未设 `GOGUI_COMPOSE=1`） | Chromium 子窗口叠在 GL 内容上 | ❌（浮层展开时临时隐藏网页） | `SetWindowRgn` 硬裁剪 | 系统原生 |
| **B · DComp 合成** | `GOGUI_COMPOSE=1` 且驱动支持 `WGL_NV_DX_interop2`（**实验性**） | 网页是 DirectComposition 图层，与 GL 输出同树 | ✅ | `IDCompositionRectangleClip` 抗锯齿 | 需转发鼠标（`SendMouseInput`） |

- B 模式**不需要 cgo**：DirectComposition 的 float 接口通过运行时机器码 trampoline（`movd xmm0,r8d`）调用。
- 切换方式：默认 A；设 `GOGUI_COMPOSE=1` 尝试 B，失败自动回退 A。
- 实现要点：
  - `internal/win32/compose.go`：D3D11 设备 → `DCompositionCreateDevice` → 视觉树 → `IDCompositionSurface` → `BeginDraw/EndDraw`，GL 纹理经 `wglDXRegisterObjectNV` 绑定到 D3D11 纹理，零 CPU 拷贝。
  - `internal/gpu/renderer.go`：`SetTarget(fbo, w, h)` 支持离屏目标（B 模式下 `frame()` 先 `BeginFrame()` 再绘制）。
  - `internal/win32/webview2.go`：`WebView2Options.CompositionRoot` 非 0 时走 `ICoreWebView2Environment3::CreateCoreWebView2CompositionController`。
- **坑（已修正，勿回退）**：dcomp 运行时虚表把「动画版」排在「float 版」前面，与 SDK 头文件相反——
  `SetOffsetX(float)=4`、`SetOffsetY(float)=6`、`SetClip=14`，矩形裁剪的左右上下/四角半径 float 版全在偶数槽（4/6/8/10/12…26）。用 3/5/7/9/11/13 会把 float 当动画指针解引用，崩在 `addr=0x50`。