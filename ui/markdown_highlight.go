package ui

import "strings"

// 轻量语法着色：按「词法扫描」切出关键字/字符串/注释/数字/函数名等片段，
// 渲染层再映射为颜色。不做完整 AST，目标是代码块可读性（对齐 Web 版
// highlight.js 的常见语言子集），而非精确高亮。

// mdHLSpan 一段着色文本。
type mdHLSpan struct {
	Text string
	Kind string // kw / str / com / num / fn / typ / bi / attr / tag / punc / txt
}

// mdHighlight 返回逐行着色结果（行数与 strings.Split(code,"\n") 一致）。
func mdHighlight(lang, code string) [][]mdHLSpan {
	lines := strings.Split(code, "\n")
	out := make([][]mdHLSpan, len(lines))
	fn := mdHighlighter(lang)
	for i, l := range lines {
		out[i] = fn(l)
	}
	return out
}

// mdHighlighter 按语言选择扫描器。
func mdHighlighter(lang string) func(string) []mdHLSpan {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go", "golang":
		return func(s string) []mdHLSpan {
			return mdScanCLike(s, mdGoKeywords, mdGoTypes, mdGoBuiltins, "//", true)
		}
	case "js", "javascript", "jsx", "ts", "typescript", "tsx", "mjs", "cjs":
		return func(s string) []mdHLSpan {
			return mdScanCLike(s, mdJSKeywords, mdJSTypes, mdJSBuiltins, "//", true)
		}
	case "java", "kt", "kotlin", "kts", "cs", "csharp", "swift", "dart":
		return func(s string) []mdHLSpan {
			return mdScanCLike(s, mdJavaKeywords, mdJavaTypes, nil, "//", false)
		}
	case "c", "h", "cpp", "c++", "cc", "hpp", "rs", "rust", "objc", "m":
		return func(s string) []mdHLSpan {
			return mdScanCLike(s, mdCKeywords, mdCTypes, nil, "//", false)
		}
	case "py", "python":
		return mdScanPython
	case "json", "jsonc":
		return mdScanJSON
	case "sh", "bash", "shell", "zsh", "console", "ps1", "powershell":
		return mdScanShell
	case "sql", "mysql", "postgres", "sqlite":
		return mdScanSQL
	case "html", "xml", "vue", "svg", "htm":
		return mdScanHTML
	case "css", "scss", "less":
		return mdScanCSS
	case "yaml", "yml", "toml", "ini", "conf":
		return mdScanYAML
	}
	return func(s string) []mdHLSpan { return []mdHLSpan{{Text: s, Kind: "txt"}} }
}

// 关键字表见 markdown_keywords.go
func mdSet(words string) map[string]bool {
	m := make(map[string]bool, 64)
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

// mdScanCLike 通用 C 家族扫描（Go/JS/Java/C/Rust 等）。
func mdScanCLike(s string, kw, types, builtins map[string]bool, commentPrefix string, backtick bool) []mdHLSpan {
	rs := []rune(s)
	var out []mdHLSpan
	add := func(txt, kind string) {
		if txt == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Kind == kind {
			out[n-1].Text += txt
			return
		}
		out = append(out, mdHLSpan{Text: txt, Kind: kind})
	}
	inBlockComment := strings.HasPrefix(strings.TrimSpace(s), "*") || strings.HasPrefix(strings.TrimSpace(s), "/*")
	if inBlockComment {
		return []mdHLSpan{{Text: s, Kind: "com"}}
	}
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == ' ' || r == '\t':
			j := i
			for j < len(rs) && (rs[j] == ' ' || rs[j] == '\t') {
				j++
			}
			add(string(rs[i:j]), "txt")
			i = j
		case strings.HasPrefix(string(rs[i:]), commentPrefix):
			add(string(rs[i:]), "com")
			i = len(rs)
		case strings.HasPrefix(string(rs[i:]), "/*"):
			j := strings.Index(string(rs[i+2:]), "*/")
			if j < 0 {
				add(string(rs[i:]), "com")
				i = len(rs)
			} else {
				add(string(rs[i:i+2+j+2]), "com")
				i += 2 + j + 2
			}
		case r == '"' || r == '\'' || (backtick && r == '`'):
			q := r
			j := i + 1
			for j < len(rs) {
				if rs[j] == '\\' {
					j += 2
					continue
				}
				if rs[j] == q {
					j++
					break
				}
				j++
			}
			if j > len(rs) {
				j = len(rs)
			}
			add(string(rs[i:j]), "str")
			i = j
		case r >= '0' && r <= '9':
			j := i
			for j < len(rs) && (isIdentChar(rs[j]) || rs[j] == '.' || rs[j] == 'x' || rs[j] == 'X') {
				j++
			}
			add(string(rs[i:j]), "num")
			i = j
		case isIdentChar(r) || r == '#' || r == '@':
			j := i
			if r == '#' || r == '@' {
				j++
			}
			for j < len(rs) && isIdentChar(rs[j]) {
				j++
			}
			word := string(rs[i:j])
			kind := "txt"
			switch {
			case kw[word]:
				kind = "kw"
			case types[word]:
				kind = "typ"
			case builtins != nil && builtins[word]:
				kind = "bi"
			case mdNextNonSpace(rs, j) == '(':
				kind = "fn"
			}
			add(word, kind)
			i = j
		default:
			add(string(r), "punc")
			i++
		}
	}
	return out
}

