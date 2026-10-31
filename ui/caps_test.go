// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// 能力表是「一套 UI 多端跑」的契约：平台差异只能体现在这里，
// 因此断言各端都满足：已知能力、无重复、HasCap 与列表一致、缺失能力可枚举。

func TestCapabilitiesAreKnownAndUnique(t *testing.T) {
	seen := map[Cap]bool{}
	for _, c := range Capabilities() {
		if c < CapVideo || c > CapLast {
			t.Fatalf("未知能力值：%d", c)
		}
		if seen[c] {
			t.Fatalf("能力重复声明：%s", CapName(c))
		}
		seen[c] = true
		if CapName(c) == "未知能力" {
			t.Fatalf("能力 %d 缺少名称", c)
		}
	}
}

func TestHasCapMatchesCapabilities(t *testing.T) {
	for c := CapVideo; c <= CapLast; c++ {
		want := false
		for _, v := range Capabilities() {
			if v == c {
				want = true
				break
			}
		}
		if got := HasCap(c); got != want {
			t.Fatalf("HasCap(%s) = %v, 期望 %v", CapName(c), got, want)
		}
	}
}

func TestUnsupportedCapsIsComplement(t *testing.T) {
	missing := UnsupportedCaps()
	for _, c := range missing {
		if HasCap(c) {
			t.Fatalf("%s 同时出现在支持与缺失列表", CapName(c))
		}
	}
	// 支持数 + 缺失数 == 全部能力
	if len(Capabilities())+len(missing) != int(CapLast)+1 {
		t.Fatalf("支持 %d + 缺失 %d != %d", len(Capabilities()), len(missing), int(CapLast)+1)
	}
}

// 关键约定：不论哪个平台，能力查询本身永远可用（不能 panic / 不能返回 nil 语义混乱）。
func TestCapabilitiesNeverPanic(t *testing.T) {
	caps := Capabilities()
	if caps == nil {
		// nil 是合法返回值（表示一个能力都没有），只要不 panic 即可
		_ = len(caps)
	}
	_ = UnsupportedCaps()
}
