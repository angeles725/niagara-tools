package rcscan

import "strings"

// mentionsGate reports a login / auth / step-up gate mention: `auth` at a word or camelCase start (auth,
// Auth, authToken, getAuth) or after the gate prefixes o / re / un (OAuth, reauth, unauthorized), and not
// author / authority (authorize / authorise still count).
func mentionsGate(line string) bool {
	low := asciiLower(line)
	if strings.Contains(low, "login") || strings.Contains(low, "stepup") || strings.Contains(low, "step-up") {
		return true
	}
	for t := line; strings.Contains(low, "auth"); {
		m := authWord.FindStringIndex(t)
		if m == nil {
			return false
		}
		i := m[0]
		var pre byte
		pre2 := ""
		if i > 0 {
			pre = t[i-1]
		}
		if i > 1 {
			pre2 = asciiLower(t[i-2 : i])
		}
		nxt := asciiLower(t[i+4 : min(i+8, len(t))])
		upper := t[i] == 'A'
		wordStart := !alnum(pre) || upper && (lowerOrDigit(pre))
		prefix := pre == 'o' || pre == 'O' || pre2 == "re" || pre2 == "un"
		if (wordStart || prefix) && (!strings.HasPrefix(nxt, "or") || strings.HasPrefix(nxt, "oris") || strings.HasPrefix(nxt, "oriz")) {
			return true
		}
		t = t[i+4:]
	}
	return false
}

func alnum(c byte) bool        { return lowerOrDigit(c) || 'A' <= c && c <= 'Z' }
func lowerOrDigit(c byte) bool { return 'a' <= c && c <= 'z' || '0' <= c && c <= '9' }

// asciiLower is gawk's tolower under LC_ALL=C: ASCII letters only.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
