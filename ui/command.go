// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"sort"
	"strings"
)

// 命令面板（Ctrl+K）：命令注册表 + 模糊搜索 + 键盘导航。
//
//	ui.RegisterCommand(ui.Command{
//	    ID: "file.save", Title: "保存文件", Group: "文件",
//	    Shortcut: "Ctrl+S", Run: save,
//	})
//	// 根 Stack 末尾：
//	ui.Stack(page, ui.CommandPaletteLayer())
//
// Ctrl+K 由框架内置处理（ToggleCommandPalette）。

// Command 一条可执行命令。
type Command struct {
	ID       string
	Title    string
	Sub      string
	Group    string
	Shortcut string
	Icon     string
	Disabled bool
	Run      func()
}

var (
	commandRegistry = map[string]Command{}
	commandOrder    []string
)

// RegisterCommand 注册（或覆盖）命令。
func RegisterCommand(c Command) {
	if c.ID == "" || c.Run == nil {
		return
	}
	if _, ok := commandRegistry[c.ID]; !ok {
		commandOrder = append(commandOrder, c.ID)
	}
	commandRegistry[c.ID] = c
}

// UnregisterCommand 移除命令。
func UnregisterCommand(id string) {
	if _, ok := commandRegistry[id]; !ok {
		return
	}
	delete(commandRegistry, id)
	for i, v := range commandOrder {
		if v == id {
			commandOrder = append(commandOrder[:i], commandOrder[i+1:]...)
			break
		}
	}
}

// Commands 返回已注册命令（注册顺序）。
func Commands() []Command {
	out := make([]Command, 0, len(commandOrder))
	for _, id := range commandOrder {
		if c, ok := commandRegistry[id]; ok {
			out = append(out, c)
		}
	}
	return out
}

// paletteState 命令面板状态。
var (
	paletteOpen  bool
	paletteQuery string
	paletteIndex int
)

// CommandPaletteOpen 面板是否打开。
func CommandPaletteOpen() bool { return paletteOpen }

// OpenCommandPalette 打开命令面板并聚焦搜索框。
func OpenCommandPalette() {
	if paletteOpen {
		return
	}
	paletteOpen = true
	paletteQuery = ""
	paletteIndex = 0
	inputCaret["cmd-palette-input"] = 0
	SetFocus("cmd-palette-input")
	focusVisible = false
	OpenOverlay("cmd-palette", func() bool {
		CloseCommandPalette()
		return true
	})
	RequestRedraw()
}

// CloseCommandPalette 关闭面板。
func CloseCommandPalette() {
	if !paletteOpen {
		return
	}
	paletteOpen = false
	paletteQuery = ""
	paletteIndex = 0
	CloseOverlay("cmd-palette")
	SetFocus("")
	ResetKeepAlive("cmd-palette-box")
	RequestRedraw()
}

// ToggleCommandPalette 切换面板；返回值表示是否处理（有命令时）。供 App 的 Ctrl+K 调用。
func ToggleCommandPalette() bool {
	if paletteOpen {
		CloseCommandPalette()
		return true
	}
	if len(commandRegistry) == 0 {
		return false
	}
	OpenCommandPalette()
	return true
}

// ---- 模糊匹配 ----

// FuzzyScore 子序列匹配评分：连续命中 / 词首命中加权，无法匹配返回 -1。
func FuzzyScore(query, target string) int {
	if query == "" {
		return 0
	}
	q := []rune(strings.ToLower(query))
	t := []rune(strings.ToLower(target))
	qi := 0
	score := 0
	prevHit := -2
	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if t[ti] != q[qi] {
			continue
		}
		score += 10
		if ti == prevHit+1 {
			score += 8 // 连续
		}
		if ti == 0 || t[ti-1] == ' ' || t[ti-1] == '.' || t[ti-1] == '/' || t[ti-1] == '-' {
			score += 12 // 词首
		}
		prevHit = ti
		qi++
	}
	if qi < len(q) {
		return -1
	}
	// 目标越短越优先
	score -= len(t) / 4
	return score
}

