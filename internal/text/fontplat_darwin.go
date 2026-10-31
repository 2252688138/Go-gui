//go:build darwin && !ios

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"os"
	"path/filepath"
	"strings"
)

// macOS 端字体来源：系统字体（PingFang SC / Hiragino Sans / Helvetica）与用户字体。
//
// macOS 的中文字体多在 .ttc 集合里（如 /System/Library/Fonts/PingFang.ttc），
// 现有 loadSystemFont 会用 opentype.ParseCollection 取第一个可用子字体，
// 因此 TTC 集合不需要额外处理。

// fontDirs 返回本机字体目录（按优先级）。
func fontDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		"/System/Library/Fonts",
		"/System/Library/Fonts/Supplemental",
		"/Library/Fonts",
	}
	if home != "" {
		dirs = append(dirs, filepath.Join(home, "Library/Fonts"))
	}
	return dirs
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
	case name == "" || containsAny(name, "pingfang", "heiti", "sans", "system-ui", "default", "雅黑"):
		return concat(withBold(files, bold, sansCJKKeywords), pickFonts(files, sansLatinKeywords...))
	case containsAny(name, "songti", "serif", "宋体", "times"):
		return withBold(files, bold, serifKeywords)
	case containsAny(name, "hei", "黑体"):
		return withBold(files, bold, sansCJKKeywords)
	case containsAny(name, "mono", "menlo", "monaco", "courier", "等宽"):
		return concat(withBold(files, bold, monoKeywords), pickFonts(files, sansCJKKeywords...))
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
