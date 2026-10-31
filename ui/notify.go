// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"strings"
)

// 现代通知中心：右侧堆叠、图标、按钮、进度、悬停暂停过期。
//
//	ui.Notify(ui.Notice{
//	    ID: "save", Kind: "success", Title: "已保存",
//	    Body: "文档已同步到云端", Duration: 4,
//	    Buttons: []ui.NoticeButton{{Label: "撤销", Run: undo}},
//	})
//	ui.NotifyProgress("save", 0.6, "上传中…")   // 同 ID 就地更新
//
// 根 Stack 末尾放 ui.NotifyLayer()。

// NoticeKind 通知类型。
const (
	NoticeInfo    = "info"
	NoticeSuccess = "success"
	NoticeWarn    = "warn"
	NoticeError   = "error"
)

// NoticeButton 通知中的按钮。
type NoticeButton struct {
	Label string
	Run   func()
}

// Notice 一条通知。
type Notice struct {
	ID       string
	Kind     string
	Title    string
	Body     string
	Icon     string
	Buttons  []NoticeButton
	Progress float32 // >=0 且 <=1 时显示进度条
	Duration float32 // 秒，<=0 用 4 秒；<0 表示常驻
}

type noticeEntry struct {
	Notice
	at      float32
	paused  bool
	enterAt float32
}

var (
	notices     []*noticeEntry
	noticeLimit = 5
)

// NoticeMax 设置同时显示的最大条数（默认 5，超出淘汰最旧）。
func NoticeMax(n int) {
	if n < 1 {
		n = 1
	}
	noticeLimit = n
}

// Notify 显示一条通知（同 ID 就地覆盖并重置计时）。
func Notify(n Notice) {
	if n.ID == "" {
		n.ID = fmt.Sprintf("notice-%d", int(AnimTime()*1000))
	}
	if n.Kind == "" {
		n.Kind = NoticeInfo
	}
	if n.Duration == 0 {
		n.Duration = 4
	}
	for _, e := range notices {
		if e.ID == n.ID {
			e.Notice = n
			e.at = AnimTime()
			e.paused = false
			RequestRedraw()
			return
		}
	}
	notices = append(notices, &noticeEntry{Notice: n, at: AnimTime()})
	for len(notices) > noticeLimit {
		notices = notices[1:]
	}
	RequestRedraw()
}

// NotifyProgress 更新带进度通知（不存在时创建）。
func NotifyProgress(id string, progress float32, body string) {
	for _, e := range notices {
		if e.ID == id {
			e.Progress = progress
			if body != "" {
				e.Body = body
			}
			if progress >= 1 && e.Duration == 0 {
				e.Duration = 3
			}
			e.at = AnimTime()
			RequestRedraw()
			return
		}
	}
	Notify(Notice{ID: id, Kind: NoticeInfo, Title: body, Body: body, Progress: progress})
}

// NotifyClose 关闭指定通知。
func NotifyClose(id string) {
	for i, e := range notices {
		if e.ID == id {
			notices = append(notices[:i], notices[i+1:]...)
			RequestRedraw()
			return
		}
	}
}

// NotifyClear 关闭全部通知。
func NotifyClear() {
	notices = nil
	RequestRedraw()
}

// NotifyCount 当前通知数量。
func NotifyCount() int { return len(notices) }

// NotifyLayer 通知浮层（根 Stack 末尾）。右上角堆叠，悬停暂停过期。
func NotifyLayer() *Node {
	if len(notices) == 0 {
		return nil
	}
	now := AnimTime()
	mx, my := MousePos()
	keep := notices[:0]
	for _, e := range notices {
		if e.Duration > 0 && !e.paused && now-e.at >= e.Duration {
			continue // 过期
		}
		keep = append(keep, e)
	}
	notices = keep
	if len(notices) == 0 {
		return nil
	}
	cols := make([]*Node, 0, len(notices))
	for i, e := range notices {
		e := e
		entry := noticeCard(e)
		// 悬停检测：用上一帧矩形判断暂停
		if r, ok := LastFrameOf("notice-" + e.ID); ok {
			inside := mx >= r.X && mx <= r.X+r.W && my >= r.Y && my <= r.Y+r.H
			e.paused = inside
		}
		// 退场动画
		t := float32(1)
		if e.Duration > 0 {
			left := e.Duration - (now - e.at)
			if left < 0.35 {
				t = left / 0.35
			}
		}
		entry.St(St().Fade(0.3+0.7*t).Translate(0, 14*(1-t)))
		_ = i
		cols = append(cols, entry)
	}
	layers := make([]*Node, 0, len(cols)+1)
	layers = append(layers, Column(cols...).St(St().GapOf(10).W(320).
		AbsTR(18, 18)))
	// 通知可点击关闭 / 执行按钮；不铺满遮罩，避免挡住主界面
	_ = layers
	RequestRedraw() // 计时与退场动画需要持续帧
	return Stack(layers...).St(St().AbsFill())
}

// noticeCard 单条通知卡片。
func noticeCard(e *noticeEntry) *Node {
	color := ColInfo
	glyph := e.Icon
	switch e.Kind {
	case NoticeSuccess:
		color = ColSuccess
		if glyph == "" {
			glyph = "✓"
		}
	case NoticeWarn:
		color = ColWarn
		if glyph == "" {
			glyph = "!"
		}
	case NoticeError:
		color = ColDanger
		if glyph == "" {
			glyph = "×"
		}
	default:
		if glyph == "" {
			glyph = "i"
		}
	}
	body := Column(
		Text(e.Title).St(St().Text(12.5, 600, ColText).WrapOn()),
	)
	if e.Body != "" {
		body.Add(Text(e.Body).St(St().Text(11.5, 400, ColTextMuted).WrapOn()))
	}
	if e.Progress > 0 {
		body.Add(Progress(e.Progress, color).St(St().Wp(100)))
	}
	if len(e.Buttons) > 0 {
		btns := make([]*Node, 0, len(e.Buttons))
		for _, b := range e.Buttons {
			btn := b
			btns = append(btns, Text(btn.Label).St(St().Text(11, 600, color).PadXY(8, 4).R(6).
				BgC(ColBgSoft).Stroke(1, ColStroke)).Tap(func() {
				if btn.Run != nil {
					btn.Run()
				}
				NotifyClose(e.ID)
			}))
		}
		body.Add(Row(btns...).St(St().GapOf(6).MarOnly(4, 0, 0, 0)))
	}
	card := Row(
		Stack(
			Box().St(St().AbsFill().R(12).BgC(paintWithAlpha(color, 0.16))),
			Text(glyph).St(St().AbsFill().Text(13, 700, color).CenterYOn().TAlign(AlignCenter)),
		).St(St().W(24).H(24)),
		body.Grow(1),
		Text("×").St(St().Text(14, 500, ColTextMuted).W(20).TAlign(AlignCenter)).Tap(func() {
			NotifyClose(e.ID)
		}),
	).St(St().W(320).Padding(12).GapOf(10).R(12).BgC(ColPanelDeep).
		Stroke(1, ColStroke).Shadow(32, 12, 0, "rgba(0,0,0,0.55)")).
		AlignTo(AlignStart)
	card.K("notice-" + e.ID)
	return card
}

// NoticeText 通知的纯文本摘要（调试 / 无障碍朗读）。
func NoticeText(n Notice) string {
	parts := []string{n.Title}
	if n.Body != "" {
		parts = append(parts, n.Body)
	}
	return strings.Join(parts, "：")
}
