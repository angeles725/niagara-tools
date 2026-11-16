package delays

import (
	"strconv"
	"strings"
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
