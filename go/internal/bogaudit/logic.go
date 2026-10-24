package bogaudit

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// The station-logic checks CHECK13-20 (D17): post-processing over the handle and link graphs.

var (
	relayWritable = regexp.MustCompile(`(?i)(Boolean|Numeric|Enum)Writable`)
	defrostSlot   = regexp.MustCompile(`(?i)(defrost|resistance)`)
	outSlot       = regexp.MustCompile(`(?i)(Out$|condenser[0-9]+$)`)
	cRoom         = regexp.MustCompile(`(?i)C([0-9]+)`)
	nameIndex     = regexp.MustCompile(`[_-]([0-9]+)$`)
	coldRoomType  = regexp.MustCompile(`(?i):ColdRoom$`)
	evapIndex     = regexp.MustCompile(`(?i)evap([0-9]+)`)
	evapUnitType  = regexp.MustCompile(`(?i):EvaporatorUnit\b`)
	ctrlType      = regexp.MustCompile(`(?i)(ColdRoom|EvaporatorUnit|DefrostController|CompressorControl)`)
	panelType     = regexp.MustCompile(`(?i)(RoomPanel|DashboardService)`)
	configSlot    = regexp.MustCompile(`(?i)(setpoint|mode|limit|protect|diff)`)
	stateSlot     = regexp.MustCompile(`(?i)(State|Out|status)$`)
	numericSrc    = regexp.MustCompile(`(?i)(Mode|Command|Cmd)`)
	boolGate      = regexp.MustCompile(`(?i)(BOrLogic|BAndLogic|BNotLogic)`)
)

func (a *audit) stationLogic() {
	a.relayDoubleSource()
	a.ownOutputUnlinked()
	a.sensorCrossed()
	a.defrostSiblings()
	a.roomIndex()
	a.tileNumbers()
	a.linkDirection()
	a.numericToBool()
}

// group1 is the first capture of re in s as Python's int() prints it.
func group1(re *regexp.Regexp, s string) (string, bool) {
	if m := re.FindStringSubmatch(s); m != nil {
		return pyInt(m[1]), true
	}
	return "", false
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// relayDoubleSource is CHECK13: two distinct source handles into one slot of a writable proxy point.
func (a *audit) relayDoubleSource() {
	srcs := map[string]map[string]map[string]bool{} // container -> target slot -> sources
	for _, l := range a.g.links {
		c := a.g.comp(l.container)
		if l.container == "" || l.tgtSl == "" || l.srcH == "" || c == nil || !relayWritable.MatchString(c.typ) {
			continue
		}
		if srcs[l.container] == nil {
			srcs[l.container] = map[string]map[string]bool{}
		}
		if srcs[l.container][l.tgtSl] == nil {
			srcs[l.container][l.tgtSl] = map[string]bool{}
		}
		srcs[l.container][l.tgtSl][l.srcH] = true
	}
	for _, h := range sortedKeys(srcs) {
		for _, tgt := range sortedKeys(srcs[h]) {
			if set := srcs[h][tgt]; len(set) >= 2 {
				hs := sortedKeys(set)
				for i := range hs {
					hs[i] = "h:" + hs[i]
				}
				a.emit("CHECK13", "FAIL", a.g.comp(h).path, fmt.Sprintf("relay slot %s driven by %d distinct sources: %s",
					pyRepr(tgt), len(set), strings.Join(hs, ", ")))
			}
		}
	}
}

// hasDefrost is the value of a component's hasDefrost slot ("" when absent or valueless).
func hasDefrost(c *comp) string { s, _ := c.slots.get("hasDefrost"); return s.value }

// ownOutputUnlinked is CHECK14: an own output (an OPERATOR *Out/condenserN slot in the bog, or an *Out
// slot other instances of its type link out) with no outgoing link; defrost outputs need hasDefrost.
func (a *audit) ownOutputUnlinked() {
	used := map[[2]string]bool{}
	typeOuts := map[string]map[string]bool{}
	for _, l := range a.g.links {
		if l.srcH == "" || l.srcSlot == "" {
			continue
		}
		used[[2]string{l.srcH, l.srcSlot}] = true
		if sc := a.ownComp(l.srcH); sc != nil && outSlot.MatchString(l.srcSlot) {
			if typeOuts[sc.typ] == nil {
				typeOuts[sc.typ] = map[string]bool{}
			}
			typeOuts[sc.typ][l.srcSlot] = true
		}
	}
	for _, c := range a.byPath() {
		if !a.own[c.module] {
			continue
		}
		cand := map[string]bool{}
		for _, n := range c.slots.keys {
			if s, _ := c.slots.get(n); strings.Contains(s.flags, "o") && outSlot.MatchString(n) {
				cand[n] = true
			}
		}
		for n := range typeOuts[c.typ] {
			cand[n] = true
		}
		off := isOff(hasDefrost(c)) && strings.ToLower(hasDefrost(c)) != "null"
		for _, n := range sortedKeys(cand) {
			if used[[2]string{c.handle, n}] || (defrostSlot.MatchString(n) && off) {
				continue
			}
			a.emit("CHECK14", "WARN", c.path, fmt.Sprintf("output slot %s has no outgoing relay link (own-output-unlinked)", pyRepr(n)))
		}
	}
}

// sensorCrossed is CHECK15: an own source slot labeled C<n> on a component whose name ends in another index.
func (a *audit) sensorCrossed() {
	for _, l := range a.g.links {
		sc := a.ownComp(l.srcH)
		if l.srcH == "" || l.srcSlot == "" || sc == nil {
			continue
		}
		cn, ok1 := group1(cRoom, l.srcSlot)
		en, ok2 := group1(nameIndex, sc.name)
		if !ok1 || !ok2 || cn == en {
			continue
		}
		a.emit("CHECK15", "WARN", a.containerPath(l), fmt.Sprintf("slot %s (C-room %s) sourced from %s (unit index %s) — label mismatch",
			pyRepr(l.srcSlot), cn, pyRepr(sc.name), en))
	}
}

// containerPath is the path of the component holding l, else "h:<handle>" as Python printed it.
func (a *audit) containerPath(l link) string {
	switch c := a.container(l); {
	case c != nil:
		return c.path
	case !l.hasContainer:
		return "h:None"
	}
	return "h:" + l.container
}
