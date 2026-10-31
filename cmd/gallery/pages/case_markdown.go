// Go-gui — gallery 页面拆分：Markdown 渲染
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
)

// ---- 案例：Markdown 渲染 ----

// MdTab 当前示例文档页签（跨帧保持）。
var MdTab = "gfm"

// mdSampleGFM GFM 基础语法。
const mdSampleGFM = `# GoGUI Markdown 渲染

参照 Web 版 **Markdown.vue** 的渲染思路：*块级解析* → ~~行内样式~~ → 组件映射。
行内支持：**加粗**、*斜体*、***粗斜体***、~~删除线~~、「行内代码」、==高亮==、[链接](https://go.dev)、<kbd>Ctrl</kbd>+<kbd>S</kbd>。

![GoGUI 图标](frontend/public/favicon.ico)

## 二级标题

### 三级标题

普通段落会自动折行：GoGUI 的 KText 只支持单一样式，组件在解析阶段把行内片段切成 token，
按可用宽度做贪心折行（中文逐字可断，英文按空格断行），因此长段落、窄窗口都不会溢出容器。

> 引用块：支持多行引用。
> 第二行仍然属于同一个引用。

> [!TIP]
> GitHub 风格的提示块：识别 [!NOTE] / [!TIP] / [!WARNING] / [!IMPORTANT] / [!CAUTION]。

- 无序列表项一
- 无序列表项二
  - 嵌套子项（按缩进识别）
  - 另一个子项
- 任务列表：
  - [x] 完成块级解析
  - [ ] 待办：Mermaid / 数学公式

1. 有序列表
2. 第二项
3. 第三项

| 能力 | Web 版 | GoGUI 版 |
| --- | :---: | --- |
| 标题 / 段落 / 列表 | ✅ | ✅ |
| 代码高亮 | highlight.js | 内置轻量着色 |
| 图片预览 | 全屏 lightbox | 全屏 lightbox |
| 远程图片 | ✅ | 仅本地文件 |

---

文字结尾。
`

// mdSampleExt 扩展块。
const mdSampleExt = `# 扩展块

[toc]

## 页签

[tabs]
[tab label="概览"]
这是 **第一个页签** 的内容。

- 支持块级语法
- 支持嵌套列表

[/tab]
[tab label="代码"]
` + "```go" + `
func main() {
    fmt.Println("hello")
}
` + "```" + `
[/tab]
[tab label="表格"]
| A | B |
| --- | --- |
| 1 | 2 |
[/tab]
[/tabs]

## 折叠

[details title="点击查看实现细节"]
[details] 内部可以继续写 Markdown：

1. 解析扩展块
2. 递归渲染子块
3. 状态按 key 保存在组件内
[/details]

## 卡片

[card title="GoGUI" subtitle="纯 Go 的即时模式 GUI" url="https://go.dev"]
卡片正文支持 **行内样式** 与列表：

- 声明式节点
- 帧内重建
- 状态按 Key 继承
[/card]

## 徽章与进度

[badge text="稳定" color="success"] [badge text="Beta" color="warning"] [badge text="实验" color="error"]

[progress value="72" max="100" label="组件完成度"]

## 图片集

[gallery]
![示例 1](frontend/public/favicon.ico)
![示例 2](frontend/public/favicon.ico)
[/gallery]

## 音视频占位

[video src="movie.mp4" title="示例视频"]
[audio src="song.mp3" title="示例音频"]
`

// mdSampleCode 代码高亮样本。
const mdSampleCode = "# 代码块渲染\n\n代码块会自动识别语言做轻量着色（关键字 / 字符串 / 注释 / 数字 / 函数名），\n并带行号、复制按钮；超过 22 行自动折叠。\n\n```go\npackage main\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n)\n\n// Greet 返回问候语\nfunc Greet(name string) string {\n\tif strings.TrimSpace(name) == \"\" {\n\t\tname = \"GoGUI\"\n\t}\n\treturn fmt.Sprintf(\"hello, %s\", name)\n}\n\nfunc main() {\n\tfmt.Println(Greet(\"markdown\"))\n}\n```\n\n```typescript\ninterface User {\n  id: number\n  name: string\n}\n\nexport async function load(id: number): Promise<User> {\n  const res = await fetch(`/api/users/${id}`)\n  if (!res.ok) throw new Error('load failed')\n  return res.json()\n}\n```\n\n```python\ndef fib(n: int) -> int:\n    \"\"\"斐波那契\"\"\"\n    a, b = 0, 1\n    for _ in range(n):\n        a, b = b, a + b\n    return a\n\nprint(fib(10))\n```\n\n```json\n{\n  \"name\": \"gogui\",\n  \"markdown\": true,\n  \"version\": 1.0\n}\n```\n\n```bash\n# 运行示例\ngo run ./cmd/gallery --case markdown\n```\n"

