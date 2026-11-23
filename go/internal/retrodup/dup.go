// Package retrodup judges whether two descriptions of a retro delta are about the same change: it extracts weighted
// tokens (identifiers such as n4_* tool names, lint ids, --options, file names and camelCase names weigh more than
// plain words) and scores two token sets with a weighted Dice coefficient. retro-issues uses it to search before it
// files: one issue per root cause, extra sources become comments.
package retrodup

import (
	"regexp"
	"sort"
	"strings"
)

// DefaultThreshold is the score from which two descriptions are a probable duplicate.
const DefaultThreshold = 0.5

const idWeight = 3

var (
	idRe   = regexp.MustCompile(`\bn4_[a-z0-9_]+|\blint-[a-z0-9-]+|--[a-z][a-z0-9-]+|[A-Za-z0-9_./-]+\.(?:go|sh|md|bats|java|json|xml|gradle|kts)\b|\b[A-Za-z][a-z0-9]*[A-Z][A-Za-z0-9]*\b`)
	wordRe = regexp.MustCompile(`[a-z][a-z0-9]{4,}`)
	stop   = map[string]bool{}
)

func init() {
	for _, w := range strings.Fields(`source target evidence retro delta issue should would which their there these those about after before being other
		every where while because using takes taken when with from that this must never always also proposed change applied open`) {
		stop[w] = true
	}
}

// Tokens extracts the weighted tokens of a title and a text (identifiers: weight 3, words of 5+ letters: weight 1).
func Tokens(title, text string) map[string]int {
	s := title + "\n" + text
	tok := map[string]int{}
	for _, id := range idRe.FindAllString(s, -1) {
		tok[strings.ToLower(id)] = idWeight
	}
	rest := idRe.ReplaceAllString(s, " ")
	for _, w := range wordRe.FindAllString(strings.ToLower(rest), -1) {
		if _, isID := tok[w]; !stop[w] && !isID {
			tok[w] = 1
		}
	}
	return tok
}

func weight(t map[string]int) (sum int) {
	for _, w := range t {
		sum += w
	}
	return sum
}

// Score is the weighted Dice coefficient of two token sets: 2·W(a∩b) / (W(a)+W(b)), 0 when either side is empty.
func Score(a, b map[string]int) float64 {
	wa, wb := weight(a), weight(b)
	if wa == 0 || wb == 0 {
		return 0
	}
	common := 0
	for k, w := range a {
		if _, ok := b[k]; ok {
			common += w
		}
	}
	return 2 * float64(common) / float64(wa+wb)
}

// Terms is the n heaviest tokens, heaviest first (ties alphabetical), space separated: the search terms.
func Terms(tok map[string]int, n int) string {
	keys := make([]string, 0, len(tok))
	for k := range tok {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if tok[keys[i]] != tok[keys[j]] {
			return tok[keys[i]] > tok[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	return strings.Join(keys, " ")
}
