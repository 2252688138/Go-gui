// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// mediacheck 无界面地验证当前平台的媒体能力（解码 + 音频输出）。
//
// 用途：CI / WSL / 无头服务器上确认「ffmpeg 解码链路 + 音频后端」是否可用，
// 不需要打开窗口，也不依赖 GL。用法：
//
//	mediacheck                        # 只打印能力表与解码后端
//	mediacheck video.mp4              # 打开视频，按两个时间点取帧并校验尺寸
//	mediacheck song.mp3               # 打开音频，播放 2 秒并观察播放位置是否推进
package main

import (
	"fmt"
	"os"
	"time"

	"gogui/ui"
)

func main() {
	fmt.Printf("caps: video=%v audio=%v dialogs=%v clipboard=%v webview=%v\n",
		ui.HasCap(ui.CapVideo), ui.HasCap(ui.CapAudio),
		ui.HasCap(ui.CapDialogs), ui.HasCap(ui.CapClipboard), ui.HasCap(ui.CapWebView))
	fmt.Printf("decoder backend: %s (%s)\n", ui.BackendName(), ui.BackendLabel(ui.BackendName()))
	if len(os.Args) < 2 {
		return
	}
	path := os.Args[1]
	if ui.IsMusicPath(path) {
		checkMusic(path)
		return
	}
	checkVideo(path)
}

// checkVideo 打开视频并按时间点取帧（验证 ffmpeg 探测 + rawvideo 管道）。
func checkVideo(path string) {
	t0 := time.Now()
	src, err := ui.OpenMedia(path)
	if err != nil {
		fmt.Println("打开视频失败：", err)
		os.Exit(1)
	}
	defer src.Close()
	info := src.Info()
	fmt.Printf("视频 %dx%d %.2ffps 时长 %v 后端 %s（探测耗时 %v）\n",
		info.Width, info.Height, info.FPS, info.Duration.Truncate(time.Millisecond),
		info.Backend, time.Since(t0).Truncate(time.Millisecond))

	for _, at := range []time.Duration{0, 1500 * time.Millisecond} {
		f, err := src.Frame(at)
		if err != nil {
			fmt.Printf("  取帧 @%v 失败：%v\n", at, err)
			continue
		}
		if f == nil || !f.Valid() {
			fmt.Printf("  取帧 @%v：无画面\n", at)
			continue
		}
		sum := 0
		for i := 0; i < len(f.Pix) && i < 4096; i++ {
			sum += int(f.Pix[i])
		}
		fmt.Printf("  取帧 @%v：%dx%d 像素和(前4KB)=%d\n", at, f.W, f.H, sum)
	}
}

// checkMusic 打开音频并播放 2 秒（验证 PCM 管道 + 音频后端是否真的推进）。
func checkMusic(path string) {
	t0 := time.Now()
	src, err := ui.OpenMusic(path)
	if err != nil {
		fmt.Println("打开音频失败：", err)
		os.Exit(1)
	}
	defer src.Close()
	info := src.Info()
	fmt.Printf("音频 时长 %v 采样率 %d 后端 %s（探测耗时 %v）\n",
		info.Duration.Truncate(time.Millisecond), info.SampleRate, info.Backend,
		time.Since(t0).Truncate(time.Millisecond))

	src.Play()
	start := src.Pos()
	time.Sleep(2 * time.Second)
	pos := src.Pos()
	fmt.Printf("播放 2s：位置 %v → %v（推进 %v）\n",
		start.Truncate(time.Millisecond), pos.Truncate(time.Millisecond),
		(pos - start).Truncate(time.Millisecond))
	if err := src.Err(); err != nil {
		fmt.Println("音频管线报错：", err)
	}
}
