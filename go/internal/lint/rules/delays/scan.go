package delays

import (
	"strconv"
	"strings"
)

// maxCont is the continuation-line window of one Clock.schedule*( call.
const maxCont = 12

var (
	relLiteral = re(`BRelTime\.make(Seconds|Minutes|Hours|Days)?\([0-9]+\)`)
	facetMin   = re(`BFacets\.MIN,[[:space:]]*BRelTime\.make(Seconds|Minutes|Hours|Days)?\([0-9]+\)`)
	constDecl  = re(`static[[:space:]]+final[[:space:]]+BRelTime[[:space:]]+`)
	propDecl   = re(`static[[:space:]]+final[[:space:]]+Property[[:space:]]+`)
	slotName   = re(`^[a-z][A-Za-z0-9_]*`)
	propName   = re(`name[[:space:]]*=[[:space:]]*"[^"]*"`)
	getterDecl = re(`public[[:space:]]+BRelTime[[:space:]]+get[A-Za-z0-9_]+[[:space:]]*\(`)
	getterMods = re(`public[[:space:]]+BRelTime[[:space:]]+`)
	getterName = re(`^get[A-Za-z0-9_]+`)
	getQuoted  = re(`get\("[^"]+"\)`)
	getField   = re(`get\([a-z][A-Za-z0-9_]*\)`)
	callSite   = re(`Clock\.schedule(Periodically)?\(`)
	maxInMake  = re(`BRelTime\.make\(Math\.max\(`)
	makeOpen   = re(`^BRelTime\.make\(`)
	qualCall   = re(`^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	bareCall   = re(`^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	callTail   = re(`[[:space:]]*\($`)
	makeVar    = re(`BRelTime\.make\([^)]+\)`)
	maxBinding = re(`Math\.max\([^,]+,[[:space:]]*`)
	digits     = re(`^[0-9]+`)
	assign     = re(`^[[:space:]]*=[^=]`)
	getterCall = re(`^get[A-Za-z0-9_]+[[:space:]]*\([[:space:]]*\)`)
	getterArgs = re(`[[:space:]]*\(.*`)
)

// unitMs is the milliseconds per unit of a BRelTime.make<Unit>( factory ("" is make(ms)).
func unitMs(u string) float64 {
	switch u {
	case "Seconds":
		return 1000
	case "Minutes":
		return 60000
	case "Hours":
		return 3600000
	case "Days":
		return 86400000
	}
	return 1
}

// relMs reads one BRelTime.make<Unit>(N) match: its value in ms.
func relMs(seg string) float64 {
	open := strings.IndexByte(seg, '(')
	n, _ := strconv.ParseFloat(seg[open+1:len(seg)-1], 64)
	return n * unitMs(seg[len("BRelTime.make"):open])
}

// literalMs is the ms of the first BRelTime literal in text, -1 when there is none.
func literalMs(text string) float64 {
	if seg := relLiteral.FindString(text); seg != "" {
		return relMs(seg)
	}
	return -1
}

// facetMinMs is the BFacets.MIN of a property declaration in ms; -1 when BFacets.MIN is present but its
// value is not a BRelTime literal, -2 when there is no BFacets.MIN.
func facetMinMs(text string) float64 {
	if m := facetMin.FindString(text); m != "" {
		return relMs(m[strings.Index(m, "BRelTime.make"):])
	}
	if strings.Contains(text, "BFacets.MIN") {
		return -1
	}
	return -2
}

func trimSpace(s string) string { return strings.Trim(s, " \t\n\v\f\r") }

// callClosed reports whether s (the text after a call's opening paren) holds its closing paren.
func callClosed(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return true
			}
			depth--
		}
	}
	return false
}

// secondArg is the second top-level argument of s, the text after a call's opening paren.
func secondArg(s string) string {
	depth, past := 0, false
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '(':
			depth++
		case c == ')':
			if depth == 0 {
				return out.String()
			}
			depth--
		case c == ',' && depth == 0:
			if past {
				return out.String()
			}
			past = true
			continue
		}
		if past {
			out.WriteByte(c)
		}
	}
	return out.String()
}

// helperName is the method a call text starts with (the last name of Class.fn( or fn( ), or "".
func helperName(s string) string {
	if m := qualCall.FindString(s); m != "" {
		m = callTail.ReplaceAllString(m, "")
		return m[strings.LastIndexByte(m, '.')+1:]
	}
	if m := bareCall.FindString(s); m != "" {
		return callTail.ReplaceAllString(m, "")
	}
	return ""
}
