package license

import (
	"strings"
	"unicode"
)

// What bounds the data read from a page, whatever the page holds: the length of every string
// kept, in characters, and how many entries of a kind and attributes of a feature are kept
// (the rest are counted). CutMarker ends a string that was cut.
const (
	MaxNameRunes    = 200 // a name, an expiry, a state, the host id, the licences qualifier
	MaxKeyRunes     = 64  // an attribute key
	MaxValueRunes   = 100 // an attribute value
	MaxEntries      = 100 // certificates, and licences
	MaxFeatureAttrs = 30  // attributes of one feature
	CutMarker       = "...[cut]"
)

// Clean is s without control characters, every run of white space as one space, trimmed: what
// is returned to a client never carries a control character of the page or of a file.
func Clean(s string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, s)), " ")
}

// Clip is s cut at max characters, ending with CutMarker when it was cut.
func Clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i] + CutMarker
		}
		n++
	}
	return s
}