// matchCommands 过滤并排序命令（标题优先，其次副标题 / 分组）。
func matchCommands(query string) []Command {
	all := Commands()
	if strings.TrimSpace(query) == "" {
		out := make([]Command, len(all))
		copy(out, all)
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Group != out[j].Group {
				return out[i].Group < out[j].Group
			}
			return false
		})
		return out
	}
	type scored struct {
		c Command
		s int
	}
	list := make([]scored, 0, len(all))
	for _, c := range all {
		s := FuzzyScore(query, c.Title)
		if s < 0 {
			if v := FuzzyScore(query, c.ID); v >= 0 {
				s = v - 5
			} else if v := FuzzyScore(query, c.Group+" "+c.Sub); v >= 0 {
				s = v - 12
			}
		}
		if s >= 0 {
			list = append(list, scored{c: c, s: s})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].s > list[j].s })
	out := make([]Command, 0, len(list))
	for _, v := range list {
		out = append(out, v.c)
	}
	return out
}

// runPaletteCommand 执行第 i 条匹配命令并关闭面板。
func runPaletteCommand(list []Command, i int) {
	if i < 0 || i >= len(list) || list[i].Disabled || list[i].Run == nil {
		return
	}
	cmd := list[i]
	paletteQuery = ""
	CloseCommandPalette()
	cmd.Run()
	RequestRedraw()
}

// CommandPaletteLayer 渲染面板（放在根 Stack 末尾）。
func CommandPaletteLayer() *Node {
	visible, t := KeepAlive("cmd-palette-box", paletteOpen, 0.14)
	if !visible {
		if !paletteOpen {
			paletteIndex = 0
		}
		return nil
	}
	if paletteOpen {
		RequestRedraw()
	}
	list := matchCommands(paletteQuery)
	if paletteIndex >= len(list) {
		paletteIndex = len(list) - 1
	}
	if paletteIndex < 0 {
		paletteIndex = 0
	}

	// Scrim 在透明度为 0（刚打开的第一帧）时返回 nil，不能直接链式建链
	scrim := Scrim(t*0.75, "cmd-palette-scrim")
	if scrim != nil {
		scrim.Tap(CloseCommandPalette)
	}

	search := InputSubmit("cmd-palette-input", paletteQuery, "输入命令名称…",
		func(v string) {
			paletteQuery = v
			paletteIndex = 0
			RequestRedraw()
		},
		func() { runPaletteCommand(matchCommands(paletteQuery), paletteIndex) })
	search.OnKey = func(k uint32) {
		switch k {
		case VK_DOWN:
			if len(list) > 0 {
				paletteIndex = (paletteIndex + 1) % len(list)
			}
		case VK_UP:
			if len(list) > 0 {
				paletteIndex = (paletteIndex - 1 + len(list)) % len(list)
			}
		case VK_ESCAPE:
			CloseCommandPalette()
		case VK_RETURN:
			runPaletteCommand(matchCommands(paletteQuery), paletteIndex)
		case VK_LEFT, VK_RIGHT, 0x24, 0x23, 0x2E, VK_BACK:
			inputNavKey("cmd-palette-input", k)
		}
		RequestRedraw()
	}
	// 让搜索框外框与面板融为一体
	search.St(St().BgC(ColBgDeep).NoFocusRing())

	width := float32(560)
	if w := ViewportW(); w < width+40 {
		width = w - 40
		if width < 280 {
			width = 280
		}
	}

	rows := make([]*Node, 0, len(list))
	if len(list) == 0 {
		rows = append(rows, Column(
			Text("没有匹配的命令").St(St().Text(12.5, 500, ColTextMuted)),
			Text("试着输入更短的关键词").St(St().Text(11.5, 400, ColTextFaint())),
		).St(St().GapOf(4).Padding(18)).AlignTo(AlignCenter))
	}
	for i, c := range list {
		idx := i
		row := commandRow(c, idx == paletteIndex)
		row.OnHoverEvt(func(h bool) {
			if h && paletteIndex != idx {
				paletteIndex = idx
				RequestRedraw()
			}
		})
		row.Tap(func() { runPaletteCommand(matchCommands(paletteQuery), idx) })
		rows = append(rows, row.K(fmt.Sprintf("cmd-row-%d", i)))
		if i >= 40 {
			break
		}
	}
	body := Column(rows...).St(St().GapOf(2))
	if len(rows) > 9 {
		body = Scroll(rows...).K("cmd-palette-scroll").St(St().H(9 * 46))
	}

	panel := Column(
		Row(Text("⌘").St(St().Text(14, 600, ColPrimary)), search).St(St().GapOf(10)).AlignTo(AlignCenter),
		Divider(),
		body,
	).St(St().W(width).Padding(12).GapOf(10).R(14).BgC(ColPanelDeep).
		Stroke(1, ColStroke).Shadow(48, 20, 0, "rgba(0,0,0,0.6)").
		Fade(t).Translate(0, 10*(1-t)))

	x := (ViewportW() - width) / 2
	if x < 12 {
		x = 12
	}
	panel.St(St().AbsTL(x, 84))

	return Stack(scrim, panel).St(St().AbsFill()).K("cmd-palette-root")
}

