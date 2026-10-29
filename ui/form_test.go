// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestValidateHelpers 表单校验函数返回对应错误文案。
func TestValidateHelpers(t *testing.T) {
	if got := ValidateRequired("名称", "  "); got != "请填写名称" {
		t.Fatalf("必填校验 = %q", got)
	}
	if got := ValidateRequired("名称", "abc"); got != "" {
		t.Fatalf("非空应为空错误，实际 %q", got)
	}
	if got := ValidateMinLen("密码", "ab", 3); got == "" {
		t.Fatal("长度不足应报错")
	}
	if got := ValidateEmail("邮箱", "a@b"); got == "" {
		t.Fatal("缺少顶级域应报错")
	}
	if got := ValidateEmail("邮箱", "a@b.com"); got != "" {
		t.Fatalf("合法邮箱不应报错，实际 %q", got)
	}
}

// TestSelectStatePick 下拉框：展开→键盘移动（跳过禁用项）→确认→值回调。
func TestSelectStatePick(t *testing.T) {
	s := NewSelect("t-role")
	opts := []SelectOption{
		{Value: "a", Label: "A"},
		{Value: "b", Label: "B", Disabled: true},
		{Value: "c", Label: "C"},
	}
	got := ""
	SelectBox(s, opts, "a", "请选择", func(v string) { got = v })
	s.Toggle()
	if !s.Open || s.Hover != 0 {
		t.Fatalf("展开后应高亮当前项 0，实际 open=%v hover=%d", s.Open, s.Hover)
	}
	selectKey(s, 0x28) // ↓：应跳过禁用的 b
	if s.Hover != 2 {
		t.Fatalf("↓ 应跳过禁用项到 2，实际 %d", s.Hover)
	}
	selectKey(s, 0x0D) // Enter
	if s.Open {
		t.Fatal("确认后应收起")
	}
	if s.Value != "c" || got != "c" {
		t.Fatalf("值应为 c，实际 state=%q cb=%q", s.Value, got)
	}
	// Esc 收起 / 禁用项不可选
	s.Toggle()
	selectKey(s, 0x1B)
	if s.Open {
		t.Fatal("Esc 应收起")
	}
	s.pick(1)
	if s.Value != "c" {
		t.Fatalf("禁用项不应被选中，实际 %q", s.Value)
	}
}

// TestTextareaCaretNav 多行输入光标导航：行首 / 行尾 / 上下行保持列位置。
func TestTextareaCaretNav(t *testing.T) {
	rs := []rune("abc\ndefg\nhi")
	up := func(rs []rune, c int) int { return taVertical(rs, c, -1) }
	down := func(rs []rune, c int) int { return taVertical(rs, c, 1) }

	if got := taLineStart(rs, 6); got != 4 {
		t.Fatalf("行首 = %d，期望 4", got)
	}
	if got := taLineEnd(rs, 6); got != 8 {
		t.Fatalf("行尾 = %d，期望 8", got)
	}
	if got := up(rs, 6); got != 2 {
		t.Fatalf("上一行同列 = %d，期望 2", got)
	}
	if got := down(rs, 6); got != 11 {
		t.Fatalf("下一行同列 = %d，期望 11", got)
	}
	// 目标行更短时截到行尾
	if got := down(rs, 3); got != 7 {
		t.Fatalf("下一行截断 = %d，期望 7", got)
	}
}

// TestFormHeaderLayout 纵向表单项：标签 + 必填星号 + 错误提示行。
func TestFormHeaderLayout(t *testing.T) {
	n := FormItem("标题", true, "必填项", Box().St(St().H(20)))
	if len(n.Children) != 3 {
		t.Fatalf("应为 标签/控件/错误 三行，实际 %d", len(n.Children))
	}
	if len(n.Children[0].Children) != 2 {
		t.Fatal("必填标签应带星号")
	}
	if n.Children[2].Text != "必填项" {
		t.Fatalf("错误文案 = %q", n.Children[2].Text)
	}
}

// TestFormLayoutSmoke 表单组件能完成布局（尺寸有效、不 panic）。
func TestFormLayoutSmoke(t *testing.T) {
	sel := NewSelect("t-sel")
	cal := NewDate("t-date")
	root := Column(
		FormItemInline("名称", true, "请填写名称", Box(Input("t-name", "", "名称", nil))),
		FormItemInline("角色", false, "", SelectBox(sel, []SelectOption{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}, "a", "请选择", nil)),
		FormItem("简介", false, "", Box(Textarea("t-bio", "第一行\n第二行", "占位", 80, nil))),
		FormItemInline("年限", false, "", NumberInput("t-num", 3, 0, 10, 1, nil)),
		FormItem("搜索", false, "", SearchInput("t-search", "go", "搜索…", nil)),
		FormItemInline("日期", false, "", DateBox(cal, "2026-10-27", nil)),
	).St(St().Padding(20).GapOf(14))
	Layout(root, 640, 800, newTestCache())

	if root.Frame.H <= 0 {
		t.Fatal("表单应有非零高度")
	}
	for _, k := range []string{"t-sel", "t-bio", "t-num", "t-search", "t-date"} {
		n := findKey(root, k)
		if n == nil {
			t.Fatalf("缺少节点 %s", k)
		}
		if n.Frame.W <= 0 || n.Frame.H <= 0 {
			t.Fatalf("%s 尺寸无效: %.1fx%.1f", k, n.Frame.W, n.Frame.H)
		}
	}
}

// TestDialogsRender 弹窗：关闭返回 nil，打开返回全屏层且卡片居中。
func TestDialogsRender(t *testing.T) {
	if n := AlertDlg("a", false, DlgInfo, "t", "b", nil); n != nil {
		t.Fatal("关闭状态应返回 nil")
	}
	n := ModalPanel(DialogOpts{
		Key: "t-modal", Open: true, Kind: DlgWarn, Title: "标题", Body: "正文",
		Closable: true, CancelLabel: "取消", OKLabel: "确定",
	})
	if n == nil || n.Kind != KStack {
		t.Fatalf("弹窗应为 Stack 层，实际 %+v", n)
	}
	if len(n.Children) != 2 {
		t.Fatalf("弹窗应有遮罩+卡片两层，实际 %d", len(n.Children))
	}
	if !n.Children[0].Style.PosAbs || !n.Children[1].Style.PosAbs {
		t.Fatal("遮罩与卡片应为绝对定位")
	}
	if !n.Children[1].Style.AbsCenterX || !n.Children[1].Style.AbsCenterY {
		t.Fatal("卡片应水平垂直居中")
	}
}
