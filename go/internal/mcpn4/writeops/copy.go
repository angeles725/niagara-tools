package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// CopyComponent is n4_copy_component: the configuration of one child of parent_ord (type, plain
// slots and child components, CopyDepth levels deep) added again under each new name, all in one
// batch under one confirmation token. Runtime slots and links are not copied. The inverse removes
// every copy under the name the station assigned.
var CopyComponent = plan.Impl{ScopeOrds: copyScope, Plan: copyPlan, Inverse: copyInverse, Readback: copyReadback}

const (
	// CopyDepth is how many levels below the source a copy captures: a Modbus device reaches its
	// point address at level 5 (points/Point/proxyExt/dataAddress/address).
	CopyDepth = 6
	// maxCopies bounds one batch: a failure at op k leaves k copies to roll back.
	maxCopies = 30
)

// copyRuntimeSlots are slots the driver or the control engine computes; a copy starts without
// them so it never shows the source's state (a down status, a failed ping) as its own.
// absoluteAddress is a Modbus proxy's read-only transient address (flags 3), derived by the driver
// from dataAddress (modbusCore BModbusClientProxyExt).
var copyRuntimeSlots = map[string]bool{"status": true, "health": true, "faultCause": true,
	"readValue": true, "writeValue": true, "absoluteAddress": true}

type copyData struct {
	parent     any
	sourceType any
	names      []string
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
	if box.Node(parentNodes, source) == nil {
		return nil, plan.ToolError("component " + pyjson.StrRepr(source) + " not found under " + parentOrd)
	}
	for _, n := range names {
		if box.Node(parentNodes, n) != nil {
			return nil, plan.ToolError("a child named " + pyjson.StrRepr(n) + " already exists under " + parentOrd)
		}
	}
	childOrd, err := ord.ChildOrd(parentOrd, source)
	if err != nil {
		return nil, err
	}
	// One level more than the copy keeps: anything there means the copy would be partial.
	_, nodes, err := plan.Handle(c, childOrd, CopyDepth+1)
	if err != nil {
		return nil, err
	}
	root := box.Node(nodes, "")
	if treeDepth(root) > CopyDepth {
		return nil, plan.ToolError(pyjson.StrRepr(source) + " is deeper than the copy depth (" +
			strconv.Itoa(CopyDepth) + " levels): a copy would miss part of it; copy a smaller component")
	}
	s := &snapshot{handles: map[string]string{}}
	body, err := s.take(root, "", CopyDepth)
	if err != nil {
		return nil, err
	}
	body = stripRuntime(body)
	ops, inverse := []any{}, []any{}
	for _, n := range names {
		ops = append(ops, obj("nm", "a", "h", parentH, "n", n, "b", body))
		inverse = append(inverse, obj("nm", "v", "h", parentH, "n", n))
	}
	notes := []any{"Each copy starts without runtime slots (status, health, out, read and write values); " +
		"the driver fills them once the copy runs.",
		"The station may rename on collision; the real inverse uses the assigned name."}
	if len(s.links) > 0 {
		notes = append(notes, strconv.Itoa(len(s.links))+" link(s) inside "+pyjson.StrRepr(source)+
			" are not copied: create them on each copy with n4_create_link")
	}
	targets := pyjson.NewObject()
	targets.Set(parentOrd, parentH)
	return &plan.Planned{Ops: ops, Inverse: inverse, Notes: notes, Targets: targets,
		Data: copyData{parentH, get(root, "t"), names}}, nil
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

// stripRuntime is body without copyRuntimeSlots, at every level, and without children the load
// lists by name only (nameOnly) (actions such as upload, frozen slots at their default such as regType):
// an add op child without a type is not a value, and the copy's type supplies those slots.
func stripRuntime(body *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k != "s" {
			out.Set(k, v)
			continue
		}
		kept := []any{}
		for _, c := range v.([]any) {
			kid := c.(*pyjson.Object)
			if n, _ := get(kid, "n").(string); !copyRuntimeSlots[n] && !nameOnly(kid) {
				kept = append(kept, stripRuntime(kid))
			}
		}
		if len(kept) > 0 {
			out.Set("s", kept)
		}
	}
	return out
}

// nameOnly is a loaded child with no type, value or children: an action or a frozen slot at its
// default, listed by name only.
func nameOnly(kid *pyjson.Object) bool {
	return get(kid, "t") == nil && get(kid, "v") == nil && get(kid, "s") == nil
}

// copyInverse removes each copy under the name its reply assigned ("nn"), else the requested one.
func copyInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(copyData)
	out := []any{}
	for i := range replies {
		nn := any(d.names[i])
		if first, ok := replies[i].([]any); ok && len(first) > 0 {
			if reply, ok := first[0].(*pyjson.Object); ok {
				if v, ok := reply.Get("nn"); ok {
					nn = v
				}
			}
		}
		out = append(out, obj("nm", "v", "h", d.parent, "n", nn))
	}
	return out, nil
}

// copyReadback finds every copy under its assigned name with the source's type.
func copyReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(copyData)
	_, nodes, err := plan.Handle(c, str(args, "parent_ord"), 1)
	if err != nil {
		return nil, nil, nil, "", err
	}
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i := range d.names {
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		requested = append(requested, obj("name", nn, "type", d.sourceType))
		var t any
		if n := box.Node(nodes, nn); n != nil {
			t = get(n, "t")
		}
		observed = append(observed, obj("name", nn, "type", t))
		if !box.PyEqual(t, d.sourceType) {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}
