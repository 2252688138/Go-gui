package ui

import (
	"regexp"
	"strconv"
	"strings"
)

// Markdown 解析层：把源码切成「块（block）+ 行内片段（inline run）」。
//
// 与 Web 版 Markdown.vue 的渲染思路对齐：
//   1. 先按扩展块（[tabs]/[details]/[card]/[gallery]/[progress]/[badge]/[video]）
//      与标准块（标题/段落/列表/引用/代码/表格/分隔线/图片）切块；
//   2. 块内文本再解析行内样式（加粗/斜体/删除线/行内代码/链接/高亮/kbd）；
//   3. 渲染层只负责把块与行内片段映射为 ui 节点。
// 解析结果按源码字符串做 LRU 缓存，流式输出时同一内容不重复解析。

type mdKind int

const (
	mdParagraph mdKind = iota
	mdHeading
	mdCode
	mdTable
	mdList
	mdQuote
	mdHR
	mdImage
	mdTabs
	mdDetails
	mdCard
	mdBadge
	mdProgress
	mdGallery
	mdMedia
	mdTOC
	// 新增：公式块 / Mermaid / 定义列表 / 脚注区
	mdMath
	mdMermaid
	mdDefList
	mdFootnotes
)

// mdInline 一段行内文本及其样式。
type mdInline struct {
	Text   string
	Bold   bool
	Italic bool
	Strike bool
	Code   bool
	Mark   bool
	Kbd    bool
	Link   bool
	Href   string
	// Br 硬换行（Markdown 行尾两个空格 / \ / <br>）。
	Br bool
	// Math 行内公式源码（$...$），非空时忽略 Text。
	Math string
	// Sup / Sub 上标与下标（^x^ / ~x~）。
	Sup string
	Sub string
	// FnRef 脚注引用 id（[^id]）。
	FnRef string
}

// mdCell 表格单元格。
type mdCell struct {
	Align Align
	Inl   []mdInline
}

// mdTable 表格。
type mdTableBlock struct {
	Header []mdCell
	Rows   [][]mdCell
}

// mdItem 列表项。
type mdItem struct {
	Inl []mdInline
	// Task -1 普通项；0 未勾选；1 已勾选
	Task int
	Body []*mdBlock
}

// mdTab 页签。
type mdTab struct {
	Label  string
	Blocks []*mdBlock
}

// mdCard 卡片块。
type mdCardBlock struct {
	Title, Subtitle, Image, URL string
	Blocks                      []*mdBlock
}

// mdBadge 徽章块。
type mdBadgeBlock struct{ Text, Color string }

// mdProgress 进度块。
type mdProgressBlock struct {
	Value, Max float32
	Label      string
}

// mdImage 图片。
type mdImageItem struct{ Src, Alt string }

// mdBlock 一个块。
type mdBlock struct {
	Kind    mdKind
	Level   int
	Inl     []mdInline
	Code    string
	Lang    string
	Items   []mdItem
	Ordered bool
	Start   int
	Quote   []*mdBlock
	Accent  string // quote 风格：note / tip / warning / success / error
	Table   *mdTableBlock
	Src     string
	Alt     string
	Title   string
	Tabs    []mdTab
	Open    bool
	Card    *mdCardBlock
	Badge   *mdBadgeBlock
	Prog    *mdProgressBlock
	Gallery []mdImageItem
	URL     string
	// Math 公式源码（mdMath）。
	Math string
	// Defs 定义列表项（mdDefList）。
	Defs []mdDefItem
	// Key 渲染时分配（标题锚点、交互状态键）。
	Key string
}

// mdDefItem 定义列表一项：术语 + 若干条释义。
type mdDefItem struct {
	Term []mdInline
	Defs [][]mdInline
}

// mdFootnote 脚注定义。
type mdFootnote struct {
	ID   string
	Inl  []mdInline
	Body []*mdBlock
}

// mdHeadingInfo 目录项。
type mdHeadingInfo struct {
	Level int
	Text  string
	Key   string
}

// mdDoc 解析结果。
type mdDoc struct {
	Blocks    []*mdBlock
	Headings  []mdHeadingInfo
	Footnotes []mdFootnote
}

