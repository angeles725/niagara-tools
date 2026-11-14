package scaffoldmodule

// checkName enforces the ModuleName rule: an ASCII uppercase letter first, then ASCII letters and digits
// only (the Java identifier convention Baja type registration needs).
func checkName(name string) *failure {
	if name == "" || name[0] < 'A' || name[0] > 'Z' {
		return failf(ExitUsage, nameFirst, name)
	}
	for i := 0; i < len(name); i++ {
		if !isAlnum(name[i]) {
			return failf(ExitUsage, nameAlnum, name)
		}
	}
	return nil
}

func isAlnum(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

// asciiLower lowercases A-Z only, as `tr '[:upper:]' '[:lower:]'` does byte by byte: a non-ASCII letter
// in a vendor name is left as typed.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// lowerFirst is name with its first byte ASCII-lowercased (MinimalPan -> minimalPan).
func lowerFirst(name string) string {
	if name == "" {
		return ""
	}
	return asciiLower(name[:1]) + name[1:]
}

// initials is the palette namespace prefix: the ASCII uppercase letters of name, lowercased
// (MinimalPan -> mp).
func initials(name string) string {
	var b []byte
	for i := 0; i < len(name); i++ {
		if c := name[i]; c >= 'A' && c <= 'Z' {
			b = append(b, c+('a'-'A'))
		}
	}
	return string(b)
}
