//go:build ios

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"os"
	"strings"
)

// iOS 端字体来源：系统字体目录（PingFang SC / Helvetica）。
//
// 沙盒内不能读系统外的字体目录；App 自带字体请用 ui.RegisterFont 注册字节。

// fontDirs 返回 iOS 字体目录（按优先级）。
func fontDirs() []string {
	return []string{
		"/System/Library/Fonts/Core",
		"/System/Library/Fonts/CoreAddition",
		"/System/Library/Fonts/CoreUI",
		"/System/Library/Fonts/LanguageSupport",
		"/System/Library/Fonts",
	}
}

// platformMapFontFile 以内存映射方式读取字体文件（见 fontscan.go）。
func platformMapFontFile(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, os.ErrNotExist
	}
	return mmapFontFile(path)
}

// platformFontCandidates 按家族名给出候选字体文件。
func platformFontCandidates(faceName string, weight int32) []string {
	bold := weight >= 600
	files := scanFontFiles(fontDirs())
	if len(files) == 0 {
		return nil
	}
	name := strings.ToLower(faceName)
	switch {
	case name == "" || containsAny(name, "pingfang", "helvetica", "sans", "system-ui", "default", "雅黑"):
		return concat(withBold(files, bold, sansCJKKeywords), pickFonts(files, sansLatinKeywords...))
	case containsAny(name, "songti", "serif", "宋体", "times"):
		return withBold(files, bold, serifKeywords)
	case containsAny(name, "mono", "menlo", "courier", "等宽"):
		return withBold(files, bold, monoKeywords)
	case containsAny(name, "emoji", "symbol", "符号"):
		return concat(pickFonts(files, emojiKeywords...), pickFonts(files, sansLatinKeywords...))
	}
	if out := pickFonts(files, faceName); len(out) > 0 {
		return out
	}
	return concat(withBold(files, bold, sansCJKKeywords), pickFonts(files, sansLatinKeywords...))
}

// platformDefaultFontCandidates 默认字体（PingFang SC 优先）。
func platformDefaultFontCandidates(weight int32) []string {
	files := scanFontFiles(fontDirs())
	return concat(withBold(files, weight >= 600, sansCJKKeywords), pickFonts(files, sansLatinKeywords...))
}
