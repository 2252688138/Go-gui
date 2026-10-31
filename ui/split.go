// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 可拖分栏：编辑器 / 管理后台骨架。分隔条支持拖动、双击复位、比例持久化。
//
//	ui.Split("main-split", sidebar, content, ui.SplitOptions{
//	    Dir:      ui.Horizontal,
//	    Pos:      0.22,
//	    MinA:     180,
//	    PrefsKey: "layout.sidebar",
//	})
//
// 比例按分隔条中心位置计算（0..1）；尺寸变化时保持比例，最小尺寸优先。

// SplitOptions 分栏参数。
type SplitOptions struct {
	// Dir Horizontal = 左右分栏（默认）；Vertical = 上下分栏。
	Dir Direction
	// Pos 初始比例 0..1（0 = 0.28）。
	Pos float32
	// MinA / MinB 两侧最小尺寸（逻辑像素）。
	MinA, MinB float32
	// HandleW 分隔条厚度（默认 8）。
	HandleW float32
	// PrefsKey 非空时把比例写入偏好（重启恢复）。
	PrefsKey string
	// OnChange 比例变化回调（拖动过程中持续触发）。
	OnChange func(pos float32)
}

const (
	splitHandleKey = "-split-handle"
	splitDragKey   = "-split-drag"
)

// SplitsPos 读取某分栏当前比例（0..1）。
func SplitPos(key string) float32 {
	if v, ok := splitState[key]; ok {
		return v
	}
	return 0
}

// splitState 当前比例（按 key）。
var (
	splitState = map[string]float32{}
	splitDrag  = map[string]splitDragInfo{}
)

type splitDragInfo struct {
	startMouse float32
	startPos   float32
	total      float32
	minA, minB float32
}

// SetSplitPos 设置分栏比例并持久化（代码里做「收起侧栏」等操作）。
func SetSplitPos(key string, pos float32, prefsKey string) {
	if pos < 0 {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}
	splitState[key] = pos
	if prefsKey != "" {
		SetPrefFloat(prefsKey, pos)
	}
	RequestRedraw()
}

// Split 构建分栏。a 占 Pos 比例，b 占剩余。
func Split(key string, a, b *Node, opts SplitOptions) *Node {
	if opts.Pos <= 0 {
		opts.Pos = 0.28
	}
	if opts.Pos >= 1 {
		opts.Pos = 0.98
	}
	if opts.HandleW <= 0 {
		opts.HandleW = 8
	}
	pos, ok := splitState[key]
	if !ok {
		if opts.PrefsKey != "" {
			pos = GetPrefFloat(opts.PrefsKey, opts.Pos)
		} else {
			pos = opts.Pos
		}
	}
	if pos < 0.02 {
		pos = 0.02
	}
	if pos > 0.98 {
		pos = 0.98
	}

	horiz := opts.Dir != Vertical
	var container *Node
	handle := splitHandle(key, horiz, opts.HandleW)

	// 拖动：换算比例并夹取最小尺寸
	apply := func(mx, my float32) {
		d, ok := splitDrag[key]
		if !ok {
			return
		}
		cur := mx
		if !horiz {
			cur = my
		}
		if d.total <= 0 {
			return
		}
		np := d.startPos + (cur-d.startMouse)/d.total
		minP := float32(0)
		if d.total > 0 {
			minP = d.minA / d.total
		}
		maxP := float32(1)
		if d.total > 0 {
			maxP = 1 - d.minB/d.total
		}
		if maxP < minP {
			maxP = minP
		}
		if np < minP {
			np = minP
		}
		if np > maxP {
			np = maxP
		}
		splitState[key] = np
		if opts.OnChange != nil {
			opts.OnChange(np)
		}
		if opts.PrefsKey != "" {
			SetPrefFloat(opts.PrefsKey, np)
		}
		RequestRedraw()
	}

	handle.K(key + splitHandleKey)
	handle.OnPress = func() {
		// 上一帧几何：总尺寸取容器矩形，起始指针位置取当前鼠标
		pos0 := splitState[key]
		mx, my := MousePos()
		total := float32(0)
		if r, ok := LastFrameOf(key + "-body"); ok {
			if horiz {
				total = r.W - opts.HandleW
			} else {
				total = r.H - opts.HandleW
			}
		}
		splitDrag[key] = splitDragInfo{startMouse: mx, startPos: pos0, total: total, minA: opts.MinA, minB: opts.MinB}
		if !horiz {
			splitDrag[key] = splitDragInfo{startMouse: my, startPos: pos0, total: total, minA: opts.MinA, minB: opts.MinB}
		}
	}
	handle.OnDrag = func(x, y float32) {
		if r, ok := LastFrameOf(key + "-body"); ok {
			mx, my := MousePos()
			_ = mx
			_ = my
			total := r.W - opts.HandleW
			if !horiz {
				total = r.H - opts.HandleW
			}
			if d, ok := splitDrag[key]; ok && total > 0 && d.total != total {
				d.total = total
				splitDrag[key] = d
			}
		}
		mx, my := MousePos()
		apply(mx, my)
	}
	handle.OnDragEnd = func() {
		delete(splitDrag, key)
		RequestRedraw()
	}
	handle.OnClick = func() {
		// 双击复位：用原生点击次数无法判定，这里按「短时间两次点击」近似
		if splitLastClick[key] != 0 && AnimTime()-splitLastClick[key] < 0.32 {
			splitState[key] = opts.Pos
			if opts.PrefsKey != "" {
				SetPrefFloat(opts.PrefsKey, opts.Pos)
			}
			delete(splitLastClick, key)
			RequestRedraw()
			return
		}
		splitLastClick[key] = AnimTime()
	}

	if horiz {
		left := Box(a).St(St().GrowTo(pos).FlexBasis(0).MinWidth(opts.MinA))
		right := Box(b).St(St().GrowTo(1 - pos).FlexBasis(0).MinWidth(opts.MinB))
		container = Row(left, handle, right).St(St().GapOf(0))
	} else {
		top := Box(a).St(St().GrowTo(pos).FlexBasis(0).MinHeight(opts.MinA))
		bottom := Box(b).St(St().GrowTo(1 - pos).FlexBasis(0).MinHeight(opts.MinB))
		container = Column(top, handle, bottom).St(St().GapOf(0))
	}
	return container.K(key + "-body")
}

// splitLastClick 双击复位计时。
var splitLastClick = map[string]float32{}

// splitHandle 分隔条（悬停/按下高亮，拖动光标由平台处理）。
func splitHandle(key string, horiz bool, w float32) *Node {
	dragging := false
	if _, ok := splitDrag[key]; ok {
		dragging = true
	}
	hk := key + splitHandleKey
	hover := IsHovered(hk) || dragging
	var col any = "transparent"
	if hover {
		col = paintWithAlpha(ColPrimary, 0.35)
	}
	bar := Box().St(St().BgC(col))
	if horiz {
		bar.St(St().W(w).Hp(100).MinWidth(w))
		inner := Box().St(St().W(2).H(28).R(1).BgC(ColLine).AbsCenter())
		if hover {
			inner.St(St().BgC(ColPrimary))
		}
		bar = Stack(bar, inner).St(St().W(w).Hp(100).MinWidth(w).NoFocusRing())
	} else {
		bar.St(St().Wp(100).H(w).MinHeight(w))
		inner := Box().St(St().W(28).H(2).R(1).BgC(ColLine).AbsCenter())
		if hover {
			inner.St(St().BgC(ColPrimary))
		}
		bar = Stack(bar, inner).St(St().Wp(100).H(w).MinHeight(w).NoFocusRing())
	}
	return bar
}