var (
	mdHeadingRe  = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	mdHRRe       = regexp.MustCompile(`^\s{0,3}(?:(?:\*\s*){3,}|(?:-\s*){3,}|(?:_\s*){3,})$`)
	mdListRe     = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)
	mdTableSepRe = regexp.MustCompile(`^\s*\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)+\|?\s*$`)
	mdImageRe    = regexp.MustCompile(`^\s*!\[([^\]]*)\]\(\s*([^\s)]+)(?:\s+"([^"]*)")?\s*\)\s*$`)
	mdAnyImageRe = regexp.MustCompile(`!\[([^\]]*)\]\(\s*([^\s)]+)(?:\s+"([^"]*)")?\s*\)`)
	mdHideRe     = regexp.MustCompile(`(?s)\[记忆\].*?\[/记忆\]`)
	mdFenceRe    = regexp.MustCompile("^\\s{0,3}(```+|~~~+)(.*)$")
	mdCalloutRe  = regexp.MustCompile(`^\[!(NOTE|TIP|WARNING|IMPORTANT|CAUTION|SUCCESS|ERROR|INFO)\]\s*`)
	mdLooseURLRe = regexp.MustCompile(`^(https?://[^\s<>()]+)$`)
	// 脚注定义 `[^id]: 内容`
	mdFootDefRe = regexp.MustCompile(`^\s{0,3}\[\^([^\]]+)\]:\s?(.*)$`)
	// 定义列表 `: 释义`
	mdDefRe = regexp.MustCompile(`^\s{0,3}:\s+(.*)$`)
	// 裸链接（自动识别成可点击链接）
	mdURLRe = regexp.MustCompile(`https?://[^\s<>()（）【】，。、；]+`)
)

// isMathFence 判断代码块语言是否为公式块。
func isMathFence(lang string) bool {
	switch strings.ToLower(lang) {
	case "math", "latex", "tex", "katex":
		return true
	}
	return false
}

// isMermaidFence 判断代码块语言是否为 Mermaid 图。
func isMermaidFence(lang string) bool {
	return strings.EqualFold(lang, "mermaid")
}

// parseMarkdown 解析整篇文档（结果可复用，调用方负责缓存）。
func parseMarkdown(src string) *mdDoc {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	src = mdHideRe.ReplaceAllString(src, "")
	p := &mdParser{lines: strings.Split(src, "\n")}
	d := &mdDoc{}
	d.Blocks = p.parseBlocks()
	d.Headings = p.headings
	d.Footnotes = p.footnotes
	return d
}

type mdParser struct {
	lines     []string
	pos       int
	headings  []mdHeadingInfo
	footnotes []mdFootnote
	slugN     int
}

func (p *mdParser) parseBlocks() []*mdBlock {
	var out []*mdBlock
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		if strings.TrimSpace(line) == "" {
			p.pos++
			continue
		}
		if b := p.parseExt(); b != nil {
			out = append(out, b)
			continue
		}
		// 脚注定义（不产出块，收进 doc.Footnotes）
		if m := mdFootDefRe.FindStringSubmatch(line); m != nil {
			p.parseFootnoteDef(m[1], m[2])
			continue
		}
		// 块级公式 $$...$$ / \[...\]
		tline := strings.TrimSpace(line)
		if strings.HasPrefix(tline, "$$") || strings.HasPrefix(tline, "\\[") {
			out = append(out, p.parseMathBlock())
			continue
		}
		// 定义列表
		if p.isDefListStart() {
			out = append(out, p.parseDefList())
			continue
		}
		if m := mdFenceRe.FindStringSubmatch(line); m != nil {
			out = append(out, p.parseFence())
			continue
		}
		if m := mdHeadingRe.FindStringSubmatch(line); m != nil {
			b := &mdBlock{Kind: mdHeading, Level: len(m[1]), Inl: parseInline(m[2])}
			b.Key = p.nextHeadingKey(inlineText(b.Inl))
			p.headings = append(p.headings, mdHeadingInfo{Level: b.Level, Text: inlineText(b.Inl), Key: b.Key})
			p.pos++
			out = append(out, b)
			continue
		}
		if mdHRRe.MatchString(line) {
			p.pos++
			out = append(out, &mdBlock{Kind: mdHR})
			continue
		}
		if mdListRe.MatchString(line) {
			out = append(out, p.parseList())
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			out = append(out, p.parseQuote())
			continue
		}
		if p.isTableStart() {
			out = append(out, p.parseTable())
			continue
		}
		if m := mdImageRe.FindStringSubmatch(line); m != nil {
			p.pos++
			out = append(out, &mdBlock{Kind: mdImage, Src: m[2], Alt: m[1], Title: m[3]})
			continue
		}
		out = append(out, p.parseParagraph())
	}
	return out
}

// nextHeadingKey 生成稳定锚点（同标题加序号后缀）。
func (p *mdParser) nextHeadingKey(text string) string {
	p.slugN++
	return "md-h-" + strconv.Itoa(p.slugN)
}

// ---- 代码块 ----

func (p *mdParser) parseFence() *mdBlock {
	cur := p.lines[p.pos]
	m := mdFenceRe.FindStringSubmatch(cur)
	fence, info := "```", ""
	if m != nil {
		fence = m[1]
		info = strings.TrimSpace(m[2])
	}
	lang := ""
	if info != "" {
		lang = strings.Fields(info)[0]
	}
	p.pos++
	var body []string
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if strings.HasPrefix(strings.TrimSpace(l), fence) {
			p.pos++
			break
		}
		body = append(body, l)
		p.pos++
	}
	code := strings.Join(body, "\n")
	switch {
	case isMathFence(lang):
		return &mdBlock{Kind: mdMath, Lang: lang, Math: code, Code: code}
	case isMermaidFence(lang):
		return &mdBlock{Kind: mdMermaid, Lang: lang, Code: code}
	}
	return &mdBlock{Kind: mdCode, Lang: strings.TrimPrefix(lang, "."), Code: code}
}

