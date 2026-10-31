//go:build (linux || freebsd || openbsd || netbsd) && !android

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

// Linux/BSD 端字体来源：系统字体目录 + 用户字体目录。
//
// 不使用 fontconfig（需要额外动态库与配置解析），直接扫目录并按文件名关键词匹配，
// 与 Windows 端「按文件名猜字体」的策略保持一致：零依赖、零 cgo、行为可测。

// fontDirs 返回本机字体目录（按优先级）。
func fontDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		"/usr/share/X11/fonts",
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "fonts"))
	}
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local/share/fonts"), filepath.Join(home, ".fonts"))
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
//
// 返回非空即代表「本平台自认领这次查找」，font.go 不会再走 Windows 的默认路径。
func platformFontCandidates(faceName string, weight int32) []string {
	bold := weight >= 600
	files := scanFontFiles(fontDirs())
	if len(files) == 0 {
		return nil
	}
	name := strings.ToLower(faceName)
	switch {
	case name == "" || containsAny(name, "雅黑", "yahei", "microsoft yahei", "sans", "system-ui", "default"):
		return concat(withBold(files, bold, sansCJKKeywords), pickFonts(files, sansLatinKeywords...))
	case containsAny(name, "宋体", "simsun", "serif", "song", "明体"):
		return withBold(files, bold, serifKeywords)
	case containsAny(name, "黑体", "simhei", "hei"):
		return withBold(files, bold, sansCJKKeywords)
	case containsAny(name, "mono", "consolas", "courier", "等宽"):
		return concat(withBold(files, bold, monoKeywords), pickFonts(files, sansCJKKeywords...))
	case containsAny(name, "emoji", "symbol", "符号"):
		return concat(pickFonts(files, emojiKeywords...), pickFonts(files, sansLatinKeywords...))
	}
	// 未识别的家族：先按名字本身模糊匹配，再回退默认无衬线。
	if out := pickFonts(files, faceName); len(out) > 0 {
		return out
	}
	return concat(withBold(files, bold, sansCJKKeywords), pickFonts(files, sansLatinKeywords...))
}

// platformDefaultFontCandidates 默认字体（未指定 FontFamily 时）。
func platformDefaultFontCandidates(weight int32) []string {
	files := scanFontFiles(fontDirs())
	return concat(withBold(files, weight >= 600, sansCJKKeywords), pickFonts(files, sansLatinKeywords...))
}
