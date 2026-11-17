package readtools

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The slots n4_users_audit (station-audits S14) reads from a baja:User, fail closed like the
// recipient audit (alarm_recipient_audit.go, PR #788): only the slots of userSafeSlots are read
// for their value; email and cellPhoneNumber show only "<set>" or ""; every other slot (the
// authenticator and its password hash, permissions, any vendor or dynamic slot) is never read
// for its value and is only named in masked_slots.

// userSafeSlots are the BUser properties whose value the audit reads, with the value a load
// omits at the type default (BUser.java:199-240, 4.14.0.162): fullName "", enabled true,
// expiration the null time, lockOut false, roles "", authenticationSchemeName DigestScheme,
// prototypeName "".
var userSafeSlots = map[string]string{"fullName": "", "enabled": "true", "expiration": "null", "lockOut": "false",
	"roles": "", "authenticationSchemeName": "DigestScheme", "prototypeName": ""}

// userContactSlots are the contact slots shown only as set or empty, by reply key.
var userContactSlots = [][2]string{{"email", "email"}, {"cellPhoneNumber", "cell_phone"}}

// userTypes are the component types the audit reads as users.
var userTypes = map[string]bool{"baja:User": true}

// userSlots is what the audit keeps of one user: the safe values (at their default when the load
// omitted them), the contact slots as set or empty, and the names of every other slot.
type userSlots struct {
	safe    map[string]string
	contact *pyjson.Object
	masked  []string
}

// readUserSlots reads kids (the children of one user node) through the allowlist.
func readUserSlots(kids *pyjson.Object) userSlots {
	out := userSlots{safe: map[string]string{}, contact: pyjson.NewObject()}
	for name, def := range userSafeSlots {
		out.safe[name] = def
	}
	contact := map[string]bool{}
	for _, c := range userContactSlots {
		contact[c[0]] = true
		out.contact.Set(c[1], maskedEmpty)
	}
	for _, name := range kids.Keys() {
		node := box.Node(kids, name)
		_, isSafe := userSafeSlots[name]
		switch {
		case isSafe && !scalarValue(node):
			out.masked = append(out.masked, name)
		case isSafe:
			if v, ok := node.Get("v"); ok {
				out.safe[name] = pyjson.Str(v)
			}
		case contact[name]:
			for _, c := range userContactSlots {
				if c[0] == name && hasText(node) {
					out.contact.Set(c[1], maskedSet)
				}
			}
		default:
			out.masked = append(out.masked, name)
		}
	}
	return out
}

// scalarValue reports whether an allowlisted slot may be echoed: not typed baja:Password, and its
// value (when sent) a string, boolean or number, never an object or a list whose Repr could carry
// a nested secret. A slot that fails it is masked like any other slot.
func scalarValue(node *pyjson.Object) bool {
	if t, _ := node.Get("t"); pyjson.Str(t) == "baja:Password" {
		return false
	}
	v, ok := node.Get("v")
	if !ok {
		return true
	}
	switch v.(type) {
	case string, bool, int, int64, float64, pyjson.Number:
		return true
	}
	return false
}

// hasText reports whether node holds a non-blank value; the value itself is never kept.
func hasText(node *pyjson.Object) bool {
	v, ok := node.Get("v")
	return ok && strings.TrimSpace(pyjson.Str(v)) != ""
}

// splitRoles is BUser.splitRoles: the role names of a roles text, trimmed, blanks dropped.
func splitRoles(text string) []string {
	out := []string{}
	for _, r := range strings.Split(text, ",") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// maskedUnion is the sorted union of every user's masked slot names plus the contact slots.
func maskedUnion(users []userSlots) []any {
	seen := map[string]bool{}
	for _, c := range userContactSlots {
		seen[c[0]] = true
	}
	for _, u := range users {
		for _, name := range u.masked {
			seen[name] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return strs(names)
}