// parseMathBlock 解析块级公式：$$...$$、\[...\]、单行 $$x$$。
func (p *mdParser) parseMathBlock() *mdBlock {
	line := strings.TrimSpace(p.lines[p.pos])
	open, closeMark := "$$", "$$"
	if strings.HasPrefix(line, "\\[") {
		open, closeMark = "\\[", "\\]"
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, open))
	p.pos++
	if i := strings.Index(rest, closeMark); i >= 0 {
		return &mdBlock{Kind: mdMath, Math: strings.TrimSpace(rest[:i])}
	}
	var body []string
	if rest != "" {
		body = append(body, rest)
	}
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		p.pos++
		if i := strings.Index(l, closeMark); i >= 0 {
			if pre := strings.TrimSpace(l[:i]); pre != "" {
				body = append(body, pre)
			}
			break
		}
		body = append(body, l)
	}
	return &mdBlock{Kind: mdMath, Math: strings.TrimSpace(strings.Join(body, "\n"))}
}

// parseFootnoteDef 收集脚注定义（含缩进续行）。
func (p *mdParser) parseFootnoteDef(id, first string) {
	p.pos++
	var body []string
	if s := strings.TrimSpace(first); s != "" {
		body = append(body, s)
	}
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if strings.TrimSpace(l) == "" {
			// 空行后仍是缩进内容才继续
			j := p.pos + 1
			for j < len(p.lines) && strings.TrimSpace(p.lines[j]) == "" {
				j++
			}
			if j < len(p.lines) && (strings.HasPrefix(p.lines[j], "    ") || strings.HasPrefix(p.lines[j], "\t")) {
				p.pos++
				continue
			}
			break
		}
		if !strings.HasPrefix(l, "    ") && !strings.HasPrefix(l, "\t") {
			break
		}
		body = append(body, strings.TrimSpace(l))
		p.pos++
	}
	p.footnotes = append(p.footnotes, mdFootnote{ID: id, Inl: parseInline(mdJoinLines(body))})
}

// isDefListStart 当前行是术语、下一行是「: 释义」时为定义列表。
func (p *mdParser) isDefListStart() bool {
	if p.pos+1 >= len(p.lines) {
		return false
	}
	cur := strings.TrimSpace(p.lines[p.pos])
	if cur == "" || strings.HasPrefix(cur, ":") || strings.HasPrefix(cur, "#") ||
		strings.HasPrefix(cur, "|") || strings.HasPrefix(cur, ">") {
		return false
	}
	if mdListRe.MatchString(p.lines[p.pos]) || mdFenceRe.MatchString(p.lines[p.pos]) {
		return false
	}
	return mdDefRe.MatchString(p.lines[p.pos+1])
}

// parseDefList 定义列表：术语 + 一行或多行「: 释义」。
func (p *mdParser) parseDefList() *mdBlock {
	b := &mdBlock{Kind: mdDefList}
	for p.pos < len(p.lines) {
		term := strings.TrimSpace(p.lines[p.pos])
		if term == "" || p.pos+1 >= len(p.lines) || !mdDefRe.MatchString(p.lines[p.pos+1]) {
			break
		}
		item := mdDefItem{Term: parseInline(term)}
		p.pos++
		for p.pos < len(p.lines) {
			m := mdDefRe.FindStringSubmatch(p.lines[p.pos])
			if m == nil {
				break
			}
			item.Defs = append(item.Defs, parseInline(strings.TrimSpace(m[1])))
			p.pos++
		}
		b.Defs = append(b.Defs, item)
	}
	return b
}

// ---- 引用 / 提示块 ----

func (p *mdParser) parseQuote() *mdBlock {
	var inner []string
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		t := strings.TrimLeft(l, " \t")
		if !strings.HasPrefix(t, ">") {
			break
		}
		t = strings.TrimPrefix(t, ">")
		t = strings.TrimPrefix(t, " ")
		inner = append(inner, t)
		p.pos++
	}
	sub := &mdParser{lines: inner}
	b := &mdBlock{Kind: mdQuote, Quote: sub.parseBlocks()}
	b.Accent = mdCalloutKind(inner)
	return b
}

