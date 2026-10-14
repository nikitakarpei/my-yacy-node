package yacyproto

import (
	"slices"
	"strings"
)

// ResponseFields is the body of a peer response: newline-separated key=value lines,
// with backslash escapes inside either part.
type ResponseFields map[string]string

// ParseResponseFields skips blank lines, comments and lines without an unescaped
// separator. A peer cannot make this fail.
func ParseResponseFields(data string) ResponseFields {
	fields := make(ResponseFields)
	for line := range strings.SplitSeq(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		pos := unescapedEquals(line)
		if pos <= 0 {
			continue
		}
		key := unescapeResponseFieldPart(strings.TrimSpace(line[:pos]))
		value := unescapeResponseFieldPart(strings.TrimSpace(line[pos+1:]))
		fields[key] = value
	}
	return fields
}

func unescapedEquals(line string) int {
	pos := 0
	for {
		next := strings.IndexByte(line[pos+1:], '=')
		if next < 0 {
			return -1
		}
		pos += next + 1
		if line[pos-1] != '\\' {
			return pos
		}
	}
}

func unescapeResponseFieldPart(s string) string {
	replacer := strings.NewReplacer(
		`\\`, `\`,
		`\n`, "\n",
		`\=`, `=`,
	)
	return replacer.Replace(s)
}

func (m ResponseFields) Encode() string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(m[k])
		b.WriteByte('\n')
	}
	return b.String()
}
