// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"strings"
)

// 多行文本省略：按行数截断 + 可展开。
//
//	ui.EllipsisText("desc", text, 2, 320, ui.EllipsisOptions{})
//
// 展开状态按 key 保存，切换后高度自动变化（布局每帧重算）。

var ellipsisOpen = map[string]bool{}

// EllipsisOptions 省略文本参数。
type EllipsisOptions struct {
	Size   float32
	Weight int32
	Color  string
	LineH  float32
	// Expandable 是否显示「展开 / 收起」。
	Expandable bool
}

// EllipsisText 多行截断文本（默认 2 行 + 展开按钮）。
func EllipsisText(key, text string, lines int, width float32, opts EllipsisOptions) *Node {
	if lines <= 0 {
		lines = 2
	}
	if width <= 0 {
		width = 320
	}
	if opts.Size <= 0 {
		opts.Size = 12.5
	}
	if opts.Weight <= 0 {
		opts.Weight = 400
	}
	if opts.Color == "" {
		opts.Color = ColTextDim
	}
	lineH := opts.LineH
	if lineH <= 0 {
		lineH = opts.Size * 1.55
	}
	open := ellipsisOpen[key]
	// 用字符数估算是否溢出（无需字体即可判断是否需要展开按钮）
	perLine := int(width / (opts.Size * 0.62))
	if perLine < 1 {
		perLine = 1
	}
	overflow := len([]rune(text)) > perLine*lines || strings.Count(text, "\n") >= lines
	show := text
	if !open && overflow {
		rs := []rune(text)
		cut := perLine * lines
		if cut > len(rs) {
			cut = len(rs)
		}
		show = string(rs[:cut])
		// 在最后一个可见行尾部补省略号
		show = strings.TrimRight(show, " \n") + "…"
	}
	n := Text(show).St(St().Text(opts.Size, opts.Weight, opts.Color).WrapOn().Lh(lineH))
	if overflow && opts.Expandable {
		label := "展开"
		if open {
			label = "收起"
		}
		btn := Text(label).St(St().Text(11.5, 600, ColPrimary).PadOnly(4, 0, 0, 0)).Tap(func() {
			ellipsisOpen[key] = !open
			RequestRedraw()
		})
		return Column(n, btn).St(St().GapOf(2).MaxWidth(width))
	}
	return n.St(St().MaxWidth(width))
}
