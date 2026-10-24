package core

import (
	"bytes"
	"strings"
)

// fenceState tracks the delimiter and minimum closing width of a code fence.
type fenceState struct {
	mark  byte
	width int
}

func (f *fenceState) skip(line string) bool {
	trim := strings.TrimLeft(line, " ")
	indent := len(line) - len(trim)
	if indent >= 4 && f.width == 0 {
		return true
	}
	if len(trim) == 0 {
		return f.width > 0
	}
	mark := trim[0]
	width := 0
	if mark == '`' || mark == '~' {
		for width < len(trim) && trim[width] == mark {
			width++
		}
	}
	if f.width > 0 {
		if mark == f.mark && width >= f.width && strings.TrimSpace(trim[width:]) == "" {
			f.width = 0
		}
		return true
	}
	if indent <= 3 && width >= 3 {
		if mark == '`' && strings.ContainsRune(trim[width:], '`') {
			return false
		}
		f.mark = mark
		f.width = width
		return true
	}
	return false
}
func escaped(line string, offset int) bool {
	slashes := 0
	for i := offset - 1; i >= 0 && line[i] == '\\'; i-- {
		slashes++
	}
	return slashes%2 == 1
}
func maskInlineCode(line string) string {
	out := []byte(line)
	for i := 0; i < len(line); {
		if line[i] != '`' || escaped(line, i) {
			i++
			continue
		}
		width := 1
		for i+width < len(line) && line[i+width] == '`' {
			width++
		}
		end := -1
		for j := i + width; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			run := 1
			for j+run < len(line) && line[j+run] == '`' {
				run++
			}
			if run == width {
				end = j + run
				break
			}
			j += run
		}
		if end < 0 {
			i += width
			continue
		}
		for j := i; j < end; j++ {
			out[j] = ' '
		}
		i = end
	}
	return string(out)
}
func visibleLines(body string) []string {
	lines := strings.Split(body, "\n")
	var fence fenceState
	comment := false
	for i, line := range lines {
		if fence.skip(line) {
			lines[i] = ""
			continue
		}
		line = maskInlineCode(line)
		out := []byte(line)
		for j := 0; j < len(line); {
			if comment {
				end := strings.Index(line[j:], "-->")
				if end < 0 {
					for k := j; k < len(out); k++ {
						out[k] = ' '
					}
					break
				}
				end += j + 3
				for k := j; k < end; k++ {
					out[k] = ' '
				}
				j = end
				comment = false
				continue
			}
			start := strings.Index(line[j:], "<!--")
			if start < 0 {
				break
			}
			start += j
			comment = true
			j = start
		}
		lines[i] = string(out)
	}
	return lines
}

type markdownTarget struct{ text, target string }

func inlineTargets(line string) []markdownTarget {
	var targets []markdownTarget
	for i := 0; i < len(line); i++ {
		if line[i] != '[' || escaped(line, i) {
			continue
		}
		labelStart := i + 1
		j := labelStart
		depth := 1
		for j < len(line) && depth > 0 {
			if escaped(line, j) {
				j++
				continue
			}
			if line[j] == '[' {
				depth++
			}
			if line[j] == ']' {
				depth--
			}
			j++
		}
		if depth != 0 || j >= len(line) || line[j] != '(' {
			continue
		}
		labelEnd := j - 1
		j++
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		start := j
		if j < len(line) && line[j] == '<' {
			j++
			start = j
			for j < len(line) && (line[j] != '>' || escaped(line, j)) {
				j++
			}
			if j == len(line) {
				continue
			}
			target := line[start:j]
			j++
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				j++
			}
			if j < len(line) && line[j] == ')' {
				targets = append(targets, markdownTarget{line[labelStart:labelEnd], unescapeDestination(target)})
				i = j
			}
			continue
		}
		depth = 0
		for j < len(line) {
			if escaped(line, j) {
				j++
				continue
			}
			if line[j] == '(' {
				depth++
				if depth > 32 {
					break
				}
			}
			if line[j] == ')' {
				if depth == 0 {
					break
				}
				depth--
			}
			if depth == 0 && (line[j] == ' ' || line[j] == '\t') {
				break
			}
			j++
		}
		if j >= len(line) || depth != 0 || j == start {
			continue
		}
		target := line[start:j]
		if line[j] != ')' {
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				j++
			}
			if j >= len(line) {
				continue
			}
			quote := line[j]
			if quote != '\'' && quote != '"' && quote != '(' {
				continue
			}
			close := quote
			if quote == '(' {
				close = ')'
			}
			j++
			for j < len(line) && (line[j] != close || escaped(line, j)) {
				j++
			}
			if j >= len(line) {
				continue
			}
			j++
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				j++
			}
			if j >= len(line) || line[j] != ')' {
				continue
			}
		}
		targets = append(targets, markdownTarget{line[labelStart:labelEnd], unescapeDestination(target)})
		i = j
	}
	return targets
}
func unescapeDestination(target string) string {
	var out strings.Builder
	for i := 0; i < len(target); i++ {
		if target[i] == '\\' && i+1 < len(target) && strings.ContainsRune(`!"#$%&'()*+,-./:;<=>?@[\]^_`+"`"+`{|}~`, rune(target[i+1])) {
			i++
		}
		out.WriteByte(target[i])
	}
	return out.String()
}
func rawBody(raw []byte) ([]byte, int) {
	newline := []byte("\n")
	if bytes.HasPrefix(raw, []byte("---\r\n")) {
		newline = []byte("\r\n")
	}
	opening := append([]byte("---"), newline...)
	if !bytes.HasPrefix(raw, opening) {
		return raw, 0
	}
	offset := len(opening)
	for offset < len(raw) {
		end := bytes.IndexByte(raw[offset:], '\n')
		next := len(raw)
		if end >= 0 {
			next = offset + end + 1
		}
		line := bytes.TrimSuffix(bytes.TrimSuffix(raw[offset:next], []byte("\n")), []byte("\r"))
		if bytes.Equal(line, []byte("---")) {
			return raw[next:], bytes.Count(raw[:next], []byte("\n"))
		}
		offset = next
	}
	return raw, 0
}
