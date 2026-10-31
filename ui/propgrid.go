// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"strconv"
	"strings"
)

// 属性表编辑器：分组 + 类型编辑器 + 搜索 + 重置。
//
//	ui.PropertyGrid("cfg", []ui.Prop{
//	    {Key: "name",  Label: "名称", Group: "常规", Kind: ui.PropText,   Value: cfg.Name},
//	    {Key: "size",  Label: "字号", Group: "常规", Kind: ui.PropNumber, Value: "13"},
//	    {Key: "dark",  Label: "深色", Group: "外观", Kind: ui.PropBool,   Value: "1"},
//	    {Key: "theme", Label: "主题", Group: "外观", Kind: ui.PropEnum,   Value: "dark",
//	        Options: []string{"dark", "light"}},
//	}, func(key, val string) { apply(key, val) })

// PropKind 编辑器类型。
type PropKind int

const (
	PropText PropKind = iota
	PropNumber
	PropBool
	PropEnum
	PropColor
	PropFile
	PropReadonly
	PropCustom
)

// Prop 一条属性。
type Prop struct {
	Key     string
	Label   string
	Group   string
	Kind    PropKind
	Value   string
	Options []string
	Hint    string
	// Custom 自定义编辑器（Kind == PropCustom）。
	Custom func(value string, onChange func(string)) *Node
}

var (
	propFilters = map[string]string{}
	propGroups  = map[string]map[string]bool{}
)

// PropertyGrid 渲染属性表；onChange 在每次修改时回调。
func PropertyGrid(key string, props []Prop, onChange func(key, val string)) *Node {
	if key == "" {
		key = "propgrid"
	}
	filter := propFilters[key]
	search := SearchInput(key+"-search", filter, "搜索属性…", func(v string) {
		propFilters[key] = v
		RequestRedraw()
	})
	search.St(St().NoFocusRing())

	groups := map[string]bool{}
	order := []string{}
	for _, p := range props {
		g := p.Group
		if g == "" {
			g = "常规"
		}
		if _, ok := groups[g]; !ok {
			groups[g] = true
			order = append(order, g)
		}
	}
	coll := propGroups[key]
	if coll == nil {
		coll = map[string]bool{}
		propGroups[key] = coll
	}
	emit := func(k, v string) {
		if onChange != nil {
			onChange(k, v)
		}
		RequestRedraw()
	}

	rows := []*Node{search}
	q := strings.ToLower(strings.TrimSpace(filter))
	for _, g := range order {
		list := make([]*Node, 0, 8)
		for _, p := range props {
			pg := p.Group
			if pg == "" {
				pg = "常规"
			}
			if pg != g {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(p.Label+" "+p.Key), q) {
				continue
			}
			list = append(list, propRow(key, p, emit))
		}
		if len(list) == 0 {
			continue
		}
		arrow := "▾"
		if coll[g] {
			arrow = "▸"
		}
		head := Row(
			Text(arrow).St(St().Text(10, 600, ColTextMuted)),
			Text(g).St(St().Text(11.5, 600, ColTextDim)),
		).St(St().GapOf(6).PadXY(6, 6).R(6).BgC(ColHeader)).AlignTo(AlignCenter)
		group := g
		head.K(key + "-g-" + g).Tap(func() {
			coll[group] = !coll[group]
			RequestRedraw()
		})
		rows = append(rows, head)
		if !coll[g] {
			rows = append(rows, Column(list...).St(St().GapOf(6).PadOnly(4, 8, 8, 8)))
		}
	}
	if len(rows) == 1 {
		rows = append(rows, EmptyState("", "没有匹配的属性", "", "", nil))
	}
	return Column(rows...).St(St().GapOf(4).Padding(10).R(12).BgC(ColPanel).
		Stroke(1, ColStroke))
}

