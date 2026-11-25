package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// subState is what n4_subscribe_point reads of a control point: its out value and status
// flags, and its proxy ext type and status flags (absent on a point with no proxy ext).
type subState struct {
	value      any
	flags      []string
	proxyType  string
	proxyFlags []string
}

// readSubState loads ord and reads its out and proxyExt; a component with no out slot is refused,
// since only a control point has a value a driver refreshes.
func (t *Tools) readSubState(ord string) (subState, error) {
	nodes, _, err := t.timedLoad(ord, subLoadDepth)
	if err != nil {
		return subState{}, err
	}
	if _, ok := nodes.Get("out"); !ok {
		return subState{}, toolError("%s has no out slot: n4_subscribe_point reads a control point", ord)
	}
	kids := box.Children(nodes, "out")
	flags, err := box.StatusFlags(box.ChildValue(kids, "status", "0"))
	if err != nil {
		return subState{}, err
	}
	st := subState{value: box.ChildValue(kids, "value", nil), flags: flags}
	if kind, ok := box.Node(nodes, "proxyExt").Get("t"); ok {
		st.proxyType = pyjson.Str(kind)
		if st.proxyFlags, err = box.StatusFlags(box.ChildValue(box.Children(nodes, "proxyExt"), "status", "0")); err != nil {
			return subState{}, err
		}
	}
	return st, nil
}

// has reports whether flag is set on the point's out.
func (s subState) has(flag string) bool {
	for _, f := range s.flags {
		if f == flag {
			return true
		}
	}
	return false
}

// notPolled reports a point its driver does not poll: down, disabled or in fault.
func (s subState) notPolled() bool {
	return s.has("down") || s.has("disabled") || s.has("fault")
}

// entry is the point state as the reply shows it.
func (s subState) entry() *pyjson.Object {
	out := obj("value", s.value, "status_flags", strList(s.flags))
	if s.proxyType != "" {
		out.Set("proxy_ext", s.proxyType)
		out.Set("proxy_status_flags", strList(s.proxyFlags))
	}
	return out
}

func strList(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
