package schemarisk

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fixtures is tests/fixtures/schema-risk, shared with tests/schema-risk.bats (SR1-SR10).
const fixtures = "../../../tests/fixtures/schema-risk"

func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

// snapshot writes a module-include.xml and one Java file per text into a new dir.
func snapshot(t *testing.T, xml string, java ...string) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "module-include.xml"), xml)
	for i, src := range java {
		write(t, filepath.Join(dir, "p", string(rune('A'+i))+".java"), src)
	}
	return dir
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestGolden replays every fixture pair against the stdout schema-risk.sh printed for it (testdata/<pair>.golden);
// the exit code is the golden's verdict line.
func TestGolden(t *testing.T) {
	t.Parallel()
	goldens, _ := filepath.Glob("testdata/*.golden")
	if len(goldens) < 8 {
		t.Fatalf("want the 8 recorded fixture goldens, found %d", len(goldens))
	}
	exits := map[string]int{"verdict=SAFE\n": 0, "verdict=LOSSY\n": 1, "verdict=OUTAGE\n": 2}
	for _, g := range goldens {
		want, err := os.ReadFile(g)
		if err != nil {
			t.Fatal(err)
		}
		pair := filepath.Join(fixtures, strings.TrimSuffix(filepath.Base(g), ".golden"))
		code, out, errs := run(pair+"/before", pair+"/after")
		last := string(want[bytes.LastIndexByte(want[:len(want)-1], '\n')+1:])
		if out != string(want) || code != exits[last] || errs != "" {
			t.Errorf("%s: exit %d out %q err %q\nwant %d %q", g, code, out, errs, exits[last], want)
		}
	}
}

func TestEmbeddedTableMatchesB795(t *testing.T) {
	t.Parallel()
	oracle, err := os.ReadFile(filepath.Join(fixtures, "b795-795.4.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(table, "\n") != strings.TrimRight(string(oracle), "\n") {
		t.Errorf("embedded b795-795.4.csv differs from %s/b795-795.4.csv", fixtures)
	}
}

func TestMainEdges(t *testing.T) {
	t.Parallel()
	const xml = "<types>\n  <dependency name=\"\"/>\n  <type class=\"p.BPan\" name=\"Pan\"/>\n</types>\n"
	prop := "@NiagaraProperty(name=\"sp\", type=\"double\")\n"
	act := "  @NiagaraAction(\n    name = \"go\"\n  )\n"
	none := snapshot(t, xml, "class A {}\n")
	cases := []struct {
		name          string
		before, after string
		code          int
		out, errs     string
	}{
		{"same slots, same order: no row, SAFE", snapshot(t, xml, prop+act), snapshot(t, xml, prop+act), 0, "verdict=SAFE\n", ""},
		// An empty <type name=""> line is skipped; the first non-empty name="..." names the type.
		{"multi-line action removed", snapshot(t, xml, prop+act), snapshot(t, xml, prop), 1,
			"LOSSY  remove_action_or_topic  Pan.go: B754-754.6-r8 (warn-skip)\nverdict=LOSSY\n", ""},
		{"no name attribute: Module", snapshot(t, "<types/>\n", prop), snapshot(t, "<types/>\n", ""), 1,
			"LOSSY  remove_slot_simple  Module.sp: B754-754.6-r7 (value-dropped)\nverdict=LOSSY\n", ""},
		{"complex removed: unknown subtype", snapshot(t, xml, "@NiagaraProperty(name=\"t\", type=\"baja:RelTime\")\n"), none, 1,
			"LOSSY  remove_slot_unknown  Pan.t: B754-754.6-r6r7-failsafe (use-when-subtype-unknown)\nverdict=LOSSY\n", ""},
		{"complex retype: retype_unknown", snapshot(t, xml, "@NiagaraProperty(name=\"t\", type=\"baja:RelTime\")\n"),
			snapshot(t, xml, "@NiagaraProperty(name=\"t\", type=\"baja:AbsTime\")\n"), 2,
			"OUTAGE  retype_unknown  Pan.t: B754-754.6-r10r11-failsafe (use-when-subtype-unknown)\nverdict=OUTAGE\n", ""},
		// bash quirk kept (awk NR==FNR on an empty first file): a before with no slot reads the after slots as removed.
		{"empty before reads after as removed", none, snapshot(t, xml, prop), 1,
			"LOSSY  remove_slot_simple  Pan.sp: B754-754.6-r7 (value-dropped)\nverdict=LOSSY\n", ""},
		{"no slot either side: SAFE", none, none, 0, "verdict=SAFE\n", ""},
		{"missing module-include.xml: env", t.TempDir(), none, 4, "", "error: cannot read %B/module-include.xml\n"},
	}
	for _, c := range cases {
		code, out, errs := run(c.before, c.after)
		if want := strings.ReplaceAll(c.errs, "%B", c.before); code != c.code || out != c.out || errs != want {
			t.Errorf("%s: exit %d out %q err %q\nwant %d %q %q", c.name, code, out, errs, c.code, c.out, want)
		}
	}
	if code, out, errs := run(none); code != 3 || out != "" || errs != "usage: schema-risk.sh <before-dir> <after-dir>\n" {
		t.Errorf("one argument: exit %d out %q err %q", code, out, errs)
	}
}

// TestUnreadableJava pins the bash fail-open kept by the port: awk stopped at an unreadable Java file (fatal on
// stderr) and the verdict was computed from the slots read before it.
func TestUnreadableJava(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root")
	}
	before := snapshot(t, "<type name=\"Pan\"/>\n", "@NiagaraProperty(name=\"sp\", type=\"double\")\n")
	after := snapshot(t, "<type name=\"Pan\"/>\n", "class A {}\n")
	locked := filepath.Join(after, "p", "A.java")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(before, after)
	want := "awk: fatal: cannot open file `" + locked + "' for reading: Permission denied\n"
	if code != 1 || out != "LOSSY  remove_slot_simple  Pan.sp: B754-754.6-r7 (value-dropped)\nverdict=LOSSY\n" || errs != want {
		t.Errorf("exit %d out %q err %q\nwant err %q", code, out, errs, want)
	}
}

// TestLockedDir: a sub-directory the walk cannot enter is env (4), never fewer slots (SR-locked).
func TestLockedDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root")
	}
	before := snapshot(t, "<type name=\"Pan\"/>\n", "class A {}\n")
	locked := filepath.Join(before, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	code, out, errs := run(before, before)
	want := "error: cannot list every file under " + before + ": find: '" + locked + "': Permission denied\n"
	if code != 4 || out != "" || errs != want {
		t.Errorf("exit %d out %q err %q\nwant %q", code, out, errs, want)
	}
}
