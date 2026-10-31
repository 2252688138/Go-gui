// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"math"
	"strconv"
)

// 表格增强：虚拟滚动、分组折叠、合计行、列宽拖拽。
//
//	ui.TableV(ui.TableVConfig{
//	    Key:     "Users",
//	    Headers: []string{"姓名", "角色", "最近活跃"},
//	    Widths:  []float32{200, 120, 0},   // 0 = 弹性列
//	    RowH:    40, Height: 460,
//	    Count:   len(users),
//	    Row:     func(i int) []*ui.Node { return []*ui.Node{...} },
//	    OnRow:   func(i int) { open(users[i]) },
//	})

// TableGroup 分组区间（Start 起 Count 行）。
type TableGroup struct {
	Label string
	Start int
	Count int
}

// TableVConfig 虚拟表格参数。
type TableVConfig struct {
	Key     string
	Headers []string
	// Widths 列宽；0 表示弹性列（均分剩余宽度）。
	Widths []float32
	RowH   float32
	Height float32
	Count  int
	// Row 构建第 i 行单元格（数量与 Headers 对齐）。
	Row func(i int) []*Node
	// Groups 非空时按分组展示（可折叠）。
	Groups []TableGroup
	// Footer 合计行内容。
	Footer []*Node
	// Resizable 允许拖拽表头分隔条调整列宽（持久化到偏好）。
	Resizable bool
	// SortCol / SortAsc 当前排序列（-1 表示无）；OnSort 点击表头时回调。
	SortCol int
	SortAsc bool
	OnSort  func(col int)
	OnRow   func(i int)
}

var (
	tableWidths  = map[string][]float32{}
	tableCollaps = map[string]map[int]bool{}
	tableDragCol = map[string]int{}
	tableDragW   = map[string]float32{}
	tableDragX   = map[string]float32{}
)

