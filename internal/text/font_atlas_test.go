// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

import (
	"math"
	"testing"
)

// TestAtlasDirtyRegionCoversGlyph 新字形只上报覆盖它的脏矩形（渲染器据此做局部上传）。
func TestAtlasDirtyRegionCoversGlyph(t *testing.T) {
	f := NewFont("Microsoft YaHei UI", 16, 400, 1)
	if f.TakeReset() {
		// 新建字体未重建过图集，允许返回 false；这里只关心脏矩形
	}

	_ = f.GlyphFor('A')
	x0, y0, x1, y1, ok := f.TakeDirty()
	if !ok {
		t.Fatal("光栅化字形后应产生脏矩形")
	}
	dw, dh := x1-x0, y1-y0
	if dw <= 0 || dh <= 0 {
		t.Fatalf("脏矩形尺寸非法: %dx%d", dw, dh)
	}
	if dw > maxAtlasDim/4 || dh > maxAtlasDim/4 {
		t.Fatalf("单个字形不应产生整个图集级别的脏区: %dx%d", dw, dh)
	}
	if _, _, _, _, ok2 := f.TakeDirty(); ok2 {
		t.Fatal("取走脏矩形后应清空，避免重复上传")
	}
}

// atlasNonZeroBox 返回图集中已写入像素（alpha != 0）的包围盒（y 为纹理行号）。
func atlasNonZeroBox(f *Font) (x0, y0, x1, y1 int, ok bool) {
	dim := f.AtlasDimNow()
	x0, y0, x1, y1 = dim, dim, -1, -1
	for row := 0; row < dim; row++ {
		base := row * dim * 4
		for x := 0; x < dim; x++ {
			if f.Atlas[base+x*4+3] == 0 {
				continue
			}
			if x < x0 {
				x0 = x
			}
			if x > x1 {
				x1 = x
			}
			if row < y0 {
				y0 = row
			}
			if row > y1 {
				y1 = row
			}
		}
	}
	if x1 < 0 {
		return 0, 0, 0, 0, false
	}
	return x0, y0, x1 + 1, y1 + 1, true
}

// TestAtlasDirtyRegionMerges 脏矩形必须覆盖本次写入的全部像素。
// 这是局部上传正确性的核心不变量：盖不到就会出现缺字/串字。
func TestAtlasDirtyRegionMerges(t *testing.T) {
	f := NewFont("Microsoft YaHei UI", 16, 400, 1)
	_ = f.GlyphFor('中')
	_ = f.GlyphFor('国')

	x0, y0, x1, y1, ok := f.TakeDirty()
	if !ok {
		t.Fatal("光栅化字形后应产生脏矩形")
	}
	bx0, by0, bx1, by1, has := atlasNonZeroBox(f)
	if !has {
		t.Fatal("图集里应有字形像素")
	}
	if bx0 < x0 || bx1 > x1 || by0 < y0 || by1 > y1 {
		t.Fatalf("脏矩形 [%d,%d,%d,%d] 未覆盖实际写入区域 [%d,%d,%d,%d]", x0, y0, x1, y1, bx0, by0, bx1, by1)
	}
	if y1-y0 > maxAtlasDim/2 {
		t.Fatalf("两个字形不应产生过大脏区: %d 行", y1-y0)
	}
}

// TestGlyphUVIsPixelExact 字形 quad（已对齐整数物理像素）的插值采样必须与图集纹素 1:1。
// 这是文字锐度的基本不变量：只要有一个 fragment 落在纹素之间，双线性就会把笔画重采样糊掉。
func TestGlyphUVIsPixelExact(t *testing.T) {
	f := NewFont("Microsoft YaHei UI", 14, 400, 1.25)
	for _, r := range "国Am鼎" {
		g := f.GlyphFor(r)
		if !g.Valid || g.Color || g.W < 1 || g.H < 1 {
			continue
		}
		dim := float64(f.AtlasDimNow())

		// UV 跨度必须等于位图尺寸（纹素），不能内缩
		if span := (float64(g.U1) - float64(g.U0)) * dim; math.Abs(span-float64(g.W)) > 1e-3 {
			t.Fatalf("%q 横向 UV 跨度 %.4f 纹素 ≠ 位图宽 %.0f", r, span, g.W)
		}
		if span := (float64(g.V1) - float64(g.V0)) * dim; math.Abs(span-float64(g.H)) > 1e-3 {
			t.Fatalf("%q 纵向 UV 跨度 %.4f 纹素 ≠ 位图高 %.0f", r, span, g.H)
		}

		// 四条边都必须落在纹素边界（整数）上
		left := float64(g.U0) * dim
		top := float64(g.V1) * dim
		for name, v := range map[string]float64{
			"U0": left,
			"U1": float64(g.U1) * dim,
			"V0": float64(g.V0) * dim,
			"V1": top,
		} {
			if math.Abs(v-math.Round(v)) > 1e-3 {
				t.Fatalf("%q %s 不在纹素边界上: %.5f", r, name, v)
			}
		}

		// 逐像素：fragment 中心（k+0.5）必须正好命中纹素中心（ox+k+0.5）
		for k := 0; k < int(g.W); k++ {
			u := float64(g.U0) + (float64(k)+0.5)/float64(g.W)*float64(g.U1-g.U0)
			if want := left + float64(k) + 0.5; math.Abs(u*dim-want) > 1e-3 {
				t.Fatalf("%q 第 %d 列采样偏离纹素中心（%.4f ≠ %.4f）", r, k, u*dim, want)
			}
		}
		for j := 0; j < int(g.H); j++ {
			v := float64(g.V1) + (float64(j)+0.5)/float64(g.H)*float64(g.V0-g.V1)
			if want := top - float64(j) - 0.5; math.Abs(v*dim-want) > 1e-3 {
				t.Fatalf("%q 第 %d 行采样偏离纹素中心（%.4f ≠ %.4f）", r, j, v*dim, want)
			}
		}
	}
}

func TestAtlasResetOnScale(t *testing.T) {
	f := NewFont("Microsoft YaHei UI", 16, 400, 1)
	f.TakeReset()
	f.SetScale(1.5)
	if !f.TakeReset() {
		t.Fatal("SetScale 后应标记图集重建")
	}
	if f.TakeReset() {
		t.Fatal("重建标记应只消费一次")
	}
}

// TestGlyphCacheReuse 同一字形只光栅化一次（缓存命中不产生脏区）。
func TestGlyphCacheReuse(t *testing.T) {
	f := NewFont("Microsoft YaHei UI", 16, 400, 1)
	g1 := f.GlyphFor('测')
	f.TakeDirty()
	g2 := f.GlyphFor('测')
	if _, _, _, _, ok := f.TakeDirty(); ok {
		t.Fatal("缓存命中的字形不应再次上传")
	}
	if g1 != g2 {
		t.Fatal("同一字形应返回同一份缓存对象")
	}
}
