package readtools

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// The classification of n4_tag_coverage (station-audits S9), pure: what one tag read of a
// component says, which control points miss their Haystack point tags, which equips no point
// references, and which direct tags no installed dictionary defines.

// tcPointTag marks a Haystack point; tcEquipRef is the relation from a point to its equip (in
// Niagara 4 hs:equipRef is a relation, written as an outbound relation slot or implied by a rule).
const (
	tcPointTag = "hs:point"
	tcEquipRef = "hs:equipRef"
)

// tcEntity is what coverage needs of one component's tag read.
type tcEntity struct {
	ord, handle string
	direct      []string // direct tag ids, in slot order
	point       bool     // hs:point, direct or implied
	equipRefs   []string // endpoints of its outbound hs:equipRef relations, direct or implied
}

// tcEntityOf reads one n4_read_tags entry (tagEntry) of the component at ord.
func tcEntityOf(ord string, entry *pyjson.Object) tcEntity {
	e := tcEntity{ord: ord}
	e.handle, _ = lookup(entry, "handle").(string)
	for _, key := range []string{"direct", "implied"} {
		list, _ := lookup(entry, key).([]any)
		for _, raw := range list {
			id, _ := lookup(raw.(*pyjson.Object), "id").(string)
			if key == "direct" {
				e.direct = append(e.direct, id)
			}
			e.point = e.point || id == tcPointTag
		}
	}
	rels, _ := lookup(entry, "relations").(*pyjson.Object)
	for _, key := range []string{"direct", "implied"} {
		list, _ := lookup(rels, key).([]any)
		for _, raw := range list {
			r, _ := raw.(*pyjson.Object)
			end, _ := lookup(r, "endpoint_ord").(string)
			if lookup(r, "id") == tcEquipRef && !pyjson.Truthy(lookup(r, "inbound")) && end != "" {
				e.equipRefs = append(e.equipRefs, end)
			}
		}
	}
	return e
}

// tcPointGaps are the points missing hs:point or an hs:equipRef relation ({point, missing}),
// and how many miss each.
func tcPointGaps(points []tcEntity) (gaps []*pyjson.Object, noPoint, noEquip int) {
	for _, p := range points {
		missing := []any{}
		if !p.point {
			missing, noPoint = append(missing, tcPointTag), noPoint+1
		}
		if len(p.equipRefs) == 0 {
			missing, noEquip = append(missing, tcEquipRef), noEquip+1
		}
		if len(missing) > 0 {
			gaps = append(gaps, obj("point", p.ord, "missing", missing))
		}
	}
	return gaps, noPoint, noEquip
}

// tcEndpoint is an endpoint ORD without its local:| and station:| schemes: h:5 or slot:/AHU1.
func tcEndpoint(ord string) string {
	return strings.TrimPrefix(strings.TrimPrefix(ord, "local:|"), "station:|")
}

// tcOrphanEquips are the equips that no point's hs:equipRef names, by handle (h:5) or by slot.
func tcOrphanEquips(points, equips []tcEntity) []*pyjson.Object {
	named := map[string]bool{}
	for _, p := range points {
		for _, end := range p.equipRefs {
			named[tcEndpoint(end)] = true
		}
	}
	out := []*pyjson.Object{}
	for _, e := range equips {
		if !named["h:"+e.handle] && !named[tcEndpoint(e.ord)] {
			out = append(out, obj("equip", e.ord))
		}
	}
	return out
}

// tcDicts is what the installed dictionaries say: the dictionary name by namespace, whether one
// defines a tag, and the one namespace checked ("" for all).
type tcDicts struct {
	installed map[string]string
	defined   func(tags.ID) (bool, error)
	only      string
}

// tcUnknown is one unknown_tags entry being counted.
type tcUnknown struct {
	tag, example, why string
	count             int
}

// tcUnknownTags are the direct tags no installed dictionary defines ({tag, count, example, why}),
// most frequent first. The tags of a namespace no dictionary installs are one entry, ns:*, so a
// station without a Haystack dictionary reports "no dictionary installed" once, not per tag.
func tcUnknownTags(ents []tcEntity, d tcDicts) ([]*pyjson.Object, error) {
	byKey, order := map[string]*tcUnknown{}, []string{}
	for _, e := range ents {
		for _, raw := range e.direct {
			id, ok := tags.ParseID(raw)
			if !ok {
				id = tags.ID{Name: raw}
			}
			if d.only != "" && id.Namespace != d.only {
				continue
			}
			key, why := raw, ""
			if name, installed := d.installed[id.Namespace]; !installed {
				key, why = id.Namespace+":*", "no dictionary installed for namespace "+pyjson.StrRepr(id.Namespace)+
					" (ad hoc tags)"
			} else if ok, err := d.defined(id); err != nil {
				return nil, err
			} else if !ok {
				why = "not defined by the installed dictionary " + name
			}
			if why == "" {
				continue
			}
			if byKey[key] == nil {
				byKey[key], order = &tcUnknown{tag: key, example: e.ord, why: why}, append(order, key)
			}
			byKey[key].count++
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return byKey[order[i]].count > byKey[order[j]].count })
	out := []*pyjson.Object{}
	for _, k := range order {
		u := byKey[k]
		out = append(out, obj("tag", u.tag, "count", u.count, "example", u.example, "why", u.why))
	}
	return out, nil
}