// mdCalloutKind 识别引用块风格：GitHub [!NOTE] 或首行 emoji。
func mdCalloutKind(lines []string) string {
	first := ""
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			first = strings.TrimSpace(l)
			break
		}
	}
	if m := mdCalloutRe.FindStringSubmatch(first); m != nil {
		switch m[1] {
		case "TIP":
			return "tip"
		case "WARNING", "CAUTION":
			return "warning"
		case "IMPORTANT":
			return "important"
		case "SUCCESS":
			return "success"
		case "ERROR":
			return "error"
		}
		return "note"
	}
	switch {
	case strings.HasPrefix(first, "⚠"):
		return "warning"
	case strings.HasPrefix(first, "💡"):
		return "tip"
	case strings.HasPrefix(first, "✅"):
		return "success"
	case strings.HasPrefix(first, "❌"):
		return "error"
	case strings.HasPrefix(first, "🎯"):
		return "important"
	case strings.HasPrefix(first, "📝"), strings.HasPrefix(first, "📌"):
		return "note"
	}
	return ""
}

// ---- 列表 ----

func (p *mdParser) parseList() *mdBlock {
	first := mdListRe.FindStringSubmatch(p.lines[p.pos])
	baseIndent := len(first[1])
	ordered := first[2][0] >= '0' && first[2][0] <= '9'
	b := &mdBlock{Kind: mdList, Ordered: ordered}
	if ordered {
		n, _ := strconv.Atoi(strings.TrimRight(first[2], ".)"))
		b.Start = n
	}

	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		if strings.TrimSpace(line) == "" {
			// 空行后仍是同级/更深列表 → 继续
			j := p.pos + 1
			for j < len(p.lines) && strings.TrimSpace(p.lines[j]) == "" {
				j++
			}
			if j < len(p.lines) {
				ind := len(p.lines[j]) - len(strings.TrimLeft(p.lines[j], " "))
				if ind >= baseIndent && (mdListRe.MatchString(p.lines[j]) || ind > baseIndent) {
					p.pos = j
					continue
				}
			}
			break
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		m := mdListRe.FindStringSubmatch(line)
		if m == nil {
			if indent > baseIndent && len(b.Items) > 0 {
				sub := p.collectIndented(indent)
				subP := &mdParser{lines: sub}
				b.Items[len(b.Items)-1].Body = append(b.Items[len(b.Items)-1].Body, subP.parseBlocks()...)
				continue
			}
			break
		}
		if indent > baseIndent+1 {
			// 子列表（更深缩进）
			sub := p.collectIndented(indent)
			if len(b.Items) == 0 {
				b.Items = append(b.Items, mdItem{Task: -1})
			}
			subP := &mdParser{lines: sub}
			b.Items[len(b.Items)-1].Body = append(b.Items[len(b.Items)-1].Body, subP.parseBlocks()...)
			continue
		}
		if indent < baseIndent {
			break
		}
		text := m[3]
		item := mdItem{Task: -1}
		switch {
		case strings.HasPrefix(text, "[ ] "):
			item.Task = 0
			text = text[4:]
		case strings.HasPrefix(strings.ToLower(text), "[x] "):
			item.Task = 1
			text = text[4:]
		}
		item.Inl = parseInline(text)
		b.Items = append(b.Items, item)
		p.pos++

		// 该列表项后续的缩进内容（续行/子块）
		for p.pos < len(p.lines) {
			l := p.lines[p.pos]
			if strings.TrimSpace(l) == "" {
				break
			}
			li := len(l) - len(strings.TrimLeft(l, " "))
			if li <= baseIndent {
				break
			}
			sub := p.collectIndented(li)
			subP := &mdParser{lines: sub}
			b.Items[len(b.Items)-1].Body = append(b.Items[len(b.Items)-1].Body, subP.parseBlocks()...)
		}
	}
	return b
}

// collectIndented 收集缩进 >= indent 的连续行，并去掉 indent 个前导空格。
func (p *mdParser) collectIndented(indent int) []string {
	var out []string
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if strings.TrimSpace(l) == "" {
			// 空行：向后看是否仍有缩进内容
			j := p.pos + 1
			for j < len(p.lines) && strings.TrimSpace(p.lines[j]) == "" {
				j++
			}
			if j < len(p.lines) && len(p.lines[j])-len(strings.TrimLeft(p.lines[j], " ")) >= indent {
				out = append(out, "")
				p.pos++
				continue
			}
			break
		}
		li := len(l) - len(strings.TrimLeft(l, " "))
		if li < indent {
			break
		}
		if len(l) >= indent {
			out = append(out, l[indent:])
		} else {
			out = append(out, strings.TrimLeft(l, " "))
		}
		p.pos++
	}
	return out
}

// ---- 表格 ----

func (p *mdParser) isTableStart() bool {
	if p.pos+1 >= len(p.lines) {
		return false
	}
	return strings.Contains(p.lines[p.pos], "|") && mdTableSepRe.MatchString(p.lines[p.pos+1])
}

