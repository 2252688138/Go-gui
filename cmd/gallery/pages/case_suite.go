// Go-gui — gallery 页面拆分：组件套件（按类型拆成独立卡片，供分类页组合）
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"fmt"
	"gogui/ui"
	"sync"
	"time"
)

// 组件套件演示卡片：原本集中在「组件套件」单页，现按类型拆开，
// 分别由 基础组件 / 表单弹窗 / 列表表格 / 系统操作 / 文本与富文本 页面组合使用。

var suiteOnce sync.Once

var (
	suiteAC    = ui.NewAutoComplete("suite-ac")
	suiteACVal = ""

	suiteTags = []string{"Go", "UI"}

	suiteRate float32 = 3.5

	suiteLo, suiteHi float32 = 0.2, 0.8

	suiteKey = ""

	suiteTime = ui.NewTime("suite-time")

	suiteCascade = ui.NewCascade("suite-cascade")
	suitePath    []string

	suiteColor = ui.NewColor("suite-color")
	suiteHex   = "#6366f1"

	suiteUpload = []ui.UploadFile{
		{Name: "report.pdf", Progress: 1},
		{Name: "demo.mp4", Progress: 0.42},
		{Name: "broken.zip", Progress: 0.7, Err: "网络中断"},
	}

	suiteSplitPos = ui.SplitPos("suite-split")

	suiteVListH = float32(320)

	suiteSeg = "all"

	suiteCal    = ui.NewCalendar("suite-cal")
	suiteCalVal string

	suiteTour = ui.NewTour("suite-tour")

	suiteRows []string

	suiteNotice int

	suiteDoc = suiteInvoice{}

	suiteGridName  = "gogui"
	suiteGridSize  = "13"
	suiteGridDark  = true
	suiteGridTheme = "dark"

	suiteEditor *ui.RichEditorState
)

// suiteInvoice 打印示例文档。
type suiteInvoice struct{}

func (suiteInvoice) Pages() int { return 2 }

func (suiteInvoice) DrawPage(c *ui.PrintCanvas, page int) {
	c.Text(64, 64, 22, "GoGUI 打印预览", "#111111")
	c.Line(64, 96, 730, 96, 1, "#cccccc")
	for i := 0; i < 8; i++ {
		c.Text(64, 120+float32(i)*40, 14, fmt.Sprintf("第 %d 页 · 第 %d 行示例内容", page+1, i+1), "#333333")
		c.Line(64, 138+float32(i)*40, 730, 138+float32(i)*40, 0.5, "#eeeeee")
	}
	c.Rect(64, 980, 180, 40, 1, "#6366f1", "rgba(99,102,241,0.12)")
	c.Text(84, 992, 13, "合计：￥1,024.00", "#6366f1")
}

// suiteInit 注册命令与生成演示数据（首次构建任一卡片时执行一次）。
func suiteInit() {
	suiteOnce.Do(func() {
		ui.RegisterCommand(ui.Command{ID: "suite.notify", Title: "弹出一条通知", Group: "套件", Run: func() {
			suiteNotice++
			ui.Notify(ui.Notice{ID: fmt.Sprintf("n-%d", suiteNotice), Kind: ui.NoticeSuccess,
				Title: "任务完成", Body: "组件套件演示通知", Duration: 5})
		}})
		ui.RegisterCommand(ui.Command{ID: "suite.tour", Title: "开始引导 Tour", Group: "套件", Run: func() { suiteTour.Start() }})
		ui.RegisterCommand(ui.Command{ID: "suite.print", Title: "打印预览示例", Group: "套件", Run: func() {
			ui.ToastShow("打印预览在下方展示")
		}})
		if suiteRows == nil {
			suiteRows = make([]string, 200)
			for i := range suiteRows {
				suiteRows[i] = fmt.Sprintf("任务 #%03d", i+1)
			}
		}
		if suiteEditor == nil {
			suiteEditor = ui.NewRichEditor("suite-editor")
			suiteEditor.Insert("富文本编辑器：支持粗体、斜体、下划线、行内代码与撤销。")
		}
	})
	_ = suiteSplitPos
}

