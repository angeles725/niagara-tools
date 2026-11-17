package writeops

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// tplIntValue is a value an integer slot encodes (baja:Integer "2", never "2.0").
var tplIntValue = regexp.MustCompile(`^-?[0-9]+$`)

// tplApplySettings checks the caller's settings (name -> value) against the exposed settings
// and writes each exposed setting's value (given, else the template's) into its slot in base.
// An unknown name, a metadata setting, a value of the wrong type or out of [min, max], and a
// required setting the template leaves empty with no value given are refused. A missing target
// slot is created from the holder type's declared default. It returns one line per setting.
func tplApplySettings(tl plan.TypeLoader, base *tplNode, exposed []*tplExposed, meta map[string]bool, given *pyjson.Object) ([]string, error) {
	byName := map[string]*tplExposed{}
	for _, e := range exposed {
		byName[e.name] = e
	}
	for _, k := range given.Keys() {
		if meta[k] {
			return nil, plan.ToolError("setting " + pyjson.StrRepr(k) + " is the template's own metadata, " +
				"not a deploy-time setting")
		}
		if byName[k] == nil {
			return nil, plan.ToolError("unknown setting " + pyjson.StrRepr(k) + "; the template's settings are: " +
				tplNames(exposed))
		}
	}
	lines := []string{}
	for _, e := range exposed {
		raw, isGiven := given.Get(e.name)
		if e.pseudo {
			if isGiven {
				return nil, plan.ToolError("setting " + pyjson.StrRepr(e.name) + " writes " + e.slot +
					" (a name attribute), which this tool does not set; pass name instead")
			}
			lines = append(lines, e.name+" (writes "+e.slot+") not applied")
			continue
		}
		value, src := e.deflt, "template"
		if isGiven {
			v, err := tplEncode(e, raw)
			if err != nil {
				return nil, err
			}
			value, src = v, "given"
		} else if e.required && (!e.hasDefault || value == "" || value == "nan") {
			return nil, plan.ToolError("setting " + pyjson.StrRepr(e.name) + " is required and the template " +
				"holds no value for it; pass it in settings")
		} else if !e.hasDefault {
			lines = append(lines, e.name+" (no template value) left as the slot's default")
			continue
		}
		target, err := tplTarget(tl, base, e)
		if err != nil {
			return nil, err
		}
		setSlotValue(target, value)
		lines = append(lines, e.name+"="+value+" ("+src+tplBounds(e)+") -> "+joinPath(e.holder, e.slot)+nullStatus(target))
	}
	return lines, nil
}

// tplEncode checks a given value against the setting's type and bounds and encodes it as the
// station writes the slot.
func tplEncode(e *tplExposed, raw any) (string, error) {
	bad := func(want string) error {
		return plan.ToolError("setting " + pyjson.StrRepr(e.name) + " must be " + want + ", got " + pyjson.Repr(raw))
	}
	switch e.typ {
	case "num":
		n, ok := raw.(pyjson.Number)
		if !ok {
			return "", bad("a number")
		}
		f, err := n.Float64()
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return "", bad("a finite number")
		}
		if e.minText != "" && f < e.min || e.maxText != "" && f > e.max {
			return "", plan.ToolError("setting " + pyjson.StrRepr(e.name) + " must be between " +
				orDash(e.minText) + " and " + orDash(e.maxText) + ", got " + pyjson.Repr(raw))
		}
		if tplIntegerSlot(e) {
			if f != math.Trunc(f) {
				return "", bad("a whole number")
			}
			return strconv.FormatFloat(f, 'f', 0, 64), nil
		}
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.ContainsAny(s, ".") {
			s += ".0"
		}
		return s, nil
	case "bool":
		b, ok := raw.(bool)
		if !ok {
			return "", bad("true or false")
		}
		return strconv.FormatBool(b), nil
	case "str":
		s, ok := raw.(string)
		if !ok {
			return "", bad("a string")
		}
		return s, nil
	}
	return "", plan.ToolError("setting " + pyjson.StrRepr(e.name) + " has type " + e.typ +
		", which this tool does not set (num, bool and str only)")
}

// tplIntegerSlot is a num setting whose slot holds an integer: an Integer or Long property, or a
// template value written without a decimal point by a type that is not Double or Float.
func tplIntegerSlot(e *tplExposed) bool {
	switch e.srcType {
	case "baja:Integer", "baja:Long":
		return true
	case "baja:Double", "baja:Float", "baja:StatusNumeric":
		return false
	}
	return e.hasDefault && tplIntValue.MatchString(e.deflt)
}

// tplTarget is the slot node e writes, created from the holder's declared default when the
// template does not hold it (a frozen slot left at its default).
func tplTarget(tl plan.TypeLoader, base *tplNode, e *tplExposed) (*tplNode, error) {
	holder := tplAt(base, e.holder)
	missing := plan.ToolError("setting " + pyjson.StrRepr(e.name) + ": its slot " + joinPath(e.holder, e.slot) +
		" is neither in template.bog nor declared by the station's type")
	if holder == nil {
		return nil, missing
	}
	if t := holder.child(e.slot); t != nil {
		return t, nil
	}
	if holder.typ == "" {
		return nil, missing
	}
	def, err := plan.DeclaredDefault(tl, holder.typ, e.slot)
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, missing
	}
	t := fromBSON(e.slot, def)
	holder.kids = append(holder.kids, t)
	return t, nil
}

// nullStatus warns that a Status struct written by a setting keeps a null status (bit 0x40, as a
// BooleanWritable's declared fallback holds it): the value is there but the point ignores it.
func nullStatus(n *tplNode) string {
	st := n.child("status")
	if st == nil || !tplStatusStruct(n) {
		return ""
	}
	bits, err := strconv.ParseUint(strings.SplitN(st.value, ";", 2)[0], 16, 32)
	if err != nil || bits&0x40 == 0 {
		return ""
	}
	return " [status null (" + st.value + "): clear it with n4_set_slot for the value to apply]"
}

func tplBounds(e *tplExposed) string {
	out := ""
	if e.units != "" {
		out += "; " + e.units
	}
	if e.minText != "" || e.maxText != "" {
		out += "; " + orDash(e.minText) + ".." + orDash(e.maxText)
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func tplNames(exposed []*tplExposed) string {
	names := []string{}
	for _, e := range exposed {
		names = append(names, e.name)
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}
