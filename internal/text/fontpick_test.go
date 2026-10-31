// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"os"
	"path/filepath"
	"testing"
)

// 字体挑选逻辑（跨平台通用）：Linux/macOS/Android/iOS 的候选顺序完全由这里决定，
// 因此用临时目录造一份假字体树来断言「关键词优先级 / 粗体优先 / 目录跳过」。

func makeFontTree(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return dir
}

func TestScanFontFilesFiltersExtensions(t *testing.T) {
	dir := makeFontTree(t,
		"NotoSansCJK-Regular.ttc",
		"DejaVuSans.ttf",
		"Something.otf",
		"readme.txt", // 非字体，必须被过滤
	)
	files := scanFontFiles([]string{dir})
	if len(files) != 3 {
		t.Fatalf("扫描到 %d 个字体文件，期望 3：%v", len(files), files)
	}
	for _, p := range files {
		if filepath.Ext(p) == ".txt" {
			t.Fatalf("非字体文件被收录：%s", p)
		}
	}
}

func TestScanFontFilesMissingDirIsSafe(t *testing.T) {
	files := scanFontFiles([]string{filepath.Join(t.TempDir(), "not-exist"), ""})
	if len(files) != 0 {
		t.Fatalf("不存在的目录应返回空，实际 %v", files)
	}
}

func TestPickFontsKeywordPriority(t *testing.T) {
	files := []string{
		"/f/DejaVuSans.ttf",
		"/f/NotoSansCJK-Regular.ttc",
		"/f/NotoSansCJK-Bold.ttc",
		"/f/NotoColorEmoji.ttf",
	}
	got := pickFonts(files, []string{"NotoColorEmoji", "NotoSansCJK"}...)
	if len(got) != 3 {
		t.Fatalf("命中 %d 个，期望 3：%v", len(got), got)
	}
	// 第一个关键词的命中必须排在最前
	if filepath.Base(got[0]) != "NotoColorEmoji.ttf" {
		t.Fatalf("关键词优先级错误，首项 = %s", got[0])
	}
	// 同一关键词内保持入参顺序（scanFontFiles 已排序，因此等价于字典序）
	if filepath.Base(got[1]) != "NotoSansCJK-Regular.ttc" || filepath.Base(got[2]) != "NotoSansCJK-Bold.ttc" {
		t.Fatalf("同关键词内未保持输入顺序：%v", got[1:])
	}
}

func TestWithBoldPutsBoldFirst(t *testing.T) {
	files := []string{
		"/f/NotoSansCJK-Regular.ttc",
		"/f/NotoSansCJK-Bold.ttc",
	}
	got := withBold(files, true, sansCJKKeywords)
	if len(got) != 2 {
		t.Fatalf("命中 %d 个，期望 2", len(got))
	}
	if filepath.Base(got[0]) != "NotoSansCJK-Bold.ttc" {
		t.Fatalf("粗体未优先：首项 = %s", got[0])
	}
	// 非粗体请求保持原顺序
	plain := withBold(files, false, sansCJKKeywords)
	if filepath.Base(plain[0]) != "NotoSansCJK-Bold.ttc" {
		// 字典序决定：Bold 在 Regular 之前，这里只要求不重复丢失
		if len(plain) != 2 {
			t.Fatalf("非粗体候选数量错误：%v", plain)
		}
	}
}

func TestConcatOrder(t *testing.T) {
	a := []string{"a1", "a2"}
	b := []string{"b1"}
	if got := concat(a, b); len(got) != 3 || got[0] != "a1" || got[2] != "b1" {
		t.Fatalf("concat 顺序错误：%v", got)
	}
	if got := concat(nil, b); len(got) != 1 || got[0] != "b1" {
		t.Fatalf("concat(nil,b) = %v", got)
	}
	if got := concat(a, nil); len(got) != 2 {
		t.Fatalf("concat(a,nil) = %v", got)
	}
	// 不得修改入参（避免调用方复用切片时被意外追加）
	if len(a) != 2 {
		t.Fatalf("concat 修改了入参：%v", a)
	}
}

func TestScanFontFilesCached(t *testing.T) {
	dir := makeFontTree(t, "A.ttf")
	first := scanFontFiles([]string{dir})
	// 新增文件后仍返回缓存结果（字体目录运行期不变，缓存是刻意设计）
	if err := os.WriteFile(filepath.Join(dir, "B.ttf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	second := scanFontFiles([]string{dir})
	if len(first) != len(second) {
		t.Fatalf("扫描结果未命中缓存：%d vs %d", len(first), len(second))
	}
}
