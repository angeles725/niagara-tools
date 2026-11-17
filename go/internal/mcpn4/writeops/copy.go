package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// CopyComponent is n4_copy_component: the configuration of one child of parent_ord (type, plain
// slots and child components, CopyDepth levels deep) added again under each new name, all in one
// batch under one confirmation token. Slots flagged readonly, transient, default-on-clone or
// remove-on-clone (stripFlags) and links are not copied; the plan lists them. The inverse removes
// every copy under the name the station assigned.
var CopyComponent = plan.Impl{ScopeOrds: copyScope, Plan: copyPlan, Inverse: copyInverse, Readback: copyReadback}

const (
	// CopyDepth is how many levels below the source a copy captures: a Modbus device reaches its
	// point address at level 5 (points/Point/proxyExt/dataAddress/address).
	CopyDepth = 6
	// maxCopies bounds one batch: a failure at op k leaves k copies to roll back.
	maxCopies = 30
)

type copyData struct {
	parent     any
	sourceType any
	names      []string
	bodies     []*pyjson.Object  // the configured subtree each copy must hold, by new name
	shared     map[string]string // the writable register a copy would share, by new name
}

func copyScope(args *pyjson.Object) []any { return []any{str(args, "parent_ord")} }

func copyPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	names, err := copyNames(args)
	if err != nil {
		return nil, err
	}
	nameArg, _ := args.Get("name")
	source, err := plan.ExistingName("name", nameArg)
	if err != nil {
		return nil, err
	}
	parentOrd := str(args, "parent_ord")
	parentH, parentNodes, err := plan.Handle(c, parentOrd, 1)
	if err != nil {
		return nil, err
	}
	if box.Node(parentNodes, source) == nil { // before the names, as ever: the first refusal wins
		return nil, plan.ToolError("component " + pyjson.StrRepr(source) + " not found under " + parentOrd)
	}
	for _, n := range names {
		if box.Node(parentNodes, n) != nil {
			return nil, plan.ToolError("a child named " + pyjson.StrRepr(n) + " already exists under " + parentOrd)
		}
	}
	src, err := loadCopySource(c, parentOrd, source, parentNodes)
	if err != nil {
		return nil, err
	}
	root, body := src.root, src.body
	bodies, dupNotes, shared, err := copyAddresses(src.ct, root, body, source, names, args)
	if err != nil {
		return nil, err
	}
	ops, inverse := []any{}, []any{}
	for i, n := range names {
		ops = append(ops, obj("nm", "a", "h", parentH, "n", n, "b", bodies[i]))
		inverse = append(inverse, obj("nm", "v", "h", parentH, "n", n))
	}
	notes := append(strippedNotes(src.stripped),
		"The station may rename on collision; the real inverse uses the assigned name.")
	notes = append(notes, src.linksNote()...)
	notes = append(notes, dupNotes...)
	notes = append(notes, fallbackNotes(source, body)...)
	targets := pyjson.NewObject()
	targets.Set(parentOrd, parentH)
	return &plan.Planned{Ops: ops, Inverse: inverse, Notes: notes, Targets: targets,
		Data: copyData{parentH, get(root, "t"), names, bodies, shared}}, nil
}

// copyNames validates new_names: 1 to maxCopies distinct plain slot names.
func copyNames(args *pyjson.Object) ([]string, error) {
	raw, _ := args.Get("new_names")
	list, _ := raw.([]any)
	refuse := plan.ToolError("new_names must list 1 to " + strconv.Itoa(maxCopies) + " distinct plain slot names")
	if len(list) == 0 || len(list) > maxCopies {
		return nil, refuse
	}
	out, seen := []string{}, map[string]bool{}
	for _, v := range list {
		n, err := plan.Name("new_name", v)
		if err != nil {
			return nil, err
		}
		if seen[n] {
			return nil, refuse
		}
		seen[n] = true
		out = append(out, n)
	}
	return out, nil
}

// treeDepth is how many levels of child components a loaded node holds.
func treeDepth(node *pyjson.Object) int {
	kids, _ := get(node, "s").([]any)
	deepest := 0
	for _, k := range kids {
		if kid, ok := k.(*pyjson.Object); ok {
			deepest = max(deepest, 1+treeDepth(kid))
		}
	}
	return deepest
}
