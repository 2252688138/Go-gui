// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"strings"
)

// Dock 停靠面板系统：左右下三侧面板 + 中间主区，面板可折叠、宽度可拖、布局可持久化。
//
//	st := ui.NewDock("ide")
//	st.Left = []string{"files", "git"}
//	st.Bottom = []string{"terminal", "output"}
//	ui.DockLayout(st, editor, map[string]ui.DockPanel{
//	    "files":    {Title: "资源管理器", Content: filesView},
//	    "git":      {Title: "源代码管理", Content: gitView},
//	    "terminal": {Title: "终端", Content: termView},
//	})
//
// 布局（各侧宽度 / 活动页签 / 折叠态）自动写入偏好，重启恢复。

// DockPanel 一个可停靠面板。
type DockPanel struct {
	Title   string
	Icon    string
	Content func() *Node
	// Badge 非空时在页签上显示小标记（如未读数）。
	Badge string
}

// DockState 停靠布局状态。
type DockState struct {
	Key string
	// Left / Right / Bottom 各侧面板 ID 顺序。
	Left, Right, Bottom []string
	// Sizes 各侧尺寸（像素）与折叠态。
	LeftW, RightW, BottomH float32
	Collapsed              map[string]bool
	// Active 每个区域当前页签（区域键：left/right/bottom）。
	Active map[string]string
}

var docks = map[string]*DockState{}

// NewDock 创建（或复用）停靠状态。
func NewDock(key string) *DockState {
	if s, ok := docks[key]; ok {
		return s
	}
	s := &DockState{
		Key: key, LeftW: 220, RightW: 260, BottomH: 180,
		Collapsed: map[string]bool{}, Active: map[string]string{},
	}
	if v := GetPrefFloat("dock."+key+".leftW", 0); v > 0 {
		s.LeftW = v
	}
	if v := GetPrefFloat("dock."+key+".rightW", 0); v > 0 {
		s.RightW = v
	}
	if v := GetPrefFloat("dock."+key+".bottomH", 0); v > 0 {
		s.BottomH = v
	}
	if v, ok := GetPref("dock." + key + ".active.left"); ok {
		s.Active["left"] = v
	}
	if v, ok := GetPref("dock." + key + ".active.right"); ok {
		s.Active["right"] = v
	}
	if v, ok := GetPref("dock." + key + ".active.bottom"); ok {
		s.Active["bottom"] = v
	}
	if v, ok := GetPref("dock." + key + ".left"); ok && v != "" {
		s.Left = strings.Split(v, ",")
	}
	if v, ok := GetPref("dock." + key + ".right"); ok && v != "" {
		s.Right = strings.Split(v, ",")
	}
	if v, ok := GetPref("dock." + key + ".bottom"); ok && v != "" {
		s.Bottom = strings.Split(v, ",")
	}
	docks[key] = s
	return s
}

// Save 持久化布局。
func (s *DockState) Save() {
	if s == nil {
		return
	}
	p := "dock." + s.Key
	SetPrefFloat(p+".leftW", s.LeftW)
	SetPrefFloat(p+".rightW", s.RightW)
	SetPrefFloat(p+".bottomH", s.BottomH)
	SetPref(p+".left", strings.Join(s.Left, ","))
	SetPref(p+".right", strings.Join(s.Right, ","))
	SetPref(p+".bottom", strings.Join(s.Bottom, ","))
	SetPref(p+".active.left", s.Active["left"])
	SetPref(p+".active.right", s.Active["right"])
	SetPref(p+".active.bottom", s.Active["bottom"])
}

// TogglePanel 显示/折叠某面板（侧栏首个面板折叠时整侧收起）。
func (s *DockState) TogglePanel(id string) {
	if s == nil {
		return
	}
	s.Collapsed[id] = !s.Collapsed[id]
	RequestRedraw()
}

// Activate 切换某区域的活动页签。
func (s *DockState) Activate(side, id string) {
	if s == nil {
		return
	}
	s.Active[side] = id
	s.Collapsed[id] = false
	s.Save()
	RequestRedraw()
}

