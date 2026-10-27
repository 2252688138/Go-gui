package ui

// Markdown 组件的共享状态（页签 / 折叠 / 复制提示 / 图片预览）与解析缓存。
// 单独成文件，避免 markdown.go 过长，也便于拷贝组件时只带必要文件。

// mdStateT Markdown 交互状态。
type mdStateT struct {
	tabs    map[string]int
	details map[string]bool
	folded  map[string]bool
	toast   string
	toastAt float32
	viewer  struct {
		Open bool
		Src  string
		Alt  string
	}
}

// mdState 全局交互状态（UI 线程独占，无需加锁）。
var mdState = mdStateT{
	tabs:    map[string]int{},
	details: map[string]bool{},
	folded:  map[string]bool{},
}

// mdParseCacheT 解析结果缓存（同一内容不重复解析；流式输出收益明显）。
type mdParseCacheT struct {
	m   map[string]*mdDoc
	max int
}

var mdParseCache = mdParseCacheT{
	m:   map[string]*mdDoc{},
	max: 24,
}
