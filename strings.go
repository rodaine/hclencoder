package hclencoder

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// EscapeString escapes a string so that it can be used in HCL. It doesn't escape anything inside a template
// expression (${...})
func EscapeString(s string) string {
	var result []byte
	stack := []string{""}
	escapeNext := false
	for i, c := range s {
		top := stack[len(stack)-1]
		switch top {
		case "":
			if c == '$' && (i == 0 || s[i-1] != '$') && (i < len(s)-1 && s[i+1] == '{') {
				stack = append(stack, "${")
			}
			result = escapeAndAppend(result, c, true)
		case "${":
			if c == '"' {
				stack = append(stack, "\"")
			}
			if c == '}' {
				stack = stack[:len(stack)-1]
			}
			result = utf8.AppendRune(result, c)
		case "\"":
			if c == '$' && (i == 0 || s[i-1] != '$') && (i < len(s)-1 && s[i+1] == '{') {
				stack = append(stack, "${")
				escapeNext = false
			}
			if c == '"' && !escapeNext {
				stack = stack[:len(stack)-1]
			}
			if c == '\\' {
				escapeNext = !escapeNext
			}
			result = utf8.AppendRune(result, c)
		default:
			panic(fmt.Errorf("unexpected stack entry: %s", top))
		}
	}
	return string(result)
}

func escapeAndAppend(buf []byte, r rune, escapeQuote bool) []byte {
	switch r {
	case '\n':
		buf = append(buf, '\\', 'n')
	case '\r':
		buf = append(buf, '\\', 'r')
	case '\t':
		buf = append(buf, '\\', 't')
	case '"':
		if escapeQuote {
			buf = append(buf, '\\', '"')
		} else {
			buf = append(buf, '"')
		}
	case '\\':
		buf = append(buf, '\\', '\\')
	default:
		if !unicode.IsPrint(r) {
			if r < 65536 {
				buf = append(buf, fmt.Sprintf("\\u%04x", r)...)
			} else {
				buf = append(buf, fmt.Sprintf("\\u%08x", r)...)
			}
		} else {
			buf = utf8.AppendRune(buf, r)
		}
	}
	return buf
}
