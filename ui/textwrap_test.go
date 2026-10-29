package ui

import (
	"strings"
	"testing"

	"gogui/internal/text"
)

// TestForEachLineMatchesSplit 逐行迭代必须与旧的 splitLines 完全等价
// （绘制/测量已改为零分配迭代，任何差异都会导致行高与绘制行数不一致）。
func TestForEachLineMatchesSplit(t *testing.T) {
	cases := []string{"", "abc", "a\nb", "a\nb\n", "\n\n\n", "多行\t文本\n第二行"}
	for _, s := range cases {
		want := splitLines(s)
		var got []string
		forEachLine(s, func(ln string) { got = append(got, ln) })
		if len(got) != len(want) {
			t.Fatalf("%q: 行数 %d != %d (%v vs %v)", s, len(got), len(want), got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%q: 第 %d 行 %q != %q", s, i, got[i], want[i])
			}
		}
		if n := lineCount(s); n != len(want) {
			t.Fatalf("%q: lineCount=%d 期望 %d", s, n, len(want))
		}
	}
}

// TestWrapTextRunsMatchesLines 零分配的折行实现必须与切片版结果一致。
func TestWrapTextRunsMatchesLines(t *testing.T) {
	fc := newFontCache("Microsoft YaHei UI", 1)
	defer fc.close()
	f := fc.get(14, 400)
	if f == nil {
		t.Skip("系统字体不可用，跳过折行一致性测试")
	}
	samples := []string{
		"Hello world, this is a long english sentence used to test wrapping behaviour.",
		"中文测试：这是一段用于验证折行结果一致性的较长文本，需要在指定宽度内换行。",
		"混合 mixed 内容 with English 和中文，以及 - 连字符 / 斜杠 break-points。",
		strings.Repeat("abcdefghij ", 12),
	}
	widths := []float32{60, 120, 240, 800}
	for _, s := range samples {
		for _, w := range widths {
			want := wrapTextLines(f, s, w)
			var got []string
			wrapTextRuns(f, s, w, func(ln string) { got = append(got, ln) })
			if len(got) != len(want) {
				t.Fatalf("宽度 %.0f：行数 %d != %d", w, len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("宽度 %.0f 第 %d 行：%q != %q", w, i, got[i], want[i])
				}
			}
			if n := wrapTextCount(f, s, w); n != len(want) {
				t.Fatalf("宽度 %.0f：wrapTextCount=%d 期望 %d", w, n, len(want))
			}
		}
	}
}

// TestMeasureCacheStable 行宽缓存不能改变测量结果（含中文与重复测量）。
func TestMeasureCacheStable(t *testing.T) {
	fc := newFontCache("Microsoft YaHei UI", 1)
	defer fc.close()
	f := fc.get(14, 400)
	if f == nil {
		t.Skip("系统字体不可用")
	}
	for _, s := range []string{"Hello", "中文测试", "Hello 中文 123"} {
		w1 := f.Measure(s)
		w2 := f.Measure(s)
		if w1 <= 0 || w1 != w2 {
			t.Fatalf("%q: 测量结果不稳定 %.2f / %.2f", s, w1, w2)
		}
	}
}

// TestColorCacheStable 颜色解析缓存不能改变解析结果。
func TestColorCacheStable(t *testing.T) {
	for _, s := range []string{"#0b0f19", "#fff", "#ffffffcc", "rgb(10,20,30)", "rgba(1,2,3,0.5)", "transparent"} {
		a := parseColor(s)
		b := parseColor(s)
		if a != b {
			t.Fatalf("%q: 缓存前后不一致 %+v / %+v", s, a, b)
		}
	}
}

// benchText 折行基准用的长文本（中英混排）。
const benchText = "这是一段用于验证折行性能的较长中文文本，包含 English words 与标点符号。" +
	"它应当被折成多行，用于衡量逐帧折行的开销。The quick brown fox jumps over the lazy dog. " +
	"重复若干次以拉长段落：这是一段用于验证折行性能的较长中文文本。"

func benchFont(b *testing.B) *text.Font {
	b.Helper()
	fc := newFontCache("Microsoft YaHei UI", 1)
	b.Cleanup(fc.close)
	f := fc.get(14, 400)
	if f == nil {
		b.Skip("系统字体不可用")
	}
	return f
}

// BenchmarkWrapTextUncached 每帧重新折行（旧行为）。
func BenchmarkWrapTextUncached(b *testing.B) {
	f := benchFont(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if lines := wrapTextLines(f, benchText, 320); len(lines) < 2 {
			b.Fatal("折行结果异常")
		}
	}
}

// BenchmarkWrapTextCached 命中折行缓存（现在的热路径）。
func BenchmarkWrapTextCached(b *testing.B) {
	f := benchFont(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if lines := wrapLinesCached(f, benchText, 320); len(lines) < 2 {
			b.Fatal("折行结果异常")
		}
	}
}

// BenchmarkMeasureUncached 逐字测量（绕过缓存，模拟旧实现）。
func BenchmarkMeasureUncached(b *testing.B) {
	f := benchFont(b)
	const s = "设置面板 · 渲染与显示 1280×800"
	// 预热：先把字形全部光栅化，否则一次性光栅化开销会被摊到每次迭代上
	for _, r := range s {
		_ = f.GlyphFor(r)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var w float32
		for _, r := range s {
			w += f.GlyphFor(r).Advance
		}
		_ = w
	}
}

// BenchmarkMeasureCached 测量缓存命中（现在的热路径）。
func BenchmarkMeasureCached(b *testing.B) {
	f := benchFont(b)
	const s = "设置面板 · 渲染与显示 1280×800"
	_ = f.Measure(s) // 预热缓存
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Measure(s)
	}
}