// DockLayout 渲染停靠布局。
func DockLayout(st *DockState, center *Node, panels map[string]DockPanel) *Node {
	if st == nil {
		return center
	}
	main := center
	if main == nil {
		main = Box().St(St().Wp(100))
	}

	left := dockSide(st, "left", st.Left, panels, &st.LeftW, true)
	right := dockSide(st, "right", st.Right, panels, &st.RightW, true)
	bottom := dockSide(st, "bottom", st.Bottom, panels, &st.BottomH, false)

	row := []*Node{}
	if left != nil {
		row = append(row, left, dockSplitter(st, "left", true, &st.LeftW, 160, 420))
	}
	row = append(row, Box(main).St(St().GrowTo(1)))
	if right != nil {
		row = append(row, dockSplitter(st, "right", true, &st.RightW, 180, 480), right)
	}
	body := Row(row...).St(St().GapOf(0)).St(St().Wp(100).Hp(100))
	if bottom != nil {
		return Column(
			Box(body).St(St().GrowTo(1)),
			dockSplitter(st, "bottom", false, &st.BottomH, 80, 480),
			bottom,
		).St(St().Wp(100).Hp(100))
	}
	return body
}

// dockSide 渲染一侧（页签 + 内容）。
func dockSide(st *DockState, side string, ids []string, panels map[string]DockPanel, size *float32, horiz bool) *Node {
	if len(ids) == 0 {
		return nil
	}
	active := st.Active[side]
	if active == "" || !containsStr(ids, active) {
		active = ids[0]
	}
	if st.Collapsed[active] {
		// 折叠态：只留一条窄边，点击展开
		label := dockTitle(panels, active)
		var btn *Node
		if horiz {
			btn = Text("»").St(St().W(22).Hp(100).Text(12, 600, ColTextMuted).
				TAlign(AlignCenter).CenterYOn().BgC(ColBgSoft))
		} else {
			btn = Text("▲ " + label).St(St().Wp(100).H(22).Text(11, 600, ColTextMuted).
				CenterYOn().TAlign(AlignCenter).BgC(ColBgSoft))
		}
		btn.K("dock-" + st.Key + "-" + side + "-expand").Tap(func() {
			st.Collapsed[active] = false
			RequestRedraw()
		})
		return btn
	}

	tabs := make([]*Node, 0, len(ids)*2)
	for _, id := range ids {
		pid := id
		label := panels[pid].Title
		if label == "" {
			label = pid
		}
		color := ColTextMuted
		weight := int32(500)
		bg := "transparent"
		if pid == active {
			color, weight = ColText, 600
			bg = ColBgSoft
		}
		tab := Row(
			Text(label).St(St().Text(11.5, weight, color)),
			If(panels[pid].Badge != "", func() *Node {
				return Tag(panels[pid].Badge, ColPrimary)
			}),
			Text("▾").St(St().Text(9, 500, ColPlaceholder)).Tap(func() {
				st.Collapsed[pid] = true
				RequestRedraw()
			}),
		).St(St().GapOf(6).PadXY(10, 5).R(7).BgC(bg)).AlignTo(AlignCenter)
		tab.K("dock-" + st.Key + "-tab-" + pid).Tap(func() {
			st.Activate(side, pid)
		})
		tabs = append(tabs, tab)
	}
	head := Row(tabs...).St(St().GapOf(2).Padding(4).BgC(ColHeader).Clipped())

	content := EmptyState("", "面板为空", "", "", nil)
	if p, ok := panels[active]; ok && p.Content != nil {
		content = p.Content()
	}
	col := Column(head, Divider(), Box(content).St(St().GrowTo(1).Wp(100).Clipped()))
	if horiz {
		col.St(St().W(*size).Hp(100).BgC(ColBg))
	} else {
		col.St(St().Wp(100).H(*size).BgC(ColBg))
	}
	return col
}

// dockSplitter 拖动调整尺寸的分隔条。
func dockSplitter(st *DockState, side string, horiz bool, size *float32, min, max float32) *Node {
	key := "dock-" + st.Key + "-" + side + "-split"
	var bar *Node
	if horiz {
		bar = Box().St(St().W(5).Hp(100).BgC(ColLine).NoFocusRing())
	} else {
		bar = Box().St(St().Wp(100).H(5).BgC(ColLine).NoFocusRing())
	}
	bar.K(key)
	var startSize, startPos float32
	bar.OnPress = func() {
		startSize = *size
		mx, my := MousePos()
		if horiz {
			startPos = mx
		} else {
			startPos = my
		}
	}
	bar.OnDrag = func(_, _ float32) {
		mx, my := MousePos()
		var d float32
		if horiz {
			d = mx - startPos
			if side == "right" {
				d = -d
			}
		} else {
			d = -(my - startPos)
		}
		v := startSize + d
		if v < min {
			v = min
		}
		if v > max {
			v = max
		}
		*size = v
		st.Save()
		RequestRedraw()
	}
	return bar
}

func dockTitle(panels map[string]DockPanel, id string) string {
	if p, ok := panels[id]; ok && p.Title != "" {
		return p.Title
	}
	return id
}

func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
