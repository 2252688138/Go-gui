//go:build !windows && !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// 非 Windows 桌面端的平台杂项：解码后端命名、系统深色模式、休眠抑制、
// 媒体恢复等。这些能力没有对应的系统 API 封装，因此按「有命令就用命令、
// 没有就安全降级」的方式实现，保持零依赖零 cgo。

// ---- 解码后端命名（Windows 端在 system_decoder.go 里按 ffmpeg / Media Foundation 判断）----

// BackendName 当前可用的解码后端名称（界面展示用）。
// 本平台只有 ffmpeg 一条路径（需要用户自备），因此只判断 PATH 里有没有 ffmpeg。
func BackendName() string {
	if toolPath("", "ffmpeg") != "" && toolPath("", "ffprobe") != "" {
		return "ffmpeg"
	}
	return "none"
}

// BackendLabel 解码后端的中文标签。
func BackendLabel(name string) string {
	switch name {
	case "ffmpeg":
		return "ffmpeg 解码"
	case "system":
		return "系统解码"
	}
	return "无可用解码器"
}

// FormatBackend 后端名短标签（"ffmpeg" / "系统"）。
func FormatBackend(name string) string {
	switch name {
	case "system":
		return "系统"
	case "ffmpeg":
		return "ffmpeg"
	}
	return "—"
}

// ---- 系统深色模式 ----

// platformPrefersDark 读取桌面环境的主题偏好；读不到时默认深色（与 Windows 端一致）。
func platformPrefersDark() bool {
	// gsettings（GNOME）
	if p, err := exec.LookPath("gsettings"); err == nil {
		out, err := exec.Command(p, "get", "org.gnome.desktop.interface", "color-scheme").Output()
		if err == nil && strings.Contains(string(out), "dark") {
			return true
		}
		if out, err := exec.Command(p, "get", "org.gnome.desktop.interface", "gtk-theme").Output(); err == nil {
			if strings.Contains(strings.ToLower(string(out)), "dark") {
				return true
			}
		}
	}
	// macOS
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output(); err == nil {
			if strings.Contains(strings.ToLower(string(out)), "dark") {
				return true
			}
		}
	}
	for _, env := range []string{"GTK_THEME", "COLOR_SCHEME"} {
		if strings.Contains(strings.ToLower(os.Getenv(env)), "dark") {
			return true
		}
	}
	return true
}

// ---- 休眠抑制 ----

var (
	keepAwakeMu  sync.Mutex
	keepAwakeCmd *exec.Cmd
	keepAwakeOn  bool
)

// platformKeepAwake 阻止系统休眠（Linux 用 systemd-inhibit，macOS 用 caffeinate）。
// 返回调用前的状态（非 0 表示之前已启用）。无对应命令时空操作，返回 0。
func platformKeepAwake(on bool) uint32 {
	keepAwakeMu.Lock()
	defer keepAwakeMu.Unlock()
	prev := uint32(0)
	if keepAwakeOn {
		prev = 1
	}
	if on == keepAwakeOn {
		return prev
	}
	if !on {
		if keepAwakeCmd != nil && keepAwakeCmd.Process != nil {
			_ = keepAwakeCmd.Process.Kill()
			_ = keepAwakeCmd.Wait()
		}
		keepAwakeCmd, keepAwakeOn = nil, false
		return prev
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		if p, err := exec.LookPath("caffeinate"); err == nil {
			cmd = exec.Command(p, "-d", "-i")
		}
	default:
		if p, err := exec.LookPath("systemd-inhibit"); err == nil {
			cmd = exec.Command(p, "sleep", "infinity")
		} else if p, err := exec.LookPath("xdg-screensaver"); err == nil {
			cmd = exec.Command(p, "suspend", "0")
		}
	}
	if cmd != nil {
		if err := cmd.Start(); err == nil {
			keepAwakeCmd, keepAwakeOn = cmd, true
		}
	}
	return prev
}
