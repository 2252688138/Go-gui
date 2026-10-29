package ui

import (
	"strings"
	"testing"
)

const mdTestExtDoc = `# 新渲染项

行内公式 $E=mc^2$、上标 x^2^、下标 H~2~O，以及脚注引用[^1]。

$$
\sum_{i=1}^{n} i^2 = \frac{n(n+1)(2n+16}
$$

~~~math
\sqrt{a^2+b^2} \le |a| + |b|
~~~

~~~mermaid
flowchart TD
  A[开始] --> B{判断}
  B -->|是| C[结束]
~~~

术语
: 释义一
: 释义二

https://go.dev/doc

[^1]: 这是脚注内容。

~~~diff
--- a/x.go
+++ b/x.go
@@ -1,2 +1,3 @@
-func a() {}
+func b() {}
+func c() {}
~~~

- [x] 已完成
- [ ] 待办
`

func TestParseMathAndExt(t *testing.T) {
	doc := parseMarkdown(mdTestExtDoc)
	seen := map[mdKind]int{}
	for _, b := range doc.Blocks {
		seen[b.Kind]++
	}
	if seen[mdMath] != 2 {
		t.Fatalf("公式块数量 = %d，期望 2", seen[mdMath])
	}
	if seen[mdMermaid] != 1 {
		t.Fatalf("mermaid 块 = %d", seen[mdMermaid])
	}
	if seen[mdDefList] != 1 {
		t.Fatalf("定义列表 = %d", seen[mdDefList])
	}
	if len(doc.Footnotes) != 1 || doc.Footnotes[0].ID != "1" {
		t.Fatalf("脚注 = %+v", doc.Footnotes)
	}

	var def *mdBlock
	for _, b := range doc.Blocks {
		if b.Kind == mdDefList {
			def = b
		}
	}
	if len(def.Defs) != 1 || len(def.Defs[0].Defs) != 2 {
		t.Fatalf("定义列表项 = %+v", def.Defs)
	}

	// 行内：公式 / 上标 / 下标 / 脚注引用 / 裸链接
	var math, sup, sub, fn, link bool
	for _, b := range doc.Blocks {
		for _, r := range b.Inl {
			switch {
			case r.Math != "":
				math = true
			case r.Sup != "":
				sup = true
			case r.Sub != "":
				sub = true
			case r.FnRef != "":
				fn = true
			case r.Link:
				link = true
			}
		}
	}
	if !math || !sup || !sub || !fn || !link {
		t.Errorf("行内解析缺失：math=%v sup=%v sub=%v fn=%v link=%v", math, sup, sub, fn, link)
	}
}

func TestMathLayoutBoxes(t *testing.T) {
	cases := []string{
		"E=mc^2",
		`\frac{a+bc}`,
		`\sqrt{x^2+y^2}`,
		`\sum_{i=1}^{n} i`,
		`\int_0^1 x^2 dx`,
		`\left( \frac{12} \right)^2`,
		`\begin{pmatrix} a & b \\ c & d \end{pmatrix}`,
		`\hat{x} + \bar{y} + \vec{v}`,
		`\alpha \le \beta \Rightarrow \gamma`,
		`\text{文字} + \mathbf{B}`,
	}
	for _, src := range cases {
		b, ok := mdMathBox(src, 14, "", "#ffffff", false)
		if !ok {
			t.Fatalf("%q 排版失败", src)
		}
		if b.w <= 0 || b.h <= 0 {
			t.Fatalf("%q 尺寸异常 w=%.1f h=%.1f", src, b.w, b.h)
		}
		if len(b.kids) == 0 {
			t.Fatalf("%q 没有子节点", src)
		}
		for _, k := range b.kids {
			if k.n == nil {
				t.Fatalf("%q 存在空子节点", src)
			}
		}
	}
}

func TestMathFracIsTaller(t *testing.T) {
	plain, ok := mdMathBox("a", 14, "", "#fff", false)
	if !ok {
		t.Fatal("atom 排版失败")
	}
	fr, ok := mdMathBox(`\frac{ab}`, 14, "", "#fff", false)
	if !ok {
		t.Fatal("分式排版失败")
	}
	if fr.h <= plain.h {
		t.Errorf("分式高度应大于单字符：frac=%.1f plain=%.1f", fr.h, plain.h)
	}
	if fr.base <= plain.base {
		t.Errorf("分式基线应下移：frac=%.1f plain=%.1f", fr.base, plain.base)
	}
}

