package jsonrpc

import (
	"strings"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// maxEcho is how many characters of a caller's value or argument name a refusal echoes, and the
// longest unknown argument name an edit distance is computed for.
const maxEcho = 64

// clip is text cut to maxEcho characters plus "…", so a huge value never lands in an error. It
// walks at most maxEcho+1 characters, whatever the length of text.
func clip(text string) string {
	i := 0
	for n := 0; i < len(text); n++ {
		if n == maxEcho {
			return text[:i] + "…"
		}
		_, size := utf8.DecodeRuneInString(text[i:])
		i += size
	}
	return text
}

// echo is value as a refusal shows it: a string clipped then quoted, so its closing quote
// survives the clip; anything else is its representation, clipped.
func echo(value any) string {
	if s, ok := value.(string); ok {
		return pyjson.StrRepr(clip(s))
	}
	return clip(pyjson.Repr(value))
}

// nearestKey is the declared key closest to key (see closeness); "" when none or a tie.
func nearestKey(key string, known []string) string {
	best, bestScore, tie := "", -1, false
	for _, k := range known {
		score := closeness(key, k)
		if score < 0 {
			continue
		}
		if bestScore < 0 || score < bestScore {
			best, bestScore, tie = k, score, false
		} else if score == bestScore {
			tie = true
		}
	}
	if tie {
		return ""
	}
	return best
}

// closeness is 0 when the declared key k starts key (capacity_records, capacity) or key starts k
// while covering at least half of it (a short word such as all or ack never names a safety
// override such as allow_ad_hoc or acknowledge_links); else their edit distance when it is
// within max(1, len(k)/4); else -1. The edit distance, O(len(key)×len(k)), is computed only for
// a key of at most maxEcho bytes whose length is within that distance of k's: the length gap is
// a lower bound of the distance, and a longer key is the caller's, of any size.
func closeness(key, k string) int {
	if strings.HasPrefix(key, k) || (strings.HasPrefix(k, key) && 2*len(key) >= len(k)) {
		return 0
	}
	allowed := max(1, len(k)/4)
	if len(key) > maxEcho || abs(len(key)-len(k)) > allowed {
		return -1
	}
	if d := editDistance(key, k); d <= allowed {
		return d
	}
	return -1
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// editDistance is the optimal-string-alignment distance of a and b: insertions, deletions,
// substitutions and swaps of two neighbours (depht, depth) cost 1 each. Bytes: argument names
// are ASCII.
func editDistance(a, b string) int {
	rows := make([][]int, len(a)+1)
	for i := range rows {
		rows[i] = make([]int, len(b)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d := min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d = min(d, rows[i-2][j-2]+1)
			}
			rows[i][j] = d
		}
	}
	return rows[len(a)][len(b)]
}
