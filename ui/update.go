// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 自更新：检查版本 → 下载 → SHA-256 校验 → 替换并重启。
//
//	cfg := ui.UpdateConfig{Current: version.String(), CheckURL: "https://example.com/latest.json"}
//	go func() {
//	    info, err := ui.CheckUpdate(cfg)
//	    if err == nil && info != nil { ui.UpdateAvailable(info.Version) }
//	}()
//	// 根 Stack：ui.UpdateLayer()
//
// 安全约定：URL 只来自调用方配置，不拼接、不猜测；校验失败拒绝替换。

// UpdateConfig 更新配置。
type UpdateConfig struct {
	// Current 当前版本（"1.2.3"）。
	Current string
	// CheckURL 返回 JSON：{"version":"1.3.0","notes":"...","url":"https://...","sha256":"...","size":123}
	// 也支持纯文本版本号（此时下载地址用 DownloadURL）。
	CheckURL string
	// DownloadURL 纯文本检查模式下的下载地址。
	DownloadURL string
}

// UpdateInfo 可用更新。
type UpdateInfo struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

const updateTimeout = 30 * time.Second

// CheckUpdate 检查更新；无更新返回 (nil, nil)。
func CheckUpdate(cfg UpdateConfig) (*UpdateInfo, error) {
	if cfg.CheckURL == "" {
		return nil, errors.New("ui: 未配置检查更新地址")
	}
	client := &http.Client{Timeout: updateTimeout}
	resp, err := client.Get(cfg.CheckURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ui: 检查更新失败 HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	info := &UpdateInfo{}
	if json.Unmarshal(raw, info) != nil || info.Version == "" {
		info.Version = strings.TrimSpace(string(raw))
		info.URL = cfg.DownloadURL
	}
	if info.Version == "" {
		return nil, errors.New("ui: 更新信息格式不正确")
	}
	if CompareVersion(info.Version, cfg.Current) <= 0 {
		return nil, nil
	}
	if info.URL == "" {
		info.URL = cfg.DownloadURL
	}
	return info, nil
}

// CompareVersion 语义化版本比较：a>b 返回 1，a<b 返回 -1，相等返回 0。
func CompareVersion(a, b string) int {
	pa := versionParts(a)
	pb := versionParts(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var va, vb int
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}
		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}
	return 0
}

func versionParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	segs := strings.Split(v, ".")
	out := make([]int, 0, len(segs))
	for _, s := range segs {
		n := 0
		fmt.Sscanf(s, "%d", &n)
		out = append(out, n)
	}
	return out
}

// DownloadUpdate 下载更新到 dst，onProgress 可为 nil。
func DownloadUpdate(info *UpdateInfo, dst string, onProgress func(done, total int64)) error {
	if info == nil || info.URL == "" {
		return errors.New("ui: 缺少下载地址")
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(info.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ui: 下载失败 HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	total := info.Size
	if total <= 0 {
		total = resp.ContentLength
	}
	var done int64
	buf := make([]byte, 64*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if onProgress != nil {
				onProgress(done, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if info.SHA256 != "" {
		return VerifySHA256(dst, info.SHA256)
	}
	return nil
}

// VerifySHA256 校验文件摘要（空 want 直接通过）。
func VerifySHA256(path, want string) error {
	if want == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("ui: 校验失败（期望 %s… 实际 %s…）", want[:minIntN(12, len(want))], got[:12])
	}
	return nil
}

// ApplyUpdate 用新文件替换当前可执行文件；Windows 上通过重命名 + 重启脚本完成。
// 返回错误时不改动现有文件。
func ApplyUpdate(newExe string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if ok, err := applyUpdatePlatform(self, newExe); ok {
		return err
	}
	// 通用回退：直接覆盖（仅当目标未被占用时可用）
	return os.Rename(newExe, self)
}

// UpdatePanel 更新面板（检查 / 下载进度 / 安装按钮）。
type updateState struct {
	checking bool
	info     *UpdateInfo
	err      string
	progress float32
}

var upd = &updateState{}

// UpdateCheck 后台检查更新（结果下一帧生效）。
func UpdateCheck(cfg UpdateConfig) {
	upd.checking = true
	upd.err = ""
	RunAsync(func() {
		info, err := CheckUpdate(cfg)
		postToUI(func() {
			upd.checking = false
			upd.info = info
			if err != nil {
				upd.err = err.Error()
			}
			RequestRedraw()
		})
	})
}

// UpdatePanelView 渲染更新面板。
func UpdatePanelView(cfg UpdateConfig) *Node {
	col := Column(Text("软件更新").St(St().Text(13.5, 600, ColText)))
	col.Add(Text("当前版本 " + cfg.Current).St(St().Text(11.5, 400, ColTextMuted)))
	switch {
	case upd.checking:
		col.Add(Row(Spinner(16, ColPrimary),
			Text("正在检查更新…").St(St().Text(12, 400, ColTextDim))).
			St(St().GapOf(8)).AlignTo(AlignCenter))
	case upd.err != "":
		col.Add(Banner("error", "检查失败", upd.err, nil))
	case upd.info == nil:
		col.Add(Text("已是最新版本").St(St().Text(12, 400, ColTextMuted)))
	default:
		col.Add(Text("发现新版本 " + upd.info.Version).St(St().Text(12.5, 600, ColPrimary)))
		if upd.info.Notes != "" {
			col.Add(Text(upd.info.Notes).St(St().Text(11.5, 400, ColTextDim).WrapOn()))
		}
		if upd.progress > 0 {
			col.Add(Progress(upd.progress, ColPrimary).St(St().Wp(100)))
		}
		col.Add(Row(
			Btn("下载并安装", BtnPrimary, func() {
				RunAsync(func() {
					dir, _ := os.UserCacheDir()
					if dir == "" {
						dir = os.TempDir()
					}
					dst := filepath.Join(dir, "gogui-update", "app-update.exe")
					err := DownloadUpdate(upd.info, dst, func(done, total int64) {
						p := float32(0)
						if total > 0 {
							p = float32(done) / float32(total)
						}
						postToUI(func() {
							upd.progress = p
							RequestRedraw()
						})
					})
					postToUI(func() {
						if err != nil {
							upd.err = err.Error()
						} else if err := ApplyUpdate(dst); err != nil {
							upd.err = err.Error()
						}
						RequestRedraw()
					})
				})
			}),
			Btn("稍后", BtnGhost, func() { upd.info = nil; RequestRedraw() }),
		).St(St().GapOf(8)))
	}
	col.Add(Btn("检查更新", BtnSecondary, func() { UpdateCheck(cfg) }))
	return col.St(St().GapOf(10).Padding(16).W(360).R(12).BgC(ColPanel).
		Stroke(1, ColStroke))
}

func minIntN(a, b int) int {
	if a < b {
		return a
	}
	return b
}
