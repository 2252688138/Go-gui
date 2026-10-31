// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"encoding/json"
	"os"
	"strings"
)

// 国际化与 RTL（P3 骨架）。
//
//	ui.LoadCatalog("zh-CN", `{"save":"保存"}`)
//	ui.SetLocale("zh-CN")
//	ui.T("save")            // 保存
//	ui.SetRTL(true)         // 行布局镜像 + 文本右对齐（业务用 ui.MirrorRow）
//
// 说明：完整 bidi（阿拉伯语整形、混排）不在本轮范围；这里提供字符串外部化、
// 缺省回退与布局镜像助手，足够支撑「中/英 + 简单 RTL」场景。

var (
	catalogs   = map[string]map[string]string{}
	curLocale  = "zh-CN"
	rtlEnabled = false
)

// LoadCatalog 载入一个语言的字符串表（JSON：{"key":"文本"}）。
func LoadCatalog(locale, jsonBody string) error {
	m := map[string]string{}
	if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
		return err
	}
	catalogs[locale] = m
	return nil
}

// LoadCatalogFile 从文件载入字符串表。
func LoadCatalogFile(locale, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return LoadCatalog(locale, string(raw))
}

// SetLocale 切换语言（未知语言保持当前）。
func SetLocale(locale string) {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return
	}
	curLocale = locale
	RequestRedraw()
}

// Locale 当前语言。
func Locale() string { return curLocale }

// T 取翻译；缺失时回退到 key 本身（并把 _ 替换为空格，至少可读）。
func T(key string) string {
	if m, ok := catalogs[curLocale]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if m, ok := catalogs["zh-CN"]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return strings.ReplaceAll(key, "_", " ")
}

// Tf 带参数的翻译（用 {name} 占位）。
func Tf(key string, args map[string]string) string {
	s := T(key)
	for k, v := range args {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// SetRTL 开启/关闭从右到左布局。
func SetRTL(on bool) {
	rtlEnabled = on
	RequestRedraw()
}

// IsRTL 当前是否 RTL。
func IsRTL() bool { return rtlEnabled }

// DirAlign 文本对齐方向：RTL 时返回 AlignEnd（即右侧）。
func DirAlign() Align {
	if rtlEnabled {
		return AlignEnd
	}
	return AlignStart
}

// MirrorRow 行布局镜像：RTL 时反转子节点顺序。
func MirrorRow(children ...*Node) *Node {
	if rtlEnabled {
		for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
			children[i], children[j] = children[j], children[i]
		}
	}
	return Row(children...)
}

// MirrorIcon 需要镜像的方向图标（← →、‹ ›）。
func MirrorIcon(glyph string) string {
	if !rtlEnabled {
		return glyph
	}
	switch glyph {
	case "→":
		return "←"
	case "←":
		return "→"
	case "›":
		return "‹"
	case "‹":
		return "›"
	}
	return glyph
}
