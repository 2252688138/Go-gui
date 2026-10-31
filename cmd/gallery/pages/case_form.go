// Go-gui — gallery 页面拆分：表单与弹窗
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"fmt"
	"gogui/ui"
	"strconv"
)

// 「表单与弹窗」案例：FormItem / Select / Textarea / NumberInput / SearchInput /
// Chip / DateBox + Alert / Confirm / Prompt / Modal / Drawer。
//
// 浮层（下拉、日历、弹窗）都在根 Stack 末尾渲染：FormOverlays / SelectLayers / DateLayers。

// ---- 案例状态 ----

var (
	formName   = "极码 AI"
	formMail   = ""
	formBio    = ""
	formSearch = ""
	formExp    = float64(3)
	formDate   = ""
	formRole   = "dev"
	formTeam   = "core"

	formErrName = ""
	formErrMail = ""
	formSaved   = ""

	formTags = map[string]bool{"Go": true, "渲染": true, "组件库": false, "OpenGL": false}

	dlgAlert    bool
	dlgConfirm  bool
	dlgPrompt   bool
	dlgModal    bool
	dlgDrawer   bool
	dlgPromptID = "极码 AI"
)

var (
	roleSel = ui.NewSelect("form-role")
	teamSel = ui.NewSelect("form-team")
	dateCal = ui.NewDate("form-date")
)

var roleOpts = []ui.SelectOption{
	{Value: "dev", Label: "研发工程师", Hint: "R&D"},
	{Value: "design", Label: "设计师", Hint: "Design"},
	{Value: "pm", Label: "产品经理", Hint: "PM"},
	{Value: "ops", Label: "运维 / SRE", Hint: "Ops"},
	{Value: "guest", Label: "只读访客", Hint: "禁用示例", Disabled: true},
}

var teamOpts = []ui.SelectOption{
	{Value: "core", Label: "核心引擎组"},
	{Value: "infra", Label: "基础设施组"},
	{Value: "growth", Label: "增长组"},
	{Value: "lab", Label: "图形实验室"},
}

// ---- 页面 ----

