package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// DeployTemplate is n4_deploy_template: the component tree of a local component template
// (.ntpl) added under parent_ord in one op, with the template's settings filled from the
// caller's values. Workbench deploys a template client-side (BulkDeployUtil writes the
// components; the station's TemplateService has no deploy action), and so does this tool: it
// reads the manifest and template.bog, checks the settings and the station's modules, and adds
// the tree. The template configuration, links, ORDs naming template handles, passwords, slot
// facets and categories are not carried; the plan notes list them. The inverse removes the
// deployed component under the name the station assigned.
var DeployTemplate = plan.Impl{ScopeOrds: copyScope, Plan: deployPlan, Inverse: deployInverse,
	Readback: deployReadback, Unapplied: deployUnapplied}

type deployData struct {
	parent   any
	name     string
	body     *pyjson.Object
	children map[string]bool // the parent's children when planned
}

func deployPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	path, ok := get(args, "template_file").(string)
	if !ok || path == "" {
		return nil, plan.ToolError("template_file must be the path of a .ntpl file")
	}
	given := pyjson.NewObject()
	if raw, has := args.Get("settings"); has && raw != nil {
		if given, ok = raw.(*pyjson.Object); !ok {
			return nil, plan.ToolError("settings must be an object of setting name -> value")
		}
	}
	arc, err := readTemplateArchive(path)
	if err != nil {
		return nil, err
	}
	m, err := parseTemplateManifest(arc.manifest)
	if err != nil {
		return nil, err
	}
	if err := tplKindAllowed(m); err != nil {
		return nil, err
	}
	bog, err := parseTemplateBog(arc.bog)
	if err != nil {
		return nil, err
	}
	name, err := deployName(args, bog.base)
	if err != nil {
		return nil, err
	}
	parentOrd := str(args, "parent_ord")
	parentH, parentNodes, err := plan.Handle(c, parentOrd, 1)
	if err != nil {
		return nil, err
	}
	if box.Node(parentNodes, name) != nil {
		return nil, plan.ToolError("a child named " + pyjson.StrRepr(name) + " already exists under " + parentOrd)
	}
	depNotes, err := tplDepsInstalled(c, m.deps) // manifest <dependencies> against the station's modules
	if err != nil {
		return nil, err
	}
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, plan.ToolError("a deploy needs the station's type definitions (registry channel) to " +
			"check the template's modules; this client cannot read them")
	}
	handles := tplHandles(bog.base)
	pre, _ := tplBody(bog.base, handles) // modules first: the settings read the types' contracts
	if err := tplModulesInstalled(tl, pre, m.deps); err != nil {
		return nil, err
	}
	exposed, meta, err := tplExposedSettings(m, bog.base, handles)
	if err != nil {
		return nil, err
	}
	lines, err := tplApplySettings(tl, bog.base, exposed, meta, given)
	if err != nil {
		return nil, err
	}
	body, left := tplBody(bog.base, handles)
	targets := pyjson.NewObject()
	targets.Set(parentOrd, parentH)
	return &plan.Planned{
		Ops:     []any{obj("nm", "a", "h", parentH, "n", name, "b", body)},
		Inverse: []any{obj("nm", "v", "h", parentH, "n", name)},
		Notes:   append(deployNotes(m, bog, name, parentOrd, lines, left), depNotes...),
		Targets: targets,
		Data:    deployData{parentH, name, body, childNames(parentNodes)},
	}, nil
}

// tplKindAllowed refuses a station or application template: neither is a component a deploy
// adds under a parent.
func tplKindAllowed(m *tplManifest) error {
	switch {
	case m.isStation:
		return plan.ToolError("template " + pyjson.StrRepr(m.title) + " is a station template " +
			"(isStation=\"true\"): it creates a new station (Workbench New Station wizard), not a component")
	case m.isApplication:
		return plan.ToolError("template " + pyjson.StrRepr(m.title) + " is an application template " +
			"(isApplication=\"true\"): it is installed on the station's Config root with Workbench, not deployed")
	}
	return nil
}

// deployName is the name argument, else the template component's own name (both plain).
func deployName(args *pyjson.Object, base *tplNode) (string, error) {
	if v, ok := args.Get("name"); ok && v != nil {
		return plan.Name("name", v)
	}
	n, err := plan.Name("name", base.name)
	if err != nil {
		return "", plan.ToolError("the template component is named " + pyjson.StrRepr(base.name) +
			", not a plain slot name: pass name")
	}
	return n, nil
}
