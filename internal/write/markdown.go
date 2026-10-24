package write

import "strings"

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
