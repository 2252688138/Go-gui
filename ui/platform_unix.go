//go:build !windows && !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// 类 Unix 端（Linux / macOS / BSD / 移动端）的系统能力实现。
//
// 策略：优先调用系统自带命令（xdg-open / zenity / pbpaste / launchctl…），
// 命令不存在时安全降级（返回 false / 空串 / 能力位缺失），绝不 panic、绝不阻塞 UI。
// 这样不需要引入任何第三方依赖，也保持零 cgo。

// errNoTray 当前平台的托盘尚未实现。
var errNoTray = errors.New("ui: 当前平台尚未实现系统托盘（可用 HasCap(CapTray) 判断）")

// ---- 窗口 ----

// Current 返回当前活跃窗口（无则 nil）。
func Current() *App { return currentApp.Load() }

// NewWindow 打开新窗口（须在 UI 线程/回调中调用）。
func NewWindow(opts Options, view func() *Node) (*App, error) {
	return OpenWindow(opts, view)
}

// ---- 打开链接 / 文件 ----

// ShellOpen 用系统默认程序打开路径或网址。
func ShellOpen(path string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", path).Start()
	case "android":
		_ = exec.Command("am", "start", "-a", "android.intent.action.VIEW", "-d", path).Start()
	case "ios":
		// 沙盒内不能起进程，交由壳层处理（Capabilities 会标记缺失）。
	default:
		_ = exec.Command("xdg-open", path).Start()
	}
}

// PickFile 选单个文件（类 Unix 端同步实现，选完立即回调）。
func PickFile(title, filter string, onPick func(path string)) {
	if p, ok := OpenFileDialog(title, filter); ok && onPick != nil {
		onPick(p)
	}
}

// PickFilesAsync 选多个文件（类 Unix 端同步实现，选完立即回调）。
func PickFilesAsync(title, filter string, onPick func(paths []string)) {
	if ps, ok := PickFiles(title, filter); ok && onPick != nil {
		onPick(ps)
	}
}

// StartupQuery 读取启动参数：类 Unix 端没有 URL，恒返回空串（Web 端实现见 platform_wasm.go）。
func StartupQuery(key string) string { return "" }

// RunAsync 在后台 goroutine 执行 fn（类 Unix 端把阻塞工作放后台，
// 完成后用 App.Post 回 UI 线程）。浏览器端实现见 platform_wasm.go。
func RunAsync(fn func()) {
	if fn != nil {
		go fn()
	}
}

// ---- 文件对话框（调用系统对话框工具，缺失则返回 false）----

// OpenFileDialog 打开文件对话框。filter 形如 "图片|*.png;*.jpg|所有文件|*.*"。
func OpenFileDialog(title, filter string) (string, bool) {
	args := dialogArgs(title, filter, false, false)
	if len(args) == 0 {
		return "", false
	}
	out, err := runCapture(args[0], args[1:]...)
	if err != nil || strings.TrimSpace(out) == "" {
		return "", false
	}
	return strings.TrimSpace(out), true
}

// SaveFileDialog 保存文件对话框。
func SaveFileDialog(title, filter, defExt string) (string, bool) {
	args := dialogArgs(title, filter, true, false)
	if len(args) == 0 {
		return "", false
	}
	out, err := runCapture(args[0], args[1:]...)
	if err != nil || strings.TrimSpace(out) == "" {
		return "", false
	}
	p := strings.TrimSpace(out)
	if defExt != "" && !strings.HasSuffix(strings.ToLower(p), "."+strings.ToLower(strings.TrimPrefix(defExt, "."))) {
		p += "." + strings.TrimPrefix(defExt, ".")
	}
	return p, true
}

// PickFolderDialog 选择文件夹。
func PickFolderDialog(title string) (string, bool) {
	args := dialogArgs(title, "", false, true)
	if len(args) == 0 {
		return "", false
	}
	out, err := runCapture(args[0], args[1:]...)
	if err != nil || strings.TrimSpace(out) == "" {
		return "", false
	}
	return strings.TrimSpace(out), true
}

// PickFiles 打开多选文件对话框（本实现一次返回一个文件，保持 API 一致）。
func PickFiles(title, filter string) ([]string, bool) {
	p, ok := OpenFileDialog(title, filter)
	if !ok {
		return nil, false
	}
	return []string{p}, true
}

