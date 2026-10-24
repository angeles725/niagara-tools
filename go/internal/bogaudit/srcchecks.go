package bogaudit

import (
	"fmt"
	"sort"
	"strings"
)

// The checks that need --source-dir: CHECK12 (B816), then CHECK2-6 per own component in path order and
// CHECK7 per link; without it, one SKIP row each for CHECK2-7.

func (a *audit) sourceChecks() {
	if a.src == nil {
		for _, c := range []string{"CHECK2", "CHECK3", "CHECK4", "CHECK5", "CHECK6", "CHECK7"} {
			a.emit(c, "SKIP", "--source-dir", "not provided; skipping source-dependent check")
		}
		return
	}
	a.dashboardWriteLink()
	for _, c := range a.byPath() {
		if a.own[c.module] {
			a.componentDrift(c)
		}
	}
	a.danglingLinks()
}

// dashboardWriteLink is CHECK12: a slot a servlet writes that is also a link target, at the first link.
func (a *audit) dashboardWriteLink() {
	var conflict []string
	for w := range a.src.writes {
		for _, l := range a.g.links {
			if l.tgtSl == w {
				conflict = append(conflict, w)
				break
			}
		}
	}
	sort.Strings(conflict)
	for _, w := range conflict {
		for _, l := range a.g.links {
			if l.tgtSl != w {
				continue
			}
			path := w
			if c := a.container(l); c != nil {
				path = c.path + "/" + w
			}
			a.emit("CHECK12", "WARN", path, "servlet-written slot is also a link target (servlet write is ephemeral — link wins)")
			break
		}
	}
}

func (s *sources) classSlots(cls string) *omap[srcSlot] {
	if m := s.slots[cls]; m != nil {
		return m
	}
	return &omap[srcSlot]{}
}

// componentDrift is CHECK2 (action flag drift), CHECK3 (below the MIN facet), CHECK4 (a TRANSIENT slot
// persisted), CHECK5 (a bog slot the class does not declare) and CHECK6 (a declared slot not in the bog).
func (a *audit) componentDrift(c *comp) {
	cls := c.className()
	declared := a.src.classSlots(cls)
	if acts := a.src.actions[cls]; acts != nil {
		for _, n := range c.actions.keys {
			bogF, _ := c.actions.get(n)
			if want, ok := acts.get(n); ok && bogF != want {
				status := "WARN"
				if a.strict {
					status = "FAIL"
				}
				a.emit("CHECK2", status, c.path+"/"+n, fmt.Sprintf("flag-drift: bog f='%s' source expects f='%s'", bogF, want))
			}
		}
	}
	for _, n := range c.slots.keys {
		s, _ := c.slots.get(n)
		d, ok := declared.get(n)
		if !ok || !d.hasMin {
			continue
		}
		if v, ok := pyFloat(s.value); ok && v < d.min {
			a.emit("CHECK3", "WARN", c.path+"/"+n, fmt.Sprintf("value %s below source MIN %s", s.value, pyFloatRepr(d.min)))
		}
	}
	for _, n := range declared.keys {
		d, _ := declared.get(n)
		if s, ok := c.slots.get(n); ok && strings.Contains(d.flags, "TRANSIENT") && !isOff(s.value) {
			a.emit("CHECK4", "WARN", c.path+"/"+n, "TRANSIENT slot has persisted value in bog")
		}
	}
	super := a.src.extends[cls]
	_, ownSlots := a.src.slots[super]
	_, ownActs := a.src.actions[super]
	extendsOwn := ownSlots || ownActs || super == "" || super == "BComponent" || super == "BAbstractService"
	for _, n := range c.slots.keys {
		s, _ := c.slots.get(n)
		if _, ok := declared.get(n); ok || strings.Contains(strings.ToLower(s.flags), "r") {
			continue
		}
		if s.typ == "" && !extendsOwn {
			a.emit("CHECK5", "WARN", c.path+"/"+n, fmt.Sprintf("frozen slot not in source %s — possibly inherited from %s "+
				"(not statically decidable; add @NiagaraProperty override to suppress)", cls, super))
		} else {
			a.emit("CHECK5", "FAIL", c.path+"/"+n, fmt.Sprintf("slot in bog not in source %s (ghost/orphan)", cls))
		}
	}
	for _, n := range declared.keys {
		if _, ok := c.slots.get(n); !ok {
			a.emit("CHECK6", "WARN", c.path+"/"+n, fmt.Sprintf("slot in source %s not in bog", cls))
		}
	}
}

// danglingLinks is CHECK7: a link into an own component whose targetSlotName its class does not declare.
func (a *audit) danglingLinks() {
	for _, l := range a.g.links {
		c := a.g.comp(l.container)
		if l.container == "" || c == nil || !a.own[c.module] || l.tgtSl == "" {
			continue
		}
		if _, ok := a.src.classSlots(c.className()).get(l.tgtSl); !ok {
			a.emit("CHECK7", "FAIL", c.path, fmt.Sprintf("link targetSlotName \"%s\" not in source %s (dangling)", l.tgtSl, c.className()))
		}
	}
}
