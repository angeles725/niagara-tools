package javasrc

import "strings"

// Unescape reads the escapes a path or ORD literal uses, \\ \" \' and (JavaScript) \`, in the text between a
// literal's quotes; any other escape (\n, \t, \u0041) is kept as written. Shared by the rules that classify
// literal text (lint-machine-default-path, lint-inline-site-literal).
func Unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && strings.IndexByte("\\\"'`", s[i+1]) >= 0 {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
