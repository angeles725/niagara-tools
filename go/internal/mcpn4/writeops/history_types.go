package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// historyFamilies are the control point types a history extension goes under (history-rt
// B<Kind>IntervalHistoryExt.isParentLegal: parent instanceof B<Kind>Point) and the kind word of
// the matching extension types, history:<Kind>IntervalHistoryExt.
var historyFamilies = []struct{ point, kind string }{
	{"control:NumericPoint", "Numeric"}, {"control:BooleanPoint", "Boolean"},
	{"control:EnumPoint", "Enum"}, {"control:StringPoint", "String"},
}

// historyExtType is the extension type of kind and mode (Interval or Cov), and its default slot
// name.
func historyExtType(kind, mode string) (typ, name string) {
	return "history:" + kind + mode + "HistoryExt", kind + mode
}

// pointKinds finds the kind of each point type from the station's type registry, by walking
// the supertypes (never by the type's name); a batch asks once per type.
type pointKinds struct {
	tl    plan.TypeLoader
	types *pyjson.Object
	kinds map[string]string
}

func newPointKinds(c plan.Client) *pointKinds {
	tl, _ := c.(plan.TypeLoader)
	return &pointKinds{tl: tl, types: pyjson.NewObject(), kinds: map[string]string{}}
}

// kind is the family kind of spec, "" when spec is not a numeric, boolean, enum or string point.
func (k *pointKinds) kind(spec string) (string, error) {
	if kind, ok := k.kinds[spec]; ok {
		return kind, nil
	}
	if k.tl == nil {
		return "", fmt.Errorf("this client cannot read it")
	}
	kind := ""
	for i, t := 0, spec; i < maxSupertypes && t != "" && kind == ""; i++ {
		for _, f := range historyFamilies {
			if t == f.point {
				kind = f.kind
			}
		}
		if box.Node(k.types, t) == nil {
			if err := k.load(t); err != nil {
				return "", err
			}
		}
		t = box.Supertype(k.types, t)
	}
	k.kinds[spec] = kind
	return kind, nil
}

func (k *pointKinds) load(spec string) error {
	reply, err := k.tl.LoadTypes([]string{spec})
	if err != nil {
		return err
	}
	got, _ := reply.(*pyjson.Object)
	if got == nil {
		return fmt.Errorf("its answer is not an object")
	}
	for _, key := range got.Keys() {
		v, _ := got.Get(key)
		k.types.Set(key, v)
	}
	return nil
}

// maxSupertypes bounds the walk up a point type's supertypes.
const maxSupertypes = 32

// forPoint is spec for the point at ord, of type pointType: the extension type follows the
// point's kind and the mode; a given ext_type must be that one; the name, when not given, is the
// type's short name (BooleanInterval).
func (s historySpec) forPoint(k *pointKinds, ord string, pointType any) (historySpec, error) {
	spec, _ := pointType.(string)
	kind, err := k.kind(spec)
	if err != nil {
		return s, plan.ToolError(fmt.Sprintf(msgHistoryNoRegistry, ord, err.Error()))
	}
	if kind == "" {
		return s, plan.ToolError(fmt.Sprintf(msgHistoryNotAPoint, ord, pyjson.Str(pointType)))
	}
	typ, name := historyExtType(kind, s.mode)
	if s.typ != "" && s.typ != typ {
		return s, plan.ToolError(fmt.Sprintf(msgHistoryTypeMismatch, s.typ, ord, strings.ToLower(kind), spec, typ))
	}
	s.ord, s.typ = ord, typ
	if s.name == "" {
		s.name = name
	}
	return s, checkTolerance(s, kind, spec)
}
