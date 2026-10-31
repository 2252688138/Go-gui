// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestSuperSampleForDPI 自动档：DPI < 150% 走 2×，≥150% 直接 1:1。
func TestSuperSampleForDPI(t *testing.T) {
	type c struct {
		dpi, mode, want float32
	}
	cases := []c{
		{1.0, SSAuto, 2},
		{1.25, SSAuto, 2},
		{1.49, SSAuto, 2},
		{1.5, SSAuto, 1},
		{2.0, SSAuto, 1},
		{1.0, SSOff, 1},
		{2.0, SSOff, 1},
		{1.0, SS125, 1.25},
		{1.0, SS150, 1.5},
		{1.0, SS200, 2},
		{1.0, 0.5, 1}, // 非法档位（0~1 之间）按关闭处理
		{1.0, 9, 2},   // 超过上限按 2× 处理
	}
	for _, k := range cases {
		if got := superSampleFor(k.dpi, k.mode); got != k.want {
			t.Fatalf("superSampleFor(dpi=%.2f, mode=%.2f) = %.2f, want %.2f", k.dpi, k.mode, got, k.want)
		}
	}
}

// TestSuperSamplePixelBudget 像素预算：窗口过大时逐级降档，绝不超过预算。
func TestSuperSamplePixelBudget(t *testing.T) {
	type c struct {
		w, h int32
		in   float32
		want float32
	}
	cases := []c{
		{1920, 1080, 2, 2},   // 1080p × 2 = 8.3M，预算内
		{2560, 1440, 2, 1.5}, // 2K × 2 = 14.7M 超预算（9.4M），降到 1.5
		{3840, 2160, 2, 1},   // 4K × 2 = 33.2M，只能回到 1
		{7680, 4320, 2, 1},   // 已到预算边界，只能回到 1
		{8000, 4000, 2, 1},
		{1920, 1080, 1, 1}, // 关闭时不变
	}
	for _, k := range cases {
		got := ssaaForBudget(k.w, k.h, k.in)
		if got != k.want {
			t.Fatalf("ssaaForBudget(%dx%d, %.2f) = %.2f, want %.2f", k.w, k.h, k.in, got, k.want)
		}
		if float64(k.w)*float64(k.h)*float64(got)*float64(got) > ssPixelBudget && got > 1.01 {
			t.Fatalf("降档后仍超预算：%dx%d × %.2f", k.w, k.h, got)
		}
	}
}

// TestSetSuperSampleClamp 档位夹取与读取。
func TestSetSuperSampleClamp(t *testing.T) {
	defer SetSuperSample(SSAuto)
	SetSuperSample(9)
	if got := SuperSampleMode(); got != 2 {
		t.Fatalf("超上限应夹到 2，实际 %.2f", got)
	}
	SetSuperSample(0.5)
	if got := SuperSampleMode(); got != 1 {
		t.Fatalf("0~1 之间应夹到 1，实际 %.2f", got)
	}
	SetSuperSample(SSAuto)
	if got := SuperSampleMode(); got != SSAuto {
		t.Fatalf("自动档应为 0，实际 %.2f", got)
	}
}
