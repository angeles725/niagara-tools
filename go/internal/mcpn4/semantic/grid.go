package semantic

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"sort"
	"strconv"
	"strings"
)

// Namespace is the tag namespace exported (Project Haystack); other namespaces are left out.
const Namespace = "hs"

// RefID is the Haystack id of a station ORD: the slot path with its $-escapes decoded, '/' as
// '.', every character a Haystack ref does not allow (a-z A-Z 0-9 _ : - . ~) as '_'.
func RefID(ord string) string {
	p := ord
	if i := strings.Index(p, "slot:"); i >= 0 {
		p = p[i+len("slot:"):]
	}
	p = SlotText(strings.Trim(p, "/"))
	var b strings.Builder
	for _, r := range p {
		switch {
		case r == '/':
			b.WriteByte('.')
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', strings.ContainsRune("_:-.~", r):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "root"
	}
	return b.String()
}

// Grid builds the Hayson grid of entities: one row per entity with id, dis and every hs: tag
// (namespace stripped) and hs: relation (as a ref to the endpoint's id); cols are the union of
// names, id and dis first, the rest sorted. Two ORDs mapping to one id get ~2, ~3 suffixes.
// Values that cannot be converted are returned as skipped.
func Grid(entities []Entity) (map[string]any, []Skipped) {
	ids := uniqueIDs(entities)
	byOrd := map[string]int{}
	for i, e := range entities {
		byOrd[e.Ord] = i
		if e.Handle != "" {
			byOrd["h:"+e.Handle] = i
		}
	}
	var skipped []Skipped
	names := map[string]bool{}
	rows := []any{}
	for i, e := range entities {
		row := map[string]any{"id": Ref(ids[i], e.Dis), "dis": e.Dis}
		for _, t := range e.Tags {
			name, ok := strings.CutPrefix(t.ID, Namespace+":")
			if !ok || name == "id" || name == "dis" {
				continue
			}
			v, why := value(t)
			if why != "" {
				skipped = append(skipped, Skipped{e.Ord, t.ID, why})
				continue
			}
			row[name], names[name] = v, true
		}
		for _, r := range e.Relations {
			name, ok := strings.CutPrefix(r.ID, Namespace+":")
			if !ok {
				continue
			}
			target := RefID(r.Endpoint)
			if j, in := byOrd[r.Endpoint]; in {
				target = ids[j]
			}
			row[name], names[name] = Ref(target, ""), true
		}
		rows = append(rows, row)
	}
	cols := []any{map[string]any{"name": "id"}, map[string]any{"name": "dis"}}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	for _, n := range sorted {
		cols = append(cols, map[string]any{"name": n})
	}
	return map[string]any{"_kind": "grid", "meta": map[string]any{"ver": "3.0"}, "cols": cols, "rows": rows}, skipped
}

func uniqueIDs(entities []Entity) []string {
	seen := map[string]int{}
	out := make([]string, len(entities))
	for i, e := range entities {
		id := RefID(e.Ord)
		seen[id]++
		if n := seen[id]; n > 1 {
			id += "~" + strconv.Itoa(n)
		}
		out[i] = id
	}
	return out
}

// SlotText is a slot name or path with its $-escapes decoded as the station decodes them
// ($20 is a space); text that is not a valid escape stays as written.
func SlotText(s string) string {
	if d, err := bql.DecodeForBql(s); err == nil {
		return d
	}
	return s
}
