// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"gogui/internal/gpu"
)

// errBadDataURL data URL 格式不合法。
var errBadDataURL = errors.New("ui: 非法的 data URL")

// imageReader 打开图片数据源：支持本地路径与 data: URL（Web 端选文件后
// 图片以 data URL 形式返回，桌面端永远是路径）。blob: 需要异步 fetch，不支持。
func imageReader(path string) (io.ReadCloser, error) {
	if !strings.HasPrefix(path, "data:") {
		return os.Open(path)
	}
	i := strings.Index(path, ",")
	if i < 0 {
		return nil, errBadDataURL
	}
	meta, body := path[:i], path[i+1:]
	if strings.Contains(meta, ";base64") {
		data, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	s, err := url.PathUnescape(body)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader(s)), nil
}

// imageMeta 图片尺寸缓存（布局阶段只需尺寸，不加载像素）。
type imageMeta struct {
	w, h int
	err  error
}

var (
	imageMetaMu sync.Mutex
	imageMetas  = map[string]imageMeta{}
)

// ImageSize 返回图片像素尺寸（带缓存，失败返回 0,0）。
func ImageSize(path string) (int, int) {
	imageMetaMu.Lock()
	defer imageMetaMu.Unlock()
	if m, ok := imageMetas[path]; ok {
		return m.w, m.h
	}
	m := imageMeta{}
	f, err := imageReader(path)
	if err != nil {
		m.err = err
	} else {
		cfg, _, derr := image.DecodeConfig(f)
		f.Close()
		if derr != nil {
			m.err = derr
		} else {
			m.w, m.h = cfg.Width, cfg.Height
		}
	}
	imageMetas[path] = m
	return m.w, m.h
}

// DecodeImage 解码图片为 NRGBA（非预乘，可直接上传 GL）。
func DecodeImage(path string) (*image.NRGBA, error) {
	f, err := imageReader(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	if n, ok := src.(*image.NRGBA); ok {
		return n, nil
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst, nil
}

// texUpload 后台解码结果（由 UI 线程完成 GL 上传，纹理只能在该线程创建）。
type texUpload struct {
	img *image.NRGBA
	err error
}

// texEntry 一张图片的纹理状态。
type texEntry struct {
	tex     *gpu.Texture
	bytes   int64
	stamp   int64 // 最近一次被绘制的帧号（LRU 回收用）
	err     error
	failed  bool
	started atomic.Bool
	ready   atomic.Pointer[texUpload]
}

// imageTextures 每个窗口一份纹理缓存（GL 资源随窗口销毁）。
type imageTextures struct {
	mu sync.Mutex
	m  map[string]*texEntry
	// bytes 当前缓存纹理的总字节数（用于按预算回收，避免长列表图片页显存只增不减）
	bytes int64
	// waiting 限流而被推迟的解码请求数（>0 时需继续刷帧重试）
	waiting atomic.Int32
}

// imageTexBudget 单窗口图片纹理预算：超出后按「最近未使用」回收纹理。
// 旧实现只增不减，翻完几百张图的相册/长列表后显存与堆都不会回落。
// 96MB 的预算对「一屏几张缩略图」的常规界面而言过于宽松，
// 现在收紧到 32MB：缩略图约可同时驻留 20~40 张，超过后按 LRU 回收、
// 再次进入视口时自动重新解码（数据层有序，观感无差异）。
const imageTexBudget int64 = 32 << 20

func newImageTextures() *imageTextures { return &imageTextures{m: map[string]*texEntry{}} }

// count 已缓存的图片条目数。
func (t *imageTextures) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.m)
}

// maxImageLoads 同时解码的图片数量上限（一下子滚到很多图片时避免 CPU 峰值）。
const maxImageLoads = 4

var imageLoads atomic.Int32

// imagePending 仍在后台解码/待上传的图片数量（>0 时需继续刷帧推进）。
func imagePending() int { return int(imageLoads.Load()) }

