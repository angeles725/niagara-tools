package writeops

// What a link will command once made (delta P2 (c), retro mcpn4-copy-keeps-status Δ2): a link
// drives its source slot's current value into the target at once, so the dry run states that
// value and its status per link, and names the case that commands equipment: an {ok} source
// into a writable's control input. 28 schedule copies at false / 0.0 {ok} would have switched 14
// units OFF and set 14 setpoints to 0 that way. This is volatile state (plan.Planned.Preview):
// it is shown in the dry run and never hashed, so a source that moves keeps the token valid.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// controlInput is a writable point's control inputs: its priority array and fallback.
var controlInput = regexp.MustCompile(`^(in([1-9]|1[0-6])|fallback)$`)

// statusDefaults are the values a station omits from a Status value at its default.
var statusDefaults = map[string]any{"baja:StatusBoolean": "false", "baja:StatusNumeric": "0.0"}

// linkSource is one link_sources entry: the link's four arguments, the source slot's value and
// status (both nil when the load holds no such slot; status nil for a value with no status), and
// a note when an {ok} source is linked into a writable's control input.
func linkSource(args, srcNodes, tgtNodes *pyjson.Object, ss, ts string) *pyjson.Object {
	raw, value, status := sourceValue(srcNodes, ss)
	out := linkEntryView(args, "value", value, "status", status)
	tgtType, _ := get(box.Node(tgtNodes, ""), "t").(string)
	if status == "{ok}" && strings.HasSuffix(tgtType, "Writable") && controlInput.MatchString(ts) {
		out.Set("note", fmt.Sprintf(msgLinkCommands, str(args, "target_ord")+"/"+ts, raw, status))
	}
	return out
}

// sourceValue is the slot's value as the station encodes it, as JSON (a StatusBoolean a bool, a
// StatusNumeric a number) and its status as the station shows it ("{ok}", "{null}",
// "{fault,stale}"); a slot that is no Status value has no status.
func sourceValue(nodes *pyjson.Object, slot string) (raw string, value, status any) {
	n := box.Node(nodes, slot)
	if n == nil {
		return "", nil, nil
	}
	typ, _ := get(n, "t").(string)
	if !statusType(typ) {
		v := get(n, "v")
		raw, _ = v.(string)
		return raw, v, nil
	}
	kids := box.Children(nodes, slot)
	v := box.ChildValue(kids, "value", statusDefaults[typ])
	raw, _ = v.(string)
	value = v
	switch typ {
	case "baja:StatusBoolean":
		value = raw == "true"
	case "baja:StatusNumeric":
		if f, err := plan.Float(v); err == nil {
			value = f
		}
	}
	return raw, value, statusText(box.ChildValue(kids, "status", "0"))
}

// statusText is an encoded status as the station shows it; an encoding it cannot read stays as is.
func statusText(encoded any) any {
	flags, err := box.StatusFlags(encoded)
	if err != nil {
		return encoded
	}
	if len(flags) == 0 {
		return "{ok}"
	}
	return "{" + strings.Join(flags, ",") + "}"
}
