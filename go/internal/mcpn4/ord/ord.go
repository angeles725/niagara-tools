// Package ord joins station ORDs and decides which ones a write may touch: child_ord and
// join_ord (mcp-n4-kit/mcp_n4/box.py:183-207), _plain_slot_ord and WriteScope
// (safety.py:115-155).
package ord

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// SlotOrdPrefix is SLOT_ORD_PREFIX: the only ORD form a write may target is one station slot
// path with nothing chained after it.
const SlotOrdPrefix = "station:|slot:/"

// ChildOrd is child_ord: the ORD of child name under parent, the one place an ORD is joined.
// It never produces "//"; name must be one plain slot name (not empty, no '/' or '|', not "..").
func ChildOrd(parent, name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "/|") || name == ".." {
		return "", &pyjson.ValueError{Msg: "ORD child name must be one plain slot name, got " + pyjson.StrRepr(name)}
	}
	if strings.Contains(parent, "//") {
		return "", &pyjson.ValueError{Msg: "parent ORD is malformed (double slashes?): " + pyjson.StrRepr(parent)}
	}
	if !strings.Contains(parent, "|slot:") {
		parent = strings.TrimRight(parent, "|") + "|slot:/"
	}
	return strings.TrimRight(parent, "/") + "/" + name, nil
}

// JoinOrd is join_ord: ChildOrd applied to each segment of a relative a/b/c path; "" is the
// parent itself (without a trailing slash unless it is a slot root).
func JoinOrd(parent, rel string) (string, error) {
	if rel == "" {
		if strings.HasSuffix(parent, ":|slot:/") {
			return parent, nil
		}
		return strings.TrimRight(parent, "/"), nil
	}
	out := parent
	for _, part := range strings.Split(rel, "/") {
		var err error
		if out, err = ChildOrd(out, part); err != nil {
			return "", err
		}
	}
	return out, nil
}

// PlainSlotOrd is _plain_slot_ord: a string "station:|slot:/<path>" with no further ORD hop
// ('|' after the prefix) and no ".." segment. A chained ORD resolves past its slot path, so a
// prefix match on its text says nothing about the component it reaches.
func PlainSlotOrd(v any) bool {
	s, ok := v.(string)
	if !ok || !strings.HasPrefix(s, SlotOrdPrefix) || strings.Contains(s[len(SlotOrdPrefix):], "|") {
		return false
	}
	for _, seg := range strings.Split(s, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// Scope is WriteScope: the ORD prefixes a write may touch, matched on slot boundaries.
type Scope struct{ prefixes []string }

// NewScope refuses a prefix that is not a plain slot ORD and keeps each without trailing '/'.
func NewScope(prefixes []string) (*Scope, error) {
	s := &Scope{}
	for _, p := range prefixes {
		if !PlainSlotOrd(p) {
			return nil, &safety.Error{Msg: "--write-scope " + pyjson.StrRepr(p) + " must be a plain slot ORD such as " +
				"'station:|slot:/Drivers' (no '|' after the slot path)"}
		}
	}
	for _, p := range prefixes {
		s.prefixes = append(s.prefixes, strings.TrimRight(p, "/"))
	}
	return s, nil
}

func noScope() error { return &safety.Error{Msg: safety.ReasonScopeNone + ": every write is refused"} }

// RequireAny refuses every write when no prefix is configured (for writes that target the
// whole station rather than one ORD).
func (s *Scope) RequireAny() error {
	if len(s.prefixes) == 0 {
		return noScope()
	}
	return nil
}

// Check refuses v (any value a caller passed) unless it is a plain slot ORD equal to or under a
// prefix.
func (s *Scope) Check(v any) error {
	if len(s.prefixes) == 0 {
		return noScope()
	}
	if !PlainSlotOrd(v) {
		return &safety.Error{Msg: "ord " + pyjson.Repr(v) + " " + safety.ReasonScopePlain +
			" (station:|slot:/<path>, nothing chained after the slot path)"}
	}
	target := strings.TrimRight(v.(string), "/")
	for _, p := range s.prefixes {
		if target == p || strings.HasPrefix(target, p+"/") {
			return nil
		}
	}
	return &safety.Error{Msg: "ord " + pyjson.Repr(v) + " " + safety.ReasonScopeOutside}
}
