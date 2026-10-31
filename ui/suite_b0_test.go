// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"strings"
	"testing"
)

// TestRichTextWrapAndMeasure 富文本按容器宽度折行，测量高度 = 行高累计。
func TestRichTextWrapAndMeasure(t *testing.T) {
	n := RichText([]TextSpan{
		{Text: "Hello "},
		{Text: "世界，", Weight: 700},
		{Text: "这是一段用于测试折行的长文本", Color: ColTextDim},
	}, RichTextOptions{Size: 13})
	root := Column(n).St(St().W(90))
	Layout(root, 90, 400, newTestCache())

	if !n.rich.valid {
		t.Fatal("富文本未完成测量")
	}
	if len(n.rich.lines) < 2 {
		t.Fatalf("90px 宽应折成多行，实际 %d 行", len(n.rich.lines))
	}
	var sum float32
	for i, ln := range n.rich.lines {
		if ln.w > 91 {
			t.Fatalf("第 %d 行宽度 %.1f 超过容器 90", i, ln.w)
		}
		sum += ln.h
	}
	if !near(sum, n.rich.h) {
		t.Fatalf("行高累计 %.1f 与测量高度 %.1f 不一致", sum, n.rich.h)
	}
	if !near(n.Frame.H, n.rich.h) {
		t.Fatalf("节点高度 %.1f 应等于富文本高度 %.1f", n.Frame.H, n.rich.h)
	}
}

// TestRichTextMaxLinesEllipsis MaxLines 截断并追加省略号。
func TestRichTextMaxLinesEllipsis(t *testing.T) {
	n := RichText([]TextSpan{{Text: strings.Repeat("很长的一段中文文本", 8)}},
		RichTextOptions{Size: 13, MaxLines: 2})
	root := Column(n).St(St().W(120))
	Layout(root, 120, 400, newTestCache())
	if len(n.rich.lines) != 2 {
		t.Fatalf("MaxLines=2 应只保留 2 行，实际 %d", len(n.rich.lines))
	}
	last := n.rich.lines[1]
	if len(last.pieces) == 0 {
		t.Fatal("最后一行不应为空")
	}
	if !strings.HasSuffix(last.pieces[len(last.pieces)-1].text, "…") {
		t.Fatalf("最后一行应以省略号结尾，实际 %q", last.pieces[len(last.pieces)-1].text)
	}
}

// TestRichTextHitLink 命中坐标落在链接片段上时返回该 span。
func TestRichTextHitLink(t *testing.T) {
	n := RichText([]TextSpan{
		{Text: "前置文字 "},
		{Text: "点击详情", Link: "detail", Underline: true},
	}, RichTextOptions{Size: 13})
	root := Column(n).St(St().W(300))
	Layout(root, 300, 100, newTestCache())

	var target richPiece
	found := false
	for _, ln := range n.rich.lines {
		for _, p := range ln.pieces {
			if p.span == 1 {
				target, found = p, true
			}
		}
	}
	if !found {
		t.Fatal("未找到链接片段")
	}
	sp, ok := richAt(n, target.x+target.w/2, n.rich.lines[0].y+4)
	if !ok || sp.Link != "detail" {
		t.Fatalf("命中应为链接片段，实际 %+v ok=%v", sp, ok)
	}
}

// TestFocusTabOrderAndScope Tab 顺序与作用域隔离。
func TestFocusTabOrderAndScope(t *testing.T) {
	a := &App{scrolls: map[string]float32{}}
	a.root = Column(
		Input("f-a", "", "", nil),
		Input("f-b", "", "", nil),
		Column(Input("f-c", "", "", nil)).K("scope"),
	).St(St().W(240))
	Layout(a.root, 240, 240, newTestCache())
	currentApp.Store(a)
	defer func() {
		currentApp.Store(nil)
		ResetFocusScopes()
		a.focusKey = ""
	}()

	ResetFocusScopes()
	a.focusKey = ""
	FocusNext()
	if FocusedKey() != "f-a" {
		t.Fatalf("首个焦点应为 f-a，实际 %q", FocusedKey())
	}
	FocusNext()
	FocusNext()
	if FocusedKey() != "f-c" {
		t.Fatalf("第三个焦点应为 f-c，实际 %q", FocusedKey())
	}
	FocusNext()
	if FocusedKey() != "f-a" {
		t.Fatalf("应循环回 f-a，实际 %q", FocusedKey())
	}
	FocusPrev()
	if FocusedKey() != "f-c" {
		t.Fatalf("反向应为 f-c，实际 %q", FocusedKey())
	}

	// 作用域：Tab 只在 scope 子树内循环
	PushFocusScope("scope", true)
	a.focusKey = ""
	FocusNext()
	if FocusedKey() != "f-c" {
		t.Fatalf("作用域内首个焦点应为 f-c，实际 %q", FocusedKey())
	}
	FocusNext()
	if FocusedKey() != "f-c" {
		t.Fatalf("作用域内应循环在 f-c，实际 %q", FocusedKey())
	}
	PopFocusScope("scope")
	if FocusScopeDepth() != 0 {
		t.Fatalf("作用域应已退出，实际 %d", FocusScopeDepth())
	}
}

