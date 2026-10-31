//go:build !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// web/server 极简静态服务器：用于本地预览 WebAssembly 版本。
//
// 为什么不用 `python -m http.server`：
//   - .wasm 必须返回 Content-Type: application/wasm，否则浏览器拒绝
//     WebAssembly.instantiateStreaming（只能退化到 arrayBuffer，慢且容易踩缓存）；
//   - 需要 -open 自动打开浏览器，一条命令就能预览。
//
// 用法（由 build.ps1 -Target web -Serve 调用）：
//
//	go run ./web/server -dir dist/web -addr 127.0.0.1:8088 -open
package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	dir := flag.String("dir", "dist/web", "静态文件目录")
	addr := flag.String("addr", "127.0.0.1:8088", "监听地址")
	open := flag.Bool("open", false, "启动后打开默认浏览器")
	flag.Parse()

	root, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("目录无效：%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "main.wasm")); err != nil {
		log.Fatalf("目录 %s 里没有 main.wasm（先执行 ./build.ps1 -Target web）", root)
	}

	// 关键：告诉 Go 的 MIME 表 .wasm 的正式类型
	_ = mime.AddExtensionType(".wasm", "application/wasm")
	_ = mime.AddExtensionType(".js", "text/javascript; charset=utf-8")

	h := &handler{root: root}
	srv := &http.Server{
		Addr:              *addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	url := "http://" + *addr + "/"
	fmt.Printf("[web] 目录 %s\n[web] 地址 %s\n", root, url)
	if *open {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openBrowser(url)
		}()
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("监听失败：%v", err)
	}
}

// handler 静态文件服务：禁缓存（改完代码刷新即可看到）+ 简单日志。
type handler struct {
	root string
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	full := filepath.Join(h.root, filepath.FromSlash(p))
	// 防目录穿越
	if rel, err := filepath.Rel(h.root, full); err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := os.Stat(full); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, full)
}

// openBrowser 用系统默认浏览器打开 URL。
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("无法自动打开浏览器，请手动访问 %s", url)
	}
}
