package write

import "strings"

type listContext struct {
	indents    []int
	quoteDepth int
}

func leadingIndent(line string) (bytes, columns int) {
	return indentAt(line, 0)
}

func indentAt(line string, startColumn int) (bytes, columns int) {
	for bytes < len(line) {
		switch line[bytes] {
		case ' ':
			columns++
		case '\t':
			columns += 4 - (startColumn+columns)%4
		default:
			return bytes, columns
		}
		bytes++
	}
	return bytes, columns
}

func listMarker(line string) int {
	if len(line) < 2 {
		return 0
	}
	width := 0
	if strings.ContainsRune("-+*", rune(line[0])) {
		width = 1
	} else {
		for width < len(line) && width < 9 && line[width] >= '0' && line[width] <= '9' {
			width++
		}
		if width == 0 || width >= len(line) || (line[width] != '.' && line[width] != ')') {
			return 0
		}
		width++
	}
	if width >= len(line) || (line[width] != ' ' && line[width] != '\t') {
		return 0
	}
	return width
}

// content removes list/quote containers for block classification, retaining offsets.
func (c *listContext) content(line string, inFence bool) (string, int) {
	start, depth := 0, 0
	for start < len(line) {
		spaces := 0
		for start+spaces < len(line) && spaces < 3 && line[start+spaces] == ' ' {
			spaces++
		}
		at := start + spaces
		if at >= len(line) || line[at] != '>' {
			break
		}
		start = at + 1
		if start < len(line) && line[start] == ' ' {
			start++
		}
		depth++
	}
	if depth != c.quoteDepth {
		c.indents = nil
		c.quoteDepth = depth
	}
	line = line[start:]
	bytes, indent := leadingIndent(line)
	if strings.TrimSpace(line) == "" {
		return line, start + bytes
	}
	for len(c.indents) > 0 && indent < c.indents[len(c.indents)-1] {
		c.indents = c.indents[:len(c.indents)-1]
	}
	base := 0
	if len(c.indents) > 0 {
		base = c.indents[len(c.indents)-1]
	}
	if width := listMarker(line[bytes:]); !inFence && width > 0 && indent-base < 4 {
		paddingBytes, paddingColumns := indentAt(line[bytes+width:], indent+width)
		// Five or more padding columns leave an indented code block after one space.
		if paddingColumns > 4 {
			paddingBytes = 1
			paddingColumns = 1
		}
		c.indents = append(c.indents, indent+width+paddingColumns)
		contentStart := bytes + width + paddingBytes
		return line[contentStart:], start + contentStart
	}
	return strings.Repeat(" ", indent-base) + line[bytes:], start + bytes
}

type codeFence struct {
	marker byte
	length int
}

// consume includes fence delimiters and indented code in the ignored region.
func (f *codeFence) consume(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	indent := 0
indentation:
	for _, c := range line {
		switch c {
		case ' ':
			indent++
		case '\t':
			indent += 4 - indent%4
		default:
			break indentation
		}
		if indent >= 4 {
			return true
		}
	}
	n := 0
	if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
		for n < len(trimmed) && trimmed[n] == trimmed[0] {
			n++
		}
	}
	if f.marker != 0 {
		if n >= f.length && trimmed[0] == f.marker && strings.Trim(trimmed[n:], " \t\r\n") == "" {
			f.marker = 0
			f.length = 0
		}
		return true
	}
	if n >= 3 && (trimmed[0] != '`' || !strings.Contains(trimmed[n:], "`")) {
		f.marker = trimmed[0]
		f.length = n
		return true
	}
	return false
}

// nextBackticks records exact-length matches without interpreting their contents.
func nextBackticks(text string) map[int]int {
	type run struct{ start, length int }
	var runs []run
	for i := 0; i < len(text); i++ {
		if text[i] != '`' {
			continue
		}
		start := i
		for i < len(text) && text[i] == '`' {
			i++
		}
		runs = append(runs, run{start, i - start})
		i--
	}
	next, byLength := map[int]int{}, map[int]int{}
	for i := len(runs) - 1; i >= 0; i-- {
		r := runs[i]
		if close, ok := byLength[r.length]; ok {
			next[r.start] = close + r.length
		}
		byLength[r.length] = r.start
	}
	return next
}

// indexRegion recognizes standalone comment markers outside code examples.
func indexRegion(text string) (start, end int, appendSafe bool, err error) {
	start, end = -1, -1
	var fence codeFence
	closingTicks := nextBackticks(text)
	inlineUntil := 0
	commentUntil := 0
	offset := 0
	frontmatter := strings.HasPrefix(text, "---\n") || strings.HasPrefix(text, "---\r\n")
	for _, line := range strings.SplitAfter(text, "\n") {
		if frontmatter {
			if offset > 0 && strings.TrimSpace(line) == "---" {
				frontmatter = false
			}
			offset += len(line)
			continue
		}
		if fence.consume(line) {
			offset += len(line)
			continue
		}
		trimmed := strings.Trim(line, " \t\r\n")
		if inlineUntil <= offset && commentUntil <= offset {
			markerOffset := offset + len(line) - len(strings.TrimLeft(line, " \t"))
			switch trimmed {
			case indexStart:
				if start >= 0 || end >= 0 {
					return start, end, false, problem("INVALID_INDEX", "index marker region is malformed")
				}
				start = markerOffset
			case indexEnd:
				if start < 0 || end >= 0 {
					return start, end, false, problem("INVALID_INDEX", "index marker region is malformed")
				}
				end = markerOffset
			}
		}
		for i := 0; i < len(line); i++ {
			if offset+i < inlineUntil || offset+i < commentUntil {
				continue
			}
			if line[i] == '\\' {
				i++
				continue
			}
			if strings.HasPrefix(line[i:], "<!--") {
				close := strings.Index(text[offset+i+4:], "-->")
				if close < 0 {
					commentUntil = len(text) + 1
				} else {
					commentUntil = offset + i + 4 + close + 3
				}
				continue
			}
			if close, ok := closingTicks[offset+i]; ok {
				inlineUntil = close
			}
		}
		offset += len(line)
	}
	if (start < 0) != (end < 0) {
		return start, end, false, problem("INVALID_INDEX", "index marker region is malformed")
	}
	return start, end, fence.marker == 0 && !frontmatter && commentUntil <= len(text), nil
}
