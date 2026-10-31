// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 文件树 / 目录树组件（可展开、可选中、按 Key 记忆展开状态）。
//
//	ui.FileTree(nodes, ui.TreeOptions{
//	    Key:      "file-tree",
//	    Expanded: []string{"cmd", "cmd/pages"}, // 仅首次构建生效
//	    Selected: "ui/layout.go",
//	    OnPick:   func(n ui.TreeNode) { sel = n.Key },
//	})

// TreeKind 树节点类型（决定图标）。
type TreeKind uint8

const (
	TreeFolder TreeKind = iota // 目录：带展开箭头 + 文件夹图标
	TreeFile                   // 文件：叶子节点
	TreeLink                   // 链接 / 远程节点：圆点图标
)

// TreeNode 文件树节点。Children 非空即为可展开分支。
type TreeNode struct {
	Key      string // 唯一键（展开 / 选中按它保存）；为空时按父路径自动生成
	Label    string
	Kind     TreeKind
	Tag      string // 右侧标签（如「12」「已改」）
	TagColor string
	Disabled bool
	Children []TreeNode
}

// TreeOptions 文件树参数。
type TreeOptions struct {
	Key      string
	Expanded []string // 默认展开的节点 Key（仅首次构建时生效，之后由点击记忆）
	Selected string
	Indent   float32 // 每级缩进，默认 16
	RowH     float32 // 行高，默认 30
	OnPick   func(n TreeNode)
	OnToggle func(n TreeNode, open bool)
}

var (
	// treeOpen 每棵树（按 Key）的展开集合。
	treeOpen = map[string]map[string]bool{}
	// treeSel 每棵树（按 Key）当前选中的节点。
	treeSel = map[string]string{}
)

// FileTree 文件树：行内「箭头 + 图标 + 名称 + 右侧标签」，选中行高亮。
func FileTree(items []TreeNode, opts TreeOptions) *Node {
	if opts.Key == "" {
		opts.Key = "tree"
	}
	if opts.Indent <= 0 {
		opts.Indent = 16
	}
	if opts.RowH <= 0 {
		opts.RowH = 30
	}
	open, ok := treeOpen[opts.Key]
	if !ok {
		open = make(map[string]bool, len(opts.Expanded))
		for _, k := range opts.Expanded {
			open[k] = true
		}
		treeOpen[opts.Key] = open
	}
	sel := opts.Selected
	if v, ok := treeSel[opts.Key]; ok {
		sel = v
	}

	rows := make([]*Node, 0, 32)
	var walk func(list []TreeNode, depth int, prefix string)
	walk = func(list []TreeNode, depth int, prefix string) {
		for _, it := range list {
			n := it
			if n.Key == "" {
				n.Key = prefix + "/" + n.Label
			}
			isOpen := open[n.Key]
			rows = append(rows, buildTreeRow(opts, n, depth, isOpen, sel))
			if len(n.Children) > 0 && isOpen {
				walk(n.Children, depth+1, n.Key)
			}
		}
	}
	walk(items, 0, opts.Key)
	return Column(rows...).St(St().GapOf(1)).K(opts.Key + "-root")
}

// TreeSetOpen 展开 / 收起某个节点（外部按钮「全部展开 / 折叠」用）。
func TreeSetOpen(key, nodeKey string, open bool) {
	m, ok := treeOpen[key]
	if !ok {
		m = map[string]bool{}
		treeOpen[key] = m
	}
	m[nodeKey] = open
	RequestRedraw()
}

// TreeSetAllOpen 展开 / 收起整棵树。
func TreeSetAllOpen(key string, items []TreeNode, open bool) {
	m, ok := treeOpen[key]
	if !ok {
		m = map[string]bool{}
		treeOpen[key] = m
	}
	var walk func(list []TreeNode, prefix string)
	walk = func(list []TreeNode, prefix string) {
		for _, it := range list {
			k := it.Key
			if k == "" {
				k = prefix + "/" + it.Label
			}
			if len(it.Children) > 0 {
				m[k] = open
				walk(it.Children, k)
			}
		}
	}
	walk(items, key)
	RequestRedraw()
}

// TreeSelected 当前选中节点的 Key（未选中返回空串）。
func TreeSelected(key string) string { return treeSel[key] }

// TreeSetSelected 设置选中节点。
func TreeSetSelected(key, nodeKey string) {
	treeSel[key] = nodeKey
	RequestRedraw()
}