// TableV 虚拟表格（只渲染可见行）。
func TableV(cfg TableVConfig) *Node {
	if cfg.Key == "" {
		cfg.Key = "tablev"
	}
	if cfg.RowH <= 0 {
		cfg.RowH = 40
	}
	if cfg.Height <= 0 {
		cfg.Height = 420
	}
	widths := tableWidths[cfg.Key]
	if len(widths) != len(cfg.Headers) {
		widths = append([]float32{}, cfg.Widths...)
		for len(widths) < len(cfg.Headers) {
			widths = append(widths, 0)
		}
		widths = widths[:len(cfg.Headers)]
		tableWidths[cfg.Key] = widths
	}

	// 条目 = 分组头 + 数据行。这里只建「条目下标 → 条目」的映射，
	// 不再每帧按 Count 预分配并填充整个 entries 切片：旧实现在十万行表格上
	// 每次重建都要 make([]entry, 10000+) 并逐行循环，光这一步就吃掉整帧预算，
	// 页面（尤其滚动时）表现为卡顿。
	type entry struct {
		row     int
		group   int
		isGroup bool
	}
	coll := tableCollaps[cfg.Key]
	if coll == nil {
		coll = map[int]bool{}
		tableCollaps[cfg.Key] = coll
	}
	// groupOf[gi] = 第 gi 个分组头在条目中的下标；groupRows[gi] = 该组数据行数。
	var groupOf []int
	var groupRows []int
	total := cfg.Count
	if len(cfg.Groups) > 0 {
		groupOf = make([]int, len(cfg.Groups))
		groupRows = make([]int, len(cfg.Groups))
		total = 0
		for gi, g := range cfg.Groups {
			n := g.Start + g.Count
			if n > cfg.Count {
				n = cfg.Count
			}
			n -= g.Start
			if n < 0 {
				n = 0
			}
			groupOf[gi] = total
			groupRows[gi] = n
			total++
			if !coll[gi] {
				total += n
			}
		}
	}
	// entryAt 按条目下标取条目；只对可见窗口内调用。
	entryAt := func(idx int) entry {
		if len(cfg.Groups) == 0 {
			return entry{row: idx, group: -1}
		}
		gi := 0
		for gi+1 < len(groupOf) && groupOf[gi+1] <= idx {
			gi++
		}
		if idx == groupOf[gi] {
			return entry{group: gi, isGroup: true, row: cfg.Groups[gi].Start}
		}
		return entry{row: cfg.Groups[gi].Start + (idx - groupOf[gi] - 1), group: gi}
	}

	// 表头（可拖拽列宽）
	header := Row().St(St().GapOf(10).PadXY(14, 9).BgC(ColHeader))
	for i, h := range cfg.Headers {
		col := i
		label := h
		if cfg.OnSort != nil && i == cfg.SortCol {
			if cfg.SortAsc {
				label += " ▲"
			} else {
				label += " ▼"
			}
		}
		cell := Text(label).St(St().Text(11.5, 600, ColTextMuted))
		if i < len(widths) && widths[i] > 0 {
			cell.St(St().W(widths[i]))
		} else {
			cell.Grow(1)
		}
		if cfg.OnSort != nil {
			cell.K(fmt.Sprintf("%s-h-%d", cfg.Key, i)).Tap(func() {
				cfg.OnSort(col)
				RequestRedraw()
			})
		}
		if cfg.Resizable {
			handle := Box().St(St().W(6).Hp(100).AbsTR(-8, 0).BgC("transparent").
				NoFocusRing())
			handle.K(fmt.Sprintf("%s-resize-%d", cfg.Key, i))
			handle.OnPress = func() {
				tableDragCol[cfg.Key] = col
				mx, _ := MousePos()
				tableDragX[cfg.Key] = mx
				if col < len(widths) {
					tableDragW[cfg.Key] = widths[col]
				}
			}
			handle.OnDrag = func(_, _ float32) {
				mx, _ := MousePos()
				if tableDragCol[cfg.Key] != col {
					return
				}
				nw := tableDragW[cfg.Key] + (mx - tableDragX[cfg.Key])
				if nw < 60 {
					nw = 60
				}
				widths[col] = nw
				SetPrefFloat(fmt.Sprintf("table.%s.w%d", cfg.Key, col), nw)
				RequestRedraw()
			}
			handle.OnDragEnd = func() {
				tableDragCol[cfg.Key] = -1
			}
			cell = Stack(cell, handle).St(St().Hp(100))
		}
		header.Add(cell)
	}

	// 行窗口
	step := cfg.RowH
	off := ScrollOffset(cfg.Key)
	first, last := VListVisible(total, off, cfg.Height, step, 4)
	items := make([]*Node, 0, last-first+2)
	if first > 0 {
		items = append(items, Box().St(St().H(float32(first)*step)))
	}
	for i := first; i < last; i++ {
		e := entryAt(i)
		if e.isGroup {
			g := cfg.Groups[e.group]
			arrow := "▾"
			if coll != nil && coll[e.group] {
				arrow = "▸"
			}
			gi := e.group
			row := Row(
				Text(arrow).St(St().Text(11, 600, ColTextMuted)),
				Text(fmt.Sprintf("%s (%d)", g.Label, g.Count)).St(St().Text(12, 600, ColText)),
			).St(St().GapOf(8).PadXY(14, 0).H(step - 4).BgC(ColBgSoft).R(6)).
				AlignTo(AlignCenter)
			row.K(fmt.Sprintf("%s-g-%d", cfg.Key, gi)).Tap(func() {
				if coll == nil {
					coll = map[int]bool{}
					tableCollaps[cfg.Key] = coll
				}
				coll[gi] = !coll[gi]
				RequestRedraw()
			})
			items = append(items, row)
			continue
		}
		cells := []*Node{}
		if cfg.Row != nil {
			cells = cfg.Row(e.row)
		}
		bg := ColRow
		if i%2 == 1 {
			bg = ColRowAlt
		}
		r := Row().St(St().GapOf(10).PadXY(14, 0).H(step).BgC(bg)).AlignTo(AlignCenter)
		for ci := 0; ci < len(cfg.Headers); ci++ {
			var c *Node
			if ci < len(cells) {
				c = cells[ci]
			}
			if c == nil {
				c = Text("—").St(St().Text(12, 400, ColTextMuted))
			}
			if ci < len(widths) && widths[ci] > 0 {
				c.St(St().W(widths[ci]))
			} else {
				c.Grow(1)
			}
			r.Add(c)
		}
		if cfg.OnRow != nil {
			idx := e.row
			r.K(fmt.Sprintf("%s-r-%d", cfg.Key, idx)).Tap(func() {
				cfg.OnRow(idx)
				RequestRedraw()
			})
		}
		items = append(items, r)
	}
	if rest := total - last; rest > 0 {
		items = append(items, Box().St(St().H(float32(rest)*step)))
	}

	body := Scroll(items...).K(cfg.Key).St(St().H(cfg.Height))
	col := Column(Divider(), header, Divider(), body)
	if len(cfg.Footer) > 0 {
		foot := Row(cfg.Footer...).St(St().GapOf(10).PadXY(14, 10).BgC(ColHeader))
		col.Add(Divider(), foot)
	}
	if cfg.Count == 0 {
		col = Column(header, Divider(), EmptyState("", "暂无数据", "", "", nil).
			St(St().H(cfg.Height)))
	}
	return col.St(St().R(12).Stroke(1, ColStroke).Clipped())
}

