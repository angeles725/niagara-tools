package readtools

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/semantic"
)

// n4_semantic_export (opportunity O10): the hs: tags of the components a NEQL query matches, as a
// Project Haystack grid in the Haystack 4 JSON encoding (Hayson).

// Bounds and defaults of n4_semantic_export.
const (
	SemanticDefaultNeql     = "hs:site or hs:space or hs:equip or hs:point"
	SemanticDefaultEntities = 100
	exportSubdir            = "exports"
)

var exportFile = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}\.json$`)

// SemanticExport is n4_semantic_export.
func (t *Tools) SemanticExport(args *pyjson.Object) (*pyjson.Object, error) {
	start := t.Ctx.Clock()
	q := obj("neql", strArg(args, "neql", SemanticDefaultNeql), "base_ord", strArg(args, "base_ord", RootOrd),
		"max_rows", pyjson.IntLiteral(strconv.Itoa(intArg(args, "max_entities", SemanticDefaultEntities))),
		"timeout_s", pyjson.IntLiteral(strconv.Itoa(intArg(args, "timeout_s", 60))))
	found, err := t.NeqlQuery(q)
	if err != nil {
		return nil, err
	}
	implied := true
	if v, ok := args.Get("implied"); ok {
		implied = v == true
	}
	ords, _ := lookup(found, "ords").([]any)
	entities := make([]semantic.Entity, 0, len(ords))
	for _, o := range ords {
		ord, _ := o.(string)
		entry, err := t.tagEntry(ord, implied, "direct")
		if err != nil {
			return nil, err
		}
		entities = append(entities, entityOf(ord, entry))
	}
	grid, skipped := semantic.Grid(entities)
	out := obj("neql", lookup(found, "neql"), "base_ord", lookup(found, "base_ord"), "entities", len(entities),
		"truncated", lookup(found, "truncated"), "skipped", skippedList(skipped))
	if _, ok := t.Ctx.Session.Client.(sessionCaller); implied && !ok {
		out.Set("implied_unavailable", msgImpliedUnavailable)
	}
	data, err := json.Marshal(grid) // map keys sorted: a stable file
	if err != nil {
		return nil, err
	}
	if name := strArg(args, "save_as", ""); name != "" {
		path, err := t.outputPath(name+".json", exportSubdir, exportFile, "(save_as) ")
		if err != nil {
			return nil, err
		}
		if err := writePrivate(path, func(f *os.File) error { _, err := f.Write(data); return err }); err != nil {
			return nil, writeFailed(path, err)
		}
		out.Set("path", path)
		out.Set("bytes", len(data))
	} else {
		inline, err := pyjson.Unmarshal(data)
		if err != nil {
			return nil, err
		}
		out.Set("grid", inline)
	}
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	return out, nil
}

// entityOf is the export entity of one n4_read_tags entry: direct and implied tags (a direct tag
// wins over an implied one with the same id) and direct relations; dis is hs:dis when set,
// else the slot name.
func entityOf(ord string, entry *pyjson.Object) semantic.Entity {
	e := semantic.Entity{Ord: ord, Dis: ord[strings.LastIndex(ord, "/")+1:]}
	seen := map[string]bool{}
	for _, key := range []string{"direct", "implied"} {
		list, _ := lookup(entry, key).([]any)
		for _, raw := range list {
			tag, _ := raw.(*pyjson.Object)
			id, _ := lookup(tag, "id").(string)
			typ, _ := lookup(tag, "type").(string)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			v := lookup(tag, "value")
			if id == "hs:dis" {
				if s, ok := v.(string); ok && s != "" {
					e.Dis = s
				}
			}
			e.Tags = append(e.Tags, semantic.Tag{ID: id, Type: typ, Value: v})
		}
	}
	rels, _ := lookup(lookup(entry, "relations").(*pyjson.Object), "direct").([]any)
	for _, raw := range rels {
		r, _ := raw.(*pyjson.Object)
		id, _ := lookup(r, "id").(string)
		end, _ := lookup(r, "endpoint_ord").(string)
		if lookup(r, "inbound") == true {
			continue // an inbound relation belongs to the other end's row
		}
		e.Relations = append(e.Relations, semantic.Relation{ID: id, Endpoint: end})
	}
	return e
}

func skippedList(s []semantic.Skipped) []any {
	out := []any{}
	for _, x := range s {
		out = append(out, obj("ord", x.Ord, "tag", x.Tag, "why", x.Why))
	}
	return out
}
