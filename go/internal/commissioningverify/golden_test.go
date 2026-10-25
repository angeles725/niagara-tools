package commissioningverify

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden replays every testdata case (README.md: recorded from the bash commissioning-verify.sh by
// golden/record/record.sh, or a truth-* case) through golden.Runner, in its tree as cwd, with only the case's
// environment: the source lints run through a copy of the real toolbelt whose members are symlinks (the
// case's toolbelt/<member> stubs kept), so their wrappers build nothing under the case's minimal PATH.
func TestGolden(t *testing.T) {
	repo, err := filepath.Abs("../../..")
	must(t, err)
	golden.Runner{
		Cases: 88,
		Subst: map[string]string{"@REPO@": repo},
		Setup: func(t *testing.T, dir string, env map[string]string) []string {
			env[ToolbeltEnv] = filepath.Join(dir, "toolbelt")
			return toolbeltCopy(t, repo, dir)
		},
		Main: func(env, args []string, s cli.Stdio) int { return mainEnv(args, s, env) },
	}.Run(t)
}

var buildTools sync.Once

// toolbeltCopy makes dir/toolbelt the toolbelt testdata/setup.sh gives the recording (without the script
// itself) and returns the entries it made. The first call builds the lints' Go binaries with the host env.
func toolbeltCopy(t *testing.T, repo, dir string) []string {
	t.Helper()
	buildTools.Do(func() {
		cmd := exec.Command("bash", "-c", `. go/go-tool.sh && go_tool_bin "$PWD/go" n4lint >/dev/null`)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building n4lint: %v\n%s", err, out)
		}
	})
	tb, cp := filepath.Join(repo, "build-n4-module-kit", "toolbelt"), filepath.Join(dir, "toolbelt")
	var made []string
	if _, err := os.Lstat(cp); err != nil {
		made = append(made, "toolbelt")
	}
	must(t, os.MkdirAll(cp, 0o755))
	entries, err := os.ReadDir(tb)
	must(t, err)
	for _, e := range entries {
		if _, err := os.Lstat(filepath.Join(cp, e.Name())); err == nil {
			continue // the case's stub
		}
		must(t, os.Symlink(filepath.Join(tb, e.Name()), filepath.Join(cp, e.Name())))
		made = append(made, "toolbelt/"+e.Name())
	}
	return made
}
