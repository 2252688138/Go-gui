// Go-gui — gallery 页面拆分：设置页（主题 / 缩放 / 清晰度）
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
	"strconv"
)

// uiScaleRatio 当前缩放对应的滑块位置（0~1）。
func uiScaleRatio() float32 {
	return (ui.UIScale() - ui.UIScaleMin) / (ui.UIScaleMax - ui.UIScaleMin)
}

// setUIScaleRatio 滑块位置 → 实际缩放（60%~150%）。
func setUIScaleRatio(v float32) {
	ui.SetUIScale(ui.UIScaleMin + v*(ui.UIScaleMax-ui.UIScaleMin))
}

// ssLabel 超采样倍数的可读文案。
func ssLabel(v float32) string {
	if v <= 1.01 {
		return "标准 1:1"
	}
	return strconv.FormatFloat(float64(v), 'g', 3, 32) + "× 超采样"
}

// setSuperSample 切换清晰度档位（自动 / 标准 / 1.25× / 1.5× / 2×），立即热生效。
func setSuperSample(mode float32) {
	ui.SetSuperSample(mode)
	toastShow("清晰度：" + ssLabel(ui.SuperSample()))
	ui.RequestRedraw()
}

// ---- 案例三：设置页 ----
func settingsCase() *ui.Node {
	return ui.Column(
		pageHead("设置", "开关 / 单选 / 滑块 / 折叠面板 / 输入框"),
		ui.Card(
			ui.Title("通用", ""),
			ui.Row(
				ui.Column(
					ui.Text("自动保存").St(ui.St().Text(13, 500, ui.ColText)),
					ui.Text("编辑内容 2 秒后自动落盘").St(ui.St().Text(11.5, 400, ui.ColTextMuted).PadOnly(3, 0, 0, 0)),
				).Grow(1),
				ui.Switch("set-save", autoSave, func(v bool) { autoSave = v }),
			).AlignTo(ui.AlignCenter),
			ui.Row(
				ui.Column(
					ui.Text("云端同步").St(ui.St().Text(13, 500, ui.ColText)),
					ui.Text("多设备保持一致").St(ui.St().Text(11.5, 400, ui.ColTextMuted).PadOnly(3, 0, 0, 0)),
				).Grow(1),
				ui.Switch("set-sync", syncOn, func(v bool) { syncOn = v }),
			).AlignTo(ui.AlignCenter),
			ui.Divider(),
			ui.Text("主题").St(ui.St().Text(12.5, 500, ui.ColTextDim)),
			ui.Row(
				ui.Radio("th-dark", "深色", Theme == "深色", func() { SetTheme("深色") }),
				ui.Radio("th-light", "浅色", Theme == "浅色", func() { SetTheme("浅色") }),
				ui.Radio("th-auto", "跟随系统", Theme == "跟随系统", func() { SetTheme("跟随系统") }),
			).St(ui.St().GapOf(22)),
			ui.Divider(),
			ui.Row(
				ui.Text("界面缩放").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(70)),
				ui.Slider("set-scale", uiScaleRatio(), setUIScaleRatio),
				ui.Text(strconv.Itoa(int(ui.UIScale()*100+0.5))+"%").St(ui.St().Text(11.5, 500, "#a5b4fc").W(44)),
				ui.Btn("重置", ui.BtnGhost, func() {
					ui.SetUIScale(1)
					ui.RequestRedraw()
				}),
			).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
			ui.Divider(),
			ui.Row(
				ui.Text("清晰度").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(70)),
				ui.Radio("ss-auto", "自动", ui.SuperSampleMode() == ui.SSAuto, func() { setSuperSample(ui.SSAuto) }),
				ui.Radio("ss-off", "标准", ui.SuperSampleMode() == ui.SSOff, func() { setSuperSample(ui.SSOff) }),
				ui.Radio("ss-125", "1.25×", ui.SuperSampleMode() == ui.SS125, func() { setSuperSample(ui.SS125) }),
				ui.Radio("ss-150", "1.5×", ui.SuperSampleMode() == ui.SS150, func() { setSuperSample(ui.SS150) }),
				ui.Radio("ss-200", "2× 超高清", ui.SuperSampleMode() == ui.SS200, func() { setSuperSample(ui.SS200) }),
				ui.Text(ssLabel(ui.SuperSample())).St(ui.St().Text(11.5, 500, "#a5b4fc").W(96)),
			).St(ui.St().GapOf(14)).AlignTo(ui.AlignCenter),
			ui.Text("超高清 = 整帧按更高分辨率渲染再降采样，并对降采样后的边缘做轻度锐化补偿，圆角、描边与文字更锐（自动档在 DPI<150% 时走 2×）。").
				St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
			ui.Divider(),
			ui.Input("set-name", settingsText, "工作区名称", func(v string) { settingsText = v }),
			ui.Accordion("set-acc", "高级选项", "渲染后端 OpenGL 3.3，字形由 opentype 矢量覆盖率抗锯齿，DPI 采用 per-monitor v2 感知。", accOpen, func() { accOpen = !accOpen }),
		),
	).St(ui.St().Padding(26).GapOf(16))
}
