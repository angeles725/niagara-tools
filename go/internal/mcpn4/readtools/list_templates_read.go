package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The station reads of n4_list_templates: the instance list, the TemplateInfo rows and one load
// per instance.

const (
	ltConfigType  = "template:ConfigBinding"
	ltLoadDepth   = 2 // the templateConfig, its configs and the slots of each binding
	ltNoInfoNote  = "no TemplateInfo row for this instance: the TemplateService lists one per registered instance (an ApplicationService's own templateConfig is never registered)"
	ltInfoRefused = "TemplateInfo not read (the station refused it: %s): vendor, version, counts and signatures are null"
)

// ltInfo is one TemplateInfo row, kept as the station answered it.
type ltInfo struct {
	name, vendor, version, ntpl, local    string
	inputs, outputs, relations, passwords any
}

// list is the templateConfig slot paths below scope (at most max); a station that refuses the
// query has no instance list, which is an error and never an empty list.
func (r *ltRun) list(scope string, max int) ([]string, bool, error) {
	columns, raw, cut, _, err := r.t.query(scope, bql.TemplateConfigQuery, max, ltTimeout)
	if err != nil {
		section, fatal := r.t.sectionError(err)
		if fatal != nil {
			return nil, false, fatal
		}
		return nil, false, toolError("no instance list: the station refused %s (%s): is the template module "+
			"installed (a TemplateService under Services)?", bql.TemplateConfigQuery, pyjson.Str(lookup(section, "error")))
	}
	rows, err := bql.ByPosition(columns, raw, []string{"path"})
	if err != nil {
		return nil, false, err
	}
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, cellText(row, "path"))
	}
	return paths, cut, nil
}

// infos reads the TemplateInfo rows by the handle of the templateConfig's root; a refused query
// leaves them out with a note.
func (r *ltRun) infos() (map[string]ltInfo, error) {
	columns, raw, _, _, err := r.t.query(bql.StationSlot+"/", bql.TemplateInfoQuery, ltMaxInfos, ltTimeout)
	if err != nil {
		section, fatal := r.t.sectionError(err)
		if fatal != nil {
			return nil, fatal
		}
		r.notes = append(r.notes, strings.Replace(ltInfoRefused, "%s", pyjson.Str(lookup(section, "error")), 1))
		return nil, nil
	}
	rows, err := bql.ByPosition(columns, raw, bql.TemplateInfoFields)
	if err != nil {
		return nil, err
	}
	out := map[string]ltInfo{}
	for _, row := range rows {
		out[ltHandle(cellText(row, "location"))] = ltInfo{
			name: cellText(row, "name"), vendor: cellText(row, "vendor"), version: cellText(row, "version"),
			ntpl: cellText(row, "ntpl"), local: cellText(row, "local"), inputs: saCount(cellText(row, "inputs")),
			outputs: saCount(cellText(row, "outputs")), relations: saCount(cellText(row, "relations")),
			passwords: saCount(cellText(row, "passwords")),
		}
	}
	return out, nil
}

// ltHandle is the handle in a handle ORD ("h:1a2b", "station:|h:1a2b" and "1a2b" are all "1a2b").
func ltHandle(ord string) string {
	ord = strings.TrimSpace(ord)
	return strings.TrimPrefix(ord[strings.LastIndex(ord, "|")+1:], "h:")
}

// ltRoot is the ORD of the template root: the parent of the templateConfig at slot path path.
func ltRoot(path string) string {
	parent := path[:max(strings.LastIndex(path, "/"), 0)]
	if parent == "slot:" || parent == "" {
		parent = "slot:/"
	}
	return "station:|" + parent
}

// configs is the exposed configs of the loaded templateConfig: one per ConfigBinding child, its
// source slot's value (null with a note when the station's answer lacks the slot), and the slot
// it is bound to as the binding stores it.
func ltConfigs(nodes *pyjson.Object) []any {
	out := []any{}
	for _, name := range ltBindings(nodes) {
		kids := box.Children(nodes, name)
		source := cellOf(kids, "sourceSlot")
		entry := obj("name", source, "value", nil, "type", nil,
			"bound_to", obj("ord", cellOf(kids, "targetOrd"), "slot", cellOf(kids, "targetSlot")))
		if tip := cellOf(kids, "userTip"); tip != "" {
			entry.Set("tip", tip)
		}
		note := "sourceSlot is not set on the binding"
		if source != "" {
			note = source + " is not in the station's answer (a slot at its default is left out of a load)"
			if box.Node(box.Children(nodes, ""), source) != nil {
				if slot, err := slotEntry(nodes, source); err == nil {
					entry.Set("value", lookup(slot, "value"))
					entry.Set("type", lookup(slot, "type"))
					note = ""
				} else {
					note = source + " could not be decoded: " + err.Error()
				}
			}
		}
		if note != "" {
			entry.Set("note", note)
		}
		out = append(out, entry)
	}
	return out
}

// ltBindings is the names of the ConfigBinding children of the loaded templateConfig.
func ltBindings(nodes *pyjson.Object) []string {
	names := []string{}
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		if kind, _ := box.Node(kids, name).Get("t"); kind == ltConfigType {
			names = append(names, name)
		}
	}
	return names
}

// cellOf is the string value of the child name of a Children result ("" when the station left
// the slot at its default out).
func cellOf(kids *pyjson.Object, name string) string {
	s, _ := box.ChildValue(kids, name, "").(string)
	return s
}