func mdNextNonSpace(rs []rune, i int) rune {
	for ; i < len(rs); i++ {
		if rs[i] != ' ' && rs[i] != '\t' {
			return rs[i]
		}
	}
	return 0
}

// mdScanPython Python 扫描（# 注释、三引号字符串）。
func mdScanPython(s string) []mdHLSpan {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return []mdHLSpan{{Text: s, Kind: "txt"}}
	}
	rs := []rune(s)
	var out []mdHLSpan
	add := func(txt, kind string) {
		if txt == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Kind == kind {
			out[n-1].Text += txt
			return
		}
		out = append(out, mdHLSpan{Text: txt, Kind: kind})
	}
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '#':
			add(string(rs[i:]), "com")
			i = len(rs)
		case r == '"' || r == '\'':
			q := r
			triple := i+2 < len(rs) && rs[i+1] == q && rs[i+2] == q
			j := i + 1
			if triple {
				j = i + 3
				for j+2 < len(rs) && !(rs[j] == q && rs[j+1] == q && rs[j+2] == q) {
					j++
				}
				j += 3
				if j > len(rs) {
					j = len(rs)
				}
			} else {
				for j < len(rs) {
					if rs[j] == '\\' {
						j += 2
						continue
					}
					if rs[j] == q {
						j++
						break
					}
					j++
				}
			}
			add(string(rs[i:j]), "str")
			i = j
		case r >= '0' && r <= '9':
			j := i
			for j < len(rs) && (isIdentChar(rs[j]) || rs[j] == '.') {
				j++
			}
			add(string(rs[i:j]), "num")
			i = j
		case isIdentChar(r):
			j := i
			for j < len(rs) && isIdentChar(rs[j]) {
				j++
			}
			w := string(rs[i:j])
			kind := "txt"
			switch {
			case mdPyKeywords[w]:
				kind = "kw"
			case mdPyBuiltins[w]:
				kind = "bi"
			case mdNextNonSpace(rs, j) == '(':
				kind = "fn"
			}
			add(w, kind)
			i = j
		default:
			add(string(r), "punc")
			i++
		}
	}
	return out
}

// mdScanJSON JSON 扫描（key 高亮）。
func mdScanJSON(s string) []mdHLSpan {
	rs := []rune(s)
	var out []mdHLSpan
	add := func(txt, kind string) {
		if txt == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Kind == kind {
			out[n-1].Text += txt
			return
		}
		out = append(out, mdHLSpan{Text: txt, Kind: kind})
	}
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '"':
			j := i + 1
			for j < len(rs) {
				if rs[j] == '\\' {
					j += 2
					continue
				}
				if rs[j] == '"' {
					j++
					break
				}
				j++
			}
			if j > len(rs) {
				j = len(rs)
			}
			if mdNextNonSpace(rs, j) == ':' {
				add(string(rs[i:j]), "attr")
			} else {
				add(string(rs[i:j]), "str")
			}
			i = j
		case r >= '0' && r <= '9' || r == '-':
			j := i + 1
			for j < len(rs) && (rs[j] >= '0' && rs[j] <= '9' || rs[j] == '.' || rs[j] == 'e' || rs[j] == 'E' || rs[j] == '+' || rs[j] == '-') {
				j++
			}
			add(string(rs[i:j]), "num")
			i = j
		case isIdentChar(r):
			j := i
			for j < len(rs) && isIdentChar(rs[j]) {
				j++
			}
			w := string(rs[i:j])
			if w == "true" || w == "false" || w == "null" {
				add(w, "kw")
			} else {
				add(w, "txt")
			}
			i = j
		default:
			add(string(r), "punc")
			i++
		}
	}
	return out
}

