package readtools

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// The station reads of n4_tag_coverage: the dictionary catalog, the control points, the equips
// and one tag read per component.

// dictionaries is the catalog of the installed dictionaries (only want's, when named) and the
// names checked; a station with none checks nothing and says so once in the notes.
func (r *tcRun) dictionaries(want string) (tcDicts, []any, error) {
	cat, err := tags.LoadCatalog(r.t.Ctx.Session.Client.(Client))
	if err != nil {
		section, err := r.t.sectionError(err)
		if err != nil {
			return tcDicts{}, nil, err
		}
		r.notes = append(r.notes, "tag dictionaries not read: "+pyjson.Str(lookup(section, "error")))
		cat = &tags.Catalog{}
	}
	list := cat.Dictionaries()
	d := tcDicts{installed: map[string]string{}, defined: func(id tags.ID) (bool, error) {
		_, ok, err := cat.Def(id)
		return ok, err
	}}
	if want != "" {
		if list, err = pickDictionary(list, want); err != nil {
			return tcDicts{}, nil, err
		}
		d.only = list[0].Namespace
	}
	names := []any{}
	for _, x := range list {
		d.installed[x.Namespace], names = x.Name, append(names, x.Name+" ("+x.Namespace+")")
	}
	if len(list) == 0 {
		r.notes = append(r.notes, "no tag dictionary installed: every tag is ad hoc")
	}
	return d, names, nil
}

// pointPaths are the control point slot paths below scope, at most max.
func (r *tcRun) pointPaths(scope string, max int) ([]string, error) {
	columns, raw, cut, _, err := r.t.query(scope, bql.HistoriesAuditPointQuery, max, tcTimeout)
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, haPointFields)
	}
	if err != nil {
		return nil, err
	}
	if cut {
		r.truncated = true
		r.notes = append(r.notes, "control points cut at "+strconv.Itoa(len(rows))+": raise max_points or narrow base")
	}
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, cellText(row, "path"))
	}
	return paths, nil
}

// equipOrds are the hs:equip components below base (NEQL, implied tags included); ok false when
// the query failed (noted, never an asserted absence); an authentication failure fails the call.
func (r *tcRun) equipOrds(base string, max int) ([]string, bool, error) {
	found, err := r.t.NeqlQuery(obj("neql", "hs:equip", "base_ord", base,
		"max_rows", pyjson.IntLiteral(strconv.Itoa(max)), "timeout_s", pyjson.IntLiteral(strconv.Itoa(tcTimeout))))
	if err != nil {
		section, err := r.t.sectionError(err)
		if err != nil {
			return nil, false, err
		}
		r.notes = append(r.notes, "equips not read (neql hs:equip): "+pyjson.Str(lookup(section, "error")))
		return nil, false, nil
	}
	if lookup(found, "truncated") == true {
		r.truncated = true
		r.notes = append(r.notes, "equips cut at "+strconv.Itoa(max))
	}
	out := []string{}
	for _, o := range lookup(found, "ords").([]any) {
		out = append(out, o.(string))
	}
	return out, true, nil
}

// read is the tag read of each ord (prefix + path): points with implied tags and relations when
// the client can call the session handler, equips with their direct tags only. A component that
// cannot be read is listed under unreadable; an authentication failure fails the call.
func (r *tcRun) read(paths []string, prefix string, point bool) ([]tcEntity, error) {
	_, caller := r.t.Ctx.Session.Client.(sessionCaller)
	implied, relations := point && caller, "direct"
	if implied {
		relations = "all"
	}
	r.impliedMissing = r.impliedMissing || point && !caller
	out := make([]tcEntity, 0, len(paths))
	for _, p := range paths {
		entry, err := r.t.tagEntry(prefix+p, implied, relations)
		if err != nil {
			section, err := r.t.sectionError(err)
			if err != nil {
				return nil, err
			}
			if r.unreadableCount++; len(r.unreadable) < tcMaxItems {
				r.unreadable = append(r.unreadable, obj("ord", prefix+p, "error", lookup(section, "error")))
			}
			continue
		}
		out = append(out, tcEntityOf(prefix+p, entry))
	}
	return out, nil
}
