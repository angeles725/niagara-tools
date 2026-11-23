// Package cardwidth bounds the line length of the live doctrine cards. A card line of several thousand
// characters shows as wholly added on a one-phrase edit, and a search cut to a terminal width misses what
// stands past the cut (retro go-toolbelt-small-port Δ5). A card is wrapped at Limit; the cards not yet
// wrapped are held at the count of overlong lines recorded in baseline.txt, which may only go down.
package cardwidth

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/prepush"
)

// Limit is the width a wrapped card keeps its lines to, in characters.
const Limit = 120

// Overlong returns the 1-based numbers of the lines of a card that exceed Limit and could have been wrapped:
// there is a space outside a code span and an [ev: ...] or [deferred: ...] tag at or past column Limit. A
// line that overflows only through one unbreakable span is not one. Fenced code, table rows, headings and
// HTML comments are exempt.
func Overlong(text string) []int {
	var out []int
	fence := false
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~"):
			fence = !fence
		case fence, strings.HasPrefix(trimmed, "|"), strings.HasPrefix(trimmed, "#"), strings.HasPrefix(trimmed, "<!--"):
		case utf8.RuneCountInString(line) > Limit && breakablePast(strings.TrimRight(line, " \t"), Limit):
			out = append(out, i+1)
		}
	}
	return out
}

// breakablePast reports whether line has a breakable space at rune index col or later.
func breakablePast(line string, col int) bool {
	inCode, inTag, n := false, false, 0
	for i, r := range line {
		switch {
		case r == '`':
			inCode = !inCode
		case !inCode && r == '[' && (strings.HasPrefix(line[i:], "[ev:") || strings.HasPrefix(line[i:], "[deferred:")):
			inTag = true
		case inTag && r == ']':
			inTag = false
		case r == ' ' && !inCode && !inTag && n >= col:
			return true
		}
		n++
	}
	return false
}

// Counts is the number of overlong lines of each live card under root, by repository-relative path.
func Counts(root string) (map[string]int, error) {
	counts := map[string]int{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "bin", "openspec", "retros", "ledger.d", "odd":
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if !prepush.IsLiveDoctrine(rel) {
			return nil
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		counts[rel] = len(Overlong(string(text)))
		return nil
	})
	return counts, err
}

// ParseBaseline reads "<card> <count>" lines (blank lines and # comments skipped).
func ParseBaseline(text string) (map[string]int, error) {
	base := map[string]int{}
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		n, err := strconv.Atoi(f[len(f)-1])
		if len(f) != 2 || err != nil || n <= 0 {
			return nil, fmt.Errorf("baseline line %d: want \"<card> <count above 0>\", got %q", i+1, line)
		}
		base[f[0]] = n
	}
	return base, nil
}

// Check compares the counts of the live cards with the baseline and returns one message per card that is
// over it, or whose baseline is above its count (a wrapped card leaves the baseline, or the count is
// lowered: the number only goes down).
func Check(counts, baseline map[string]int) []string {
	var msgs []string
	for card, n := range counts {
		switch want := baseline[card]; {
		case n > want:
			msgs = append(msgs, fmt.Sprintf("%s: %d line(s) over %d characters, %d allowed: wrap them (a line holds at most %d characters unless one code span or tag fills it)", card, n, Limit, want, Limit))
		case n < want:
			msgs = append(msgs, fmt.Sprintf("%s: %d line(s) over %d characters, baseline says %d: lower it in go/internal/cardwidth/baseline.txt", card, n, Limit, want))
		}
	}
	for card := range baseline {
		if _, ok := counts[card]; !ok {
			msgs = append(msgs, card+": in the baseline but not a live card")
		}
	}
	sort.Strings(msgs)
	return msgs
}
