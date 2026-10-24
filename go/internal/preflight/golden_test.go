package preflight

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

// TestGolden runs every testdata case recorded from the bash preflight.sh (each tests/preflight.bats run,
// plus edge cases), in the golden format of internal/lint/rules/golden with four kinds of section key:
// "@env" holds the KEY=VALUE environment of the run (nothing else is visible: PATH points at the case's
// fakebin, N4_FSTYPE_MOUNTS_FILE at its mounts file), "<dir>/" is an empty directory ("<dir>/ (locked)"
// one with mode 000), "<file> (exec)" an executable file and "<link> (symlink)" a symlink to the body.
// Fake commands (lsof, bin/java) are scripts in the tree that the System host runs.
func TestGolden(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) < 120 {
		t.Fatalf("found %d golden cases, want at least 120", len(paths))
	}
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			c, err := golden.Read(p)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			env := map[string]string{}
			for key, body := range c.Files {
				if strings.HasSuffix(key, "/ (locked)") && os.Geteuid() == 0 {
					t.Skip("running as root: mode 000 does not block reads")
				}
				materialize(t, dir, key, strings.ReplaceAll(body, "@DIR@", dir), env)
			}
			args := make([]string, len(c.Args))
			for i, a := range c.Args {
				args[i] = strings.ReplaceAll(a, "@DIR@", dir)
			}
			var o, e bytes.Buffer
			code := Main(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
			got := strings.ReplaceAll(fmt.Sprintf("exit %d\n%s", code, o.String()), dir, "@DIR@")
			if stderr := strings.ReplaceAll(e.String(), dir, "@DIR@"); got != c.Want || stderr != c.Stderr {
				t.Errorf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr)
			}
		})
	}
}

// materialize creates one golden section under dir (or, for "@env", reads its variables into env).
func materialize(t *testing.T, dir, key, body string, env map[string]string) {
	t.Helper()
	if key == "@env" {
		for _, kv := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
			k, v, _ := strings.Cut(kv, "=")
			env[k] = v
		}
		return
	}
	name, kind, _ := strings.Cut(key, " (")
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	var err error
	switch kind {
	case "symlink)":
		err = os.Symlink(body, full)
	case "locked)":
		if err = os.MkdirAll(full, 0o755); err == nil {
			err = os.Chmod(full, 0)
			t.Cleanup(func() { _ = os.Chmod(full, 0o755) })
		}
	case "exec)":
		err = os.WriteFile(full, []byte(body), 0o755)
	default:
		if strings.HasSuffix(name, "/") {
			err = os.MkdirAll(full, 0o755)
		} else {
			err = os.WriteFile(full, []byte(body), 0o644)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}
