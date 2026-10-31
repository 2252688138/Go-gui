# 纯 Go、零 CGO：我手写了一个不用 WebView 的 Web 风格 GUI 框架

> 6.45 MB → 4.54 MB → 1.52 MB；一个矩形 6 个顶点、22 个 float；圆角、描边、阴影、毛玻璃、超采样、中文排版全部自己画。
> 这篇文章拆解 GoGUI 的渲染管线、布局引擎、字体图集，以及 9 个把我按在地上摩擦过的坑。

仓库地址：<https://github.com/2252688138/Go-gui>
规模：129 个 Go 源文件、约 3.8 万行（含示例与工具），MIT 协议。

---

## 0. 为什么又造一个 GUI 轮子

Go 写桌面程序，绕不开三条路：

| 方案 | 代表 | 优点 | 代价 |
| --- | --- | --- | --- |
| 绑定原生控件 | walk、Win32 封装 | 系统原生观感 | 只能 Windows，样式几乎不可定制 |
| 前端 + WebView | Wails、Tauri 思路 | 前端生态成熟 | 多一层运行时，包体大，IPC 有延迟 |
| 纯 Go 自绘 | Fyne、gio | 无浏览器内核，包体小 | 控件与 Web 观感需要自己补 |

我想要的其实很具体：

- **单文件、无运行库**：拷一个 exe 就能跑，不依赖 WebView2（浏览器组件可选）、不依赖 VC++ 运行库；
- **Web 观感**：圆角、渐变、投影、毛玻璃、flex 布局、深浅主题；
- **中文要清晰**：小字号横竖笔不能糊，输入法要能用；
- **样式可编程**：不要 CSS 字符串解析，要类型安全的链式 API；
- **性能可控**：静止时不重绘，滚动和动画不掉帧。

于是有了 GoGUI：`ui` 层完全平台无关，下面用 `purego` 动态加载 Win32 / WGL，用 OpenGL 3.3 做 2D 批处理渲染，`x/image/font/opentype` 做字形光栅化。**没有一行 cgo**。

---

## 1. 先看结果

真实运行截图（1180×760，深色主题；图片托管在仓库 `ui/assets`，若掘金未自动转存，可在本地同名路径找到原图）：

