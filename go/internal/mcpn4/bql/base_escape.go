package bql

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// escapedSlotPath reports whether path holds only what a slot path holds in the form the station
// matches it: letters, digits, _, $ (an escape such as $20) and / between names. A raw space,
// quote, &, dot, hyphen or non-ASCII character names no component, so the read would fail at the
// station without saying why; the station's escape for it ($20 for a space, $u00e9 for é) does.
func escapedSlotPath(path string) bool {
	for _, r := range path {
		if !(r == '/' || r == '_' || r == '$' || ('0' <= r && r <= '9') || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')) {
			return false
		}
	}
	return true
}

// escapedBaseError is the refusal of a base that is not in escaped form: the older refusal's text
// (the frozen Python differential compares it) then the escaped-form hint.
func escapedBaseError(base string) error {
	return valueError("base must be a plain slot path such as station:|slot:/Drivers, got " + pyjson.StrRepr(base) +
		" (the station matches slot names in their escaped form: " +
		"only letters, digits, _, $ and / are allowed, and a space is $20, so write Cold$20Room, not Cold Room)")
}
