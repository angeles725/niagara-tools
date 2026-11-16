package cisteps

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ExitEnv is the exit code of the two subcommands for an I/O or git failure, and of the lint for a
// usage error or an empty input set (the kit's lint convention).
const ExitEnv = 3

const changedUsage = "usage: ci-steps changed --ci-file <workflow.yml> --tree <dir> [--range <from>..<to>]\n"

// changedMain implements `ci-steps changed`: it lists the files a change touches (git, in the process
// directory), writes the ones a check needs under --tree (for shellcheck: the changed files of CI's set with
// the files they source and that source them, or the whole set when a source path cannot be resolved) and
// prints one `<check>\t<path>` line per check and file: shellcheck, bats (the bats lints) and gofmt, in that order. Exit 0, 2 usage or an
// unreadable workflow, 3 a git or write failure.
func changedMain(args []string, s cli.Stdio) int {
	fl := flag.NewFlagSet("ci-steps changed", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	ciFile := fl.String("ci-file", "", "")
	tree := fl.String("tree", "", "")
	rng := fl.String("range", "", "")
	if err := fl.Parse(args); err != nil || fl.NArg() != 0 || *ciFile == "" || *tree == "" {
		fmt.Fprint(s.Err, changedUsage)
		return ExitBad
	}
	data, err := os.ReadFile(*ciFile)
	if err != nil {
		fmt.Fprintln(s.Err, osError(*ciFile, err))
		return ExitBad
	}
	steps, err := Parse(*ciFile, data)
	if err != nil {
		fmt.Fprintln(s.Err, err)
		return ExitBad
	}
	globs, err := ShellcheckGlobs(steps)
	if err != nil {
		fmt.Fprintf(s.Err, "ci-steps changed: %s: %v\n", *ciFile, err)
		return ExitBad
	}
	set, err := listChanges(*rng, s.Err)
	if err != nil {
		fmt.Fprintf(s.Err, "ci-steps changed: %v\n", err)
		return ExitEnv
	}
	plan := Select(globs, set.paths)
	if len(plan.Shellcheck) > 0 {
		// A sourced file is read with its sourcer: the changed files go with what they source and what
		// sources them (sourced.go); CI's whole set only when a source path cannot be resolved.
		all, err := set.allFiles()
		if err != nil {
			fmt.Fprintf(s.Err, "ci-steps changed: %v\n", err)
			return ExitEnv
		}
		whole := Select(globs, all).Shellcheck
		closure, unresolved, err := shellcheckClosure(plan.Shellcheck, whole, set.read)
		if err != nil {
			fmt.Fprintf(s.Err, "ci-steps changed: %v\n", err)
			return ExitEnv
		}
		plan.Shellcheck = closure
		if unresolved != "" {
			fmt.Fprintf(s.Err, "ci-steps changed: %s: cannot resolve the sourced path; shellchecking CI's whole file set\n", unresolved)
			plan.Shellcheck = whole
		}
	}
	need := append(append(append([]string{}, plan.Shellcheck...), plan.Bats...), plan.Gofmt...)
	if len(plan.Shellcheck) > 0 && hasPath(set, shellcheckConfig) {
		need = append(need, shellcheckConfig) // the suppressions CI's shellcheck reads; not a target itself
	}
	if err := set.materialize(*tree, need); err != nil {
		fmt.Fprintf(s.Err, "ci-steps changed: %v\n", err)
		return ExitEnv
	}
	for _, group := range []struct {
		check string
		paths []string
	}{{"shellcheck", plan.Shellcheck}, {"bats", plan.Bats}, {"gofmt", plan.Gofmt}} {
		for _, p := range group.paths {
			fmt.Fprintf(s.Out, "%s\t%s\n", group.check, p)
		}
	}
	return 0
}

// shellcheckConfig is the project config shellcheck reads from the repository root.
const shellcheckConfig = ".shellcheckrc"

// hasPath reports whether the change's end state holds a file at path.
func hasPath(set changeSet, path string) bool {
	all, err := set.allFiles()
	if err != nil {
		return false
	}
	for _, p := range all {
		if p == path {
			return true
		}
	}
	return false
}

// lintMain implements `ci-steps lint-bats-heredoc <file|dir>...`: one
// `FAIL  bats-heredoc-test  <file>:<line>  <why>` row per here-document line that starts with the test
// keyword. A directory is walked for *.bats with fixtures/ and testdata/ pruned. Exit 0 clean, 1 any
// row, 3 usage, an unreadable input or no *.bats at all.
func lintMain(args []string, s cli.Stdio) int {
	if len(args) == 0 {
		fmt.Fprint(s.Err, "usage: ci-steps lint-bats-heredoc <file|dir>...\n")
		return ExitEnv
	}
	files, err := batsFiles(args)
	if err == nil && len(files) == 0 {
		err = fmt.Errorf("no *.bats under %s", strings.Join(args, " "))
	}
	if err != nil {
		fmt.Fprintf(s.Err, "lint-bats-heredoc: %v\n", err)
		return ExitEnv
	}
	code := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(s.Err, "lint-bats-heredoc: %v\n", err)
			return ExitEnv
		}
		for _, n := range HeredocTestLines(data) {
			fmt.Fprintf(s.Out, "FAIL  bats-heredoc-test  %s:%d  a here-document line starts with @test: bats reads it as a "+
				"case of this file (bats 1.10: Duplicate test name(s)); write a placeholder and sed it: sed 's/^%%test /@test /'\n", f, n)
			code = 1
		}
	}
	return code
}

// batsFiles expands the arguments: a file is taken as given, a directory walked for *.bats.
func batsFiles(args []string) ([]string, error) {
	var files []string
	for _, a := range args {
		st, err := os.Stat(a)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() {
			files = append(files, a)
			continue
		}
		err = filepath.WalkDir(a, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && p != a && (d.Name() == "fixtures" || d.Name() == "testdata") {
				return fs.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(p, ".bats") {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}
