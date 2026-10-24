package bogaudit

import (
	"fmt"
	"sort"
	"strings"
)

// defrostSiblings is CHECK16 both ways, per parent path in dict order: an own hasDefrost=true component
// with no DefrostController sibling, and an own DefrostController with no hasDefrost=true sibling.
func (a *audit) defrostSiblings() {
	var parents omap[[]*comp]
	for _, h := range a.g.comps.keys {
		c := a.g.comp(h)
		p := ""
		if i := strings.LastIndexByte(c.path, '/'); i >= 0 {
			p = c.path[:i]
		}
		kids, _ := parents.get(p)
		parents.set(p, append(kids, c))
	}
	for _, p := range parents.keys {
		kids, _ := parents.get(p)
		var hd, dc []*comp
		for _, c := range kids {
			if a.own[c.module] && strings.ToLower(hasDefrost(c)) == "true" {
				hd = append(hd, c)
			}
			if strings.Contains(c.name, "DefrostController") || strings.Contains(c.typ, "DefrostController") {
				dc = append(dc, c)
			}
		}
		label := "(root)"
		if p != "" {
			label = pyRepr(p)
		}
		for _, c := range hd {
			if len(dc) == 0 {
				a.emit("CHECK16", "FAIL", c.path, "hasDefrost=true but no DefrostController sibling under "+label)
			}
		}
		for _, c := range dc {
			if len(hd) == 0 && a.own[c.module] {
				a.emit("CHECK16", "FAIL", c.path, "DefrostController without hasDefrost=true unit sibling under "+label)
			}
		}
	}
}

// roomIndex is CHECK17: a link held by an own ColdRoom whose target slot names evap<n> for another room.
func (a *audit) roomIndex() {
	for _, l := range a.g.links {
		c := a.ownComp(l.container)
		if l.container == "" || l.tgtSl == "" || c == nil || !coldRoomType.MatchString(c.typ) {
			continue
		}
		room, ok1 := group1(nameIndex, c.name)
		evap, ok2 := group1(evapIndex, l.tgtSl)
		if ok1 && ok2 && room != evap {
			a.emit("CHECK17", "FAIL", c.path, fmt.Sprintf("link %s carries evap-index %s but room suffix is %s", pyRepr(l.tgtSl), evap, room))
		}
	}
}

// tileNumbers is CHECK18: the evap<n> tile numbers in the links into and out of one own EvaporatorUnit
// that has a numeric suffix disagree.
func (a *audit) tileNumbers() {
	tiles := map[string]map[string]bool{}
	add := func(h, slotName string) {
		if n, ok := group1(evapIndex, slotName); ok {
			if tiles[h] == nil {
				tiles[h] = map[string]bool{}
			}
			tiles[h][n] = true
		}
	}
	isUnit := func(c *comp) bool { return c != nil && evapUnitType.MatchString(c.typ) }
	for _, l := range a.g.links {
		if l.container != "" && isUnit(a.ownComp(l.container)) {
			add(l.container, l.srcSlot)
			add(l.container, l.tgtSl)
		}
		if l.srcH != "" && isUnit(a.ownComp(l.srcH)) {
			add(l.srcH, l.tgtSl)
		}
	}
	for _, h := range sortedKeys(tiles) {
		c := a.g.comp(h)
		if len(tiles[h]) < 2 || c == nil || !nameIndex.MatchString(c.name) {
			continue
		}
		ns := sortedKeys(tiles[h])
		sort.Slice(ns, func(i, j int) bool { return intLess(ns[i], ns[j]) })
		a.emit("CHECK18", "FAIL", c.path, fmt.Sprintf("link tile numbers %s disagree across HOA/state/freeze links (tile-number mismatch)",
			strings.Join(ns, ", ")))
	}
}

func isCtrl(c *comp) bool { return ctrlType.MatchString(c.typ) || ctrlType.MatchString(c.name) }
func isPanel(c *comp) bool {
	return panelType.MatchString(c.typ) || strings.Contains(strings.ToLower(c.module), "dashboardpan")
}

// linkDirection is CHECK19: an own control source writing a config slot into a panel, or an own panel
// source writing a state slot into a control component.
func (a *audit) linkDirection() {
	for _, l := range a.g.links {
		sc, c := a.ownComp(l.srcH), a.container(l)
		if l.srcH == "" || l.tgtSl == "" || sc == nil || c == nil {
			continue
		}
		switch {
		case isCtrl(sc) && isPanel(c) && configSlot.MatchString(l.tgtSl):
			a.emit("CHECK19", "WARN", c.path, fmt.Sprintf("control %s writes config slot %s into panel container (reverse direction)",
				pyRepr(sc.path), pyRepr(l.tgtSl)))
		case isPanel(sc) && isCtrl(c) && stateSlot.MatchString(l.tgtSl):
			a.emit("CHECK19", "WARN", c.path, fmt.Sprintf("panel %s writes state slot %s into control container (reverse direction)",
				pyRepr(sc.path), pyRepr(l.tgtSl)))
		}
	}
}

// numericToBool is CHECK20: a Mode/Command/Cmd source slot linked straight into a boolean gate.
func (a *audit) numericToBool() {
	for _, l := range a.g.links {
		c := a.container(l)
		if !numericSrc.MatchString(l.srcSlot) || c == nil || !boolGate.MatchString(c.typ) {
			continue
		}
		a.emit("CHECK20", "FAIL", c.path, fmt.Sprintf("numeric/enum slot %s links directly into boolean gate %s "+
			"with no Equal(x,N) intermediate (!=0-as-true trap)", pyRepr(l.srcSlot), pyRepr(c.typ)))
	}
}
