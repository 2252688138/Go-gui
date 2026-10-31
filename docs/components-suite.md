# 组件套件补全（P0–P2，约 60 项）

> 目标包 `gogui/ui`；演示：`go run ./cmd/gallery --case suite`。
> 约定不变：节点树每帧重建，状态按 `Key` 存在包级 map / 状态对象里；
> 浮层由对应 `*Layers()` 放在根 `Stack` 末尾。

## 0. 基础设施

| 能力 | 入口 | 说明 |
| --- | --- | --- |
| 富文本 | `RichText(spans, opts)` / `OnSpanClickEvt` | `Kind=KRichText`，行内折行/绘制/命中复用同一份 `richLines`；支持链接、行内代码、下划线/删除线、`MaxLines` 省略 |
| 焦点 | `PushFocusScope` / `PopFocusScope` / `FocusNext` / `FocusPrev` / `FocusVisible` | Tab 遍历、模态焦点陷阱、`Enter/Space` 激活、统一焦点环（`Style.NoFocusRing()` 可关） |
| 覆盖层栈 | `OpenOverlay` / `CloseOverlay` / `OverlayDepth` | ESC 从最上层逐层消费；Modal/下拉/日历/命令面板/引导均已接入 |
| 主题 token | `SetTokens` / `TokensOf` / `RegisterTheme` / `UseThemeName` | 语义色（surface/onSurface/border/accent/danger/focusRing…）+ 用户覆盖 + 第三套主题 |
| 过渡 | `KeepAlive` / `Scrim` / `PopOffset` / `Duration` / `ReducedMotion` | 浮层进出场；`ReducedMotion` 自动归零（Windows 读系统动画设置，Web 读 `prefers-reduced-motion`） |
| 编排 | `SequenceAt` / `ParallelAt` / `StaggerAt` / `AnimatedNumber` / `FormatNumber` / `ShimmerPhase` | 串行/并行/错峰、数字滚动、骨架流光 |
| 偏好 | `SetPref` / `GetPref*` / `SavePrefs` / `PrefsPath` | JSON 原子写、400ms 去抖、Web 用 localStorage |
| FLIP | `FLIP` / `FLIPFrom` / `ListEnter` / `TabSlide` | 绘制期位移，不改布局 |
| 无障碍 | `A11yOn(role,name,desc)` / `A11ySnapshot` | 元数据导出（尚未注册 UIA Provider） |
| i18n/RTL | `LoadCatalog` / `SetLocale` / `T` / `Tf` / `SetRTL` / `MirrorRow` / `MirrorIcon` | 字符串外部化 + 行镜像骨架 |

### 根 Stack 末尾需要哪些 Layer

```go
ui.Stack(page,
    ui.SelectLayers(), ui.DateLayers(),          // 原生下拉 / 日历
    ui.AutoCompleteLayers(), ui.TimeLayers(),    // 自动完成 / 时间
    ui.CascaderLayers(), ui.ColorLayers(),       // 级联 / 取色
    ui.CommandPaletteLayer(),                    // Ctrl+K
    ui.TourLayer(tour, steps),                   // 引导
    ui.NotifyLayer(),                            // 通知中心
    ui.ImagePreviewLayer(),                      // 图片灯箱
    ui.MenuLayer(bar, menus, x, y), ui.TooltipLayer(),
    ui.ToastLayer(),
)
```

## 1. P0 门面刚需

| 组件 | 入口 | 要点 |
| --- | --- | --- |
| 虚拟列表 | `VList(key,count,rowH,height,build)` / `VListOpt` / `VListVar` | 读 `ScrollOffset` 算可见区 + overscan；顶部/底部占位保持总高；10 万行示例 |
| 骨架屏 | `Skeleton(w,h)` / `SkeletonBlock` / `SkeletonText` / `SkeletonCard` | shimmer 流光 |
| 加载遮罩 | `LoadingOverlay(show,label)` | 区域级、拦截点击 |
| 空状态 | `EmptyState(icon,title,sub,action,fn)` | 插画位 + 操作按钮 |
| 结果页 | `ResultPage(kind,title,sub,actions...)` | success/warn/error/404/500 |
| 四态容器 | `AsyncSection(loading,err,empty,retry,body)` | loading/error/empty/body |
| 可拖分栏 | `Split(key,a,b,opts)` / `SetSplitPos` / `SplitPos` | 横/纵、最小尺寸、双击复位、`PrefsKey` 持久化 |
| 命令面板 | `RegisterCommand` / `CommandPaletteLayer` / `ToggleCommandPalette` | Ctrl+K、子序列模糊匹配、分组/快捷键提示、↑↓/Enter/Esc |
| 富文本显示 | `RichText` | 见上 |

## 2. P1 输入类

| 组件 | 状态对象 | 说明 |
| --- | --- | --- |
| AutoComplete | `NewAutoComplete(key)` | 过滤、↑↓/Enter/Esc、浮层 `AutoCompleteLayers` |
| TagInput | — | 回车/逗号添加、退格删末项、去重 |
| Rate | — | 半星、悬停预览、再次点击清除 |
| RangeSlider | — | 双把手、最近把手优先、键盘微调 |
| KeyCapture | — | 录制 `Ctrl+Shift+K`、Esc 取消、Backspace 清空 |
| TimePicker | `NewTime(key)` | 时/分/秒滚轮列、现在/确定；`TimeLayers` |
| Cascader | `NewCascade(key)` | 多级、`Load` 懒加载、叶子提交路径；`CascaderLayers` |
| Upload | — | 进度、失败重试、移除、大小提示、拖入 |
| ColorPicker | `NewColor(key)` | SV 面板 + 色相条 + HEX + 预设 + 吸管（Windows）；`ColorLayers` |

