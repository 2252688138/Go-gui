//go:build js && wasm

package text

import (
	"errors"
	"sync"
)

// 浏览器端没有系统字体目录：字体由 Web 运行时用 fetch 取回后注册进来
// （见 ui.SetWebFont），之后 mapFontFile / fontCandidates 都从这张表里查。

var (
	webFontMu   sync.RWMutex
	webFonts    = map[string][]byte{} // 名字/路径 -> 字体字节
	webFontList []string
	webFontDef  string // 默认字体名（"" = 第一个注册的）
)

// RegisterFont 注册一份字体数据（ttf/otf/ttc 均可）。name 用于 FontFamily 匹配。
func RegisterFont(name string, data []byte) {
	webFontMu.Lock()
	defer webFontMu.Unlock()
	if _, ok := webFonts[name]; !ok {
		webFontList = append(webFontList, name)
	}
	webFonts[name] = data
	if webFontDef == "" {
		webFontDef = name
	}
}

// SetDefaultFont 指定默认字体名。
func SetDefaultFont(name string) {
	webFontMu.Lock()
	webFontDef = name
	webFontMu.Unlock()
}

// HasFont 是否已注册字体。
func HasFont() bool {
	webFontMu.RLock()
	defer webFontMu.RUnlock()
	return len(webFonts) > 0
}

func platformMapFontFile(path string) ([]byte, error) {
	webFontMu.RLock()
	defer webFontMu.RUnlock()
	if b, ok := webFonts[path]; ok {
		return b, nil
	}
	return nil, errors.New("font not registered: " + path)
}

// platformFontCandidates 浏览器端候选：优先按家族名 + 字重挑，其次家族名模糊匹配，
// 最后回退默认字体。注册时可把粗体存成 "<名字>/Bold"（见 ui.RegisterFontBold）。
func platformFontCandidates(faceName string, weight int32) []string {
	webFontMu.RLock()
	defer webFontMu.RUnlock()
	bold := weight >= 600
	var out []string
	if faceName != "" {
		if bold {
			for _, n := range webFontList {
				if containsAny(n, faceName) && containsAny(n, "Bold") {
					out = append(out, n)
				}
			}
		}
		for _, n := range webFontList {
			if faceName == "" || containsAny(n, faceName) || containsAny(faceName, n) {
				out = append(out, n)
			}
		}
	}
	if len(out) == 0 && webFontDef != "" {
		out = append(out, webFontDef)
	}
	return out
}