// propRow 单条属性行（左标签 + 右编辑器）。
func propRow(grid string, p Prop, emit func(string, string)) *Node {
	label := Row(
		Text(p.Label).St(St().Text(12, 500, ColTextDim)),
		If(p.Hint != "", func() *Node {
			return TooltipOf(Text(" ?").St(St().Text(10, 600, ColPlaceholder)), p.Hint)
		}),
	).St(St().GapOf(4).W(110)).AlignTo(AlignCenter)

	var editor *Node
	ek := grid + "-" + p.Key
	switch p.Kind {
	case PropText:
		editor = Input(ek, p.Value, "请输入", func(v string) { emit(p.Key, v) })
	case PropNumber:
		f, _ := strconv.ParseFloat(p.Value, 64)
		editor = NumberInput(ek, f, -1e9, 1e9, 1, func(v float64) {
			emit(p.Key, strconv.FormatFloat(v, 'f', -1, 64))
		})
	case PropBool:
		on := p.Value == "1" || strings.EqualFold(p.Value, "true")
		editor = Switch(ek, on, func(v bool) {
			if v {
				emit(p.Key, "1")
			} else {
				emit(p.Key, "0")
			}
		})
	case PropEnum:
		opts := p.Options
		if len(opts) == 0 {
			opts = []string{p.Value}
		}
		editor = Row(
			Text(p.Value).St(St().Text(12, 500, ColText)).Grow(1),
			Text("▾").St(St().Text(9, 500, ColTextMuted)),
		).St(St().Wp(100).H(32).PadXY(10, 0).GapOf(8).R(8).BgC(ColInput).
			Stroke(1, ColStroke).FocusOn()).AlignTo(AlignCenter)
		idx := 0
		for i, o := range opts {
			if o == p.Value {
				idx = i
			}
		}
		editor.K(ek).Tap(func() {
			next := opts[(idx+1)%len(opts)]
			emit(p.Key, next)
		})
	case PropColor:
		st := NewColor(ek)
		editor = ColorPicker(st, p.Value, func(v string) { emit(p.Key, v) })
		editor.St(St().W(180))
	case PropFile:
		editor = Row(
			Text(p.Value).St(St().Text(11.5, 400, ColTextDim).GrowTo(1).Clipped()),
			Text("选择…").St(St().Text(11, 600, ColPrimary)),
		).St(St().Wp(100).H(32).PadXY(10, 0).GapOf(8).R(8).BgC(ColInput).
			Stroke(1, ColStroke).FocusOn().Clipped()).AlignTo(AlignCenter)
		editor.K(ek).Tap(func() {
			if path, ok := OpenFileDialog("选择文件", "所有文件|*.*"); ok {
				emit(p.Key, path)
			}
		})
	case PropCustom:
		if p.Custom != nil {
			editor = p.Custom(p.Value, func(v string) { emit(p.Key, v) })
		}
	case PropReadonly:
		editor = Text(p.Value).St(St().Text(12, 400, ColTextMuted))
	}
	if editor == nil {
		editor = Text("—").St(St().Text(12, 400, ColPlaceholder))
	}
	row := Row(label, Box(editor).St(St().GrowTo(1))).St(St().GapOf(10).PadOnly(2, 0, 2, 0)).
		AlignTo(AlignCenter)
	count := Row(
		row,
		If(p.Kind != PropReadonly && p.Kind != PropBool, func() *Node {
			return Text("↺").St(St().Text(12, 500, ColPlaceholder).W(18).TAlign(AlignCenter)).
				K(ek + "-reset")
		}),
	).St(St().GapOf(4)).AlignTo(AlignCenter)
	return count
}

// PropResetButton 给属性行绑定「重置为默认值」动作（默认渲染的 ↺ 只做展示，业务可自行接管）。
func PropResetButton(grid, propKey string, def string, onChange func(key, val string)) *Node {
	return Text("↺").St(St().Text(12, 500, ColTextMuted).W(18).TAlign(AlignCenter)).
		K(fmt.Sprintf("%s-%s-reset", grid, propKey)).Tap(func() {
		if onChange != nil {
			onChange(propKey, def)
		}
		RequestRedraw()
	})
}
