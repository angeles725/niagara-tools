package pyjson

import (
	"math"
	"strconv"
	"strings"
)

// FloatRepr returns Python's repr(f): the shortest digits that round-trip, written fixed while
// the decimal point position decpt satisfies -4 < decpt <= 16 (with ".0" when there is no
// fraction), otherwise as d.ddde+XX with at least two exponent digits; "nan", "inf", "-inf".
func FloatRepr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	sign := ""
	if math.Signbit(f) {
		sign, f = "-", -f
	}
	// 'e' with precision -1 gives the shortest round-trip digits as "d.ddde±XX".
	e := strconv.FormatFloat(f, 'e', -1, 64)
	at := strings.IndexByte(e, 'e')
	digits := strings.Replace(e[:at], ".", "", 1)
	exp, _ := strconv.Atoi(e[at+1:])
	decpt := exp + 1
	if decpt > -4 && decpt <= 16 {
		return sign + fixed(digits, decpt)
	}
	mant := digits[:1]
	if len(digits) > 1 {
		mant += "." + digits[1:]
	}
	esign := "+"
	if exp < 0 {
		esign, exp = "-", -exp
	}
	es := strconv.Itoa(exp)
	if len(es) < 2 {
		es = "0" + es
	}
	return sign + mant + "e" + esign + es
}

// fixed places the decimal point decpt digits into digits, padding with zeros.
func fixed(digits string, decpt int) string {
	switch {
	case decpt <= 0:
		return "0." + strings.Repeat("0", -decpt) + digits
	case decpt >= len(digits):
		return digits + strings.Repeat("0", decpt-len(digits)) + ".0"
	default:
		return digits[:decpt] + "." + digits[decpt:]
	}
}

// floatJSON is json.dumps(f): repr, except the non-finite values Python spells as JavaScript.
func floatJSON(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return FloatRepr(f)
}