![仪表盘](https://raw.githubusercontent.com/2252688138/Go-gui/main/ui/assets/%E4%BB%AA%E8%A1%A8%E7%9B%98.png)

![图表组件](https://raw.githubusercontent.com/2252688138/Go-gui/main/ui/assets/%E5%9B%BE%E8%A1%A8%E7%BB%84%E4%BB%B6.png)

![Markdown 渲染](https://raw.githubusercontent.com/2252688138/Go-gui/main/ui/assets/markdown%E6%B8%B2%E6%9F%93.png)

![浅色主题](https://raw.githubusercontent.com/2252688138/Go-gui/main/ui/assets/%E6%B5%85%E8%89%B2%E4%B8%BB%E9%A2%98.png)

这些界面里没有 HTML、没有 CSS、没有 WebView 参与主体渲染，全部是 OpenGL 画的。

最小可运行程序：

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
go run ./cmd/gallery
# 无控制台窗口的发布构建
go build -ldflags "-H=windowsgui" -o app.exe ./cmd/gallery
```

---

## 2. 架构：五层，零 cgo 是怎么做到的

```
cmd/gallery          示例应用（8 个可复用界面）
ui/                  声明式组件树 / 样式 DSL / 布局 / 事件 / 主题
internal/gpu         OpenGL 3.3 批处理渲染器 + GLSL
internal/text        opentype 字形光栅化 / 图集 / 度量
internal/win32       Win32 + WGL + COM 绑定（purego 动态加载）
internal/geom        几何基础类型
```

零 cgo 的关键是 `github.com/ebitengine/purego`：

```go
// 伪代码：运行时从 DLL 取函数地址，再包装成 Go 函数
lib := purego.Dlopen("opengl32.dll", purego.RTLD_NOW)
var wglCreateContext func(hdc uintptr) uintptr
purego.RegisterLibFunc(&wglCreateContext, lib, "wglCreateContext")
```

代价也很真实：

- 没有头文件和类型检查，函数签名、结构体布局、调用约定全靠手写；
- COM 接口要自己排 vtable，槽位错一个就是访问违例（后面有个血案）；
- 一些 ABI 细节（比如 float 参数走 XMM 寄存器）Go 侧没有现成支持，得写汇编 trampoline。

收益是：**整个程序只有一个 exe**，不依赖 MinGW、不依赖 MSVC 运行库，交叉编译也不会因为 cgo 卡住。

### 2.1 渲染目标不是「画布」，是一组 Draw 指令

`ui` 层每帧把组件树编译成绘制指令：形状、文字、裁剪入栈/出栈。`gpu.Renderer` 顺序消费，遇到类型、字体、纹理或裁剪区域变化才 flush 一次，合并成尽可能少的 `glDrawArrays`。

```go
// internal/gpu/renderer.go（简化）
func (r *Renderer) Draw(instrs []Instr) {
	for _, in := range instrs {
		switch in.Kind {
		case InstrShape:
			r.pushShape(in.Shape)
		case InstrText:
			r.pushText(in.Text)
		case InstrClipPush:
			r.clipSaved = append(r.clipSaved, in.Clip)
			r.setClip(in.Clip)
		case InstrClipPop:
			// 恢复到上一层裁剪（嵌套取交集）
		}
	}
}
```

### 2.2 一个矩形 = 6 顶点 × 22 float

形状顶点格式：

```
pos(2) | local(2) | color(4) | color2(4) | halfSize(2) | radius4(4) | params(4)
```

`local` 是相对矩形中心的坐标，`halfSize` 是半宽高，`radius4` 支持四角不同半径，`params` 打包了 `feather / mode / strokeWidth / glow`。一个矩形 6 个顶点，528 字节。

为什么不用「一个样式一个 draw call」？因为一屏几百个圆角卡片、标签、描边，draw call 会直接压垮驱动。批处理之后，一屏通常只有个位数到几十次提交。

---

## 3. 声明式 UI：每帧重建，不做 diff

GoGUI 的 `view()` 是一个**纯函数**：每次重绘都重新构造整棵组件树，框架重新 measure / arrange / paint。没有虚拟 DOM，没有 diff，没有 Key 复用（Key 只用于保存滚动偏移和交互状态）。

```go
var current = "dashboard"

ui.Button("组件库", func() {
	current = "components"
	ui.RequestRedraw() // 下一帧重建 → 重新布局 → 重绘
}).St(ui.St().Text(13, 500, "#8b93ad").PadXY(14, 10).R(10))
```

这套模型能成立，靠三个设计：

**1）状态外置。** 组件不持有状态，状态在调用方的包级变量或结构体里。组件只是「当前状态的投影」。

**2）回调延迟一帧。** 点击回调不在事件回调里立刻执行，而是缓存到下一帧、基于最新组件树触发：

```go
// ui/app.go 注释原文：
// 点击回调在下一帧基于最新组件树触发（避免旧指针失效），无需给节点设置 Key
```

否则闭包捕获的 `*Node` 在树重建后已经失效，轻则改错节点，重则悬空指针。

**3）重绘请求是原子的。** `ui.RequestRedraw()` 可以从任意 goroutine 调用：

```go
// 任意 goroutine 都可以调用；只置位，不跨线程碰 UI
func RequestRedraw() {
	for _, a := range snapshotApps() {
		a.redrawRequest.Store(true)
	}
}
```

主循环的判定条件大致是：

```go
if a.dirty || a.redrawRequest.Swap(false) || needAnim || a.hoverProbe {
	// 重建 + 布局 + 绘制
}
```

静止时没有动画、没有输入，就不产生绘制帧。空闲轮询还会从 120Hz 降到 60Hz，避免消息泵空转吃 CPU。

### 3.1 事件路由：命中测试 + 冒泡

鼠标事件先做命中测试，从最深的节点往上冒泡到最近的可交互祖先。滚动容器的偏移按节点 `Key` 存在 App 里，所以：

```go
ui.Scroll(content).K("scroll-main")
```

**滚动容器必须给唯一 Key**，否则两个滚动区域会共享偏移。

---

## 4. 布局引擎：Flutter 式 Constraints + 两阶段

布局分两步：

1. **measure**：自底向上算尺寸（文字用 opentype 度量，图片只读文件头）；
2. **arrange**：自顶向下定位，分配 `Grow` 剩余空间。

约束模型是 Flutter 风格：

```go
type Constraints struct {
	MinW, MaxW, MinH, MaxH float32
}
```

`Row` / `Column` 默认 `AlignStretch`，与 CSS `align-items: stretch` 一致。

### 4.1 flex 为什么要测两遍

第一遍按父可用尺寸量所有子项，统计固定项占用；第二遍把剩余主轴空间按 `Grow` 比例分给可伸缩子项，**只重测没有显式主轴尺寸的子项**。

```go
// ui/layout.go（简化）
if growSum > 0 {
	free := availW - fixedMain
	if !horiz {
		free = availH - fixedMain
	}
	if free < 0 {
		free = 0
	}
	for _, c := range n.Children {
		if c.Style.Grow <= 0 || c.Style.PosAbs {
			continue
		}
		// 已经显式定尺寸的项不需要重测
		if horiz && (c.Style.Width > 0 || c.Style.WidthPct > 0 || c.Style.Basis > 0) {
			continue
		}
		share := free * c.Style.Grow / growSum
		measureNode(lc, c, Constraints{0, share, 0, availH}, n)
	}
}
```

这里有一个很容易踩的坑：**测量子项时，主轴方向不要给 `MaxW` 上限**。

```go
// 测量约束：主轴方向不设上限（与 CSS flex 一致），交叉轴按父可用尺寸。
// 若主轴也给上限，子项内部的百分比尺寸会按父级满宽计算，
// 导致「自动宽」容器被撑大。
childC := func() Constraints {
	if horiz {
		return Constraints{0, 0, 0, availH} // MaxW = 0，不限
	}
	return Constraints{0, availW, 0, 0}
}
```

同理，百分比尺寸在 arrange 阶段还要再算一次——因为父容器可能在 `AlignStretch` 下被拉伸，measure 阶段的旧尺寸会让 `Wp(100)` 溢出。

### 4.2 内容型容器不受父高截断

`Scroll` 里的子项必须按内容自然伸展，否则内容会被截成视口高度，根本滚不动。`Wrap` / `Grid` / `Masonry` 也一样：

```go
content := n.Kind == KWrap || n.Kind == KGrid || n.Kind == KMasonry
inScroll := parent != nil && parent.Kind == KScroll
if c.MaxH > 0 && h > c.MaxH && !st.Scrollable && !content && !inScroll {
	h = c.MaxH
}
```

### 4.3 不止 flex

| 构造器 | 能力 |
| --- | --- |
| `Grid` | 等宽列网格，固定列数或按列宽自适应 |
| `Wrap` | `flex-wrap: wrap`，行内支持 space-between |
| `Masonry` | 瀑布流，子项落入当前最矮的列 |
| `EqualRow` / `EqualCol` | 主轴等分栏 |
| `Pick / BP` | 按视口宽度切响应式分支 |

```go
ui.Grid(tiles...).St(ui.St().ColWidth(180).GapOf(12))
ui.Wrap(tags...).St(ui.St().GapOf(8)).JustifyTo(ui.AlignSpaceBetween)
ui.Masonry(cards...).St(ui.St().ColsN(3).GapOf(12))
ui.Pick(narrow, wide, wide, wide)
```

---

## 5. 渲染：一个 SDF shader 解决圆角、描边、阴影

圆角矩形用解析式有向距离场（SDF）：

```glsl
float sdRoundBox(vec2 p, vec2 b, vec4 r) {
    r.xy = (p.x > 0.0) ? r.xy : r.zw;
    r.x  = (p.y > 0.0) ? r.x  : r.y;
    vec2 q = abs(p) - b + r.x;
    return min(max(q.x, q.y), 0.0) + length(max(q, 0.0)) - r.x;
}
```

片元着色器按 `mode` 分三路：

```glsl
float d = sdRoundBox(vLocal, vHalf, vRadius);
if (glow > 0.0) {
    // 阴影：SDF 向外扩散 glow 像素，距离衰减近似高斯
    float k = clamp(-d / max(glow, 0.001), 0.0, 1.0);
    d = d + glow * (1.0 - k);
}

if (mode < 0.5) {
    alpha = 1.0 - smoothstep(-feather, feather, d);              // 填充
} else if (mode < 1.5) {
    alpha = 1.0 - smoothstep(-feather, feather,
            abs(d + strokeW * 0.5) - strokeW * 0.5);             // 描边
} else {
    alpha = (1.0 - smoothstep(-glow, glow, d)) * 0.9;            // 阴影
}
```

一次 draw 同时完成：填充、描边、阴影。阴影不是单独一张纹理，也不需要 `blur` 通道——CPU 侧把矩形外扩 `spread + blur`，片元里做距离衰减。

`feather` 默认 0.7 逻辑像素，配合 `smoothstep` 得到抗锯齿边缘：

```go
r.appendQuad(rs, sh.Radius, sh.Fill, Paint{A: -1}, 0.7, 0, 0)
```

### 5.1 渐变与批处理

渐变用两个顶点色插值，`vColor2.a >= 0` 时启用：

```glsl
vec4 col = vColor;
if (vColor2.a >= 0.0) {
    col = mix(vColor, vColor2, vT.x);
}
fragColor = vec4(col.rgb, col.a * alpha);
```

批次在四种情况下切分：

- 指令类型变化（shape → text → image）；
- 字体变化（每张字形图集是独立纹理）；
- 图片纹理变化；
- 裁剪区域变化（`glScissor` 是全局状态）。

裁剪嵌套取交集，同一矩形重复设置不会打断批次：

```go
func (r *Renderer) setClip(c geom.Rect) {
	if r.clipOn && r.clipCur == c {
		return // 嵌套滚动容器常见，避免无意义 flush
	}
	r.flush()
	// glScissor 坐标原点在左下，需要翻转 y
}
```

### 5.2 顶点缓冲：孤儿化 + SubData

早期实现每帧 `glBufferData(data)` 重新分配整块缓冲，滚动时驱动频繁分配、还要和上一帧的绘制同步。现在只在容量不足时扩容，其余帧孤儿化后 `glBufferSubData` 覆盖：

```go
func (r *Renderer) uploadVerts(vbo *uint32, capBytes *int, data []float32) {
	n := len(data) * 4
	glBindBuffer(GL_ARRAY_BUFFER, *vbo)
	if *capBytes < n {
		*capBytes = n + n/2 + 4096
	}
	// 孤儿化：解除与上一帧的依赖
	glBufferAlloc(GL_ARRAY_BUFFER, *capBytes, GL_STREAM_DRAW)
	glBufferSubData(GL_ARRAY_BUFFER, 0, data)
}
```

另外，顶点切片的初始容量从「按 60000 顶点硬上限预留」改成「按典型一屏预留」：

```go
maxVertsShapeInitial = 8192
maxVertsTextInitial  = 20480
```

旧实现一开窗就分配 `60000×22×4B ≈ 5.3MB` + `60000×9×4B ≈ 2.2MB` 的 Go 切片，而实际一屏通常只有几百到几千个顶点。

---

## 6. 文字：中文清晰度是自绘 GUI 的生死线

文字管线：

1. `x/image/font/opentype` 读 TTF/TTC，取字形轮廓；
2. 矢量轮廓光栅化成 8bit 覆盖率；
3. 写进 RGBA8 字形图集（普通字形写 `(v,v,v,255)`，Emoji 写真实颜色）；
4. 顶点带 `mode`，`mode=1` 直接输出纹理颜色；
5. 每个字形一个四边形，UV 指向图集。

### 6.1 图集策略：256² 起步，按需扩容，LRU 回收

这是内存优化里收益最大的一刀。

旧版每个字体固定 `2048²` 图集 = **16MB**。一个页面用十几种字号/字重，直接吃掉上百 MB。

现在：

| 阶段 | 图集 | 单字体内存 |
| --- | --- | --- |
| 起步 | 256² | 256 KB |
| 装不下 | 512² | 1 MB |
| 常规上限 | 1024² | 4 MB |
| 光栅倍率 ≥ 1.75（超采样） | 2048² | 16 MB |

字体实例上限 24 个，LRU 回收时同时释放对应纹理：

```go
const maxFonts = 24

func (fc *fontCache) evictOldest() {
	// 找最久未使用的字体
	if f, ok := fc.fonts[oldKey]; ok {
		if fc.onEvict != nil {
			fc.onEvict(f) // 上层释放图集纹理
		}
		f.Close()
		delete(fc.fonts, oldKey)
	}
	// 折行缓存以 *Font 为键，字体关闭必须整体失效
	clearWrapMemo()
}
```

实测（`GOGUI_MEMSTATS=1`，gallery 运行时的示例输出）：

```
[gogui] frames=120 heap=19.4MB inuse=20.6MB sys=27.7MB gc=2 fonts=13 atlas=3.2MB images=0
```

13 个字体实例，图集总共 3.2MB。换成旧的固定 2048²，光图集就是 13 × 16MB = 208MB。

### 6.2 只上传脏矩形

新字形出现时，不再重传整张图集，而是记录脏矩形，`glTexSubImage2D` 局部更新：

```go
if x0, y0, x1, y1, dirty := f.TakeDirty(); dirty {
	r.uploadAtlasRegion(f, t, x0, y0, x1, y1)
}
```

整行范围直接上传（内存连续），否则逐行拷贝到复用缓冲再提交。滚动时新文字不断出现，这一刀把「连续全量上传」的卡顿消掉了。

### 6.3 字体文件走内存映射

`msyh.ttc` 这类字体文件用 `MapViewOfFile` 映射，数据由系统页缓存承载，不占 Go 堆。

### 6.4 字形 gamma：为什么自绘文字总是「发虚」

这是最隐蔽的一个坑。

浏览器（Skia / DirectWrite）在 sRGB 空间混合前会对覆盖率做 gamma 校正，等效于把中间覆盖率拉高：**亮度 0.5 的像素在浏览器里看起来接近 0.7**。

而自绘渲染通常是 sRGB 直混，同样的覆盖率就显得偏淡偏虚，小字号中文的横竖笔尤其明显。

解决办法是给覆盖率加一道 gamma：

```glsl
float lum = dot(vColor.rgb, vec3(0.2126, 0.7152, 0.0722));
if (uTextGamma != 1.0 && lum > 0.5) {
	a = pow(a, uTextGamma);
}
```

只对「亮字」（暗底浅字）补偿；暗字浅底时线性混合反而更淡，不需要再拉。实测 `1/1.8 ≈ 0.55` 与浏览器观感最接近，可用 `GOGUI_TEXT_GAMMA` 覆盖。

### 6.5 Emoji 回退链

主字体缺字形时依次回退：

```
Microsoft YaHei UI → Segoe UI Symbol → GDI 彩色 Emoji（Segoe UI Emoji）
```

GDI 渲染到 32bpp DIB 后，第二遍白色蒙版补齐深色像素 alpha，再写进图集，顶点 `mode=1` 直出颜色。

### 6.6 一个输入框，半部输入法血泪史

自绘输入框不是「画个矩形 + 收键盘事件」那么简单。要处理：

- **光标位置**：按 rune 存，不能按 byte（中文一个字 3 字节）；
- **焦点**：`St().FocusOn()` + 全局 `focusKey`，点空白处要清焦点；
- **密码掩码**：显示 `•••`，但光标位置仍按真实 rune 数算；
- **编辑键**：← → Home End Backspace Delete；
- **回车提交**：地址栏、搜索框需要 `InputSubmit`；
- **输入法**：组合串、候选窗位置、上屏去重。

输入法这三段消息必须全链路接：

```go
case EvCompositionStart:
	a.imeText, a.imeCaret = "", 0
case EvComposition:
	a.imeText, a.imeCaret = ev.Text, ev.Caret   // GCS_COMPSTR
case EvCompositionEnd:
	a.imeText, a.imeCaret = "", 0
```

- `GCS_COMPSTR` 组合串通过 `ui.ComposingText()` / `ComposingCaret()` 暴露，输入框用高亮块显示预编辑文本，此时**光标不闪烁**；
- `GCS_RESULTSTR` 上屏时逐字符派发 `EvChar`，并且不再重复产生 `WM_CHAR`，否则一个汉字会输入两次；
- 焦点输入框的位置经 `ImmSetCompositionWindow` / `ImmSetCandidateWindow` 同步给系统，候选窗才会跟着走，而不是永远停在窗口左上角。

```go
// ui/widgets.go（简化）
if comp := ComposingText(); comp != "" && !password {
	cp := ComposingCaret()
	cr := []rune(comp)
	cells = append(cells, Text(string(cr[:cp])).St(St().Text(13, 400, ColText).BgC(ColBgSoft).R(3)))
	cells = append(cells, Text(string(cr[cp:])).St(St().Text(13, 400, ColText).BgC(ColHover).R(3)))
	showCaret = false // 组合期间不闪光标
}
```

---

## 7. 超采样：把「字体发虚」和「边缘发毛」一起解决

圆角 SDF 的抗锯齿只有 1 个像素的过渡带；小字号笔画更惨。整帧 SSAA 的思路很直接：

**整帧先按 N 倍分辨率画进离屏 FBO，再平均降采样回窗口像素。**

```go
ui.SetSuperSample(ui.SSAuto) // DPI < 150% 时 2×，否则 1×（推荐）
ui.SetSuperSample(ui.SS200)  // 强制 2×
ui.SetSuperSample(ui.SSOff)  // 1:1，GPU 开销最低
// 另有 SS125 / SS150
```

降采样是 4×4 子带采样的 box 滤波：

```glsl
vec2 base = uTexel * (uSpan * 0.125);
vec4 c = vec4(0.0);
for (int y = 0; y < 4; y++) {
    for (int x = 0; x < 4; x++) {
        vec2 o = vec2(float(x) * 2.0 - 3.0, float(y) * 2.0 - 3.0) * base;
        c += texture(uTex, vUV + o);
    }
}
c *= 0.0625;
```

逻辑布局、命中测试、鼠标坐标完全不变，只改变渲染分辨率。但这里有四个反直觉的细节。

### 7.1 边缘过渡带要按设备像素恒定

`feather` 原本按逻辑像素给 0.7。超采样 2× 后画布是 2 倍分辨率，0.7 逻辑像素在设备空间变成 1.4 像素，降采样后边缘反而更软——**超采样白做了**。

所以 CPU 侧要把过渡带除以倍率：

```go
// 填充/描边的 SDF 边缘过渡带宽度按「设备像素」恒定
if r.ssaa > 1.01 && mode < 1.5 && feather > 0 {
	feather /= r.ssaa
}
```

片元里还得把下限从 0.5 降到 0.25，否则 2× 下收紧后的 0.35 又被抬回 0.5，锐度被吃掉一半。

### 7.2 字形 gamma 不能按倍率收敛

一开始想当然：超采样后平均了 N² 个样本，gamma 补偿应该弱化成 `1-(1-g)/ss`。结果 2× 下文字明显发灰发虚，观感还不如 1:1。

原因：**降采样是子样本线性平均，gamma 补偿作用在平均之后并不失真**。所以所有倍率统一使用同一个补偿值，SSAA 只提升几何/边缘精度，不改变笔画实度。

### 7.3 box 平均会让 1px 细节变「软」

数学上 box 降采样是正确的，但 1px 的横竖笔、1px 描边被平均后对比度下降，人眼会主观判断「还不如 1:1 清晰」。

所以降采样后补一道轻度 unsharp：

```glsl
if (uSharpen > 0.0) {
    vec2 o = uTexel * uSpan;
    vec4 blur = (texture(uTex, vUV + vec2(o.x, 0.0)) +
                 texture(uTex, vUV - vec2(o.x, 0.0)) +
                 texture(uTex, vUV + vec2(0.0, o.y)) +
                 texture(uTex, vUV - vec2(0.0, o.y))) * 0.25;
    c = clamp(c + (c - blur) * uSharpen, 0.0, 1.0);
}
```

默认 0.4（`GOGUI_SS_SHARPEN` 可调），只补回被平滑掉的一小部分高频，既保留抗锯齿，又不出现白色光晕。

### 7.4 像素预算与半透明背景

- 像素预算 `8192×4320`：窗口很大时自动降档 `2× → 1.5× → 1.25× → 1×`，避免 4K/5K 下填充率爆炸；
- 降采样是整屏覆盖写，必须**关掉混合**，否则半透明背景会和目标里的旧内容再混一次；
- 字形图集在光栅倍率 ≥ 1.75 时自动放宽到 2048²，否则 2× 下很容易写满触发「清空重用」，同帧 UV 失效导致缺字；
- 运行时热切换会重建渲染目标与字形图集，适合放在设置页做清晰度选项。

---

## 8. 多媒体：不引 cgo，也能播 mp4

播放音视频不一定非要链接 FFmpeg 动态库。GoGUI 的做法是**把 ffmpeg 当子进程用**：

```go
p, err := ui.NewMediaPlayer("movie.mkv") // 或 m3u8 直链
if err != nil { /* 未安装 ffmpeg / 无视频流 / 链接不可达 */ }
defer p.Close() // 释放 ffmpeg 进程与音频设备
p.Play()

ui.VideoPlayer("vp", p).St(ui.St().H(420))
```

内部是两条独立管线：

| 管线 | 实现 |
| --- | --- |
| 视频 | `ffmpeg -f rawvideo -pix_fmt rgba` → 管道 → 帧环（上限 48MB / 2 秒）→ 按播放位置取帧 → `glTexSubImage2D` 局部更新纹理 |
| 音频 | 第二个 ffmpeg 进程输出 s16le / 48k / 立体声 → WinMM `waveOut` |

### 8.1 用音频当主时钟

视频帧率和音频采样率不可能精确对齐。有音轨时，播放器以 `waveOut` 已播样本数作为时钟（`MediaClock`），画面跟随音频位置取帧；倍速用 `atempo` 滤镜重建整条音频链路。

这比「用 `time.Now()` 估算播放位置」稳得多——后者跑几分钟就会和声音差出几百毫秒。

### 8.2 拖动进度条不要每个事件都重启 ffmpeg

最开始的实现是 `mousemove` 就 `-ss` 重启解码进程，结果拖动时 CPU 直接打满，画面还卡住。

现在的策略是 `MediaScrub`：

- 拖动期间只记录目标位置，**直接返回现有帧**；
- 等解码等待预算降到 80ms 以下，才允许抓一帧预览；
- 松手后一次性 `-ss` 定位，只重启一次。

拖动因此不再卡顿，代价是拖动过程中预览帧不是完全跟手——这个取舍在播放器里是值得的。

### 8.3 没有 ffmpeg 就回退系统解码

`ui.OpenMedia` 先探测 PATH 里有没有 `ffmpeg` / `ffprobe`，没有则走 Windows Media Foundation：

```go
os.Setenv("GOGUI_DECODER", "system")
src, _ := ui.NewMediaPlayer("movie.mp4")
fmt.Println(src.Info().Backend) // system
```

| | ffmpeg 后端 | 系统解码后端 |
| --- | --- | --- |
| 容器 | mp4/mkv/mov/avi/webm/flv/ts/m3u8 | 系统支持的容器 |
| 编码 | 基本全部 | 系统已安装的 MFT |
| 倍速 | 支持（atempo） | 仅 1x |
| 拖动预览 | 单帧抓取 | 保留上一帧 |

MF 的坑前面提过：`IMFSourceReader` 的所有调用必须串行在同一个锁定线程上（`win32.MFRun`），否则创建第二个 reader 时直接访问违例。

---

## 9. 组件化：把浏览器的一部分重新实现

自绘框架最容易「 demo 很漂亮，一用就少十个控件」。所以 `ui/widgets.go` 把常用控件都做了一遍，而且状态全部由调用方持有——组件本身无隐藏状态，拷走就能用。

### 9.1 Markdown 渲染器

```go
ui.Stack(
	ui.Scroll(
		ui.Markdown("md", src, ui.MarkdownOptions{Width: 760}),
	).K("scroll-md"),
	ui.MarkdownLightbox(), // 点击图片后全屏预览
)
```

支持：标题 / 段落 / 粗体 / 斜体 / 删除线 / 行内代码 / 链接 / 图片 / 有序无序列表 / 任务列表 / 引用与 GitHub 提示块 / 代码块（行号 + 着色 + 复制 + 超 22 行折叠）/ 表格 / 分隔线 / `[toc]` / `[tabs]` / `[details]` / `[card]` / `[badge]` / `[progress]` / `[gallery]`，以及数学公式与 Mermaid。

解析器、语法着色、关键字表都是自己写的，没有引入 goldmark / chroma。代码高亮覆盖 go / js / ts / py / json / bash / sql / html / css / yaml 十种语言，够用且体积极小。

### 9.2 图表：数据不变时只发 1 个 draw call

```go
chart := ui.NewChart(ui.ChartLine, labels,
	ui.Series{Name: "本周", Values: week, Color: "#6366f1"},
	ui.Series{Name: "上周", Values: last, Color: "#22d3ee"},
)
chart.Smooth = true
chart.OnPick = func(series, index int) { /* 点击数据点 */ }

ui.ChartView("trend", chart).St(ui.St().H(300))
```

图表不是每帧重新光栅化，而是按「节点 Key + 尺寸 + 缩放 + 数据指纹」缓存一张 RGBA 纹理：

- 数据不变：每帧 1 个 draw call，零 CPU 光栅化；
- 缓存总预算 24MB，超出按最近使用淘汰，纹理随窗口销毁；
- 悬浮高亮与点击命中复用同一份布局结果（`chart.cache.hitS / hitI`），回调里不用重算。

支持折线 / 面积 / 柱状 / 饼 / 环形 / 雷达 / 散点，堆叠、平滑（Catmull-Rom）、自定义轴范围、`OnPick` 点击回调。

### 9.3 其余开箱即用

| 分类 | 组件 |
| --- | --- |
| 表单 | `Input` `PasswordInput` `Checkbox` `Radio` `Switch` `Slider` `Progress` `Ring` |
| 展示 | `Card` `Title` `Divider` `Tag` `Dot` `Avatar` `Badge` `KeyValue` `Empty` `Skeleton` `Spinner` |
| 导航 | `Tabs` `TabsBar` `Breadcrumb` `Accordion` |
| 数据 | `Table`（固定/弹性列宽、斑马纹、整行点击）`ListItem` |
| 浮层 | `Modal` `MenuPanel` `TooltipOf` `Toast` `Banner` |
| 媒体 | `VideoPlayer` `MusicPlayer`（含频谱 / 封面 / 播放列表） |
| 浏览器 | `WebView` + 自绘地址栏 / 状态行 |

`ui/COMPONENTS.md` 里有每个组件的文件归属和最小示例，按文件模块化引入即可。

### 9.4 列表与内容组件（新增）

列表页是 Web 观感最容易翻车的地方：既要层级缩进、悬停高亮，又要图文混排与分页。本轮把这类积木一次补齐：

| 组件 | 文件 | 关键点 |
| --- | --- | --- |
| `FileTree` 文件树 | `ui/tree.go` + `ui/tree_row.go` | 展开态按 `Key` 记忆；折叠的子树**不构建节点**，长目录也不会掉帧；箭头 / 文件夹 / 文件图标全是矢量色块，不依赖字体字形 |
| `MediaCard` 图文卡片 | `ui/cards.go` | 封面（可选真图或渐变占位）+ 底部压暗 + 角标 + 评分 + 标题 / 标签 / 操作按钮；`MediaCards` 用 `Wrap` 自动换行 |
| `ListRows` 列表行 | `ui/cards.go` | 图标 / 头像 + 标题与副标题 + 右侧标签与说明；支持多选、分隔线与选中高亮 |
| `PostCard` 社区帖子 | `ui/cards.go` | 头像 / 昵称 / 身份 / 时间 → 正文 → 话题 → 配图 → 点赞 / 评论 / 转发 |
| `Pager` 分页器 | `ui/pager.go` | 当前页渐变高亮、省略号折叠、`Locked` 之后的页显示矢量锁标且直接禁用点击 |

```go
ui.FileTree(nodes, ui.TreeOptions{Key: "files", Expanded: []string{"ui"}, Selected: "ui/layout.go",
    OnPick: func(n ui.TreeNode) { sel = n.Key }})
ui.MediaCards(items, ui.MediaCardOptions{Width: 186, OnAction: setWallpaper}, 12)
ui.ListRows(rows, ui.ListRowsOptions{Key: "rows", Dividers: true, CheckBox: true, Selected: sel})
ui.PostCard(post, ui.PostCardOptions{OnLike: like})
ui.Pager(ui.PagerOptions{Key: "list", Page: 2, Total: 9, Locked: 6, OnPick: goPage})
```

示例页也从「一页堆到底」改成了**页内下划线页签**：「列表与表格」页顶部用 `TabsBar` 切「列表 / 表格」，
两边各自一个 `Scroll`，切换时滚动位置互不干扰（与系统页的「系统能力 / WebView」同构）。

顺带修了一批交互细节，都是实际用出来才发现的：

- **开关族**：`SwitchEx` + 样式预设（默认 / 星光 / 带字 / 渐变 / 大中小 / 禁用）与自定义图标、文案、宽度构建器；图标居中用字形包围盒算 `dy` 后走绘制期 `Translate`（布局期负偏移会被吞成 0）。
- **取色器**：色相条旧实现用 `Stack` 叠了 12 段（只剩一片红），改为 `Row` 顺排 + 指示线；HEX 输入框宽度被 `Input` 自带的 `Wp(100)` 吞成 0；SV 面板因「透明端色被 `merge` 当成未设置」而变灰。吸管改为屏幕坐标拾色模式。
- **渐变角度**：`BgAngle` 在 `appendQuad` 里按 CSS 角度算四角颜色写入顶点色，由 GPU 插值；同时把顶点格式定死（多一个 `location` 属性会让整窗出现斜向条纹）。
- **拖拽重排**：App 只在 `pressNode.OnDrag != nil` 时才建立 dragKey，只注册 `OnDragEnd` 的节点永远收不到松手事件——`ReorderList` 必须同时挂 `OnDrag`。
- **滚轮**：偏移表是懒建的，滚轮只写目标位，`stepScrollAnim` 遇到不存在的条目会直接丢弃目标，表现就是「有滚动条但滚轮不滚，拖一下才恢复」；现在首次滚动即建 0 起点条目。
- **百分比尺寸**：`Hp(100)` 在主轴无上限测量时解析成 0，会回退成内容高——WebView2 白屏就是这么来的；要撑满一律 `Grow(1)`。
- **多开**：gallery 去掉了单实例互斥，每次启动都是独立窗口与独立页面状态（对比同一组件在不同主题下的效果很方便）。

---

## 10. 性能清单：省下来的都是帧时间

| 优化 | 做法 | 效果 |
| --- | --- | --- |
| 按需重绘 | `dirty / redrawRequest / animBusy / hoverProbe` 四选一 | 静止时不产生绘制帧 |
| 最小化不渲染 | 窗口最小化直接跳过 frame | 后台不烧 GPU |
| 视口外跳过 | 滚动容器只遍历可见子树 | 长列表滚动更稳 |
| 悬停去重 | 鼠标移动只做命中测试，悬停目标没变不重绘 | 鼠标划过不抖 |
| VBO 复用 | 容量足够时孤儿化 + SubData | 减少驱动分配与同步 |
| 图集局部上传 | 脏矩形 `glTexSubImage2D` | 新字形不再全量重传 |
| 图片异步解码 | 后台 goroutine 解码（并发上限 4），UI 线程只上传纹理 | 大量图片不卡顿 |
| 字体 mmap | `MapViewOfFile` 映射字体文件 | 不占 Go 堆 |
| 媒体帧环上限 | 48MB / 2 秒 | 播放长视频内存可控 |

`GOGUI_MEMSTATS=1` 可以每 120 帧打印一次内存构成：

```
[gogui] frames=120 heap=19.4MB inuse=20.6MB sys=27.7MB gc=2 fonts=13 atlas=3.2MB images=0
```

---

## 11. 踩坑合集：那些文档里不会写的

### 11.1 DirectComposition 的 vtable 槽位是反的

为了让 WebView2 网页能被自绘菜单盖住，需要走 DComp 合成。dcomp 运行时的虚表把「动画版」排在「float 版」前面，**和 SDK 头文件相反**：

```
SetOffsetX(float) = 4
SetOffsetY(float) = 6
SetClip           = 14
矩形裁剪的左右上下/四角半径 float 版全在偶数槽（4/6/8/10/12…26）
```

用 3/5/7/9/11/13 会把 float 当动画指针解引用，崩在 `addr=0x50`。这个坑查了很久，因为崩溃现场完全看不出和 DComp 有关。

### 11.2 Go 链接器只接受一个 `.rsrc` 节

图标和版本信息必须由**同一个** syso 生成：

```
too many .rsrc sections
```

`tools/winres` 直接输出 COFF `.syso`，把 `RT_ICON` + `RT_GROUP_ICON` + `RT_VERSION` 写进同一个 `.rsrc` 节，不需要 windres、不需要 cgo。

### 11.3 NSIS 脚本含中文必须 UTF-8 带 BOM

否则 `makensis` 直接报：

```
Bad text encoding
```

### 11.4 `InstallDirRegKey` 会污染目录页

它在 `.onInit` 之前就把旧目录塞进 `$INSTDIR`，导致「为所有用户安装」误用当前用户的旧路径。正确做法是只在范围切换时调 `SetScope`。

### 11.5 安装 Section 内不要再给 `$INSTDIR` 赋值

会覆盖用户在目录页手选的路径。自定义路径必须完整保留，范围切换只在两个默认目录之间切换。

### 11.6 WebView2 的 HWND 永远在 OpenGL 内容之上

HWND 托管模式下，网页像素不进入 GL 批次，所以：

- 无法被菜单 / Modal 遮挡；
- 不能做 3D 变换；
- 圆角只能 `SetWindowRgn` 硬裁剪。

现在的策略是：浮层展开时临时隐藏网页；`GOGUI_COMPOSE=1` 时尝试 DComp 合成（实验性），失败自动回退。

### 11.7 `WM_DROPFILES` 没有 dragover

系统只在松手时投递消息，没有 dragenter/dragover，所以**做不了悬停高亮**。想要悬停反馈必须改用 OLE `IDropTarget`。

### 11.8 Media Foundation 必须串行在同一线程

`IMFSourceReader` 对跨线程调度极其敏感。所有调用（创建 / `ReadSample` / `Seek` / `Release`）都要通过 `win32.MFRun` 串行在一个锁定线程上执行，否则创建第二个 reader 时直接访问违例。

### 11.9 无标题栏窗口的 1px 白边

`NoTitleBar` 窗口要额外把 `DWMWA_BORDER_COLOR` 置为 `DWMWA_COLOR_NONE`，否则 DWM 会画一圈 1px 白色系统边框，在深色界面上非常刺眼。

---

## 12. 体积与发布：6.45MB → 1.52MB

`build.ps1` 把发布参数固定下来：

```powershell
./build.ps1                 # gallery -> dist\gallery.exe
./build.ps1 -Target all     # gallery + demo + tools
./build.ps1 -Upx            # 再叠加 UPX -9
./build.ps1 -Dev            # 开发构建：保留符号 + 控制台日志
./build.ps1 -Installer      # 构建 + NSIS 安装器
./build.ps1 -Bump patch -Installer
```

实测（gallery，go1.26 windows/amd64）：

| 构建方式 | 体积 | 说明 |
| --- | --- | --- |
| `go build`（默认） | 6.45 MB | 带符号表、调试信息、本机路径 |
| `build.ps1` | **4.54 MB** | −30%，去掉符号/调试信息与构建路径 |
| `build.ps1 -Upx` | **1.52 MB** | −76%，启动自解压（偶发杀软误报） |

关键参数：`-trimpath -buildvcs=false`、`-s -w -H windowsgui -buildid=`、`CGO_ENABLED=0`。

### 12.1 版本号只写一处

`VERSION.txt` 是唯一来源，一次构建注入三处：

| 位置 | 注入方式 | 读取方式 |
| --- | --- | --- |
| 运行时 | `-ldflags -X gogui/internal/version.Number=...` | `version.String()` |
| exe 文件属性 | `tools/winres` 生成 syso | 右键 → 详细信息 |
| 安装器 | `makensis /DVERSION=...` | 「应用和功能」里的版本 |

```go
version.String()   // v0.1.0
version.Title()    // GoGUI v0.1.0（窗口标题）
version.Full()     // v0.1.0 · release · 1a2b3c4 · 2026-10-29T03:49:04Z
```

### 12.2 安装器支持两种范围

| 选项 | 安装目录 | 注册表 | 权限 |
| --- | --- | --- | --- |
| 仅为我安装（默认） | `%LOCALAPPDATA%\Programs\GoGUI` | HKCU | 无需管理员 |
| 为所有用户安装 | `%ProgramFiles%\GoGUI` | HKLM | 未提权自动 UAC 重入 |

静默安装：

```powershell
GoGUI-0.1.0-setup.exe /S /currentuser /D=D:\Apps\GoGUI
GoGUI-0.1.0-setup.exe /S /allusers
```

`/D` 必须是最后一个参数，路径不加引号。

---

## 13. 跨平台：换后端不改业务代码

`internal/gpu.Backend` 已经抽象，新增平台只需要实现窗口层 + 渲染层：

| 平台 | 窗口层 | 渲染层 | 状态 |
| --- | --- | --- | --- |
| Windows | Win32 | OpenGL 3.3 | 已完成 |
| macOS | Cocoa + CAMetalLayer | Metal / OpenGL 4.1 | 规划 |
| Linux | X11 / Wayland | OpenGL / Vulkan | 规划 |
| Android | ANativeWindow + EGL | GLES 3 | 规划 |
| Web | canvas | WebGL 2 + WASM | 适配层已在仓库 |

WASM 侧复用同一套 `ui` 树：`internal/gpu/glport_wasm.go` 提供 WebGL2 的 shader/program/buffer/texture 封装，`ui/*_wasm.go` 提供平台桩。浏览器示例目录尚未随主干发布，先把适配层留在仓库里。

---

## 14. 已知限制（先说清楚）

- 富文本（行内混排 / 图文环绕）与竖排未实现；
- 图片节点按目标框拉伸绘制（无 cover 裁切），封面类场景建议用同比例图或渐变占位；
- 视频解码依赖外部 `ffmpeg` / `ffprobe` 子进程（大分辨率软解 CPU 占用高），没有 ffmpeg 时回退 Windows Media Foundation；
- WebView2 的 DComp 合成仍是实验性开关；
- 当前生产可用平台只有 Windows。

---

## 15. 上手

```bash
git clone https://github.com/2252688138/Go-gui
cd Go-gui

go run ./cmd/gallery          # 8 个分类页（含页内页签）
go run ./cmd/gallery --case table
go run ./cmd/gallery
```

调试工具：

```bash
go run ./tools/layouttest     # 打印完整布局树（不需要开窗口）
go run ./tools/fonttest       # 打印字形度量
go run ./tools/atlasdump      # ASCII 打印字形图集
```

或者在运行参数里设置 `DumpLayout: "layout.txt"`，首帧后导出布局树并退出。

组件清单与「按文件模块化引入」的说明在 [`ui/COMPONENTS.md`](https://github.com/2252688138/Go-gui/blob/main/ui/COMPONENTS.md)：每个组件只依赖 `internal/*` 和基础节点，拷进新项目即可用。

---

## 最后：什么时候该用，什么时候别用

**适合：**

- 想要单文件、无运行库的 Windows 工具；
- 需要 Web 风格的深色界面，但不想背一个浏览器内核；
- 对启动速度、包体、内存有要求；
- 愿意接受「控件要自己攒」的代价。

**不适合：**

- 需要复杂富文本编辑、表格公式、无障碍支持；
- 团队更熟悉前端，且能接受 WebView 的体积；
- 需要立刻上 macOS / Linux 生产环境。

自绘 GUI 的本质是**把浏览器那一层重新实现一遍，但只实现你需要的部分**。它不轻松，但每一行都在掌控之中——这大概就是造轮子的全部乐趣。

如果这个项目对你有帮助，欢迎 Star：<https://github.com/2252688138/Go-gui>

有问题也可以在仓库提 Issue，或者加 QQ 群交流（见仓库作者信息）。
