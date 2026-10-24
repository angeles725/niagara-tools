package bogaudit

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/htmltext"
)

// The Python value formatting and parsing the audit's rows were written with (bog-audit.sh before the port).

// pyRepr is repr(s) of a Python str: single quotes unless s holds ' and no ", the quote and \ escaped,
// \t \n \r named and other non-printable code points as \xhh, \uhhhh or \Uhhhhhhhh.
func pyRepr(s string) string {
	q := byte('\'')
	if strings.IndexByte(s, '\'') >= 0 && strings.IndexByte(s, '"') < 0 {
		q = '"'
	}
	var b strings.Builder
	b.WriteByte(q)
	for _, r := range s {
		switch {
		case r == rune(q) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case unicode.IsPrint(r):
			b.WriteRune(r)
		case r < 0x100:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r < 0x10000:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			fmt.Fprintf(&b, `\U%08x`, r)
		}
	}
	b.WriteByte(q)
	return b.String()
}

// pyFloatRepr is str(f) of a Python float: the shortest round-trip digits, positional when the decimal
// exponent is in [-4, 16) (".0" added to an integral value), else d.ddde±XX.
func pyFloatRepr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	e := strconv.FormatFloat(f, 'e', -1, 64) // [-]d[.ddd]e±XX
	sign := ""
	if e[0] == '-' {
		sign, e = "-", e[1:]
	}
	mant, exps, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(exps)
	digits := strings.Replace(mant, ".", "", 1)
	if exp < -4 || exp >= 16 {
		m := digits[:1]
		if len(digits) > 1 {
			m += "." + digits[1:]
		}
		es := "+"
		if exp < 0 {
			es, exp = "-", -exp
		}
		return fmt.Sprintf("%s%se%s%02d", sign, m, es, exp)
	}
	if exp < 0 {
		return sign + "0." + strings.Repeat("0", -exp-1) + digits
	}
	if len(digits) <= exp+1 {
		return sign + digits + strings.Repeat("0", exp+1-len(digits)) + ".0"
	}
	return sign + digits[:exp+1] + "." + digits[exp+1:]
}

// pyFloatSyntax is the grammar Python's float() accepts after strip(): decimal digits with single "_"
// between them, an optional exponent, or inf/infinity/nan, each with an optional sign. Hex is refused.
var pyFloatSyntax = regexp.MustCompile(`(?i)^[+-]?(inf|infinity|nan|` +
	`([0-9](_?[0-9])*(\.([0-9](_?[0-9])*)?)?|\.[0-9](_?[0-9])*)(e[+-]?[0-9](_?[0-9])*)?)$`)

// pyFloat is float(s) of a Python str: ok false where Python raises ValueError. An out-of-range value is
// ±inf, as float() returns it.
func pyFloat(s string) (float64, bool) {
	s = htmltext.Strip(s)
	if !pyFloatSyntax.MatchString(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, "_", ""), 64)
	if err != nil && !math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// pyInt is str(int(d)) of an ASCII digit run: the leading zeros dropped, so two runs are equal ints when
// their pyInt strings are equal.
func pyInt(d string) string {
	if t := strings.TrimLeft(d, "0"); t != "" {
		return t
	}
	return "0"
}

// intLess orders two pyInt strings as numbers.
func intLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
