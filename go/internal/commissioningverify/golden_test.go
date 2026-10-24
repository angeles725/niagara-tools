package commissioningverify

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
func materialize(t *testing.T, dir, repo string, c golden.Case) string {
	t.Helper()
	tb := filepath.Join(repo, "build-n4-module-kit", "toolbelt")
	sub := strings.NewReplacer("@DIR@", dir, "@REPO@", repo)
	stubs, modes := map[string]string{}, map[string]os.FileMode{}
	for _, d := range c.Dirs { // golden.Read reads "<path> (dir)" sections into Dirs
		must(t, os.MkdirAll(filepath.Join(dir, d), 0o755))
	}
	for name, text := range c.Files {
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

// rootVerdict decides a case holding "(mode 000)" files for a run as uid with RootOptIn set to optIn: "" runs it;
// otherwise skip says whether to skip (optIn "1") or fail, and msg says why, naming the first such file in sorted
// order so the message is deterministic.
func rootVerdict(uid int, optIn string, files map[string]string) (skip bool, msg string) {
	var locked []string
	for name := range files {
		if strings.HasSuffix(name, " (mode 000)") {
			locked = append(locked, name)
		}
	}
	if uid != 0 || len(locked) == 0 {
		return false, ""
	}
	sort.Strings(locked)
	if optIn == "1" {
		return true, "running as root: chmod 000 does not block reads (" + RootOptIn + "=1)"
	}
	return false, fmt.Sprintf("running as root: %s cannot be made unreadable; run as a normal user or set %s=1 to skip", locked[0], RootOptIn)
}

// unreadableAsRoot applies rootVerdict to this process. Callers run it before anything of the case is written, so
// a case is never skipped half materialized.
func unreadableAsRoot(t *testing.T, files map[string]string) {
	t.Helper()
	switch skip, msg := rootVerdict(os.Geteuid(), os.Getenv(RootOptIn), files); {
	case skip:
		t.Skip(msg)
	case msg != "":
		t.Fatal(msg)
	}
}

// Case counts of testdata (README.md): recorded from the bash tool, and truth-* cases whose expectation is a
// declared fix. Exact, so a lost or stray case file fails.
const (
	recordedCases = 77
	truthCases    = 11
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

func TestRootVerdict(t *testing.T) {
	t.Parallel()
	locked := map[string]string{"z (mode 000)": "", "a (mode 000)": "", "b": ""}
	fail := "running as root: a (mode 000) cannot be made unreadable; run as a normal user or set " + RootOptIn + "=1 to skip"
	for _, c := range []struct {
		uid   int
		optIn string
		files map[string]string
		skip  bool
		msg   string
	}{
		{1000, "", locked, false, ""},
		{0, "", map[string]string{"b": ""}, false, ""},
		{0, "", locked, false, fail},
		{0, "yes", locked, false, fail},
		{0, "1", locked, true, "running as root: chmod 000 does not block reads (" + RootOptIn + "=1)"},
	} {
		if skip, msg := rootVerdict(c.uid, c.optIn, c.files); skip != c.skip || msg != c.msg {
			t.Errorf("rootVerdict(%d, %q) = %v, %q; want %v, %q", c.uid, c.optIn, skip, msg, c.skip, c.msg)
		}
	}
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
			tb := materialize(t, dir, repo, c)
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