// suiteCmdCard 命令面板 / 通知中心（系统级浮层能力）。
func suiteCmdCard() *ui.Node {
	suiteInit()
	return ui.Card(
		ui.Title("命令面板 · 通知中心", "Ctrl+K 打开命令面板；通知支持按钮、进度与悬停暂停"),
		ui.Row(
			ui.Btn("打开命令面板", ui.BtnPrimary, func() { ui.OpenCommandPalette() }),
			ui.Btn("发通知", ui.BtnSecondary, func() {
				suiteNotice++
				ui.Notify(ui.Notice{ID: fmt.Sprintf("n-%d", suiteNotice), Kind: ui.NoticeInfo,
					Title: "正在同步", Body: "3 个文件待上传", Progress: 0.4,
					Buttons: []ui.NoticeButton{{Label: "查看", Run: func() { ui.ToastShow("打开任务列表") }}}})
			}),
			ui.Btn("开始引导", ui.BtnGhost, func() { suiteTour.Start() }),
		).St(ui.St().GapOf(10)),
	)
}

// suiteVListCard 虚拟列表（列表类）。
func suiteVListCard() *ui.Node {
	suiteInit()
	rows := ui.VList("suite-vlist", 100000, 36, suiteVListH, func(i int) *ui.Node {
		rowBg := ui.ColRow
		if i%2 == 1 {
			rowBg = ui.ColRowAlt
		}
		return ui.Row(
			ui.Text(fmt.Sprintf("%06d", i+1)).St(ui.St().Text(11, 500, ui.ColTextMuted).W(70)),
			ui.Text(fmt.Sprintf("虚拟列表第 %d 行 · 只渲染可见区域", i+1)).St(ui.St().Text(12, 400, ui.ColTextDim)),
			ui.Spacer(),
			ui.Tag("ready", ui.ColSuccess),
		).St(ui.St().GapOf(10).PadXY(12, 0).H(36).BgC(rowBg))
	})
	return ui.Card(
		ui.Row(
			ui.Title("虚拟列表 VList", "10 万行只创建可见行").Grow(1),
			ui.Text("100000 行").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).AlignTo(ui.AlignCenter),
		rows,
	)
}

// suiteStatesCard 加载 / 空 / 错误三态（列表与内容区通用）。
func suiteStatesCard() *ui.Node {
	suiteInit()
	return ui.Card(
		ui.Title("加载 / 空 / 错误三态", "AsyncSection + Skeleton + LoadingOverlay"),
		ui.Grid(
			ui.Column(
				ui.SkeletonBlock(160, 12),
				ui.SkeletonText(3, 200),
			).St(ui.St().GapOf(8)),
			ui.AsyncSection(true, "", false, nil, nil),
			ui.EmptyState("", "没有新消息", "收到的通知会出现在这里", "刷新", func() { ui.ToastShow("已刷新") }),
			ui.ResultPage("error", "上传失败", "网络中断，请重试",
				ui.Btn("重试", ui.BtnPrimary, func() { ui.ToastShow("重试中…") })),
		).St(ui.St().ColsN(4).GapOf(16)),
	)
}

// suiteRichCard 富文本显示 + 富文本编辑器（文本类）。
func suiteRichCard() *ui.Node {
	suiteInit()
	rich := ui.RichText([]ui.TextSpan{
		{Text: "富文本 span 树：", Weight: 700},
		{Text: "粗体、", Weight: 700, Color: ui.ColPrimary},
		{Text: "斜体占位、", Underline: true},
		{Text: "行内代码 ", Color: ui.ColTextDim},
		{Text: "go build ./...", Code: true},
		{Text: "，以及可点击的 "},
		{Text: "链接", Link: "demo", Underline: true},
		{Text: "。Emoji 由字体回退链处理：🎉🚀"},
	}, ui.RichTextOptions{Size: 13}).OnSpanClickEvt(func(sp ui.TextSpan) {
		if sp.Link != "" {
			ui.ToastShow("点击了链接：" + sp.Link)
		}
	})
	return ui.Card(
		ui.Title("富文本显示 RichText", "测量 / 绘制 / 命中复用同一份行内片段数据"),
		rich.St(ui.St().MaxWidth(720)),
		ui.RichToolbar(suiteEditor, nil),
		ui.RichEditor(suiteEditor, 150, nil),
	)
}

