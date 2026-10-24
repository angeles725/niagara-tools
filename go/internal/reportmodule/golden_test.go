package reportmodule

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden replays every testdata case (README.md: recorded from the bash report-module.sh by
// golden/record/record.sh, or a truth-* case) through golden.Runner, in its tree as cwd, with only the case's
// environment: the members run through a copy of the real toolbelt whose entries are symlinks (the case's
// toolbelt/<entry> sections kept, every other member a no-op stub when the case holds toolbelt/.stub-rest), so
// their wrappers build nothing under the case's minimal PATH.
func TestGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("runs every toolbelt member as a subprocess")
	}
	repo, err := filepath.Abs("../../..")
	must(t, err)
	golden.Runner{
		Cases: 117,
		Subst: map[string]string{"@REPO@": repo},
		Setup: func(t *testing.T, dir string, env map[string]string) []string {
			env[ToolbeltEnv] = filepath.Join(dir, "toolbelt")
			return toolbeltCopy(t, repo, dir)
		},
		Main: func(env, args []string, s cli.Stdio) int { return mainEnv(args, s, env) },
	}.Run(t)
}

var buildTools sync.Once

// memberTools are the Go binaries the toolbelt member wrappers exec (testdata/setup.sh builds the same).
var memberTools = []string{"n4lint", "verify-module", "slot-coverage", "schema-risk", "rc-scan", "lint-bundled-jar-class-version"}

// toolbeltCopy makes dir/toolbelt the toolbelt testdata/setup.sh gives the recording and returns the entries it
// made. The first call builds the members' Go binaries with the host env.
func toolbeltCopy(t *testing.T, repo, dir string) []string {
	t.Helper()
	buildTools.Do(func() {
		script := `. go/go-tool.sh && for t in "$@"; do go_tool_bin "$PWD/go" "$t" >/dev/null || exit 1; done`
		cmd := exec.Command("bash", append([]string{"-c", script, "build"}, memberTools...)...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building the member tools: %v\n%s", err, out)
		}
	})
	tb, cp := filepath.Join(repo, "build-n4-module-kit", "toolbelt"), filepath.Join(dir, "toolbelt")
	var made []string
	if _, err := os.Lstat(cp); err != nil {
		made = append(made, "toolbelt")
	}
	_, err := os.Stat(filepath.Join(cp, ".stub-rest"))
	stubRest := err == nil
	must(t, os.MkdirAll(cp, 0o755))
	entries, err := os.ReadDir(tb)
	must(t, err)
	for _, e := range entries {
		name, at := e.Name(), filepath.Join(cp, e.Name())
		if _, err := os.Lstat(at); err == nil {
			continue // the case's own entry
		}
		if stubRest && filepath.Ext(name) == ".sh" {
			must(t, os.WriteFile(at, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755))
		} else {
			must(t, os.Symlink(filepath.Join(tb, name), at))
		}
		made = append(made, "toolbelt/"+name)
	}
	return made
}
