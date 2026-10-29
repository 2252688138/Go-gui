package ui

import (
	"math"
	"strconv"

	"gogui/internal/geom"
	"gogui/internal/gpu"
)
// ChartType 图表类型。
type ChartType uint8

const (
	// ChartLine 折线图。
	ChartLine ChartType = iota
	// ChartArea 面积图。
	ChartArea
	// ChartBar 柱状图。
	ChartBar
	// ChartPie 饼图。
	ChartPie
	// ChartDonut 环形图。
	ChartDonut
	// ChartRadar 雷达图。
	ChartRadar
	// ChartScatter 散点图。
	ChartScatter
)

// Series 一条数据系列。
//
// 折线 / 面积 / 柱状 / 雷达：Values 为等距采样值（雷达图按 Labels 的轴顺序取值）；
// 散点：Values 为 Y，X 为可选横坐标（为空时用下标）。
// 饼图 / 环形图：每个 Series 是一个扇区，取其 Values 之和。
type Series struct {
	Name   string
	Color  string
	Values []float64
	X      []float64
}

// Chart 图表数据与配置，由业务持有（与 Player 一样不要每帧新建）。
//
// 性能与内存：图表按「节点 Key + 尺寸 + 缩放 + 数据指纹」缓存一张 RGBA 纹理。
// 数据不变时每帧只发 1 个 draw call、零 CPU 光栅化；缓存总预算 24MB，
// 超出按最近使用淘汰，纹理随窗口销毁。
type Chart struct {
	Type ChartType

	Series []Series
	// Labels X 轴标签 / 饼图扇区名 / 雷达轴名。
	Labels []string

	Title string
	Unit  string

	ShowLegend bool
	ShowGrid   bool
	ShowAxis   bool
	// Stacked 柱状 / 面积按系列堆叠。
	Stacked bool
	// Smooth 折线平滑（Catmull-Rom 插值，采样后仍是折线段）。
	Smooth bool
	// Donut 环形图内径比例（0~0.9，仅 ChartDonut 生效）。
	Donut float32
	// Min / Max 纵轴范围；Max 为 0 表示自动（Min 可为负值）。
	Min, Max float64
	// YTicks 纵轴刻度数（默认 4）。
	YTicks int
	// Padding 画布内边距（默认 14）。
	Padding float32
	// OnPick 点击柱 / 数据点 / 扇区回调（series 系列下标，index 数据点下标）。
	OnPick func(series, index int)

	applied bool
	cache   chartCache
}

// chartPalette 默认系列配色（深浅底都够亮）。
var chartPalette = []string{
	"#6366f1", "#22d3ee", "#34d399", "#fbbf24",
	"#f472b6", "#a78bfa", "#38bdf8", "#fb7185",
}

// colorOf 第 i 条系列的画笔：显式 Color 优先，否则取调色板。
func (c *Chart) colorOf(i int) gpu.Paint {
	if i >= 0 && i < len(c.Series) && c.Series[i].Color != "" {
		return C(c.Series[i].Color)
	}
	if len(chartPalette) == 0 {
		return C(ColPrimary)
	}
	return C(chartPalette[i%len(chartPalette)])
}

// applyDefaults 补齐默认开关（零值 Chart 也开箱可用）。
func (c *Chart) applyDefaults() {
	if c.applied {
		return
	}
	c.applied = true
	if c.YTicks <= 0 {
		c.YTicks = 4
	}
	if c.Padding <= 0 {
		c.Padding = 14
	}
	c.ShowLegend, c.ShowGrid, c.ShowAxis = true, true, true
}

// NewChart 新建图表（已填默认开关与刻度数）。
func NewChart(t ChartType, labels []string, series ...Series) *Chart {
	c := &Chart{Type: t, Labels: labels, Series: series}
	c.applyDefaults()
	if c.Type == ChartDonut && c.Donut <= 0 {
		c.Donut = 0.58
	}
	return c
}

// ChartView 把图表包成节点（默认高度 280、可被 St() 覆盖）。
//
// 可选交互：chart.OnPick 设置后，鼠标悬浮单位高亮并显示数值提示，点击回调命中项。
func ChartView(key string, c *Chart) *Node {
	n := &Node{Kind: KChart, Key: key, Chart: c, Style: Style{Opacity: 1}}
	n.Style.Height = 280
	n.OnClickAt = func(x, y float32) {
		if c == nil || c.OnPick == nil || !c.cache.hitOK {
			return
		}
		c.OnPick(c.cache.hitS, c.cache.hitI)
	}
	return n
}

