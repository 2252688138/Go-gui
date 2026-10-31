// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"time"

	"gogui/internal/gpu"
)

// 主题系统：深色 / 浅色 / 跟随系统。
//
// 组件里的颜色统一走包级 token（ColText / ColBg / ColInput / ColPanel …），
// SetTheme 时整体换色板并请求重绘；组件按帧重建，因此下一帧即生效。
// 业务自定义颜色仍可传任意字符串（C() 解析），不受主题影响。

// Theme 一套色板。
type Theme struct {
	Name string

	Text      string // 主文字
	TextDim   string // 次要文字
	TextMuted string // 弱化文字 / 占位说明

	Bg     string // 页面背景
	BgSoft string // 次级背景（卡片内分区、按钮底）
	BgDeep string // 更深背景（表格行/代码块）
	Stroke string // 描边
	Line   string // 细分隔线 / 半透明描边

	Primary  string
	Primary2 string
	Success  string
	Warn     string
	Danger   string
	Info     string

	Input       string // 输入框底色
	Panel       string // 浮层（菜单/下拉/提示）底色
	PanelDeep   string // 弹窗/抽屉底色
	Row         string // 表格偶数行
	RowAlt      string // 表格奇数行
	Header      string // 表头底色
	Placeholder string
	Hover       string // 半透明悬停底色
	Scrim       string // 弹窗遮罩
	Track       string // 滑块/进度轨道
	Glass       string // 毛玻璃叠色（Backdrop tint）
	FocusRing   string // 键盘焦点环
}

// Dark 深色主题（默认）。
var Dark = Theme{
	Name:        "dark",
	Text:        "#eef2ff",
	TextDim:     "#98a3c0",
	TextMuted:   "#6b7694",
	Bg:          "#101827",
	BgSoft:      "#151f33",
	BgDeep:      "#0a1120",
	Stroke:      "#243049",
	Line:        "rgba(255,255,255,0.08)",
	Primary:     "#6366f1",
	Primary2:    "#8b5cf6",
	Success:     "#34d399",
	Warn:        "#fbbf24",
	Danger:      "#f87171",
	Info:        "#38bdf8",
	Input:       "#0c1322",
	Panel:       "#131c2e",
	PanelDeep:   "#0f1728",
	Row:         "#0e1626",
	RowAlt:      "#0b1220",
	Header:      "#0c1424",
	Placeholder: "#4d5670",
	Hover:       "rgba(255,255,255,0.05)",
	Scrim:       "rgba(4,7,14,0.62)",
	Track:       "#1e2740",
	Glass:       "rgba(11,16,32,0.40)",
	FocusRing:   "#8b93ff",
}

// Light 浅色主题。
var Light = Theme{
	Name:        "light",
	Text:        "#0f172a",
	TextDim:     "#475569",
	TextMuted:   "#7c8aa0",
	Bg:          "#ffffff",
	BgSoft:      "#f2f5fb",
	BgDeep:      "#f8fafc",
	Stroke:      "#dbe2ee",
	Line:        "rgba(15,23,42,0.10)",
	Primary:     "#4f46e5",
	Primary2:    "#7c3aed",
	Success:     "#059669",
	Warn:        "#d97706",
	Danger:      "#dc2626",
	Info:        "#0284c7",
	Input:       "#f7f9fd",
	Panel:       "#ffffff",
	PanelDeep:   "#ffffff",
	Row:         "#ffffff",
	RowAlt:      "#f7f9fd",
	Header:      "#eef2f9",
	Placeholder: "#94a3b8",
	Hover:       "rgba(15,23,42,0.05)",
	Scrim:       "rgba(15,23,42,0.38)",
	Track:       "#e2e8f0",
	Glass:       "rgba(255,255,255,0.55)",
	FocusRing:   "#4f46e5",
}

// paintOf 返回指定颜色的半透明画笔（主题相关的叠加色用）。
func paintOf(c string, a float32) gpu.Paint {
	p := C(c)
	p.A = a
	return p
}

// ---- 包级颜色 token（组件直接引用，切换主题后下一帧生效）----

var (
	ColText      = Dark.Text
	ColTextDim   = Dark.TextDim
	ColTextMuted = Dark.TextMuted

	ColBg     = Dark.Bg
	ColBgSoft = Dark.BgSoft
	ColBgDeep = Dark.BgDeep
	ColStroke = Dark.Stroke
	ColLine   = Dark.Line

	ColPrimary  = Dark.Primary
	ColPrimary2 = Dark.Primary2
	ColSuccess  = Dark.Success
	ColWarn     = Dark.Warn
	ColDanger   = Dark.Danger
	ColInfo     = Dark.Info

	ColInput       = Dark.Input
	ColPanel       = Dark.Panel
	ColPanelDeep   = Dark.PanelDeep
	ColRow         = Dark.Row
	ColRowAlt      = Dark.RowAlt
	ColHeader      = Dark.Header
	ColPlaceholder = Dark.Placeholder
	ColHover       = Dark.Hover
	ColScrim       = Dark.Scrim
	ColTrack       = Dark.Track
	ColGlass       = Dark.Glass

	// 语义 token 快捷变量（与 theme_tokens.go 的 Tokens 对应）
	ColFocus         = Dark.FocusRing
	ColSurface       = Dark.Bg
	ColSurfaceMuted  = Dark.BgSoft
	ColSurfaceRaised = Dark.Panel
	ColSurfaceDeep   = Dark.BgDeep
	ColOnAccent      = "#ffffff"
)

// 主题模式。
const (
	ThemeDark   = "dark"
	ThemeLight  = "light"
	ThemeSystem = "system"
)