// TableColWidth 读取某列当前宽度（拖拽后）。
func TableColWidth(key string, col int) float32 {
	if ws, ok := tableWidths[key]; ok && col >= 0 && col < len(ws) {
		return ws[col]
	}
	return 0
}

// SetTableColWidth 设置某列宽度（并持久化）。
func SetTableColWidth(key string, col int, w float32) {
	ws := tableWidths[key]
	if col < 0 || col >= len(ws) {
		return
	}
	ws[col] = w
	SetPrefFloat(fmt.Sprintf("table.%s.w%d", key, col), w)
	RequestRedraw()
}

// ---- 通用拖拽重排 ----

// ReorderListOptions 拖拽重排列表参数。
type ReorderListOptions struct {
	Key      string
	Count    int
	RowH     float32
	Height   float32
	Row      func(i int) *Node
	OnMove   func(from, to int)
	Disabled bool
}

var (
	reorderDrag   = map[string]int{}  // 正在拖动的源行
	reorderTarget = map[string]int{}  // 拖动过程中的目标行
	reorderStale  = map[string]bool{} // 松手后待清理（防拖动行滚出可视区后状态残留）
)

// reorderTargetAt 由鼠标位置算出目标行（布局或鼠标状态缺失时返回 false）。
func reorderTargetAt(key string, count int, rowH float32) (int, bool) {
	if key == "" || count <= 0 || rowH <= 0 {
		return 0, false
	}
	r, ok := LastFrameOf(key)
	if !ok {
		return 0, false
	}
	_, my := MousePos()
	// rel 相对滚动内容顶部：容器 Y 不含滚动偏移，指针位置要补上。
	rel := my - r.Y + ScrollOffset(key)
	t := int(rel / rowH)
	if t < 0 {
		t = 0
	}
	if t >= count {
		t = count - 1
	}
	return t, true
}

