// Go-gui — gallery 页面拆分：系统能力
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"fmt"
	"gogui/ui"
	"path/filepath"
	"strconv"
	"strings"
)

// ---- 系统能力演示：图片 / 毛玻璃 / Emoji / 动画 / 滚动条 / IME / 多窗口 ----

var (
	sysAnimOn    bool
	sysImgPath   string
	sysOpened    int
	sysTray      *ui.Tray
	sysMsg       = "点击下方按钮体验文件对话框 / 多窗口 / 托盘"
	sysScrollRow = -1
	sysIme       = ""

	// 拖入文件 / 系统操作演示状态
	sysDropPaths  []string
	sysDropMsg    = "把文件或文件夹拖到上面的框里松手，这里会显示结果"
	sysClipText   = ""
	sysTopMost    bool
	sysHotkeyID   int32
	sysWakeLock   bool
	sysHotkeyNote = "未注册"
)

// sysSampleImage 找一个系统自带壁纸作为图片演示素材。
func sysSampleImage() string {
	if sysImgPath != "" {
		return sysImgPath
	}
	cands := []string{
		`C:\Windows\Web\Wallpaper\Windows\img0.jpg`,
		`C:\Windows\Web\Wallpaper\Windows\img19.jpg`,
		`C:\Windows\Web\Wallpaper\Theme1\img1.jpg`,
		`C:\Windows\Web\Wallpaper\ThemeA\img1.jpg`,
		`C:\Windows\Web\4K\Wallpaper\Windows\img0_3840x2160.jpg`,
	}
	for _, p := range cands {
		if w, _ := ui.ImageSize(p); w > 0 {
			return p
		}
	}
	return ""
}

func sysSection(title, sub string, body *ui.Node) *ui.Node {
	return ui.Column(
		ui.Title(title, sub),
		body,
	).St(ui.St().GapOf(10).PadOnly(0, 0, 10, 0))
}

// sysPickImage 从拖入的路径里挑一张能解码的图片（自动填到图片预览区）。
func sysPickImage(paths []string) string {
	for _, p := range paths {
		switch strings.ToLower(filepath.Ext(p)) {
		case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp":
			if w, _ := ui.ImageSize(p); w > 0 {
				return p
			}
		}
	}
	return ""
}

// sysDropAccept 统一的拖入处理（节点级 DropZone 与窗口级兜底共用）。
func sysDropAccept(paths []string) {
	if len(paths) == 0 {
		return
	}
	sysDropPaths = paths
	if p := sysPickImage(paths); p != "" {
		sysImgPath = p
		sysDropMsg = fmt.Sprintf("收到 %d 项，已把图片送入上方预览：%s", len(paths), filepath.Base(p))
	} else if len(paths) == 1 {
		sysDropMsg = "收到 1 项：" + paths[0]
	} else {
		sysDropMsg = fmt.Sprintf("收到 %d 项：%s …", len(paths), filepath.Base(paths[0]))
	}
	ui.RequestRedraw()
}

