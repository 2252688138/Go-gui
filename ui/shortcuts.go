// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"sort"
	"strings"
)

// 应用内快捷键注册表：作用域 + 冲突检测 + 序列化。
//
// 与全局热键（RegisterHotkey，焦点不在窗口也触发）不同，这里只处理窗口内的按键。
//
//	ui.RegisterShortcut(ui.ScopeGlobal, ui.Shortcut{
//	    ID: "file.save", Keys: "Ctrl+S", Label: "保存", Group: "文件", Run: save,
//	})
//	// 帮助面板：
//	ui.ShortcutsHelp(ui.ScopeGlobal)
//
// 无修饰键的普通字符不会登记为快捷键（避免吞掉输入），F1~F12 例外。

// ShortcutScope 快捷键作用域。
type ShortcutScope string

const (
	// ScopeGlobal 全局（窗口内任意位置）。
	ScopeGlobal ShortcutScope = "global"
	// ScopePanel 当前面板（如编辑器）。
	ScopePanel ShortcutScope = "panel"
	// ScopeModal 模态浮层（优先级最高）。
	ScopeModal ShortcutScope = "modal"
)

// Shortcut 一条快捷键。
type Shortcut struct {
	ID    string
	Keys  string // "Ctrl+S" / "F11" / "Ctrl+Shift+P"
	Label string
	Group string
	Run   func()
}

var (
	shortcutSets = map[ShortcutScope]map[string]*Shortcut{}
	scopeStack   []ShortcutScope
	lastShortcut struct {
		keys string
		at   float32
	}
)

// RegisterShortcut 登记快捷键；同一作用域内组合冲突时返回 false 并保留原绑定。
func RegisterShortcut(scope ShortcutScope, sc Shortcut) bool {
	if sc.ID == "" || sc.Keys == "" || sc.Run == nil {
		return false
	}
	if !shortcutKeysAllowed(sc.Keys) {
		return false
	}
	m := shortcutSets[scope]
	if m == nil {
		m = map[string]*Shortcut{}
		shortcutSets[scope] = m
	}
	norm := normalizeCombo(sc.Keys)
	for _, other := range m {
		if other.ID != sc.ID && normalizeCombo(other.Keys) == norm {
			return false
		}
	}
	cp := sc
	m[sc.ID] = &cp
	return true
}

// UnregisterShortcut 移除快捷键。
func UnregisterShortcut(scope ShortcutScope, id string) {
	if m := shortcutSets[scope]; m != nil {
		delete(m, id)
	}
}

// ShortcutConflict 返回占用该组合的快捷键 ID（无冲突返回空串）。
func ShortcutConflict(scope ShortcutScope, keys string) string {
	norm := normalizeCombo(keys)
	for _, other := range shortcutSets[scope] {
		if normalizeCombo(other.Keys) == norm {
			return other.ID
		}
	}
	return ""
}

// SetShortcutScope 压入/设置当前面板作用域（空串回到 global）。
func SetShortcutScope(scope ShortcutScope) {
	if scope == "" || scope == ScopeGlobal {
		scopeStack = nil
		return
	}
	scopeStack = []ShortcutScope{scope}
}

// PushShortcutScope 进入模态作用域（退出时 PopShortcutScope）。
func PushShortcutScope(scope ShortcutScope) { scopeStack = append(scopeStack, scope) }

// PopShortcutScope 退出模态作用域。
func PopShortcutScope() {
	if len(scopeStack) > 0 {
		scopeStack = scopeStack[:len(scopeStack)-1]
	}
}

// shortcutKeysAllowed 只拦截带修饰键的组合或功能键，避免吞输入。
func shortcutKeysAllowed(keys string) bool {
	k := strings.ToLower(keys)
	if strings.Contains(k, "ctrl+") || strings.Contains(k, "alt+") || strings.Contains(k, "shift+") {
		return true
	}
	up := strings.ToUpper(strings.TrimSpace(keys))
	return strings.HasPrefix(up, "F") && len(up) >= 2
}

