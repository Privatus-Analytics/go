package privatus

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxProps       = 30
	maxKeyLength   = 64
	maxValueLength = 500
	maxListItems   = 30
	maxEventName   = 120
)

type prop struct {
	key   string
	value any
}

// normalizeKey lowercases the key, turns any character other than a-z, 0-9
// and _ into _, and cuts it to 64 characters.
func normalizeKey(key string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if b.Len() >= maxKeyLength {
			break
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// normalizeProps cleans keys and values, drops nil values, sorts by key and
// keeps at most 30. With keepTypes, numbers and booleans stay as they are and
// lists stay lists (for JSON). Without it, every value becomes a string.
func normalizeProps(props map[string]any, keepTypes bool) []prop {
	out := []prop{}
	seen := map[string]bool{}
	for _, raw := range sortedKeys(props) {
		key := normalizeKey(raw)
		if key == "" || seen[key] {
			continue
		}
		value, ok := normalizeValue(props[raw], keepTypes)
		if !ok {
			continue
		}
		seen[key] = true
		out = append(out, prop{key, value})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].key < out[j].key })
	if len(out) > maxProps {
		out = out[:maxProps]
	}
	return out
}

func normalizeValue(v any, keepTypes bool) (any, bool) {
	rv := reflect.ValueOf(v)
	for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil, false
	}
	v = rv.Interface()

	if n, ok := v.(json.Number); ok {
		if _, err := strconv.ParseFloat(string(n), 64); keepTypes && err == nil {
			return n, true
		}
		return cut(string(n), maxValueLength), true
	}
	if s, ok := v.(fmt.Stringer); ok {
		return cut(s.String(), maxValueLength), true
	}

	switch rv.Kind() {
	case reflect.Bool:
		if keepTypes {
			return rv.Bool(), true
		}
		return strconv.FormatBool(rv.Bool()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if keepTypes {
			return rv.Int(), true
		}
		return strconv.FormatInt(rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if keepTypes {
			return rv.Uint(), true
		}
		return strconv.FormatUint(rv.Uint(), 10), true
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		bits := 64
		if rv.Kind() == reflect.Float32 {
			bits = 32
		}
		if keepTypes {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, false
			}
			return json.Number(strconv.FormatFloat(f, 'f', -1, bits)), true
		}
		return strconv.FormatFloat(f, 'f', -1, bits), true
	case reflect.String:
		return cut(rv.String(), maxValueLength), true
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return cut(string(rv.Bytes()), maxValueLength), true
		}
		items := []string{}
		for i := 0; i < rv.Len(); i++ {
			item, ok := normalizeValue(rv.Index(i).Interface(), false)
			if !ok {
				continue
			}
			items = append(items, item.(string))
		}
		if keepTypes {
			if len(items) > maxListItems {
				items = items[:maxListItems]
			}
			return items, true
		}
		return cut(strings.Join(items, ","), maxValueLength), true
	default:
		return cut(fmt.Sprint(v), maxValueLength), true
	}
}

// propsJSON returns the page properties as a JSON object that is safe inside
// a <script> element, or "" when there are none. encoding/json already
// escapes <, >, &, U+2028 and U+2029.
func propsJSON(props map[string]any) string {
	list := normalizeProps(props, true)
	if len(list) == 0 {
		return ""
	}
	m := make(map[string]any, len(list))
	for _, p := range list {
		m[p.key] = p.value
	}
	data, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(data)
}

// cut keeps the first n characters (not bytes) of s.
func cut(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
