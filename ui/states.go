// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "gogui/internal/gpu"

// 加载 / 空 / 错误三态与结果页。
//
//	ui.AsyncSection(loading, err, len(items) == 0, reload, func() *ui.Node {
//	    return ui.VList("items", len(items), 44, 400, row)
//	})
//
//	ui.ResultPage("success", "支付成功", "订单已提交", ui.Btn("返回首页", ui.BtnPrimary, back))

// SkeletonBlock 骨架块（shimmer 流光）。
func SkeletonBlock(w, h float32) *Node { return Skeleton(w, h) }

// SkeletonText 多行文本骨架（最后一行 60% 宽）。
func SkeletonText(lines int, width float32) *Node {
	if lines <= 0 {
		lines = 3
	}
	if width <= 0 {
		width = 240
	}
	rows := make([]*Node, 0, lines)
	for i := 0; i < lines; i++ {
		w := width
		if i == lines-1 {
			w = width * 0.6
		}
		rows = append(rows, Skeleton(w, 11))
	}
	return Column(rows...).St(St().GapOf(9))
}

// SkeletonCard 卡片骨架（头像 + 两行文字）。
func SkeletonCard(width float32) *Node {
	if width <= 0 {
		width = 280
	}
	return Row(
		Skeleton(40, 40),
		Column(Skeleton(width*0.5, 13), Skeleton(width*0.7, 11)).St(St().GapOf(8)).Grow(1),
	).St(St().GapOf(12).Padding(16).R(12).BgC(ColBgSoft).Stroke(1, ColLine)).
		AlignTo(AlignCenter)
}

// LoadingOverlay 区域级加载遮罩：覆盖父容器、拦截点击、居中 spinner + 文案。
//
// 放在需要遮罩的容器（Stack）的最后一层；show=false 返回 nil。
func LoadingOverlay(show bool, label string) *Node {
	if !show {
		return nil
	}
	sp := Spinner(26, ColPrimary)
	content := Column(sp).St(St().GapOf(10)).AlignTo(AlignCenter)
	if label != "" {
		content.Add(Text(label).St(St().Text(12, 500, ColTextMuted)))
	}
	veil := Box().St(St().AbsFill().BgC(paintWithAlpha(ColBgDeep, 0.62))).Tap(func() {}).
		K("loading-veil")
	panel := content.St(St().AbsCenter())
	return Stack(veil, panel).St(St().AbsFill().AnimSlowOn())
}

// EmptyState 空状态：插画位（图标字形）+ 标题 + 说明 + 可选操作按钮。
func EmptyState(icon, title, sub, actionLabel string, onAction func()) *Node {
	if title == "" {
		title = "暂无数据"
	}
	if icon == "" {
		icon = "◇"
	}
	art := Stack(
		Box().St(St().W(64).H(64).R(20).BgC(ColHover).Stroke(1, ColStroke)),
		Text(icon).St(St().AbsFill().Text(24, 500, ColPlaceholder).CenterYOn().TAlign(AlignCenter)),
	).St(St().W(64).H(64))
	col := Column(art, Text(title).St(St().Text(13.5, 600, ColText).PadOnly(14, 0, 0, 0)))
	if sub != "" {
		col.Add(Text(sub).St(St().Text(11.5, 400, ColTextMuted).WrapOn().TAlign(AlignCenter)))
	}
	if actionLabel != "" {
		col.Add(Box().St(St().MarOnly(14, 0, 0, 0)))
		col.Add(Btn(actionLabel, BtnPrimary, onAction))
	}
	return col.St(St().GapOf(6).Padding(28)).AlignTo(AlignCenter)
}

// ResultKind 结果页类型。
func resultVisual(kind string) (glyph, color, title string) {
	switch kind {
	case "success":
		return "✓", ColSuccess, "操作成功"
	case "warn":
		return "!", ColWarn, "需要注意"
	case "error", "fail":
		return "×", ColDanger, "操作失败"
	case "403":
		return "⚿", ColWarn, "无权访问"
	case "404":
		return "?", ColInfo, "页面不存在"
	case "500":
		return "!", ColDanger, "服务器错误"
	}
	return "i", ColInfo, "提示"
}

// ResultPage 整页结果反馈（成功 / 失败 / 警示 / 404 / 500）。
func ResultPage(kind, title, sub string, actions ...*Node) *Node {
	glyph, color, defTitle := resultVisual(kind)
	if title == "" {
		title = defTitle
	}
	badge := Stack(
		Box().St(St().AbsFill().R(36).BgC(paintWithAlpha(color, 0.14)).Stroke(1, paintWithAlpha(color, 0.55))),
		Text(glyph).St(St().AbsFill().Text(30, 700, color).CenterYOn().TAlign(AlignCenter)),
	).St(St().W(72).H(72))
	col := Column(badge)
	col.Add(Text(title).St(St().Text(19, 700, ColText).PadOnly(18, 0, 0, 0)))
	if sub != "" {
		col.Add(Text(sub).St(St().Text(12.5, 400, ColTextMuted).WrapOn().TAlign(AlignCenter).MaxWidth(420)))
	}
	if len(actions) > 0 {
		col.Add(Row(actions...).St(St().GapOf(10).MarOnly(18, 0, 0, 0)).AlignTo(AlignCenter))
	}
	return col.St(St().GapOf(10).Padding(40)).AlignTo(AlignCenter)
}

// AsyncSection 四态容器：加载中 / 错误 / 空 / 正常内容。
// body 只在正常态调用，避免为占位状态构建重列表。
func AsyncSection(loading bool, errMsg string, empty bool, onRetry func(), body func() *Node) *Node {
	switch {
	case loading:
		return Column(
			SkeletonText(3, 320),
			Row(Spinner(16, ColPrimary), Text("加载中…").St(St().Text(12, 400, ColTextMuted))).
				St(St().GapOf(8).MarOnly(14, 0, 0, 0)).AlignTo(AlignCenter),
		).St(St().GapOf(8).Padding(20))
	case errMsg != "":
		col := Column(
			Text("加载失败").St(St().Text(14, 600, ColDanger)),
			Text(errMsg).St(St().Text(12, 400, ColTextMuted).WrapOn()),
		).St(St().GapOf(8).Padding(24)).AlignTo(AlignCenter)
		if onRetry != nil {
			col.Add(Btn("重试", BtnSecondary, onRetry))
		}
		return col
	case empty:
		return EmptyState("", "暂无数据", "换个筛选条件试试", "", nil)
	}
	if body == nil {
		return nil
	}
	return body()
}

// paintWithAlpha 返回指定颜色的半透明画笔。
func paintWithAlpha(c string, a float32) gpu.Paint {
	p := C(c)
	p.A = a
	return p
}
