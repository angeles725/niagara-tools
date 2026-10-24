package bogaudit

import (
	"fmt"
	"sort"
)

// The checks still to land, one commit per group; until then they print nothing.

func (a *audit) duplicates()      {}
func (a *audit) orphans()         {}
func (a *audit) hoaLeftover()     {}
func (a *audit) proxyLinkSafety() {}
func (a *audit) stationLogic()    {}

type sources struct{}

func scanSources(string) *sources { return &sources{} }

func (a *audit) sourceChecks() {
	if a.src == nil {
		for _, c := range []string{"CHECK2", "CHECK3", "CHECK4", "CHECK5", "CHECK6", "CHECK7"} {
			a.emit(c, "SKIP", "--source-dir", "not provided; skipping source-dependent check")
		}
	}
}

// inventory is CHECK1: the component count of each own module; false when there is none.
func (a *audit) inventory() bool {
	counts := map[string]int{}
	for _, h := range a.g.comps.keys {
		if c := a.g.comp(h); a.own[c.module] {
			counts[c.module]++
		}
	}
	mods := make([]string, 0, len(counts))
	for m := range counts {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	for _, m := range mods {
		a.emit("CHECK1", "PASS", m, fmt.Sprintf("components=%d", counts[m]))
	}
	return len(mods) > 0
}
