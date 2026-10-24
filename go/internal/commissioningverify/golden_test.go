package commissioningverify

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// materialize writes a case's tree under dir and returns the toolbelt to run. "@DIR@" and "@REPO@" in a section
// stand for dir and the repository root. "<path> (symlink)" is a symlink to the section's one line, "<path> (dir)"
// an empty directory, "<path> (mode 000)" an unreadable file; "toolbelt/<member>" sections make dir/toolbelt a
// copy of the toolbelt with those members replaced by executable stubs (tests/helpers/stub-toolbelt.bash).
func materialize(t *testing.T, dir, repo string, files map[string]string) string {
	t.Helper()
	tb := filepath.Join(repo, "build-n4-module-kit", "toolbelt")
	sub := strings.NewReplacer("@DIR@", dir, "@REPO@", repo)
	stubs := map[string]string{}
	for name, text := range files {
		if m, ok := strings.CutPrefix(name, "toolbelt/"); ok {
			stubs[m] = text
			continue
		}
		path, kind, _ := strings.Cut(name, " (")
		full := filepath.Join(dir, path)
		must(t, os.MkdirAll(filepath.Dir(full), 0o755))
		switch kind {
		case "symlink)":
			must(t, os.Symlink(sub.Replace(strings.TrimSuffix(text, "\n")), full))
		case "dir)":
			must(t, os.MkdirAll(full, 0o755))
		default:
			must(t, os.WriteFile(full, []byte(text), 0o644))
		}
		if kind == "mode 000)" {
			if os.Geteuid() == 0 {
				t.Skip("running as root: chmod 000 does not block reads")
			}
			must(t, os.Chmod(full, 0))
			t.Cleanup(func() { os.Chmod(full, 0o644) })
		}
	}
	if len(stubs) == 0 {
		return tb
	}
	cp := filepath.Join(dir, "toolbelt")
	must(t, os.Mkdir(cp, 0o755))
	entries, err := os.ReadDir(tb)
	must(t, err)
	for _, e := range entries {
		if _, stubbed := stubs[e.Name()]; !stubbed {
			must(t, os.Symlink(filepath.Join(tb, e.Name()), filepath.Join(cp, e.Name())))
		}
	}
	for m, text := range stubs {
		must(t, os.WriteFile(filepath.Join(cp, m), []byte(text), 0o755))
	}
	return cp
}

// TestGolden replays every testdata case, recorded from the bash commissioning-verify.sh on the same tree and
// toolbelt, and compares exit code, stdout and stderr byte for byte. The source lints run through the real
// toolbelt wrappers (go/bin/n4lint is built on demand), so cases run one at a time in their tree as cwd.
func TestGolden(t *testing.T) {
	repo, err := filepath.Abs("../../..")
	must(t, err)
	paths, _ := filepath.Glob(filepath.Join(repo, "go/internal/commissioningverify/testdata/*.txt"))
	if len(paths) < 86 {
		t.Fatalf("found %d golden cases, want at least 86", len(paths))
	}
	cwd, err := os.Getwd()
	must(t, err)
	defer os.Chdir(cwd)
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			c, err := golden.Read(p)
			must(t, err)
			dir := t.TempDir()
			tb := materialize(t, dir, repo, c.Files)
			sub := strings.NewReplacer("@DIR@", dir, "@REPO@", repo)
			args := make([]string, len(c.Args))
			for i, a := range c.Args {
				args[i] = sub.Replace(a)
			}
			must(t, os.Chdir(dir))
			var o, e bytes.Buffer
			getenv := func(n string) string {
				if n == ToolbeltEnv {
					return tb
				}
				return os.Getenv(n)
			}
			code := mainEnv(args, cli.Stdio{Out: &o, Err: &e, Getenv: getenv}, minimalEnv())
			back := strings.NewReplacer(dir, "@DIR@", repo, "@REPO@")
			got, stderr := back.Replace(fmt.Sprintf("exit %d\n%s", code, o.String())), back.Replace(e.String())
			if got != c.Want || stderr != c.Stderr {
				t.Errorf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr)
			}
		})
	}
}
