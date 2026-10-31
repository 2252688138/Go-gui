// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 文件树的行构建（从 tree.go 拆出，便于阅读）。

// buildTreeRow 构建一行：缩进 → 展开箭头 → 图标 → 名称 → 右侧标签。
func buildTreeRow(opts TreeOptions, n TreeNode, depth int, open bool, sel string) *Node {
	hasKids := len(n.Children) > 0
	cells := make([]*Node, 0, 5)
	if depth > 0 {
		cells = append(cells, Box().St(St().W(float32(depth)*opts.Indent).H(1)))
	}
	// 展开箭头（叶子节点留同宽占位，保证图标纵向对齐）
	if hasKids {
		arrow := "▸"
		if open {
			arrow = "▾"
		}
		cells = append(cells, Text(arrow).St(St().W(14).H(opts.RowH).
			Text(9, 500, ColTextMuted).CenterYOn().TAlign(AlignCenter)))
	} else {
		cells = append(cells, Box().St(St().W(14)))
	}
	cells = append(cells, treeIcon(n.Kind, open))
	cells = append(cells, Text(n.Label).St(St().Text(12.5, 500, ColTextDim)).Grow(1))
	if n.Tag != "" {
		c := n.TagColor
		if c == "" {
			c = ColTextMuted
		}
		cells = append(cells, Text(n.Tag).St(St().Text(10.5, 500, c).PadXY(6, 2).R(5).BgC(ColHover)))
	}

	row := Row(cells...).St(St().GapOf(6).PadOnly(0, 8, 0, 8).H(opts.RowH).R(8)).
		AlignTo(AlignCenter)
	key := opts.Key + ":" + n.Key
	row.K(key)
	switch {
	case n.Disabled:
		row.St(St().Fade(0.45))
	case n.Key == sel:
		row.St(St().BgC(paintWithAlpha(ColPrimary, 0.16)).
			Stroke(1, paintWithAlpha(ColPrimary, 0.36)))
		row.Add(Box().St(St().W(2).H(opts.RowH - 12).R(1).BgC(ColPrimary).
			AbsTL(0, 6)))
	case IsHovered(key):
		row.St(St().BgC(ColHover))
	}
	if !n.Disabled {
		row.Tap(func() {
			if hasKids {
				m := treeOpen[opts.Key]
				if m == nil {
					m = map[string]bool{}
					treeOpen[opts.Key] = m
				}
				m[n.Key] = !open
				if opts.OnToggle != nil {
					opts.OnToggle(n, !open)
				}
			}
			treeSel[opts.Key] = n.Key
			if opts.OnPick != nil {
				opts.OnPick(n)
			}
			RequestRedraw()
		})
	}
	return row
}

// treeIcon 节点图标：文件夹 / 文件 / 链接圆点（全部矢量色块，不依赖字形）。
func treeIcon(k TreeKind, open bool) *Node {
	switch k {
	case TreeFile:
		return Stack(
			Box().St(St().W(12).H(14).R(3).BgC(ColHover).Stroke(1, ColLine)),
			Box().St(St().W(5).H(1.4).R(1).BgC(ColPlaceholder).AbsTL(3.5, 5.5)),
			Box().St(St().W(5).H(1.4).R(1).BgC(ColPlaceholder).AbsTL(3.5, 8.5)),
		).St(St().W(16).H(14))
	case TreeLink:
		return Stack(Dot(ColInfo, 7)).St(St().W(16).H(14))
	}
	blue := "#3f7ff0"
	if open {
		blue = "#5b9dff"
	}
	return Stack(
		Box().St(St().W(16).H(11).R(3).BgGrad(blue, "#8ab6ff", 120).AbsTL(0, 3)),
		Box().St(St().W(7).H(4).Rx(2, 2, 0, 0).BgC(blue).AbsTL(0, 0)),
	).St(St().W(16).H(14))
}
