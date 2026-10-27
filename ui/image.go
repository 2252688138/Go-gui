package ui

import (
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sync"
	"sync/atomic"

	"gogui/internal/gpu"
)

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
	f, err := os.Open(path)
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
	f, err := os.Open(path)
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
	err     error
	failed  bool
	started atomic.Bool
	ready   atomic.Pointer[texUpload]
}

// imageTextures 每个窗口一份纹理缓存（GL 资源随窗口销毁）。
type imageTextures struct {
	mu sync.Mutex
	m  map[string]*texEntry
	// waiting 限流而被推迟的解码请求数（>0 时需继续刷帧重试）
	waiting atomic.Int32
}

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
		return e.tex
	}
	if e.failed {
		return nil
	}
	t.start(e, a, path)
	return nil
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
		defer imageLoads.Add(-1)
		img, err := DecodeImage(path)
		e.ready.Store(&texUpload{img: img, err: err})
		RequestRedrawIn(a)
	}()
}
