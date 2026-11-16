// Package ntplmanifest is lint-ntpl-manifest-deps (G92): the template archives a module ships. A .ntpl is a zip
// holding template.bog and template-manifest.xml, whose <dependencies> list the modules the template needs
// (types/logic-authoring.md § Shipping a template module); a station template, the kind Workbench lists
// from newStations/, carries isStation="true" on its manifest root (types/distribution.md § 8). For every
// *.ntpl under a module root the lint FAILs an archive without one of the two entries, a manifest that is not
// well-formed XML, a <dependency> naming a module no build file of the module declares, and a template under
// newStations/ whose manifest is not a station manifest. toolbelt/lint-ntpl-manifest-deps.sh is the wrapper
// and its header is the user-facing contract.
//
// It is a lint.RootRule: it lists the root itself (the shared fail-closed walk) and opens each archive
// within fixed bounds, never through the engine's whole-file text read. One file per concern: messages.go
// (names, bounds and every text), archive.go (the bounded zip read), manifest.go (the XML read), declared.go
// (the module's declared dependencies); this file composes them.
package ntplmanifest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Spec registers the rule: [--] <module-root>... (each root on its own). A root with neither a build file
// nor a template is the ERROR row (exit 3); its roots are module roots, so n4lint --all leaves it out.
func Spec() lint.Spec {
	return lint.Spec{Root: rule{}, Mode: lint.Each, Usage: usage, EmptyError: lint.EmptyRow{Msg: emptyRoot}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// CheckRoot checks every *.ntpl under root, in path order. A module that ships none is one PASS row. A
// directory the walk cannot enter, an archive that cannot be opened and a build file that cannot be read
// are errors (exit 3) after the rows of what was read.
func (rule) CheckRoot(root string) ([]lint.Finding, error) {
	templates, walkErr := lint.Walk(root, lint.Selection{Names: []string{"*.ntpl"}, PrunePaths: lint.PartOutputs()})
	mod, err := declaredIn(root)
	if err == nil && walkErr != nil {
		err = fmt.Errorf(partialWalk, root, walkErr)
	}
	if len(templates) == 0 {
		if err != nil {
			return nil, err
		}
		if len(mod.files) == 0 {
			return nil, lint.ErrEmptyRoot
		}
		return []lint.Finding{{Severity: lint.Pass, Path: root, Msg: noTemplates}}, nil
	}
	var out []lint.Finding
	for _, path := range templates {
		a, openErr := readArchive(path)
		if openErr != nil {
			if err == nil {
				err = fmt.Errorf(scanFailure, path, openErr)
			}
			continue
		}
		out = append(out, check(root, path, a, mod)...)
	}
	return out, err
}

// check is the rows of one archive: its entries, then its manifest, then what the manifest declares.
func check(root, path string, a archive, mod module) []lint.Finding {
	var out []lint.Finding
	row := func(check, format string, args ...any) {
		out = append(out, lint.Finding{Check: check, Path: path, Msg: fmt.Sprintf(format, args...)})
	}
	for _, msg := range a.entries {
		row(checkEntries, "%s", msg)
	}
	if a.unread != "" {
		row(checkManifest, "%s", a.unread)
	}
	if a.manifest == nil {
		return out
	}
	m, why := parseManifest(a.manifest)
	if why != "" {
		// Mutation: NTPL2 -- taking a manifest that does not parse for one with no dependencies passes it
		row(checkManifest, "%s", why)
		return out
	}
	for _, line := range m.nameless {
		row(checkManifest, namelessDep, line)
	}
	if underStationDir(root, path) && !m.station {
		// Mutation: NTPL4 -- dropping this row ships a newStations/ template whose manifest is not a station manifest
		row(checkStation, notAStation, m.root)
	}
	switch missing := mod.undeclared(m.deps); {
	case !mod.known: // the scan fails closed on the build file; a row here would compare against a partial list
	case len(m.deps) > 0 && len(mod.files) == 0:
		row(checkDeps, noBuildFile, root, len(m.deps))
	case len(missing) > 0:
		// Mutation: NTPL3 -- dropping this row passes a template that needs a module the build never declares
		row(checkDeps, undeclared, root, strings.Join(missing, ", "))
	}
	return out
}

// underStationDir reports a template with a newStations directory between the root and itself.
func underStationDir(root, path string) bool {
	rel := strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
	parts := strings.Split(rel, "/")
	for _, dir := range parts[:len(parts)-1] {
		if dir == stationDir {
			return true
		}
	}
	return false
}

// undeclared are the names of deps the module does not declare, sorted, each once.
func (m module) undeclared(deps []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range deps {
		if !m.names[d] && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	sort.Strings(out)
	return out
}
