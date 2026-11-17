package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// versionMatches is n4_connect's expected_version rule (field retro delta U14): expected is the
// detected version or a prefix of it by whole dotted parts ("4.14" matches 4.14.0.162, "4.1"
// does not); an undetected version never matches.
func versionMatches(detected, expected string) bool {
	return detected != "" && (detected == expected || strings.HasPrefix(detected, expected+"."))
}

// checkExpectedVersion refuses a connect whose detected version does not match the optional
// expected_version argument; without the argument every version passes.
func checkExpectedVersion(args *pyjson.Object, detected string) error {
	e, ok := args.Get("expected_version")
	if !ok {
		return nil
	}
	expected := e.(string) // validated against the schema: a string
	if versionMatches(detected, expected) {
		return nil
	}
	shown := any(detected)
	if detected == "" {
		shown = nil
	}
	return toolError("station version mismatch: expected %s but the station reports %s",
		pyjson.StrRepr(expected), pyjson.Repr(shown))
}