// mdScanShell Shell / PowerShell。
func mdScanShell(s string) []mdHLSpan {
	rs := []rune(s)
	var out []mdHLSpan
	add := func(txt, kind string) {
		if txt == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Kind == kind {
			out[n-1].Text += txt
			return
		}
		out = append(out, mdHLSpan{Text: txt, Kind: kind})
	}
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '#':
			add(string(rs[i:]), "com")
			i = len(rs)
		case r == '"' || r == '\'':
			q := r
			j := i + 1
			for j < len(rs) && rs[j] != q {
				j++
			}
			if j < len(rs) {
				j++
			}
			add(string(rs[i:j]), "str")
			i = j
		case r == '-' && i+1 < len(rs) && (rs[i+1] == '-' || isIdentChar(rs[i+1])):
			j := i + 1
			if rs[i+1] == '-' {
				j++
			}
			for j < len(rs) && isIdentChar(rs[j]) {
				j++
			}
			add(string(rs[i:j]), "attr")
			i = j
		case isIdentChar(r) || r == '.' || r == '/':
			j := i
			for j < len(rs) && (isIdentChar(rs[j]) || rs[j] == '.' || rs[j] == '/' || rs[j] == '-') {
				j++
			}
			w := string(rs[i:j])
			kind := "txt"
			if mdShellKeywords[w] {
				kind = "kw"
			}
			add(w, kind)
			i = j
		default:
			add(string(r), "punc")
			i++
		}
	}
	return out
}

// mdScanSQL SQL。
func mdScanSQL(s string) []mdHLSpan {
	rs := []rune(s)
	var out []mdHLSpan
	add := func(txt, kind string) {
		if txt == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Kind == kind {
			out[n-1].Text += txt
			return
		}
		out = append(out, mdHLSpan{Text: txt, Kind: kind})
	}
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case strings.HasPrefix(string(rs[i:]), "--"):
			add(string(rs[i:]), "com")
			i = len(rs)
		case r == '\'':
			j := i + 1
			for j < len(rs) {
				if rs[j] == '\'' {
					if j+1 < len(rs) && rs[j+1] == '\'' {
						j += 2
						continue
					}
					j++
					break
				}
				j++
			}
			add(string(rs[i:j]), "str")
			i = j
		case r >= '0' && r <= '9':
			j := i
			for j < len(rs) && (rs[j] >= '0' && rs[j] <= '9' || rs[j] == '.') {
				j++
			}
			add(string(rs[i:j]), "num")
			i = j
		case isIdentChar(r):
			j := i
			for j < len(rs) && isIdentChar(rs[j]) {
				j++
			}
			w := string(rs[i:j])
			if mdSQLKeywords[strings.ToLower(w)] {
				add(w, "kw")
			} else {
				add(w, "txt")
			}
			i = j
		default:
			add(string(r), "punc")
			i++
		}
	}
	return out
}

// mdScanHTML 标签 / 属性 / 文本。
func mdScanHTML(s string) []mdHLSpan {
	rs := []rune(s)
	var out []mdHLSpan
	add := func(txt, kind string) {
		if txt == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Kind == kind {
			out[n-1].Text += txt
			return
		}
		out = append(out, mdHLSpan{Text: txt, Kind: kind})
	}
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case strings.HasPrefix(string(rs[i:]), "<!--"):
			j := strings.Index(string(rs[i+4:]), "-->")
			if j < 0 {
				add(string(rs[i:]), "com")
				i = len(rs)
			} else {
				add(string(rs[i:i+4+j+3]), "com")
				i += 4 + j + 3
			}
		case r == '<':
			j := i + 1
			if j < len(rs) && rs[j] == '/' {
				j++
			}
			for j < len(rs) && rs[j] != '>' {
				j++
			}
			if j < len(rs) {
				j++
			}
			// 标签内属性/字符串简单着色
			seg := string(rs[i:j])
			parts := mdSplitTag(seg)
			for _, p := range parts {
				add(p.Text, p.Kind)
			}
			i = j
		case r == '"' || r == '\'':
			q := r
			j := i + 1
			for j < len(rs) && rs[j] != q {
				j++
			}
			if j < len(rs) {
				j++
			}
			add(string(rs[i:j]), "str")
			i = j
		default:
			j := i
			for j < len(rs) && rs[j] != '<' && rs[j] != '"' && rs[j] != '\'' {
				j++
			}
			add(string(rs[i:j]), "txt")
			i = j
		}
	}
	return out
}