// get 取纹理：首次访问返回 nil（本帧不画），解码完成后请求重绘再上传。
// 以前是在绘制路径里同步解码 + 上传，遇到大图或一次出现多张图时会让 UI 线程
// 停顿几十上百毫秒，表现出来就是滚动/翻页时的卡顿。
func (t *imageTextures) get(r *gpu.Renderer, a *App, path string) *gpu.Texture {
	e := t.entry(path)
	e.stamp = int64(a.frameCount)
	if e.tex != nil {
		return e.tex
	}
	if up := e.ready.Swap(nil); up != nil {
		if up.err != nil || up.img == nil {
			e.err, e.failed = up.err, true
			return nil
		}
		b := up.img.Bounds()
		e.tex = r.NewTextureRGBA(b.Dx(), b.Dy(), up.img.Pix)
		if e.tex != nil {
			e.bytes = int64(b.Dx()) * int64(b.Dy()) * 4
			t.mu.Lock()
			t.bytes += e.bytes
			t.mu.Unlock()
			t.evict(r, e.stamp)
		}
		return e.tex
	}
	if e.failed {
		return nil
	}
	t.start(e, a, path)
	return nil
}

// evict 显存超预算时按最近未使用回收纹理。
// keep 为本帧帧号：本帧正在绘制的纹理即使很旧也不能删，
// 否则已经进批次但尚未提交的顶点会引用已删除的纹理。
func (t *imageTextures) evict(r *gpu.Renderer, keep int64) {
	t.mu.Lock()
	if t.bytes <= imageTexBudget {
		t.mu.Unlock()
		return
	}
	var victimKey string
	var victim *texEntry
	for k, e := range t.m {
		if e.tex == nil || e.stamp == keep {
			continue
		}
		if victim == nil || e.stamp < victim.stamp {
			victimKey, victim = k, e
		}
	}
	if victim == nil {
		t.mu.Unlock()
		return
	}
	t.bytes -= victim.bytes
	delete(t.m, victimKey)
	t.mu.Unlock()

	r.DeleteTexture(victim.tex)
	victim.tex = nil
	victim.bytes = 0
	victim.started.Store(false) // 再次进入视口时重新解码
}

// beginFrame 每帧开始前重置限流标记。
// 旧实现里 waiting 一旦被置 1 就再也没人清 0（图片滚出视口后不会再有 start 调用），
// frame() 看到 waiting>0 会持续请求重绘 —— 表现为滚过一批图片后窗口永久 100% CPU。
func (t *imageTextures) beginFrame() { t.waiting.Store(0) }

// releaseAll 释放全部图片纹理（空闲 / 最小化时调用），保留条目以便重新解码。
// 条目里的 started 要复位，否则下次回到该页面时不会重新拉起解码。
func (t *imageTextures) releaseAll(r *gpu.Renderer) {
	if t == nil {
		return
	}
	t.mu.Lock()
	victims := make([]*texEntry, 0, len(t.m))
	for _, e := range t.m {
		if e.tex != nil {
			victims = append(victims, e)
		}
	}
	t.bytes = 0
	t.mu.Unlock()
	for _, e := range victims {
		if r != nil {
			r.DeleteTexture(e.tex)
		}
		e.tex, e.bytes = nil, 0
		e.started.Store(false)
	}
}

func (t *imageTextures) entry(path string) *texEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	e, ok := t.m[path]
	if !ok {
		e = &texEntry{}
		t.m[path] = e
	}
	return e
}

// start 启动一次后台解码（已有结果、失败或进行中则跳过）。
func (t *imageTextures) start(e *texEntry, a *App, path string) {
	if !e.started.CompareAndSwap(false, true) {
		return
	}
	if imageLoads.Add(1) > maxImageLoads {
		imageLoads.Add(-1)
		e.started.Store(false) // 稍后的帧再试
		t.waiting.Store(1)
		return
	}
	t.waiting.Store(0)
	go func() {
		defer recoverReport("image-decode")
		defer imageLoads.Add(-1)
		img, err := DecodeImage(path)
		e.ready.Store(&texUpload{img: img, err: err})
		RequestRedrawIn(a)
	}()
}
