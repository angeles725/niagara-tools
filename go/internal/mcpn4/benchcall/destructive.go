package benchcall

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

// Lines a run with writes prints on stderr before the server starts.
const (
	modeDryRunOnly = "dry run only: no write executes (only --execute executes writes)"
	modeExecute    = "writes execute (dry run, then the confirmation token)"
)

// destructiveTools is every write tool the server annotates destructiveHint: true; the registry
// is the one list, so a new destructive tool is guarded without touching n4call.
func destructiveTools() map[string]bool {
	out := map[string]bool{}
	for _, t := range registry.Default().Write {
		ann, _ := t.Descriptor.Get("annotations")
		if a, ok := ann.(*pyjson.Object); ok {
			if v, _ := a.Get("destructiveHint"); v == true {
				out[t.Name] = true
			}
		}
	}
	return out
}

// checkDestructive refuses, in a run that executes, a destructive write unless the operator
// passed --allow-destructive and the line carries its own "execute": true. A run without
// --execute dry-runs every write, so a lost flag never executes one (retro 2026-11-16 row M17).
func checkDestructive(calls []call, o *options) error {
	if !o.execute {
		return nil
	}
	destructive := destructiveTools()
	for _, c := range calls {
		if !c.write || !destructive[c.tool] {
			continue
		}
		head := fmt.Sprintf("%s: line %d: %s is a destructive tool: ", o.calls, c.line, c.tool)
		if !c.execute {
			return fmt.Errorf(`%sits line needs its own "execute": true to execute (a run-wide flag never `+
				"executes a destructive call), or run without --execute for a dry run", head)
		}
		if !o.destructive {
			return fmt.Errorf("%spass --allow-destructive to execute it, or run without --execute for a dry run", head)
		}
	}
	return nil
}

// writeMode is the stderr line that says whether the writes of the run execute; "" for a run
// with no write.
func writeMode(calls []call, o *options) string {
	for _, c := range calls {
		if c.write {
			if !o.execute {
				return modeDryRunOnly
			}
			return modeExecute
		}
	}
	return ""
}
