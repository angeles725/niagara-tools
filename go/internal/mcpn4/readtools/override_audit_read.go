package readtools

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The reads of n4_override_audit: the writables of each type, the schedule handles, and the link
// that feeds in16 of each overridden writable.

var oaFields = []string{"path", "out", "expires"}

// readWritables reads every writable type below scope; a type the station refuses is listed under
// unreadable and the others are still read (an authentication failure fails the call).
func (r *oaRun) readWritables(scope string) error {
	for _, typ := range bql.WritableTypes {
		columns, raw, cut, _, err := r.t.query(scope, bql.OverrideQuery(typ), r.max, oaTimeout)
		var rows []*pyjson.Object
		if err == nil {
			rows, err = bql.ByPosition(columns, raw, oaFields)
		}
		if err != nil {
			section, fatal := r.t.sectionError(err)
			if fatal != nil {
				return fatal
			}
			r.unread = append(r.unread, obj("type", typ, "error", lookup(section, "error")))
			continue
		}
		if cut {
			r.cut = true
			r.notes = append(r.notes, typ+" cut at "+strconv.Itoa(len(rows))+" writables: raise max_writables or narrow base")
		}
		for _, row := range rows {
			w := oaWritable{path: cellText(row, "path"), typ: typ, out: cellText(row, "out")}
			w.level, w.levelName = bql.OutLevel(w.out)
			if text := strings.TrimSpace(cellText(row, "expires")); text != "" && text != "null" {
				w.expires = text
			}
			r.byLevel[w.levelName]++
			r.writes = append(r.writes, w)
		}
	}
	return nil
}

// handles maps a schedule's handle to its slot path; nil with a note when the station refuses the
// handle column.
func (r *oaRun) handles() map[string]string {
	columns, raw, _, _, err := r.t.query(bql.StationSlot+"/", bql.ScheduleHandlesQuery, bql.MaxRowsLimit, oaTimeout)
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, []string{"path", "handle"})
	}
	if err != nil {
		section, _ := r.t.sectionError(err)
		r.notes = append(r.notes, "schedule handles not read ("+pyjson.Str(lookup(section, "error"))+
			"): in16.schedule is null; in16.source_ord names the source")
		return nil
	}
	out := map[string]string{}
	for _, row := range rows {
		if h := cellText(row, "handle"); h != "" {
			out[h] = cellText(row, "path")
		}
	}
	return out
}

// in16 is what feeds a writable's in16: the link read from the writable's own load.
type in16 struct {
	linked   bool
	err      string
	ord, out any
	schedule any
}

// feedIn16 sets the in16 of each overridden writable; the first oaMaxIn16Loads are loaded.
func (r *oaRun) feedIn16(over []oaWritable) error {
	if len(over) == 0 {
		return nil
	}
	handles := r.handles()
	for i := range over {
		if i >= oaMaxIn16Loads {
			r.notes = append(r.notes, "in16 read for the first "+strconv.Itoa(oaMaxIn16Loads)+" overrides only")
			break
		}
		nodes, _, err := r.t.timedLoad("station:|"+over[i].path, 2)
		if err != nil {
			section, fatal := r.t.sectionError(err)
			if fatal != nil {
				return fatal
			}
			over[i].in16 = &in16{err: pyjson.Str(lookup(section, "error"))}
			continue
		}
		over[i].in16 = linkTo16(nodes, handles)
	}
	return nil
}

// linkTo16 is the first link whose target slot is in16 among the writable's children.
func linkTo16(nodes *pyjson.Object, handles map[string]string) *in16 {
	for _, name := range box.Children(nodes, "").Keys() {
		node := box.Node(box.Children(nodes, ""), name)
		if kind, _ := node.Get("t"); !isLinkType(kind) {
			continue
		}
		kids := box.Children(nodes, name)
		if box.ChildValue(kids, "targetSlotName", nil) != "in16" {
			continue
		}
		src := box.ChildValue(kids, "sourceOrd", nil)
		f := &in16{linked: true, ord: src, out: box.ChildValue(kids, "sourceSlotName", nil)}
		tail, _ := src.(string)
		tail = tail[strings.LastIndex(tail, "|")+1:]
		if h, ok := strings.CutPrefix(tail, handlePrefix); ok {
			if p, ok := handles[h]; ok {
				f.schedule = "station:|" + p
			}
		} else if p, ok := strings.CutPrefix(tail, slotPrefix); ok {
			for _, sp := range handles {
				if sp == p {
					f.schedule = "station:|" + p
				}
			}
		}
		return f
	}
	return &in16{}
}

// oaEntries is the overrides as reply entries.
func oaEntries(over []oaWritable) []any {
	out := make([]any, len(over))
	for i, w := range over {
		out[i] = obj("writable", "station:|"+w.path, "type", w.typ, "level", w.level, "kind", oaKind(w.level), "out", w.out,
			"expires", w.expires, "permanent", w.expires == nil, "in16", w.in16.view())
	}
	return out
}

func (f *in16) view() any {
	switch {
	case f == nil:
		return nil
	case f.err != "":
		return obj("linked", nil, "error", f.err)
	}
	return obj("linked", f.linked, "source_ord", f.ord, "source_slot", f.out, "schedule", f.schedule)
}
