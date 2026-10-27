package ui

import "testing"

// newTestCache 构造测试用字体缓存（真实字体，保证文本度量可用）。
func newTestCache() *fontCache {
	return newFontCache("Microsoft YaHei UI", 1)
}

func near(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.6
}

// TestGridLayout 三列等宽网格：列宽 = (innerW - 2*gap)/3，行高取行内最高项。
func TestGridLayout(t *testing.T) {
	root := Column(
		Grid(
			Box().St(St().H(30)),
			Box().St(St().H(50)),
			Box().St(St().H(20)),
			Box().St(St().H(20)),
		).St(St().ColsN(3).GapOf(10)).K("grid"),
	)
	Layout(root, 320, 400, newTestCache())
	g := findKey(root, "grid")
	if g == nil {
		t.Fatal("grid not found")
	}
	wantCol := float32(320-20) / 3
	// 行高：第一行 50（最高子项），第二行 20，加间距 10 → 80
	if !near(g.Frame.W, 320) || !near(g.Frame.H, 80) {
		t.Fatalf("grid size = %.1fx%.1f, want 320x80", g.Frame.W, g.Frame.H)
	}
	first := g.Children[0]
	if !near(first.Frame.W, wantCol) {
		t.Fatalf("col width = %.1f, want %.1f", first.Frame.W, wantCol)
	}
	if !near(g.Children[1].Frame.X, wantCol+10) {
		t.Fatalf("second col x = %.1f, want %.1f", g.Children[1].Frame.X, wantCol+10)
	}
	// 第二行起始 y = 行高 50 + gap 10
	if !near(g.Children[3].Frame.Y, 60) {
		t.Fatalf("second row y = %.1f, want 60", g.Children[3].Frame.Y)
	}
}

// TestColWidthAuto 固定列宽自动算列数：380 可用宽、150 列宽 → 2 列。
func TestColWidthAuto(t *testing.T) {
	root := Column(
		Grid(Box(), Box(), Box(), Box()).St(St().ColWidth(150).GapOf(10)).K("grid"),
	).St(St().W(380))
	Layout(root, 380, 300, newTestCache())
	g := findKey(root, "grid")
	if int(gridCols(&g.Style, 380)) != 2 {
		t.Fatalf("cols = %d, want 2", gridCols(&g.Style, 380))
	}
	if !near(g.Children[2].Frame.Y, g.Children[0].Frame.H+10) && g.Children[0].Frame.H != 0 {
		t.Fatalf("row 2 y = %.1f", g.Children[2].Frame.Y)
	}
}

// TestWrapLayout 换行：可用宽 300、每项 100、间距 10 → 每行 2 项，共 5 行。
func TestWrapLayout(t *testing.T) {
	items := make([]*Node, 0, 10)
	for i := 0; i < 10; i++ {
		items = append(items, Box().St(St().W(100).H(20)))
	}
	root := Column(
		Wrap(items...).St(St().GapOf(10)).K("wrap"),
	).St(St().W(300))
	Layout(root, 300, 400, newTestCache())
	w := findKey(root, "wrap")
	if !near(w.Frame.W, 300) {
		t.Fatalf("wrap width = %.1f, want 300", w.Frame.W)
	}
	// 5 行 * 20 + 4 * 10 = 140
	if !near(w.Frame.H, 140) {
		t.Fatalf("wrap height = %.1f, want 140", w.Frame.H)
	}
	if !near(w.Children[2].Frame.X, 0) {
		t.Fatalf("third item x = %.1f, want 0 (新行)", w.Children[2].Frame.X)
	}
	if !near(w.Children[2].Frame.Y, 30) {
		t.Fatalf("third item y = %.1f, want 30", w.Children[2].Frame.Y)
	}
}

