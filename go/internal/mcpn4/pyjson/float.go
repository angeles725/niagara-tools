package pyjson

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var floatText = regexp.MustCompile(`^[+-]?(([0-9](_?[0-9])*(\.([0-9](_?[0-9])*)?)?|\.[0-9](_?[0-9])*)` +
	`([eE][+-]?[0-9](_?[0-9])*)?|(?i:inf|infinity|nan))$`)

// DecimalValue is the digit value of a decimal digit: Unicode encodes each set as a run of ten.
func DecimalValue(r rune) rune {
	start := r
	for unicode.Is(unicode.Nd, start-1) {
		start--
	}
	return (r - start) % 10
}

// ParseFloat is float(text) on a str: str.strip() whitespace around, any Unicode decimal digit
// read as its ASCII digit, then a decimal literal (single underscores between digits), inf,
// infinity or nan in any case with an optional sign. A literal past the float range is ±inf.
func ParseFloat(text string) (float64, bool) {
	var b strings.Builder
	for _, r := range strings.TrimFunc(text, IsSpace) {
		if r >= utf8.RuneSelf && unicode.Is(unicode.Nd, r) {
			r = '0' + DecimalValue(r)
		}
		b.WriteRune(r)
	}
	s := b.String()
	if !floatText.MatchString(s) {
		return 0, false
	}
	if strings.HasSuffix(strings.ToLower(s), "nan") { // Go refuses a signed NaN
		return math.NaN(), true
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, "_", ""), 64)
	if err != nil && !math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}
