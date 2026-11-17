package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/px"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Whether a binding's target resolves (n4_px_binding_audit). A target that loads resolves. One
// that does not is missing only when proven: the deepest ancestor that loads holds no slot by the
// next name, neither loaded nor declared by its type contract. Anything else (a timeout, a
// station that hides the error, an ancestor whose contract cannot be read, a handle) is unknown,
// never counted as broken. Each path is loaded once per audit.

// Results of one target.
const (
	pbResolves = "resolves"
	pbMissing  = "missing"
	pbUnknown  = "unknown"
)

// pbLoaded is one cached load: the nodes, or why the load failed.
type pbLoaded struct {
	nodes *pyjson.Object
	why   string
}

type pbResolver struct {
	t     *Tools
	loads map[string]pbLoaded
}

// load is the ORD loaded one level deep, cached; only an authentication failure is an error.
func (r *pbResolver) load(ord string) (pbLoaded, error) {
	if l, ok := r.loads[ord]; ok {
		return l, nil
	}
	var l pbLoaded
	nodes, _, err := r.t.timedLoad(ord, 1)
	if err != nil {
		section, fatal := r.t.sectionError(err)
		if fatal != nil {
			return l, fatal
		}
		l.why = pyjson.Str(lookup(section, "error"))
	} else {
		l.nodes = nodes
	}
	r.loads[ord] = l
	return l, nil
}

// handle is the result of a target named by handle: a failed load proves nothing.
func (r *pbResolver) handle(h string) (string, string, error) {
	l, err := r.load("station:|h:" + h)
	switch {
	case err != nil:
		return "", "", err
	case l.nodes != nil:
		return pbResolves, "", nil
	}
	return pbUnknown, "the handle did not load (" + l.why + "); a handle cannot be checked by its parent", nil
}

// slot is the result of the target at the absolute slot path, and why when it does not resolve.
func (r *pbResolver) slot(path string) (string, string, error) {
	if _, err := bql.BaseSlot(path); err != nil {
		return pbUnknown, "not a plain slot path: " + path, nil
	}
	l, err := r.load(bql.StationSlot + path)
	if err != nil || l.nodes != nil {
		return pbResolves, "", err
	}
	segs := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segs) - 1; i >= 0; i-- {
		parent := bql.StationSlot + "/" + strings.Join(segs[:i], "/")
		pl, err := r.load(parent)
		if err != nil {
			return "", "", err
		}
		if pl.nodes == nil {
			continue
		}
		d := r.t.withDefaults(pl.nodes, []string{""})
		if _, held := d.nodes.Get(segs[i]); held {
			return pbUnknown, "the ORD did not load (" + l.why + ") although " + parent + " holds " + segs[i], nil
		}
		if len(d.notes) > 0 {
			return pbUnknown, "the ORD did not load (" + l.why + ") and " + parent + "'s declared slots are not known: " +
				strings.Join(d.notes, "; "), nil
		}
		return pbMissing, parent + " holds no slot " + segs[i], nil
	}
	return pbUnknown, "the ORD did not load (" + l.why + ") and no ancestor loaded", nil
}

// binding records one binding of file whose ORD reads as target: counted under its result and,
// unless it resolves, listed under broken (missing) or unknown with why. A target is checked once.
func (r *pbaRun) binding(file string, b px.Binding, target px.Target) error {
	var ord any
	result, why := pbUnknown, target.Why
	switch {
	case target.Path != "":
		ord = bql.StationSlot + target.Path
	case target.Handle != "":
		ord = "station:|h:" + target.Handle
	}
	if key, ok := ord.(string); ok {
		got, seen := r.targets[key]
		if !seen {
			r.counts["unique_targets"]++
			var err error
			if target.Path != "" {
				got[0], got[1], err = r.res.slot(target.Path)
			} else {
				got[0], got[1], err = r.res.handle(target.Handle)
			}
			if err != nil {
				return err
			}
			r.targets[key] = got
		}
		result, why = got[0], got[1]
	}
	r.counts[result]++
	if result == pbResolves {
		return nil
	}
	entry := obj("px_file", file, "widget", b.Widget, "element", b.Element, "ord", b.Ord, "target", ord, "why", why)
	if result == pbMissing {
		r.broken = append(r.broken, entry)
	} else {
		r.unknown = append(r.unknown, entry)
	}
	return nil
}