func formCase() *ui.Node {
	return ui.Column(
		pageHead("表单与弹窗", "下拉框 / 多行输入 / 数字 / 日期 / 标签 / 校验，以及四类弹窗与抽屉"),

		ui.If(formSaved != "", func() *ui.Node {
			return ui.Banner("success", "操作完成", formSaved, func() {
				formSaved = ""
				ui.RequestRedraw()
			})
		}),

		ui.Card(
			ui.Title("基础信息", "含必填校验：保存时校验名称与邮箱"),
			ui.FormItemInline("名称", true, formErrName, ui.Box(
				ui.Input("form-name", formName, "请输入名称", func(v string) {
					formName = v
					if formErrName != "" {
						formErrName = ui.ValidateRequired("名称", v)
					}
				}),
			)),
			ui.FormItemInline("邮箱", true, formErrMail, ui.Box(
				ui.Input("form-mail", formMail, "name@example.com", func(v string) {
					formMail = v
					if formErrMail != "" {
						formErrMail = ui.ValidateEmail("邮箱", v)
					}
				}),
			)),
			ui.FormItemInline("角色", false, "", ui.SelectBox(roleSel, roleOpts, formRole, "请选择角色", func(v string) {
				formRole = v
			})),
			ui.FormItemInline("团队", false, "", ui.SelectBox(teamSel, teamOpts, formTeam, "请选择团队", func(v string) {
				formTeam = v
			})),
			ui.FormItemInline("工作年限", false, "", ui.NumberInput("form-exp", formExp, 0, 40, 1, func(v float64) {
				formExp = v
			})),
			ui.FormItemInline("入职日期", false, "", ui.DateBox(dateCal, formDate, func(v string) {
				formDate = v
			})),
			ui.FormItemInline("擅长方向", false, "", ui.Wrap(
				chipCell("Go"), chipCell("渲染"), chipCell("组件库"), chipCell("OpenGL"),
			).St(ui.St().GapOf(8))),
			ui.FormItemInline("简介", false, "", ui.Box(
				ui.Textarea("form-bio", formBio, "写点自我介绍…（支持换行，Enter 换行 / Esc 失焦）", 104, func(v string) {
					formBio = v
				}),
			)),
			ui.FormFooter(
				ui.Btn("重置", ui.BtnSecondary, func() {
					formName, formMail, formBio, formSearch = "极码 AI", "", "", ""
					formExp, formDate, formRole, formTeam = 3, "", "dev", "core"
					formErrName, formErrMail, formSaved = "", "", ""
					ui.RequestRedraw()
				}),
				ui.Btn("保存", ui.BtnPrimary, func() {
					formErrName = ui.ValidateRequired("名称", formName)
					formErrMail = ui.ValidateRequired("邮箱", formMail)
					if formErrMail == "" {
						formErrMail = ui.ValidateEmail("邮箱", formMail)
					}
					if formErrName == "" && formErrMail == "" {
						formSaved = fmt.Sprintf("已保存：%s · %s · %d 年 · %s", formName, roleLabel(formRole), int(formExp), formDate)
					}
					ui.RequestRedraw()
				}),
			),
		),

		ui.Card(
			ui.Title("搜索与标签", "SearchInput 带清除按钮；Chip 可切换选中"),
			ui.FormItem("", false, "", ui.SearchInput("form-search", formSearch, "搜索组件 / 成员…", func(v string) {
				formSearch = v
			})),
			ui.Row(
				ui.Text("多选："+tagSummary()).St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
				ui.Spacer(),
				ui.ChipRemovable("清空选择", func() {
					formTags = map[string]bool{}
					ui.RequestRedraw()
				}),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		),

		ui.Card(
			ui.Title("弹窗", "Alert / Confirm / Prompt / 自定义 Modal / 右侧抽屉"),
			ui.Row(
				ui.Btn("提示 Alert", ui.BtnSecondary, func() {
					dlgAlert = true
					ui.RequestRedraw()
				}),
				ui.Btn("确认 Confirm", ui.BtnSecondary, func() {
					dlgConfirm = true
					ui.RequestRedraw()
				}),
				ui.Btn("输入 Prompt", ui.BtnSecondary, func() {
					dlgPrompt = true
					ui.RequestRedraw()
				}),
				ui.Btn("自定义 Modal", ui.BtnPrimary, func() {
					dlgModal = true
					ui.RequestRedraw()
				}),
				ui.Btn("抽屉 Drawer", ui.BtnSuccess, func() {
					dlgDrawer = true
					ui.RequestRedraw()
				}),
			).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
			ui.KeyValue("当前角色", roleLabel(formRole)),
			ui.KeyValue("入职日期", orDash(formDate)),
			ui.KeyValue("工作年限", strconv.Itoa(int(formExp))+" 年"),
			ui.KeyValue("简介字数", strconv.Itoa(len([]rune(formBio)))),
		),
	).St(ui.St().Padding(26).GapOf(16))
}

// chipCell 单个可选标签。
func chipCell(name string) *ui.Node {
	on := formTags[name]
	return ui.Chip(name, on, func() {
		formTags[name] = !on
	})
}

// tagSummary 已选标签汇总文案。
func tagSummary() string {
	out := ""
	for _, k := range []string{"Go", "渲染", "组件库", "OpenGL"} {
		if formTags[k] {
			if out != "" {
				out += "、"
			}
			out += k
		}
	}
	if out == "" {
		return "未选择"
	}
	return out
}

// roleLabel 角色值转显示名。
func roleLabel(v string) string {
	for _, o := range roleOpts {
		if o.Value == v {
			return o.Label
		}
	}
	return "未选择"
}

func orDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

// FormOverlays 表单案例的弹窗层（放在根 Stack 末尾）。关闭时返回 nil。
func FormOverlays() *ui.Node {
	alert := ui.AlertDlg("dlg-alert", dlgAlert, ui.DlgSuccess, "保存成功",
		"配置已写入本地存储，下次启动自动生效。", func() {
			dlgAlert = false
			ui.RequestRedraw()
		})
	confirm := ui.ConfirmDlg("dlg-confirm", dlgConfirm, ui.DlgWarn, "删除确认",
		"删除后不可恢复，确定继续？", "删除", func() {
			dlgConfirm = false
			ui.RequestRedraw()
		}, func() {
			dlgConfirm = false
			formSaved = "已删除（示例，不影响真实数据）"
			ui.RequestRedraw()
		})
	prompt := ui.PromptDlg("dlg-prompt", dlgPrompt, ui.DlgInfo, "重命名",
		"新名称", dlgPromptID, "请输入名称", "保存", func(v string) {
			dlgPrompt = false
			if v != "" {
				formName = v
				dlgPromptID = v
				formSaved = "已重命名为 " + v
			}
			ui.RequestRedraw()
		}, func() {
			dlgPrompt = false
			ui.RequestRedraw()
		})
	modal := ui.ModalPanel(ui.DialogOpts{
		Key:   "dlg-custom",
		Open:  dlgModal,
		Kind:  ui.DlgInfo,
		Title: "表单快照",
		BodyNode: ui.Column(
			ui.Text("确认以下信息后提交：").St(ui.St().Text(12.5, 400, ui.ColTextDim)),
			ui.Box().St(ui.St().H(6)),
			ui.KeyValue("名称", formName),
			ui.KeyValue("邮箱", orDash(formMail)),
			ui.KeyValue("角色", roleLabel(formRole)),
			ui.KeyValue("年限", strconv.Itoa(int(formExp))+" 年"),
			ui.KeyValue("日期", orDash(formDate)),
			ui.KeyValue("标签", tagSummary()),
		).St(ui.St().GapOf(6)),
		Width:    440,
		Closable: true,
		OnClose: func() {
			dlgModal = false
			ui.RequestRedraw()
		},
		CancelLabel: "再改改",
		OKLabel:     "确认提交",
		OnCancel: func() {
			dlgModal = false
			ui.RequestRedraw()
		},
		OnOK: func() {
			dlgModal = false
			formSaved = "已提交表单快照"
			ui.RequestRedraw()
		},
		Blur: 6,
	})
	drawer := ui.Drawer(ui.DrawerOpts{
		Key:   "dlg-drawer",
		Open:  dlgDrawer,
		Title: "成员详情",
		Width: 400,
		OnClose: func() {
			dlgDrawer = false
			ui.RequestRedraw()
		},
		Body: ui.Column(
			ui.KeyValue("名称", formName),
			ui.KeyValue("角色", roleLabel(formRole)),
			ui.KeyValue("团队", teamLabel(formTeam)),
			ui.KeyValue("年限", strconv.Itoa(int(formExp))+" 年"),
			ui.KeyValue("入职", orDash(formDate)),
			ui.Divider(),
			ui.Text("抽屉内容可以放任意组件，外层自动纵向滚动。").St(ui.St().Text(12, 400, ui.ColTextMuted).WrapOn()),
		).St(ui.St().GapOf(10)),
	})
	layers := make([]*ui.Node, 0, 5)
	for _, n := range []*ui.Node{alert, confirm, prompt, modal, drawer} {
		if n != nil {
			layers = append(layers, n)
		}
	}
	if len(layers) == 0 {
		return nil
	}
	return ui.Stack(layers...).St(ui.St().AbsFill())
}

// teamLabel 团队值转显示名。
func teamLabel(v string) string {
	for _, o := range teamOpts {
		if o.Value == v {
			return o.Label
		}
	}
	return "未选择"
}
