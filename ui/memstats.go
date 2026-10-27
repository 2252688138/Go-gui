//go:build windows

package ui

import (
	"fmt"
	"os"
	"runtime"
)

// memStatsEnabled 由环境变量 GOGUI_MEMSTATS=1 开启：每 120 帧输出一次内存构成，
// 便于定位占用来源（字形图集 / Go 堆 / 每帧节点树）。
var memStatsEnabled = os.Getenv("GOGUI_MEMSTATS") != ""

// logMemStats 输出内存统计（默认关闭，开启后写 stderr）。
func (a *App) logMemStats() {
	if !memStatsEnabled || a.frameCount%120 != 0 {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	atlas := 0.0
	fonts := 0
	if a.fonts != nil {
		atlas = float64(a.fonts.atlasBytes()) / (1 << 20)
		fonts = len(a.fonts.fonts)
	}
	imgs := 0
	if a.textures != nil {
		imgs = a.textures.count()
	}
	fmt.Fprintf(os.Stderr,
		"[gogui] frames=%d heap=%.1fMB inuse=%.1fMB sys=%.1fMB gc=%d fonts=%d atlas=%.1fMB images=%d\n",
		a.frameCount,
		float64(m.HeapAlloc)/(1<<20),
		float64(m.HeapInuse)/(1<<20),
		float64(m.Sys)/(1<<20),
		m.NumGC, fonts, atlas, imgs)
}
