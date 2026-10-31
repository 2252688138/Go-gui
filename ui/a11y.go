// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 无障碍元数据（P3 占位）。
//
// 当前只做「声明 + 导出」：组件把角色/名称/描述挂在节点上，
// A11yTree() 可按顺序导出，供调试、自动化测试或未来的 UIA/AT-SPI Provider 使用。
// 本框架尚未向系统注册 UI Automation Provider（全自绘窗口的系统级无障碍另需
// 平台层改造），因此屏幕阅读器目前读不到这些内容。

// A11yRole 无障碍角色（对应 UIA ControlType 的常用子集）。
type A11yRole int

const (
	A11yNone A11yRole = iota
	A11yButton
	A11yText
	A11yLink
	A11yCheckBox
	A11yRadio
	A11ySwitch
	A11ySlider
	A11yTextField
	A11yList
	A11yListItem
	A11yTab
	A11yDialog
	A11yMenu
	A11yMenuItem
	A11yProgress
	A11yImage
	A11yTable
	A11yTree
)

// A11yRoleName 角色名称（调试与导出用）。
func A11yRoleName(r A11yRole) string {
	switch r {
	case A11yButton:
		return "button"
	case A11yText:
		return "text"
	case A11yLink:
		return "link"
	case A11yCheckBox:
		return "checkbox"
	case A11yRadio:
		return "radio"
	case A11ySwitch:
		return "switch"
	case A11ySlider:
		return "slider"
	case A11yTextField:
		return "textfield"
	case A11yList:
		return "list"
	case A11yListItem:
		return "listitem"
	case A11yTab:
		return "tab"
	case A11yDialog:
		return "dialog"
	case A11yMenu:
		return "menu"
	case A11yMenuItem:
		return "menuitem"
	case A11yProgress:
		return "progress"
	case A11yImage:
		return "image"
	case A11yTable:
		return "table"
	case A11yTree:
		return "tree"
	}
	return "none"
}

// A11yInfo 节点无障碍信息。
type A11yInfo struct {
	Role A11yRole
	Name string
	Desc string
}

// A11yEntry 导出的一项（Key 为节点 Key，Path 为树中位置）。
type A11yEntry struct {
	Key   string
	Role  A11yRole
	Name  string
	Desc  string
	Focus bool
}

// A11ySnapshot 按先序导出界面中的无障碍节点（无名无角色的节点跳过）。
//
// 用途：调试焦点顺序、生成自动化测试选择器；不代表系统级无障碍已可用。
func A11ySnapshot() []A11yEntry { return a11yTreeOf(currentApp.Load()) }

func a11yTreeOf(a *App) []A11yEntry {
	if a == nil || a.root == nil {
		return nil
	}
	var out []A11yEntry
	var walk func(n *Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if n.A11y.Role != A11yNone || n.A11y.Name != "" {
			out = append(out, A11yEntry{
				Key:   n.Key,
				Role:  n.A11y.Role,
				Name:  n.A11y.Name,
				Desc:  n.A11y.Desc,
				Focus: n.Key != "" && n.Key == a.focusKey,
			})
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(a.root)
	return out
}
