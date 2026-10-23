package buildstate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usageLine = "usage: gen-build-state.sh [--check] [<kit-root>]\n"

const archiveText = "<!-- ledger: kit -->\n<!-- date: 2026-10-22 -->\n<!-- last-change-note: old note -->\n" +
	"log: 2026-10-23 · late line\nlog: 2026-09-01 · early line\nopen_issue: OLD A\nopen_issue: OLD B\n"

// kit makes a kit root with BUILD-STATE.md (content) and ledger.d/ holding the given fragments.
func kit(t *testing.T, content string, frags map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "ledger.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "BUILD-STATE.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, text := range frags {
		if err := os.WriteFile(filepath.Join(root, "ledger.d", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func run(defaultRoot string, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(name string) string {
		if name == DefaultRootEnv {
			return defaultRoot
		}
		return ""
	}}
	code = Main(args, s)
	return code, o.String(), e.String()
}

var fresh = state("last_change: 2026-10-22              # old note", "  - OLD A\n  - OLD B\n",
	"2026-10-23 · late line\n2026-09-01 · early line\n")

func TestMainUsage(t *testing.T) {
	t.Parallel()
	root := kit(t, fresh, nil)
	for _, args := range [][]string{{"--help"}, {"-x"}, {"--check", "--bogus"}, {root, root}} {
		code, out, errs := run("", args...)
		if code != ExitEnv || out != "" || errs != usageLine {
			t.Errorf("%q: got (%d, %q, %q), want (3, \"\", usage)", args, code, out, errs)
		}
	}
	if code, _, errs := run(""); code != ExitEnv || !strings.Contains(errs, DefaultRootEnv) {
		t.Errorf("no root: got (%d, %q)", code, errs)
	}
}

func TestMainMissingInputsExit3(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if code, _, errs := run("", root); code != ExitEnv || errs != "gen-build-state: no ledger.d/ under "+root+"\n" {
		t.Errorf("no ledger.d: got (%d, %q)", code, errs)
	}
	if err := os.Mkdir(filepath.Join(root, "ledger.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := run("", root); code != ExitEnv || errs != "gen-build-state: no BUILD-STATE.md under "+root+"\n" {
		t.Errorf("no BUILD-STATE.md: got (%d, %q)", code, errs)
	}
}

func TestMainCheckThenWrite(t *testing.T) {
	t.Parallel()
	root := kit(t, fresh, map[string]string{"0000-archive-pre-2026-10-23.md": archiveText})
	path := root + "/BUILD-STATE.md"
	code, out, errs := run("", "--check", root)
	if code != ExitOK || out != "gen-build-state: "+path+" is fresh\n" || errs != "" {
		t.Errorf("--check fresh: got (%d, %q, %q)", code, out, errs)
	}
	frag := "<!-- ledger: kit -->\n<!-- date: 2026-10-24 -->\n<!-- last-change-note: new -->\nlog: 2026-10-24 · new line\nopen_issue: NEW\n"
	if err := os.WriteFile(filepath.Join(root, "ledger.d", "2026-10-24-new.md"), []byte(frag), 0o644); err != nil {
		t.Fatal(err)
	}
	// Mutation: GBS-stale -- dropping the generated-vs-committed compare makes --check vacuously clean
	code, out, errs = run(root, "--check")
	want := "gen-build-state: " + path + " is stale vs ledger.d/; run gen-build-state.sh and commit the result\n"
	if code != ExitStale || out != "" || !strings.HasPrefix(errs, want) || !strings.Contains(errs, "< 2026-10-24 · new line") {
		t.Errorf("--check stale: got (%d, %q, %q)", code, out, errs)
	}
	code, out, errs = run(root)
	if code != ExitOK || out != "gen-build-state: wrote "+path+"\n" || errs != "" {
		t.Errorf("write: got (%d, %q, %q)", code, out, errs)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "last_change: 2026-10-24              # new\nopen_issues:\n  - NEW\n  - OLD A\n") ||
		!strings.HasSuffix(string(got), "2026-09-01 · early line\n2026-10-24 · new line\n<!-- /build-state.v1 -->\n") {
		t.Errorf("written file:\n%s", got)
	}
	if code, _, _ = run("", "--check", root); code != ExitOK {
		t.Errorf("--check after write: exit %d", code)
	}
}

func TestMainViolationWritesNothing(t *testing.T) {
	t.Parallel()
	bad := "<!-- ledger: kit -->\n<!-- date: 2026-10-24 -->\nlog: 2026-10-24 · x\n  - stray\n"
	root := kit(t, "hand edit\n"+fresh, map[string]string{"0000-archive-pre-2026-10-23.md": archiveText, "2026-10-24-bad.md": bad})
	code, out, errs := run("", root)
	want := "gen-build-state: 2026-10-24-bad.md: line 4: " + errLine + "\n" +
		"gen-build-state: ledger fragment violation; BUILD-STATE.md not written\n"
	if code != ExitStale || out != "" || errs != want {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
	if got, _ := os.ReadFile(root + "/BUILD-STATE.md"); string(got) != "hand edit\n"+fresh {
		t.Error("BUILD-STATE.md was rewritten despite a violation")
	}
}

func TestMainEmptyLedgerAndStructureViolation(t *testing.T) {
	t.Parallel()
	root := kit(t, fresh, nil)
	if code, _, errs := run("", root); code != ExitStale || errs != "gen-build-state: ledger.d/ holds no fragment; BUILD-STATE.md not written\n" {
		t.Errorf("empty ledger: got (%d, %q)", code, errs)
	}
	root = kit(t, strings.Replace(fresh, "module: kit ", "module: kat ", 1), map[string]string{"0000-archive-pre-2026-10-23.md": archiveText})
	if code, _, errs := run("", "--check", root); code != ExitStale || errs != "gen-build-state: "+errNoKit+"; BUILD-STATE.md not written\n" {
		t.Errorf("no kit envelope: got (%d, %q)", code, errs)
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()
	root := kit(t, fresh, map[string]string{"0000-archive-pre-2026-10-23.md": archiveText})
	var diag bytes.Buffer
	if !Check(root, &diag) || diag.String() != "" {
		t.Errorf("fresh kit: Check = false or diagnostics %q", diag.String())
	}
	frag := "<!-- ledger: kit -->\n<!-- date: 2026-10-24 -->\n<!-- last-change-note: new -->\nlog: 2026-10-24 · new line\n"
	if err := os.WriteFile(filepath.Join(root, "ledger.d", "2026-10-24-new.md"), []byte(frag), 0o644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(root + "/BUILD-STATE.md")
	diag.Reset()
	if Check(root, &diag) || !strings.HasPrefix(diag.String(), "gen-build-state: "+root+"/BUILD-STATE.md is stale vs ledger.d/;") {
		t.Errorf("stale kit: Check = true or diagnostics %q", diag.String())
	}
	if after, _ := os.ReadFile(root + "/BUILD-STATE.md"); string(after) != string(before) {
		t.Error("Check rewrote BUILD-STATE.md")
	}
}

// TestMainCommittedBytes ports the key cases of the gen.File parity matrix (#344). BUILD-STATE.md is
// compared byte-exact, but the generator only rewrites the kit block and carries every byte outside
// it through: an extra trailing newline is fresh because the generated file keeps it too, while a
// stale kit block is stale. A read-only BUILD-STATE.md is replaced atomically (its directory is
// writable) and keeps its mode.
func TestMainCommittedBytes(t *testing.T) {
	t.Parallel()
	root := kit(t, fresh+"\n", map[string]string{"0000-archive-pre-2026-10-23.md": archiveText})
	path := root + "/BUILD-STATE.md"
	if code, out, errs := run("", "--check", root); code != ExitOK || out != "gen-build-state: "+path+" is fresh\n" || errs != "" {
		t.Errorf("extra trailing newline: got (%d, %q, %q), want fresh", code, out, errs)
	}
	stale := strings.Replace(fresh, "late line", "edited line", 1) + "\n"
	if err := os.WriteFile(path, []byte(stale), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	want := "gen-build-state: " + path + " is stale vs ledger.d/; run gen-build-state.sh and commit the result\n"
	if code, out, errs := run("", "--check", root); code != ExitStale || out != "" || !strings.HasPrefix(errs, want) {
		t.Errorf("stale kit block: got (%d, %q, %q), want stale", code, out, errs)
	}
	if code, out, errs := run("", root); code != ExitOK || out != "gen-build-state: wrote "+path+"\n" || errs != "" {
		t.Errorf("read-only write: got (%d, %q, %q)", code, out, errs)
	}
	st, err := os.Stat(path)
	if b, _ := os.ReadFile(path); err != nil || string(b) != fresh+"\n" || st.Mode().Perm() != 0o444 {
		t.Errorf("read-only write: target %q, want the fresh content (trailing newline kept) with mode 0444", b)
	}
}
