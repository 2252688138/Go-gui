// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"strings"
)

// 弹窗组件：Alert（提示）/ Confirm（确认）/ Prompt（输入）/ ModalPanel（自定义内容）。
//
// 约定：弹窗与 Modal 一样是根 Stack 里的全屏层（AbsFill），调用方在 view() 末尾叠加：
//
//	root := ui.Stack(shell, ui.AlertDlg(...), ui.ConfirmDlg(...))
//
// 打开/关闭由调用方持有的 bool 控制；打开时淡入并轻微上浮（AnimTime 驱动逐帧重绘）。

// DialogKind 弹窗语义（决定图标色与默认按钮色）。
type DialogKind int

const (
	DlgInfo DialogKind = iota
	DlgSuccess
	DlgWarn
	DlgError
)

func dlgColor(k DialogKind) string {
	switch k {
	case DlgSuccess:
		return ColSuccess
	case DlgWarn:
		return ColWarn
	case DlgError:
		return ColDanger
	}
	return ColInfo
}

// dlgAnim 每个弹窗的打开时刻（淡入动画起点）。
var dlgAnim = map[string]float32{}

// dlgT 打开进度 0~1（1 表示动画结束）。
func dlgT(key string, open bool) float32 {
	if !open {
		delete(dlgAnim, key)
		return 1
	}
	start, ok := dlgAnim[key]
	if !ok {
		start = AnimTime()
		dlgAnim[key] = start
	}
	t := (AnimTime() - start) / 0.14
	if t >= 1 {
		return 1
	}
	if t < 0 {
		t = 0
	}
	RequestRedraw()
	return easeOut(t)
}

// dlgScrim 遮罩层（点击关闭可选；透明度随淡入动画变化）。
func dlgScrim(t float32, onClose func()) *Node {
	c := C(ColScrim)
	c.A *= t
	n := Box().St(St().AbsFill().BgC(c))
	if onClose != nil {
		n.Tap(onClose)
	}
	return n
}

// DialogOpts 弹窗内容配置。
type DialogOpts struct {
	Key      string // 动画 Key（同一弹窗保持固定）
	Open     bool
	Kind     DialogKind
	Title    string
	Body     string // 纯文本正文（BodyNode 非空时忽略）
	BodyNode *Node  // 自定义正文
	Width    float32
	// Closable 右上角显示关闭按钮，并允许点遮罩关闭。
	Closable bool
	OnClose  func()
	// Footer 自定义底部（为 nil 时用 OKLabel/CancelLabel 生成）。
	Footer      *Node
	OKLabel     string
	CancelLabel string
	OnOK        func()
	OnCancel    func()
	// Blur 遮罩是否叠加毛玻璃（真模糊窗口背景）。
	Blur float32
}

// ModalPanel 通用弹窗：遮罩 + 卡片（标题 / 图标 / 正文 / 底部按钮），带淡入动画。
func ModalPanel(o DialogOpts) *Node {
	if !o.Open {
		return nil
	}
	if o.Key == "" {
		o.Key = "dlg-" + o.Title
	}
	if o.Width <= 0 {
		o.Width = 420
	}
	t := dlgT(o.Key, true)

	head := []*Node{Dot(dlgColor(o.Kind), 8)}
	head = append(head, Text(o.Title).St(St().Text(15.5, 600, ColText)).Grow(1))
	if o.Closable {
		head = append(head, Text("×").St(St().Text(15, 500, ColTextMuted)).Tap(func() {
			if o.OnClose != nil {
				o.OnClose()
			}
			RequestRedraw()
		}))
	}
	title := Row(head...).St(St().GapOf(10)).AlignTo(AlignCenter)

	var body *Node
	switch {
	case o.BodyNode != nil:
		body = o.BodyNode
	case o.Body != "":
		body = Text(o.Body).St(St().Text(13, 400, ColTextDim).WrapOn())
	}

	footer := o.Footer
	if footer == nil {
		items := []*Node{Spacer()}
		if o.CancelLabel != "" {
			cancel := o.OnCancel
			items = append(items, Btn(o.CancelLabel, BtnSecondary, func() {
				if cancel != nil {
					cancel()
				}
				RequestRedraw()
			}))
		}
		if o.OKLabel != "" {
			ok := o.OnOK
			kind := BtnPrimary
			if o.Kind == DlgError {
				kind = BtnDanger
			}
			items = append(items, Btn(o.OKLabel, kind, func() {
				if ok != nil {
					ok()
				}
				RequestRedraw()
			}))
		}
		if len(items) > 1 {
			footer = Row(items...).St(St().GapOf(10)).AlignTo(AlignCenter)
		}
	}

	cells := []*Node{title}
	if body != nil {
		cells = append(cells, body)
	}
	if footer != nil {
		cells = append(cells, footer)
	}
	card := Column(cells...).St(St().W(o.Width).Padding(22).R(18).GapOf(14).
		BgC(ColPanelDeep).Stroke(1, ColStroke).
		Shadow(48, 18+8*(1-t), 0, "rgba(0,0,0,0.55)").
		Fade(t).AbsCenter())

	scrim := dlgScrim(t, nil)
	if o.Closable {
		scrim = dlgScrim(t, func() {
			if o.OnClose != nil {
				o.OnClose()
			}
			RequestRedraw()
		})
	}
	if o.Blur > 0 {
		scrim.St(St().Backdrop(o.Blur, nil))
	}
	return Stack(scrim, card).St(St().AbsFill()).K(o.Key)
}

