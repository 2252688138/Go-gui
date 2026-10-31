// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com

package ui

import "testing"

// TestColorPickerLayout 取色面板布局回归：
//  1. SV 方块三层（色相底 / 白色横向 / 黑色纵向）都要铺满，渐变端色为全透明也必须保留；
//  2. 色相条 12 段必须横向顺排，不能叠在一起（曾用 Stack 导致只看到一小块红色）；
//  3. HEX 输入框必须有实际宽度（Input 自带 Wp(100)，固定宽需真正生效）。
func TestColorPickerLayout(t *testing.T) {
	fc := newTestCache()
	prev := currentApp.Swap(&App{fonts: fc})
	defer currentApp.Store(prev)

	saved := colorStates
	colorStates = map[string]*ColorState{}
	defer func() { colorStates = saved }()

	st := NewColor("t-color")
	st.SetColor("#6366f1")
	st.Value = "#6366f1"

	root := Stack(
		Column(ColorPicker(st, "#6366f1", nil)).St(St().W(300).Padding(40)),
		ColorLayers(),
	).St(St().W(400).H(700))
	currentApp.Load().root = root
	Layout(root, 400, 700, fc)

	st.Open = true
	root.Children[1] = ColorLayers()
	if root.Children[1] == nil {
		t.Fatal("打开状态应返回取色浮层")
	}
	Layout(root, 400, 700, fc)

	// 1. SV 方块
	sv := findKey(root, "t-color-sv")
	if sv == nil {
		t.Fatal("缺少 SV 方块节点")
	}
	const svW, svH = float32(200), float32(140)
	if len(sv.Children) < 3 {
		t.Fatalf("SV 方块应有色相底 + 白 + 黑三层，实际 %d", len(sv.Children))
	}
	for i, c := range sv.Children[:3] {
		if c.Frame.W != svW || c.Frame.H != svH {
			t.Fatalf("SV 第 %d 层未铺满: %.0fx%.0f", i, c.Frame.W, c.Frame.H)
		}
	}
	white := sv.Children[1]
	if !white.Style.BgToSet || white.Style.Bg.A <= 0 || white.Style.BgTo.A != 0 {
		t.Fatalf("白色遮罩渐变被样式合并吞掉: BgSet=%v BgToSet=%v Bg=%+v BgTo=%+v",
			white.Style.BgSet, white.Style.BgToSet, white.Style.Bg, white.Style.BgTo)
	}
	black := sv.Children[2]
	if !black.Style.BgToSet || black.Style.BgTo.A <= 0 {
		t.Fatalf("黑色遮罩渐变丢失: BgToSet=%v BgTo=%+v", black.Style.BgToSet, black.Style.BgTo)
	}
	if black.Style.BgAngle != 180 {
		t.Fatalf("黑色遮罩应纵向（180 = 向下），实际 %.0f", black.Style.BgAngle)
	}

	// 2. 色相条：12 段横向铺满且互不重叠
	hue := findKey(root, "t-color-hue")
	if hue == nil {
		t.Fatal("缺少色相条节点")
	}
	segs := make([]*Node, 0, 12)
	for _, c := range hue.Children {
		if !c.Style.PosAbs {
			segs = append(segs, c)
		}
	}
	if len(segs) != 12 {
		t.Fatalf("色相条应有 12 段，实际 %d", len(segs))
	}
	var sum float32
	for i, c := range segs {
		if c.Frame.H <= 0 {
			t.Fatalf("色相条第 %d 段高度无效: %.1f", i, c.Frame.H)
		}
		if i > 0 && c.Frame.X < segs[i-1].Frame.X+segs[i-1].Frame.W-0.5 {
			t.Fatalf("色相条第 %d 段与前段重叠（x=%.1f，前段右边缘 %.1f）",
				i, c.Frame.X, segs[i-1].Frame.X+segs[i-1].Frame.W)
		}
		sum += c.Frame.W
	}
	if sum < svW-0.5 || sum > svW+0.5 {
		t.Fatalf("色相条各段宽度合计 %.1f，应约等于 %.0f", sum, svW)
	}

	// 3. HEX 输入框：宽度有效且不溢出面板
	hex := findKey(root, "t-color-hex")
	if hex == nil {
		t.Fatal("缺少 HEX 输入框")
	}
	if hex.Frame.W < 60 || hex.Frame.H <= 0 {
		t.Fatalf("HEX 输入框尺寸无效: %.1fx%.1f", hex.Frame.W, hex.Frame.H)
	}
	if right := hex.Frame.X + hex.Frame.W; right > sv.Frame.X+svW+0.5 {
		t.Fatalf("HEX 输入框溢出面板：右边缘 %.1f > %.1f", right, sv.Frame.X+svW)
	}
}
