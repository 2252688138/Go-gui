//go:build windows

package ui

import (
	"fmt"
	"strings"
)

func dumpNode(sb *strings.Builder, n *Node, depth int) {
	if n == nil {
		return
	}
	ind := strings.Repeat("  ", depth)
	kinds := [...]string{"Box", "Text", "Button", "Scroll", "Image", "Stack", "Grid", "Wrap", "Masonry", "Video"}
	kind := "Node"
	if int(n.Kind) < len(kinds) {
		kind = kinds[n.Kind]
	}
	txt := n.Text
	if len([]rune(txt)) > 16 {
		txt = string([]rune(txt)[:16])
	}
	fmt.Fprintf(sb, "%s%-6s key=%-12q frame=(%.0f,%.0f %.0fx%.0f) grow=%.1f pad=(%.0f,%.0f) dir=%d align=%d contentH=%.0f %q\n",
		ind, kind, n.Key, n.Frame.X, n.Frame.Y, n.Frame.W, n.Frame.H,
		n.Style.Grow, n.Style.Pad.Top, n.Style.Pad.Left, n.Dir, n.Align, n.ContentH, txt)
	for _, c := range n.Children {
		dumpNode(sb, c, depth+1)
	}
}

// LayoutForTest 在无窗口环境下布局一棵树（供 tools/layouttest 使用）。
func LayoutForTest(root *Node, w, h float32, face string, scale float32) {
	fc := newFontCache(face, scale)
	Layout(root, w, h, fc)
}

// DebugDump 输出整棵布局树到 stdout（用于 tools 调试）。
func DebugDump(n *Node) {
	var sb strings.Builder
	dumpNode(&sb, n, 0)
	fmt.Print(sb.String())
}
