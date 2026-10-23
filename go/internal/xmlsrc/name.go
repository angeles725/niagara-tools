package xmlsrc

import (
	"strconv"
	"strings"
)

// UnescapeName decodes a Niagara SlotPath-escaped slot name as a .bog writes it in n="...":
// "$20" -> " ", "$u00e9" -> "é". A '$' that starts neither form is kept. Ported from niagara-go
// internal/bog/scan.go UnescapeName; the XML entities are already decoded by Parse.
func UnescapeName(name string) string {
	i := strings.IndexByte(name, '$')
	if i < 0 {
		return name
	}
	var sb strings.Builder
	sb.Grow(len(name))
	sb.WriteString(name[:i])
	for i < len(name) {
		if name[i] != '$' {
			sb.WriteByte(name[i])
			i++
			continue
		}
		if i+6 <= len(name) && name[i+1] == 'u' && isHex(name[i+2:i+6]) {
			v, _ := strconv.ParseUint(name[i+2:i+6], 16, 32)
			sb.WriteRune(rune(v))
			i += 6
			continue
		}
		if i+3 <= len(name) && isHex(name[i+1:i+3]) {
			v, _ := strconv.ParseUint(name[i+1:i+3], 16, 32)
			sb.WriteRune(rune(v))
			i += 3
			continue
		}
		sb.WriteByte('$')
		i++
	}
	return sb.String()
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F') {
			return false
		}
	}
	return true
}