// suiteInputCard 高级输入类组件（表单类）。
func suiteInputCard() *ui.Node {
	suiteInit()
	return ui.Card(
		ui.Title("高级输入组件", "AutoComplete / TagInput / Rate / RangeSlider / KeyCapture / Time / Cascader / Color / Upload"),
		ui.Grid(
			ui.FormItem("自动完成", false, "", ui.AutoComplete(suiteAC,
				[]string{"Go", "Golang", "GoGUI", "Goroutine", "GraphQL"}, suiteACVal, "输入 G…",
				func(v string) { suiteACVal = v })),
			ui.FormItem("标签输入", false, "", ui.TagInput("suite-tags", suiteTags, "回车添加",
				func(v []string) { suiteTags = v })),
			ui.FormItem("评分", false, "", ui.Rate("suite-rate", suiteRate, 5, true,
				func(v float32) { suiteRate = v })),
			ui.FormItem("区间", false, "", ui.RangeSlider("suite-range", suiteLo, suiteHi,
				func(lo, hi float32) { suiteLo, suiteHi = lo, hi })),
			ui.FormItem("快捷键录制", false, "", ui.KeyCapture("suite-key", suiteKey,
				func(v string) { suiteKey = v })),
			ui.FormItem("时间", false, "", ui.TimePicker(suiteTime, suiteTime.TimeValue(), nil)),
			ui.FormItem("级联", false, "", ui.Cascader(suiteCascade, suiteRegions(), suitePath, "请选择地区",
				func(v []string) { suitePath = v })),
			ui.FormItem("颜色", false, "", ui.ColorPicker(suiteColor, suiteHex,
				func(v string) { suiteHex = v })),
		).St(ui.St().ColsN(2).GapOf(14)),
		ui.FormItem("上传", false, "", ui.Upload("suite-upload", suiteUpload, ui.UploadOptions{
			Filter: "所有文件|*.*",
		}, func() {
			suiteUpload = append(suiteUpload, ui.UploadFile{Name: "new-file.txt", Progress: 0.15})
			ui.ToastShow("已添加文件")
		}, func(i int) {
			if i >= 0 && i < len(suiteUpload) {
				suiteUpload = append(suiteUpload[:i], suiteUpload[i+1:]...)
			}
		}, func(i int) {
			if i >= 0 && i < len(suiteUpload) {
				suiteUpload[i].Err = ""
				suiteUpload[i].Progress = 0.3
			}
		})),
	)
}

// suiteDisplayCard 数据展示类组件（基础组件页的数据展示分组）。
func suiteDisplayCard() *ui.Node {
	suiteInit()
	slides := []*ui.Node{
		ui.Stack(
			ui.Box().St(ui.St().AbsFill().BgGrad("#6366f1", "#8b5cf6", 120)),
			ui.Text("轮播图 1 / 3").St(ui.St().AbsFill().Text(20, 700, "#ffffff").
				CenterYOn().TAlign(ui.AlignCenter)),
		).St(ui.St().Hp(100)),
		ui.Stack(
			ui.Box().St(ui.St().AbsFill().BgGrad("#0ea5e9", "#22d3ee", 120)),
			ui.Text("轮播图 2 / 3").St(ui.St().AbsFill().Text(20, 700, "#ffffff").
				CenterYOn().TAlign(ui.AlignCenter)),
		).St(ui.St().Hp(100)),
		ui.Stack(
			ui.Box().St(ui.St().AbsFill().BgGrad("#f59e0b", "#ef4444", 120)),
			ui.Text("轮播图 3 / 3").St(ui.St().AbsFill().Text(20, 700, "#ffffff").
				CenterYOn().TAlign(ui.AlignCenter)),
		).St(ui.St().Hp(100)),
	}
	return ui.Card(
		ui.Title("数据展示", "Carousel / Timeline / Steps / Segmented / Descriptions / Statistic / Calendar / AvatarGroup"),
		ui.Carousel("suite-carousel", slides, ui.CarouselOptions{Height: 150, Autoplay: true,
			Loop: true, Arrows: true, Dots: true}),
		ui.Steps([]ui.StepItem{
			{Title: "填写信息", Sub: "1 分钟"},
			{Title: "验证邮箱"},
			{Title: "完成"},
		}, 1),
		ui.Segmented([]string{"all", "todo", "done"}, []string{"全部", "待办", "已完成"},
			suiteSeg, func(v string) { suiteSeg = v }),
		ui.Grid(
			ui.Statistic("users", "活跃用户", 12345, ui.StatOptions{Animate: true, Trend: 12.4, Suffix: " 人"}),
			ui.Statistic("revenue", "本月收入", 98765.4, ui.StatOptions{Animate: true, Prefix: "￥", Decimals: 2}),
			ui.Statistic("latency", "平均延迟", 86, ui.StatOptions{Animate: true, Suffix: " ms", Trend: -4.1}),
		).St(ui.St().ColsN(3).GapOf(16)),
		ui.Descriptions([]ui.KV{
			{K: "项目", V: "GoGUI"},
			{K: "版本", V: "组件套件"},
			{K: "渲染", V: "OpenGL / 超采样"},
			{K: "平台", V: "Windows · Linux · macOS · Web"},
		}, 2),
		ui.Row(
			ui.Calendar(suiteCal, []ui.CalendarEvent{
				{Date: time.Now().Format("2006-01-02"), Label: "发布"},
			}, suiteCalVal, func(v string) { suiteCalVal = v }).St(ui.St().W(330)),
			ui.Column(
				ui.Timeline([]ui.TimelineItem{
					{Time: "09:12", Title: "创建仓库", Sub: "初始化项目结构"},
					{Time: "10:40", Title: "接入渲染层", Sub: "OpenGL + 超采样", Color: ui.ColInfo},
					{Time: "14:05", Title: "发布 v1.0", Color: ui.ColSuccess},
				}),
				ui.AvatarGroup([]ui.AvatarItem{
					{Initial: "S", Color: "#6366f1"},
					{Initial: "J", Color: "#22d3ee"},
					{Initial: "M", Color: "#f472b6"},
					{Initial: "K", Color: "#34d399"},
					{Initial: "L", Color: "#fbbf24"},
				}, 4),
			).St(ui.St().GapOf(16)).Grow(1),
		).St(ui.St().GapOf(20)),
	)
}