func mdSplitTag(seg string) []mdHLSpan {
	rs := []rune(seg)
	var out []mdHLSpan
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '<' || r == '>' || r == '/' || r == '=':
			out = append(out, mdHLSpan{Text: string(r), Kind: "punc"})
			i++
		case isIdentChar(r) || r == '-' || r == ':' || r == '@':
			j := i
			for j < len(rs) && (isIdentChar(rs[j]) || rs[j] == '-' || rs[j] == ':' || rs[j] == '@' || rs[j] == '.') {
				j++
			}
			kind := "attr"
			// 标签名（紧随 < 或 </）
			k := i - 1
			for k >= 0 && (rs[k] == ' ' || rs[k] == '\t') {
				k--
			}
			if k >= 0 && (rs[k] == '<' || rs[k] == '/') {
				kind = "tag"
			}
			out = append(out, mdHLSpan{Text: string(rs[i:j]), Kind: kind})
			i = j
		case r == '"' || r == '\'':
			q := r
			j := i + 1
			for j < len(rs) && rs[j] != q {
				j++
			}
			if j < len(rs) {
				j++
			}
			out = append(out, mdHLSpan{Text: string(rs[i:j]), Kind: "str"})
			i = j
		default:
			out = append(out, mdHLSpan{Text: string(r), Kind: "txt"})
			i++
		}
	}
	return out
}

// mdScanCSS 选择器 / 属性 / 值。
func mdScanCSS(s string) []mdHLSpan {
	rs := []rune(s)
	var out []mdHLSpan
	add := func(txt, kind string) {
		if txt == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Kind == kind {
			out[n-1].Text += txt
			return
		}
		out = append(out, mdHLSpan{Text: txt, Kind: kind})
	}
	colon := false
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case strings.HasPrefix(string(rs[i:]), "/*"):
			j := strings.Index(string(rs[i+2:]), "*/")
			if j < 0 {
				add(string(rs[i:]), "com")
				i = len(rs)
			} else {
				add(string(rs[i:i+2+j+2]), "com")
				i += 2 + j + 2
			}
		case r == ':':
			colon = true
			add(":", "punc")
			i++
		case r == ';' || r == '{' || r == '}':
			colon = false
			add(string(r), "punc")
			i++
		case r == '"' || r == '\'':
			q := r
			j := i + 1
			for j < len(rs) && rs[j] != q {
				j++
			}
			if j < len(rs) {
				j++
			}
			add(string(rs[i:j]), "str")
			i = j
		case isIdentChar(r) || r == '-' || r == '#' || r == '.':
			j := i
			for j < len(rs) && (isIdentChar(rs[j]) || rs[j] == '-' || rs[j] == '#' || rs[j] == '.' || rs[j] == '%') {
				j++
			}
			w := string(rs[i:j])
			kind := "txt"
			switch {
			case colon:
				kind = "str"
			case r == '#' || r == '.':
				kind = "fn"
			case mdCSSKeywords[w]:
				kind = "kw"
			case mdNextNonSpace(rs, j) == ':':
				kind = "attr"
			}
			add(w, kind)
			i = j
		default:
			add(string(r), "punc")
			i++
		}
	}
	return out
}

// mdScanYAML YAML / TOML / ini。
func mdScanYAML(s string) []mdHLSpan {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return []mdHLSpan{{Text: s, Kind: "txt"}}
	}
	if strings.HasPrefix(trim, "#") {
		return []mdHLSpan{{Text: s, Kind: "com"}}
	}
	rs := []rune(s)
	// key:
	if idx := strings.Index(string(rs), ":"); idx > 0 {
		pre := string(rs[:idx])
		if !strings.ContainsAny(pre, " \"'") {
			out := []mdHLSpan{{Text: pre, Kind: "attr"}, {Text: ":", Kind: "punc"}}
			rest := string(rs[idx+1:])
			out = append(out, mdScanYAMLValue(rest)...)
			return out
		}
	}
	return mdScanYAMLValue(s)
}

func mdScanYAMLValue(s string) []mdHLSpan {
	rs := []rune(s)
	var out []mdHLSpan
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '#':
			out = append(out, mdHLSpan{Text: string(rs[i:]), Kind: "com"})
			i = len(rs)
		case r == '"' || r == '\'':
			q := r
			j := i + 1
			for j < len(rs) && rs[j] != q {
				j++
			}
			if j < len(rs) {
				j++
			}
			out = append(out, mdHLSpan{Text: string(rs[i:j]), Kind: "str"})
			i = j
		case r >= '0' && r <= '9':
			j := i
			for j < len(rs) && (isIdentChar(rs[j]) || rs[j] == '.' || rs[j] == '-') {
				j++
			}
			out = append(out, mdHLSpan{Text: string(rs[i:j]), Kind: "num"})
			i = j
		case isIdentChar(r):
			j := i
			for j < len(rs) && isIdentChar(rs[j]) {
				j++
			}
			w := string(rs[i:j])
			kind := "txt"
			if w == "true" || w == "false" || w == "null" || w == "yes" || w == "no" {
				kind = "kw"
			}
			out = append(out, mdHLSpan{Text: w, Kind: kind})
			i = j
		default:
			out = append(out, mdHLSpan{Text: string(r), Kind: "punc"})
			i++
		}
	}
	return out
}
