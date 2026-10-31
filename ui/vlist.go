// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"reflect"
	"sync"
)

// 虚拟列表：只创建视口内的行，十万行也能流畅滚动。
//
//	ui.VList("files", len(files), 44, 520, func(i int) *ui.Node {
//	    return ui.FileRow(files[i], nil)
//	})
//
// 原理：读取上一帧的滚动偏移（ScrollOffset），算出可见区间（含 overscan 行），
// 用「顶部占位 + 可见行 + 底部占位」保持总高度准确，滚动条/滚轮/平滑滚动沿用
// 现有 Scroll 实现。行高固定时用 VList，行高不等时用 VListVar。

// VListOptions 虚拟列表参数（VListOpt 使用）。
type VListOptions struct {
	// RowHeight 行高（逻辑像素）。
	RowHeight float32
	// Height 视口高度。
	Height float32
	// Overscan 视口外额外渲染的行数（默认 4，减少快速滚动白边）。
	Overscan int
	// Gap 行间距。
	Gap float32
	// Empty 列表为空时显示的内容（默认 EmptyState）。
	Empty *Node
}

// VList 固定行高虚拟列表。
func VList(key string, count int, rowH, height float32, build func(i int) *Node) *Node {
	return VListOpt(key, count, func(i int) *Node {
		if build == nil {
			return nil
		}
		return build(i)
	}, VListOptions{RowHeight: rowH, Height: height})
}

// VListOpt 带完整参数的虚拟列表。
func VListOpt(key string, count int, build func(i int) *Node, opts VListOptions) *Node {
	if opts.Height <= 0 {
		opts.Height = 360
	}
	if opts.RowHeight <= 0 {
		opts.RowHeight = 44
	}
	if opts.Overscan <= 0 {
		opts.Overscan = 4
	}
	if count <= 0 {
		var empty *Node
		if opts.Empty != nil {
			empty = opts.Empty
		} else {
			empty = EmptyState("", "暂无数据", "这里还没有内容", "", nil)
		}
		return Scroll(empty).K(key).St(St().H(opts.Height))
	}
	step := opts.RowHeight + opts.Gap
	off := ScrollOffset(key)
	if off < 0 {
		off = 0
	}
	first := int(off/step) - opts.Overscan
	if first < 0 {
		first = 0
	}
	last := int((off+opts.Height)/step) + opts.Overscan + 1
	if last > count {
		last = count
	}
	if last < first {
		last = first
	}
	items := make([]*Node, 0, last-first+2)
	if first > 0 {
		items = append(items, Box().St(St().H(float32(first)*step)))
	}
	for i := first; i < last; i++ {
		row := build(i)
		if row == nil {
			continue
		}
		if opts.Gap > 0 && i < last-1 {
			row.St(St().MarOnly(0, 0, opts.Gap, 0))
		}
		items = append(items, row)
	}
	if rest := count - last; rest > 0 {
		items = append(items, Box().St(St().H(float32(rest)*step)))
	}
	return Scroll(items...).K(key).St(St().H(opts.Height))
}

// vlistPrefix 变行高列表的前缀和（带缓存）。
//
// 业务通常每帧传入同一个 heights 切片，而前缀和是 O(n) 的：聊天上万条时
// 每帧重算会直接拉高整帧耗时。这里用「底层指针 + 长度」判断是否同一份数据，
// 相同则复用上次结果。
type vlistPrefixEntry struct {
	ptr    uintptr
	n      int
	prefix []float32
}

var (
	vlistPrefixMu    sync.Mutex
	vlistPrefixCache = map[string]vlistPrefixEntry{}
)

func vlistPrefix(key string, heights []float32) []float32 {
	ptr := reflect.ValueOf(heights).Pointer()
	vlistPrefixMu.Lock()
	ent, ok := vlistPrefixCache[key]
	vlistPrefixMu.Unlock()
	if ok && ent.ptr == ptr && ent.n == len(heights) {
		return ent.prefix
	}
	prefix := make([]float32, len(heights)+1)
	for i, h := range heights {
		if h <= 0 {
			h = 24
		}
		prefix[i+1] = prefix[i] + h
	}
	vlistPrefixMu.Lock()
	vlistPrefixCache[key] = vlistPrefixEntry{ptr: ptr, n: len(heights), prefix: prefix}
	vlistPrefixMu.Unlock()
	return prefix
}

// VListVar 变行高虚拟列表（heights 为每行高度，内部做前缀和）。
// 适合聊天记录、折叠列表；前缀和按 key 缓存，超大列表也只需首帧计算一次。
func VListVar(key string, heights []float32, height float32, build func(i int) *Node) *Node {
	if height <= 0 {
		height = 360
	}
	n := len(heights)
	if n == 0 {
		return Scroll(EmptyState("", "暂无数据", "这里还没有内容", "", nil)).K(key).St(St().H(height))
	}
	prefix := vlistPrefix(key, heights)
	total := prefix[n]
	off := ScrollOffset(key)
	const overscan = 3
	// 二分查找可见起点
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		if prefix[mid+1] < off {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	first := lo - overscan
	if first < 0 {
		first = 0
	}
	endY := off + height
	last := first
	for last < n && prefix[last] < endY {
		last++
	}
	last += overscan
	if last > n {
		last = n
	}
	items := make([]*Node, 0, last-first+2)
	if first > 0 {
		items = append(items, Box().St(St().H(prefix[first])))
	}
	for i := first; i < last; i++ {
		row := build(i)
		if row == nil {
			continue
		}
		items = append(items, row.St(St().H(prefix[i+1]-prefix[i])))
	}
	if rest := total - prefix[last]; rest > 0 {
		items = append(items, Box().St(St().H(rest)))
	}
	return Scroll(items...).K(key).St(St().H(height))
}

// VListVisible 返回给定偏移下应渲染的行区间 [first, last)（供自定义虚拟化复用 / 测试）。
func VListVisible(count int, off, viewH, rowH float32, overscan int) (int, int) {
	if rowH <= 0 {
		rowH = 1
	}
	if overscan < 0 {
		overscan = 0
	}
	first := int(off/rowH) - overscan
	if first < 0 {
		first = 0
	}
	last := int((off+viewH)/rowH) + overscan + 1
	if last > count {
		last = count
	}
	if last < first {
		last = first
	}
	return first, last
}
