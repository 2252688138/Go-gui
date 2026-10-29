//go:build windows

package win32

import (
	"unicode/utf16"
	"unsafe"
)

// handleIME 读取输入法组合串并派发事件。
//
// 只处理 GCS_RESULTSTR（上屏）与 GCS_COMPSTR（组合中）；返回 0 阻止
// DefWindowProc 再次生成 WM_CHAR，否则中文会被插入两次。
func (w *Window) handleIME(hwnd uintptr, lp uintptr) {
	himc := ImmGetContext(hwnd)
	if himc == 0 {
		return
	}
	defer ImmReleaseContext(hwnd, himc)

	// 上屏字符串：逐字符派发 EvChar，输入框按普通字符处理
	if lp&GCS_RESULTSTR != 0 {
		if s := immString(himc, GCS_RESULTSTR); s != "" {
			if w.onEvent != nil {
				for _, r := range s {
					w.onEvent(Event{Type: EvChar, Char: r})
				}
			}
		}
		w.compText, w.compCaret = "", 0
		if w.onEvent != nil {
			w.onEvent(Event{Type: EvCompositionEnd})
		}
	}

	// 组合串：仅用于显示（带下划线），不上屏
	if lp&GCS_COMPSTR != 0 {
		s := immString(himc, GCS_COMPSTR)
		w.compText = s
		w.compCaret = int(ImmGetCompositionStringW(himc, GCS_CURSORPOS, 0, 0))
		if w.compCaret < 0 || w.compCaret > len([]rune(s)) {
			w.compCaret = len([]rune(s))
		}
		if w.onEvent != nil {
			w.onEvent(Event{Type: EvComposition, Text: s, Caret: w.compCaret})
		}
	}
}

// immString 读取一段输入法字符串。
func immString(himc uintptr, kind uint32) string {
	n := ImmGetCompositionStringW(himc, kind, 0, 0)
	if n <= 0 {
		return ""
	}
	buf := make([]uint16, n/2+1)
	got := ImmGetCompositionStringW(himc, kind, uintptr(unsafe.Pointer(&buf[0])), int32(len(buf)*2))
	if got <= 0 {
		return ""
	}
	return string(utf16.Decode(buf[:got/2]))
}

// SetImePosition 把输入法组合窗口/候选窗口移动到客户区坐标 (x, y)（物理像素）。
func (w *Window) SetImePosition(x, y int32) {
	himc := ImmGetContext(w.Hwnd)
	if himc == 0 {
		return
	}
	defer ImmReleaseContext(w.Hwnd, himc)
	pt := POINT{X: x, Y: y}
	cf := COMPOSITIONFORM{DwStyle: CFS_POINT, PtCurrentPos: pt}
	ImmSetCompositionWindow(himc, &cf)
	cand := CANDIDATEFORM{DwStyle: CFS_POINT, PtCurrentPos: pt}
	ImmSetCandidateWindow(himc, &cand)
}

// CompositionString 返回当前组合串与光标位置（供绘制预览）。
func (w *Window) CompositionString() (string, int) { return w.compText, w.compCaret }

// ---- 窗口控制（多窗口 / 托盘）----

// SetVisible 显示或隐藏窗口。
func (w *Window) SetVisible(v bool) {
	if v {
		ShowWindow(w.Hwnd, SW_SHOW)
		ShowWindow(w.Hwnd, SW_SHOW)
		UpdateWindow(w.Hwnd)
		return
	}
	ShowWindow(w.Hwnd, SW_HIDE)
}

// Minimize 最小化窗口。
func (w *Window) Minimize() { ShowWindow(w.Hwnd, SW_MINIMIZE) }

// Maximize 最大化窗口。
func (w *Window) Maximize() { ShowWindow(w.Hwnd, SW_MAXIMIZE) }

// IsMaximized 窗口是否已最大化。
func (w *Window) IsMaximized() bool { return w != nil && w.Hwnd != 0 && IsZoomed(w.Hwnd) != 0 }

// ToggleMaximize 在最大化与还原之间切换。
func (w *Window) ToggleMaximize() {
	if w.IsMaximized() {
		ShowWindow(w.Hwnd, SW_RESTORE)
		return
	}
	ShowWindow(w.Hwnd, SW_MAXIMIZE)
}

// BeginDragMove 让系统接管窗口拖动（自绘标题栏按下时调用）。
// 内部 SendMessage 会进入系统的模态拖动循环，返回时拖动已结束。
func (w *Window) BeginDragMove() {
	if w == nil || w.Hwnd == 0 {
		return
	}
	ReleaseCapture()
	SendMessageW(w.Hwnd, WM_NCLBUTTONDOWN, HTCAPTION, 0)
}