// normalizeCombo 统一大小写与顺序：Ctrl+Shift+K → ctrl+shift+k。
func normalizeCombo(keys string) string {
	parts := strings.Split(keys, "+")
	mods := []string{}
	main := ""
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		switch p {
		case "ctrl", "control":
			mods = append(mods, "ctrl")
		case "shift":
			mods = append(mods, "shift")
		case "alt", "menu":
			mods = append(mods, "alt")
		case "win", "meta":
			mods = append(mods, "win")
		case "":
		default:
			main = p
		}
	}
	sort.Strings(mods)
	return strings.Join(append(mods, main), "+")
}

// comboOf 当前按键状态 → 组合字符串。
func comboOf(vk uint32) string {
	var parts []string
	if CtrlDown() {
		parts = append(parts, "ctrl")
	}
	if ShiftDown() {
		parts = append(parts, "shift")
	}
	if AltDown() {
		parts = append(parts, "alt")
	}
	parts = append(parts, strings.ToLower(KeyName(vk)))
	return strings.Join(parts, "+")
}

// DispatchShortcuts 在当前按键下查找并执行快捷键（App 的事件循环调用）。
func DispatchShortcuts(vk uint32) bool {
	combo := comboOf(vk)
	scopes := []ShortcutScope{ScopeGlobal}
	for _, s := range scopeStack {
		if s != ScopeGlobal {
			scopes = append(scopes, s)
		}
	}
	scopes = append(scopes, ScopeModal)
	for i := len(scopes) - 1; i >= 0; i-- {
		m := shortcutSets[scopes[i]]
		for _, sc := range m {
			if normalizeCombo(sc.Keys) == combo {
				sc.Run()
				RequestRedraw()
				return true
			}
		}
	}
	return false
}

// ShortcutList 按作用域列出快捷键（帮助面板用）。
func ShortcutList(scope ShortcutScope) []Shortcut {
	out := make([]Shortcut, 0, len(shortcutSets[scope]))
	for _, sc := range shortcutSets[scope] {
		out = append(out, *sc)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Keys < out[j].Keys
	})
	return out
}

// ShortcutsHelp 快捷键帮助列表（分组展示）。
func ShortcutsHelp(scope ShortcutScope) *Node {
	list := ShortcutList(scope)
	if len(list) == 0 {
		return EmptyState("", "还没有绑定快捷键", "在设置里添加快捷键后这里会显示", "", nil)
	}
	rows := make([]*Node, 0, len(list))
	lastGroup := "\x00"
	for _, sc := range list {
		if sc.Group != lastGroup {
			if lastGroup != "\x00" {
				rows = append(rows, Box().St(St().H(8)))
			}
			rows = append(rows, Text(sc.Group).St(St().Text(11, 600, ColTextMuted)))
			lastGroup = sc.Group
		}
		rows = append(rows, Row(
			Text(sc.Label).St(St().Text(12.5, 500, ColText)).Grow(1),
			Text(sc.Keys).St(St().Text(10.5, 500, ColTextMuted).PadXY(7, 3).R(5).
				BgC(ColBgSoft).Stroke(1, ColStroke)),
		).St(St().PadOnly(0, 0, 6, 0)).AlignTo(AlignCenter))
	}
	return Column(rows...).St(St().GapOf(4).Padding(14).R(12).BgC(ColPanel).
		Stroke(1, ColStroke))
}

// ShortcutSerialize 把当前全局快捷键序列化为 "id=keys" 行（保存到偏好）。
func ShortcutSerialize(scope ShortcutScope) string {
	list := ShortcutList(scope)
	lines := make([]string, 0, len(list))
	for _, sc := range list {
		lines = append(lines, sc.ID+"="+sc.Keys)
	}
	return strings.Join(lines, "\n")
}

// ShortcutRebind 修改绑定（冲突时返回 false）。
func ShortcutRebind(scope ShortcutScope, id, keys string) bool {
	m := shortcutSets[scope]
	if m == nil {
		return false
	}
	sc := m[id]
	if sc == nil || !shortcutKeysAllowed(keys) {
		return false
	}
	norm := normalizeCombo(keys)
	for _, other := range m {
		if other.ID != id && normalizeCombo(other.Keys) == norm {
			return false
		}
	}
	sc.Keys = keys
	RequestRedraw()
	return true
}
