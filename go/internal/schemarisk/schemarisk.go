// Package schemarisk is the B795 §795.4 slot-diff survival classifier behind toolbelt/schema-risk.sh: it
// harvests the @NiagaraProperty/@NiagaraAction/@NiagaraTopic slots of two module snapshots, classifies each
// change and reports the worst cell (SAFE < LOSSY < OUTAGE), so a saved-data-breaking change is caught before
// deploy (a .bog binds by name, ungated). Limits L1-L4 are documented in toolbelt/schema-risk.sh.
package schemarisk

import (
	_ "embed"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// table is the B795 §795.4 classifier table (change_kind,verdict,evidence,note); SR-CSV pins it to
// tests/fixtures/schema-risk/b795-795.4.csv.
//
//go:embed b795-795.4.csv
var table string

// Row is one table row: the verdict of a change kind, its evidence and note.
type Row struct{ Verdict, Evidence, Note string }

// lookup is the table row of kind; a kind with no row is the UNKNOWN fail-safe (never downgrade on uncertainty).
func lookup(kind string) Row {
	for _, ln := range strings.Split(table, "\n")[1:] {
		if f := strings.Split(ln, ","); len(f) >= 4 && f[0] == kind {
			return Row{f[1], f[2], strings.Join(f[3:], ",")}
		}
	}
	return Row{"OUTAGE", "B795-795.2-failsafe", "default-fail-safe"}
}

// score is a verdict's severity, which is also the exit code: SAFE 0, LOSSY 1, anything else OUTAGE 2.
func score(verdict string) int {
	switch verdict {
	case "SAFE":
		return 0
	case "LOSSY":
		return 1
	}
	return 2
}

// Slot is one declared slot: Kind "property" or "action" (an action or a topic), its name and, for a
// property, its type attribute (an action's type reads "action").
type Slot struct{ Kind, Name, Type string }

var (
	slotStart = regexp.MustCompile(`^[[:space:]]*@Niagara(Property|Action|Topic)`)
	attrRe    = map[string]*regexp.Regexp{"name": attrPattern("name"), "type": attrPattern("type")}
)

func attrPattern(a string) *regexp.Regexp {
	return regexp.MustCompile(a + `[[:space:]]*=[[:space:]]*"([^"]*)"`)
}

// attr is the value of the leftmost `<attr> = "<value>"` in buf ("" when there is none).
func attr(buf, a string) string {
	if m := attrRe[a].FindStringSubmatch(buf); m != nil {
		return m[1]
	}
	return ""
}

// Parse harvests the slots of lines (every Java file's lines, in walk order, as one stream) in declaration
// order. An annotation that starts a line is joined with the following lines, one space apart, until its
// parentheses balance or the stream ends; one without a non-empty name is skipped.
func Parse(lines []string) []Slot {
	var slots []Slot
	for i := 0; i < len(lines); i++ {
		if !slotStart.MatchString(lines[i]) {
			continue
		}
		buf, depth := lines[i], javasrc.ParenDepth(lines[i])
		for ; depth > 0 && i+1 < len(lines); depth += javasrc.ParenDepth(lines[i]) {
			i++
			buf += " " + lines[i]
		}
		name := attr(buf, "name")
		switch {
		case name == "":
		case strings.HasPrefix(strings.TrimLeft(buf, " \t\n\v\f\r"), "@NiagaraProperty"):
			slots = append(slots, Slot{"property", name, attr(buf, "type")})
		default:
			slots = append(slots, Slot{"action", name, "action"})
		}
	}
	return slots
}

// Change is one classified slot change: its change kind and the slot label ("<name>" or "<from>-><to>").
type Change struct{ Kind, Label string }

// side is one snapshot's slots: the last declaration of each name and its position, and every name in
// declaration order.
type side struct {
	by    map[string]Slot
	last  map[string]int
	order []string
}

func index(slots []Slot) side {
	s := side{by: map[string]Slot{}, last: map[string]int{}}
	for i, sl := range slots {
		s.by[sl.Name], s.last[sl.Name] = sl, i
		s.order = append(s.order, sl.Name)
	}
	return s
}

// names is the names of s that keep holds, in the order of their last declaration.
func (s side) names(keep func(Slot) bool) []string {
	var out []string
	for i, nm := range s.order {
		if s.last[nm] == i && keep(s.by[nm]) {
			out = append(out, nm)
		}
	}
	return out
}

func simple(typ string) bool { return !strings.Contains(typ, ":") }

// Diff classifies every slot change from before to after: one rename when exactly one slot was removed and one
// added with the same kind and type (L1), else removals, additions, kind swaps (UNKNOWN, L3) and retypes; a
// reorder of an otherwise unchanged set lists every before slot as reorder_slot. Each category is listed in declaration
// order: the after snapshot's for additions, the before snapshot's for the others (bash printed gawk for-in
// hash order).
func Diff(before, after []Slot) []Change {
	b, a := index(before), index(after)
	removed := b.names(func(s Slot) bool { _, ok := a.by[s.Name]; return !ok })
	added := a.names(func(s Slot) bool { _, ok := b.by[s.Name]; return !ok })
	swapped := b.names(func(s Slot) bool { o, ok := a.by[s.Name]; return ok && o.Kind != s.Kind })
	retyped := b.names(func(s Slot) bool { o, ok := a.by[s.Name]; return ok && o.Kind == s.Kind && o.Type != s.Type })
	var out []Change
	if len(removed) == 1 && len(added) == 1 && b.by[removed[0]].Kind == a.by[added[0]].Kind && b.by[removed[0]].Type == a.by[added[0]].Type {
		out = append(out, Change{"rename_slot", removed[0] + "->" + added[0]})
		removed, added = nil, nil
	} else if len(removed)+len(added)+len(swapped)+len(retyped) == 0 && reordered(b.order, a.order) {
		for _, nm := range b.order {
			out = append(out, Change{"reorder_slot", nm})
		}
		return out
	}
	for _, nm := range removed {
		kind := "remove_slot_unknown"
		if s := b.by[nm]; s.Kind == "action" {
			kind = "remove_action_or_topic"
		} else if simple(s.Type) {
			kind = "remove_slot_simple"
		}
		out = append(out, Change{kind, nm})
	}
	for _, nm := range added {
		out = append(out, Change{"add_slot", nm})
	}
	for _, nm := range swapped {
		out = append(out, Change{"UNKNOWN", nm})
	}
	for _, nm := range retyped {
		kind := "retype_unknown"
		if simple(b.by[nm].Type) && simple(a.by[nm].Type) {
			kind = "retype_simple"
		}
		out = append(out, Change{kind, nm})
	}
	return out
}

// reordered reports whether some before position holds another name in after (a missing one reads "").
func reordered(b, a []string) bool {
	for i, nm := range b {
		if i >= len(a) || a[i] != nm {
			return true
		}
	}
	return false
}
