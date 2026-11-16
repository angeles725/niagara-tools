package ntplmanifest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// usedFree are the modules a template may use with no <dependency> row: baja, which every Niagara
// installation runs (types/logic-authoring.md § Shipping a template module).
var usedFree = map[string]bool{"baja": true}

// uses adds the ntpl-uses rows of one archive (retro coordination-stretch K4): for template.bog and each
// Px file, the modules it uses (modulesUsed) that no <dependency> of the manifest names (a part of the
// module counts: control-rt lists control), that are not the module's own parts and that the installation
// does not ship. As for ntpl-deps the row is FAIL with the installation's module list and WARN without it.
// An entry that could not be read or parsed is a FAIL row with its reason, never a template that uses nothing.
func (c *checker) uses(path string, deps []string, members []member) {
	listed := moduleSet(c.mod.parts, deps)
	shipped := moduleSet(c.stock)
	row := func(sev lint.Severity, format string, args ...any) {
		c.out = append(c.out, lint.Finding{Check: checkUses, Severity: sev, Path: path, Msg: fmt.Sprintf(format, args...)})
	}
	for _, m := range members {
		mods, why := modulesUsed(m.name, m.data)
		if m.unread != "" {
			why = m.unread
		}
		if why != "" {
			// Mutation: NTPL-uses-unread -- reading an entry that does not parse as one that uses nothing passes it
			row(lint.Fail, "%s", why)
			continue
		}
		var missing []string
		for _, mod := range mods {
			if !usedFree[mod] && !listed[mod] && !shipped[mod] {
				missing = append(missing, mod)
			}
		}
		if len(missing) == 0 {
			continue
		}
		sort.Strings(missing)
		names := strings.Join(missing, ", ")
		if c.stock == nil {
			c.unknown = true
			row(lint.Warn, usesMissingAny, m.name, names)
			continue
		}
		// Mutation: NTPL-uses -- dropping this row passes a template whose bog or Px uses a module it does not list
		row(lint.Fail, usesMissing, m.name, names, c.home.dir)
	}
}