// TestFocusVisibleTracksKeyboard 键盘导航显示焦点环，鼠标点击隐藏。
func TestFocusVisibleTracksKeyboard(t *testing.T) {
	a := &App{scrolls: map[string]float32{}}
	a.root = Column(Input("v-a", "", "", nil)).St(St().W(200))
	Layout(a.root, 200, 100, newTestCache())
	currentApp.Store(a)
	defer func() { currentApp.Store(nil); ResetFocusScopes() }()

	SetFocusVisible(false)
	FocusNext()
	if !FocusVisible() {
		t.Fatal("Tab 后应显示焦点环")
	}
	SetFocusVisible(false)
	if FocusVisible() {
		t.Fatal("鼠标点击后应隐藏焦点环")
	}
}

// TestThemeTokensOverride 语义 token 覆盖并随主题切换保留。
func TestThemeTokensOverride(t *testing.T) {
	oldPrimary := ColPrimary
	oldAccent := tokenOverrides.Accent
	defer func() {
		tokenOverrides.Accent = oldAccent
		ClearTokenOverrides()
		UseTheme(ThemeDark)
		_ = oldPrimary
	}()

	UseTheme(ThemeDark)
	SetTokens(Tokens{Accent: "#ff0055"})
	if ColPrimary != "#ff0055" {
		t.Fatalf("Accent 覆盖应同步 ColPrimary，实际 %q", ColPrimary)
	}
	// 切换浅色后覆盖仍生效
	UseTheme(ThemeLight)
	if ColPrimary != "#ff0055" {
		t.Fatalf("切主题后覆盖应保留，实际 %q", ColPrimary)
	}
	ClearTokenOverrides()
	if ColPrimary != Light.Primary {
		t.Fatalf("清除覆盖后应回到浅色主色，实际 %q", ColPrimary)
	}
}

// TestA11ySnapshot 无障碍元数据导出。
func TestA11ySnapshot(t *testing.T) {
	a := &App{scrolls: map[string]float32{}}
	a.root = Column(
		Btn("保存", BtnPrimary, nil).A11yOn(A11yButton, "保存", "保存当前文档"),
		Text("说明").A11yOn(A11yText, "说明", ""),
	)
	Layout(a.root, 300, 100, newTestCache())
	currentApp.Store(a)
	defer currentApp.Store(nil)

	list := A11ySnapshot()
	if len(list) != 2 {
		t.Fatalf("应导出 2 个无障碍节点，实际 %d", len(list))
	}
	if list[0].Role != A11yButton || list[0].Name != "保存" {
		t.Fatalf("首项应为保存按钮，实际 %+v", list[0])
	}
	if A11yRoleName(A11yButton) != "button" {
		t.Fatal("角色名映射错误")
	}
}

// TestFormatNumber 千分位格式化。
func TestFormatNumber(t *testing.T) {
	cases := map[string]string{
		"1234567": "1,234,567",
		"-1234.5": "-1,234.5",
		"999":     "999",
		"0":       "0",
	}
	got := map[string]string{
		"1234567": FormatNumber(1234567, 0),
		"-1234.5": FormatNumber(-1234.5, 1),
		"999":     FormatNumber(999, 0),
		"0":       FormatNumber(0, 0),
	}
	for k, want := range cases {
		if got[k] != want {
			t.Fatalf("FormatNumber(%s) = %q，期望 %q", k, got[k], want)
		}
	}
}
