package buildstate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

const usageLine = "usage: gen-build-state.sh [--check] [<kit-root>]\n"

const archiveText = "<!-- ledger: kit -->\n<!-- date: 2026-10-22 -->\n<!-- last-change-note: old note -->\n" +
	"log: 2026-10-23 · late line\nlog: 2026-09-01 · early line\nopen_issue: OLD A\nopen_issue: OLD B\n"

// kit makes a kit root with BUILD-STATE.src.md (source) and ledger.d/ holding the given fragments.
func kit(t *testing.T, source string, frags map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "ledger.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, SourceName), []byte(source), 0o644); err != nil {
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

// source is the committed form (the kit envelope's generated parts empty); fresh is what the archive
// fragment generates from it, as the kit's BUILD-STATE.md, after the GENERATED line.
var (
	source = state(sourceLastChange, "", "")
	fresh  = state("last_change: 2026-10-22              # old note", "  - OLD A\n  - OLD B\n",
		"2026-10-23 · late line\n2026-09-01 · early line\n")
	archiveOnly = map[string]string{"0000-archive-pre-2026-10-23.md": archiveText}
)

func TestMainUsage(t *testing.T) {
	t.Parallel()
	root := kit(t, source, nil)
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
	// a generated BUILD-STATE.md alone is not a source
	if err := os.WriteFile(filepath.Join(root, OutputName), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := run("", root); code != ExitEnv || errs != "gen-build-state: no BUILD-STATE.src.md under "+root+"\n" {
		t.Errorf("no BUILD-STATE.src.md: got (%d, %q)", code, errs)
	}
}

func TestMainCheckValidatesAndWritesNothing(t *testing.T) {
	t.Parallel()
	root := kit(t, source, archiveOnly)
	code, out, errs := run("", "--check", root)
	if code != ExitOK || out != "gen-build-state: "+root+"/BUILD-STATE.src.md and ledger.d/ are valid\n" || errs != "" {
		t.Errorf("--check: got (%d, %q, %q)", code, out, errs)
	}
	if _, err := os.Stat(root + "/" + OutputName); err == nil {
		t.Error("--check wrote BUILD-STATE.md")
	}
}

func TestMainWrite(t *testing.T) {
	t.Parallel()
	root := kit(t, source, archiveOnly)
	path := root + "/" + OutputName
	code, out, errs := run(root)
	if code != ExitOK || out != "gen-build-state: wrote "+path+"\n" || errs != "" {
		t.Errorf("write: got (%d, %q, %q)", code, out, errs)
	}
	if got, _ := os.ReadFile(path); string(got) != gen.GeneratedLine+"\n"+fresh {
		t.Errorf("written file:\n%s", got)
	}
	frag := "<!-- ledger: kit -->\n<!-- date: 2026-10-24 -->\n<!-- last-change-note: new -->\nlog: 2026-10-24 · new line\nopen_issue: NEW\n"
	if err := os.WriteFile(filepath.Join(root, "ledger.d", "2026-10-24-new.md"), []byte(frag), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, _ = run("", root); code != ExitOK {
		t.Fatalf("rewrite: exit %d", code)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "last_change: 2026-10-24              # new\nopen_issues:\n  - NEW\n  - OLD A\n") ||
		!strings.HasSuffix(string(got), "2026-09-01 · early line\n2026-10-24 · new line\n<!-- /build-state.v1 -->\n") {
		t.Errorf("rewritten file:\n%s", got)
	}
	if src, _ := os.ReadFile(root + "/" + SourceName); string(src) != source {
		t.Error("writing BUILD-STATE.md changed BUILD-STATE.src.md")
	}
}

func TestMainViolationWritesNothing(t *testing.T) {
	t.Parallel()
	bad := "<!-- ledger: kit -->\n<!-- date: 2026-10-24 -->\nlog: 2026-10-24 · x\n  - stray\n"
	root := kit(t, source, map[string]string{"0000-archive-pre-2026-10-23.md": archiveText, "2026-10-24-bad.md": bad})
	for _, args := range [][]string{{root}, {"--check", root}} {
		code, out, errs := run("", args...)
		want := "gen-build-state: 2026-10-24-bad.md: line 4: " + errLine + "\n" +
			"gen-build-state: ledger fragment violation; nothing generated\n"
		if code != ExitStale || out != "" || errs != want {
			t.Errorf("%q: got (%d, %q, %q)", args, code, out, errs)
		}
	}
	if _, err := os.Stat(root + "/" + OutputName); err == nil {
		t.Error("BUILD-STATE.md was written despite a violation")
	}
}

func TestMainEmptyLedgerAndStructureViolation(t *testing.T) {
	t.Parallel()
	root := kit(t, source, nil)
	if code, _, errs := run("", root); code != ExitStale || errs != "gen-build-state: ledger.d/ holds no fragment; nothing generated\n" {
		t.Errorf("empty ledger: got (%d, %q)", code, errs)
	}
	root = kit(t, strings.Replace(source, "module: kit ", "module: kat ", 1), archiveOnly)
	if code, _, errs := run("", "--check", root); code != ExitStale || errs != "gen-build-state: BUILD-STATE.src.md: "+errNoKit+"; nothing generated\n" {
		t.Errorf("no kit envelope: got (%d, %q)", code, errs)
	}
}

// TestGenerateRefusesGeneratedContentInTheSource: a generated part pasted into BUILD-STATE.src.md (here a
// whole generated kit block) would be replaced on every generation and so lost silently.
func TestGenerateRefusesGeneratedContentInTheSource(t *testing.T) {
	t.Parallel()
	// Mutation: GBS-source -- dropping the source-form check generates a file and loses the pasted lines
	for name, src := range map[string]string{
		"generated kit block":  fresh,
		"one log line":         state(sourceLastChange, "", "2026-10-23 · pasted\n"),
		"one open_issues item": state(sourceLastChange, "  - pasted\n", ""),
	} {
		root := kit(t, src, archiveOnly)
		var diag bytes.Buffer
		got, code := Generate(root, &diag)
		if code != ExitStale || got != "" ||
			!strings.HasPrefix(diag.String(), "gen-build-state: BUILD-STATE.src.md holds generated content") {
			t.Errorf("%s: got (%q, %d, %q)", name, got, code, diag.String())
		}
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	root := kit(t, source, archiveOnly)
	var diag bytes.Buffer
	got, code := Generate(root, &diag)
	if code != ExitOK || got != gen.GeneratedLine+"\n"+fresh || diag.String() != "" {
		t.Errorf("Generate = (%q, %d, %q)", got, code, diag.String())
	}
	if _, err := os.Stat(root + "/" + OutputName); err == nil {
		t.Error("Generate wrote a file")
	}
}

// TestMainWriteReadOnlyTarget: a read-only BUILD-STATE.md left by an earlier run is replaced atomically
// (its directory is writable) and keeps its mode.
func TestMainWriteReadOnlyTarget(t *testing.T) {
	t.Parallel()
	root := kit(t, source, archiveOnly)
	path := root + "/" + OutputName
	if err := os.WriteFile(path, []byte("stale\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run("", root); code != ExitOK || out != "gen-build-state: wrote "+path+"\n" || errs != "" {
		t.Errorf("read-only write: got (%d, %q, %q)", code, out, errs)
	}
	st, err := os.Stat(path)
	if b, _ := os.ReadFile(path); err != nil || string(b) != gen.GeneratedLine+"\n"+fresh || st.Mode().Perm() != 0o444 {
		t.Errorf("read-only write: target %q, want the generated content with mode 0444", b)
	}
}
