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

// numbered is grep -n: "<line>:<text>" for every printed line re matches.
func (t grepText) numbered(re *regexp.Regexp) []string {
	var out []string
	for i, ln := range t.lines {
		if re.MatchString(ln) && t.shown(ln) {
			out = append(out, itoa(i+1)+":"+ln)
		}
	}
	return out
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

// basename is basename(1) <p> <suffix>: the last component, minus suffix unless that is all of it.
func basename(p, suffix string) string {
	b := lint.Basename(p)
	if b != suffix {
		b = strings.TrimSuffix(b, suffix)
	}
	return b
}

func itoa(n int) string { return strconv.Itoa(n) }

// arithErr is a bash arithmetic error. msg is bash's text after its "<script>: line <n>: " prefix; unbound
// is an operand naming an unset variable, which ends the script under set -u (exit 1).
type arithErr struct {
	msg     string
	unbound bool
}

// bashInt is s as an operand of bash arithmetic: an optional sign, then a decimal constant, an octal one
// (a leading 0) or a hex one (0x), wrapping on overflow as bash does; the empty string is 0.
func bashInt(s string) (int64, *arithErr) {
	num := strings.TrimLeft(s, "+-")
	neg := strings.Count(s[:len(s)-len(num)], "-")%2 == 1
	base, digits := int64(10), num
	switch {
	case num == "":
		return 0, nil
	case identRe.MatchString(num):
		return 0, &arithErr{msg: num + ": unbound variable", unbound: true}
	case strings.HasPrefix(num, "0x") || strings.HasPrefix(num, "0X"):
		base, digits = 16, num[2:]
	case num[0] == '0':
		base = 8
	}
	end := 0
	for end < len(digits) && digitValue(digits[end]) < base {
		end++
	}
	if end < len(digits) {
		if base == 8 && strings.Trim(digits, "0123456789") == "" {
			return 0, &arithErr{msg: s + ": value too great for base (error token is \"" + s + "\")"}
		}
		return 0, &arithErr{msg: s + ": syntax error: invalid arithmetic operator (error token is \"" + digits[end:] + "\")"}
	}
	var n int64
	for i := 0; i < len(digits); i++ {
		n = n*base + digitValue(digits[i])
	}
	if neg {
		n = -n
	}
	return n, nil
}

// digitValue is c as a digit of base up to 16 (36 for a byte that is no digit).
func digitValue(c byte) int64 {
	switch l := c | 0x20; {
	case '0' <= c && c <= '9':
		return int64(c - '0')
	case 'a' <= l && l <= 'f':
		return int64(l-'a') + 10
	}
	return 36
}

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
