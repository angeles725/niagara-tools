package citeresolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOutsideRepo pins retro 2026-11-13-remove-dead-bash-lint-libs Δ2: an [ev: …] token that names a home or
// absolute path is OUTSIDE_REPO whether or not the file exists on this machine; prose and [deferred: …] keep
// resolving by existence.
func TestOutsideRepo(t *testing.T) {
	t.Parallel()
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"home file that exists here", "[ev: ~/notes/home-plan.md §3 Provenance markers]", []want{{OutsideRepo, "~/notes/home-plan.md"}}},
		{"home directory", "[ev: ~/notes/ model: frontmatter]", []want{{OutsideRepo, "outside the repository"}}},
		{"home file missing here", "[ev: ~/notes/nope.md]", []want{{OutsideRepo, "~/notes/nope.md"}}},
		{"absolute path", "[ev: /etc/hostname]", []want{{OutsideRepo, "/etc/hostname"}}},
		{"absolute file after code", "[ev: code /opt/x/Foo.java:3]", []want{{OutsideRepo, "/opt/x/Foo.java"}}},
		{"absolute zip", "[ev: code /opt/x/a.zip!b/C.java]", []want{{OutsideRepo, "/opt/x/a.zip"}}},
		{"a repo-relative tag still resolves", "[ev: tests/awk.bats AWB1]", []want{{Resolved, ""}}},
		{"a kit path is never home-expanded", "[ev: kit ~/notes/home-plan.md]", []want{{MissingFile, "~/notes/home-plan.md"}}},
	})
}

// TestRepoPath pins Δ1: a backticked toolbelt/, scripts/, go/ or tests/ path of prose must exist under the
// kit or the repository root; a generated, uncommitted path resolves on any machine.
func TestRepoPath(t *testing.T) {
	t.Parallel()
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"repo file", "see `tests/awk.bats`", []want{{Resolved, ""}}},
		{"kit file", "see `types/ok.md` and `toolbelt/../types/ok.md`", nil},
		{"deleted file", "reuse `toolbelt/lib/awk-lint.sh`", []want{{MissingFile, "toolbelt/lib/awk-lint.sh under the kit or the repository"}}},
		{"no basename search", "reuse `tests/kit/awk.bats`", []want{{MissingFile, ""}}},
		{"generated lint index", "see `toolbelt/INDEX.md`", []want{{Resolved, "generated, not committed"}}},
		{"built binary", "run `go/bin/kit-cite-resolve`", []want{{Resolved, "generated, not committed"}}},
	})
}

// TestMainRepoPathAndOutsideByScope: through the command, a repo path is checked on live cards only (history
// keeps old names), an OUTSIDE_REPO on a history card is HISTORICAL_DRIFT, and on a live card both fail --strict.
func TestMainRepoPathAndOutsideByScope(t *testing.T) {
	t.Parallel()
	r := fixtureRoots(t)
	kit := t.TempDir()
	text := "reuse `toolbelt/lib/awk-lint.sh` [ev: ~/notes/home-plan.md]\n"
	live, retro := filepath.Join(kit, "types", "x.md"), filepath.Join(kit, "retros", "x.md")
	for path, body := range map[string]string{filepath.Join(kit, "BUILD-STATE.src.md"): "# s\n", live: text, retro: text} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := []string{"--kit", kit, "--research", r.Research, "--modules", r.Modules, "--memory", r.Memory}
	code, out, errs := run(t, append(base, "--strict", live, retro)...)
	var got []string
	for _, row := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Split(row, "\t")
		got = append(got, filepath.Base(filepath.Dir(f[0]))+" "+f[2]+" "+f[5])
	}
	want := "types path OUTSIDE_REPO|types repo-path MISSING_FILE|retros path HISTORICAL_DRIFT"
	if code != 1 || strings.Join(got, "|") != want || !strings.Contains(errs, "strict findings: LIVE 2, HISTORY 0") {
		t.Errorf("exit %d, rows %q, stderr %q; want exit 1, rows %q, LIVE 2 findings", code, strings.Join(got, "|"), errs, want)
	}
	if code, _, _ := run(t, append(base, "--strict", retro)...); code != 0 {
		t.Errorf("history card alone: exit %d, want 0", code)
	}
}