// AlertDlg 提示弹窗（单个「知道了」按钮）。
func AlertDlg(key string, open bool, kind DialogKind, title, body string, onClose func()) *Node {
	return ModalPanel(DialogOpts{
		Key: key, Open: open, Kind: kind, Title: title, Body: body,
		Closable: true, OnClose: onClose,
		OKLabel: "知道了", OnOK: onClose,
	})
}

// ConfirmDlg 确认弹窗（取消 + 确定）。
func ConfirmDlg(key string, open bool, kind DialogKind, title, body, okLabel string, onCancel, onOK func()) *Node {
	if okLabel == "" {
		okLabel = "确定"
	}
	return ModalPanel(DialogOpts{
		Key: key, Open: open, Kind: kind, Title: title, Body: body,
		Closable: true, OnClose: onCancel,
		CancelLabel: "取消", OKLabel: okLabel,
		OnCancel: onCancel, OnOK: onOK,
	})
}

// promptText 每个输入弹窗的编辑缓冲。
var promptText = map[string]string{}

// promptOpen 记录上一帧是否打开（用于重置缓冲与抢焦点）。
var promptOpen = map[string]bool{}

// PromptDlg 输入弹窗（回车提交 / Esc 取消）。
func PromptDlg(key string, open bool, kind DialogKind, title, label, value, placeholder, okLabel string, onOK func(string), onCancel func()) *Node {
	if !open {
		promptOpen[key] = false
		return nil
	}
	inputKey := key + "-input"
	if !promptOpen[key] {
		promptOpen[key] = true
		promptText[key] = value
		inputCaret[inputKey] = len([]rune(value))
		SetFocus(inputKey)
	}
	cur := promptText[key]
	field := Input(inputKey, cur, placeholder, func(s string) {
		promptText[key] = s
	})
	field.OnKey = func(k uint32) {
		switch k {
		case 0x0D: // Enter 提交
			if onOK != nil {
				onOK(strings.TrimSpace(promptText[key]))
			}
			SetFocus("")
			RequestRedraw()
		case 0x1B: // Esc 取消
			if onCancel != nil {
				onCancel()
			}
			SetFocus("")
			RequestRedraw()
		}
	}
	body := Column(field)
	if label != "" {
		body = Column(
			Text(label).St(St().Text(12.5, 500, ColTextDim)),
			field,
		).St(St().GapOf(8))
	}
	if okLabel == "" {
		okLabel = "确定"
	}
	return ModalPanel(DialogOpts{
		Key: key, Open: true, Kind: kind, Title: title, BodyNode: body,
		Closable: true, OnClose: func() {
			if onCancel != nil {
				onCancel()
			}
			SetFocus("")
		},
		CancelLabel: "取消", OKLabel: okLabel,
		OnCancel: func() {
			if onCancel != nil {
				onCancel()
			}
			SetFocus("")
		},
		OnOK: func() {
			if onOK != nil {
				onOK(strings.TrimSpace(promptText[key]))
			}
			SetFocus("")
		},
	})
}

// DrawerOpts 侧边抽屉配置。
type DrawerOpts struct {
	Key     string
	Open    bool
	Title   string
	Body    *Node
	Width   float32
	OnClose func()
}

// Drawer 右侧滑出抽屉（AbsFill 遮罩 + 右侧面板，淡入）。
func Drawer(o DrawerOpts) *Node {
	if !o.Open {
		return nil
	}
	if o.Key == "" {
		o.Key = "drawer-" + o.Title
	}
	if o.Width <= 0 {
		o.Width = 380
	}
	t := dlgT(o.Key, true)
	head := Row(
		Text(o.Title).St(St().Text(15, 600, ColText)).Grow(1),
		Text("×").St(St().Text(16, 500, ColTextMuted)).Tap(func() {
			if o.OnClose != nil {
				o.OnClose()
			}
			RequestRedraw()
		}),
	).St(St().GapOf(10)).AlignTo(AlignCenter)
	cells := []*Node{head, Divider()}
	if o.Body != nil {
		cells = append(cells, Scroll(o.Body).Grow(1))
	}
	panel := Column(cells...).St(St().W(o.Width).Hp(100).Padding(20).GapOf(14).
		BgC(ColPanelDeep).Stroke(1, ColStroke).
		Shadow(48, 8, 0, "rgba(0,0,0,0.55)").Fade(t).AbsTR(0, 0))
	scrim := dlgScrim(t, func() {
		if o.OnClose != nil {
			o.OnClose()
		}
		RequestRedraw()
	})
	return Stack(scrim, panel).St(St().AbsFill()).K(o.Key)
}
