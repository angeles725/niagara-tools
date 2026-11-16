package benchcall

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

// Lines a run with writes prints on stderr before the server starts.
const (
	modeDryRunOnly = "dry run only: no write executes"
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

// checkDestructive refuses a calls file that writes with a destructive tool unless the run is a
// dry run only or the operator passed --allow-destructive.
func checkDestructive(calls []call, o *options) error {
	if o.dryRunOnly || o.destructive {
		return nil
	}
	destructive := destructiveTools()
	for _, c := range calls {
		if c.write && destructive[c.tool] {
			return fmt.Errorf("%s: line %d: %s is a destructive tool: pass --allow-destructive to execute it, or --dry-run-only",
				o.calls, c.line, c.tool)
		}
	}
	return nil
}

// writeMode is the stderr line that says whether the writes of the run execute; "" for a run
// with no write.
func writeMode(calls []call, o *options) string {
	for _, c := range calls {
		if c.write {
			if o.dryRunOnly {
				return modeDryRunOnly
			}
			return modeExecute
		}
	}
	return ""
}