func (p *mdParser) parseTable() *mdBlock {
	head := mdSplitRow(p.lines[p.pos])
	seps := mdSplitRow(p.lines[p.pos+1])
	aligns := make([]Align, len(seps))
	for i, s := range seps {
		s = strings.TrimSpace(s)
		switch {
		case strings.HasPrefix(s, ":") && strings.HasSuffix(s, ":"):
			aligns[i] = AlignCenter
		case strings.HasSuffix(s, ":"):
			aligns[i] = AlignEnd
		default:
			aligns[i] = AlignStart
		}
	}
	t := &mdTableBlock{}
	for i, c := range head {
		a := AlignStart
		if i < len(aligns) {
			a = aligns[i]
		}
		t.Header = append(t.Header, mdCell{Align: a, Inl: parseInline(strings.TrimSpace(c))})
	}
	p.pos += 2
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if strings.TrimSpace(l) == "" || !strings.Contains(l, "|") {
			break
		}
		cells := mdSplitRow(l)
		var row []mdCell
		for i := range t.Header {
			c := ""
			if i < len(cells) {
				c = strings.TrimSpace(cells[i])
			}
			row = append(row, mdCell{Align: t.Header[i].Align, Inl: parseInline(c)})
		}
		t.Rows = append(t.Rows, row)
		p.pos++
	}
	return &mdBlock{Kind: mdTable, Table: t}
}

// mdSplitRow 按 | 切分表格行（支持 \| 转义）。
func mdSplitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	var out []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			cur.WriteByte('|')
			i++
		case s[i] == '|':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(s[i])
		}
	}
	out = append(out, cur.String())
	return out
}

// ---- 段落 ----

func (p *mdParser) parseParagraph() *mdBlock {
	var lines []string
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if strings.TrimSpace(l) == "" {
			break
		}
		if len(lines) > 0 && p.startsBlock(l) {
			break
		}
		lines = append(lines, l)
		p.pos++
	}
	return &mdBlock{Kind: mdParagraph, Inl: parseInline(mdJoinLines(lines))}
}

// startsBlock 判断该行会不会打断段落。
func (p *mdParser) startsBlock(line string) bool {
	t := strings.TrimSpace(line)
	if mdFenceRe.MatchString(line) || mdHeadingRe.MatchString(line) || mdHRRe.MatchString(line) {
		return true
	}
	if mdListRe.MatchString(line) || strings.HasPrefix(t, ">") {
		return true
	}
	if strings.HasPrefix(t, "[") && mdExtTags(t) {
		return true
	}
	if mdImageRe.MatchString(line) {
		return true
	}
	if strings.HasPrefix(t, "$$") || strings.HasPrefix(t, "\\[") {
		return true
	}
	if mdFootDefRe.MatchString(line) || p.isDefListStart() {
		return true
	}
	return p.isTableStart()
}