func TestMathSuperscriptRaises(t *testing.T) {
	sqrtBox, ok := mdMathBox("x^2", 14, "", "#fff", false)
	if !ok {
		t.Fatal("上标排版失败")
	}
	plain, _ := mdMathBox("x2", 14, "", "#fff", false)
	if sqrtBox.w >= plain.w {
		t.Logf("上标宽 %.1f，普通 %.1f（上标字号更小，通常更窄）", sqrtBox.w, plain.w)
	}
	if sqrtBox.h < plain.h {
		t.Errorf("上标盒高度不应小于普通文本：sup=%.1f plain=%.1f", sqrtBox.h, plain.h)
	}
}

func TestMarkdownNewBlocksSmoke(t *testing.T) {
	MarkdownResetState()
	n := Markdown("md-ext", mdTestExtDoc, MarkdownOptions{Width: 640})
	if n == nil || len(n.Children) == 0 {
		t.Fatal("空渲染树")
	}
	LayoutForTest(n, 900, 2000, "", 1)
	if n.Frame.H <= 0 {
		t.Fatalf("布局高度 = %.1f", n.Frame.H)
	}
	// 锚点表应包含各级标题（目录点击依赖）
	if len(mdState.anchors) == 0 {
		t.Error("锚点表为空")
	}
	found := false
	for k := range mdState.anchors {
		if strings.HasPrefix(k, "md-ext|") {
			found = true
			break
		}
	}
	if !found {
		t.Error("锚点键未按文档前缀登记")
	}
}

func TestMarkdownTOCAnchorRegistered(t *testing.T) {
	MarkdownResetState()
	src := "# 一级\n\n[toc]\n\n## 二级\n\n正文"
	_ = Markdown("md-toc", src, MarkdownOptions{Width: 600})
	doc := parseMarkdown(src)
	if len(doc.Headings) != 2 {
		t.Fatalf("标题数 = %d", len(doc.Headings))
	}
	for _, h := range doc.Headings {
		if _, ok := mdState.anchors[mdAnchorID("md-toc", h.Key)]; !ok {
			t.Errorf("标题 %q 未登记锚点", h.Text)
		}
	}
}

func TestHighlightDiff(t *testing.T) {
	lines := mdHighlight("diff", "--- a\n+++ b\n@@ -1 +1 @@\n-old\n+new\n ctx")
	if len(lines) != 6 {
		t.Fatalf("行数 = %d", len(lines))
	}
	want := []string{"meta", "meta", "meta", "del", "add", "txt"}
	for i, w := range want {
		if len(lines[i]) == 0 || lines[i][0].Kind != w {
			t.Errorf("第 %d 行类型 = %v，期望 %s", i, lines[i], w)
		}
	}
}

// TestInlineMathPiece 行内公式应成为独立 token，宽度取自排版盒。
func TestInlineMathPiece(t *testing.T) {
	runs := parseInline("前 $a+b$ 后")
	base := mdBaseSpan(MarkdownOptions{})
	pieces := mdTokens(runs, base, MarkdownOptions{})
	found := false
	for _, p := range pieces {
		if p.math != nil {
			found = true
			if w := mdPieceWidth(p); w <= 0 || w != p.math.w {
				t.Errorf("公式片段宽度 = %.1f，盒宽 = %.1f", w, p.math.w)
			}
			if mdPieceNode(p) == nil {
				t.Error("公式片段未生成节点")
			}
		}
	}
	if !found {
		t.Fatal("未解析出行内公式 token")
	}
}

// TestInlineScriptNodes 上标/下标应生成有高度的节点（不被行高裁掉）。
func TestInlineScriptNodes(t *testing.T) {
	runs := parseInline("x^2^ 与 H~2~O")
	base := mdBaseSpan(MarkdownOptions{})
	for _, p := range mdTokens(runs, base, MarkdownOptions{}) {
		if p.script == 0 {
			continue
		}
		n := mdPieceNode(p)
		if n == nil {
			t.Fatalf("脚本片段 %q 未生成节点", p.text)
		}
		if n.Style.Height <= 0 {
			t.Errorf("脚本片段 %q 高度 = %.1f", p.text, n.Style.Height)
		}
	}
}

