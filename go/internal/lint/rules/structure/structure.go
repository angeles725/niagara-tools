// Package structure is lint-structure (toolbelt/lint-structure.sh): the module-structure lint of a
// Niagara N4 source tree. It is a lint.RootRule: it reads the module root's profiles, the files under
// them, the gradle.properties in each parent up to the .git sentinel and those of sibling modules.
package structure

import "github.com/angeles725/niagara-tools/go/internal/lint"

type rule struct{}

func (rule) ID() string              { return "lint-structure" }
func (rule) Severity() lint.Severity { return lint.Fail }

// CheckRoot runs every check in the bash lint's row order: L10, then L6-L11 per profile, then L12-L15.
func (rule) CheckRoot(root string) ([]lint.Finding, error) {
	s := &scan{root: root}
	profiles := s.profiles()
	if len(profiles) == 0 && s.err == nil { // a wrong path, never a clean pass (#351)
		return nil, lint.ErrEmptyRoot
	}
	s.hostPaths()
	for _, p := range profiles {
		s.profile(p)
	}
	s.jdkPin()
	s.siblingHomes()
	s.pluginPin()
	return s.out, s.err
}

// Spec is lint-structure's contract: one <module-root> (later arguments ignored, no flag parsed), rows
// "<sev>  lint-structure  <path>  L<n>: <reason>", exit 1 on any FAIL, 3 on a usage or read error.
func Spec() lint.Spec {
	return lint.Spec{Root: rule{}, Mode: lint.Raw, Usage: "usage: lint-structure.sh <module-root>", BarePathRow: true,
		EmptyError: lint.EmptyRow{Msg: "no module profiles found (no *-rt/-ux/-wb/-se/-doc or build-file sub-directory: wrong path or not a module root)"}}
}
