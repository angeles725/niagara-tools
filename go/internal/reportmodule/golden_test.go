package reportmodule

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		Cases: 122,
		Subst: map[string]string{"@REPO@": repo},
		Setup: func(t *testing.T, dir string, env map[string]string) []string {
			env[ToolbeltEnv] = filepath.Join(dir, "toolbelt")
			return toolbeltCopy(t, repo, dir)
		},
		Main: func(env, args []string, s cli.Stdio) int { return mainEnv(args, s, env) },
	}.Run(t)
}

var buildTools sync.Once

// goToolRe finds the Go binary a toolbelt wrapper execs (testdata/setup.sh greps the same line).
var goToolRe = regexp.MustCompile(`go_tool_bin "\$GO_DIR" ([A-Za-z0-9-]+)\)`)

// memberTools are the Go binaries the toolbelt wrappers exec, read from the wrappers themselves so a member
// ported to Go later is built too (a hand list missed triage-console and the case env has no Go toolchain).
func memberTools(t *testing.T, tb string) []string {
	t.Helper()
	scripts, err := filepath.Glob(filepath.Join(tb, "*.sh"))
	must(t, err)
	seen := map[string]bool{}
	var tools []string
	for _, sc := range scripts {
		b, err := os.ReadFile(sc)
		must(t, err)
		for _, m := range goToolRe.FindAllSubmatch(b, -1) {
			if n := string(m[1]); !seen[n] {
				seen[n] = true
				tools = append(tools, n)
			}
		}
	}
	return tools
}

// toolbeltCopy makes dir/toolbelt the toolbelt testdata/setup.sh gives the recording and returns the entries it
// made. The first call builds the members' Go binaries with the host env.
func toolbeltCopy(t *testing.T, repo, dir string) []string {
	t.Helper()
	buildTools.Do(func() {
		script := `. go/go-tool.sh && for t in "$@"; do go_tool_bin "$PWD/go" "$t" >/dev/null || exit 1; done`
		tools := memberTools(t, filepath.Join(repo, "build-n4-module-kit", "toolbelt"))
		cmd := exec.Command("bash", append([]string{"-c", script, "build"}, tools...)...)
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
		switch {
		case stubRest && filepath.Ext(name) == ".sh":
			must(t, os.WriteFile(at, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755))
		case name == "eslint" && e.IsDir():
			// Hermetic: toolbelt/eslint/ is linked file by file WITHOUT node_modules/, so a checkout where the pinned
			// npm tools are installed (CI's bats job, a ci-local replay) runs the same PATH-fallback cases as one
			// without; a case that needs a kit eslint brings its own toolbelt/eslint.
			must(t, os.Mkdir(at, 0o755))
			sub, err := os.ReadDir(filepath.Join(tb, name))
			must(t, err)
			for _, f := range sub {
				if f.Name() != "node_modules" {
					must(t, os.Symlink(filepath.Join(tb, name, f.Name()), filepath.Join(at, f.Name())))
				}
			}
		default:
			must(t, os.Symlink(filepath.Join(tb, name), at))
		}
		made = append(made, "toolbelt/"+name)
	}
	return made
}
