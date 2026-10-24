package xmlsrc

import (
	"strconv"
	"strings"
	"unicode/utf16"
)

// UnescapeName decodes a Niagara SlotPath-escaped slot name as a .bog writes it in n="...":
// "$20" -> " ", "$u00e9" -> "é", a UTF-16 surrogate pair "$ud83d$ude00" -> one rune. A '$' that starts neither form is kept. Ported from niagara-go
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
		if r, ok := hex4(name, i); ok {
			i += 6
			if lo, ok := hex4(name, i); ok && utf16.IsSurrogate(r) {
				if pair := utf16.DecodeRune(r, lo); pair != '\uFFFD' {
					r, i = pair, i+6
				}
			}
			sb.WriteRune(r) // a lone surrogate is written as U+FFFD
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

// hex4 reads the "$uXXXX" form at name[i].
func hex4(name string, i int) (rune, bool) {
	if i+6 > len(name) || name[i] != '$' || name[i+1] != 'u' || !isHex(name[i+2:i+6]) {
		return 0, false
	}
	v, _ := strconv.ParseUint(name[i+2:i+6], 16, 32)
	return rune(v), true
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