## 3. P1 数据展示

`Carousel`（自动播放/循环/指示器/箭头）、`Timeline`、`Steps`、`Segmented`、
`Descriptions`、`Statistic`（滚动数字 + 趋势）、`Calendar`（`NewCalendar` + 事件点）、
`Watermark`、`ImagePreviewOpen/Close + ImagePreviewLayer`、`AvatarGroup`、
`Tour`（`NewTour` + `TourLayer`，已看状态持久化）。

## 4. P1 表格 / 拖拽

```go
ui.TableV(ui.TableVConfig{
    Key: "Users", Headers: []string{"姓名","角色","最近活跃"},
    Widths: []float32{200, 120, 0}, RowH: 40, Height: 460, Count: len(users),
    Row: func(i int) []*ui.Node { return row(users[i]) },
    Groups: []ui.TableGroup{{Label:"进行中", Start:0, Count:120}},
    Footer: []*ui.Node{...}, Resizable: true, OnSort: sortBy, OnRow: open,
})
ui.ReorderList(ui.ReorderListOptions{Key:"list", Count:n, RowH:40, Height:360,
    Row: rowAt, OnMove: func(from,to int){ items = ui.MoveSlice(items, from, to) }})
```

- 列宽拖拽写入 `table.<key>.w<i>` 偏好；`SetTableColWidth` 可代码设置。
- 分组折叠状态按 `Key` 保存，表头虚拟化（分组头与数据行统一窗口）。

## 5. P2 桌面能力

| 能力 | 入口 | 平台 |
| --- | --- | --- |
| 快捷键注册表 | `RegisterShortcut(scope, sc)` / `ShortcutConflict` / `ShortcutsHelp` / `ShortcutSerialize` | 全平台；作用域 global/panel/modal，冲突检测，无修饰键不拦截 |
| 通知中心 | `Notify` / `NotifyProgress` / `NotifyClose` / `NotifyLayer` | 全平台；堆叠、按钮、进度、悬停暂停过期 |
| Dock | `NewDock` / `DockLayout` | 全平台；三侧、折叠、拖拽尺寸、布局持久化 |
| 属性表 | `PropertyGrid(key, props, onChange)` | 全平台；文本/数字/开关/枚举/颜色/文件/只读/自定义 + 搜索 + 分组折叠 |
| 多显示器 | `Monitors` / `MonitorAt` / `VirtualScreen` | Windows 真实现；其他端空 |
| 屏幕捕获 | `CaptureScreen` / `CaptureFullScreen` / `ScreenPickColor` | Windows（BitBlt）；其他端返回不支持 |
| 剪贴板监听 | `WatchClipboard` / `Clipboard` / `ClipboardImage` | Windows 序号轮询；其他端文本轮询 |
| 单实例 | `SingleInstance` / `NotifyFirstInstance` / `WatchSecondInstance` | Windows 命名互斥 + 临时文件参数 + 广播唤醒 |
| 打印 | `Print(title, doc)` / `PrintPreview(key, doc, zoom)` + `Printable` | Windows GDI（PrintDlg）；其他端返回不支持 |
| 打印/预览画布 | `PrintCanvas.Text/Rect/Line` | 同一套坐标（96 DPI A4 794×1123），打印按 DPI 缩放 |
| 自更新 | `CheckUpdate` / `DownloadUpdate` / `VerifySHA256` / `ApplyUpdate` / `UpdatePanelView` | 全平台；校验失败拒绝替换 |
| 崩溃报告 | `SetCrashHandler` / `ReportCrash` / `CrashPanel` / `RecoverToReport` | 写 `%AppData%/gogui/crashes` |
| Undo/Redo | `NewUndo` / `Do` / `Undo` / `Redo` / `BeginTx` / `CommitTx` | 默认栈绑定 Ctrl+Z / Ctrl+Y |
| 富文本编辑 | `NewRichEditor` / `RichEditor` / `RichToolbar` / `RichEditorEx` | 行内样式 + 选区 + IME + 撤销 |
| 跨节点选择 | `SelectionArea` / `SelectedText` / `ClearSelection` | 拖选矩形命中文本节点，Ctrl+C 复制 |
| 多行省略 | `EllipsisText` | 2 行截断 + 展开 |

### 平台能力表新增项

`CapPrint`、`CapScreenCapture`、`CapClipboardWatch`、`CapMultiMonitor`、
`CapSingleInstance`、`CapModernNotify`；`CapLast` 为枚举上界（新增能力时同步）。

## 6. 已知限制

- OLE `IDropTarget` 尚未接入：仍走 `WM_DROPFILES`，因此**没有拖入悬停高亮**（松手才收到事件）。
- 屏幕捕获不含鼠标指针；DXGI Desktop Duplication 未实现（BitBlt 足够静态截图）。
- 打印对话框结构按 64 位 `PRINTDLGW` 布局；如遇系统差异，优先检查 `lStructSize`。
- 单实例 IPC 用临时文件 + 广播，首个实例需调用 `WatchSecondInstance` 才能收到参数。
- 跨节点选择按「整文本节点命中」拼接，不做字形级切分。
- 富文本编辑器只支持行内样式（无图片/表格）；UIA Provider 未注册。
- RTL 仅提供行镜像与对齐助手，完整 bidi 排版未实现。
