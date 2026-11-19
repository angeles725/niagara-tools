// Package dashboardsave is the lint-dashboard-write-save rule (retro lint-dashboard-write-save Δ1,
// issue #821): a -ux part whose code writes station state with the request Context FAILs when nothing in its
// module requests a station save. A write that is never saved is lost on the next restart, and a driver that
// re-sends its saved value (writeOnStart/writeOnUp) then drives the equipment from the stale one: a dashboard
// shipped three releases that way and a restart switched a unit ON. It is the gate form of BUILD-LOOP §6.b
// item 6 ("Persisted operator state survives the first restart").
//
// A lint.TreeRule over a module root (Spec.ModuleRoot, as lint-duplicate-helper): every *.java under a part's
// src/ (srcTest/ and build/ pruned), grouped by module (the path before <part>-rt|-ux|-wb|-se/src/). In code
// only (comments and string literal contents blanked, go/internal/javasrc):
//
//   - write: in a -ux file, a .set( / .add( / .remove( / .invoke( call whose last argument is a Context or
//     WebOp declared in that file (a parameter or a local: the request's Context), or any addSpecialEvent(.
//     A list add, a one-argument remove or a null-Context write is no write here (writes.go).
//   - save: anywhere in the module, BStation.save (the action, Clock.schedule(station, d, BStation.save, null)
//     in the reference) or a station save call: Sys.getStation().save(), station.saveAsync(, Station.save*( .
//   - declared: MODULE-MAP.md at the module root holds a non-empty "persistence: <where>" line outside a
//     ``` fence, for a module that saves on purpose elsewhere (another module, a station-side service).
//
// One FAIL per write line of a module with writes, no save and no declaration; a module root with no -ux
// src/ file is the empty-root ERROR (exit 3). Mutation pins (testdata/): matching any last argument instead of
// a declared Context (truth-write-no-save: the list add and the null-Context set turn red), reading comments
// or literals (truth-save-not-code), counting an empty persistence: line (truth-save-not-code), dropping the
// multi-line argument scan (truth-write-no-save: day.add), and limiting the save search to -ux
// (truth-save-in-rt) each turn a case red.
package dashboardsave

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-dashboard-write-save"

// msg is the text of every FAIL row.
const msg = "writes station state with the request Context and nothing in the module requests a station save" +
	" (lost on the next restart): schedule BStation.save after the write (types/dashboard.md § Write path)" +
	" or declare persistence: <where> in MODULE-MAP.md"

// Spec registers the rule: one module root, *.java outside srcTest/ and build/.
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.One, ModuleRoot: true, Usage: "usage: lint-dashboard-write-save.sh [--strict] <module-root>",
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest", "build"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -ux part's src/ (wrong path: pass the module root; a module with no -ux part has no dashboard writes)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// module is what one module root holds: its write sites in walk order and whether any file saves.
type module struct {
	writes []lint.Finding
	saves  bool
}

// CheckTree groups the part files by module, then reports each module's writes when it neither saves nor
// declares where it persists, modules in order of first appearance.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	mods, order := map[string]*module{}, []string(nil)
	ux := false
	for _, f := range files {
		root, kind, ok := partOf(f.Path)
		if !ok {
			continue
		}
		m := mods[root]
		if m == nil {
			m = &module{}
			mods[root], order = m, append(order, root)
		}
		code := codeView(f)
		m.saves = m.saves || savesStation(code)
		if kind == "ux" {
			ux = true
			for _, line := range writeLines(code) {
				m.writes = append(m.writes, lint.Finding{Path: f.Path, Line: line, Msg: msg})
			}
		}
	}
	if !ux {
		return nil, lint.ErrEmptyRoot
	}
	var out []lint.Finding
	for _, root := range order {
		m := mods[root]
		if m.saves || len(m.writes) == 0 {
			continue
		}
		declared, err := declaresPersistence(root)
		if err != nil {
			return out, err
		}
		if !declared {
			out = append(out, m.writes...)
		}
	}
	return out, nil
}