// suiteTableCard 表格增强 / 属性表 / 拖拽重排（列表类）。
func suiteTableCard() *ui.Node {
	suiteInit()
	return ui.Card(
		ui.Title("表格增强 · 属性表 · 拖拽重排", "虚拟滚动 / 分组 / 合计 / 列宽拖拽 / 通用重排"),
		ui.TableV(ui.TableVConfig{
			Key: "suite-table", Headers: []string{"#", "名称", "状态", "耗时"},
			Widths: []float32{80, 0, 120, 100},
			RowH:   34, Height: 240, Count: 10000,
			Groups: []ui.TableGroup{
				{Label: "进行中", Start: 0, Count: 120},
				{Label: "已完成", Start: 120, Count: 400},
			},
			Resizable: true,
			SortCol:   0,
			OnSort:    func(col int) { ui.ToastShow(fmt.Sprintf("按第 %d 列排序", col+1)) },
			Row: func(i int) []*ui.Node {
				return []*ui.Node{
					ui.Text(fmt.Sprintf("%d", i+1)).St(ui.St().Text(11.5, 500, ui.ColTextMuted)),
					ui.Text(fmt.Sprintf("任务 %d", i+1)).St(ui.St().Text(12, 400, ui.ColText)),
					ui.Tag([]string{"运行中", "已完成", "失败"}[i%3],
						[]string{ui.ColInfo, ui.ColSuccess, ui.ColDanger}[i%3]),
					ui.Text(fmt.Sprintf("%d ms", 20+i%400)).St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
				}
			},
			Footer: []*ui.Node{
				ui.Text("合计").St(ui.St().Text(12, 600, ui.ColText)),
				ui.Spacer(),
				ui.Text("10,000 行").St(ui.St().Text(12, 500, ui.ColSuccess)),
			},
		}),
		ui.Grid(
			ui.PropertyGrid("suite-grid", []ui.Prop{
				{Key: "name", Label: "名称", Group: "常规", Kind: ui.PropText, Value: suiteGridName},
				{Key: "size", Label: "字号", Group: "常规", Kind: ui.PropNumber, Value: suiteGridSize},
				{Key: "dark", Label: "深色", Group: "外观", Kind: ui.PropBool, Value: boolStr(suiteGridDark)},
				{Key: "theme", Label: "主题", Group: "外观", Kind: ui.PropEnum, Value: suiteGridTheme,
					Options: []string{"dark", "light"}},
				{Key: "color", Label: "强调色", Group: "外观", Kind: ui.PropColor, Value: suiteHex},
			}, func(k, v string) {
				switch k {
				case "name":
					suiteGridName = v
				case "size":
					suiteGridSize = v
				case "dark":
					suiteGridDark = v == "1"
				case "theme":
					suiteGridTheme = v
				case "color":
					suiteHex = v
				}
			}),
			ui.Column(
				ui.Text("拖拽重排（按住行拖动松手）").St(ui.St().Text(11.5, 500, ui.ColTextMuted)),
				ui.ReorderList(ui.ReorderListOptions{
					Key: "suite-reorder", Count: len(suiteRows), RowH: 30, Height: 220,
					Row: func(i int) *ui.Node {
						return ui.Row(
							ui.Text("⠿").St(ui.St().Text(12, 600, ui.ColPlaceholder)),
							ui.Text(suiteRows[i]).St(ui.St().Text(12, 400, ui.ColTextDim)),
						).St(ui.St().GapOf(10).PadXY(8, 0).H(30).BgC(ui.ColBgSoft)).AlignTo(ui.AlignCenter)
					},
					OnMove: func(from, to int) {
						suiteRows = ui.MoveSlice(suiteRows, from, to)
					},
				}),
			).St(ui.St().GapOf(8)),
		).St(ui.St().ColsN(2).GapOf(16)),
	)
}