// TestMasonryLayout 瀑布流：第三项落到最矮列（列 1）。
func TestMasonryLayout(t *testing.T) {
	root := Column(
		Masonry(
			Box().St(St().H(100)),
			Box().St(St().H(40)),
			Box().St(St().H(30)),
		).St(St().ColsN(2).GapOf(10)).K("mason"),
	)
	Layout(root, 410, 500, newTestCache())
	m := findKey(root, "mason")
	colW := float32(410-10) / 2
	if !near(m.Children[1].Frame.X, colW+10) {
		t.Fatalf("item2 x = %.1f, want %.1f", m.Children[1].Frame.X, colW+10)
	}
	if !near(m.Children[2].Frame.Y, 50) {
		t.Fatalf("item3 y = %.1f, want 50", m.Children[2].Frame.Y)
	}
	if !near(m.Frame.H, 100) {
		t.Fatalf("masonry height = %.1f, want 100", m.Frame.H)
	}
}

// TestJustifySpaceBetween 主轴分布：3 个 50 宽子项在 300 宽内均匀分布。
func TestJustifySpaceBetween(t *testing.T) {
	row := Row(
		Box().St(St().W(50).H(10)),
		Box().St(St().W(50).H(10)),
		Box().St(St().W(50).H(10)),
	).St(St().W(300).H(10)).JustifyTo(AlignSpaceBetween).K("just")
	root := Column(row)
	Layout(root, 320, 100, newTestCache())
	j := findKey(root, "just")
	if !near(j.Children[0].Frame.X, 0) || !near(j.Children[1].Frame.X, 125) || !near(j.Children[2].Frame.X, 250) {
		t.Fatalf("space-between x = %.0f/%.0f/%.0f, want 0/125/250",
			j.Children[0].Frame.X, j.Children[1].Frame.X, j.Children[2].Frame.X)
	}
}

// TestJustifyCenterEnd space-evenly 与 end。
func TestJustifyCenterEnd(t *testing.T) {
	even := Row(
		Box().St(St().W(50).H(10)),
		Box().St(St().W(50).H(10)),
	).St(St().W(250).H(10)).JustifyTo(AlignSpaceEvenly).K("even")
	root := Column(even)
	Layout(root, 250, 100, newTestCache())
	e := findKey(root, "even")
	// free = 150，evenly 单元 = 50 → 起点 50，第二项 50+50+50=150
	if !near(e.Children[0].Frame.X, 50) || !near(e.Children[1].Frame.X, 150) {
		t.Fatalf("space-evenly x = %.0f/%.0f, want 50/150", e.Children[0].Frame.X, e.Children[1].Frame.X)
	}
}

// TestStackAbsFill 叠层内的 AbsFill 子项应铺满父内边距区（不被自身测量结果撑宽）。
func TestStackAbsFill(t *testing.T) {
	root := Column(
		Column(
			Stack(
				Box().St(St().AbsFill()),
				Text("底层"),
			).St(St().H(132).Clipped()).K("stk"),
		).St(St().Padding(20)).K("card"),
	).St(St().W(880))
	Layout(root, 880, 721, newTestCache())
	stk := findKey(root, "stk")
	fill := stk.Children[0]
	if !near(stk.Frame.W, 840) || !near(fill.Frame.W, 840) || !near(fill.Frame.H, 132) {
		t.Fatalf("stack=%.1f fill=%.1fx%.1f, want stack 840 fill 840x132", stk.Frame.W, fill.Frame.W, fill.Frame.H)
	}
}

// TestEqualRow 等分栏：可用宽 300、gap 10 → 每项 95。
func TestEqualRow(t *testing.T) {
	root := Column(
		EqualRow(Box().St(St().H(10)), Box().St(St().H(10)), Box().St(St().H(10))).
			St(St().GapOf(10)).K("eq"),
	).St(St().W(300))
	Layout(root, 300, 100, newTestCache())
	q := findKey(root, "eq")
	if !near(q.Children[0].Frame.W, 280.0/3) {
		t.Fatalf("equal width = %.2f, want 93.33", q.Children[0].Frame.W)
	}
}