// LineChart 折线图。
func LineChart(key string, labels []string, series ...Series) *Node {
	return ChartView(key, NewChart(ChartLine, labels, series...))
}

// AreaChart 面积图。
func AreaChart(key string, labels []string, series ...Series) *Node {
	return ChartView(key, NewChart(ChartArea, labels, series...))
}

// BarChart 柱状图（堆叠用 chart.Stacked = true）。
func BarChart(key string, labels []string, series ...Series) *Node {
	return ChartView(key, NewChart(ChartBar, labels, series...))
}

// PieChart 饼图：values 每项一个扇区，labels 为扇区名（可少传）。
func PieChart(key string, values []float64, labels []string) *Node {
	return ChartView(key, NewChart(ChartPie, labels, PieSeries(values, labels)...))
}

// DonutChart 环形图（内径 58%）。
func DonutChart(key string, values []float64, labels []string) *Node {
	return ChartView(key, NewChart(ChartDonut, labels, PieSeries(values, labels)...))
}

// RadarChart 雷达图。
func RadarChart(key string, labels []string, series ...Series) *Node {
	return ChartView(key, NewChart(ChartRadar, labels, series...))
}

// ScatterChart 散点图。
func ScatterChart(key string, series ...Series) *Node {
	return ChartView(key, NewChart(ChartScatter, nil, series...))
}

// PieSeries 把数值切片摊成扇区系列（名称取 labels[i]，颜色走调色板）。
// 饼图 / 环形图可先用它构造 Series，再交给 NewChart。
func PieSeries(values []float64, labels []string) []Series {
	out := make([]Series, 0, len(values))
	for i, v := range values {
		name := "扇区 " + strconv.Itoa(i+1)
		if i < len(labels) && labels[i] != "" {
			name = labels[i]
		}
		out = append(out, Series{Name: name, Values: []float64{v}})
	}
	return out
}

// dataMax 数据最大值（堆叠时按堆叠和）。
func (c *Chart) dataMax() float64 {
	m := 0.0
	if c.Stacked && (c.Type == ChartBar || c.Type == ChartArea) {
		n := c.points()
		for i := 0; i < n; i++ {
			sum := 0.0
			for j := range c.Series {
				if i < len(c.Series[j].Values) {
					sum += c.Series[j].Values[i]
				}
			}
			if sum > m {
				m = sum
			}
		}
		return m
	}
	for _, s := range c.Series {
		for _, v := range s.Values {
			if v > m {
				m = v
			}
		}
	}
	return m
}

// points 采样点数量（取最长系列）。
func (c *Chart) points() int {
	n := len(c.Labels)
	for _, s := range c.Series {
		if len(s.Values) > n {
			n = len(s.Values)
		}
	}
	return n
}

// niceMax 把轴上限抬到「好看」的刻度（1/2/2.5/5 × 10^n）。
func niceMax(v float64, ticks int) float64 {
	if v <= 0 {
		return 1
	}
	if ticks <= 0 {
		ticks = 4
	}
	raw := v / float64(ticks)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	norm := raw / mag
	step := 1.0
	switch {
	case norm <= 1:
		step = 1
	case norm <= 2:
		step = 2
	case norm <= 2.5:
		step = 2.5
	case norm <= 5:
		step = 5
	default:
		step = 10
	}
	return step * mag * float64(ticks)
}

// fmtNum 轴刻度 / 提示里的数值格式化（万 / 亿 + 去尾零）。
func fmtNum(v float64) string {
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	var s string
	switch {
	case v >= 1e8:
		s = trimZero(v/1e8, 1) + "亿"
	case v >= 1e4:
		s = trimZero(v/1e4, 1) + "万"
	case v >= 100:
		s = strconv.FormatFloat(v, 'f', 0, 64)
	case v >= 1:
		s = trimZero(v, 1)
	default:
		s = trimZero(v, 2)
	}
	return sign + s
}

// trimZero 保留 dec 位小数并去掉末尾 0。
func trimZero(v float64, dec int) string {
	s := strconv.FormatFloat(v, 'f', dec, 64)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

// chartCache 一张图表的纹理缓存（随窗口销毁）。
//
// 同时记录本帧几何与悬浮命中，供点击回调（OnPick）复用，
// 避免回调里重新跑一遍布局。
type chartCache struct {
	tex   *gpu.Texture
	key   uint64
	w, h  int
	bytes int64
	stamp int64

	plot geom.Rect
	pieC fpt
	pieR float32

	// 本帧布局与节点原点（点击回调复用，避免重算布局）
	lg    chartLayout
	lgOK  bool
	nodeX float32
	nodeY float32

	hitS, hitI int
	hitOK     bool
}
