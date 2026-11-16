package cisteps

import (
	"errors"
	"path"
	"strings"
)

// The changed-file selection of `scripts/ci-local.sh --changed` (retro coordination-stretch C3): the CI
// jobs that failed on #684 (the kit's bats lint, shellcheck 0.10) were reproducible locally, but only the
// files of the change need them before a push. The shellcheck file set is read from ci.yml itself, so
// the local selection cannot drift from what CI checks. When one file of that set changed, the whole set
// is shellchecked as the change leaves it, as CI does: a sourced file is read with its sourcer.

// Plan is what the changed files need: the pinned shellcheck, the bats lints and gofmt.
type Plan struct {
	Shellcheck, Bats, Gofmt []string
}

// Empty reports whether no changed file needs a check.
func (p Plan) Empty() bool { return len(p.Shellcheck)+len(p.Bats)+len(p.Gofmt) == 0 }

// ShellcheckGlobs returns the file globs of the one run step whose script is a single
// `shellcheck <glob>...` line. None or several such steps is an error: an empty selection would pass
// every change unchecked.
func ShellcheckGlobs(steps []Step) ([]string, error) {
	var globs []string
	found := 0
	for _, st := range steps {
		f := strings.Fields(st.Body)
		if st.Kind != "run" || strings.Contains(st.Body, "\n") || len(f) < 2 || f[0] != "shellcheck" {
			continue
		}
		found++
		globs = f[1:]
	}
	switch found {
	case 0:
		return nil, errors.New("no run step is a single `shellcheck <files>` line; --changed cannot tell which files CI shellchecks")
	case 1:
		return globs, nil
	}
	return nil, errors.New("several run steps are `shellcheck <files>` lines; --changed reads exactly one")
}

// Select sorts changed repository-relative paths into the checks they need, keeping their order.
// A glob matches as path.Match does (a * never crosses a /), the way the shell expands it in CI.
// The bats lints read tests/**/*.bats outside fixtures/ and testdata/, as the kit's BEX-kit case does.
func Select(globs, paths []string) Plan {
	var p Plan
	for _, f := range paths {
		for _, g := range globs {
			if ok, _ := path.Match(g, f); ok {
				p.Shellcheck = append(p.Shellcheck, f)
				break
			}
		}
		if isLintedBats(f) {
			p.Bats = append(p.Bats, f)
		}
		if strings.HasPrefix(f, "go/") && strings.HasSuffix(f, ".go") {
			p.Gofmt = append(p.Gofmt, f)
		}
	}
	return p
}

func isLintedBats(f string) bool {
	if !strings.HasPrefix(f, "tests/") || !strings.HasSuffix(f, ".bats") {
		return false
	}
	for _, seg := range strings.Split(path.Dir(f), "/") {
		if seg == "fixtures" || seg == "testdata" {
			return false
		}
	}
	return true
}
