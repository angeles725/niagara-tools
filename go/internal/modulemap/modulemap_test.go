package modulemap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	mmrule "github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
)

// baseMap is a strict four-row map (control, frontend, station-test, plumbing) over a logic-shaped module with rt
// and ux parts (the fixture layout: srcTest/test/<pkg>), followed by prose after the table.
const baseMap = "# MODULE-MAP — M\n\nmodularity: strict\n\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n" +
	"| control | rt | M-rt/src/com/v/M/model/MLogic.java; M-rt/src/com/v/M/BM.java | M-rt/srcTest/test/com/v/M/model/MLogicTest.java | a control decision |\n" +
	"| frontend | ux | M-ux/src/rc/js/** | M-ux/srcTest/rc/spec/MSpec.js | the page |\n" +
	"| station-test | rt | — | M-rt/srcTest/test/com/v/M/BMLinkTest.java | a link-in slot |\n" +
	"| plumbing | rt | M-rt/module-include.xml | — | build only |\n\nTrailing prose.\n"

// module writes a module root M (map, rt and ux parts) under a temp dir and returns its path.
func module(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "M")
	for rel, body := range map[string]string{
		lint.MapFile:                                      baseMap,
		"M-rt/src/com/v/M/BM.java":                        "package com.v.M;\nimport javax.baja.sys.BComponent;\npublic final class BM extends BComponent {}\n",
		"M-rt/src/com/v/M/model/MLogic.java":              "package com.v.M.model;\npublic final class MLogic {}\n",
		"M-rt/srcTest/test/com/v/M/model/MLogicTest.java": "package com.v.M.model;\npublic class MLogicTest {}\n",
		"M-rt/srcTest/test/com/v/M/BMLinkTest.java":       "package com.v.M;\nimport javax.baja.test.BTestNg;\npublic class BMLinkTest {}\n",
		"M-rt/module-include.xml":                         "<x/>\n",
		"M-ux/src/rc/js/main.js":                          "x\n",
		"M-ux/srcTest/rc/spec/MSpec.js":                   "x\n",
	} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func run(env map[string]string, args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Main(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
	return code, o.String(), e.String()
}

// TestAddConcern: a new concern gets its folder under the part's role folder, a pure class, its test and a map
// row inserted after the table's last row; the module stays clean under lint-module-map.
func TestAddConcern(t *testing.T) {
	root := module(t)
	code, out, errs := run(nil, "add-concern", "Battery", "--root", root)
	if code != 0 {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
	for _, rel := range []string{"M-rt/src/com/v/M/model/battery/BatteryLogic.java", "M-rt/srcTest/test/com/v/M/model/battery/BatteryLogicTest.java"} {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s not written: %v", rel, err)
		}
		if !strings.HasPrefix(string(b), "package com.v.M.model.battery;\n") || strings.Contains(string(b), "javax.baja") {
			t.Errorf("%s: want package com.v.M.model.battery and no javax.baja import, got\n%s", rel, b)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "M-rt/src/com/v/M/adapter")); err == nil {
		t.Error("an adapter was written without --adapter")
	}
	m, err := mmrule.ReadMap(root)
	if err != nil {
		t.Fatal(err)
	}
	last := m.Rows[len(m.Rows)-1]
	if last.Concern != "battery" || last.Part != "rt" || last.Line != 11 ||
		strings.Join(last.Files, ";") != "M-rt/src/com/v/M/model/battery/BatteryLogic.java" ||
		strings.Join(last.Tests, ";") != "M-rt/srcTest/test/com/v/M/model/battery/BatteryLogicTest.java" {
		t.Errorf("last row %+v", last)
	}
	if !strings.Contains(strings.Join(m.Lines, "\n"), "\n\nTrailing prose.\n") {
		t.Error("the prose after the table was not kept")
	}
	var lo, le bytes.Buffer
	if c := lint.Command(mmrule.Spec())([]string{mmrule.ID, root}, cli.Stdio{Out: &lo, Err: &le}); c != 0 || lo.Len() != 0 {
		t.Errorf("lint-module-map after add-concern: exit %d\n%s%s", c, lo.String(), le.String())
	}
}

// TestAddConcernAdapter: --adapter adds an rt adapter class under adapter/<concern>/ to the same row.
func TestAddConcernAdapter(t *testing.T) {
	root := module(t)
	if code, out, errs := run(nil, "add-concern", "Cooling", "--root", root, "--part", "rt", "--adapter"); code != 0 {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
	b, err := os.ReadFile(filepath.Join(root, "M-rt/src/com/v/M/adapter/cooling/CoolingAdapter.java"))
	if err != nil || !strings.Contains(string(b), "import javax.baja.sys.BComponent;") || !strings.Contains(string(b), "import com.v.M.model.cooling.CoolingLogic;") {
		t.Fatalf("adapter: %v\n%s", err, b)
	}
	m, _ := mmrule.ReadMap(root)
	if got := strings.Join(m.Rows[len(m.Rows)-1].Files, ";"); got != "M-rt/src/com/v/M/model/cooling/CoolingLogic.java;M-rt/src/com/v/M/adapter/cooling/CoolingAdapter.java" {
		t.Errorf("files %q", got)
	}
}

// TestAddConcernRefuses: an existing concern row, an existing folder, a bad name, a bad part and --adapter outside
// rt are refused with nothing written.
func TestAddConcernRefuses(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		pre  string // a directory created first, relative to root
		code int
	}{
		{"existing row", []string{"Control"}, "", 1},
		{"existing folder", []string{"Battery"}, "M-rt/src/com/v/M/model/battery", 1},
		{"lowercase name", []string{"battery"}, "", 2},
		{"unknown part", []string{"Battery", "--part", "xx"}, "", 2},
		{"absent part", []string{"Battery", "--part", "wb"}, "", 3},
		{"adapter outside rt", []string{"Battery", "--part", "ux", "--adapter"}, "", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := module(t)
			if tc.pre != "" {
				if err := os.MkdirAll(filepath.Join(root, tc.pre), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(filepath.Join(root, lint.MapFile))
			code, _, errs := run(nil, append([]string{"add-concern"}, append(tc.args, "--root", root)...)...)
			after, _ := os.ReadFile(filepath.Join(root, lint.MapFile))
			if code != tc.code || errs == "" || !bytes.Equal(before, after) {
				t.Errorf("exit %d (want %d) err %q, map changed %v", code, tc.code, errs, !bytes.Equal(before, after))
			}
			if _, err := os.Stat(filepath.Join(root, "M-rt/src/com/v/M/model/battery/BatteryLogic.java")); err == nil {
				t.Error("a class was written")
			}
		})
	}
}

// TestWhere: by concern name, by an exact file, by a file a glob lists; an unknown target exits 1.
func TestWhere(t *testing.T) {
	root := module(t)
	want := "concern  control\npart     rt\nfiles    M-rt/src/com/v/M/model/MLogic.java\nfiles    M-rt/src/com/v/M/BM.java\n" +
		"tests    M-rt/srcTest/test/com/v/M/model/MLogicTest.java\nchange   a control decision\n"
	for _, arg := range []string{"control", "M-rt/src/com/v/M/BM.java", filepath.Join(root, "M-rt/src/com/v/M/BM.java")} {
		if code, out, errs := run(nil, "where", arg, "--root", root); code != 0 || out != want {
			t.Errorf("where %s: exit %d err %q\n%s", arg, code, errs, out)
		}
	}
	if code, out, _ := run(nil, "where", "M-ux/src/rc/js/main.js", "--root", root); code != 0 || !strings.HasPrefix(out, "concern  frontend\n") {
		t.Errorf("glob-listed file: exit %d\n%s", code, out)
	}
	if code, _, errs := run(nil, "where", "nothing", "--root", root); code != 1 || errs == "" {
		t.Errorf("unknown: exit %d err %q", code, errs)
	}
}

// TestTests: the concern's pure JUnit tests run through run-pure-test.sh <part-dir> <fqcn>; a Baja-importing
// station test and a JS spec are skipped with a reason; the runner's exit code is relayed.
func TestTests(t *testing.T) {
	root := module(t)
	tb := t.TempDir()
	log := filepath.Join(tb, "argv")
	stub := "#!/bin/sh\nprintf '%s|%s\\n' \"$1\" \"$2\" >> " + log + "\nexit ${STUB_EXIT:-0}\n"
	if err := os.WriteFile(filepath.Join(tb, "run-pure-test.sh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"MODULE_MAP_TOOLBELT": tb}
	code, out, errs := run(env, "tests", "control", "--root", root)
	if code != 0 {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
	got, _ := os.ReadFile(log)
	if string(got) != filepath.Join(root, "M-rt")+"|com.v.M.model.MLogicTest\n" {
		t.Errorf("runner argv %q", got)
	}
	if code, out, _ := run(env, "tests", "station-test", "--root", root); code != 0 ||
		!strings.Contains(out, "skip  M-rt/srcTest/test/com/v/M/BMLinkTest.java  imports javax.baja") {
		t.Errorf("station-test: exit %d, no skip row:\n%s", code, out)
	}
	if code, out, _ := run(env, "tests", "frontend", "--root", root); code != 0 || !strings.Contains(out, "skip  M-ux/srcTest/rc/spec/MSpec.js  Jasmine browser spec") {
		t.Errorf("frontend: exit %d\n%s", code, out)
	}
	// The runner is a child process: it reads the PROCESS environment, so the stub's exit is set there.
	t.Setenv("STUB_EXIT", "1")
	if code, _, _ := run(env, "tests", "control", "--root", root); code != 1 {
		t.Errorf("a failing test: exit %d, want 1", code)
	}
	// Mutation: MMT-env -- reading every runner failure as a failed test hides a runner that could not run
	t.Setenv("STUB_EXIT", "3")
	if code, _, errs := run(env, "tests", "control", "--root", root); code != 3 || !strings.Contains(errs, "could not run") {
		t.Errorf("a runner env error: exit %d err %q, want 3", code, errs)
	}
	if err := os.Chmod(filepath.Join(tb, "run-pure-test.sh"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := run(env, "tests", "control", "--root", root); code != 3 || !strings.Contains(errs, "could not run") {
		t.Errorf("a runner that cannot start: exit %d err %q, want 3", code, errs)
	}
	if code, _, _ := run(env, "tests", "nothing", "--root", root); code != 1 {
		t.Errorf("unknown concern: exit %d, want 1", code)
	}
}

// TestUsage: no subcommand, an unknown one and a missing --root are usage errors (exit 2); a root with no map is 3.
func TestUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"bogus"}, {"where", "x"}, {"add-concern"}} {
		if code, _, errs := run(nil, args...); code != 2 || errs == "" {
			t.Errorf("%v: exit %d err %q", args, code, errs)
		}
	}
	if code, _, _ := run(nil, "where", "x", "--root", t.TempDir()); code != 3 {
		t.Errorf("no map: exit %d, want 3", code)
	}
}

// writeMap replaces the module's map with baseMap plus extra rows (each a full table line).
func writeMap(t *testing.T, root string, extra ...string) {
	t.Helper()
	body := strings.Replace(baseMap, "\n\nTrailing prose.\n", "\n"+strings.Join(extra, "\n")+"\n\nTrailing prose.\n", 1)
	if err := os.WriteFile(filepath.Join(root, lint.MapFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTestsStationImportLine: a station test is one with an `import javax.baja…` line; a pure test that only
// mentions javax.baja in a comment or string still runs (review R3-002 of e9aeb6e8).
func TestTestsStationImportLine(t *testing.T) {
	root := module(t)
	pure := "package com.v.M.model;\n// not a station test: javax.baja is only named here\npublic class MNoteTest { String s = \"javax.baja\"; }\n"
	if err := os.WriteFile(filepath.Join(root, "M-rt/srcTest/test/com/v/M/model/MNoteTest.java"), []byte(pure), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMap(t, root, "| notes | rt | — | M-rt/srcTest/test/com/v/M/model/MNoteTest.java | a note |")
	tb := t.TempDir()
	if err := os.WriteFile(filepath.Join(tb, "run-pure-test.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Mutation: MMT-import -- a substring match skips a pure test that merely names javax.baja
	code, out, errs := run(map[string]string{"MODULE_MAP_TOOLBELT": tb}, "tests", "notes", "--root", root)
	if code != 0 || !strings.Contains(out, "run   M-rt/srcTest/test/com/v/M/model/MNoteTest.java  com.v.M.model.MNoteTest") {
		t.Errorf("exit %d err %q\n%s", code, errs, out)
	}
}

// TestTestsCellContainment: a Tests entry that leaves --root (absolute, or .. after cleaning) is refused before
// any runner starts, as lint-module-map MM3 refuses it (review R1 of e9aeb6e8).
func TestTestsCellContainment(t *testing.T) {
	for _, entry := range []string{"../outside/XTest.java", "M-rt/../../outside/XTest.java", "/etc/XTest.java", "../**/*Test.java"} {
		root := module(t)
		writeMap(t, root, "| escape | rt | — | "+entry+" | never |")
		tb := t.TempDir()
		log := filepath.Join(tb, "ran")
		if err := os.WriteFile(filepath.Join(tb, "run-pure-test.sh"), []byte("#!/bin/sh\ntouch "+log+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		// Mutation: MMT-contain -- expanding an entry outside the root runs (or lists) files the map does not own
		code, _, errs := run(map[string]string{"MODULE_MAP_TOOLBELT": tb}, "tests", "escape", "--root", root)
		if code != 3 || !strings.Contains(errs, "is not a relative path inside the module root") {
			t.Errorf("%s: exit %d err %q, want 3", entry, code, errs)
		}
		if _, err := os.Stat(log); err == nil {
			t.Errorf("%s: the runner started", entry)
		}
	}
}

// TestAddConcernRollback: a failed run removes what it wrote — files and the concern directories it created,
// deepest first, never a directory that was there before — so the same command can be retried (review R4/R3-001).
func TestAddConcernRollback(t *testing.T) {
	root := module(t)
	before, _ := os.ReadFile(filepath.Join(root, lint.MapFile))
	tmp := filepath.Join(root, lint.MapFile+".tmp")
	if err := os.Mkdir(tmp, 0o755); err != nil { // the map write cannot open its temp file
		t.Fatal(err)
	}
	// Mutation: MMA-undo-dirs -- removing only the files leaves the concern folders, and the retry is refused
	if code, _, errs := run(nil, "add-concern", "Battery", "--root", root, "--adapter"); code != 3 || errs == "" {
		t.Fatalf("exit %d err %q, want 3", code, errs)
	}
	for _, d := range []string{"M-rt/src/com/v/M/model/battery", "M-rt/src/com/v/M/adapter", "M-rt/srcTest/test/com/v/M/model/battery"} {
		if _, err := os.Lstat(filepath.Join(root, d)); err == nil {
			t.Errorf("%s left behind", d)
		}
	}
	for _, d := range []string{"M-rt/src/com/v/M/model", "M-rt/srcTest/test/com/v/M/model"} {
		if !lint.IsDir(filepath.Join(root, d)) {
			t.Errorf("pre-existing %s removed", d)
		}
	}
	if after, _ := os.ReadFile(filepath.Join(root, lint.MapFile)); !bytes.Equal(before, after) {
		t.Error("the map changed")
	}
	if err := os.Remove(tmp); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run(nil, "add-concern", "Battery", "--root", root, "--adapter"); code != 0 {
		t.Errorf("retry: exit %d out %q err %q", code, out, errs)
	}
}

// TestAddConcernTmpLeftover: a map write that fails after its temp file was opened removes MODULE-MAP.md.tmp
// (review R4-tmp-leftover). /dev/full opens and then fails every write with ENOSPC.
func TestAddConcernTmpLeftover(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("no /dev/full on this system")
	}
	root := module(t)
	tmp := filepath.Join(root, lint.MapFile+".tmp")
	if err := os.Symlink("/dev/full", tmp); err != nil {
		t.Fatal(err)
	}
	// Mutation: MMA-tmp -- a failed write leaves MODULE-MAP.md.tmp behind
	if code, _, errs := run(nil, "add-concern", "Battery", "--root", root); code != 3 || errs == "" {
		t.Fatalf("exit %d err %q, want 3", code, errs)
	}
	if _, err := os.Lstat(tmp); err == nil {
		t.Error("MODULE-MAP.md.tmp left behind")
	}
}

// TestAddConcernDefaultPackage: when the shallowest Java source sits in the default package (directly in src/),
// there is no package to place the concern under: refused, nothing written (review R3-004).
func TestAddConcernDefaultPackage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "M")
	for rel, body := range map[string]string{
		lint.MapFile:              "modularity: strict\n\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n| control | rt | M-rt/src/A.java | M-rt/srcTest/ATest.java | x |\n",
		"M-rt/src/A.java":         "public class A {}\n",
		"M-rt/srcTest/ATest.java": "public class ATest {}\n",
	} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, _, errs := run(nil, "add-concern", "Battery", "--root", root)
	if code != 3 || !strings.Contains(errs, "default package") {
		t.Errorf("exit %d err %q, want 3 naming the default package", code, errs)
	}
	if _, err := os.Lstat(filepath.Join(root, "M-rt/src/model")); err == nil {
		t.Error("a role folder was written")
	}
}
