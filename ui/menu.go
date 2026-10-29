package ui

import (
	"strconv"
	"time"
)

// 菜单组件：标题栏菜单栏（MenuBar）+ 下拉浮层（MenuLayer），支持多级子菜单。
//
// 用法（浮层必须放在根 Stack 里，否则会被后续兄弟节点覆盖）：
//
//	var mb = ui.NewMenuBar()
//	var menus = []ui.MenuDef{{Label: "文件", Items: []ui.MenuItem{ ... }}}
//
//	// 每帧：
//	col := ui.Column(ui.Row(ui.MenuBar(mb, menus), ...), body)
//	root := ui.Stack(col, ui.MenuLayer(mb, menus, anchorX, anchorY))
//
// 菜单项 Children 非空即为子菜单（鼠标悬停展开，可继续嵌套）。

// MenuItem 菜单项。
type MenuItem struct {
	Label     string
	Hint      string // 右侧快捷键/说明
	Icon      string // 左侧符号（可选）
	OnPick    func()
	Children  []MenuItem // 非空 = 子菜单
	Separator bool       // 分隔线
	Disabled  bool
	Checked   bool
}

// MenuDef 顶级菜单（标题 + 下拉项）。
type MenuDef struct {
	Label string
	Items []MenuItem
}

// 菜单几何（浮层定位依赖固定行高，保证与绘制一致）。
const (
	menuRowH = 30
	menuSepH = 9
	menuPad  = 6
	menuGap  = 2
	menuMinW = 150
	menuMaxW = 280
)

// menuAutoHide 鼠标离开菜单后自动隐藏的默认等待时间。
const menuAutoHide = 3 * time.Second

// MenuBarState 菜单栏状态（跨帧保持；由调用方持有）。
type MenuBarState struct {
	Open int   // 打开的顶级菜单索引，-1 = 关闭
	Sub  []int // 子菜单展开路径（逐级的项索引）
	// ItemX 由 MenuBar 计算：各顶级菜单按钮相对菜单栏左边缘的 x 偏移。
	ItemX []float32
	// AutoHide 鼠标离开菜单后自动隐藏的等待时间（0 = 默认 3s，<0 = 不自动隐藏）。
	AutoHide time.Duration

	// lastHover 最近一次悬停在菜单（菜单栏或浮层）内的时间；
	// hideTimer 到点请求重绘，让帧内检查能关掉菜单（期间不持续刷新，省 CPU）。
	lastHover time.Time
	hideTimer *time.Timer
	// inside 指针当前是否停在菜单栏/浮层内（停住不动时不应超时隐藏）
	inside bool
}

// NewMenuBar 创建菜单栏状态。
func NewMenuBar() *MenuBarState {
	st := &MenuBarState{Open: -1}
	registerMenuState(st)
	return st
}

// registerMenuState 登记菜单栏状态（ESC 统一关闭用）。
// 每次重算页面都会新建菜单栏，重复登记会让列表无限增长，
// 于是「关掉所有菜单」的遍历越来越慢、点菜单越来越卡；同一状态只登记一次。
func registerMenuState(st *MenuBarState) {
	for _, s := range menuStates {
		if s == st {
			return
		}
	}
	menuStates = append(menuStates, st)
}

// menuStates 已创建的菜单栏状态（ESC 统一关闭用）。
var menuStates []*MenuBarState

// closeOpenMenus 关闭所有展开的菜单，返回是否真的关掉了某个菜单。
func closeOpenMenus() bool {
	closed := false
	for _, st := range menuStates {
		if st.IsOpen() {
			st.Close()
			closed = true
		}
	}
	return closed
}

// autoHideAfter 自动隐藏等待时间（默认 3s；负数表示不自动隐藏）。
func (st *MenuBarState) autoHideAfter() time.Duration {
	if st.AutoHide < 0 {
		return -1
	}
	if st.AutoHide == 0 {
		return menuAutoHide
	}
	return st.AutoHide
}

// touch 记录最近一次悬停在菜单内的时间（新子菜单展开也算活跃）。
func (st *MenuBarState) touch() {
	if st == nil || st.Open < 0 {
		return
	}
	st.lastHover = time.Now()
}

// enter 指针进入菜单栏/浮层。
func (st *MenuBarState) enter() {
	if st == nil || st.Open < 0 {
		return
	}
	st.inside = true
	st.touch()
}

