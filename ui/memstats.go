// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"os"
	"runtime"
)

// memStatsEnabled 由环境变量 GOGUI_MEMSTATS=1 开启：每 120 帧输出一次内存构成，
// 便于定位占用来源（字形图集 / Go 堆 / 每帧节点树 / GPU 纹理）。
var memStatsEnabled = os.Getenv("GOGUI_MEMSTATS") != ""

// logTrim 输出一次空闲/最小化回收的结果（需 GOGUI_MEMSTATS=1）。
// 回收后不会再有新帧，逐 120 帧的常规统计刷新不到，因此单独打一行。
func (a *App) logTrim(offBefore, offAfter int64, deep bool) {
	if !memStatsEnabled {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	level := "cache"
	if deep {
		level = "cache+pages"
	}
	fmt.Fprintf(os.Stderr, "[gogui] idle-trim(%s) frame=%d heap=%.1fMB offscreen=%.1fMB->%.1fMB\n",
		level, a.frameCount, float64(m.HeapAlloc)/(1<<20),
		float64(offBefore)/(1<<20), float64(offAfter)/(1<<20))
}

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
	// GPU 侧：离屏目标（超采样 RT + 毛玻璃中间层）与纹理缓存。
	// 这两项才是常驻显存的大头，Go 堆通常只有十几 MB。
	var off, imgBytes int64
	if a.r != nil {
		off = a.r.OffscreenBytes()
	}
	if a.textures != nil {
		a.textures.mu.Lock()
		imgBytes = a.textures.bytes
		a.textures.mu.Unlock()
	}
	fmt.Fprintf(os.Stderr,
		"[gogui] frames=%d heap=%.1fMB inuse=%.1fMB sys=%.1fMB gc=%d fonts=%d atlas=%.1fMB images=%d imgtex=%.1fMB charttex=%.1fMB offscreen=%.1fMB\n",
		a.frameCount,
		float64(m.HeapAlloc)/(1<<20),
		float64(m.HeapInuse)/(1<<20),
		float64(m.Sys)/(1<<20),
		m.NumGC, fonts, atlas, imgs,
		float64(imgBytes)/(1<<20),
		float64(a.chartBytes)/(1<<20),
		float64(off)/(1<<20))
}
