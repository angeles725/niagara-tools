package readtools

import (
	"math"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_list_templates (retro R13 ΔM1): the deployed template instances below a base in one
// read-only call. It composes two BQL reads (the templateConfig components below the base, the
// TemplateService's TemplateInfo rows) with one load per instance (its templateConfig and the
// ConfigBinding children that expose its configs). A value the station does not answer is null
// with a note, never guessed.

// Bounds of n4_list_templates.
const (
	LtDefaultMaxInstances = 200
	ltMaxInfos            = 5000
	ltTimeout             = 60
)

// ltRun is one call: what it noted along the way.
type ltRun struct {
	t     *Tools
	notes []any
}

// ListTemplates is n4_list_templates.
func (t *Tools) ListTemplates(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	slot, err := bql.BaseSlot(base)
	if err != nil {
		return nil, toolError("base must be a station slot ORD such as %s, got %s", pyjson.StrRepr(RootOrd),
			pyjson.StrRepr(base))
	}
	start := t.Ctx.Clock()
	max := intArg(args, "max_instances", LtDefaultMaxInstances)
	r := &ltRun{t: t, notes: []any{}}
	paths, cut, err := r.list(bql.StationSlot+slot, max)
	if err != nil {
		return nil, err
	}
	if cut {
		r.notes = append(r.notes, "instances cut at "+strconv.Itoa(len(paths))+": raise max_instances or narrow base")
	}
	var infos map[string]ltInfo
	if len(paths) > 0 {
		if infos, err = r.infos(); err != nil {
			return nil, err
		}
	}
	instances, unreadable, configs, err := r.readAll(paths, infos)
	if err != nil {
		return nil, err
	}
	out := obj("base", base, "instances", instances, "counts", ltCounts(instances, configs, len(unreadable)),
		"truncated", cut, "unreadable", unreadable, "notes", r.notes,
		"elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_list_templates", "base", args, len(paths))
	return out, nil
}

// readAll loads each instance once; one that cannot be loaded is listed under unreadable
// {root, error}, and an authentication failure fails the call.
func (r *ltRun) readAll(paths []string, infos map[string]ltInfo) ([]any, []any, int, error) {
	instances, unreadable, configs := []any{}, []any{}, 0
	for _, p := range paths {
		ord := "station:|" + p
		nodes, _, err := r.t.timedLoad(ord, ltLoadDepth)
		if err != nil {
			section, fatal := r.t.sectionError(err)
			if fatal != nil {
				return nil, nil, 0, fatal
			}
			unreadable = append(unreadable, obj("root", ltRoot(p), "error", lookup(section, "error")))
			continue
		}
		handle := ""
		if root := box.Node(nodes, ""); root != nil {
			handle, _ = lookup(root, "h").(string)
		}
		info, known := infos[handle]
		entry := ltEntry(ltRoot(p), ord, info, known && handle != "", infos != nil)
		list := ltConfigs(nodes)
		entry.Set("configs", list)
		configs += len(list)
		instances = append(instances, entry)
	}
	return instances, unreadable, configs, nil
}

// ltEntry is one instance: its TemplateInfo row when it has one, else those fields are null.
// withInfo is whether the TemplateInfo rows were read at all (a refusal is noted once, not per
// instance).
func ltEntry(root, configOrd string, info ltInfo, has, withInfo bool) *pyjson.Object {
	e := obj("root", root, "config_ord", configOrd)
	if !has {
		for _, k := range []string{"template_name", "vendor", "version", "inputs", "outputs", "relations",
			"default_passwords", "has_default_passwords", "signature"} {
			e.Set(k, nil)
		}
		if withInfo {
			e.Set("notes", []any{ltNoInfoNote})
		}
		return e
	}
	e.Set("template_name", info.name)
	e.Set("vendor", info.vendor)
	e.Set("version", info.version)
	e.Set("inputs", info.inputs)
	e.Set("outputs", info.outputs)
	e.Set("relations", info.relations)
	e.Set("default_passwords", info.passwords)
	var flag any
	if n, ok := info.passwords.(int); ok {
		flag = n > 0
	}
	e.Set("has_default_passwords", flag)
	// BTemplateInfo.getModifiedState: a template signature of at most one character is unknown
	var modified any
	if len(info.ntpl) > 1 {
		modified = info.ntpl != info.local
	}
	e.Set("signature", obj("template", info.ntpl, "local", info.local, "modified", modified))
	return e
}

// ltCounts is the totals of a reply.
func ltCounts(instances []any, configs, unreadable int) *pyjson.Object {
	passwords, modified := 0, 0
	for _, i := range instances {
		e := i.(*pyjson.Object)
		if lookup(e, "has_default_passwords") == true {
			passwords++
		}
		if sig, ok := lookup(e, "signature").(*pyjson.Object); ok && lookup(sig, "modified") == true {
			modified++
		}
	}
	return obj("instances", len(instances), "configs", configs, "with_default_passwords", passwords,
		"signature_modified", modified, "unreadable", unreadable)
}
