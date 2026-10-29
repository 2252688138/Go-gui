package ui

import "testing"

// TestToastAutoHide 轻提示到点自动隐藏（默认 2 秒）。
func TestToastAutoHide(t *testing.T) {
	ToastShow("已保存")
	if toastState.dur != ToastDefaultSeconds {
		t.Fatalf("默认时长应为 %v 秒，实际 %v", ToastDefaultSeconds, toastState.dur)
	}
	if ToastLayer() == nil {
		t.Fatal("刚显示时应返回节点")
	}
	if !ToastVisible() {
		t.Fatal("刚显示时 ToastVisible 应为 true")
	}
	// 模拟时间过去 2.5 秒
	toastState.at -= ToastDefaultSeconds + 0.5
	if n := ToastLayer(); n != nil {
		t.Fatal("超过默认时长后应自动隐藏")
	}
	if ToastVisible() {
		t.Fatal("隐藏后 ToastVisible 应为 false")
	}
}

// TestToastCustomSeconds 可自定义显示秒数，重复触发会重置计时。
func TestToastCustomSeconds(t *testing.T) {
	ToastShowFor("导出中", 5)
	if toastState.dur != 5 {
		t.Fatalf("自定义时长应为 5 秒，实际 %v", toastState.dur)
	}
	toastState.at -= 4
	if ToastLayer() == nil {
		t.Fatal("5 秒内应仍在显示")
	}
	// 重新触发同一文案：计时归零
	ToastShowFor("导出中", 5)
	if age := AnimTime() - toastState.at; age > 0.2 {
		t.Fatalf("重新触发后应重置计时，age=%v", age)
	}
	// seconds <= 0 回落默认时长
	ToastShowFor("x", 0)
	if toastState.dur != ToastDefaultSeconds {
		t.Fatalf("seconds<=0 应回落默认时长，实际 %v", toastState.dur)
	}
}

// TestToastHide 立即隐藏；空消息等同隐藏。
func TestToastHide(t *testing.T) {
	ToastShow("a")
	ToastHide()
	if ToastLayer() != nil || ToastVisible() {
		t.Fatal("ToastHide 后应不可见")
	}
	ToastShow("b")
	ToastShow("")
	if ToastLayer() != nil {
		t.Fatal("空消息应等同隐藏")
	}
}