func sysCase() *ui.Node {
	// 动画：Tween / Spring / 关键帧
	t := float32(0)
	if sysAnimOn {
		t = 1
	}
	ease := ui.Tween("sys-ease", t, 0.5, ui.EaseOutCubic)
	back := ui.Tween("sys-back", t, 0.6, ui.EaseOutBack)
	sp := ui.Spring("sys-spring", t, 210, 15)
	kf := ui.Keyframes("sys-kf", []ui.Key{
		{At: 0, Value: 0},
		{At: 0.5, Value: 1},
		{At: 1, Value: 0},
	}, 2.4, true)

	// 图片区
	imgBody := ui.Column()
	if p := sysSampleImage(); p != "" {
		w, h := ui.ImageSize(p)
		scale := float32(280) / float32(w)
		imgBody.Add(
			ui.Row(
				ui.ImageSized(p, 280, float32(h)*scale).St(ui.St().R(14).Clipped()),
				ui.Column(
					ui.KeyValue("文件", p),
					ui.KeyValue("尺寸", fmt.Sprintf("%d × %d px", w, h)),
					ui.Text("支持 PNG / JPEG / GIF / BMP，纹理上传后按圆角裁剪绘制。").St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
				).St(ui.St().GrowTo(1).GapOf(8)),
			).St(ui.St().GapOf(16)),
		)
	} else {
		imgBody.Add(ui.Empty("未找到示例图片", "点击下方「选择图片」载入本地图片"))
	}
	imgBody.Add(ui.Row(
		ui.Btn("选择图片", ui.BtnPrimary, func() {
			ui.PickFile("选择图片", "图片|*.png;*.jpg;*.jpeg;*.gif;*.bmp|所有文件|*.*", func(p string) {
				sysImgPath = p
				sysMsg = "已载入：" + p
				ui.RequestRedraw()
			})
		}),
		ui.Btn("打开文件夹", ui.BtnSecondary, func() {
			if p, ok := ui.PickFolderDialog("选择目录"); ok {
				sysMsg = "已选择目录：" + p
			} else {
				sysMsg = "当前平台不支持选择文件夹（浏览器沙箱限制）"
			}
			ui.RequestRedraw()
		}),
		ui.Btn("另存为", ui.BtnSecondary, func() {
			if p, ok := ui.SaveFileDialog("保存文件", "文本|*.txt|所有文件|*.*", "txt"); ok {
				sysMsg = "另存为：" + p
			} else {
				sysMsg = "当前平台不支持另存为（浏览器端用 a[download] 代替）"
			}
			ui.RequestRedraw()
		}),
	).St(ui.St().GapOf(10)))

	// 毛玻璃
	glass := ui.Stack(
		ui.Box().St(ui.St().AbsFill().BgGrad("#312e81", "#0891b2", 45).R(18)),
		ui.Box().St(ui.St().AbsTL(30, 24).W(130).H(130).R(65).BgC("#a78bfa")),
		ui.Box().St(ui.St().AbsTL(210, 78).W(160).H(160).R(80).BgC("#f472b6")),
		ui.Box().St(ui.St().AbsTL(420, 20).W(100).H(100).R(50).BgC("#fbbf24")),
		ui.Box().St(ui.St().AbsTL(560, 60).W(120).H(120).R(60).BgC("#34d399")),
		ui.Column(
			ui.Text("毛玻璃 backdrop-filter").St(ui.St().Text(16, 700, "#ffffff")),
			ui.Text("离屏 FBO 捕获背景 → 双 Kawase 降采样模糊 → 圆角回绘 + 色调叠加。").St(ui.St().Text(12, 400, "#dbeafe").WrapOn()),
		).St(ui.St().AbsTL(70, 60).W(340).Padding(18).R(16).GapOf(8).
			Backdrop(22, ui.ColGlass).Stroke(1, "rgba(255,255,255,0.28)"))).
		St(ui.St().H(230).R(18).Clipped())

	// 动画
	anim := ui.Column(
		ui.Row(
			ui.Text("Tween").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Box().St(ui.St().W(80+240*ease).H(14).R(7).BgGrad(ui.ColPrimary, ui.ColPrimary2, 90)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("Spring").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Stack(
				ui.Box().St(ui.St().Wp(100).H(14).R(7).BgC("rgba(255,255,255,0.06)")),
				ui.Box().St(ui.St().W(22).H(14).R(7).BgC(ui.ColSuccess).AbsTL(240*sp, 0)),
			).St(ui.St().W(280).H(14)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("Back").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Box().St(ui.St().W(26).H(26).R(13).BgC(ui.ColPrimary2).MarOnly(0, 0, 0, 240*back)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("Keyframes").St(ui.St().Text(12, 500, ui.ColTextDim).W(70)),
			ui.Stack(
				ui.Box().St(ui.St().Wp(100).H(32).R(8).BgC("rgba(255,255,255,0.04)")),
				ui.Box().St(ui.St().W(8).H(8).R(4).BgC(ui.ColWarn).AbsTL(8+260*kf, 12)),
			).St(ui.St().W(280).H(32)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Btn("播放 / 复位", ui.BtnSecondary, func() { sysAnimOn = !sysAnimOn }),
			ui.Text("Tween 缓动 + Spring 弹簧积分 + Keyframes 循环").St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(10)),
	).St(ui.St().GapOf(10))

	// 字体与 Emoji
	fontRow := ui.Column(
		ui.Text("😀 🚀 🔥 🎉 🌈 🐳 ✅ 🎯").St(ui.St().Text(26, 400, "#ffffff")),
		ui.Text("彩色 Emoji 走 GDI 彩色位图光栅化，普通字行走 opentype 覆盖率抗锯齿。").St(ui.St().Text(12, 400, ui.ColTextDim)),
		ui.Row(
			ui.Text("微软雅黑 UI").St(ui.St().Text(14, 400, "#cbd5e1")),
			ui.Text("Segoe UI").St(ui.St().Text(14, 400, "#cbd5e1").Font("Segoe UI")),
			ui.Text("Consolas 0123").St(ui.St().Text(14, 400, "#cbd5e1").Font("Consolas")),
			ui.Text("宋体 SimSun").St(ui.St().Text(14, 400, "#cbd5e1").Font("SimSun")),
		).St(ui.St().GapOf(18)),
	).St(ui.St().GapOf(10))

	// IME 输入
	ime := ui.Column(
		ui.Row(
			ui.Text("姓名").St(ui.St().Text(12, 500, ui.ColTextDim).W(60)),
			ui.Input("sys-ime", sysIme, "点击后用中文输入法输入…", func(v string) {
				sysIme = v
				sysMsg = "输入内容：" + v
			}),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Text("组合串会以高亮块显示在光标处；上屏由 WM_IME_COMPOSITION(GCS_RESULTSTR) 派发。").St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	)

	// 滚动条
	rows := make([]*ui.Node, 0, 16)
	for i := 0; i < 16; i++ {
		idx := i
		color := ui.ColTextMuted
		if sysScrollRow == i {
			color = ui.ColPrimary
		}
		rows = append(rows, ui.Row(
			ui.Box().St(ui.St().W(6).H(6).R(3).BgC(color)),
			ui.Text("列表项 "+strconv.Itoa(i+1)).St(ui.St().Text(12.5, 500, color)),
			ui.Spacer(),
			ui.Text("拖拽右侧滚动条或滚轮").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(10).PadXY(12, 8).R(8).BgC("rgba(255,255,255,0.02)")).
			Tap(func() { sysScrollRow = idx; ui.RequestRedraw() }))
	}
	scrollDemo := ui.Scroll(ui.Column(rows...).St(ui.St().GapOf(6))).
		St(ui.St().H(190).R(10).Padding(8).BgC("rgba(255,255,255,0.02)").Stroke(1, ui.ColStroke)).
		K("sys-scroll")

	// 多窗口 / 托盘
	platform := ui.Column(
		ui.Row(
			ui.Btn("打开子窗口", ui.BtnPrimary, func() {
				sysOpened++
				title := fmt.Sprintf("子窗口 #%d", sysOpened)
				if _, err := ui.NewWindow(ui.Options{Title: title, Width: 520, Height: 360}, sysChildView); err != nil {
					sysMsg = "打开窗口失败：" + err.Error()
				} else {
					sysMsg = "已打开 " + title
				}
			}),
			ui.Btn("创建托盘图标", ui.BtnSecondary, func() {
				if sysTray != nil {
					sysMsg = "托盘图标已存在"
					return
				}
				owner := ui.Current()
				t, err := ui.NewTray("GoGUI 系统能力演示", func() {
					if owner != nil {
						owner.ShowWindow(true)
						owner.Restore()
					}
				}, []ui.TrayItem{
					{ID: 1, Label: "显示主窗口"},
					{ID: 2, Label: "换成托盘提示", Checked: true},
					{Separator: true},
					{ID: 9, Label: "退出"},
				}, func(id int) {
					switch id {
					case 1:
						if owner != nil {
							owner.ShowWindow(true)
							owner.Restore()
						}
					case 2:
						if sysTray != nil {
							sysTray.SetTooltip("GoGUI · " + strconv.Itoa(sysOpened) + " 个窗口")
						}
					case 9:
						if owner != nil {
							owner.Close()
						}
					}
				})
				if err != nil {
					sysMsg = "创建托盘失败"
					return
				}
				sysTray = t
				sysMsg = "已创建托盘图标（右键看菜单）"
			}),
			ui.Btn("移除托盘", ui.BtnGhost, func() {
				if sysTray != nil {
					sysTray.Remove()
					sysTray = nil
					sysMsg = "托盘图标已移除"
				}
			}),
		).St(ui.St().GapOf(10)),
		ui.Text(sysMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	).St(ui.St().GapOf(10))

	// ---- 拖入文件：节点级 DropZone + 窗口级兜底回调 ----
	if a := ui.Current(); a != nil {
		a.SetOnDropFiles(func(paths []string, x, y float32) {
			sysDropMsg = fmt.Sprintf("窗口级回调：%d 项，落点 %.0f, %.0f", len(paths), x, y)
			sysDropPaths = paths
			if p := sysPickImage(paths); p != "" {
				sysImgPath = p
			}
			ui.RequestRedraw()
		})
	}
	dropBody := ui.Column(
		ui.DropZone("sys-drop", "把文件 / 文件夹拖到这里松手", 138, sysDropAccept),
		ui.Text(sysDropMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	)
	if len(sysDropPaths) > 0 {
		dropBody.Add(ui.Scroll(
			ui.Column(ui.Each(sysDropPaths, func(p string, _ int) *ui.Node {
				return ui.FileRow(p, nil)
			})...).St(ui.St().GapOf(4)),
		).St(ui.St().H(168).R(10).Padding(6).BgC("rgba(255,255,255,0.02)").Stroke(1, ui.ColStroke)).
			K("sys-drop-list"))
	}
	dropBody.Add(ui.Wrap(
		ui.Btn("选择文件（多选）", ui.BtnSecondary, func() {
			ui.PickFilesAsync("选择文件（可多选）", "所有文件|*.*", func(ps []string) {
				sysDropAccept(ps)
			})
		}),
		ui.Btn("复制路径", ui.BtnGhost, func() {
			if len(sysDropPaths) == 0 {
				sysDropMsg = "还没有拖入文件"
			} else if ui.ClipboardSetFiles(sysDropPaths) {
				sysDropMsg = fmt.Sprintf("已复制 %d 个路径到剪贴板（可粘贴到资源管理器）", len(sysDropPaths))
			} else {
				sysDropMsg = "复制到剪贴板失败"
			}
			ui.RequestRedraw()
		}),
		ui.Btn("定位文件", ui.BtnGhost, func() {
			if len(sysDropPaths) == 0 {
				sysDropMsg = "还没有拖入文件"
			} else if ui.RevealInExplorer(sysDropPaths[0]) {
				sysDropMsg = "已在资源管理器中定位：" + sysDropPaths[0]
			}
			ui.RequestRedraw()
		}),
		ui.Btn("移入回收站", ui.BtnDanger, func() {
			if len(sysDropPaths) == 0 {
				sysDropMsg = "还没有拖入文件"
				return
			}
			n := 0
			for _, p := range sysDropPaths {
				if ui.MoveToRecycleBin(p) {
					n++
				}
			}
			sysDropMsg = fmt.Sprintf("已移入回收站 %d / %d 项（可从回收站还原）", n, len(sysDropPaths))
			sysDropPaths = nil
			ui.RequestRedraw()
		}),
		ui.Btn("清空列表", ui.BtnGhost, func() {
			sysDropPaths = nil
			sysDropMsg = "已清空拖入列表"
			ui.RequestRedraw()
		}),
	).St(ui.St().GapOf(8)))

	// ---- 系统操作能力 ----
	opsBody := ui.Column(
		ui.Row(
			ui.Text("窗口置顶").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Switch("sys-top", sysTopMost, func(v bool) {
				sysTopMost = v
				ui.SetAlwaysOnTop(v)
				if v {
					sysMsg = "已置顶：切到其它窗口时本窗口仍在最上层"
				} else {
					sysMsg = "已取消置顶"
				}
				ui.RequestRedraw()
			}),
			ui.Text("HWND_TOPMOST / HWND_NOTOPMOST").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("全局热键").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Btn("注册 Ctrl+Alt+G", ui.BtnSecondary, func() {
				if sysHotkeyID != 0 {
					sysHotkeyNote = "已注册，先注销"
					ui.RequestRedraw()
					return
				}
				id := ui.RegisterHotkey(ui.HotCtrl|ui.HotAlt, 'G', func() {
					sysHotkeyNote = "热键已触发（窗口失焦也有效）"
					ui.RequestRedraw()
				})
				sysHotkeyID = id
				if id == 0 {
					sysHotkeyNote = "注册失败：组合可能被其它程序占用"
				} else {
					sysHotkeyNote = fmt.Sprintf("已注册 id=%d", id)
				}
				ui.RequestRedraw()
			}),
			ui.Btn("注销", ui.BtnGhost, func() {
				if sysHotkeyID != 0 {
					ui.UnregisterHotkey(sysHotkeyID)
					sysHotkeyID = 0
					sysHotkeyNote = "已注销"
				} else {
					sysHotkeyNote = "当前没有热键"
				}
				ui.RequestRedraw()
			}),
			ui.Text(sysHotkeyNote).St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("屏保常亮").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Switch("sys-awake", sysWakeLock, func(v bool) {
				sysWakeLock = v
				ui.SetKeepAwake(v)
				if v {
					sysMsg = "已调用 SetThreadExecutionState：不会休眠、不会关屏"
				} else {
					sysMsg = "已恢复默认电源策略"
				}
				ui.RequestRedraw()
			}),
			ui.Text("ES_CONTINUOUS | ES_DISPLAY_REQUIRED").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("剪贴板").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Input("sys-clip", sysClipText, "输入文本后点「写入剪贴板」", func(v string) {
				sysClipText = v
			}),
			ui.Btn("写入", ui.BtnSecondary, func() {
				if ui.ClipboardSetText(sysClipText) {
					sysMsg = "已写入剪贴板：" + sysClipText
				} else {
					sysMsg = "写入剪贴板失败"
				}
				ui.RequestRedraw()
			}),
			ui.Btn("读取", ui.BtnGhost, func() {
				sysClipText = ui.ClipboardText()
				sysMsg = "剪贴板内容：" + sysClipText
				ui.RequestRedraw()
			}),
		).St(ui.St().GapOf(10)).AlignTo(ui.AlignCenter),
		ui.Row(
			ui.Text("托盘通知").St(ui.St().Text(12.5, 500, ui.ColTextDim).W(84)),
			ui.Btn("弹出气泡", ui.BtnSecondary, func() {
				if sysTray == nil {
					sysMsg = "请先在上方「多窗口 / 托盘」里创建托盘图标"
				} else {
					sysTray.Notify("GoGUI 系统能力", "这是一条 NIF_INFO 气泡通知")
					sysMsg = "已弹出托盘气泡通知"
				}
				ui.RequestRedraw()
			}),
			ui.Btn("检测系统主题", ui.BtnGhost, func() {
				mode := "浅色"
				if ui.SystemDarkMode() {
					mode = "深色"
				}
				sysMsg = "注册表 AppsUseLightTheme：系统当前为" + mode + "模式"
				ui.RequestRedraw()
			}),
			ui.Text("NIF_INFO 气泡 / RegQueryValueEx").St(ui.St().Text(11, 400, ui.ColTextMuted)),
		).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter),
		ui.Text(sysMsg).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn()),
	).St(ui.St().GapOf(10))

	return ui.Column(
		pageHead("系统能力", "图片 / 拖入文件 / 毛玻璃 / Emoji / 动画 / 输入法 / 滚动条 / 系统操作"),
		sysSection("图片解码与纹理绘制", "PNG/JPEG/GIF/BMP → RGBA 纹理 → 圆角裁剪", imgBody),
		sysSection("拖入文件 WM_DROPFILES", "拖到窗口内松手 → 落点命中节点 OnDrop → 路径列表 / 图片自动预览", dropBody),
		sysSection("毛玻璃 backdrop-filter", "离屏 FBO + Kawase 模糊", glass),
		sysSection("动画系统", "Tween / Spring / Keyframes，仅在有动画时逐帧重绘", anim),
		sysSection("多字体与彩色 Emoji", "opentype 主字体 + Segoe UI Symbol fallback + GDI 彩色 Emoji", fontRow),
		sysSection("输入法（IME）", "WM_IME_COMPOSITION 组合串预览与上屏", ime),
		sysSection("滚动条", "内容溢出时显示，悬停加宽、可拖拽", scrollDemo),
		sysSection("多窗口 / 托盘 / 文件对话框", "独立 GL 上下文窗口、任务栏托盘、系统对话框", platform),
		sysSection("系统操作能力", "置顶 / 全局热键 / 托盘气泡 / 剪贴板 / 阻止休眠 / 深浅色 / 回收站", opsBody),
	).St(ui.St().Padding(26).GapOf(18))
}

// sysChildView 子窗口内容。
func sysChildView() *ui.Node {
	return ui.Column(
		ui.Text("独立窗口").St(ui.St().Text(22, 700, ui.ColText)),
		ui.Text("每个窗口拥有自己的 GL 上下文、字形图集与布局树。").St(ui.St().Text(12.5, 400, ui.ColTextDim).WrapOn()),
		ui.Spacer(),
		ui.Row(
			ui.Btn("再开一个", ui.BtnSecondary, func() {
				sysOpened++
				ui.NewWindow(ui.Options{Title: fmt.Sprintf("子窗口 #%d", sysOpened), Width: 480, Height: 320}, sysChildView)
			}),
			ui.Btn("关闭本窗口", ui.BtnDanger, func() {
				if a := ui.Current(); a != nil {
					a.Close()
				}
			}),
		).St(ui.St().GapOf(10)),
	).St(ui.St().Padding(26).GapOf(12).Hp(100).BgC(ui.ColBg))
}
