package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// flagName is the name of a flag letter in flagTable; ok is false for a letter it does not hold.
func flagName(letter string) (name string, ok bool) {
	for _, f := range flagTable {
		if f.letter == letter {
			return f.name, true
		}
	}
	return "", false
}

// namedFlags is each letter of flags with its name ("h hidden, o operator"), in the order given;
// "none" for no flags. A letter the table does not hold (a newer station's) is named unknown.
func namedFlags(flags string) string {
	if flags == "" {
		return "none"
	}
	items := []string{}
	for _, r := range flags {
		name, ok := flagName(string(r))
		if !ok {
			name = "unknown"
		}
		items = append(items, string(r)+" "+name)
	}
	return strings.Join(items, ", ")
}

// shownFlags is a flags string as a plan or a refusal shows it: quoted, then its named letters.
func shownFlags(flags string) string { return `"` + flags + `" (` + namedFlags(flags) + ")" }

// canonicalFlags is flags in the table's order, each letter once; a letter the table does not
// hold keeps its place after them.
func canonicalFlags(flags string) string {
	out := ""
	for _, f := range flagTable {
		if strings.Contains(flags, f.letter) {
			out += f.letter
		}
	}
	for _, r := range flags {
		if _, ok := flagName(string(r)); !ok && !strings.ContainsRune(out, r) {
			out += string(r)
		}
	}
	return out
}

// sameFlags is whether a and b hold the same set of letters, in any order.
func sameFlags(a, b string) bool { return canonicalFlags(a) == canonicalFlags(b) }

// checkFlags refuses a requested flags string holding a letter the table does not have (the
// station would ignore it silently, Flags.decodeFromString) or a letter twice.
func checkFlags(flags string) error { return checkFlagLetters("flags", flags) }

// checkFlagLetters is checkFlags for the argument named label (flags, add or remove).
func checkFlagLetters(label, flags string) error {
	seen := map[rune]bool{}
	for _, r := range flags {
		if _, ok := flagName(string(r)); !ok {
			return plan.ToolError("unknown flag letter " + pyjson.StrRepr(string(r)) + " in " + label + " " +
				pyjson.StrRepr(flags) + "; valid letters: " + namedFlags(allFlagLetters()))
		}
		if seen[r] {
			return plan.ToolError("flag letter " + pyjson.StrRepr(string(r)) + " is repeated in " + label + " " +
				pyjson.StrRepr(flags) + "; give each letter once")
		}
		seen[r] = true
	}
	return nil
}

func allFlagLetters() string {
	out := ""
	for _, f := range flagTable {
		out += f.letter
	}
	return out
}
