// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 语义化主题 token 层。
//
// 在原有 Theme（深/浅两套具名色板）之上再加一层「语义名」：
// Surface / SurfaceMuted / OnSurface / Border / Accent / Danger / FocusRing …
// 组件写语义名，业务可以整体替换其中任意一项（品牌色、护眼配色），
// 也可以注册第三套完整主题并用 UseThemeName 运行时切换。
//
// 兼容性：原有 ColText / ColBg / ColPrimary 等变量继续有效，applyTheme 时与 token 同步刷新。

// Tokens 语义色板。
type Tokens struct {
	Surface        string // 页面主背景
	SurfaceMuted   string // 次级背景（卡片内分区、按钮底）
	SurfaceRaised  string // 浮层/弹窗背景
	SurfaceDeep    string // 更深背景（代码块、输入框）
	OnSurface      string // 主文字
	OnSurfaceMuted string // 次要文字
	OnSurfaceFaint string // 弱化文字 / 占位
	Border         string // 描边
	BorderSubtle   string // 分隔线
	Accent         string // 主色
	AccentAlt      string // 主色渐变终点
	OnAccent       string // 主色上的文字
	Success        string
	Warning        string
	Danger         string
	Info           string
	Overlay        string // 遮罩
	Track          string // 轨道
	FocusRing      string // 焦点环
	Glass          string // 毛玻璃叠色
	Header         string
	Row            string
	RowAlt         string
	Hover          string
	Placeholder    string
}

// TokensOf 当前主题的语义 token。
func TokensOf() Tokens { return tokensFromTheme(curTheme) }

func tokensFromTheme(t Theme) Tokens {
	return Tokens{
		Surface:        t.Bg,
		SurfaceMuted:   t.BgSoft,
		SurfaceRaised:  t.Panel,
		SurfaceDeep:    t.BgDeep,
		OnSurface:      t.Text,
		OnSurfaceMuted: t.TextDim,
		OnSurfaceFaint: t.TextMuted,
		Border:         t.Stroke,
		BorderSubtle:   t.Line,
		Accent:         t.Primary,
		AccentAlt:      t.Primary2,
		OnAccent:       "#ffffff",
		Success:        t.Success,
		Warning:        t.Warn,
		Danger:         t.Danger,
		Info:           t.Info,
		Overlay:        t.Scrim,
		Track:          t.Track,
		FocusRing:      t.FocusRing,
		Glass:          t.Glass,
		Header:         t.Header,
		Row:            t.Row,
		RowAlt:         t.RowAlt,
		Hover:          t.Hover,
		Placeholder:    t.Placeholder,
	}
}

// tokenOverrides 由 SetTokens 记录的用户覆盖（切换主题后继续生效）。
// 空串表示不覆盖该项。
var tokenOverrides Tokens

// SetTokens 覆盖当前主题的语义色（空串字段保持原值），立即生效并记住覆盖。
func SetTokens(t Tokens) {
	mergeTokens(&tokenOverrides, t)
	applyThemeTokens(tokenOverrides)
	RequestRedraw()
}

// ClearTokenOverrides 清除用户覆盖，回到主题自带配色。
func ClearTokenOverrides() {
	tokenOverrides = Tokens{}
	curTheme.Name = "" // 强制 refreshTheme 重新应用
	refreshTheme()
	RequestRedraw()
}

func mergeTokens(dst *Tokens, src Tokens) {
	set := func(d *string, v string) {
		if v != "" {
			*d = v
		}
	}
	set(&dst.Surface, src.Surface)
	set(&dst.SurfaceMuted, src.SurfaceMuted)
	set(&dst.SurfaceRaised, src.SurfaceRaised)
	set(&dst.SurfaceDeep, src.SurfaceDeep)
	set(&dst.OnSurface, src.OnSurface)
	set(&dst.OnSurfaceMuted, src.OnSurfaceMuted)
	set(&dst.OnSurfaceFaint, src.OnSurfaceFaint)
	set(&dst.Border, src.Border)
	set(&dst.BorderSubtle, src.BorderSubtle)
	set(&dst.Accent, src.Accent)
	set(&dst.AccentAlt, src.AccentAlt)
	set(&dst.OnAccent, src.OnAccent)
	set(&dst.Success, src.Success)
	set(&dst.Warning, src.Warning)
	set(&dst.Danger, src.Danger)
	set(&dst.Info, src.Info)
	set(&dst.Overlay, src.Overlay)
	set(&dst.Track, src.Track)
	set(&dst.FocusRing, src.FocusRing)
	set(&dst.Glass, src.Glass)
	set(&dst.Header, src.Header)
	set(&dst.Row, src.Row)
	set(&dst.RowAlt, src.RowAlt)
	set(&dst.Hover, src.Hover)
	set(&dst.Placeholder, src.Placeholder)
}

// applyThemeTokens 把用户覆盖写入当前主题并同步包级变量。
func applyThemeTokens(ov Tokens) {
	t := curTheme
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
	applyTheme(t)
}

// ---- 主题注册表（运行时切换第三套主题）----

var themeRegistry = map[string]Theme{}

// RegisterTheme 注册一套具名主题（name 为 "dark"/"light" 时会覆盖内置主题）。
func RegisterTheme(name string, t Theme) {
	if name == "" {
		return
	}
	t.Name = name
	themeRegistry[name] = t
}

// ThemeNames 已注册的主题名（含内置 dark / light）。
func ThemeNames() []string {
	out := []string{ThemeDark, ThemeLight}
	for n := range themeRegistry {
		if n != ThemeDark && n != ThemeLight {
			out = append(out, n)
		}
	}
	return out
}

// UseThemeName 切换到已注册/内置的具名主题；未知名称保持当前主题。
func UseThemeName(name string) {
	t, ok := themeRegistry[name]
	if !ok {
		switch name {
		case ThemeDark:
			t, ok = Dark, true
		case ThemeLight:
			t, ok = Light, true
		}
	}
	if !ok {
		return
	}
	themeMode = name
	applyThemeTokens(tokenOverrides)
	applyTheme(t)
	RequestRedraw()
}

// ThemeTitle 主题显示名（界面切换器用）。
func ThemeTitle(name string) string {
	switch name {
	case ThemeDark:
		return "深色"
	case ThemeLight:
		return "浅色"
	case ThemeSystem:
		return "跟随系统"
	}
	if t, ok := themeRegistry[name]; ok && t.Name != "" {
		return t.Name
	}
	return name
}
