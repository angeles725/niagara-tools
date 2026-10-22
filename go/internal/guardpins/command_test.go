package guardpins

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usageText = "usage: lint-guard-pins.sh [--strict] <repo-root>\n"

// mklint mirrors tests/guard-pins.bats _mklint: toolbelt/<s>.sh with an optional mutation id and
// tests/<s>.bats with an optional @test "<def>: a guard".
func mklint(t *testing.T, d, s, mid, def string) {
	t.Helper()
	for _, sub := range []string{"toolbelt", "tests"} {
		if err := os.MkdirAll(filepath.Join(d, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sh := "#!/usr/bin/env bash\n# " + s + "\n"
	if mid != "" {
		sh += "# Mutation: " + mid + " -- drops the guard\n"
	}
	bats := "#!/usr/bin/env bats\n"
	if def != "" {
		bats += "@test \"" + def + ": a guard\" { true; }\n"
	}
	write(t, filepath.Join(d, "toolbelt", s+".sh"), sh)
	write(t, filepath.Join(d, "tests", s+".bats"), bats)
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

const noMut = "  no # Mutation: line (K24(7): add '# Mutation: <fixture-id> -- <what it flips>')\n"

func TestMain(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	match, nofix, nomut, scope := filepath.Join(d, "match"), filepath.Join(d, "nofix"), filepath.Join(d, "nomut"), filepath.Join(d, "scope")
	mklint(t, match, "lint-foo", "FOO1", "FOO1")
	mklint(t, nofix, "lint-bar", "BAR9", "")
	mklint(t, nomut, "lint-baz", "", "")
	mklint(t, scope, "lint-ok", "OK1", "OK1")
	write(t, filepath.Join(scope, "toolbelt", "report-module.sh"), "#!/usr/bin/env bash\n# report-module\n")
	nested := filepath.Join(d, "nested")
	mklint(t, filepath.Join(d, "src"), "lint-a", "A", "A")
	for _, sub := range []string{"build-n4-module-kit/toolbelt", "tests"} {
		if err := os.MkdirAll(filepath.Join(nested, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(nested, "build-n4-module-kit", "toolbelt", "lint-a.sh"), "#!/bin/sh\n# Mutation: A -- x\n")
	write(t, filepath.Join(nested, "tests", "a.bats"), "@test \"A: x\" { true; }\n")
	notests := filepath.Join(d, "notests")
	if err := os.MkdirAll(filepath.Join(notests, "toolbelt"), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		args      []string
		code      int
		out, errs string
	}{
		{"T4-usage: no kit-root", nil, ExitEnv, "", usageText},
		{"--strict alone", []string{"--strict"}, ExitEnv, "", usageText},
		{"two roots", []string{match, match}, ExitEnv, "", usageText},
		{"--strict twice", []string{"--strict", "--strict", match}, ExitEnv, "", usageText},
		{"--strict after the root", []string{match, "--strict"}, ExitEnv, "", usageText},
		{"no toolbelt", []string{filepath.Join(d, "nope")}, ExitEnv, "", "lint-guard-pins: no toolbelt/ found under " + filepath.Join(d, "nope") + "\n"},
		{"empty root", []string{""}, ExitEnv, "", "lint-guard-pins: no toolbelt/ found under \n"},
		{"no tests", []string{notests}, ExitEnv, "", "lint-guard-pins: no tests/ found under " + notests + "\n"},
		{"T4-match", []string{match}, ExitOK,
			"MATCH  lint-guard-pins  " + match + "/toolbelt/lint-foo.sh:3  mutation FOO1 -> tests/lint-foo.bats:2\n", ""},
		{"T4-nofixture", []string{nofix}, ExitOK, "WARN  lint-guard-pins  " + nofix + "/toolbelt/lint-bar.sh:3  mutation BAR9 -> no fixture\n", ""},
		{"T4-nomutation", []string{nomut}, ExitOK, "WARN  lint-guard-pins  " + nomut + "/toolbelt/lint-baz.sh" + noMut, ""},
		{"T4-strict", []string{"--strict", nomut}, ExitWarn, "WARN  lint-guard-pins  " + nomut + "/toolbelt/lint-baz.sh" + noMut, ""},
		{"--strict, no WARN", []string{"--strict", match}, ExitOK,
			"MATCH  lint-guard-pins  " + match + "/toolbelt/lint-foo.sh:3  mutation FOO1 -> tests/lint-foo.bats:2\n", ""},
		{"T4-scope: non-lint scripts are not scanned", []string{scope}, ExitOK,
			"MATCH  lint-guard-pins  " + scope + "/toolbelt/lint-ok.sh:3  mutation OK1 -> tests/lint-ok.bats:2\n", ""},
		{"a trailing slash is kept in the printed path", []string{match + "/"}, ExitOK,
			"MATCH  lint-guard-pins  " + match + "//toolbelt/lint-foo.sh:3  mutation FOO1 -> tests/lint-foo.bats:2\n", ""},
		{"<root>/build-n4-module-kit/toolbelt with <root>/tests", []string{nested}, ExitOK,
			"MATCH  lint-guard-pins  " + nested + "/build-n4-module-kit/toolbelt/lint-a.sh:2  mutation A -> tests/a.bats:1\n", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, out, errs := run(tt.args...)
			if code != tt.code || out != tt.out || errs != tt.errs {
				t.Errorf("got (%d, %q, %q)\nwant (%d, %q, %q)", code, out, errs, tt.code, tt.out, tt.errs)
			}
		})
	}
}

// TestMainOddEntries: every toolbelt entry named lint-*.sh is scanned (a directory, a broken link
// or an unreadable file is a WARN plus a stderr note, never skipped); other names and sub-directories
// are not.
func TestMainOddEntries(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 file")
	}
	k := filepath.Join(t.TempDir(), "k")
	mklint(t, k, "lint-a", "A", "A")
	tb := filepath.Join(k, "toolbelt")
	write(t, filepath.Join(tb, "lint-empty.sh"), "")
	write(t, filepath.Join(tb, "lint-a.txt"), "# Mutation: A -- x\n")
	write(t, filepath.Join(tb, "lint-locked.sh"), "# Mutation: A -- x\n")
	if err := os.Chmod(filepath.Join(tb, "lint-locked.sh"), 0); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tb, "lint-dir.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("lint-a.sh", filepath.Join(tb, "lint-link.sh")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tb, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(tb, "sub", "lint-sub.sh"), "# Mutation: A -- x\n")
	code, out, errs := run(k)
	want := "MATCH  lint-guard-pins  " + tb + "/lint-a.sh:3  mutation A -> tests/lint-a.bats:2\n" +
		"WARN  lint-guard-pins  " + tb + "/lint-dir.sh" + noMut +
		"WARN  lint-guard-pins  " + tb + "/lint-empty.sh" + noMut +
		"MATCH  lint-guard-pins  " + tb + "/lint-link.sh:3  mutation A -> tests/lint-a.bats:2\n" +
		"WARN  lint-guard-pins  " + tb + "/lint-locked.sh" + noMut
	if code != ExitOK || out != want {
		t.Errorf("got (%d, %q)\nwant %q", code, out, want)
	}
	if !strings.Contains(errs, "lint-guard-pins: cannot read "+tb+"/lint-dir.sh") || !strings.Contains(errs, "lint-guard-pins: cannot read "+tb+"/lint-locked.sh") {
		t.Errorf("stderr = %q", errs)
	}
}

// TestMainKitRootGiven: the argument is the repository root (it holds tests/); a kit root, whose
// toolbelt/ resolves but whose tests/ is its parent's, is exit 3 naming the repository root to pass.
func TestMainKitRootGiven(t *testing.T) {
	t.Parallel()
	repo := filepath.Join(t.TempDir(), "repo")
	kit := filepath.Join(repo, "build-n4-module-kit")
	for _, d := range []string{filepath.Join(kit, "toolbelt"), filepath.Join(repo, "tests")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(kit, "toolbelt", "lint-a.sh"), "#!/bin/sh\n# Mutation: A -- x\n")
	write(t, filepath.Join(repo, "tests", "a.bats"), "@test \"A: x\" { true; }\n")
	code, out, errs := run(kit)
	want := "lint-guard-pins: no tests/ found under " + kit + " (a kit root? pass the repository root " + repo + ")\n"
	if code != ExitEnv || out != "" || errs != want {
		t.Errorf("kit root: got (%d, %q, %q)\nwant errs %q", code, out, errs, want)
	}
	code, out, _ = run(repo)
	if code != ExitOK || out != "MATCH  lint-guard-pins  "+kit+"/toolbelt/lint-a.sh:2  mutation A -> tests/a.bats:1\n" {
		t.Errorf("repo root: got (%d, %q)", code, out)
	}
}

// TestMainUnreadableToolbelt: a toolbelt/ that cannot be listed is exit 3 (bash `find` printed an
// error and the lint passed with zero rows, even under --strict).
func TestMainUnreadableToolbelt(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root lists a mode-000 directory")
	}
	k := filepath.Join(t.TempDir(), "k")
	mklint(t, k, "lint-a", "A", "A")
	tb := filepath.Join(k, "toolbelt")
	if err := os.Chmod(tb, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tb, 0o755) })
	code, out, errs := run("--strict", k)
	if code != ExitEnv || out != "" || !strings.HasPrefix(errs, "lint-guard-pins: cannot list "+tb+": ") {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}

// TestMainRealKit is T4-smoke: the repository's own lints all resolve, S21-misparse to its line.
func TestMainRealKit(t *testing.T) {
	t.Parallel()
	repo := filepath.Join("..", "..", "..")
	code, out, errs := run("--strict", repo)
	if code != ExitOK || errs != "" || strings.Contains(out, "WARN") {
		t.Fatalf("got (%d, %q, %q)", code, out, errs)
	}
	scripts := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || f[0] != "MATCH" {
			t.Fatalf("not a MATCH row: %q", line)
		}
		scripts[strings.SplitN(f[2], ":", 2)[0]] = true
	}
	if len(scripts) < 13 {
		t.Errorf("%d distinct scripts, want >= 13", len(scripts))
	}
	b, err := os.ReadFile(filepath.Join(repo, "tests", "lint-timers.bats"))
	if err != nil {
		t.Fatal(err)
	}
	ln := 0
	for i, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "@test \"S21-misparse") {
			ln = i + 1
			break
		}
	}
	if want := "mutation S21-misparse -> tests/lint-timers.bats:"; ln == 0 || !strings.Contains(out, want+strconv.Itoa(ln)+"\n") {
		t.Errorf("S21-misparse row missing (line %d)", ln)
	}
}
