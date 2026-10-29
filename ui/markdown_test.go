// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"strings"
	"testing"
)

const mdTestDoc = `# 标题一

段落包含 **粗体**、*斜体*、` + "`代码`" + `、~~删除~~、[链接](https://go.dev) 与 ==高亮==。

> 引用第一行
> 引用第二行

- 列表项 A
- 列表项 B
  - 嵌套项

1. 有序一
2. 有序二

- [x] 已完成
- [ ] 未完成

| 名称 | 值 |
| --- | :---: |
| a | 1 |

` + "```go\nfunc main() {}\n```" + `

---

[tabs]
[tab label="一"]
页签一的内容
[/tab]
[tab label="二"]
页签二的内容
[/tab]
[/tabs]

[details title="折叠标题"]
折叠内容
[/details]

[card title="卡片" subtitle="副标题"]
卡片正文
[/card]

[badge text="徽章" color="success"]

[progress value="50" max="100" label="进度"]

[gallery]
![图1](a.png)
![图2](b.png)
[/gallery]

[video src="movie.mp4" title="视频"]
`

func TestParseMarkdownBlocks(t *testing.T) {
	doc := parseMarkdown(mdTestDoc)
	if len(doc.Blocks) == 0 {
		t.Fatal("no blocks parsed")
	}
	seen := map[mdKind]bool{}
	for _, b := range doc.Blocks {
		seen[b.Kind] = true
	}
	for _, k := range []mdKind{mdHeading, mdParagraph, mdQuote, mdList, mdTable, mdCode, mdHR, mdTabs, mdDetails, mdCard, mdBadge, mdProgress, mdGallery, mdMedia} {
		if !seen[k] {
			t.Errorf("block kind %d not parsed", k)
		}
	}
	if len(doc.Headings) != 1 || doc.Headings[0].Level != 1 {
		t.Errorf("headings = %+v", doc.Headings)
	}

	var tabs *mdBlock
	var table *mdBlock
	var list *mdBlock
	for _, b := range doc.Blocks {
		switch b.Kind {
		case mdTabs:
			tabs = b
		case mdTable:
			table = b
		case mdList:
			if list == nil && len(b.Items) >= 2 {
				list = b
			}
		}
	}
	if tabs == nil || len(tabs.Tabs) != 2 {
		t.Fatalf("tabs = %+v", tabs)
	}
	if !strings.Contains(tabs.Tabs[1].Label, "二") {
		t.Errorf("tab label = %q", tabs.Tabs[1].Label)
	}
	if len(tabs.Tabs[1].Blocks) == 0 || tabs.Tabs[1].Blocks[0].Kind != mdParagraph {
		t.Errorf("tab body = %+v", tabs.Tabs[1].Blocks)
	}
	if table == nil || table.Table == nil || len(table.Table.Rows) != 1 {
		t.Fatalf("table = %+v", table)
	}
	if table.Table.Header[1].Align != AlignCenter {
		t.Errorf("table align = %v", table.Table.Header[1].Align)
	}
	if list == nil || len(list.Items) < 2 {
		t.Fatalf("list = %+v", list)
	}
	if got := list.Items[0].Task; got != 0 && got != -1 {
		t.Errorf("task flag = %d", got)
	}
}

func TestParseInlineStyles(t *testing.T) {
	runs := parseInline("a **b** *c* `d` ~~e~~ [f](u) ==g== <kbd>K</kbd>")
	if len(runs) < 7 {
		t.Fatalf("runs = %+v", runs)
	}
	var bold, italic, code, strike, link, mark, kbd bool
	for _, r := range runs {
		switch {
		case r.Bold:
			bold = true
		case r.Italic:
			italic = true
		case r.Code:
			code = true
		case r.Strike:
			strike = true
		case r.Link:
			link = r.Href == "u"
		case r.Mark:
			mark = true
		case r.Kbd:
			kbd = true
		}
	}
	if !bold || !italic || !code || !strike || !link || !mark || !kbd {
		t.Errorf("missing style: bold=%v italic=%v code=%v strike=%v link=%v mark=%v kbd=%v",
			bold, italic, code, strike, link, mark, kbd)
	}
}

func TestWrapTextLines(t *testing.T) {
	fc := newFontCache("", 1)
	f := fc.getFamily(13, 400, "")
	if f == nil {
		t.Skip("no font")
	}
	defer fc.close()
	text := "中文自动折行测试，包含 English words and a very long token abcdefghijklmnopqrstuvwxyz0123456789"
	lines := wrapTextLines(f, text, 200)
	if len(lines) < 2 {
		t.Fatalf("expected wrapped lines, got %v", lines)
	}
	for _, ln := range lines {
		if w := f.Measure(ln); w > 200+1 {
			t.Errorf("line too wide: %.1f %q", w, ln)
		}
	}
	joined := strings.ReplaceAll(strings.Join(lines, ""), " ", "")
	if strings.ReplaceAll(strings.ReplaceAll(text, " ", ""), "", "") != joined {
		// 空格在折行处被丢弃，只校验非空格内容不丢失
		t.Errorf("content lost: %q", joined)
	}
}

func TestMarkdownLayoutSmoke(t *testing.T) {
	MarkdownResetState()
	n := Markdown("test-md", mdTestDoc, MarkdownOptions{Width: 520})
	if n == nil || len(n.Children) == 0 {
		t.Fatal("empty markdown node")
	}
	LayoutForTest(n, 900, 1200, "", 1)
	if n.Frame.H <= 0 {
		t.Errorf("layout height = %.1f", n.Frame.H)
	}
	if n.Children[0].Frame.W <= 0 || n.Children[0].Frame.W > 900 {
		t.Errorf("child width = %.1f", n.Children[0].Frame.W)
	}
}

func TestMarkdownHighlight(t *testing.T) {
	lines := mdHighlight("go", "func main() {\n\t// hi\n\tx := \"str\"\n}")
	if len(lines) != 4 {
		t.Fatalf("lines = %d", len(lines))
	}
	var hasKw, hasCom, hasStr bool
	for _, ln := range lines {
		for _, sp := range ln {
			switch sp.Kind {
			case "kw":
				hasKw = true
			case "com":
				hasCom = true
			case "str":
				hasStr = true
			}
		}
	}
	if !hasKw || !hasCom || !hasStr {
		t.Errorf("highlight kinds missing: kw=%v com=%v str=%v", hasKw, hasCom, hasStr)
	}
}