// mdJoinLines 拼接段落行：中文之间不加空格，行尾两空格 / <br> 视为硬换行。
func mdJoinLines(lines []string) string {
	var sb strings.Builder
	for i, ln := range lines {
		hard := strings.HasSuffix(ln, "  ") || strings.HasSuffix(strings.TrimRight(ln, " "), "\\") || strings.HasSuffix(strings.TrimSpace(ln), "<br>") || strings.HasSuffix(strings.TrimSpace(ln), "<br/>")
		ln = strings.TrimRight(ln, " ")
		ln = strings.TrimSuffix(ln, "\\")
		if i > 0 {
			if !hard {
				prev := lastRune(sb.String())
				next := firstRune(ln)
				if !(isWideRune(prev) && isWideRune(next)) && prev != ' ' {
					sb.WriteString(" ")
				}
			}
		}
		sb.WriteString(strings.TrimLeft(ln, " \t"))
		if hard {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func lastRune(s string) rune {
	rs := []rune(s)
	if len(rs) == 0 {
		return 0
	}
	return rs[len(rs)-1]
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

// ---- 扩展块 ----

// mdExtTags 是否为扩展块起始行。
func mdExtTags(t string) bool {
	low := strings.ToLower(t)
	for _, tag := range []string{"[gallery]", "[tabs]", "[details", "[card", "[badge", "[progress", "[video", "[audio", "[toc]"} {
		if strings.HasPrefix(low, tag) {
			return true
		}
	}
	return false
}

func (p *mdParser) parseExt() *mdBlock {
	t := strings.TrimSpace(p.lines[p.pos])
	low := strings.ToLower(t)
	switch {
	case low == "[toc]":
		p.pos++
		return &mdBlock{Kind: mdTOC}
	case strings.HasPrefix(low, "[gallery]"):
		body := p.collectBlock("[/gallery]")
		b := &mdBlock{Kind: mdGallery}
		for _, m := range mdAnyImageRe.FindAllStringSubmatch(body, -1) {
			b.Gallery = append(b.Gallery, mdImageItem{Src: m[2], Alt: m[1]})
		}
		return b
	case strings.HasPrefix(low, "[tabs]"):
		body := p.collectBlock("[/tabs]")
		b := &mdBlock{Kind: mdTabs}
		rest := body
		for {
			i := strings.Index(strings.ToLower(rest), "[tab")
			if i < 0 {
				break
			}
			j := strings.Index(rest[i:], "]")
			if j < 0 {
				break
			}
			head := rest[i : i+j+1]
			label := mdAttr(head, "label")
			rest = rest[i+j+1:]
			k := strings.Index(strings.ToLower(rest), "[/tab]")
			if k < 0 {
				break
			}
			content := rest[:k]
			rest = rest[k+len("[/tab]"):]
			sub := &mdParser{lines: strings.Split(content, "\n")}
			b.Tabs = append(b.Tabs, mdTab{Label: label, Blocks: sub.parseBlocks()})
		}
		return b
	case strings.HasPrefix(low, "[details"):
		headEnd := strings.Index(t, "]")
		head := t
		if headEnd >= 0 {
			head = t[:headEnd+1]
		}
		body := p.collectBlock("[/details]")
		sub := &mdParser{lines: strings.Split(body, "\n")}
		return &mdBlock{Kind: mdDetails, Title: mdAttr(head, "title"), Quote: sub.parseBlocks()}
	case strings.HasPrefix(low, "[card"):
		headEnd := strings.Index(t, "]")
		head := t
		if headEnd >= 0 {
			head = t[:headEnd+1]
		}
		body := p.collectBlock("[/card]")
		sub := &mdParser{lines: strings.Split(body, "\n")}
		return &mdBlock{Kind: mdCard, Card: &mdCardBlock{
			Title:    mdAttr(head, "title"),
			Subtitle: mdAttr(head, "subtitle"),
			Image:    mdAttr(head, "image"),
			URL:      mdAttr(head, "url"),
			Blocks:   sub.parseBlocks(),
		}}
	case strings.HasPrefix(low, "[badge"):
		p.pos++
		return &mdBlock{Kind: mdBadge, Badge: &mdBadgeBlock{Text: mdAttr(t, "text"), Color: strings.ToLower(mdAttr(t, "color"))}}
	case strings.HasPrefix(low, "[progress"):
		p.pos++
		v, _ := strconv.ParseFloat(mdAttr(t, "value"), 32)
		mx, err := strconv.ParseFloat(mdAttr(t, "max"), 32)
		if err != nil || mx <= 0 {
			mx = 100
		}
		return &mdBlock{Kind: mdProgress, Prog: &mdProgressBlock{Value: float32(v), Max: float32(mx), Label: mdAttr(t, "label")}}
	case strings.HasPrefix(low, "[video"), strings.HasPrefix(low, "[audio"):
		p.pos++
		kind := "video"
		if strings.HasPrefix(low, "[audio") {
			kind = "audio"
		}
		return &mdBlock{Kind: mdMedia, URL: mdAttr(t, "src"), Title: mdAttr(t, "title"), Lang: kind}
	}
	return nil
}

// collectBlock 收集到闭合标记为止的正文（支持游离的未闭合块）。
func (p *mdParser) collectBlock(closing string) string {
	p.pos++
	var body []string
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), closing) {
			p.pos++
			break
		}
		body = append(body, l)
		p.pos++
	}
	return strings.Join(body, "\n")
}

// mdAttr 解析 [tag name="value"] 里的属性。
func mdAttr(s, name string) string {
	low := strings.ToLower(s)
	i := strings.Index(low, strings.ToLower(name)+"=")
	for i >= 0 {
		// 确保是完整属性名（前一个字符不是标识符字符）
		if i == 0 || !isIdentChar(rune(low[i-1])) {
			break
		}
		next := strings.Index(low[i+1:], strings.ToLower(name)+"=")
		if next < 0 {
			return ""
		}
		i += 1 + next
	}
	if i < 0 {
		return ""
	}
	rest := s[i+len(name)+1:]
	rest = strings.TrimLeft(rest, " =")
	if rest == "" {
		return ""
	}
	switch rest[0] {
	case '"':
		if j := strings.IndexByte(rest[1:], '"'); j >= 0 {
			return rest[1 : 1+j]
		}
	case '\'':
		if j := strings.IndexByte(rest[1:], '\''); j >= 0 {
			return rest[1 : 1+j]
		}
	default:
		end := strings.IndexAny(rest, " ]")
		if end < 0 {
			return rest
		}
		return rest[:end]
	}
	return ""
}

func isIdentChar(r rune) bool {
	return r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// ---- 行内解析 ----

// parseInline 解析行内样式（支持嵌套，最多按顺序递归处理）。
func parseInline(s string) []mdInline {
	if s == "" {
		return nil
	}
	rs := []rune(s)
	out := make([]mdInline, 0, 8)
	var buf []rune
	flush := func() {
		if len(buf) > 0 {
			out = append(out, mdInline{Text: mdDecodeEntities(string(buf))})
			buf = buf[:0]
		}
	}
	emit := func(runs []mdInline, mod func(*mdInline)) {
		for i := range runs {
			if mod != nil {
				mod(&runs[i])
			}
			out = append(out, runs[i])
		}
	}

	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch r {
		case '\\':
			if i+1 < len(rs) {
				i++
				if rs[i] == '\n' {
					flush()
					out = append(out, mdInline{Br: true})
				} else {
					buf = append(buf, rs[i])
				}
			}
		case '\n':
			flush()
			out = append(out, mdInline{Br: true})
		case '`':
			if j := mdFindRune(rs, i+1, '`'); j > i {
				flush()
				out = append(out, mdInline{Text: strings.TrimSpace(string(rs[i+1 : j])), Code: true})
				i = j
			} else {
				buf = append(buf, r)
			}
		case '*', '_':
			// 避免误判 snake_case
			if r == '_' && i > 0 && isIdentChar(rs[i-1]) {
				buf = append(buf, r)
				continue
			}
			if i+1 < len(rs) && rs[i+1] == r {
				if j := mdFindPair(rs, i+2, r); j > 0 {
					flush()
					emit(parseInline(string(rs[i+2:j])), func(x *mdInline) { x.Bold = true })
					i = j + 1
					continue
				}
			}
			if j := mdFindRune(rs, i+1, r); j > i {
				flush()
				emit(parseInline(string(rs[i+1:j])), func(x *mdInline) { x.Italic = true })
				i = j
				continue
			}
			buf = append(buf, r)
		case '~':
			if i+1 < len(rs) && rs[i+1] == '~' {
				if j := mdFindPair(rs, i+2, '~'); j > 0 {
					flush()
					emit(parseInline(string(rs[i+2:j])), func(x *mdInline) { x.Strike = true })
					i = j + 1
					continue
				}
			}
			// 下标 ~x~
			if j := mdFindRune(rs, i+1, '~'); j > i+1 {
				sub := string(rs[i+1 : j])
				if !strings.ContainsAny(sub, " \t") && len([]rune(sub)) <= 24 {
					flush()
					out = append(out, mdInline{Text: sub, Sub: sub})
					i = j
					continue
				}
			}
			buf = append(buf, r)
		case '^':
			// 上标 ^x^
			if j := mdFindRune(rs, i+1, '^'); j > i+1 {
				sup := string(rs[i+1 : j])
				if !strings.ContainsAny(sup, " \t") && len([]rune(sup)) <= 24 {
					flush()
					out = append(out, mdInline{Text: sup, Sup: sup})
					i = j
					continue
				}
			}
			buf = append(buf, r)
		case '$':
			// 行内公式 $...$（两端不成对/内容为空时按普通字符处理）
			if j := mdInlineMathEnd(rs, i); j > 0 {
				flush()
				out = append(out, mdInline{Math: strings.TrimSpace(string(rs[i+1 : j]))})
				i = j
				continue
			}
			buf = append(buf, r)
		case '=':
			if i+1 < len(rs) && rs[i+1] == '=' {
				if j := mdFindPair(rs, i+2, '='); j > 0 {
					flush()
					emit(parseInline(string(rs[i+2:j])), func(x *mdInline) { x.Mark = true })
					i = j + 1
					continue
				}
			}
			buf = append(buf, r)
		case '!':
			if i+1 < len(rs) && rs[i+1] == '[' {
				if text, href, next, ok := mdParseLink(rs, i+1); ok {
					flush()
					out = append(out, mdInline{Text: text, Link: true, Href: href})
					i = next
					continue
				}
			}
			buf = append(buf, r)
		case '[':
			// 脚注引用 [^id]
			if i+1 < len(rs) && rs[i+1] == '^' {
				if j := mdFindRune(rs, i+2, ']'); j > i+2 {
					flush()
					out = append(out, mdInline{FnRef: string(rs[i+2 : j])})
					i = j
					continue
				}
			}
			if text, href, next, ok := mdParseLink(rs, i); ok {
				flush()
				out = append(out, mdInline{Text: text, Link: true, Href: href})
				i = next
				continue
			}
			buf = append(buf, r)
		case '<':
			if tag, end, ok := mdReadTag(rs, i); ok {
				switch strings.ToLower(tag) {
				case "br", "br/", "br /":
					flush()
					out = append(out, mdInline{Br: true})
					i = end
					continue
				case "kbd":
					if j := mdFindTagClose(rs, end+1, "kbd"); j > 0 {
						flush()
						out = append(out, mdInline{Text: string(rs[end+1 : j]), Kbd: true})
						i = j + len("</kbd>") - 1
						continue
					}
				}
			}
			buf = append(buf, r)
		default:
			buf = append(buf, r)
		}
	}
	flush()
	return mdLinkify(out)
}

// mdInlineMathEnd 找到行内公式的结束 $（要求两端不留空白、内容非空）。
func mdInlineMathEnd(rs []rune, open int) int {
	if open+1 >= len(rs) {
		return -1
	}
	nxt := rs[open+1]
	if nxt == ' ' || nxt == '\t' || nxt == '$' {
		return -1
	}
	for i := open + 1; i < len(rs); i++ {
		switch rs[i] {
		case '\\':
			i++
			continue
		case '\n':
			return -1
		case '$':
			// 结束符前不能是空白
			if rs[i-1] == ' ' || rs[i-1] == '\t' {
				return -1
			}
			// 结束符后不能紧跟数字（避免 $5 与 $6 被误判）
			if i+1 < len(rs) && rs[i+1] >= '0' && rs[i+1] <= '9' {
				return -1
			}
			return i
		}
	}
	return -1
}

// mdLinkify 把纯文本里的裸 URL 拆成可点击链接（markdown 语法之外的兜底）。
func mdLinkify(runs []mdInline) []mdInline {
	hasURL := false
	for _, r := range runs {
		if r.Text != "" && !r.Code && !r.Kbd && !r.Link && r.Math == "" && r.FnRef == "" && mdURLRe.MatchString(r.Text) {
			hasURL = true
			break
		}
	}
	if !hasURL {
		return runs
	}
	out := make([]mdInline, 0, len(runs)+4)
	for _, r := range runs {
		if r.Text == "" || r.Code || r.Kbd || r.Link || r.Math != "" || r.FnRef != "" || !mdURLRe.MatchString(r.Text) {
			out = append(out, r)
			continue
		}
		s := r.Text
		for {
			loc := mdURLRe.FindStringIndex(s)
			if loc == nil {
				break
			}
			// 去掉链接尾部常见的标点
			end := loc[1]
			for end > loc[0] && strings.ContainsRune(".,;:!?)]}>'\"”，。、；）】", rune(s[end-1])) {
				end--
			}
			if end <= loc[0] {
				break
			}
			if pre := s[:loc[0]]; pre != "" {
				out = append(out, mdInline{Text: pre})
			}
			u := s[loc[0]:end]
			out = append(out, mdInline{Text: u, Link: true, Href: u})
			s = s[end:]
			if s == "" {
				break
			}
		}
		if s != "" {
			out = append(out, mdInline{Text: s})
		}
	}
	return out
}

// mdParseLink 解析 [text](href)，返回 text / href / 结束下标。
func mdParseLink(rs []rune, open int) (text, href string, next int, ok bool) {
	if open >= len(rs) || rs[open] != '[' {
		return "", "", 0, false
	}
	depth := 0
	closeIdx := -1
	for i := open; i < len(rs); i++ {
		switch rs[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				closeIdx = i
			}
		}
		if closeIdx >= 0 {
			break
		}
	}
	if closeIdx < 0 || closeIdx+1 >= len(rs) || rs[closeIdx+1] != '(' {
		return "", "", 0, false
	}
	end := -1
	for i := closeIdx + 2; i < len(rs); i++ {
		if rs[i] == ')' {
			end = i
			break
		}
	}
	if end < 0 {
		return "", "", 0, false
	}
	text = strings.TrimSpace(string(rs[open+1 : closeIdx]))
	raw := strings.TrimSpace(string(rs[closeIdx+2 : end]))
	if raw == "" {
		return "", "", 0, false
	}
	if i := strings.Index(raw, " \""); i > 0 && strings.HasSuffix(raw, "\"") {
		raw = raw[:i]
	}
	return text, strings.TrimSpace(raw), end, true
}