// Restore 还原并置前。
func (w *Window) Restore() {
	ShowWindow(w.Hwnd, SW_RESTORE)
	SetForegroundWindow(w.Hwnd)
}

// Center 把窗口居中到其所在显示器的工作区（水平 + 垂直，已排除任务栏）。
// 多显示器时以窗口当前所在显示器为准；窗口比工作区还大时贴左上角。
func (w *Window) Center() {
	if w == nil || w.Hwnd == 0 {
		return
	}
	if GetMonitorInfoW == nil || MonitorFromWindow == nil {
		return
	}
	var wr RECT
	if GetWindowRect(w.Hwnd, &wr) == 0 {
		return
	}
	ww, wh := wr.Right-wr.Left, wr.Bottom-wr.Top
	mon := MonitorFromWindow(w.Hwnd, MONITOR_DEFAULTTONEAREST)
	if mon == 0 {
		mon = MonitorFromWindow(w.Hwnd, MONITOR_DEFAULTTOPRIMARY)
	}
	if mon == 0 {
		return
	}
	mi := MONITORINFO{CbSize: 40}
	if GetMonitorInfoW(mon, &mi) == 0 {
		return
	}
	work := mi.RcWork
	x := work.Left + (work.Right-work.Left-ww)/2
	y := work.Top + (work.Bottom-work.Top-wh)/2
	if x < work.Left {
		x = work.Left
	}
	if y < work.Top {
		y = work.Top
	}
	SetWindowPos(w.Hwnd, 0, x, y, 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
	dbg("center: work=(%d,%d %dx%d) win=%dx%d -> (%d,%d)",
		work.Left, work.Top, work.Right-work.Left, work.Bottom-work.Top, ww, wh, x, y)
}

// Visible 窗口是否可见。
func (w *Window) Visible() bool { return IsWindowVisible(w.Hwnd) != 0 }

// Close 请求关闭窗口（走 WM_CLOSE）。
func (w *Window) Close() { PostMessageW(w.Hwnd, WM_CLOSE, 0, 0) }

// Focus 置前并激活。
func (w *Window) Focus() {
	SetForegroundWindow(w.Hwnd)
	SetFocus(w.Hwnd)
}

// Running 窗口是否仍存活。
func (w *Window) Running() bool { return w.running && IsWindow(w.Hwnd) != 0 }

// SetFullscreen 切换无边框全屏（去标题栏 + 覆盖当前显示器，含任务栏区域）。
// 退出时恢复原样式与窗口矩形。
func (w *Window) SetFullscreen(on bool) {
	if w == nil || w.Hwnd == 0 || on == w.fullscreen {
		return
	}
	if on {
		if GetWindowLongPtrW == nil || SetWindowLongPtrW == nil {
			return
		}
		w.savedStyle = GetWindowLongPtrW(w.Hwnd, GWL_STYLE)
		GetWindowRect(w.Hwnd, &w.savedRect)
		style := w.savedStyle &^ uintptr(WS_OVERLAPPEDWINDOW)
		style |= uintptr(WS_POPUP) | uintptr(WS_VISIBLE)
		SetWindowLongPtrW(w.Hwnd, GWL_STYLE, style)

		// MONITORINFO 在 amd64 下大小为 40 字节（4 + 16 + 16 + 4）
		mi := MONITORINFO{CbSize: 40}
		mon := MonitorFromWindow(w.Hwnd, MONITOR_DEFAULTTONEAREST)
		if mon != 0 && GetMonitorInfoW(mon, &mi) != 0 {
			SetWindowPos(w.Hwnd, 0,
				mi.RcMonitor.Left, mi.RcMonitor.Top,
				mi.RcMonitor.Right-mi.RcMonitor.Left, mi.RcMonitor.Bottom-mi.RcMonitor.Top,
				SWP_FRAMECHANGED|SWP_NOZORDER|SWP_SHOWWINDOW)
		}
		w.fullscreen = true
		// 全屏圆角一并取消（applyCorner 会在 fullscreen 时清除）
		w.applyCorner()
		SetForegroundWindow(w.Hwnd)
		return
	}

	SetWindowLongPtrW(w.Hwnd, GWL_STYLE, w.savedStyle)
	SetWindowPos(w.Hwnd, 0,
		w.savedRect.Left, w.savedRect.Top,
		w.savedRect.Right-w.savedRect.Left, w.savedRect.Bottom-w.savedRect.Top,
		SWP_FRAMECHANGED|SWP_NOZORDER|SWP_SHOWWINDOW)
	w.fullscreen = false
	w.refreshMetrics()
	// 退出全屏后恢复窗口圆角
	w.applyCorner()
}

// IsFullscreen 当前是否处于全屏。
func (w *Window) IsFullscreen() bool { return w != nil && w.fullscreen }
