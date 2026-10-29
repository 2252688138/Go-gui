//go:build !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Markdown 远程图片：下载到本地缓存目录后交给普通图片管线（异步解码 + 纹理 LRU）。
//
// 只做「显示」用途的最小实现：GET 一次、限时 12s、限 12MB，
// 成功/失败都写回缓存表，失败后不反复重试（避免滚动时反复打网络）。

const (
	mdRemoteMaxBytes = 12 << 20
	mdRemoteTimeout  = 12 * time.Second
)

type mdRemoteEntry struct {
	path  string
	err   error
	done  bool
	start chan struct{}
}

var errRemoteTooLarge = errors.New("远程图片超过 12MB 限制")

type mdRemoteTable struct {
	mu sync.Mutex
	m  map[string]*mdRemoteEntry
}

var mdRemotes = &mdRemoteTable{m: map[string]*mdRemoteEntry{}}

// mdStatusError HTTP 状态码错误。
func mdStatusError(code int) error { return fmt.Errorf("HTTP %d", code) }

// mdRemoteImagePath 返回远程图片的本地缓存路径；未就绪时 ok=false 并触发下载。
func mdRemoteImagePath(url string) (string, bool) {
	mdRemotes.mu.Lock()
	if e, ok := mdRemotes.m[url]; ok {
		mdRemotes.mu.Unlock()
		if !e.done {
			return "", false
		}
		if e.err != nil || e.path == "" {
			return "", false
		}
		return e.path, true
	}
	e := &mdRemoteEntry{start: make(chan struct{})}
	mdRemotes.m[url] = e
	mdRemotes.mu.Unlock()

	go mdFetchRemote(url, e)
	return "", false
}

// mdFetchRemote 后台下载（结果写回条目并请求重绘）。
func mdFetchRemote(url string, e *mdRemoteEntry) {
	path, err := mdDownload(url)
	mdRemotes.mu.Lock()
	e.path, e.err, e.done = path, err, true
	mdRemotes.mu.Unlock()
	close(e.start)
	// 任意窗口重绘：图片就绪后由 imageTextures 走正常异步解码
	RequestRedraw()
}

func mdDownload(url string) (string, error) {
	client := &http.Client{Timeout: mdRemoteTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoGUI-Markdown/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", mdStatusError(resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, mdRemoteMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > mdRemoteMaxBytes {
		return "", errRemoteTooLarge
	}
	dir := filepath.Join(os.TempDir(), "gogui-md-img")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(url))
	name := hex.EncodeToString(sum[:8]) + mdImageExt(url, resp.Header.Get("Content-Type"))
	dst := filepath.Join(dir, name)
	if st, err := os.Stat(dst); err == nil && st.Size() == int64(len(data)) {
		return dst, nil // 已有同尺寸缓存
	}
	tmp := dst + ".part"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dst); err != nil {
		return "", err
	}
	return dst, nil
}

// mdImageExt 依据 URL 后缀与 Content-Type 决定缓存文件扩展名。
func mdImageExt(url, ct string) string {
	l := strings.ToLower(url)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".bmp"} {
		if strings.Contains(l, ext) {
			if ext == ".jpeg" {
				return ".jpg"
			}
			return ext
		}
	}
	switch {
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "gif"):
		return ".gif"
	case strings.Contains(ct, "bmp"):
		return ".bmp"
	}
	return ".jpg"
}

// mdRemoteError 远程图片失败原因（占位卡展示用）。
func mdRemoteError(url string) string {
	mdRemotes.mu.Lock()
	defer mdRemotes.mu.Unlock()
	e, ok := mdRemotes.m[url]
	if !ok {
		return "未开始"
	}
	if !e.done {
		return "下载中…"
	}
	if e.err != nil {
		return e.err.Error()
	}
	return ""
}

// mdRemotePending 是否仍有下载进行中（用于占位卡文案）。
func mdRemotePending(url string) bool {
	mdRemotes.mu.Lock()
	defer mdRemotes.mu.Unlock()
	e, ok := mdRemotes.m[url]
	return ok && !e.done
}