// ColTextFaint 弱化文字色（token 别名）。
func ColTextFaint() string { return ColTextMuted }

// commandRow 单条命令行。
func commandRow(c Command, active bool) *Node {
	color := ColText
	if c.Disabled {
		color = ColPlaceholder
	}
	left := Column(
		Text(c.Title).St(St().Text(13, 500, color)),
	)
	if c.Sub != "" {
		left.Add(Text(c.Sub).St(St().Text(11, 400, ColTextMuted)))
	}
	cells := []*Node{}
	if c.Icon != "" {
		cells = append(cells, Text(c.Icon).St(St().Text(13, 500, ColTextMuted).W(20)))
	}
	cells = append(cells, left.Grow(1))
	if c.Group != "" {
		cells = append(cells, Text(c.Group).St(St().Text(10.5, 500, ColTextMuted).PadXY(7, 3).
			R(6).BgC(ColHover)))
	}
	if c.Shortcut != "" {
		cells = append(cells, Text(c.Shortcut).St(St().Text(10.5, 500, ColTextMuted).PadXY(6, 3).
			R(5).BgC(ColBgSoft).Stroke(1, ColStroke)))
	}
	row := Row(cells...).St(St().Wp(100).Padding(10).GapOf(8).R(9)).AlignTo(AlignCenter)
	if active {
		row.St(St().BgC(paintWithAlpha(ColPrimary, 0.20)).Stroke(1, paintWithAlpha(ColPrimary, 0.45)))
	}
	return row
}

// inputNavKey 把光标移动 / 删除键交给输入框（命令面板自定义 OnKey 时复用）。
func inputNavKey(key string, k uint32) {
	n := currentFocused()
	if n == nil || n.Key != key {
		return
	}
	if fn := rawInputKey(key); fn != nil {
		fn(k)
	}
}

// rawInputKey 取出输入框注册的原始按键处理（inputNode 内部把逻辑写在闭包里，
// 这里通过节点 OnChar 无法复用，故命令面板自行维护光标）。
func rawInputKey(key string) func(uint32) {
	// 输入框的自定义 OnKey 已被面板覆盖，这里退化为只处理光标与删除。
	return func(k uint32) {
		runes := []rune(paletteQuery)
		c := inputCaret[key]
		if c < 0 || c > len(runes) {
			c = len(runes)
		}
		switch k {
		case VK_LEFT:
			if c > 0 {
				c--
			}
		case VK_RIGHT:
			if c < len(runes) {
				c++
			}
		case 0x24:
			c = 0
		case 0x23:
			c = len(runes)
		case VK_BACK:
			if c > 0 {
				runes = append(runes[:c-1], runes[c:]...)
				c--
			}
		case VK_DELETE:
			if c < len(runes) {
				runes = append(runes[:c], runes[c+1:]...)
			}
		}
		inputCaret[key] = c
		paletteQuery = string(runes)
	}
}

func currentFocused() *Node {
	a := currentApp.Load()
	if a == nil {
		return nil
	}
	return a.focusedNode()
}
