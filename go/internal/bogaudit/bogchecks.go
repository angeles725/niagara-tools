package bogaudit

import (
	"fmt"
	"regexp"
	"sort"
)

var hoaSlot = regexp.MustCompile(`^in[1-8]$`)

// The checks that read the bog alone: CHECK10, CHECK9, CHECK1, CHECK8 and CHECK11 (B810).

// duplicates is CHECK10: a handle on more than one component, at the path of the last one.
func (a *audit) duplicates() {
	for _, h := range a.g.counts.keys {
		if n, _ := a.g.counts.get(h); n > 1 {
			a.emit("CHECK10", "FAIL", a.g.comp(h).path, fmt.Sprintf("handle h:%s appears %d times (duplicate)", h, n))
		}
	}
}

// orphans is CHECK9: a link whose sourceOrd handle names no component.
func (a *audit) orphans() {
	for _, l := range a.g.links {
		if l.srcH != "" && a.g.comp(l.srcH) == nil {
			a.emit("CHECK9", "FAIL", "h:"+l.srcH, "link sourceOrd references an unknown handle (orphan)")
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

// hoaLeftover is CHECK8: a writable whose in1..in8 holds a value other than null/false/0.
func (a *audit) hoaLeftover() {
	for _, h := range a.g.comps.keys {
		c := a.g.comp(h)
		if !c.writable {
			continue
		}
		for _, n := range c.slots.keys {
			if s, _ := c.slots.get(n); hoaSlot.MatchString(n) && !isOff(s.value) {
				a.emit("CHECK8", "WARN", c.path+"/"+n, "HOA override active (value="+pyRepr(s.value)+")")
			}
		}
	}
}

// proxyLinkSafety is CHECK11: a writable fed by an own-module output with no explicit fallback, one row
// per writable (containers in handle order), naming its first own-module link.
func (a *audit) proxyLinkSafety() {
	byContainer := map[string][]link{}
	for _, l := range a.g.links {
		if l.container != "" {
			byContainer[l.container] = append(byContainer[l.container], l)
		}
	}
	hs := make([]string, 0, len(byContainer))
	for h := range byContainer {
		hs = append(hs, h)
	}
	sort.Strings(hs)
	for _, h := range hs {
		c := a.g.comp(h)
		if c == nil || !c.writable || c.hasFallback {
			continue
		}
		for _, l := range byContainer[h] {
			if l.srcH != "" && a.ownComp(l.srcH) != nil {
				a.emit("CHECK11", "FAIL", c.path, fmt.Sprintf("own-module h:%s.%s -> writable with no explicit fallback "+
					"(relay holds last state on stop/reload)", l.srcH, none(l.srcSlot)))
				break
			}
		}
	}
}

// none prints a never-recorded link child as Python printed None.
func none(s string) string {
	if s == "" {
		return "None"
	}
	return s
}
