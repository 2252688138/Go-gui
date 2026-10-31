// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "testing"

// TestTrimNilSafe 回收接口必须对空 App / 空渲染器安全：
// 关窗竞态、后台 goroutine 调用、无 GL 上下文的平台都会走到这些路径。
func TestTrimNilSafe(t *testing.T) {
	var a *App
	a.trimCaches(true) // nil App
	(&App{}).trimCaches(false)
	(&App{}).chartReleaseAll()
	(&imageTextures{}).releaseAll(nil)
	(&App{}).logTrim(0, 0, true) // 统计关闭时不输出
}

// TestImageReleaseAll 释放后条数保留但纹理记账归零，便于回到页面时重新解码。
func TestImageReleaseAll(t *testing.T) {
	tx := newImageTextures()
	if n := tx.count(); n != 0 {
		t.Fatalf("初始应为 0，实际 %d", n)
	}
	e := tx.entry("a.png")
	e.bytes = 1024
	tx.bytes = 1024
	tx.releaseAll(nil) // 无渲染器：只清记账，不删纹理
	if tx.bytes != 0 {
		t.Fatalf("记账应归零，实际 %d", tx.bytes)
	}
	if n := tx.count(); n != 1 {
		t.Fatalf("条目应保留以便重新解码，实际 %d", n)
	}
}
