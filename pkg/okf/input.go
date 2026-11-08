package okf

import (
	"encoding/json"
	"unicode/utf8"
)

// Check caller-created trees before recursive equality/encoding. A depth bound
// also terminates cycles without retaining pointers to caller maps or slices.
// Unchanged parsed scalars need no rendering budget: aliases can expand their
// detached view beyond the size of the supported source document.
func checkInput(value, old any) error {
	count, size := 0, 0
	var visit func(any, any, int) error
	visit = func(value, old any, depth int) error {
		count++
		if depth > 32 || count > 50000 {
			return fault("YAML_LIMIT", "YAML nesting or node limit exceeded")
		}
		switch v := value.(type) {
		case map[string]any:
			previous, _ := old.(map[string]any)
			for key, child := range v {
				var oldKey any
				if _, exists := previous[key]; exists {
					oldKey = key
				}
				if err := visit(key, oldKey, depth+1); err != nil {
					return err
				}
				if err := visit(child, previous[key], depth+1); err != nil {
					return err
				}
			}
		case []any:
			previous, _ := old.([]any)
			for i, child := range v {
				var before any
				if i < len(previous) {
					before = previous[i]
				}
				if err := visit(child, before, depth+1); err != nil {
					return err
				}
			}
		case json.Number:
			previous, unchanged := old.(json.Number)
			unchanged = unchanged && v == previous
			if !unchanged && len(v) > MaxMetadataBytes-size {
				return fault("LIMIT_EXCEEDED", "metadata exceeds 4 MiB processing budget")
			}
			if !validNumber(v) {
				return fault("INVALID_INPUT", "invalid JSON number")
			}
			if !unchanged {
				size += len(v)
			}
		case string:
			if previous, ok := old.(string); ok && v == previous {
				return nil
			}
			if len(v) > MaxMetadataBytes-size {
				return fault("LIMIT_EXCEEDED", "metadata exceeds 4 MiB processing budget")
			}
			if !utf8.ValidString(v) {
				return fault("INVALID_UTF8", "metadata is not valid UTF-8")
			}
			size += len(v)
		case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		default:
			return fault("INVALID_INPUT", "metadata must contain string-keyed maps, lists and scalar values")
		}
		return nil
	}
	return visit(value, old, 0)
}

func validNumber(value json.Number) bool {
	if value == "" {
		return false
	}
	_, err := json.Marshal(value)
	return err == nil
}
