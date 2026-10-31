// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"gogui/internal/geom"
	"gogui/internal/gpu"
)

// 键盘焦点管理：Tab 遍历 / 焦点环 / 作用域（模态焦点陷阱）/ Enter 激活 / ESC 逐层关闭。
//
// 约定：
//   - 可聚焦节点 = Key != "" 且 Style.Focusable（输入框、下拉、日历、按钮等）。
//   - Tab 顺序 = 先序遍历顺序；模态弹窗打开时用 PushFocusScope(key, true) 把遍历限制在
//     该节点子树内，关闭时 PopFocusScope(key) 恢复原焦点。
//   - 鼠标点击后不显示焦点环（focusVisible=false），按 Tab 后才显示，符合桌面惯例。
//   - 浮层（菜单 / 下拉 / 命令面板 / 弹窗）用 OpenOverlay/CloseOverlay 登记，
//     ESC 从最上层开始逐个消费，不会一次关掉多层或误关窗口。

type focusScopeEntry struct {
	key  string
	trap bool
	prev string
}

var (
	focusScopes  []focusScopeEntry
	focusVisible = false
)

// FocusVisible 当前是否应绘制焦点环（键盘导航中为 true）。
func FocusVisible() bool { return focusVisible }

// SetFocusVisible 手动控制焦点环可见性（触摸端可关闭）。
func SetFocusVisible(v bool) {
	if focusVisible != v {
		focusVisible = v
		RequestRedraw()
	}
}

// PushFocusScope 进入一个焦点作用域（trap=true 时 Tab 只在作用域内循环）。
// 同 key 重复调用无副作用；prev 记录进入前的焦点，用于退出时恢复。
func PushFocusScope(key string, trap bool) {
	if key == "" {
		return
	}
	for _, s := range focusScopes {
		if s.key == key {
			return
		}
	}
	focusScopes = append(focusScopes, focusScopeEntry{key: key, trap: trap, prev: FocusedKey()})
}

// PopFocusScope 退出作用域并恢复进入前的焦点（若该节点仍存在）。
func PopFocusScope(key string) {
	if key == "" {
		return
	}
	for i := len(focusScopes) - 1; i >= 0; i-- {
		if focusScopes[i].key != key {
			continue
		}
		prev := focusScopes[i].prev
		focusScopes = append(focusScopes[:i], focusScopes[i+1:]...)
		if a := currentApp.Load(); a != nil {
			if prev != "" && a.root != nil && findKey(a.root, prev) != nil {
				a.focusKey = prev
			} else {
				a.focusKey = ""
			}
			a.redrawRequest.Store(true)
		}
		return
	}
}

// FocusScopeDepth 当前作用域层数（调试 / 测试用）。
func FocusScopeDepth() int { return len(focusScopes) }

// ResetFocusScopes 清空作用域（窗口销毁 / 切页时调用）。
func ResetFocusScopes() {
	focusScopes = nil
	focusVisible = false
}

// focusRoot 返回当前作用域的根节点（无作用域时返回整棵树）。
func (a *App) focusRoot() *Node {
	if a == nil || a.root == nil {
		return nil
	}
	for i := len(focusScopes) - 1; i >= 0; i-- {
		if n := findKey(a.root, focusScopes[i].key); n != nil {
			return n
		}
	}
	return a.root
}

// focusables 收集作用域内可聚焦节点（先序）。
func (a *App) focusables() []*Node {
	root := a.focusRoot()
	if root == nil {
		return nil
	}
	out := make([]*Node, 0, 16)
	var walk func(n *Node)
	walk = func(n *Node) {
		if n == nil || n.Disabled || n.Style.Opacity <= 0 {
			return
		}
		if n.Style.Focusable && n.Key != "" {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

// FocusNext 焦点移到下一个可聚焦节点（dir=1）或上一个（dir=-1）。
func FocusNext() { focusMove(1) }
func FocusPrev() { focusMove(-1) }

func focusMove(dir int) {
	a := currentApp.Load()
	if a == nil {
		return
	}
	list := a.focusables()
	if len(list) == 0 {
		return
	}
	idx := -1
	for i, n := range list {
		if n.Key == a.focusKey {
			idx = i
			break
		}
	}
	var next int
	if idx < 0 {
		if dir > 0 {
			next = 0
		} else {
			next = len(list) - 1
		}
	} else {
		next = (idx + dir + len(list)) % len(list)
	}
	a.focusKey = list[next].Key
	focusVisible = true
	a.redrawRequest.Store(true)
}

// activateFocused 触发焦点节点的键盘激活（Enter / Space）。
func (a *App) activateFocused() bool {
	n := a.focusedNode()
	if n == nil {
		return false
	}
	if n.OnActivate != nil {
		n.OnActivate()
		return true
	}
	if n.OnClick != nil {
		n.OnClick()
		return true
	}
	return false
}

// ---- 浮层栈（ESC 逐层关闭）----

type overlayEntry struct {
	key   string
	close func() bool
}

var (
	overlays    []overlayEntry
	overlayTick int
)

// OpenOverlay 登记一个打开的浮层。close 返回 true 表示已消费 ESC。
// 同一 key 重复登记会更新 close 并保持原有层级。
func OpenOverlay(key string, close func() bool) {
	if key == "" || close == nil {
		return
	}
	for i := range overlays {
		if overlays[i].key == key {
			overlays[i].close = close
			return
		}
	}
	overlays = append(overlays, overlayEntry{key: key, close: close})
	overlayTick++
}

// CloseOverlay 移除登记（浮层已经自行关闭时调用）。
func CloseOverlay(key string) {
	for i := len(overlays) - 1; i >= 0; i-- {
		if overlays[i].key == key {
			overlays = append(overlays[:i], overlays[i+1:]...)
			return
		}
	}
}

// OverlayDepth 当前登记的浮层数量。
func OverlayDepth() int { return len(overlays) }

// closeTopOverlay 让最上层浮层消费 ESC；返回是否已消费。
func closeTopOverlay() bool {
	for i := len(overlays) - 1; i >= 0; i-- {
		if overlays[i].close() {
			return true
		}
		CloseOverlay(overlays[i].key)
	}
	return false
}

// ResetOverlays 清空浮层登记（窗口销毁时调用）。
func ResetOverlays() { overlays = nil }

// paintFocusRing 在焦点节点外框绘制 2px 焦点环（键盘导航期间）。
func (a *App) paintFocusRing(n *Node, r geom.Rect, st *Style) {
	if !focusVisible || st.NoRing || n.Key == "" || n.Key != a.focusKey {
		return
	}
	pad := float32(2)
	rad := st.Radius
	for i := range rad {
		if rad[i] > 0 {
			rad[i] += pad
		}
	}
	col := C(ColFocus)
	a.r.Rect(gpu.Shape{
		Rect:        geom.Rect{X: r.X - pad, Y: r.Y - pad, W: r.W + pad*2, H: r.H + pad*2},
		Radius:      rad,
		Stroke:      2,
		StrokeColor: col,
	})
}
