package guardpins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of lint-guard-pins (unchanged from the bash tool).
const (
	ExitOK   = 0 // clean, or WARN rows without --strict
	ExitWarn = 1 // any WARN under --strict
	ExitEnv  = 3 // usage or environment error (K20)
)

// Main implements `lint-guard-pins [--strict] <repo-root>`. The toolbelt is <repo-root>/toolbelt or
// else <repo-root>/build-n4-module-kit/toolbelt; fixtures are <repo-root>/tests/*.bats. Paths are
// printed as built from <repo-root> (no cleaning), as the bash tool's find printed them.
func Main(args []string, s cli.Stdio) int {
	strict := false
	if len(args) > 0 && args[0] == "--strict" {
		strict, args = true, args[1:]
	}
	if len(args) != 1 {
		fmt.Fprint(s.Err, "usage: lint-guard-pins.sh [--strict] <repo-root>\n")
		return ExitEnv
	}
	root := args[0]
	toolbelt := root + "/toolbelt"
	if !isDir(toolbelt) {
		if toolbelt = root + "/build-n4-module-kit/toolbelt"; !isDir(toolbelt) {
			fmt.Fprintf(s.Err, "lint-guard-pins: no toolbelt/ found under %s\n", root)
			return ExitEnv
		}
	}
	testsDir := root + "/tests"
	if !isDir(testsDir) {
		// The contract is the repository root: <repo>/tests holds the fixtures. A kit root resolves
		// its toolbelt/ but not its tests/, so name the parent that would work.
		if parent := filepath.Dir(filepath.Clean(root)); toolbelt == root+"/toolbelt" && isDir(parent+"/tests") {
			fmt.Fprintf(s.Err, "lint-guard-pins: no tests/ found under %s (a kit root? pass the repository root %s)\n", root, parent)
			return ExitEnv
		}
		fmt.Fprintf(s.Err, "lint-guard-pins: no tests/ found under %s\n", root)
		return ExitEnv
	}

	scripts, err := lintScripts(toolbelt)
	if err != nil {
		fmt.Fprintf(s.Err, "lint-guard-pins: cannot list %s: %v\n", toolbelt, err)
		return ExitEnv
	}
	fixtures := IndexFixtures(batsFiles(testsDir))
	warned := false
	for _, script := range scripts {
		b, err := os.ReadFile(script)
		if err != nil {
			fmt.Fprintf(s.Err, "lint-guard-pins: cannot read %s: %v\n", script, err)
		}
		decls, stray := Declarations(string(b))
		for _, d := range decls {
			if hit, ok := fixtures[d.ID]; ok {
				fmt.Fprintf(s.Out, "MATCH  lint-guard-pins  %s:%d  mutation %s -> %s:%d\n",
					script, d.Line, d.ID, strings.TrimPrefix(hit.Path, root+"/"), hit.Line)
			} else {
				fmt.Fprintf(s.Out, "WARN  lint-guard-pins  %s:%d  mutation %s -> no fixture\n", script, d.Line, d.ID)
				warned = true
			}
		}
		// Mutation: GP-stray -- skipping a declaration below the header hides a pin that is never checked
		for _, d := range stray {
			fmt.Fprintf(s.Out, "WARN  lint-guard-pins  %s:%d  mutation %s -> below the header (move it above the first code line)\n", script, d.Line, d.ID)
			warned = true
		}
		// Mutation: T4-nomutation -- dropping the no-declaration WARN makes an unpinned lint vacuously clean
		switch {
		case len(decls) > 0:
		case len(stray) > 0: // declarations exist, but only below the header: say so instead of "add one"
			fmt.Fprintf(s.Out, "WARN  lint-guard-pins  %s  no # Mutation: line in the header (K24(7): move the declarations below it above the first code line)\n", script)
			warned = true
		default:
			fmt.Fprintf(s.Out, "WARN  lint-guard-pins  %s  no # Mutation: line (K24(7): add '# Mutation: <fixture-id> -- <what it flips>')\n", script)
			warned = true
		}
	}
	if strict && warned {
		return ExitWarn
	}
	return ExitOK
}

// lintScripts lists every entry of toolbelt named lint-*.sh, whatever its type (a directory or an
// unreadable file is then a WARN, never silently skipped), in C byte order.
func lintScripts(toolbelt string) ([]string, error) {
	entries, err := os.ReadDir(toolbelt)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if ok, _ := filepath.Match("lint-*.sh", e.Name()); ok {
			out = append(out, toolbelt+"/"+e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// batsFiles reads testsDir/*.bats in C byte order; an entry that cannot be read (a directory, an
// unreadable file) defines no fixture, as the bash tool's `grep ... 2>/dev/null` did.
func batsFiles(testsDir string) []BatsFile {
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil
	}
	var out []BatsFile
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".bats") {
			continue
		}
		path := testsDir + "/" + name
		if b, err := os.ReadFile(path); err == nil {
			out = append(out, BatsFile{path, string(b)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
