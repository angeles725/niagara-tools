package verifymodule

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// The text model of the bash this package replaces: a shell variable is what a command substitution keeps
// (subst), a here-string loop reads its lines back (here), and grep reads a file or stream as GNU grep 3.11
// does under LANG=C.UTF-8 (grepText).

// subst is $(...): the output without its NUL bytes (bash drops them, warning on stderr) and its trailing
// newlines.
func subst(s string) string { return strings.TrimRight(strings.ReplaceAll(s, "\x00", ""), "\n") }

// here is the lines a `while read` loop reads from <<< "$v" (one empty line for an empty v).
func here(v string) []string { return strings.Split(v, "\n") }

// rx compiles an ERE with grep's and awk's leftmost-longest match.
func rx(s string) *regexp.Regexp {
	re := regexp.MustCompile(s)
	re.Longest()
	return re
}

// grepText is one grep input: its lines and whether a NUL byte makes it a binary file.
type grepText struct {
	lines  []string
	binary bool
}

func newGrepText(s string) grepText {
	t := grepText{binary: strings.IndexByte(s, 0) >= 0}
	if s != "" {
		t.lines = strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	}
	return t
}

// shown reports whether grep prints out (a matched line, or one -o match): never from a binary file and
// never text with an encoding error (grep only says "binary file matches" on stderr, which the bash
// discarded or shared with the row stream's caller).
func (t grepText) shown(out string) bool { return !t.binary && utf8.ValidString(out) }

// hits is `grep -an`: "<line>:<text>" for every line re matches, read as bytes, so a line with an encoding
// error or in a file holding a NUL byte is a hit too (grep under C.UTF-8 never printed one). The text is
// display(line).
func (t grepText) hits(re *regexp.Regexp) []string {
	var out []string
	for i, ln := range t.lines {
		if re.MatchString(ln) {
			out = append(out, itoa(i+1)+":"+display(ln))
		}
	}
	return out
}

// display is a source line as a report row quotes it: a NUL byte as \0, an invalid UTF-8 byte as U+FFFD.
func display(ln string) string {
	return strings.ToValidUTF8(strings.ReplaceAll(ln, "\x00", `\0`), "\uFFFD")
}

// only is grep -o: every printed match, line by line.
func (t grepText) only(re *regexp.Regexp) []string {
	var out []string
	for _, ln := range t.lines {
		for _, m := range re.FindAllString(ln, -1) {
			if m != "" && t.shown(m) {
				out = append(out, m)
			}
		}
	}
	return out
}

// count is grep -c: the number of lines re matches (printed or not).
func (t grepText) count(re *regexp.Regexp) int {
	n := 0
	for _, ln := range t.lines {
		if re.MatchString(ln) {
			n++
		}
	}
	return n
}

// sortU is sort -u in byte order (C.UTF-8 collation of valid UTF-8).
func sortU(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	n := 0
	for i, x := range out {
		if i == 0 || x != out[n-1] {
			out[n] = x
			n++
		}
	}
	return out[:n]
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// awkFields is awk's default field split: runs of blanks, tabs and newlines.
func awkFields(s string) []string {
	return strings.FieldsFunc(s, func(c rune) bool { return c == ' ' || c == '\t' || c == '\n' })
}

// basename is basename(1) <p> <suffix>: the last component, minus suffix unless that is all of it.
func basename(p, suffix string) string {
	b := lint.Basename(p)
	if b != suffix {
		b = strings.TrimSuffix(b, suffix)
	}
	return b
}

func itoa(n int) string { return strconv.Itoa(n) }

// intOperand is s as a coverage count: a decimal integer with an optional sign
// (a leading zero is decimal). ok is false for anything else, overflow included. Bash arithmetic read 010
// as octal and, on 08, abandoned the block (TestIntegerOperands).
func intOperand(s string) (n int64, ok bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}
