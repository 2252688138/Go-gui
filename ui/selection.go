// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"sort"
	"strings"
)

// 跨节点文本选择：拖选一段区域内的文字，Ctrl+C 复制。
//
//	ui.SelectionArea("doc", ui.Markdown(...))
//
// 实现方式：拖拽产生一个矩形选区，按上一帧文本节点的实际矩形取交集，
// 按「从上到下、从左到右」顺序拼接文字，并绘制半透明高亮块。
// 限制：不按字形精确切分（整节点命中即整段入选），适合正文阅读场景。

// SelectionState 选择状态。
type SelectionState struct {
	Active bool
	X0, Y0 float32
	X1, Y1 float32
	Text   string
}

var sel = &SelectionState{}

// SelectedText 当前选中的文字。
func SelectedText() string { return sel.Text }

// HasSelection 是否存在选中文字。
func HasSelection() bool { return sel.Active && sel.Text != "" }

// ClearSelection 清除选区。
func ClearSelection() {
	sel.Active = false
	sel.Text = ""
	RequestRedraw()
}

func init() {
	RegisterShortcut(ScopeGlobal, Shortcut{
		ID: "edit.copy", Keys: "Ctrl+C", Label: "复制", Group: "编辑",
		Run: func() {
			if HasSelection() {
				ClipboardSetText(sel.Text)
				ToastShow("已复制")
			}
		},
	})
}

// SelectionArea 包裹内容并允许拖选文本。
func SelectionArea(key string, children ...*Node) *Node {
	area := Stack(children...).St(St().Wp(100))
	area.K(key)
	area.OnPress = func() {
		x, y := MousePos()
		if r, ok := LastFrameOf(key); ok {
			sel.Active = true
			sel.X0, sel.Y0 = x-r.X, y-r.Y
			sel.X1, sel.Y1 = sel.X0, sel.Y0
			RequestRedraw()
		}
	}
	area.OnDrag = func(x, y float32) {
		sel.X1, sel.Y1 = x, y
		sel.Text = collectSelection(key)
		RequestRedraw()
	}
	area.OnDragEnd = func() {
		RequestRedraw()
	}
	// 高亮块：基于上一帧文本节点矩形生成
	if sel.Active {
		if r, ok := LastFrameOf(key); ok {
			rect := normRect(sel.X0, sel.Y0, sel.X1, sel.Y1)
			for _, n := range textNodesIn(currentRoot(), r.X+rect.X, r.Y+rect.Y, r.X+rect.X+rect.W, r.Y+rect.Y+rect.H, 0, 0) {
				nr := n.Frame
				nr.X -= r.X
				nr.Y -= r.Y
				area.Add(Box().St(St().AbsTL(nr.X, nr.Y).W(nr.W).H(nr.H).
					BgC(paintWithAlpha(ColPrimary, 0.28)).NoFocusRing()))
			}
		}
	}
	return area
}

// normRect 规整为左上角 + 宽高。
func normRect(x0, y0, x1, y1 float32) (out struct{ X, Y, W, H float32 }) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	out.X, out.Y, out.W, out.H = x0, y0, x1-x0, y1-y0
	return out
}

func currentRoot() *Node {
	if a := currentApp.Load(); a != nil {
		return a.root
	}
	return nil
}

// textNodesIn 收集与矩形相交的文本节点。
func textNodesIn(root *Node, x0, y0, x1, y1, sx, sy float32) []*Node {
	if root == nil {
		return nil
	}
	var out []*Node
	var walk func(n *Node, sy float32)
	walk = func(n *Node, sy float32) {
		if n == nil || n.Style.Opacity <= 0 {
			return
		}
		r := n.Frame
		r.Y += sy
		if n.Kind == KText && n.Text != "" {
			if r.X < x1 && r.Right() > x0 && r.Y < y1 && r.Bottom() > y0 {
				out = append(out, n)
			}
			return
		}
		childSy := sy
		if n.Kind == KScroll {
			if a := currentApp.Load(); a != nil {
				childSy = sy - a.scrolls[n.Key]
			}
		}
		for _, c := range n.Children {
			walk(c, childSy)
		}
	}
	walk(root, sy)
	return out
}

// collectSelection 按阅读顺序拼接选区文字。
func collectSelection(areaKey string) string {
	ar, ok := LastFrameOf(areaKey)
	if !ok {
		return ""
	}
	rect := normRect(sel.X0, sel.Y0, sel.X1, sel.Y1)
	nodes := textNodesIn(currentRoot(), ar.X+rect.X, ar.Y+rect.Y,
		ar.X+rect.X+rect.W, ar.Y+rect.Y+rect.H, 0, 0)
	sort.SliceStable(nodes, func(i, j int) bool {
		ri, rj := nodes[i].Frame, nodes[j].Frame
		if absF(ri.Y-rj.Y) > 4 {
			return ri.Y < rj.Y
		}
		return ri.X < rj.X
	})
	var b strings.Builder
	lastY := float32(-1e9)
	for _, n := range nodes {
		if lastY > -1e8 && absF(n.Frame.Y-lastY) > 4 {
			b.WriteByte('\n')
		}
		b.WriteString(n.Text)
		lastY = n.Frame.Y
	}
	return b.String()
}
