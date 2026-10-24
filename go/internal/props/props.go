// Package props reads a .properties file the way java.util.Properties.load(Reader) does, keeping the line
// each entry starts on: logical lines joined over an odd count of trailing backslashes (the continuation's
// leading whitespace dropped), # and ! comment lines, the key ending at the first unescaped =, : or
// whitespace, and the \t \n \r \f \uXXXX escapes (any other \c is c). A lint reads the values it classifies
// through it, never through a line regex.
package props

import (
	"strconv"
	"strings"
)

// Entry is one key/value pair; Line is the 0-based index of the line its logical line starts on.
type Entry struct {
	Line       int
	Key, Value string
}

const space = " \t\f"

// Read parses lines (as javasrc.Lines splits them: a trailing "\r" is a line terminator and is dropped).
func Read(lines []string) []Entry {
	var out []Entry
	for i := 0; i < len(lines); i++ {
		start := i
		ln := strings.TrimLeft(strings.TrimSuffix(lines[i], "\r"), space)
		if ln == "" || ln[0] == '#' || ln[0] == '!' {
			continue // a comment line never continues
		}
		for continues(ln) && i+1 < len(lines) {
			i++
			ln = ln[:len(ln)-1] + strings.TrimLeft(strings.TrimSuffix(lines[i], "\r"), space)
		}
		if continues(ln) {
			ln = ln[:len(ln)-1] // a continuation at end of input is dropped, as load does
		}
		key, value := split(ln)
		out = append(out, Entry{Line: start, Key: unescape(key), Value: unescape(value)})
	}
	return out
}

// continues reports a line ending in an odd number of backslashes.
func continues(ln string) bool {
	n := len(ln) - len(strings.TrimRight(ln, `\`))
	return n%2 == 1
}

// split cuts a logical line into its raw key and raw value.
func split(ln string) (key, value string) {
	k := 0
	for k < len(ln) && !strings.ContainsRune("=:"+space, rune(ln[k])) {
		if ln[k] == '\\' {
			k++
		}
		k++
	}
	k = min(k, len(ln))
	rest := strings.TrimLeft(ln[k:], space)
	if rest != "" && (rest[0] == '=' || rest[0] == ':') {
		rest = strings.TrimLeft(rest[1:], space)
	}
	return ln[:k], rest
}

// unescape applies the load escapes.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 == len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch c = s[i]; c {
		case 't':
			b.WriteByte('\t')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 'f':
			b.WriteByte('\f')
		case 'u':
			if r, err := strconv.ParseUint(s[i+1:min(i+5, len(s))], 16, 32); err == nil && i+5 <= len(s) {
				b.WriteRune(rune(r))
				i += 4
			} else {
				b.WriteByte('u') // load throws on a malformed \u; a lint keeps reading
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// buildLocationKeys are the host build-location keys (R10-D8): where Niagara, its user home, Node and the
// JDK live on the building machine. They belong in the user-level gradle.properties
// ($GRADLE_USER_HOME/gradle.properties, ~/.gradle by default), never in a repository; lint-structure L10
// owns them and lint-machine-default-path leaves them to it.
var buildLocationKeys = []string{"niagara_home", "niagara_user_home", "nodeHome", "org.gradle.java.installations.paths"}

// BuildLocationKey reports whether key is one of the host build-location keys.
func BuildLocationKey(key string) bool {
	for _, k := range buildLocationKeys {
		if k == key {
			return true
		}
	}
	return false
}

// BuildLocationKeys returns a copy of the host build-location keys, in their documented order.
func BuildLocationKeys() []string { return append([]string(nil), buildLocationKeys...) }
