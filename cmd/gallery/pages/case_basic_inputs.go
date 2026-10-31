// Go-gui — gallery 页面拆分：基础组件的输入类控件
// 下拉框 / 多行输入 / 数字 / 日期 / 搜索 / 标签 / 复选，全部来自 ui 包，
// 浮层（下拉、日历）由 main.go 根 Stack 末尾的 SelectLayers / DateLayers 统一渲染。

package pages

import (
	"gogui/ui"
	"strconv"
)

// ---- 案例状态 ----

var (
	basicRole     = "dev"
	basicTeam     = "core"
	basicBio      = "支持换行：第一行\n第二行，Enter 可继续输入，Esc 失焦。"
	basicMail     = ""
	basicMailErr  = ""
	basicNum      = float64(3)
	basicDate     = ""
	basicSearch   = ""
	basicAgree    = true
	basicReadOnly = false
	basicTags     = map[string]bool{"Go": true, "渲染": true, "组件库": false, "OpenGL": false}
)

var (
	basicRoleSel = ui.NewSelect("basic-role")
	basicTeamSel = ui.NewSelect("basic-team")
	basicDateSel = ui.NewDate("basic-date")
)

// basicTagSummary 已选标签汇总文案。
func basicTagSummary() string {
	out := ""
	for _, k := range []string{"Go", "渲染", "组件库", "OpenGL"} {
		if basicTags[k] {
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

// basicTagCell 单个可选标签。
func basicTagCell(name string) *ui.Node {
	on := basicTags[name]
	return ui.Chip(name, on, func() {
		basicTags[name] = !on
		ui.RequestRedraw()
	})
}

// basicInputsSection 基础页的输入类控件区（卡片组合，与其它案例一致的页面内边距）。
func basicInputsSection() *ui.Node {
	return ui.Column(
		basicInputCard(),
		basicMoreInputsCard(),
		basicSwitchesCard(),
	).St(ui.St().Padding(26).GapOf(16))
}

// ---- 开关：默认样式 + 多样式 ----

var (
	basicSwDefault  = true
	basicSwSmall    = false
	basicSwLarge    = true
	basicSwDisabled = true
	basicSwSparkle  = true
	basicSwIcon     = false
	basicSwLabel    = true
	basicSwGrad     = true
	basicSwIconNode = true
	basicSwText     = false
	basicSwNode     = true
	basicSwGlyph    = true
)

// basicSwitchCell 开关示例单元：控件 + 说明文字。
func basicSwitchCell(label string, n *ui.Node) *ui.Node {
	return ui.Column(n, ui.Text(label).St(ui.St().Text(11, 400, ui.ColTextMuted))).
		St(ui.St().GapOf(9)).AlignTo(ui.AlignCenter)
}

// basicSwitchesCard 开关组件族：默认样式 + 尺寸 / 图标 / 文字 / 渐变 / 禁用。
func basicSwitchesCard() *ui.Node {
	return ui.Card(
		ui.Title("开关", "默认样式 + 多样式：尺寸 / 图标胶囊 / 轨道文字 / 渐变发光 / 禁用态"),
		ui.Row(
			basicSwitchCell("默认样式", ui.SwitchDefault("basic-sw-def", basicSwDefault, func(v bool) {
				basicSwDefault = v
			})),
			basicSwitchCell("小号", ui.SwitchSized("basic-sw-sm", ui.SwitchSmall, basicSwSmall, func(v bool) {
				basicSwSmall = v
			})),
			basicSwitchCell("大号", ui.SwitchSized("basic-sw-lg", ui.SwitchLarge, basicSwLarge, func(v bool) {
				basicSwLarge = v
			})),
			basicSwitchCell("禁用（开）", ui.SwitchDisabled("basic-sw-dis", basicSwDisabled)),
		).St(ui.St().GapOf(34)).AlignTo(ui.AlignStart),
		ui.Row(
			basicSwitchCell("✦ 图标胶囊", ui.SwitchSparkle("basic-sw-sparkle", basicSwSparkle, func(v bool) {
				basicSwSparkle = v
			})),
			basicSwitchCell("图标 + 轨道", ui.SwitchIcon("basic-sw-icon", "☾", basicSwIcon, func(v bool) {
				basicSwIcon = v
			})),
			basicSwitchCell("ON / OFF", ui.SwitchLabeled("basic-sw-label", "ON", "OFF", basicSwLabel, func(v bool) {
				basicSwLabel = v
			})),
			basicSwitchCell("渐变发光", ui.SwitchGradient("basic-sw-grad", "#f43f5e", "#fb923c", basicSwGrad, func(v bool) {
				basicSwGrad = v
			})),
		).St(ui.St().GapOf(34)).AlignTo(ui.AlignStart),
		ui.Row(
			basicSwitchCell("自定义图标节点", ui.SwitchEx("basic-sw-iconnode", basicSwIconNode,
				ui.SwitchStyleSparkle().WithIconNode(ui.Dot("#22d3ee", 10)),
				func(v bool) { basicSwIconNode = v })),
			basicSwitchCell("自定义图标字形", ui.SwitchEx("basic-sw-glyph", basicSwGlyph,
				ui.SwitchStyleSparkle().WithIcon("★", "#fbbf24"),
				func(v bool) { basicSwGlyph = v })),
			basicSwitchCell("自定义文案 开 / 关", ui.SwitchEx("basic-sw-text", basicSwText,
				ui.SwitchStyleLabeled("开", "关").WithLabelStyle("#ffffff", "#c7d2fe", 11),
				func(v bool) { basicSwText = v })),
			basicSwitchCell("自定义轨道内容", ui.SwitchEx("basic-sw-node", basicSwNode,
				ui.SwitchStyleDefault().WithLabelNodes(
					ui.Text("✓").St(ui.St().Text(12, 700, "#ffffff")),
					ui.Text("✕").St(ui.St().Text(12, 700, "#94a3b8")),
				).WithLabelWidth(12),
				func(v bool) { basicSwNode = v })),
		).St(ui.St().GapOf(34)).AlignTo(ui.AlignStart),
		ui.Divider(),
		ui.KeyValue("当前状态", "默认 "+onOff(basicSwDefault)+" · 胶囊 "+onOff(basicSwSparkle)+
			" · 文字 "+onOff(basicSwLabel)+" · 渐变 "+onOff(basicSwGrad)),
		ui.KeyValue("自定义", "图标节点 "+onOff(basicSwIconNode)+" · 文字 "+onOff(basicSwText)+
			" · 轨道内容 "+onOff(basicSwNode)),
	)
}

// basicInputCard 下拉框 / 多行输入 / 数字 / 日期。
func basicInputCard() *ui.Node {
	return ui.Card(
		ui.Title("下拉框与多行输入", "Select：点击展开浮层，↑↓ 切换 / Enter 确认 / Esc 关闭；Textarea：换行 + 方向键 + Esc 失焦"),
		ui.FormItemInline("角色", false, "", ui.SelectBox(basicRoleSel, roleOpts, basicRole, "请选择角色", func(v string) {
			basicRole = v
		})),
		ui.FormItemInline("团队", false, "", ui.SelectBox(basicTeamSel, teamOpts, basicTeam, "请选择团队", func(v string) {
			basicTeam = v
		})),
		ui.FormItemInline("工作年限", false, "", ui.NumberInput("basic-num", basicNum, 0, 40, 1, func(v float64) {
			basicNum = v
		})),
		ui.FormItemInline("入职日期", false, "", ui.DateBox(basicDateSel, basicDate, func(v string) {
			basicDate = v
		})),
		ui.FormItemInline("简介", false, "", ui.Textarea("basic-bio", basicBio, "写点介绍…（Enter 换行）", 88, func(v string) {
			basicBio = v
		})),
		ui.Divider(),
		ui.KeyValue("当前选择", roleLabel(basicRole)+" · "+teamLabel(basicTeam)+" · "+
			strconv.Itoa(int(basicNum))+" 年 · "+orDash(basicDate)),
		ui.KeyValue("简介字数", strconv.Itoa(len([]rune(basicBio)))),
	)
}

// basicMoreInputsCard 搜索 / 校验 / 标签 / 复选。
func basicMoreInputsCard() *ui.Node {
	return ui.Card(
		ui.Title("搜索 · 校验 · 标签 · 复选", "SearchInput 带清除按钮；FormItem 内联错误提示；Chip 多选；Checkbox 复选"),
		ui.FormItemInline("邮箱", true, basicMailErr, ui.Input("basic-mail", basicMail, "name@example.com", func(v string) {
			basicMail = v
			if basicMailErr != "" {
				basicMailErr = ui.ValidateEmail("邮箱", v)
			}
		})),
		ui.FormItemInline("测试校验", false, "", ui.Row(
			ui.Btn("校验邮箱", ui.BtnSecondary, func() {
				basicMailErr = ui.ValidateEmail("邮箱", basicMail)
				ui.RequestRedraw()
			}),
			ui.Text("空值会提示必填，格式错误会提示格式不正确").St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter)),
		ui.FormItemInline("搜索", false, "", ui.SearchInput("basic-search", basicSearch, "搜索组件 / 成员…", func(v string) {
			basicSearch = v
		})),
		ui.Row(
			ui.Text("擅长方向").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(ui.LabelW)),
			basicTagCell("Go"),
			basicTagCell("渲染"),
			basicTagCell("组件库"),
			basicTagCell("OpenGL"),
			ui.Spacer(),
			ui.Text("已选："+basicTagSummary()).St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(8)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Checkbox("basic-agree", "启用本地缓存", basicAgree, func(v bool) { basicAgree = v }),
			ui.Checkbox("basic-readonly", "只读模式", basicReadOnly, func(v bool) { basicReadOnly = v }),
		).St(ui.St().GapOf(26)).AlignTo(ui.AlignCenter),
		ui.Divider(),
		ui.KeyValue("搜索关键词", orDash(basicSearch)),
		ui.KeyValue("邮箱校验", orDash(basicMailErr)),
		ui.KeyValue("复选状态", "本地缓存 "+onOff(basicAgree)+" / 只读 "+onOff(basicReadOnly)),
	)
}

// onOff 布尔值转「开 / 关」文案。
func onOff(v bool) string {
	if v {
		return "开"
	}
	return "关"
}
