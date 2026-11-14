package prepush

import "testing"

// The live doctrine set has one definition, the two patterns of paths.go. These tests pin which
// repository-relative paths each of them selects: a path that leaves the set stops being checked by
// the citation gate without any other test noticing.

// TestLiveDoctrineSelectsTheLiveCards: the live cards and nothing beside them.
func TestLiveDoctrineSelectsTheLiveCards(t *testing.T) {
	for path, want := range map[string]bool{
		"build-n4-module-kit/types/driver.md":       true,
		"build-n4-module-kit/METHODOLOGY.md":        true,
		"build-n4-module-kit/BUILD-LOOP.md":         true,
		"build-n4-module-kit/ORCHESTRATION.md":      true,
		"build-n4-module-kit/SOURCES.md":            true,
		"build-n4-module-kit/build-verify.md":       true,
		"build-n4-module-kit/corpus-index.md":       true,
		"build-n4-module-kit/README.md":             true,
		"build-n4-module-kit/skill/SKILL.md":        true,
		"build-n4-module-kit/toolbelt/INDEX.md":     true,
		"go/README.md":                              true,
		"CONTRIBUTING.md":                           true,
		"README.md":                                 true,
		"build-n4-module-kit/types/a/b.md":          false, // one level under types/ only
		"build-n4-module-kit/types/driver.txt":      false,
		"build-n4-module-kit/BUILD-STATE.md":        false, // history, not doctrine
		"build-n4-module-kit/BUILD-STATE.src.md":    false,
		"build-n4-module-kit/ledger.d/2026-x.md":    false,
		"build-n4-module-kit/retros/2026-x.md":      false,
		"build-n4-module-kit/retros/INDEX.md":       false,
		"build-n4-module-kit/toolbelt/other.md":     false,
		"build-n4-module-kit/skill/OTHER.md":        false,
		"build-n4-module-kit/METHODOLOGYxmd":        false, // the dot is a dot
		"x/build-n4-module-kit/METHODOLOGY.md":      false, // anchored at the toplevel
		"docs/README.md":                            false,
		"go/internal/README.md":                     false,
		"CHANGELOG.md":                              false,
		"changelog.d/2026-x.md":                     false,
		"README.md.bak":                             false,
		"build-n4-module-kit/METHODOLOGY.md.orig":   false,
		"build-n4-module-kit/toolbelt/INDEX.md/x":   false,
		"build-n4-module-kit/types/":                false,
		"build-n4-module-kit/types/.md":             false,
		"go/internal/citeresolve/resolve.go":        false, // the resolver is the other pattern
		"build-n4-module-kit/toolbelt/kit-state.sh": false,
	} {
		if got := liveDoctrine.MatchString(path); got != want {
			t.Errorf("liveDoctrine matches %q = %v, want %v", path, got, want)
		}
	}
}

// TestLiveCiteToolSelectsTheResolver: the resolver's packages, command and wrapper, whose change
// re-checks the whole set.
func TestLiveCiteToolSelectsTheResolver(t *testing.T) {
	for path, want := range map[string]bool{
		"go/internal/citeparse/parse.go":                       true,
		"go/internal/citeresolve/resolve.go":                   true,
		"go/internal/citeresolve/testdata/a/b.txt":             true,
		"go/cmd/kit-cite-resolve/main.go":                      true,
		"build-n4-module-kit/toolbelt/kit-cite-resolve.sh":     true,
		"build-n4-module-kit/toolbelt/kit-cite-resolve.sh.bak": false,
		"build-n4-module-kit/toolbelt/kit-state.sh":            false,
		"go/internal/citeparsex/parse.go":                      false,
		"go/internal/citeparse":                                false, // a directory prefix needs its slash
		"go/internal/prepush/livecite.go":                      false,
		"go/cmd/kit-cite-resolve-x/main.go":                    false,
		"x/go/cmd/kit-cite-resolve/main.go":                    false,
		"go/README.md":                                         false,
	} {
		if got := liveCiteTool.MatchString(path); got != want {
			t.Errorf("liveCiteTool matches %q = %v, want %v", path, got, want)
		}
	}
}
