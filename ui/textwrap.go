package ui

import "gogui/internal/text"

// wrapTextLines 按可用宽度折行（贪心 + 断行优先级：空格 / 连字符 / CJK 字符边界）。
//
// 布局测量（layoutCtx.measure）与绘制（App.paintText）共用同一函数，
// 保证「测量出来的行数/高度」与「实际绘制的行数」完全一致。
// 以前 WrapOn 只影响测量、绘制侧仍按单行画，导致长文本溢出容器。
func wrapTextLines(f *text.Font, s string, maxW float32) []string {
	if f == nil {
		return []string{s}
	}
	if maxW <= 0 {
		maxW = 1
	}
	rs := []rune(s)
	out := make([]string, 0, 4)
	start := 0
	width := float32(0)
	breakAt := -1 // 可断行位置（该下标之前的内容成行）

	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r == '\n' {
			out = append(out, string(rs[start:i]))
			start = i + 1
			width = 0
			breakAt = -1
			continue
		}
		adv := f.GlyphFor(r).Advance / f.Scale()
		if width+adv > maxW && i > start {
			cut := i
			if breakAt > start && breakAt < i {
				cut = breakAt
			}
			out = append(out, string(rs[start:cut]))
			start = cut
			// 断行点后的空格不带到下一行
			for start < len(rs) && rs[start] == ' ' {
				start++
			}
			width = 0
			breakAt = -1
			i = start - 1
			continue
		}
		width += adv
		switch {
		case r == ' ' || r == '\t' || r == '-' || r == '/' || r == '\u00a0':
			breakAt = i + 1
		case isWideRune(r):
			// 中文/日文/韩文逐字可断；收尾标点前不断（断在标点之前）
			if isCJKPunct(r) {
				breakAt = i
			} else {
				breakAt = i + 1
			}
		}
	}
	out = append(out, string(rs[start:]))
	return out
}

// isWideRune 是否为全角（CJK）字符。
func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0x303E, // CJK Radicals / Kangxi / CJK Symbols
		r >= 0x3041 && r <= 0x33FF, // Hiragana..CJK Compatibility
		r >= 0x3400 && r <= 0x4DBF, // CJK Ext A
		r >= 0x4E00 && r <= 0x9FFF, // CJK Unified
		r >= 0xA000 && r <= 0xA4CF, // Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul Syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK Compatibility Ideographs
		r >= 0xFE30 && r <= 0xFE4F, // CJK Compatibility Forms
		r >= 0xFF00 && r <= 0xFF60, // Fullwidth Forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1FAFF: // Emoji
		return true
	}
	return false
}

// isCJKPunct 中文收尾标点（不在其前面断行会更好看，这里断在其之前）。
func isCJKPunct(r rune) bool {
	switch r {
	case '，', '。', '、', '；', '：', '！', '？', '）', '】', '》', '」', '』', '”', '’',
		',', '.', ';', ':', '!', '?', ')', ']', '}':
		return true
	}
	return false
}
