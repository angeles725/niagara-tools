// Package facetscard diffs the BFacets key reference card (build-n4-module-kit/types/logic-authoring.md)
// against the BFacets key constants, in declaration order. It is the Go port of the original awk/sed
// check-facets-card.sh and keeps its output byte for byte. [ev: retro devguide415-runtime-api-corrections Δ8]
//
// Card rows are "| `NAME` | `\"key\"` | ..." lines from the `## BFacets key reference card` heading to
// the first `**Not` line or the next `## ` heading. The source is BFacets.java (every
// `public static final String NAME = "key";`) or a keys file (`NAME=key` lines; anything else, such
// as a `#` comment, is skipped).
package facetscard

import (
	"fmt"
	"regexp"
	"strings"
)

const cardHeading = "## BFacets key reference card"

var (
	cardRow = regexp.MustCompile("^\\| `([A-Z_]+)` \\| `\"([A-Za-z]+)\"` \\|")
	javaKey = regexp.MustCompile(`^[[:space:]]*public[[:space:]]+static[[:space:]]+final[[:space:]]+String[[:space:]]+` +
		`([A-Z_]+)[[:space:]]*=[[:space:]]*"([A-Za-z]+)";`)
	keysLine = regexp.MustCompile(`^[A-Z_]+=[A-Za-z]+$`)
)

// lines splits text on "\n"; a final line without a newline is still a line, and a trailing
// newline adds none.
func lines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// CardKeys returns the card rows of text as NAME=key, in card order. A repeated heading line (any
// line starting with the heading) keeps the section open, as the awk rule order did.
func CardKeys(text string) []string {
	var keys []string
	on := false
	for _, l := range lines(text) {
		switch {
		case strings.HasPrefix(l, cardHeading):
			on = true
		case !on:
		case strings.HasPrefix(l, "**Not") || strings.HasPrefix(l, "## "):
			return keys
		default:
			if m := cardRow.FindStringSubmatch(l); m != nil {
				keys = append(keys, m[1]+"="+m[2])
			}
		}
	}
	return keys
}

// SourceKeys returns the key constants of text as NAME=key, in source order: Java declarations
// when path ends in .java, otherwise whole `NAME=key` lines (duplicates kept).
func SourceKeys(path, text string) []string {
	var keys []string
	java := strings.HasSuffix(path, ".java")
	for _, l := range lines(text) {
		if java {
			if m := javaKey.FindStringSubmatch(l); m != nil {
				keys = append(keys, m[1]+"="+m[2])
			}
		} else if keysLine.MatchString(l) {
			keys = append(keys, l)
		}
	}
	return keys
}

// Compare returns one line per difference: `MISSING k` for each source key not on the card, then
// `EXTRA k` for each card row not in the source; only when both sets agree, `ORDER card #n c !=
// source s` for each position (up to the shorter list) where the two lists differ.
func Compare(card, src []string) []string {
	inCard, inSrc := set(card), set(src)
	var diffs []string
	for _, k := range src {
		if !inCard[k] {
			diffs = append(diffs, "MISSING "+k)
		}
	}
	for _, k := range card {
		if !inSrc[k] {
			diffs = append(diffs, "EXTRA "+k)
		}
	}
	if len(diffs) > 0 {
		return diffs
	}
	// Mutation: FC-order -- dropping the positional compare lets a reordered card pass
	for i := 0; i < len(card) && i < len(src); i++ {
		if card[i] != src[i] {
			diffs = append(diffs, fmt.Sprintf("ORDER card #%d %s != source %s", i+1, card[i], src[i]))
		}
	}
	return diffs
}

func set(keys []string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}
