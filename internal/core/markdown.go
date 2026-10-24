package core

import (
	"bytes"
	"strings"
)

func escaped(line string, offset int) bool {
	slashes := 0
	for i := offset - 1; i >= 0 && line[i] == '\\'; i-- {
		slashes++
	}
	return slashes%2 == 1
}
func visibleLines(body string) []string {
	lines := strings.SplitAfter(body, "\n")
	var fence codeFence
	var containers listContext
	closingTicks := nextBackticks(body)
	inlineUntil, commentUntil, offset := 0, 0, 0
	for i, line := range lines {
		content, _ := containers.content(line, fence.marker != 0)
		if fence.consume(content) {
			lines[i] = ""
			offset += len(line)
			continue
		}
		out := []byte(line)
		for j := 0; j < len(line); j++ {
			at := offset + j
			if at < inlineUntil || at < commentUntil {
				out[j] = ' '
				continue
			}
			if line[j] == '\\' {
				j++
				continue
			}
			if strings.HasPrefix(line[j:], "<!--") {
				end := strings.Index(body[at+4:], "-->")
				if end < 0 {
					commentUntil = len(body)
				} else {
					commentUntil = at + 4 + end + 3
				}
				out[j] = ' '
				continue
			}
			if end, ok := closingTicks[at]; ok {
				inlineUntil = end
				out[j] = ' '
			}
		}
		lines[i] = string(out)
		offset += len(line)
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
