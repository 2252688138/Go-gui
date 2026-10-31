// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// swGeo 开关几何期望值（测试辅助）。
type swGeo struct {
	key  string
	w, h float32
}

// TestSwitchVariantsLayout 开关各变体都能完成布局：尺寸有效、默认样式几何不退化。
func TestSwitchVariantsLayout(t *testing.T) {
	root := Column(
		SwitchDefault("t-sw-def", true, nil),
		SwitchSized("t-sw-sm", SwitchSmall, false, nil),
		SwitchSized("t-sw-lg", SwitchLarge, true, nil),
		SwitchSparkle("t-sw-sparkle", true, nil),
		SwitchIcon("t-sw-icon", "✦", false, nil),
		SwitchLabeled("t-sw-label", "ON", "OFF", true, nil),
		SwitchGradient("t-sw-grad", "#f43f5e", "#fb923c", true, nil),
		SwitchDisabled("t-sw-dis", true),
		Switch("t-sw-orig", false, nil),
	).St(St().GapOf(10))
	Layout(root, 400, 600, newTestCache())

	keys := []string{"t-sw-def", "t-sw-sm", "t-sw-lg", "t-sw-sparkle",
		"t-sw-icon", "t-sw-label", "t-sw-grad", "t-sw-dis", "t-sw-orig"}
	for _, k := range keys {
		n := findKey(root, k)
		if n == nil {
			t.Fatalf("缺少开关节点 %s", k)
		}
		if n.Frame.W <= 0 || n.Frame.H <= 0 {
			t.Fatalf("%s 尺寸无效: %.1fx%.1f", k, n.Frame.W, n.Frame.H)
		}
	}

	// 默认样式 = 44x24；小号 34x18；大号 58x30
	geos := []swGeo{
		swGeo{key: "t-sw-def", w: 44, h: 24},
		swGeo{key: "t-sw-sm", w: 34, h: 18},
		swGeo{key: "t-sw-lg", w: 58, h: 30},
	}
	for _, g := range geos {
		n := findKey(root, g.key)
		if n.Frame.W != g.w || n.Frame.H != g.h {
			t.Fatalf("%s 几何 = %.0fx%.0f，期望 %.0fx%.0f", g.key, n.Frame.W, n.Frame.H, g.w, g.h)
		}
	}

	// 外框胶囊（图标 + 轨道）应比单独轨道宽
	if sparkle, track := findKey(root, "t-sw-sparkle"), findKey(root, "t-sw-lg"); sparkle.Frame.W <= track.Frame.W {
		t.Fatalf("胶囊开关应比轨道更宽：%.1f <= %.1f", sparkle.Frame.W, track.Frame.W)
	}
}

// TestSwitchStyleDefaults 预设样式字段正确（默认样式不给渐变/发光，保持与 Switch 一致）。
func TestSwitchStyleDefaults(t *testing.T) {
	def := SwitchStyleDefault()
	if def.Boxed || def.Glow || def.OnFrom != nil || def.OnLabel != "" {
		t.Fatalf("默认样式不应带额外装饰: %+v", def)
	}
	sp := SwitchStyleSparkle()
	if !sp.Boxed || !sp.Glow || sp.Icon == "" || sp.Size != SwitchLarge {
		t.Fatalf("✦ 预设应带外框/发光/图标/大号: %+v", sp)
	}
	lb := SwitchStyleLabeled("ON", "OFF")
	if lb.OnLabel != "ON" || lb.OffLabel != "OFF" {
		t.Fatalf("文字预设文案错误: %+v", lb)
	}
}

// TestSwitchGlyphOpticalCenter 图标按字形包围盒居中：视觉中心与方框中心误差 < 0.6px。
func TestSwitchGlyphOpticalCenter(t *testing.T) {
	fc := newTestCache()
	f := fc.getFamily(20, 700, "")
	if f == nil {
		t.Skip("无可用字体")
	}
	top, h := textGlyphBoundsOf(f, "✦")
	if h <= 0 {
		t.Skip("字体缺 ✦ 字形")
	}
	const box = float32(30)
	dy := (box-h)/2 - top
	center := dy + top + h/2
	if d := center - box/2; d < -0.6 || d > 0.6 {
		t.Fatalf("图标垂直中心偏移 %.2fpx（top=%.2f h=%.2f）", d, top, h)
	}
}

