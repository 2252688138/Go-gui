// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"strings"
	"testing"
)

// TestVListWindow 虚拟列表只创建可见行。
func TestVListWindow(t *testing.T) {
	first, last := VListVisible(100000, 4000, 400, 40, 4)
	if first != 96 {
		t.Fatalf("起始行应为 96，实际 %d", first)
	}
	if last != 115 {
		t.Fatalf("结束行应为 115，实际 %d", last)
	}
	n := VList("v", 100000, 40, 400, func(i int) *Node {
		return Text("row").St(St().H(40))
	})
	if n == nil || n.Kind != KScroll {
		t.Fatal("VList 应返回滚动容器")
	}
	if len(n.Children) > 40 {
		t.Fatalf("只应创建可见行 + 占位，实际 %d 个子节点", len(n.Children))
	}
}

// TestVListEmpty 空列表返回空状态而不是崩溃。
func TestVListEmpty(t *testing.T) {
	n := VList("v0", 0, 40, 300, func(i int) *Node { return nil })
	if n == nil || n.Kind != KScroll {
		t.Fatal("空列表也应返回滚动容器")
	}
}

// TestSplitPosClampAndPref 分栏比例夹取与持久化。
func TestSplitPosClampAndPref(t *testing.T) {
	SetSplitPos("s1", 1.5, "")
	if got := SplitPos("s1"); got != 1 {
		t.Fatalf("比例应夹到 1，实际 %v", got)
	}
	SetSplitPos("s1", -0.4, "test.split")
	if got := SplitPos("s1"); got != 0 {
		t.Fatalf("比例应夹到 0，实际 %v", got)
	}
	if v := GetPrefFloat("test.split", -1); v != 0 {
		t.Fatalf("偏好应写入 0，实际 %v", v)
	}
	n := Split("s2", Text("left"), Text("right"), SplitOptions{Pos: 0.3, MinA: 120, MinB: 80})
	Layout(n, 800, 400, newTestCache())
	if n.Frame.W <= 0 {
		t.Fatal("分栏布局宽度应大于 0")
	}
}

// TestFuzzyCommand 模糊匹配评分与命令注册。
func TestFuzzyCommand(t *testing.T) {
	if FuzzyScore("sv", "save file") < 0 {
		t.Fatal("子序列应匹配")
	}
	if FuzzyScore("xyz", "save file") >= 0 {
		t.Fatal("不存在的字符不应匹配")
	}
	if FuzzyScore("save", "save") <= FuzzyScore("save", "save as file forever") {
		t.Fatal("完全匹配应优先于长文本")
	}
	RegisterCommand(Command{ID: "t.a", Title: "打开文件", Group: "文件", Run: func() {}})
	RegisterCommand(Command{ID: "t.b", Title: "保存文件", Group: "文件", Run: func() {}})
	list := matchCommands("保存")
	if len(list) != 1 || list[0].ID != "t.b" {
		t.Fatalf("过滤结果不正确：%+v", list)
	}
	UnregisterCommand("t.a")
	UnregisterCommand("t.b")
}

// TestColorHSV 取色器 HSV 与 HEX 往返。
func TestColorHSV(t *testing.T) {
	names := []string{"#ff0000", "#00ff00", "#0000ff", "#6366f1", "#ffffff", "#000000"}
	for _, hex := range names {
		r, g, b, ok := parseHexRGB(hex)
		if !ok {
			t.Fatalf("解析失败 %s", hex)
		}
		h, s, v := rgbToHSV(r, g, b)
		r2, g2, b2 := hsvToRGB(h, s, v)
		diff := absIntN(r-r2) + absIntN(g-g2) + absIntN(b-b2)
		if diff > 3 {
			t.Fatalf("HSV 往返偏差过大 %s → %d,%d,%d", hex, r2, g2, b2)
		}
	}
}

