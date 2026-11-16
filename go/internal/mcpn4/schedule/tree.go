package schedule

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Defaults is the declared default, in BSON ({nm, t, v[, s]}), of the frozen slot name of
// typeSpec or one of its supertypes; nil when none declares it (plan.DeclaredDefault).
type Defaults func(typeSpec, name string) (*pyjson.Object, error)

// tree reads a LoadTree result ({path: node}) of one schedule, filling every slot the station
// left out because it held its type's default.
type tree struct {
	nodes *pyjson.Object
	def   Defaults
}

func (t *tree) node(path string) *pyjson.Object { return box.Node(t.nodes, path) }

func (t *tree) typeOf(path string) string { s, _ := field(t.node(path), "t").(string); return s }

// kids are the direct children of path that are components of a type in module "schedule",
// in slot order, as their paths.
func (t *tree) schedules(path string) []string {
	var out []string
	kids := box.Children(t.nodes, path)
	for _, name := range kids.Keys() {
		if s, _ := field(box.Node(kids, name), "t").(string); strings.HasPrefix(s, "schedule:") {
			out = append(out, join(path, name))
		}
	}
	return out
}

// slot is the node of path/name, or its declared default when the station left it out. A slot
// that neither the station sent nor the type registry declares is an error: the decoder never
// guesses a value.
func (t *tree) slot(path, name string) (*pyjson.Object, error) {
	if n := t.node(join(path, name)); n != nil {
		return n, nil
	}
	d, err := t.def(t.typeOf(path), name)
	if err == nil && d == nil {
		err = fmt.Errorf("the station left out %s of %s (%s) and its type registry does not describe "+
			"that slot", name, displayPath(path), t.typeOf(path))
	}
	return d, err
}

// simple is the encoded value ("v") of the simple slot path/name ("" when it has none).
func (t *tree) simple(path, name string) (string, error) {
	n, err := t.slot(path, name)
	if err != nil {
		return "", err
	}
	v, _ := field(n, "v").(string)
	return v, nil
}

func displayPath(path string) string {
	if path == "" {
		return "the schedule"
	}
	return path
}

// status is the Status value at path/name as {value, status}: the value plain (bool, number,
// ordinal, string; nil when the status is null) and the status "ok", "null" or its encoding.
// A slot that neither the station nor the type has is nil.
func (t *tree) status(path, name string) (*pyjson.Object, error) {
	n, err := t.slot(path, name)
	if err != nil {
		return nil, err
	}
	typ, _ := field(n, "t").(string)
	return statusValue(typ, kidValue(n, "value", ""), kidValue(n, "status", "0"))
}

func statusValue(typ, value, status string) (*pyjson.Object, error) {
	ok, null, err := box.ParseStatus(status)
	if err != nil {
		return nil, err
	}
	label := status
	switch {
	case ok:
		label = "ok"
	case null && strings.SplitN(status, ";", 2)[0] == "40":
		label = "null"
	}
	var v any
	if !null {
		v = plain(typ, value)
	}
	return obj("value", v, "status", label), nil
}

// plain is an encoded Status value as JSON: StatusBoolean true/false, StatusNumeric a number,
// StatusEnum its ordinal, anything else the text.
func plain(typ, value string) any {
	switch typ {
	case "baja:StatusBoolean":
		return value == "true"
	case "baja:StatusNumeric":
		if f, err := strconv.ParseFloat(value, 64); err == nil || value == "" {
			return f
		}
	case "baja:StatusEnum":
		head, _, _ := strings.Cut(value, "@")
		if n, err := strconv.Atoi(head); err == nil || value == "" {
			return n
		}
	}
	return value
}

// kidValue is the "v" of the child named name inside a BSON body's "s", def when absent.
func kidValue(body *pyjson.Object, name, def string) string {
	subs, _ := field(body, "s").([]any)
	for _, s := range subs {
		if k, ok := s.(*pyjson.Object); ok && field(k, "n") == name {
			if v, ok := field(k, "v").(string); ok {
				return v
			}
		}
	}
	return def
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "/" + name
}

func field(o *pyjson.Object, k string) any {
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
