package verifymodule

// verLE is the bash ver_le: `printf '%s\n%s\n' a b | sort -V | head -1` is a. sort -V orders by gnulib
// filevercmp and breaks a tie by byte order (no -s), so a equal-ranked b that sorts first bytewise wins.
func verLE(a, b string) bool {
	c := filevercmp(a, b)
	return c < 0 || (c == 0 && a <= b)
}

// filevercmp is gnulib's filenvercmp (coreutils 9.4, sort -V): "" first, then "." and ".." and other
// dot names, then a version compare of the names without their file suffixes, and of the whole names
// when that ties.
func filevercmp(a, b string) int {
	switch {
	case a == "" || b == "":
		return boolInt(b == "") - boolInt(a == "")
	case a[0] == '.' && b[0] != '.':
		return -1
	case a[0] != '.' && b[0] == '.':
		return 1
	case a[0] == '.':
		for _, special := range []string{".", ".."} {
			if a == special || b == special {
				return boolInt(b == special) - boolInt(a == special)
			}
		}
	}
	ap, bp := prefixLen(a), prefixLen(b)
	if r := verrevcmp(a[:ap], b[:bp]); r != 0 || (ap == len(a) && bp == len(b)) {
		return r
	}
	return verrevcmp(a, b)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// prefixLen is the length of s without its longest suffix matching (\.[A-Za-z~][A-Za-z0-9~]*)*$.
func prefixLen(s string) int {
	prefix := 0
	for i := 0; i < len(s); {
		i++
		prefix = i
		for i+1 < len(s) && s[i] == '.' && (isAlpha(s[i+1]) || s[i+1] == '~') {
			for i += 2; i < len(s) && (isAlpha(s[i]) || isDigit(s[i]) || s[i] == '~'); i++ {
			}
		}
	}
	return prefix
}

// order is the rank of s's byte at pos: the end sorts before everything but '~', a digit is 0, a letter
// itself, '~' first of all, any other byte after every letter.
func order(s string, pos int) int {
	if pos == len(s) {
		return -1
	}
	switch c := s[pos]; {
	case isDigit(c):
		return 0
	case isAlpha(c):
		return int(c)
	case c == '~':
		return -2
	default:
		return int(c) + 256
	}
}

// verrevcmp is dpkg's version compare as gnulib keeps it: non-digit runs by order, digit runs by value.
func verrevcmp(a, b string) int {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		firstDiff := 0
		for (i < len(a) && !isDigit(a[i])) || (j < len(b) && !isDigit(b[j])) {
			if ac, bc := order(a, i), order(b, j); ac != bc {
				return ac - bc
			}
			i, j = i+1, j+1
		}
		for i < len(a) && a[i] == '0' {
			i++
		}
		for j < len(b) && b[j] == '0' {
			j++
		}
		for i < len(a) && j < len(b) && isDigit(a[i]) && isDigit(b[j]) {
			if firstDiff == 0 {
				firstDiff = int(a[i]) - int(b[j])
			}
			i, j = i+1, j+1
		}
		if i < len(a) && isDigit(a[i]) {
			return 1
		}
		if j < len(b) && isDigit(b[j]) {
			return -1
		}
		if firstDiff != 0 {
			return firstDiff
		}
	}
	return 0
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }
func isAlpha(c byte) bool { return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') }
