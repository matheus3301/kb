package setup

import "bytes"

// maskJSONC keeps byte offsets and newlines stable. Strings are copied verbatim;
// only comment bytes and optional trailing commas become spaces.
func maskJSONC(raw []byte, trailing bool) ([]byte, error) {
	out := append([]byte(nil), raw...)
	quoted := false
	for i := 0; i < len(out); i++ {
		if quoted {
			if out[i] == '\\' {
				i++
				continue
			}
			if out[i] == '"' {
				quoted = false
			}
			continue
		}
		if out[i] == '"' {
			quoted = true
			continue
		}
		if out[i] != '/' || i+1 >= len(out) {
			continue
		}
		if out[i+1] == '/' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			for i < len(out) && out[i] != '\n' && out[i] != '\r' {
				out[i] = ' '
				i++
			}
			i--
			continue
		}
		if out[i+1] == '*' {
			out[i] = ' '
			out[i+1] = ' '
			i += 2
			closed := false
			for i < len(out) {
				if i+1 < len(out) && out[i] == '*' && out[i+1] == '/' {
					out[i] = ' '
					out[i+1] = ' '
					i++
					closed = true
					break
				}
				if out[i] != '\n' && out[i] != '\r' {
					out[i] = ' '
				}
				i++
			}
			if !closed {
				return nil, conflict("unterminated JSONC comment")
			}
		}
	}
	if trailing {
		quoted = false
		for i := 0; i < len(out); i++ {
			if quoted {
				if out[i] == '\\' {
					i++
					continue
				}
				if out[i] == '"' {
					quoted = false
				}
				continue
			}
			if out[i] == '"' {
				quoted = true
				continue
			}
			if out[i] == ',' {
				j := i + 1
				for j < len(out) && bytes.ContainsRune([]byte(" \r\n\t"), rune(out[j])) {
					j++
				}
				if j < len(out) && (out[j] == '}' || out[j] == ']') {
					out[i] = ' '
				}
			}
		}
	}
	return out, nil
}
func parseJSONFormat(raw []byte, kind string) (*jsonNode, error) {
	if kind != "jsonc" && kind != "jsonc-array" {
		return parseJSON(raw)
	}
	masked, err := maskJSONC(raw, true)
	if err != nil {
		return nil, err
	}
	return parseJSON(masked)
}
func commaOffset(raw []byte, start, end int) int {
	masked, err := maskJSONC(raw[start:end], false)
	if err != nil {
		return -1
	}
	return bytes.IndexByte(masked, ',')
}
func trailingComma(raw []byte, n *jsonNode) bool {
	start := n.start + 1
	if len(n.members) > 0 {
		start = n.members[len(n.members)-1].value.end
	} else if len(n.items) > 0 {
		start = n.items[len(n.items)-1].end
	}
	return commaOffset(raw, start, n.end-1) >= 0
}