// leave 指针离开菜单栏/浮层：从离开时刻起计时，到时隐藏。
func (st *MenuBarState) leave() {
	if st == nil || st.Open < 0 {
		return
	}
	st.inside = false
	st.lastHover = time.Now()
	st.scheduleAutoHide()
}

// scheduleAutoHide 在预计到点时请求一次重绘；期间不持续刷新。
func (st *MenuBarState) scheduleAutoHide() {
	if st == nil || st.Open < 0 {
		return
	}
	d := st.autoHideAfter()
	if d < 0 {
		return
	}
	wait := d - time.Since(st.lastHover)
	if wait < 0 {
		wait = 0
	}
	if st.hideTimer != nil {
		st.hideTimer.Stop()
	}
	st.hideTimer = time.AfterFunc(wait+10*time.Millisecond, RequestRedraw)
}

// sweepAutoHide 帧内检查：超时未悬停则关掉菜单（返回是否已关闭）。
// 指针仍停在菜单内时不隐藏（停住不动不应被当成“已移开”）。
func (st *MenuBarState) sweepAutoHide() bool {
	if st == nil || st.Open < 0 {
		return false
	}
	d := st.autoHideAfter()
	if d < 0 {
		return false
	}
	if st.inside {
		return false
	}
	if st.lastHover.IsZero() {
		st.lastHover = time.Now()
		st.scheduleAutoHide()
		return false
	}
	if time.Since(st.lastHover) < d {
		return false
	}
	st.Close()
	return true
}

// Close 关闭菜单（含所有子菜单）。
func (st *MenuBarState) Close() {
	if st == nil {
		return
	}
	st.Open = -1
	st.Sub = nil
	st.inside = false
	st.lastHover = time.Time{}
	if st.hideTimer != nil {
		st.hideTimer.Stop()
		st.hideTimer = nil
	}
}

// IsOpen 当前是否有菜单展开。
func (st *MenuBarState) IsOpen() bool { return st != nil && st.Open >= 0 }

// openMenuAt 打开第 idx 个顶级菜单（已有菜单会立即切换，连同子菜单路径一起重置）。
func openMenuAt(st *MenuBarState, idx int) {
	if st == nil {
		return
	}
	st.Open = idx
	st.Sub = nil
	st.inside = true
	st.touch()
	if st.hideTimer != nil {
		st.hideTimer.Stop()
		st.hideTimer = nil
	}
}

// MenuBar 渲染标题栏里的横向菜单按钮行。
// 下拉浮层由 MenuLayer 渲染（anchorX 传菜单栏左边缘的窗口坐标）。
func MenuBar(st *MenuBarState, menus []MenuDef) *Node {
	if st == nil {
		return nil
	}
	st.sweepAutoHide()
	bar := Row().St(St().GapOf(2)).AlignTo(AlignCenter)
	x := float32(0)
	for i, m := range menus {
		idx := i
		open := st.Open == idx
		color := ColTextDim
		if open {
			color = ColText
		}
		label := Text(m.Label).St(St().Text(12.5, 500, color).PadXY(10, 6))
		if open {
			label.St(St().BgC(ColBgSoft).R(6))
		} else {
			label.St(St().R(6))
		}
		w := TextWidth(m.Label, 12.5, 500) + 20
		if len(st.ItemX) <= idx {
			st.ItemX = append(st.ItemX, make([]float32, idx-len(st.ItemX)+1)...)
		}
		st.ItemX[idx] = x
		x += w + 2
		item := label.K("menubar-" + m.Label)
		item.OnHoverEvt(func(h bool) {
			if !h {
				st.leave()
				return
			}
			st.enter()
			// 已有菜单展开时，鼠标划过其它顶级菜单直接切换
			if st.Open >= 0 && st.Open != idx {
				openMenuAt(st, idx)
				RequestRedraw()
			}
		})
		item.Tap(func() {
			if st.Open == idx {
				st.Close()
			} else {
				// 已有菜单展开时直接切到被点的菜单（旧菜单立即隐藏），不再要求先关闭
				openMenuAt(st, idx)
			}
			RequestRedraw()
		})
		bar.Add(item)
	}
	return bar
}

