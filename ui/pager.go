// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "strconv"

// 分页器：数字页码 + 省略号折叠 + 上锁页（超出权限的分页显示锁标且不可点）。
//
//	ui.Pager(ui.PagerOptions{
//	    Key:   "media",
//	    Page:  page,            // 1 起
//	    Total: 9,
//	    Locked: 6,              // 第 7 页起上锁
//	    OnPick: func(p int) { page = p },
//	})

// PagerOptions 分页器参数。
type PagerOptions struct {
	Key    string
	Page   int // 当前页（1 起）
	Total  int // 总页数
	Window int // 当前页左右各显示几页，默认 2
	Locked int // 前 Locked 页之后上锁（0 = 不上锁）
	OnPick func(page int)
}

// Pager 数字分页器。
func Pager(opts PagerOptions) *Node {
	total := opts.Total
	if total < 1 {
		total = 1
	}
	page := opts.Page
	if page < 1 {
		page = 1
	}
	if page > total {
		page = total
	}
	win := opts.Window
	if win <= 0 {
		win = 2
	}

	show := map[int]bool{1: true, total: true, page: true}
	for p := page - win; p <= page+win; p++ {
		if p >= 1 && p <= total {
			show[p] = true
		}
	}

	items := make([]*Node, 0, total)
	// wrapW 逐项累加宽度：Wrap 在 Row 里测量时主轴无上限（可用宽 0），
	// 不显式给宽度会被折成一列，因此按「一层排满」算好总宽。
	var wrapW float32
	addW := func(v float32) {
		if len(items) > 0 {
			wrapW += 6
		}
		wrapW += v
	}
	for p := 1; p <= total; p++ {
		if !show[p] {
			// 连续折叠的页码只画一个省略号
			next := p
			for next <= total && !show[next] {
				next++
			}
			items = append(items, Text("…").St(St().W(22).H(34).
				Text(12.5, 500, ColTextMuted).TAlign(AlignCenter).CenterYOn()))
			addW(22)
			p = next - 1
			continue
		}
		locked := opts.Locked > 0 && p > opts.Locked
		items = append(items, pagerButton(opts.Key, p, p == page, locked, opts.OnPick))
		addW(34)
	}
	return Wrap(items...).St(St().GapOf(6).W(wrapW).H(34)).K("pager:" + opts.Key)
}

// pagerButton 单个页码按钮（当前页渐变高亮；上锁页显示锁标且不可点）。
func pagerButton(key string, p int, active, locked bool, onPick func(int)) *Node {
	const size float32 = 34
	color := ColTextDim
	if active {
		color = "#ffffff"
	} else if locked {
		color = ColPlaceholder
	}
	layers := []*Node{
		Text(strconv.Itoa(p)).St(St().AbsFill().
			Text(12.5, 600, color).TAlign(AlignCenter).CenterYOn()),
	}
	if locked {
		layers = append(layers, pagerLock(color).St(St().AbsTR(4, 4)))
	}
	st := St().W(size).H(size).R(9)
	switch {
	case active:
		st.BgGrad(ColPrimary, ColPrimary2, 135).
			Stroke(1, paintWithAlpha(ColPrimary, 0.85)).
			Shadow(14, 4, 0, paintWithAlpha(ColPrimary, 0.45))
	case locked:
		st.BgC(ColHover).Stroke(1, ColLine)
	default:
		st.BgC(ColBgSoft).Stroke(1, ColStroke)
	}
	btn := Stack(layers...).St(st)
	k := "pager:" + key + ":" + strconv.Itoa(p)
	btn.K(k)
	if locked {
		return TooltipOf(btn.DisabledOn(true), "第 "+strconv.Itoa(p)+" 页需要更高权限")
	}
	if onPick != nil {
		btn.Tap(func() { onPick(p); RequestRedraw() })
	}
	return btn
}

// pagerLock 小锁图标（矢量绘制，不依赖字形）。
func pagerLock(color string) *Node {
	c := C(color)
	return Stack(
		Box().St(St().W(6).H(5).R(1.2).Stroke(1, c).AbsTL(1.5, 5.5)),
		Box().St(St().W(5).H(5).R(2.5).Stroke(1, c).AbsTL(2, 1.5)),
	).St(St().W(9).H(11))
}
