// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"sync"
)

// Undo/Redo 基础设施：命令栈 + 事务 + 合并。
//
//	st := ui.NewUndo()
//	st.Do("重命名", func() { name = newName }, func() { name = oldName })
//	st.Undo(); st.Redo()
//
// 默认栈（DefaultUndo）已接好 Ctrl+Z / Ctrl+Y。

// UndoEntry 一条可撤销操作。
type UndoEntry struct {
	Label string
	Undo  func()
	Redo  func()
}

// UndoStack 命令栈（非线程安全，只在 UI 线程操作）。
type UndoStack struct {
	entries []UndoEntry
	index   int
	Limit   int
	txLabel string
	txUndo  []func()
	txRedo  []func()
	busy    bool
}

var undoMu sync.Mutex

// NewUndo 创建命令栈（默认上限 100 步）。
func NewUndo() *UndoStack { return &UndoStack{Limit: 100} }

var defaultUndo = NewUndo()

// DefaultUndo 默认命令栈（已绑定 Ctrl+Z / Ctrl+Y）。
func DefaultUndo() *UndoStack { return defaultUndo }

func init() {
	RegisterShortcut(ScopeGlobal, Shortcut{
		ID: "edit.undo", Keys: "Ctrl+Z", Label: "撤销", Group: "编辑",
		Run: func() { defaultUndo.Undo() },
	})
	RegisterShortcut(ScopeGlobal, Shortcut{
		ID: "edit.redo", Keys: "Ctrl+Y", Label: "重做", Group: "编辑",
		Run: func() { defaultUndo.Redo() },
	})
}

// Do 执行并记录一步操作。
func (u *UndoStack) Do(label string, redo func(), undo func()) {
	if u == nil {
		return
	}
	if redo != nil {
		redo()
	}
	u.Push(label, undo, redo)
}

// Push 只记录（操作已由业务自己完成）。
func (u *UndoStack) Push(label string, undo, redo func()) {
	if u == nil {
		return
	}
	u.busy = true
	defer func() { u.busy = false }()
	if u.index < len(u.entries) {
		u.entries = u.entries[:u.index]
	}
	u.entries = append(u.entries, UndoEntry{Label: label, Undo: undo, Redo: redo})
	u.index = len(u.entries)
	if u.Limit > 0 && len(u.entries) > u.Limit {
		cut := len(u.entries) - u.Limit
		u.entries = u.entries[cut:]
		u.index -= cut
	}
	RequestRedraw()
}

// CanUndo / CanRedo 是否可撤销 / 重做。
func (u *UndoStack) CanUndo() bool { return u != nil && u.index > 0 }
func (u *UndoStack) CanRedo() bool { return u != nil && u.index < len(u.entries) }

// Undo 撤销一步。
func (u *UndoStack) Undo() {
	if u == nil || !u.CanUndo() {
		return
	}
	u.index--
	e := u.entries[u.index]
	if e.Undo != nil {
		e.Undo()
	}
	RequestRedraw()
}

// Redo 重做一步。
func (u *UndoStack) Redo() {
	if u == nil || !u.CanRedo() {
		return
	}
	e := u.entries[u.index]
	u.index++
	if e.Redo != nil {
		e.Redo()
	}
	RequestRedraw()
}

// Clear 清空历史。
func (u *UndoStack) Clear() {
	if u == nil {
		return
	}
	u.entries = nil
	u.index = 0
}

// Label 当前可撤销操作的名称（菜单显示「撤销：xxx」）。
func (u *UndoStack) Label() string {
	if u == nil || !u.CanUndo() {
		return ""
	}
	return u.entries[u.index-1].Label
}

// BeginTx 开始事务（多次修改合并为一步）。
func (u *UndoStack) BeginTx(label string) {
	if u == nil {
		return
	}
	u.txLabel = label
	u.txUndo = nil
	u.txRedo = nil
}

// Tx 向当前事务追加一段反向 / 正向操作。
func (u *UndoStack) Tx(undo, redo func()) {
	if u == nil || u.txLabel == "" {
		return
	}
	if undo != nil {
		u.txUndo = append(u.txUndo, undo)
	}
	if redo != nil {
		u.txRedo = append(u.txRedo, redo)
	}
}

// CommitTx 提交事务（逆序执行反向操作）。
func (u *UndoStack) CommitTx() {
	if u == nil || u.txLabel == "" {
		return
	}
	undo := u.txUndo
	redo := u.txRedo
	label := u.txLabel
	u.txLabel = ""
	u.txUndo, u.txRedo = nil, nil
	if len(undo) == 0 && len(redo) == 0 {
		return
	}
	u.Push(label, func() {
		for i := len(undo) - 1; i >= 0; i-- {
			undo[i]()
		}
	}, func() {
		for _, fn := range redo {
			fn()
		}
	})
}

// UndoHistory 历史列表（调试 / 面板展示）。
func (u *UndoStack) History() []string {
	if u == nil {
		return nil
	}
	out := make([]string, 0, len(u.entries))
	for i, e := range u.entries {
		mark := " "
		if i < u.index {
			mark = "✓"
		}
		out = append(out, fmt.Sprintf("%s %s", mark, e.Label))
	}
	return out
}

// UndoHistoryPanel 历史面板。
func UndoHistoryPanel(u *UndoStack) *Node {
	if u == nil {
		return nil
	}
	rows := make([]*Node, 0, len(u.entries))
	for i, e := range u.entries {
		color := ColTextMuted
		if i < u.index {
			color = ColTextDim
		}
		if i == u.index-1 {
			color = ColPrimary
		}
		rows = append(rows, Text(e.Label).St(St().Text(12, 500, color).PadXY(8, 5)))
	}
	if len(rows) == 0 {
		return EmptyState("", "没有历史记录", "", "", nil)
	}
	return Column(rows...).St(St().GapOf(2).Padding(8).R(10).BgC(ColPanel).
		Stroke(1, ColStroke))
}