// MenuLayer 下拉菜单浮层（含多级子菜单）。
// anchorX / anchorY 为菜单栏左边缘与菜单栏底部的窗口坐标。
// 必须放在根 Stack 里（最后几层），否则会被内容覆盖。
func MenuLayer(st *MenuBarState, menus []MenuDef, anchorX, anchorY float32) *Node {
	if st == nil || st.Open < 0 {
		return nil
	}
	// 鼠标离开菜单超过 AutoHide（默认 3s）后自动收起
	st.sweepAutoHide()
	if st.Open < 0 || st.Open >= len(menus) {
		return nil
	}
	items := menus[st.Open].Items
	if len(items) == 0 {
		return nil
	}
	if len(st.Sub) > 0 {
		// 切换顶级菜单后旧路径失效时自动丢弃
		if st.Sub[0] >= len(items) {
			st.Sub = nil
		}
	}
	w := menuWidth(items)
	ox := anchorX
	if len(st.ItemX) > st.Open {
		ox += st.ItemX[st.Open]
	}
	if ox+w+8 > ViewportW() {
		ox = ViewportW() - w - 8
	}
	if ox < 4 {
		ox = 4
	}
	// 浮层只铺「菜单栏下方」这块区域。
	// 以前整层用 AbsFill 盖满窗口：命中测试在菜单栏上会先停在这一层（它没有任何回调
	// 也没有交互祖先），永远落不到菜单按钮，表现为「已有菜单展开时必须先关掉它
	// 才能点别的顶级菜单」。收窄到菜单栏下方后，菜单栏重新可命中（悬停/点击都能切换）。
	regionH := viewportH - anchorY
	if regionH < 0 {
		regionH = 0
	}
	panel := menuPanelNode(st, items, w, 0, nil)
	panel.St(St().AbsTL(ox, 0))

	// 点击菜单外部关闭（只铺菜单下方区域，菜单栏仍可悬停/点击切换）
	catcher := Box().St(St().AbsTL(0, 0).W(ViewportW()).H(regionH))
	catcher.Tap(func() {
		st.Close()
		RequestRedraw()
	})
	return Stack(catcher, panel).St(St().AbsTL(0, anchorY).W(ViewportW()).H(regionH))
}

// MenuPanel 浮层菜单（放在 Stack 中，用绝对坐标定位）。支持子菜单、分隔线、勾选与禁用。
func MenuPanel(x, y float32, items []MenuItem, width float32) *Node {
	if width <= 0 {
		width = 168
	}
	if panelMenuState.X != x || panelMenuState.Y != y {
		panelMenuState.Close()
	}
	panelMenuState.X, panelMenuState.Y = x, y
	return menuPanelNode(&panelMenuState.MenuBarState, items, width, 0, nil).St(St().AbsTL(x, y))
}

// panelMenuState MenuPanel 的内部状态（同一时间只保留一个浮层的展开路径）。
var panelMenuState = &panelState{MenuBarState: MenuBarState{Open: 0}}

type panelState struct {
	MenuBarState
	X, Y float32
}

// menuPanelNode 构建一块面板；子菜单面板作为绝对定位子节点叠在同一 Stack 内，
// 因此子菜单坐标只需相对本面板计算。
func menuPanelNode(st *MenuBarState, items []MenuItem, width float32, depth int, path []int) *Node {
	col := Column().St(St().W(width).Padding(menuPad).GapOf(menuGap).R(12).
		BgC(ColPanel).Stroke(1, ColStroke).Shadow(28, 12, 0, "rgba(0,0,0,0.55)"))
	// 面板自身（行间空隙 / 内边距）也算“在菜单内”，避免移向内边距时误计时
	col.K("menupanel-" + strconv.Itoa(depth)).OnHoverEvt(func(h bool) {
		if h {
			st.enter()
			return
		}
		st.leave()
	})
	holder := Stack(col).St(St().W(width))
	// rowTop[i] 第 i 项相对面板顶部的偏移（含 padding 与分隔线）
	rowTop := make([]float32, len(items))
	cy := float32(menuPad)
	for i, it := range items {
		rowTop[i] = cy
		if it.Separator {
			line := Box(Box().St(St().Wp(100).H(1).BgC(ColLine).CenterYOn())).
				St(St().H(menuSepH))
			col.Add(line)
			cy += menuSepH + menuGap
			continue
		}
		item := it
		idx := i
		curPath := append(append([]int{}, path...), idx)
		active := len(st.Sub) > depth && st.Sub[depth] == idx
		row := menuRowNode(item, width-menuPad*2, active)
		row.K("menu-" + item.Label)
		if len(item.Children) > 0 {
			row.OnHoverEvt(func(h bool) {
				if !h {
					st.leave()
					return
				}
				st.enter()
				if len(st.Sub) <= depth || st.Sub[depth] != idx {
					st.Sub = append(st.Sub[:depth], curPath...)
					RequestRedraw()
				}
			})
			row.Tap(func() {
				st.enter()
				st.Sub = append(st.Sub[:depth], curPath...)
				RequestRedraw()
			})
		} else {
			row.OnHoverEvt(func(h bool) {
				if !h {
					st.leave()
					return
				}
				st.enter()
				// 悬停到叶子项时收起更深层子菜单
				if len(st.Sub) > depth {
					st.Sub = st.Sub[:depth]
					RequestRedraw()
				}
			})
			row.Tap(func() {
				if item.Disabled {
					return
				}
				st.Close()
				if item.OnPick != nil {
					item.OnPick()
				}
				RequestRedraw()
			})
		}
		col.Add(row)
		cy += menuRowH + menuGap
	}
	// 展开的子菜单
	if len(st.Sub) > depth {
		si := st.Sub[depth]
		if si >= 0 && si < len(items) && len(items[si].Children) > 0 {
			subW := menuWidth(items[si].Children)
			sub := menuPanelNode(st, items[si].Children, subW, depth+1, st.Sub[:depth+1])
			sub.St(St().AbsTL(width+4, rowTop[si]-menuPad))
			holder.Add(sub)
		}
	}
	return holder
}