// ReorderList 固定行高、整行拖动重排的列表（插入位置高亮 + 松手回调）。
func ReorderList(opts ReorderListOptions) *Node {
	if opts.Key == "" {
		opts.Key = "reorder"
	}
	if opts.RowH <= 0 {
		opts.RowH = 40
	}
	if opts.Height <= 0 {
		opts.Height = 360
	}
	from, dragging := reorderDrag[opts.Key]
	if dragging && !MouseDown() {
		// 松手帧 App 先渲染再分发 OnDragEnd，故延迟一帧清理；
		// 若拖动行已滚出可视区导致 OnDragEnd 丢失，这里兜底清掉残留拖动态。
		if reorderStale[opts.Key] {
			delete(reorderDrag, opts.Key)
			delete(reorderTarget, opts.Key)
			delete(reorderStale, opts.Key)
			dragging = false
			from = -1
		} else {
			reorderStale[opts.Key] = true
			RequestRedraw()
		}
	} else if dragging {
		delete(reorderStale, opts.Key)
	}
	if !dragging {
		from = -1
	}
	to, hasTo := reorderTarget[opts.Key]
	if dragging && !hasTo {
		to = from
	}
	off := ScrollOffset(opts.Key)
	first, last := VListVisible(opts.Count, off, opts.Height, opts.RowH, 3)
	items := make([]*Node, 0, last-first+2)
	if first > 0 {
		items = append(items, Box().St(St().H(float32(first)*opts.RowH)))
	}
	for i := first; i < last; i++ {
		idx := i
		key := opts.Key
		row := opts.Row(i)
		if row == nil {
			row = Box().St(St().H(opts.RowH))
		}
		if from == idx {
			row.St(St().Fade(0.45))
		}
		row.K(fmt.Sprintf("%s-item-%d", key, idx))
		if !opts.Disabled && opts.OnMove != nil {
			row.OnPress = func() {
				reorderDrag[key] = idx
				reorderTarget[key] = idx
				RequestRedraw()
			}
			// OnDrag 必须注册：App 只有在 OnDrag != nil 时才建立 dragKey，
			// 否则松手时的 OnDragEnd 永远不会分发（现象就是拖拽完全没反应）。
			row.OnDrag = func(_, _ float32) {
				if t, ok := reorderTargetAt(key, opts.Count, opts.RowH); ok {
					reorderTarget[key] = t
				}
				RequestRedraw()
			}
			row.OnDragEnd = func() {
				start, ok := reorderDrag[key]
				target, tok := reorderTarget[key]
				delete(reorderDrag, key)
				delete(reorderTarget, key)
				if !ok {
					RequestRedraw()
					return
				}
				if !tok {
					target = start
				}
				if target != start {
					opts.OnMove(start, target)
				}
				RequestRedraw()
			}
		}
		items = append(items, row)
	}
	if rest := opts.Count - last; rest > 0 {
		items = append(items, Box().St(St().H(float32(rest)*opts.RowH)))
	}
	layers := append([]*Node{}, items...)
	// 插入位置指示线：仅在拖动中按目标行绘制（滚动后仍跟随内容坐标系）。
	if dragging && to >= first && to < last {
		layers = append(layers, Box().St(St().Wp(100).H(2).BgC(ColPrimary).
			AbsTL(0, float32(to-first)*opts.RowH).NoFocusRing()))
	}
	return Scroll(layers...).K(opts.Key).St(St().H(opts.Height))
}

// MoveSlice 就地移动元素（from → to），供 OnMove 回调使用。
func MoveSlice[T any](items []T, from, to int) []T {
	if from < 0 || from >= len(items) || to < 0 || to >= len(items) || from == to {
		return items
	}
	v := items[from]
	if from < to {
		copy(items[from:to], items[from+1:to+1])
	} else {
		copy(items[to+1:from+1], items[to:from])
	}
	items[to] = v
	return items
}

// ---- 兼容小工具 ----

// parseFloatDefault 解析浮点，失败返回 def。
func parseFloatDefault(s string, def float32) float32 {
	if v, err := strconv.ParseFloat(s, 32); err == nil {
		return float32(v)
	}
	return def
}

// roundTo 四舍五入到指定小数位。
func roundTo(v float32, dec int) float32 {
	p := math.Pow(10, float64(dec))
	return float32(math.Round(float64(v)*p) / p)
}
