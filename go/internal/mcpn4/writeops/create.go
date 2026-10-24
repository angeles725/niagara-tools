package writeops

import (
	"math/big"
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// CreateComponent is n4_create_component: one add under parent_ord; the inverse removes the
// component under the name the station assigned (it may rename on collision).
var CreateComponent = plan.Impl{ScopeOrds: func(args *pyjson.Object) []any { return []any{str(args, "parent_ord")} },
	Plan: createPlan, Inverse: createInverse, Readback: createReadback}

// typeSpec is _TYPE_SPEC under re.match: "$" also matches before one final "\n".
var typeSpec = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*:[A-Za-z][A-Za-z0-9_]*\n?$`)

type createData struct {
	parentH any
	ws      any // the wsAnnotation value, or nil
}

// wireSheet is _wire_sheet: the wsAnnotation slot of {x, y, w} (w defaults to 8), or nil.
func wireSheet(v any) (*pyjson.Object, error) {
	ws, ok := v.(*pyjson.Object) // the schema lets only an object (or null) through
	if !ok {
		return nil, nil
	}
	ints := make([]*big.Int, 3)
	for i, k := range []string{"x", "y", "w"} {
		n, ok := ws.Get(k)
		if !ok && k == "w" {
			n = pyjson.IntLiteral("8")
		}
		num, isNum := n.(pyjson.Number) // a bool is never a Number
		if isNum {
			ints[i], ok = num.BigInt() // false for a float
		}
		if !isNum || !ok {
			return nil, plan.ToolError("wire_sheet needs integer x, y and an optional integer w >= 1")
		}
	}
	if ints[2].Sign() < 1 {
		return nil, plan.ToolError("wire_sheet needs integer x, y and an optional integer w >= 1")
	}
	return obj("nm", "p", "n", "wsAnnotation", "t", "baja:WsAnnotation",
		"v", ints[0].String()+","+ints[1].String()+","+ints[2].String()), nil
}

func createPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	nameArg, _ := args.Get("name")
	name, err := plan.Name("name", nameArg)
	if err != nil {
		return nil, err
	}
	typ := str(args, "type")
	if !typeSpec.MatchString(typ) {
		return nil, plan.ToolError(safety.ReasonTypeSpec + " module:Type, got " + pyjson.StrRepr(typ))
	}
	wsArg, _ := args.Get("wire_sheet")
	ws, err := wireSheet(wsArg)
	if err != nil {
		return nil, err
	}
	parentOrd := str(args, "parent_ord")
	h, _, err := plan.Handle(c, parentOrd, 1)
	if err != nil {
		return nil, err
	}
	slots, data := []any{}, createData{parentH: h}
	if ws != nil {
		slots, data.ws = []any{ws}, get(ws, "v")
	}
	targets := pyjson.NewObject()
	targets.Set(parentOrd, h)
	return &plan.Planned{
		Ops:     []any{obj("nm", "a", "h", h, "n", name, "b", obj("nm", "p", "t", typ, "s", slots))},
		Inverse: []any{obj("nm", "v", "h", h, "n", name)},
		Notes:   []any{"The station may rename on collision; the real inverse uses the assigned name."},
		Targets: targets, Data: data}, nil
}

// createInverse is _create_component_inverse: remove the name the station answered ("nn"),
// else the requested one.
func createInverse(p *plan.Planned, replies []any) ([]any, error) {
	nn := get(p.Ops[0].(*pyjson.Object), "n")
	if first, ok := replies[0].([]any); ok && len(first) > 0 {
		if reply, ok := first[0].(*pyjson.Object); ok {
			if v, ok := reply.Get("nn"); ok {
				nn = v
			}
		}
	}
	return []any{obj("nm", "v", "h", p.Data.(createData).parentH, "n", nn)}, nil
}

func createReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	nn := get(inverse[0].(*pyjson.Object), "n")
	requested := obj("name", nn, "type", str(args, "type"), "wsAnnotation", p.Data.(createData).ws)
	nodes, err := c.LoadTree(str(args, "parent_ord"), 2, box.LoadOptions{})
	if err != nil {
		return nil, nil, nil, "", err
	}
	var observed any
	switch name := nn.(type) {
	case []any, *pyjson.Object: // `nn in nodes` hashes it
		return nil, nil, nil, "", unhashable(nn)
	case string:
		if n := box.Node(nodes, name); n != nil {
			observed = obj("name", nn, "type", get(n, "t"), "wsAnnotation", get(box.Node(nodes, name+"/wsAnnotation"), "v"))
		}
	}
	accepted, err := first(replies[0])
	if err != nil {
		return nil, nil, nil, "", err
	}
	return requested, accepted, observed, verdict(observed, requested), nil
}

// first is `r[0] if r else None` for a decoded reply, with Python's errors.
func first(r any) (any, error) {
	if !pyjson.Truthy(r) {
		return nil, nil
	}
	switch x := r.(type) {
	case []any:
		return x[0], nil
	case string:
		_, size := pyjson.DecodeRune(x) // one code point, a lone surrogate too
		return x[:size], nil
	case *pyjson.Object: // its keys are strings: never 0
		return nil, &plan.PyError{Type: "KeyError", Msg: "0"}
	}
	return nil, &plan.PyError{Type: "TypeError", Msg: "'" + plan.TypeName(r) + "' object is not subscriptable"}
}

func unhashable(v any) error {
	t := plan.TypeName(v)
	return &plan.PyError{Type: "TypeError", Msg: "cannot use '" + t + "' as a dict key (unhashable type: '" + t + "')"}
}

// get is o.get(k) on a node or op; nil for a nil o.
func get(o *pyjson.Object, k string) any {
	if o == nil {
		return nil
	}
	v, _ := o.Get(k)
	return v
}

func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}
