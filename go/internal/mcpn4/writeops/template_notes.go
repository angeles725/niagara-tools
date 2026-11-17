package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// deployNotes are the dry run's account of a deploy: the template, its dependencies, the
// resolved settings and everything the body leaves out.
func deployNotes(m *tplManifest, bog *tplBog, name, parentOrd string, settings []string, left *tplLeftOut) []any {
	b := bog.base
	notes := []any{"Template " + pyjson.StrRepr(m.title) + " version " + orDash(m.version) + " (vendor " +
		orDash(m.vendor) + ", id " + orDash(m.id) + "): deploys " + pyjson.StrRepr(b.name) + " (" + b.typ +
		") as " + pyjson.StrRepr(name) + " under " + parentOrd}
	deps := "(none listed)"
	if len(m.deps) > 0 {
		deps = strings.Join(m.deps, ", ")
	}
	notes = append(notes, "Dependencies (manifest): "+deps+"; the station's type registry describes every "+
		"type the deploy adds")
	if len(settings) > 0 {
		notes = append(notes, "Settings: "+strings.Join(settings, ", "))
	}
	if bog.rootKids > 1 {
		notes = append(notes, "template.bog holds "+strconv.Itoa(bog.rootKids)+" components; only the "+
			"first, the template component, is deployed")
	}
	if cfg, _ := tplConfig(b); cfg != nil {
		notes = append(notes, "Not deployed: "+cfg.name+" ("+tplConfigType+"): its bindings name components "+
			"by template handle, which on the station would point at unrelated components; the deployed "+
			"component holds the settings' values but is not linked to the template (a Workbench upgrade "+
			"does not see it)")
	}
	if len(left.links) > 0 {
		items := []string{}
		for _, l := range left.links {
			items = append(items, l.source+"."+l.sourceSlot+" -> "+l.holder+"."+l.targetSlot)
		}
		notes = append(notes, strconv.Itoa(len(left.links))+" link(s) inside the template are not deployed: "+
			strings.Join(items, ", ")+"; create them on the deployed component with n4_create_link")
	}
	for _, g := range []struct {
		list []string
		why  string
	}{
		{left.handleOrds, "ORD values naming a template handle are left out (they would name unrelated station components): "},
		{left.passwords, "Passwords are left out (the template's encoding is not this station's): "},
		{left.facets, "Slot facets (x) are not carried: "},
		{left.cats, "Categories (c) are not carried: "},
	} {
		if len(g.list) > 0 {
			notes = append(notes, g.why+strings.Join(g.list, ", "))
		}
	}
	return append(notes, "The station may rename on collision; the real inverse uses the assigned name.")
}
