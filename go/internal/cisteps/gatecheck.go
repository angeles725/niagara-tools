package cisteps

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The parity guard of `ci-local.sh --changed` (kit #973/#983): every ci.yml run step and every Go package whose
// tests read the repository tree must have a twin in gates.go, or the guard names what to add and where.

var (
	upLiteral = regexp.MustCompile(`"((?:\.\./)*\.\.)(/[^"]*)?"`)      // "../../x" or "../.."
	upJoin    = regexp.MustCompile(`((?:"\.\.",\s*)*"\.\.")\s*(,|\))`) // filepath.Join("..", "..", "x") or ("..", "..")
)

// TreeReaders returns the packages under goRoot/internal whose tests read the repository outside their own
// directory: a relative path that climbs above go/, or a bare path to go/ itself (a tree walk). Test files are
// scanned as text; a literal that is no path (a normalisation case) is listed in NotTreeGates with its reason.
func TreeReaders(goRoot string) ([]string, error) {
	found := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(goRoot, "internal"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, "_test.go") {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(goRoot, filepath.Dir(p))
		pkg := filepath.ToSlash(rel)
		depth := strings.Count(pkg, "/") + 1
		for _, m := range upLiteral.FindAllStringSubmatch(string(src), -1) {
			if up := strings.Count(m[1], ".."); up > depth || (up == depth && m[2] == "") {
				found[pkg] = true
			}
		}
		for _, m := range upJoin.FindAllStringSubmatch(string(src), -1) {
			if up := strings.Count(m[1], `".."`); up > depth || (up == depth && m[2] == ")") {
				found[pkg] = true
			}
		}
		return nil
	})
	var pkgs []string
	for p := range found {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	return pkgs, err
}

// CheckParity returns one message per gap, each naming the step or package and where to add its twin. steps are
// ci.yml's; skip are the step globs ci-local.sh skips on purpose (its SKIP_RULES); goRoot is the go/ directory.
func CheckParity(steps []Step, skip []string, goRoot string) []string {
	var msgs []string
	covered := func(name string) bool {
		for _, g := range append(append(skip, planSteps...), fullOnlyGlobs()...) {
			if stepMatch(g, name) {
				return true
			}
		}
		for _, g := range Gates {
			for _, t := range append([]string{g.Step}, g.Twins...) {
				if t != "" && stepMatch(t, name) {
					return true
				}
			}
		}
		return false
	}
	for _, st := range steps {
		if st.Kind == "run" && !covered(st.Name) {
			msgs = append(msgs, fmt.Sprintf("ci.yml step %q has no local twin: add a Gate (with its Triggers) to Gates in "+
				"go/internal/cisteps/gates.go, or a FullOnly row with the reason it cannot run in --changed", st.Name))
		}
	}
	hasStep := func(glob string) bool {
		for _, st := range steps {
			if stepMatch(glob, st.Name) {
				return true
			}
		}
		return false
	}
	for _, g := range Gates {
		for _, t := range append([]string{g.Step}, g.Twins...) {
			if t != "" && !hasStep(t) {
				msgs = append(msgs, fmt.Sprintf("gate %q names ci.yml step %q, which no step matches: fix it in gates.go", g.Name, t))
			}
		}
	}
	for _, f := range FullOnly {
		if !hasStep(f.Step) {
			msgs = append(msgs, fmt.Sprintf("FullOnly row %q matches no ci.yml step: delete it from gates.go", f.Step))
		}
	}
	readers, err := TreeReaders(goRoot)
	if err != nil {
		return append(msgs, "cannot scan the Go tests: "+err.Error())
	}
	for _, pkg := range readers {
		if !gatePackage(pkg) && NotTreeGates[pkg] == "" {
			msgs = append(msgs, fmt.Sprintf("package %s reads the repository tree in its tests and has no gate: add it to a Gate's Pkgs "+
				"(with the Triggers that change what it reads) in go/internal/cisteps/gates.go, or to NotTreeGates with the reason", pkg))
		}
	}
	for pkg := range NotTreeGates {
		if _, err := os.Stat(filepath.Join(goRoot, filepath.FromSlash(pkg))); err != nil {
			msgs = append(msgs, fmt.Sprintf("NotTreeGates row %q names no package: delete it from gates.go", pkg))
		}
	}
	return msgs
}

// stepMatch is bash's `[[ name == glob ]]` for the globs of step names, where * crosses a /.
func stepMatch(glob, name string) bool {
	parts := strings.Split(glob, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile("^" + strings.Join(parts, ".*") + "$").MatchString(name)
}

func gatePackage(pkg string) bool {
	for _, g := range Gates {
		for _, p := range g.Pkgs {
			if p == pkg {
				return true
			}
		}
	}
	return false
}

func fullOnlyGlobs() []string {
	var out []string
	for _, f := range FullOnly {
		out = append(out, f.Step)
	}
	return out
}