// dialogArgs 生成系统对话框命令（zenity / kdialog / osascript），都不存在时返回 nil。
func dialogArgs(title, filter string, save, folder bool) []string {
	if runtime.GOOS == "darwin" {
		if save {
			return []string{"osascript", "-e",
				fmt.Sprintf(`POSIX path of (choose file name with prompt "%s")`, title)}
		}
		if folder {
			return []string{"osascript", "-e",
				fmt.Sprintf(`POSIX path of (choose folder with prompt "%s")`, title)}
		}
		return []string{"osascript", "-e",
			fmt.Sprintf(`POSIX path of (choose file with prompt "%s")`, title)}
	}
	// Linux / BSD：zenity 优先，kdialog 兜底
	pattern := firstFilterPattern(filter)
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"zenity", "--file-selection"}
		if title != "" {
			args = append(args, "--title="+title)
		}
		if save {
			args = append(args, "--save", "--confirm-overwrite")
		}
		if folder {
			args = append(args, "--directory")
		}
		if pattern != "" {
			args = append(args, "--file-filter="+pattern)
		}
		return args
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		if save {
			return []string{"kdialog", "--getsavefilename", os.Getenv("HOME")}
		}
		if folder {
			return []string{"kdialog", "--getexistingdirectory", os.Getenv("HOME")}
		}
		return []string{"kdialog", "--getopenfilename", os.Getenv("HOME")}
	}
	return nil
}

// firstFilterPattern 从 "图片|*.png;*.jpg|所有文件|*.*" 里取第一组通配符。
func firstFilterPattern(filter string) string {
	parts := strings.Split(filter, "|")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// runCapture 执行命令并取回 stdout（对话框类命令都是阻塞的，由调用方在 UI 线程触发）。
func runCapture(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

// ---- 剪贴板（xclip / xsel / wl-paste / pbcopy / pbpaste）----

// ClipboardText 读取剪贴板文本（无文本返回空串）。
func ClipboardText() string {
	for _, cmd := range [][]string{
		{"pbpaste"},
		{"wl-paste", "--no-newline"},
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
	} {
		if _, err := exec.LookPath(cmd[0]); err != nil {
			continue
		}
		if out, err := runCapture(cmd[0], cmd[1:]...); err == nil {
			return out
		}
	}
	return ""
}

// ClipboardSetText 写入剪贴板文本。
func ClipboardSetText(s string) bool {
	for _, cmd := range [][]string{
		{"pbcopy"},
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
	} {
		if _, err := exec.LookPath(cmd[0]); err != nil {
			continue
		}
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Stdin = strings.NewReader(s)
		if err := c.Run(); err == nil {
			return true
		}
	}
	return false
}

// ClipboardSetFiles 把文件路径写入剪贴板（本实现退化为文本形式，保持 API 一致）。
func ClipboardSetFiles(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	return ClipboardSetText(strings.Join(paths, "\n"))
}

// ---- 电源 / 资源管理器 ----

// SetKeepAwake 阻止系统休眠（Linux 用 systemd-inhibit，macOS 用 caffeinate）。
// 返回调用前的标志位（0 = 之前未启用）。
func SetKeepAwake(on bool) uint32 {
	return platformKeepAwake(on)
}

// RevealInExplorer 在文件管理器中定位给定路径（打开父目录即视为成功）。
func RevealInExplorer(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	ShellOpen(path)
	return true
}

// MoveToRecycleBin 把文件移入回收站（使用 gio trash 命令，失败返回 false）。
func MoveToRecycleBin(path string) bool {
	if _, err := exec.LookPath("gio"); err == nil {
		if err := exec.Command("gio", "trash", path).Run(); err == nil {
			return true
		}
	}
	return false
}

// SystemDarkMode 系统是否处于深色模式（读取桌面环境的主题偏好）。
func SystemDarkMode() bool { return platformPrefersDark() }

// ---- 托盘（待实现，先给出可用但不可见的最小实现）----

// TrayItem 托盘菜单项（字段与 Windows 端一致，保证同一份菜单代码可编译）。
type TrayItem struct {
	ID        int
	Label     string
	Separator bool
	Checked   bool
	Disabled  bool
	// Submenu 非空时生成二级菜单。
	Submenu []TrayItem
}

// Tray 托盘图标（本平台占位：创建成功但不可见，SetTooltip/SetItems 安全空操作）。
type Tray struct {
	tooltip string
	items   []TrayItem
	onClick func()
	onMenu  func(id int)
}

// NewTray 创建托盘图标。本平台尚未接入（Linux 需 StatusNotifierItem、macOS 需 NSStatusBar），
// 返回的错误可被业务忽略：调用方应先查 HasCap(CapTray)。
func NewTray(tooltip string, onClick func(), items []TrayItem, onMenu func(id int)) (*Tray, error) {
	return &Tray{tooltip: tooltip, items: items, onClick: onClick, onMenu: onMenu}, errNoTray
}

// SetTooltip 更新托盘提示。
func (t *Tray) SetTooltip(s string) {
	if t != nil {
		t.tooltip = s
	}
}

// SetItems 更新右键菜单。
func (t *Tray) SetItems(items []TrayItem) {
	if t != nil {
		t.items = items
	}
}

// Notify 气泡通知（本平台暂不支持，空操作）。
func (t *Tray) Notify(title, msg string) {}

// Remove 移除托盘图标。
func (t *Tray) Remove() { t = nil }