// mdReadTag 读取 <tag ...> 标签名与结束下标。
func mdReadTag(rs []rune, i int) (tag string, end int, ok bool) {
	j := i + 1
	start := j
	for j < len(rs) && (isIdentChar(rs[j]) || rs[j] == '/') {
		j++
	}
	if j == start {
		return "", 0, false
	}
	for j < len(rs) && rs[j] != '>' {
		j++
	}
	if j >= len(rs) {
		return "", 0, false
	}
	return string(rs[start:j]), j, true
}

// mdFindTagClose 找到 </tag> 的起始位置。
func mdFindTagClose(rs []rune, from int, tag string) int {
	needle := []rune("</" + tag + ">")
	for i := from; i+len(needle) <= len(rs); i++ {
		match := true
		for k, r := range needle {
			if rs[i+k] != r {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func mdFindRune(rs []rune, from int, target rune) int {
	for i := from; i < len(rs); i++ {
		if rs[i] == target {
			return i
		}
	}
	return -1
}

// mdFindPair 找到成对的同类字符（跳过转义）。
func mdFindPair(rs []rune, from int, target rune) int {
	for i := from; i < len(rs); i++ {
		if rs[i] == '\\' {
			i++
			continue
		}
		if rs[i] == target {
			return i
		}
	}
	return -1
}

// mdDecodeEntities 解码常见 HTML 实体。
func mdDecodeEntities(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	r := strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", "\"", "&#39;", "'", "&copy;", "©", "&mdash;", "—", "&hellip;", "…",
	)
	return r.Replace(s)
}

// inlineText 提取行内纯文本（标题锚点/目录用）。
func inlineText(runs []mdInline) string {
	var sb strings.Builder
	for _, r := range runs {
		if r.Br {
			sb.WriteString(" ")
			continue
		}
		sb.WriteString(r.Text)
	}
	return strings.TrimSpace(sb.String())
}