// menuRowNode 单个菜单项。
func menuRowNode(it MenuItem, width float32, active bool) *Node {
	color := ColTextDim
	if it.Disabled {
		color = ColPlaceholder
	} else if active {
		color = ColText
	}
	label := it.Label
	if it.Checked {
		label = "✓ " + label
	}
	kids := []*Node{}
	// 子菜单项的 Icon 旧写法是「▸」，它是子菜单标识而不是行首图标：
	// 这里直接忽略，统一在右侧用矢量三角绘制，避免一个菜单项出现两个箭头。
	if it.Icon != "" && it.Icon != "▸" && it.Icon != "›" {
		kids = append(kids, Text(it.Icon).St(St().Text(12, 500, color).W(18)))
	}
	kids = append(kids, Text(label).St(St().Text(12.5, 500, color)))
	kids = append(kids, Spacer())
	if it.Hint != "" {
		kids = append(kids, Text(it.Hint).St(St().Text(11, 400, ColTextMuted)))
	}
	if len(it.Children) > 0 {
		// 矢量三角（旧实现是「›」字符：字重/基线随字体漂移，看着不像三角）
		tint := ColTextMuted
		if active {
			tint = ColTextDim
		}
		kids = append(kids, Arrow(tint))
	}
	row := Row(kids...).St(St().W(width).H(menuRowH).PadXY(8, 0).GapOf(8).R(7)).
		AlignTo(AlignCenter)
	if active {
		row.St(St().BgC("rgba(99,102,241,0.18)"))
	}
	return row
}

// menuWidth 依据文本估算面板宽度。
func menuWidth(items []MenuItem) float32 {
	w := float32(menuMinW)
	for _, it := range items {
		if it.Separator {
			continue
		}
		iw := float32(16) // 左右内边距 + 行内边距
		if it.Icon != "" && it.Icon != "▸" && it.Icon != "›" {
			iw += 18 + 8
		}
		iw += TextWidth(it.Label, 12.5, 500)
		if it.Checked {
			iw += TextWidth("✓ ", 12.5, 500)
		}
		if it.Hint != "" {
			iw += TextWidth(it.Hint, 11, 400) + 24
		}
		if len(it.Children) > 0 {
			iw += 18
		}
		if iw > w {
			w = iw
		}
	}
	if w > menuMaxW {
		w = menuMaxW
	}
	return w
}

// WindowButtons 自绘标题栏右侧的最小化 / 最大化 / 关闭按钮。
func WindowButtons() *Node {
	btn := func(key, glyph string) *Node {
		return Text(glyph).St(St().W(32).H(26).R(6).Text(12, 500, ColTextDim).
			TAlign(AlignCenter).CenterYOn()).K(key)
	}
	min := btn("win-min", "–").Tap(func() {
		if a := Current(); a != nil {
			a.Minimize()
		}
	})
	max := btn("win-max", "□").Tap(func() {
		if a := Current(); a != nil {
			a.ToggleMaximize()
		}
	})
	cls := Text("×").St(St().W(32).H(26).R(6).Text(14, 500, ColTextDim).
		TAlign(AlignCenter).CenterYOn()).K("win-close").Tap(func() {
		if a := Current(); a != nil {
			a.Close()
		}
	})
	return Row(min, max, cls).St(St().GapOf(2)).AlignTo(AlignCenter)
}
