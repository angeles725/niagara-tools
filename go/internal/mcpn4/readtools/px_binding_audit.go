package readtools

import (
	"errors"
	"math"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/files"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/px"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_px_binding_audit (station-audits S15): the Px bindings whose ORD does not resolve, read-only.
// It composes existing reads: n4_list_px_views' query for the views and their owners, the file
// fetch of n4_px_summary for each .px, the px parser for the binding ORDs, and one cached BOX load
// per distinct target (px_binding_resolve.go).

// Bounds of n4_px_binding_audit.
const (
	PbaDefaultMaxViews    = 50
	PbaDefaultMaxBindings = 500
)

// pbaRun is one audit: the bindings seen, what each target gave, and the reply lists.
type pbaRun struct {
	t           *Tools
	args        *pyjson.Object
	res         *pbResolver
	maxBindings int
	truncated   bool
	targets     map[string][2]string // target ORD -> result, why
	views       []any
	broken      []any
	unknown     []any
	counts      map[string]int
}

// PxBindingAudit is n4_px_binding_audit: one file (file_ord, with an optional base) or every Px
// view below root, each relative ORD read against the component its view is on.
func (t *Tools) PxBindingAudit(args *pyjson.Object) (*pyjson.Object, error) {
	start := t.Ctx.Clock()
	r := &pbaRun{t: t, args: args, res: &pbResolver{t: t, loads: map[string]pbLoaded{}},
		maxBindings: intArg(args, "max_bindings", PbaDefaultMaxBindings), targets: map[string][2]string{},
		views: []any{}, broken: []any{}, unknown: []any{}, counts: map[string]int{}}
	out, key := pyjson.NewObject(), "root"
	if v, ok := args.Get("file_ord"); ok {
		key = "file_ord"
		var baseOrd any
		if b, ok := args.Get("base"); ok {
			slot, err := bql.BaseSlot(b)
			if err != nil {
				return nil, toolError("base must be a station slot ORD such as station:|slot:/Plant/AHU1, got %s",
					pyjson.Repr(b))
			}
			baseOrd = bql.StationSlot + slot
		}
		if err := r.view(v.(string), baseOrd); err != nil {
			return nil, err
		}
		out.Set("file_ord", v)
	} else {
		root := strArg(args, "root", RootOrd)
		listed, err := t.ListPxViews(obj("root", root, "max_rows",
			pyjson.IntLiteral(strconv.Itoa(intArg(args, "max_views", PbaDefaultMaxViews)))))
		if err != nil {
			return nil, err
		}
		if cut, _ := listed.Get("truncated"); cut == true {
			r.truncated = true
		}
		seen := map[string]bool{}
		for _, item := range lookup(listed, "views").([]any) {
			file, base := lookup(item.(*pyjson.Object), "px_file"), lookup(item.(*pyjson.Object), "owner_path")
			key := pyjson.Str(file) + "\x00" + pyjson.Str(base)
			if seen[key] {
				continue
			}
			seen[key] = true
			if file == nil {
				r.views = append(r.views, obj("px_file", nil, "base", base, "status", "no_file", "bindings", 0))
				continue
			}
			if err := r.view(file.(string), base); err != nil {
				return nil, err
			}
		}
		out.Set("root", root)
	}
	r.counts["views"] = len(r.views)
	counts := pyjson.NewObject()
	for _, k := range []string{"views", "bindings", "unique_targets", pbResolves, pbMissing, pbUnknown} {
		counts.Set(k, r.counts[k])
	}
	for _, kv := range [][2]any{{"views", r.views}, {"broken", r.broken}, {"unknown", r.unknown},
		{"counts", counts}, {"truncated", r.truncated},
		{"elapsed_ms", int(math.RoundToEven((t.Ctx.Clock() - start) * 1000))}} {
		out.Set(kv[0].(string), kv[1])
	}
	t.recordRead("n4_px_binding_audit", key, args, r.counts["bindings"])
	return out, nil
}

// view audits one .px file with base (the owner's ORD, or nil) as the base of relative ORDs.
func (r *pbaRun) view(file string, base any) error {
	fetch := obj("file_ord", file)
	for _, k := range []string{"max_bytes", "timeout_s"} {
		if v, ok := r.args.Get(k); ok {
			fetch.Set(k, v)
		}
	}
	got, body, ok, err := r.t.fetchFile("n4_px_binding_audit", fetch)
	if err != nil {
		return err
	}
	entry := obj("px_file", file, "base", base)
	r.views = append(r.views, entry)
	if !ok {
		entry.Set("status", lookup(got, "status"))
		entry.Set("bindings", 0)
		return nil
	}
	if cut, _ := got.Get("truncated"); cut == true {
		entry.Set("status", "truncated")
		entry.Set("bindings", 0)
		entry.Set("error", "larger than max_bytes: raise max_bytes (up to "+strconv.Itoa(files.MaxBytesLimit)+")")
		return nil
	}
	text, _ := pyjson.DecodeUTF8(body, true)
	bindings, err := px.Bindings(text)
	var ve *pyjson.ValueError
	if errors.As(err, &ve) {
		entry.Set("status", "unparsable")
		entry.Set("bindings", 0)
		entry.Set("error", ve.Msg)
		return nil
	}
	if err != nil {
		return err
	}
	entry.Set("status", "ok")
	baseSlot := ""
	if base != nil {
		baseSlot, _ = bql.BaseSlot(base)
	}
	n := 0
	for _, b := range bindings {
		if r.counts["bindings"] >= r.maxBindings {
			r.truncated = true
			break
		}
		r.counts["bindings"]++
		n++
		if err := r.binding(file, b, px.Absolutize(b.Ord, baseSlot)); err != nil {
			return err
		}
	}
	entry.Set("bindings", n)
	return nil
}
