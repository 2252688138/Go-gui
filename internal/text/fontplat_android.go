//go:build android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"os"
	"strings"
)

// Android 端字体来源：只读系统分区里的 /system/fonts（Noto Sans CJK / Roboto）。
//
// Android 没有 fontconfig，App 也不能写系统字体目录；APK 自带字体时可用
// ui.RegisterFont 直接注册字节（走内存字体表）。

// fontDirs 返回 Android 字体目录（按优先级）。
func fontDirs() []string {
	return []string{
		"/system/fonts",
		"/product/fonts",
		"/system/font",
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
	case name == "" || containsAny(name, "noto", "sans", "system-ui", "default", "roboto", "雅黑"):
		return withBold(files, bold, androidSansKeywords)
	case containsAny(name, "serif", "song", "宋体"):
		return withBold(files, bold, serifKeywords)
	case containsAny(name, "mono", "等宽"):
		return withBold(files, bold, monoKeywords)
	case containsAny(name, "emoji", "symbol", "符号"):
		return concat(pickFonts(files, emojiKeywords...), pickFonts(files, androidSansKeywords...))
	}
	if out := pickFonts(files, faceName); len(out) > 0 {
		return out
	}
	return withBold(files, bold, androidSansKeywords)
}

// platformDefaultFontCandidates 默认字体（Noto Sans CJK → Roboto）。
func platformDefaultFontCandidates(weight int32) []string {
	files := scanFontFiles(fontDirs())
	return withBold(files, weight >= 600, androidSansKeywords)
}

// androidSansKeywords Android 无衬线字族关键词（CJK 优先，随后 Roboto）。
var androidSansKeywords = concat(append([]string{}, sansCJKKeywords...), []string{"Roboto-Regular", "Roboto", "DroidSans"})
