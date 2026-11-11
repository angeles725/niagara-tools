package writeops

import (
	"strconv"
	"strings"
)

// ResolveHistoryName evaluates the subset of a baja:Format history name that the ORD alone
// answers: literals, %name% and %parent.name%, %parent.parent.name%, ... of the extension at
// extOrd (a station slot ORD). Escaped slot names ($32004) are unescaped (2004). Any other
// token, a path above the root or an unterminated % gives ok=false: the caller then cannot
// check the name for repeats.
func ResolveHistoryName(pattern, extOrd string) (string, bool) {
	i := strings.Index(extOrd, "slot:")
	if i < 0 {
		return "", false
	}
	var names []string
	for _, seg := range strings.Split(extOrd[i+len("slot:"):], "/") {
		if seg != "" {
			names = append(names, unescapeSlot(seg))
		}
	}
	var b strings.Builder
	for rest := pattern; rest != ""; {
		open := strings.IndexByte(rest, '%')
		if open < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:open])
		end := strings.IndexByte(rest[open+1:], '%')
		if end < 0 {
			return "", false
		}
		token := rest[open+1 : open+1+end]
		rest = rest[open+2+end:]
		up := 0
		for strings.HasPrefix(token, "parent.") {
			token, up = token[len("parent."):], up+1
		}
		k := len(names) - 1 - up
		if token != "name" || k < 0 {
			return "", false
		}
		b.WriteString(names[k])
	}
	return b.String(), true
}

// unescapeSlot undoes SlotPath escaping: $xx is the byte with hex value xx.
func unescapeSlot(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '$' && i+2 < len(s) {
			if v, err := strconv.ParseUint(s[i+1:i+3], 16, 8); err == nil {
				b.WriteByte(byte(v))
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