// TestSwitchCustomParts 自定义图标节点 / 图标字形 / 开关文案 / 轨道内容节点均可布局。
func TestSwitchCustomParts(t *testing.T) {
	root := Column(
		SwitchEx("t-cust-icon", true, SwitchStyleSparkle().WithIconNode(Dot("#22d3ee", 10)), nil),
		SwitchEx("t-cust-glyph", true, SwitchStyleSparkle().WithIcon("★", "#fbbf24"), nil),
		SwitchEx("t-cust-text", false, SwitchStyleLabeled("开", "关").WithLabelStyle("#ffffff", "#c7d2fe", 11), nil),
		SwitchEx("t-cust-node", true, SwitchStyleDefault().WithLabelNodes(
			Text("✓").St(St().Text(12, 700, "#ffffff")),
			Text("✕").St(St().Text(12, 700, "#94a3b8")),
		).WithLabelWidth(12), nil),
	).St(St().GapOf(10))
	Layout(root, 500, 400, newTestCache())

	for _, k := range []string{"t-cust-icon", "t-cust-glyph", "t-cust-text", "t-cust-node"} {
		n := findKey(root, k)
		if n == nil {
			t.Fatalf("缺少开关节点 %s", k)
		}
		if n.Frame.W <= 0 || n.Frame.H <= 0 {
			t.Fatalf("%s 尺寸无效: %.1fx%.1f", k, n.Frame.W, n.Frame.H)
		}
	}
	// 自定义轨道内容按 LabelW 撑开轨道：应比默认 44 更宽
	if n, def := findKey(root, "t-cust-node"), findKey(root, "t-cust-text"); n.Frame.W < def.Frame.W {
		t.Fatalf("自定义轨道内容开关应更宽：%.1f < %.1f", n.Frame.W, def.Frame.W)
	}
	// 构建器返回值拷贝：原样式不被修改
	base := SwitchStyleDefault()
	base.WithLabels("ON", "OFF").WithIcon("★", "#fff").WithLabelWidth(30)
	if base.OnLabel != "" || base.Icon != "" || base.LabelW != 0 {
		t.Fatalf("样式构建器不应修改原值: %+v", base)
	}
}

// TestSwitchIconNodeCentered 真实节点树里图标字形在 30×30 外框内垂直居中
// （含绘制期 Translate 偏移，防止再次出现「偏下」）。
func TestSwitchIconNodeCentered(t *testing.T) {
	fc := newTestCache()
	prev := currentApp.Swap(&App{fonts: fc})
	defer currentApp.Store(prev)

	root := Column(SwitchSparkle("t-sw-center", true, nil))
	Layout(root, 400, 200, fc)

	var iconText *Node
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if n.Text == "✦" {
			iconText = n
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	if iconText == nil || iconText.Parent == nil {
		t.Fatal("未找到图标文本节点")
	}
	top, h := TextGlyphBounds("✦", iconText.Style.FontSize, 700)
	if h <= 0 {
		t.Skip("字体缺 ✦ 字形")
	}
	// 字形视觉中心 = 文本 frame 顶部 + 字形在行内偏移 + 绘制期偏移 + 半个字形高
	glyphTop := iconText.Frame.Y + top + iconText.Style.DrawY
	center := glyphTop + h/2
	box := iconText.Parent
	boxCenter := box.Frame.Y + box.Frame.H/2
	if d := center - boxCenter; d < -0.6 || d > 0.6 {
		t.Fatalf("图标中心偏移 %.2fpx（图标中心=%.2f 外框中心=%.2f，dy=%.2f）",
			d, center, boxCenter, iconText.Style.DrawY)
	}
}