// TestMathDisplayBlockNode 块级公式节点宽度应受限且高度为正。
func TestMathDisplayBlockNode(t *testing.T) {
	n := mdMathTypeset(`\frac{a+b}`, 15, "", "#fff", true)
	if n == nil {
		t.Fatal("块级公式未生成节点")
	}
	if n.Style.Height <= 0 || n.Style.Width <= 0 {
		t.Fatalf("公式节点尺寸 = %.1f x %.1f", n.Style.Width, n.Style.Height)
	}
}

const mmTestFlow = `flowchart LR
  A[解析] --> B{有缓存?}
  B -->|是| C[复用渲染树]
  B -->|否| D[重建节点]
  D --> E[写回缓存]
`

func TestMermaidFlowParse(t *testing.T) {
	g, ok := mmParse(mmTestFlow)
	if !ok {
		t.Fatal("流程图解析失败")
	}
	if len(g.nodes) != 5 {
		t.Fatalf("节点数 = %d，期望 5", len(g.nodes))
	}
	if len(g.edges) != 4 {
		t.Fatalf("边数 = %d，期望 4", len(g.edges))
	}
	if g.dir != "LR" {
		t.Errorf("方向 = %q，期望 LR", g.dir)
	}
	// 标签应属于「B→C」这条边，而不是前一条 A→B
	for _, e := range g.edges {
		if e.from == "B" && e.to == "C" && e.label != "是" {
			t.Errorf("B->C 标签 = %q，期望 是", e.label)
		}
	}
	// 形状与文本
	if n := g.node("B"); n == nil || n.shape != mmShapeDiamond || n.label != "有缓存?" {
		t.Errorf("节点 B = %+v", g.node("B"))
	}
	if n := g.node("A"); n == nil || n.label != "解析" {
		t.Errorf("节点 A = %+v", g.node("A"))
	}
}

func TestMermaidFlowLayout(t *testing.T) {
	g, ok := mmParse(mmTestFlow)
	if !ok {
		t.Fatal("流程图解析失败")
	}
	w, h := g.layout(13.5, "")
	if w <= 0 || h <= 0 {
		t.Fatalf("布局尺寸 = %.1f x %.1f", w, h)
	}
	// LR：B 在 A 右侧，C/D 在 B 右侧
	a, b, c := g.node("A"), g.node("B"), g.node("C")
	if !(a.x < b.x && b.x < c.x) {
		t.Errorf("LR 横向顺序不对：A.x=%.1f B.x=%.1f C.x=%.1f", a.x, b.x, c.x)
	}
	if a.layer != 0 || b.layer != 1 || c.layer != 2 {
		t.Errorf("分层 = A%d B%d C%d，期望 0/1/2", a.layer, b.layer, c.layer)
	}
	// 所有节点尺寸为正
	for _, n := range g.nodes {
		if n.w <= 0 || n.h <= 0 {
			t.Errorf("节点 %s 尺寸 = %.1f x %.1f", n.id, n.w, n.h)
		}
	}
}

func TestMermaidFlowView(t *testing.T) {
	g, ok := mmParse(mmTestFlow)
	if !ok {
		t.Fatal("流程图解析失败")
	}
	n, ok := mmView(g, 640, 13.5, "", "mm-test")
	if !ok || n == nil {
		t.Fatal("mmView 未生成节点")
	}
	if n.Style.Width <= 0 || n.Style.Height <= 0 {
		t.Fatalf("视图尺寸 = %.1f x %.1f", n.Style.Width, n.Style.Height)
	}
}

func TestMermaidFlowBlockRendered(t *testing.T) {
	MarkdownResetState()
	src := "~~~mermaid\n" + mmTestFlow + "~~~\n"
	n := Markdown("mm-doc", src, MarkdownOptions{Width: 640})
	if n == nil || len(n.Children) == 0 {
		t.Fatal("空渲染树")
	}
	LayoutForTest(n, 900, 1200, "", 1)
	if n.Frame.H <= 0 {
		t.Fatalf("布局高度 = %.1f", n.Frame.H)
	}
}