var (
	curTheme  = Dark
	themeMode = ThemeDark
	// themeTimer 跟随系统时定期复查（系统主题可能在应用运行期间被切换）
	themeTimer *time.Timer
)

// ThemeMode 当前主题模式（dark / light / system）。
func ThemeMode() string { return themeMode }

// IsDark 当前是否深色。
func IsDark() bool { return curTheme.Name == "dark" }

// CurrentTheme 当前生效的色板。
func CurrentTheme() Theme { return curTheme }

// UseTheme 切换主题模式并立即应用（dark / light / system）。
func UseTheme(mode string) {
	switch mode {
	case ThemeDark, ThemeLight, ThemeSystem:
	default:
		mode = ThemeDark
	}
	themeMode = mode
	refreshTheme()
	startThemeWatch()
	RequestRedraw()
}

// applyTheme 把色板写入包级 token。
func applyTheme(t Theme) {
	curTheme = t
	ColText, ColTextDim, ColTextMuted = t.Text, t.TextDim, t.TextMuted
	ColBg, ColBgSoft, ColBgDeep = t.Bg, t.BgSoft, t.BgDeep
	ColStroke, ColLine = t.Stroke, t.Line
	ColPrimary, ColPrimary2 = t.Primary, t.Primary2
	ColSuccess, ColWarn, ColDanger, ColInfo = t.Success, t.Warn, t.Danger, t.Info
	ColInput, ColPanel, ColPanelDeep = t.Input, t.Panel, t.PanelDeep
	ColRow, ColRowAlt, ColHeader = t.Row, t.RowAlt, t.Header
	ColPlaceholder, ColHover, ColScrim, ColTrack = t.Placeholder, t.Hover, t.Scrim, t.Track
	ColGlass = t.Glass
	// 语义 token 快捷变量同步
	colFocus := t.FocusRing
	if colFocus == "" {
		colFocus = t.Primary
	}
	ColFocus = colFocus
	ColSurface, ColSurfaceMuted, ColSurfaceRaised, ColSurfaceDeep = t.Bg, t.BgSoft, t.Panel, t.BgDeep
	// Markdown 自带色板同步刷新（正文/代码块/语法高亮）
	mdApplyTheme(t)
	// 色板变了，Markdown 渲染树缓存必须失效
	mdTouchState()
	// 窗口清屏色跟随主题，但保留 Options.Background 里设定的透明度
	// （圆角边缘的 alpha 过渡不能被覆盖成不透明）
	if app := Current(); app != nil {
		bg := C(t.Bg)
		bg.A = app.bg.A
		app.bg = bg
	}
}

// refreshTheme 依据模式重新解析并应用色板（同色板时不重复应用）。
func refreshTheme() {
	dark := true
	switch themeMode {
	case ThemeLight:
		dark = false
	case ThemeSystem:
		dark = SystemPrefersDark()
	default:
		// 已注册的具名主题：直接应用（不再走深/浅判断）
		if t, ok := themeRegistry[themeMode]; ok {
			applyTheme(themeWithOverrides(t))
			return
		}
	}
	if dark && curTheme.Name == "dark" {
		return
	}
	if !dark && curTheme.Name == "light" {
		return
	}
	if dark {
		applyTheme(themeWithOverrides(Dark))
		return
	}
	applyTheme(themeWithOverrides(Light))
}

// themeWithOverrides 把用户 token 覆盖叠加到基础主题上。
func themeWithOverrides(t Theme) Theme {
	ov := tokenOverrides
	set := func(d *string, v string) {
		if v != "" {
			*d = v
		}
	}
	set(&t.Bg, ov.Surface)
	set(&t.BgSoft, ov.SurfaceMuted)
	set(&t.Panel, ov.SurfaceRaised)
	set(&t.BgDeep, ov.SurfaceDeep)
	set(&t.Text, ov.OnSurface)
	set(&t.TextDim, ov.OnSurfaceMuted)
	set(&t.TextMuted, ov.OnSurfaceFaint)
	set(&t.Stroke, ov.Border)
	set(&t.Line, ov.BorderSubtle)
	set(&t.Primary, ov.Accent)
	set(&t.Primary2, ov.AccentAlt)
	set(&t.Success, ov.Success)
	set(&t.Warn, ov.Warning)
	set(&t.Danger, ov.Danger)
	set(&t.Info, ov.Info)
	set(&t.Scrim, ov.Overlay)
	set(&t.Track, ov.Track)
	set(&t.FocusRing, ov.FocusRing)
	set(&t.Glass, ov.Glass)
	set(&t.Header, ov.Header)
	set(&t.Row, ov.Row)
	set(&t.RowAlt, ov.RowAlt)
	set(&t.Hover, ov.Hover)
	set(&t.Placeholder, ov.Placeholder)
	return t
}

// startThemeWatch 跟随系统时每 3 秒复查一次系统主题。
func startThemeWatch() {
	if themeTimer != nil {
		themeTimer.Stop()
		themeTimer = nil
	}
	if themeMode != ThemeSystem {
		return
	}
	themeTimer = time.AfterFunc(3*time.Second, func() {
		if themeMode != ThemeSystem {
			return
		}
		before := curTheme.Name
		refreshTheme()
		if curTheme.Name != before {
			RequestRedraw()
		}
		startThemeWatch()
	})
}

// ---- 系统主题探测（桌面读注册表，浏览器读 prefers-color-scheme）----

// SystemPrefersDark 读取系统是否使用深色模式（读取失败时按深色处理）。
func SystemPrefersDark() bool { return platformPrefersDark() }

// platformReducedMotion 系统是否开启「减弱动态效果」（各平台文件实现）。
func platformReducedMotion() bool { return platformReducedMotionImpl() }
