// Go-gui — gallery 页面拆分：聊天
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
)

// ---- 案例四：聊天 ----
func chatCase() *ui.Node {
	items := make([]*ui.Node, 0, len(messages))
	for _, m := range messages {
		items = append(items, bubble(m))
	}
	items = append(items, ui.Spacer())

	return ui.Column(
		ui.Row(
			ui.Avatar("极", "#6366f1"),
			ui.Column(
				ui.Text("极码 AI").St(ui.St().Text(13.5, 600, ui.ColText)),
				ui.Row(ui.Dot("#34d399", 7), ui.Text("在线").St(ui.St().Text(11, 400, "#6ee7b7"))).St(ui.St().GapOf(6)),
			).Grow(1).St(ui.St().GapOf(3)),
			ui.Tag("示例会话", "#a5b4fc"),
		).St(ui.St().GapOf(12).Padding(16).BgC(ui.ColBgSoft).
			Stroke(1, ui.ColStroke)).AlignTo(ui.AlignCenter),

		ui.Scroll(append(items, ui.Box().St(ui.St().H(1)))...).St(ui.St().GrowTo(1).Padding(18)).K("scroll-chat"),

		ui.Row(
			ui.Box(
				ui.Input("chat-input", msgInput, "输入消息后回车无效，点发送…", func(v string) { msgInput = v }),
			).Grow(1),
			ui.Button("发送", func() {
				if msgInput == "" {
					return
				}
				messages = append(messages, chatMsg{true, msgInput})
				msgInput = ""
				toastShow("已发送")
				ui.RequestRedraw()
			}).St(ui.St().Text(13, 600, "#ffffff").PadXY(18, 9).R(10).
				BgGrad("#6366f1", "#8b5cf6", 90)),
		).St(ui.St().GapOf(12).Padding(16)).AlignTo(ui.AlignCenter),
	).St(ui.St().Hp(100))
}

func bubble(m chatMsg) *ui.Node {
	st := ui.St().Text(13, 400, "#e6ecff").Padding(12).R(14).WrapOn()
	var row *ui.Node
	if m.me {
		st.BgGrad("#4f46e5", "#6366f1", 90)
		row = ui.Row(
			ui.Spacer(),
			ui.Text(m.text).St(st),
			ui.Avatar("我", "#0ea5e9"),
		)
	} else {
		st.BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke)
		row = ui.Row(
			ui.Avatar("极", "#6366f1"),
			ui.Text(m.text).St(st),
			ui.Spacer(),
		)
	}
	return row.St(ui.St().GapOf(10).PadOnly(0, 0, 10, 0)).AlignTo(ui.AlignStart)
}
