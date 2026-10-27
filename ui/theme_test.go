package ui

import "testing"

// TestThemeSwitch 主题切换后 token 与生效色板同步更新（组件按帧重建即换色）。
func TestThemeSwitch(t *testing.T) {
	defer UseTheme(ThemeDark)

	UseTheme(ThemeDark)
	if !IsDark() {
		t.Fatal("默认应为深色")
	}
	if ColText != Dark.Text || ColBg != Dark.Bg || ColInput != Dark.Input {
		t.Fatal("深色 token 未生效")
	}

	UseTheme(ThemeLight)
	if IsDark() {
		t.Fatal("切到浅色后 IsDark 应为 false")
	}
	if ColText != Light.Text || ColBg != Light.Bg || ColInput != Light.Input {
		t.Fatalf("浅色 token 未生效: text=%q bg=%q input=%q", ColText, ColBg, ColInput)
	}
	if ColScrim != Light.Scrim || ColPanel != Light.Panel {
		t.Fatal("浮层/遮罩 token 未跟随主题")
	}

	UseTheme(ThemeDark)
	if ColText != Dark.Text {
		t.Fatal("切换回深色失败")
	}
}

// TestUseThemeInvalid 非法模式回落到深色，不应 panic。
func TestUseThemeInvalid(t *testing.T) {
	defer UseTheme(ThemeDark)
	UseTheme("rainbow")
	if ThemeMode() != ThemeDark || !IsDark() {
		t.Fatalf("非法模式应回落深色，实际 %q", ThemeMode())
	}
}

// TestSystemThemeDetect 系统主题探测有确定返回值且不 panic。
func TestSystemThemeDetect(t *testing.T) {
	_ = SystemPrefersDark()
	UseTheme(ThemeSystem)
	if ThemeMode() != ThemeSystem {
		t.Fatalf("模式应为 system，实际 %q", ThemeMode())
	}
	// system 模式下生效色板必须是深/浅之一
	if n := CurrentTheme().Name; n != "dark" && n != "light" {
		t.Fatalf("系统主题色板异常: %q", n)
	}
}

// TestMarkdownThemeFollows Markdown 色板跟随主题（否则浅色下仍是黑底白字）。
func TestMarkdownThemeFollows(t *testing.T) {
	defer UseTheme(ThemeDark)

	UseTheme(ThemeDark)
	darkText, darkCodeBg := mdColText, mdColCodeBg
	if mdColBorder != Dark.Stroke {
		t.Fatalf("深色下 Markdown 边框应等于主题描边，实际 %q", mdColBorder)
	}

	UseTheme(ThemeLight)
	if mdColText == darkText {
		t.Fatal("Markdown 正文颜色未跟随主题变化")
	}
	if mdColCodeBg == darkCodeBg {
		t.Fatal("Markdown 代码块底色未跟随主题变化")
	}
	if mdColBorder != Light.Stroke {
		t.Fatalf("浅色下 Markdown 边框应为 %q，实际 %q", Light.Stroke, mdColBorder)
	}
	if mdColCodeBg != "#f6f8fc" || mdColText != "#1f2733" {
		t.Fatalf("浅色 Markdown 色板异常: text=%q code=%q", mdColText, mdColCodeBg)
	}
	if mdSynText != "#1f2328" {
		t.Fatalf("代码高亮默认色未切浅色: %q", mdSynText)
	}

	UseTheme(ThemeDark)
	if mdColText != darkText || mdColCodeBg != darkCodeBg {
		t.Fatal("切回深色后 Markdown 色板未恢复")
	}
}

// TestScrollbarPaintFollowsTheme 滚动条颜色跟随主题：深色=半透明白，浅色=半透明深灰。
func TestScrollbarPaintFollowsTheme(t *testing.T) {
	defer UseTheme(ThemeDark)

	UseTheme(ThemeDark)
	d := scrollbarPaint(0.4)
	if d.R < 0.9 || d.A != 0.4 {
		t.Fatalf("深色滚动条应为白色半透明，实际 R=%.2f A=%.2f", d.R, d.A)
	}

	UseTheme(ThemeLight)
	l := scrollbarPaint(0.4)
	if l.R > 0.4 {
		t.Fatalf("浅色滚动条应为深色，实际 R=%.2f", l.R)
	}
	if l.A <= 0.4 {
		t.Fatalf("浅色下应提升透明度以保对比度，实际 A=%.2f", l.A)
	}
	// 激活态更深更实
	if a := scrollbarPaint(0.42).A; a <= l.A {
		t.Fatalf("激活态应比空闲态更明显: idle=%.2f active=%.2f", l.A, a)
	}
}

// TestUIScaleClamp 界面缩放夹取到 60%~150%。
func TestUIScaleClamp(t *testing.T) {
	defer SetUIScale(1)

	SetUIScale(0.1)
	if got := UIScale(); got != UIScaleMin {
		t.Fatalf("下限应为 %v，实际 %v", UIScaleMin, got)
	}
	SetUIScale(9)
	if got := UIScale(); got != UIScaleMax {
		t.Fatalf("上限应为 %v，实际 %v", UIScaleMax, got)
	}
	SetUIScale(1.25)
	if got := UIScale(); got != 1.25 {
		t.Fatalf("应为 1.25，实际 %v", got)
	}
}