// mdSampleLong 长文 + 目录。
const mdSampleLong = `# 长文与目录

[toc]

## 一、设计目标

组件需要在「只有单一样式文本节点」的框架上还原 Web 版 Markdown 的观感：

1. **块级结构** —— 标题、段落、列表、引用、代码、表格、分隔线、图片；
2. **行内样式** —— 粗体、斜体、删除线、行内代码、链接、高亮、kbd；
3. **扩展块** —— tabs / details / card / badge / progress / gallery；
4. **交互** —— 代码复制、折叠展开、页签切换、图片全屏预览、链接外部打开。

## 二、排版策略

### 2.1 折行

框架的 KText 绘制只按换行符分行，Wrap 只影响测量。组件在解析后把行内片段切成 token，
用字体度量做贪心折行，再把每一行生成为 Row（每个 token 一个 Text 节点），
从而保证「测量高度 = 实际绘制高度」。

### 2.2 状态

页签、折叠、代码展开、复制提示、图片预览都按「父级 key + 块序号」保存在包级状态里，
因此界面每帧重建也不丢失交互状态。

### 2.3 性能

- 解析结果按源码字符串缓存（最多 24 篇，超出清空）；
- 每帧只重建节点、不重复解析；
- 字体走窗口级字体缓存（LRU 24 个字体实例）。

## 三、已知限制

- 远程图片会下载到临时目录缓存后正常显示（浏览器版仍为占位卡）；
- 音视频块降级为占位卡，真正播放请用 MusicPlayer / VideoPlayer；
- 数学公式为 LaTeX 子集，未覆盖的写法会退化成等宽源码；
- 表格列宽按内容比例分配，列很多时会出现横向压缩。

## 四、后续计划

- [x] 数学公式（LaTeX 子集真排版）
- [x] Mermaid 代码块 → 提示卡
- [x] 目录锚点滚动
- [x] 代码高亮
- [x] 图片 lightbox

> 备注：组件文件为 ui/markdown.go（渲染）、ui/markdown_parse.go（解析）、
> ui/markdown_highlight.go（着色）、ui/markdown_state.go（状态）。
`

// mdSampleNew 本轮新增渲染项的演示文档。
const mdSampleNew = `# 新增渲染项

本页演示本轮新增能力：数学公式、Mermaid 提示卡、脚注、定义列表、链接卡、diff 着色、上标下标、任务勾选与目录锚点滚动。

## 一、数学公式

行内：$E=mc^2$，求根公式写成 $x = (-b+\sqrt{b^2-4ac})/(2a)$，欧拉恒等式 $e^{i\pi}+1=0$。

块级：

$$
f(x) = \frac{ab} + \sqrt{x^2+1}
$$

$$
\sum_{k=1}^{n} k = \frac{n(n+12} + \int_0^1 x\,dx = \frac{12}
$$

矩阵与分段：

$$
A = \begin{pmatrix} a_{11} & a_{12} \\ a_{21} & a_{22} \end{pmatrix}, \quad |x| = \begin{cases} x & x \ge 0 \\ -x & x < 0 \end{cases}
$$

## 二、Mermaid 流程图（原生绘制）

~~~mermaid
flowchart LR
  A[解析] --> B{有缓存?}
  B -->|是| C[复用渲染树]
  B -->|否| D[重建节点]
  D --> E[写回缓存]
~~~

~~~mermaid
flowchart TD
  S[请求] --> P{命中缓存?}
  P -->|是| R[返回渲染树]
  P -->|否| B[解析源码]
  B --> L[分层排版]
  L --> R
~~~

## 三、脚注与定义列表

脚注引用会自动上标并与文末条目互链[^1]，支持缩进续行[^2]。

渲染树缓存
: 相同输入复用同一批节点，滚动只做一次 map 查找。
: 交互状态（页签 / 折叠 / 勾选）参与缓存键。

性能
: 长文档命中缓存后单帧只剩节点指针复制。

## 四、链接卡与自动链接

https://go.dev/doc/

裸链接同样可点击：https://github.com/golang/go

## 五、diff 代码块

~~~diff
--- a/ui/markdown.go
+++ b/ui/markdown.go
@@ -12,3 +12,4 @@
-// 旧：每帧重建整篇文档
+// 新：渲染树缓存 + 状态版本号
+// 新：diff 行底色
~~~

## 六、任务列表可勾选

- [x] 数学公式排版
- [ ] 点击方块可切换勾选状态

## 七、目录锚点滚动

点击下面目录条目可平滑滚动到对应小节。

[toc]

## 八、远程图片

桌面端会下载并缓存到临时目录，浏览器版显示占位卡。

[^1]: 脚注内容由 doc.Footnotes 统一渲染在文末。
[^2]: 第二行缩进续写也归属于同一条脚注。
`

// markdownCase Markdown 组件示例页。
func markdownCase() *ui.Node {
	docs := []struct{ id, label, src string }{
		{"gfm", "GFM 基础", mdSampleGFM},
		{"ext", "扩展块", mdSampleExt},
		{"code", "代码高亮", mdSampleCode},
		{"long", "长文 / 目录", mdSampleLong},
		{"new", "新增渲染", mdSampleNew},
	}
	keys := make([]string, 0, len(docs))
	labels := make([]string, 0, len(docs))
	src := docs[0].src
	for _, d := range docs {
		keys = append(keys, d.id)
		labels = append(labels, d.label)
		if d.id == MdTab {
			src = d.src
		}
	}
	w := ui.ViewportW() - 232 - 96
	if w < 320 {
		w = 320
	}
	return ui.Column(
		ui.Column(
			ui.Row(
				ui.Text("Markdown 渲染").St(ui.St().Text(17, 700, ui.ColText)),
				ui.Spacer(),
				ui.Tag("ui.Markdown", "#a5b4fc"),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
			ui.Text("对齐 Web 版 Markdown.vue：块级解析 + 行内样式 + 扩展块；代码块带行号 / 着色 / 复制 / 折叠，图片点击可全屏预览。").St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
			ui.Tabs(keys, labels, MdTab, func(k string) {
				MdTab = k
				ui.RequestRedraw()
			}),
			ui.Divider(),
			ui.Markdown("gallery-md", src, ui.MarkdownOptions{Width: w}),
		).St(ui.St().Padding(22).R(16).BgC(ui.ColBgDeep).Stroke(1, ui.ColStroke).GapOf(12)),
	).St(ui.St().Padding(24).Wp(100))
}
