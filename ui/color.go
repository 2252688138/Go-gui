// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"gogui/internal/gpu"
)

// C 颜色构造：支持 "#rgb" "#rrggbb" "#rrggbbaa" "rgb(1,2,3)" 以及 uint32 0xRRGGBB。
func C(v any) gpu.Paint {
	switch x := v.(type) {
	case gpu.Paint:
		return x
	case uint32:
		return gpu.Hex(x)
	case int:
		return gpu.Hex(uint32(x))
	case string:
		return parseColor(x)
	}
	return gpu.Paint{R: 1, G: 0, B: 1, A: 1}
}

// colorCache 颜色字符串解析缓存。
//
// 主题 token 与组件写死的色值每帧会被解析几十上百次，旧实现每次都走
// strings.TrimSpace / Split / strconv，rgb() 与 3 位 hex 分支还会分配切片。
// 解析结果是字符串的纯函数，按原文缓存即可，不存在失效问题。
type colorMemo struct {
	sync.RWMutex
	m map[string]gpu.Paint
}

var colorCache = &colorMemo{m: make(map[string]gpu.Paint, 64)}

// colorCacheMax 缓存条数上限（超出整体丢弃，防止动态色值把缓存撑大）。
const colorCacheMax = 512

func parseColor(s string) gpu.Paint {
	colorCache.RLock()
	c, ok := colorCache.m[s]
	colorCache.RUnlock()
	if ok {
		return c
	}
	c = parseColorSlow(s)
	colorCache.Lock()
	if len(colorCache.m) >= colorCacheMax {
		colorCache.m = make(map[string]gpu.Paint, 64)
	}
	colorCache.m[s] = c
	colorCache.Unlock()
	return c
}

func parseColorSlow(s string) gpu.Paint {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" || s == "transparent" {
		return gpu.Paint{}
	}
	if strings.HasPrefix(s, "rgb(") || strings.HasPrefix(s, "rgba(") {
		body := s[strings.IndexByte(s, '(')+1:]
		body = strings.TrimSuffix(body, ")")
		parts := strings.Split(body, ",")
		var vals [4]float32
		vals[3] = 1
		for i := 0; i < len(parts) && i < 4; i++ {
			p := strings.TrimSpace(parts[i])
			if strings.HasSuffix(p, "%") {
				f, _ := strconv.ParseFloat(strings.TrimSuffix(p, "%"), 64)
				vals[i] = float32(f) / 100
			} else {
				f, _ := strconv.ParseFloat(p, 64)
				if i < 3 {
					vals[i] = float32(f) / 255
				} else {
					vals[i] = float32(f)
				}
			}
		}
		return gpu.Paint{R: vals[0], G: vals[1], B: vals[2], A: vals[3]}
	}
	if !strings.HasPrefix(s, "#") {
		return gpu.Paint{R: 1, G: 0, B: 1, A: 1}
	}
	h := s[1:]
	switch len(h) {
	case 3, 4:
		exp := make([]byte, 0, len(h)*2)
		for i := 0; i < len(h); i++ {
			exp = append(exp, h[i], h[i])
		}
		h = string(exp)
	}
	if len(h) != 6 && len(h) != 8 {
		return gpu.Paint{R: 1, G: 0, B: 1, A: 1}
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return gpu.Paint{R: 1, G: 0, B: 1, A: 1}
	}
	if len(h) == 6 {
		return gpu.RGBA(uint8(v>>16), uint8(v>>8), uint8(v), 255)
	}
	return gpu.RGBA(uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v))
}

// Blend 两色插值。
func Blend(a, b gpu.Paint, t float32) gpu.Paint {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return gpu.Paint{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}

func hexOf(p gpu.Paint) string {
	return fmt.Sprintf("#%02x%02x%02x", int(p.R*255), int(p.G*255), int(p.B*255))
}
