package ui

// Markdown 组件的共享状态（页签 / 折叠 / 复制提示 / 图片预览）与解析缓存。
// 单独成文件，避免 markdown.go 过长，也便于拷贝组件时只带必要文件。

// mdStateT Markdown 交互状态。
type mdStateT struct {
	tabs    map[string]int
	details map[string]bool
	folded  map[string]bool
	// checks 任务列表勾选覆盖（nil = 用源码里的默认值）
	checks map[string]bool
	// codeX 代码块横向偏移（超宽且未折行时用拖动查看）
	codeX map[string]float32
	// anchors 各级标题锚点节点（目录点击滚动用；渲染时重建）
	anchors map[string]*Node
	toast   string
	toastAt float32
	viewer  struct {
		Open bool
		Src  string
		Alt  string
		Zoom float32
	}
}

// mdState 全局交互状态（UI 线程独占，无需加锁）。
var mdState = mdStateT{
	tabs:    map[string]int{},
	details: map[string]bool{},
	folded:  map[string]bool{},
	checks:  map[string]bool{},
	codeX:   map[string]float32{},
	anchors: map[string]*Node{},
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
