package citeresolve

import (
	"path/filepath"
	"strings"
)

// Scope says whether a card states live doctrine or records history.
type Scope string

// The scopes. --strict gates live doctrine; a history card's moved or shrunk targets are drift, not findings.
const (
	Live    Scope = "LIVE"
	History Scope = "HISTORY"
)

// PathRule marks the cards of one history family. A Pattern ending in "/" matches every card under that
// directory of the kit root; any other Pattern matches a card's file name anywhere.
type PathRule struct{ Pattern, Family string }

// HistoryRules is the one table of history cards (kit issue #415). A retro, a ledger fragment and a
// CHANGELOG entry record what was true when written; BUILD-STATE.md is generated from the ledger.d/
// fragments, so its rows repeat theirs, and BUILD-STATE.src.md, its hand-written part in the generated-ledger
// layout, is history the same way. Every other card is live doctrine.
var HistoryRules = []PathRule{
	{"retros/", "retros"},
	{"ledger.d/", "ledger.d"},
	{"BUILD-STATE.md", "BUILD-STATE"},
	{"BUILD-STATE.src.md", "BUILD-STATE"},
	{"CHANGELOG.md", "CHANGELOG"},
}

// Classifier returns the scope of a card path and, for a history card, its family.
type Classifier func(card string) (Scope, string)

// NewClassifier classifies cards by rules, with directory patterns relative to the kit root.
func NewClassifier(kit string, rules []PathRule) Classifier {
	absKit, _ := filepath.Abs(kit)
	return func(card string) (Scope, string) {
		abs, _ := filepath.Abs(card)
		rel, err := filepath.Rel(absKit, abs)
		rel = filepath.ToSlash(rel)
		for _, r := range rules {
			dir := strings.HasSuffix(r.Pattern, "/")
			if dir && err == nil && strings.HasPrefix(rel, r.Pattern) || !dir && filepath.Base(abs) == r.Pattern {
				return History, r.Family
			}
		}
		return Live, ""
	}
}

// drifting are the statuses that become HISTORICAL_DRIFT on a history card: the cited place moved or shrank,
// or its anchored text moved.
var drifting = map[Status]bool{MissingFile: true, MissingSection: true, MissingDelta: true, LineOutOfRange: true, Drift: true}

// passing are the statuses --strict accepts.
var passing = map[Status]bool{Resolved: true, HistoricalDrift: true, UnverifiableKind: true, SkippedTemplate: true}

// scoped applies the scope of the card to its result: history drift keeps the original status in the detail.
func scoped(res Result, scope Scope, family string) Result {
	if scope == History && drifting[res.Status] {
		res.Detail = strings.TrimSpace("was " + string(res.Status) + " (history " + family + "): " + res.Detail)
		res.Status = HistoricalDrift
	}
	return res
}
