package plan

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
)

// ComponentOracle says whether a type spec is a component (the station creates it with its own add
// op) or a slot value (a struct or a simple). known is false when it cannot tell, and the caller
// then falls back to the naming rule (box.IsComponentType).
type ComponentOracle interface {
	IsComponent(spec any) (yes, known bool)
}

// ClassifyComponent is box.IsComponentType improved by o: the oracle's answer when it has one, else
// the naming rule. A nil oracle is the naming rule alone (a client without a registry channel).
func ClassifyComponent(o ComponentOracle, spec any) bool {
	if o != nil {
		if yes, known := o.IsComponent(spec); known {
			return yes
		}
	}
	return box.IsComponentType(spec)
}

// RegistryOracle classifies by the supertype chain the station's registry describes (B1200-G3: the
// alarm:LimitEnable of an out-of-range algorithm is a baja:Struct, a value the naming rule takes for
// a component). Order: the static table first, then the registry; a type the registry does not
// describe (an answer {}, an unanswered type, a chain cut short) is left to the naming rule and noted.
type RegistryOracle struct {
	ct       *Contracts
	answers  map[string][2]bool // spec -> {is, known}
	fallback map[string]bool    // the types the naming rule decided
}

// A nil *RegistryOracle answers nothing (every type is left to the naming rule).

// NewRegistryOracle classifies through ct, loading any type it has not held yet on demand.
func NewRegistryOracle(ct *Contracts) *RegistryOracle {
	return &RegistryOracle{ct: ct, answers: map[string][2]bool{}, fallback: map[string]bool{}}
}

// Preload reads every spec in one registry call; a failed call or an unanswered type only leaves
// that type to the on-demand lookup (and then the naming rule).
func (o *RegistryOracle) Preload(specs []string) { _ = o.ct.Load(specs) }

// IsComponent is ComponentOracle: known only for a type the table lists or whose chain the
// registry describes completely.
func (o *RegistryOracle) IsComponent(spec any) (yes, known bool) {
	if o == nil { // a client without a registry channel: the naming rule decides
		return false, false
	}
	s, ok := spec.(string)
	if !ok || !strings.Contains(s, ":") {
		return false, true
	}
	if c, listed := box.TableComponentType(s); listed {
		return c, true
	}
	if a, seen := o.answers[s]; seen {
		return a[0], a[1]
	}
	yes, known = o.classify(s)
	o.answers[s] = [2]bool{yes, known}
	if !known {
		o.fallback[s] = true
	}
	return yes, known
}

func (o *RegistryOracle) classify(s string) (yes, known bool) {
	is, complete, err := o.ct.IsKnown(s, "baja:Component")
	if err != nil || (!is && !complete) {
		return false, false
	}
	if is {
		return true, true
	}
	e, _ := lookup(o.ct.types, s)
	if super, _ := field(e, "p").(string); super == "" {
		return false, false // an entry {}: the station did not describe the type
	}
	return false, true
}

// Fallbacks are the types the naming rule decided, sorted: the ones outside the baja module, where
// the rule is a guess (a baja type is a value by the rule, which the station's common types are).
func (o *RegistryOracle) Fallbacks() []string {
	out := []string{}
	for s := range o.fallback {
		if !strings.HasPrefix(s, "baja:") {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// FallbackNote is the plan note naming the types the naming rule classified ("" when none).
func (o *RegistryOracle) FallbackNote() string {
	if o == nil {
		return ""
	}
	fb := o.Fallbacks()
	if len(fb) == 0 {
		return ""
	}
	return "The station's type registry does not describe " + strings.Join(fb, ", ") + ", so the naming rule " +
		"(every type outside the baja module is a component) classified them: a struct among them would be " +
		"re-created as a component"
}

// IsStruct says whether the registry describes spec as a baja:Struct (its supertype chain reaches
// it): a value made of slots that belongs to the body of the component holding it. false when the
// registry does not say (a nil oracle, an unanswered type, a chain cut short).
func (o *RegistryOracle) IsStruct(spec any) bool {
	s, ok := spec.(string)
	if o == nil || !ok || s == "" {
		return false
	}
	if _, listed := box.TableComponentType(s); listed {
		return false
	}
	is, _, err := o.ct.IsKnown(s, "baja:Struct")
	return err == nil && is
}