func absIntN(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// TestCompareVersion 语义化版本比较。
func TestCompareVersion(t *testing.T) {
	if CompareVersion("1.2.3", "1.2.3") != 0 {
		t.Fatal("相同版本应返回 0")
	}
	if CompareVersion("1.2.4", "1.2.3") != 1 {
		t.Fatal("更高补丁号应返回 1")
	}
	if CompareVersion("v1.2", "1.1.9") != 1 {
		t.Fatal("v 前缀应被忽略")
	}
	if CompareVersion("1.0.0-beta", "1.0.0") != 0 {
		t.Fatal("预发布后缀应被忽略")
	}
	if CompareVersion("2", "10") != -1 {
		t.Fatal("数字段应按数值比较")
	}
}

// TestMoveSlice 拖拽重排。
func TestMoveSlice(t *testing.T) {
	got := MoveSlice([]string{"a", "b", "c", "d"}, 0, 2)
	if strings.Join(got, "") != "bcad" {
		t.Fatalf("向前移动结果错误：%v", got)
	}
	got = MoveSlice(got, 3, 1)
	if strings.Join(got, "") != "bdca" {
		t.Fatalf("向后移动结果错误：%v", got)
	}
}

// TestUndoStack 撤销 / 重做 / 标签。
func TestUndoStack(t *testing.T) {
	u := NewUndo()
	v := 0
	u.Do("加一", func() { v++ }, func() { v-- })
	u.Do("加十", func() { v += 10 }, func() { v -= 10 })
	if v != 11 {
		t.Fatalf("执行后应为 11，实际 %d", v)
	}
	u.Undo()
	if v != 1 {
		t.Fatalf("撤销后应为 1，实际 %d", v)
	}
	u.Undo()
	if v != 0 {
		t.Fatalf("再撤销应为 0，实际 %d", v)
	}
	u.Redo()
	if v != 1 {
		t.Fatalf("重做后应为 1，实际 %d", v)
	}
	if u.Label() != "加一" {
		t.Fatalf("撤销标签应为「加一」，实际 %q", u.Label())
	}
}

// TestShortcutConflict 快捷键冲突检测。
func TestShortcutConflict(t *testing.T) {
	ok1 := RegisterShortcut(ScopeGlobal, Shortcut{ID: "t.save", Keys: "Ctrl+Shift+S",
		Label: "保存", Run: func() {}})
	if !ok1 {
		t.Fatal("首次注册应成功")
	}
	ok2 := RegisterShortcut(ScopeGlobal, Shortcut{ID: "t.save2", Keys: "shift+ctrl+s",
		Label: "另存", Run: func() {}})
	if ok2 {
		t.Fatal("同组合重复注册应失败")
	}
	if ShortcutConflict(ScopeGlobal, "Ctrl+Shift+S") != "t.save" {
		t.Fatal("冲突检测应返回已占用 ID")
	}
	UnregisterShortcut(ScopeGlobal, "t.save")
	UnregisterShortcut(ScopeGlobal, "t.save2")
}

// TestI18nFallback 翻译缺失回退。
func TestI18nFallback(t *testing.T) {
	if err := LoadCatalog("xx", `{"save":"Save"}`); err != nil {
		t.Fatal(err)
	}
	SetLocale("xx")
	if T("save") != "Save" {
		t.Fatalf("翻译未生效：%q", T("save"))
	}
	if T("open_file") != "open file" {
		t.Fatalf("缺失 key 应回退为可读文本，实际 %q", T("open_file"))
	}
	SetLocale("zh-CN")
	SetRTL(false)
}

// TestEllipsisToggle 多行省略与展开。
func TestEllipsisToggle(t *testing.T) {
	text := strings.Repeat("很长的一段文字，", 40)
	n := EllipsisText("e1", text, 2, 200, EllipsisOptions{Expandable: true})
	if n == nil {
		t.Fatal("可展开时应返回节点")
	}
	ellipsisOpen["e1"] = true
	n2 := EllipsisText("e1", text, 2, 200, EllipsisOptions{Expandable: true})
	if n2 == nil {
		t.Fatal("展开态应正常渲染")
	}
	delete(ellipsisOpen, "e1")
}
