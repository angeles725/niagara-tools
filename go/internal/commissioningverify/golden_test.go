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
// copy of the toolbelt with those members replaced by executable stubs (tests/helpers/stub-toolbelt.bash);
// "toolbelt/<member> (mode 644)" is such a stub written without the execute bit.
func materialize(t *testing.T, dir, repo string, files map[string]string) string {
	t.Helper()
	tb := filepath.Join(repo, "build-n4-module-kit", "toolbelt")
	sub := strings.NewReplacer("@DIR@", dir, "@REPO@", repo)
	stubs, modes := map[string]string{}, map[string]os.FileMode{}
	for name, text := range files {
		if m, ok := strings.CutPrefix(name, "toolbelt/"); ok {
			m, kind, _ := strings.Cut(m, " (")
			stubs[m], modes[m] = text, 0o755
			if kind == "mode 644)" {
				modes[m] = 0o644
			}
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
		p := filepath.Join(cp, m)
		must(t, os.WriteFile(p, []byte(text), modes[m]))
		must(t, os.Chmod(p, modes[m])) // WriteFile's mode is masked by the umask
	}
	return cp
}

// RootOptIn is the variable that lets a run as root skip the cases holding a "(mode 000)" file (root reads it,
// so the case would test nothing); without it such a run fails.
const RootOptIn = "CV_GOLDEN_ALLOW_ROOT_SKIP"

// unreadableAsRoot fails (or, with RootOptIn=1, skips) a case holding a "(mode 000)" file when the test runs as
// root. It runs before anything of the case is written, so a case is never skipped half materialized.
func unreadableAsRoot(t *testing.T, files map[string]string) {
	t.Helper()
	for name := range files {
		if !strings.HasSuffix(name, " (mode 000)") || os.Geteuid() != 0 {
			continue
		}
		if os.Getenv(RootOptIn) == "1" {
			t.Skip("running as root: chmod 000 does not block reads (" + RootOptIn + "=1)")
		}
		t.Fatalf("running as root: %s cannot be made unreadable; run as a normal user or set %s=1 to skip", name, RootOptIn)
	}
}

// Case counts of testdata (README.md): recorded from the bash tool, and truth-* cases whose expectation is a
// declared fix. Exact, so a lost or stray case file fails.
const (
	recordedCases = 77
	truthCases    = 10
)

// cases lists the testdata case files (absolute: the cases run with their tree as cwd) and checks the counts.
func cases(t *testing.T, repo string) []string {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join(repo, "go/internal/commissioningverify/testdata/*.txt"))
	truth := 0
	for _, p := range paths {
		if strings.HasPrefix(filepath.Base(p), "truth-") {
			truth++
		}
	}
	if len(paths)-truth != recordedCases || truth != truthCases {
		t.Fatalf("found %d recorded and %d truth-* cases, want %d and %d", len(paths)-truth, truth, recordedCases, truthCases)
	}
	return paths
}

// TestGolden replays every testdata case, recorded from the bash commissioning-verify.sh on the same tree and
// toolbelt, and compares exit code, stdout and stderr byte for byte. The source lints run through the real
// toolbelt wrappers (go/bin/n4lint is built on demand), so cases run one at a time in their tree as cwd.
func TestGolden(t *testing.T) {
	repo, err := filepath.Abs("../../..")
	must(t, err)
	paths := cases(t, repo)
	cwd, err := os.Getwd()
	must(t, err)
	defer os.Chdir(cwd)
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			c, err := golden.Read(p)
			must(t, err)
			unreadableAsRoot(t, c.Files)
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