// suiteDeskCard 桌面能力（系统操作）：多显示器 / 截屏 / 剪贴板 / 打印。
func suiteDeskCard() *ui.Node {
	suiteInit()
	return ui.Card(
		ui.Title("桌面能力", "多显示器 / 截屏 / 剪贴板监听 / 打印 / 单实例"),
		suiteMonitorBlock(),
		ui.Row(
			ui.Btn("开始剪贴板监听", ui.BtnSecondary, func() {
				ui.WatchClipboard(func(info ui.ClipboardInfo) {
					ui.Notify(ui.Notice{ID: "clip", Kind: ui.NoticeInfo, Title: "剪贴板变化",
						Body:     fmt.Sprintf("文本 %v / 图片 %v / 文件 %v", info.HasText, info.HasImage, info.HasFiles),
						Duration: 3})
				})
				ui.ToastShow("已开始监听剪贴板")
			}),
			ui.Btn("打印预览", ui.BtnPrimary, func() { ui.ToastShow("下面就是打印预览") }),
		).St(ui.St().GapOf(10)),
		ui.PrintPreview("suite-print", suiteDoc, 0.55).St(ui.St().H(520)),
	)
}

// SuiteLayers 套件相关浮层（挂在根 Stack 末尾；无浮层时返回 nil）。
func SuiteLayers() *ui.Node {
	layers := []*ui.Node{
		ui.AutoCompleteLayers(),
		ui.TimeLayers(),
		ui.CascaderLayers(),
		ui.ColorLayers(),
		ui.CommandPaletteLayer(),
		ui.TourLayer(suiteTour, []ui.TourStep{
			{Title: "欢迎使用组件套件", Body: "这是分步引导示例，可以一步步了解页面结构。"},
			{Title: "虚拟列表", Body: "10 万行数据只渲染可见行，滚动依然流畅。"},
			{Title: "命令面板", Body: "按 Ctrl+K 可以随时打开命令面板。"},
		}),
		ui.NotifyLayer(),
	}
	// 没有任何浮层时返回 nil：不能返回一个全空的 AbsFill Stack，
	// 否则它会盖住整窗并吃光点击（含标题栏按钮），表现为“只能滚动、点不动”。
	has := false
	for _, n := range layers {
		if n != nil {
			has = true
			break
		}
	}
	if !has {
		return nil
	}
	return ui.Stack(layers...).St(ui.St().AbsFill())
}

// suiteRegions 级联示例数据。
func suiteRegions() []ui.CascadeOption {
	return []ui.CascadeOption{
		{Value: "zj", Label: "浙江省", Children: []ui.CascadeOption{
			{Value: "hz", Label: "杭州市", Children: []ui.CascadeOption{
				{Value: "xh", Label: "西湖区"},
				{Value: "yh", Label: "余杭区"},
			}},
			{Value: "nb", Label: "宁波市", Children: []ui.CascadeOption{
				{Value: "hs", Label: "海曙区"},
			}},
		}},
		{Value: "gd", Label: "广东省", Children: []ui.CascadeOption{
			{Value: "gz", Label: "广州市", Children: []ui.CascadeOption{
				{Value: "th", Label: "天河区"},
			}},
			{Value: "sz", Label: "深圳市", Children: []ui.CascadeOption{
				{Value: "nss", Label: "南山区"},
			}},
		}},
	}
}

func boolStr(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// suiteMonitorBlock 显示器信息块。
func suiteMonitorBlock() *ui.Node {
	ms := ui.Monitors()
	lines := make([]*ui.Node, 0, len(ms)+1)
	if len(ms) == 0 {
		lines = append(lines, ui.Text("当前平台未报告显示器（非 Windows 或未实现）").
			St(ui.St().Text(11.5, 400, ui.ColTextMuted)))
	}
	for _, m := range ms {
		flag := ""
		if m.Primary {
			flag = "（主）"
		}
		lines = append(lines, ui.KeyValue(
			fmt.Sprintf("显示器 %d%s", m.Index+1, flag),
			fmt.Sprintf("%dx%d @ %.0f%%  工作区 %d,%d %dx%d",
				m.W, m.H, m.Scale*100, m.WorkX, m.WorkY, m.WorkW, m.WorkH)))
	}
	return ui.Column(lines...).St(ui.St().GapOf(4))
}
