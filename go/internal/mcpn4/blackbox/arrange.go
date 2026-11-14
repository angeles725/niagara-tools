package blackbox

import (
	"fmt"
	"strconv"
)

// This file builds station state through the fake's arranging API (box_client in the kit):
// components under the Folder and their status slots, failing the test on a fake error.

// Component is a component the test added: its name and handle.
type Component struct{ Name, Handle string }

// Add adds a component of typ (kitControl:NumericConst when "") under the component with handle
// parent ("3" is the Folder).
func (r *Rig) Add(parent, name, typ string) Component {
	r.T.Helper()
	if typ == "" {
		typ = "kitControl:NumericConst"
	}
	nn, h, err := r.Station.Add(parent, name, typ)
	if err != nil {
		r.T.Fatal(err)
	}
	return Component{nn, h}
}

// AddWithOut adds a NumericConst under the Folder whose out slot holds value with status ok.
func (r *Rig) AddWithOut(name string, value float64) Component {
	r.T.Helper()
	c := r.Add("3", name, "")
	r.Set(c.Handle, "out", StatusNumeric(value, "0"))
	return c
}

// Set writes a BOX body to a slot of the component with handle h.
func (r *Rig) Set(h, slot string, body map[string]any) {
	r.T.Helper()
	if err := r.Station.Set(h, slot, body); err != nil {
		r.T.Fatal(err)
	}
}

// Link links src.srcSlot to tgt.tgtSlot and returns the link's name.
func (r *Rig) Link(src Component, srcSlot string, tgt Component, tgtSlot string) string {
	r.T.Helper()
	name, err := r.Station.Link(src.Handle, srcSlot, tgt.Handle, tgtSlot)
	if err != nil {
		r.T.Fatal(err)
	}
	return name
}

// StatusNumeric is box_client.bson_status_numeric: a whole StatusNumeric BOX body.
func StatusNumeric(value float64, status string) map[string]any {
	return statusWhole("baja:StatusNumeric", map[string]any{"t": "baja:Double", "v": pyFloat(value)}, status)
}

// StatusBoolean is box_client.bson_status_boolean.
func StatusBoolean(value bool, status string) map[string]any {
	return statusWhole("baja:StatusBoolean", map[string]any{"t": "baja:Boolean", "v": strconv.FormatBool(value)}, status)
}

func statusWhole(typ string, value map[string]any, status string) map[string]any {
	value["nm"], value["n"] = "p", "value"
	return map[string]any{"nm": "p", "t": typ, "s": []any{value,
		map[string]any{"nm": "p", "n": "status", "t": "baja:Status", "v": status}}}
}

// pyFloat is str(float(x)): 1 -> "1.0", 5.5 -> "5.5".
func pyFloat(x float64) string {
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if x == float64(int64(x)) {
		s = fmt.Sprintf("%d.0", int64(x))
	}
	return s
}
