package silentprotection

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-silent-protection.bats run on a tree and every edge tree, recorded from the
// bash lint (the empty-root, unreadable and literal cases are table tests below).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 39) }

// run runs the rule alone, as its wrapper does, and returns the exit code, stdout and stderr with dir
// spelled @DIR@.
func run(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{ID}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, strings.ReplaceAll(o.String(), dir, "@DIR@"), strings.ReplaceAll(e.String(), dir, "@DIR@")
}

// write creates the files of a tree under a new temporary directory.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, text := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// shed is a one-method class whose line 4 is an unsurfaced Pattern 1 trip.
const shed = "package com.x;\npublic class S {\n  int a(int target, int onCount, boolean c) {\n" +
	"    if (c) target = Math.min(target, onCount - 1);\n    return target;\n  }\n}\n"

func warnRow(path string, line int) string {
	return "WARN  " + ID + "  " + path + ":" + strconv.Itoa(line) + "  a forces stage/sheds stage on condition" + warnMsg + "\n"
}

// TestEmptyRoot: SP9's empty directory (a golden case cannot hold one) is the K20 ERROR row, exit 3.
func TestEmptyRoot(t *testing.T) {
	dir := t.TempDir()
	code, out, stderr := run(t, dir, dir)
	want := "ERROR  lint-silent-protection  @DIR@  no Java sources found (K20: wrong path or empty scaffold)\n"
	if code != 3 || out != want || stderr != "" {
		t.Errorf("exit %d out %q err %q, want 3 and %q", code, out, stderr, want)
	}
}

// TestWalkError: SP-finderr. An unreadable sub-directory stops the rule as scan_files_or_exit did: the
// walk line on stderr, no row (the readable trip is not reported) and exit 3.
func TestWalkError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := write(t, map[string]string{"a/S.java": shed})
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	code, out, stderr := run(t, dir, dir)
	want := ID + ": cannot list every file under @DIR@: find: '@DIR@/locked': Permission denied\n"
	if code != 3 || out != "" || stderr != want {
		t.Errorf("exit %d out %q err %q, want 3, no row and %q", code, out, stderr, want)
	}
}

// TestUnreadableFile pins a declared difference: the bash lint ran awk with 2>/dev/null and skipped an
// unreadable .java file (exit 0, as if it were clean); the port names it on stderr and exits 3, and
// still reports the readable files.
func TestUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := write(t, map[string]string{"a/S.java": shed})
	u := filepath.Join(dir, "a", "U.java")
	if err := os.WriteFile(u, []byte(shed), 0); err != nil {
		t.Fatal(err)
	}
	code, out, stderr := run(t, dir, dir)
	wantErr := ID + ": cannot scan @DIR@/a/U.java: awk: fatal: cannot open file `@DIR@/a/U.java' for reading: Permission denied\n"
	if code != 3 || out != warnRow("@DIR@/a/S.java", 4) || stderr != wantErr {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, out, stderr, warnRow("@DIR@/a/S.java", 4), wantErr)
	}
}

// TestLiteralComments pins a declared difference [ev: retro mb-strip-string-literals Δ1]: comments are
// stripped by the shared javasrc.Strip, which keeps "..." literals. The bash private stripper read the
// "/*" string on line 8 as a block comment and hid the trip on line 10 until the "*/" string on line 13.
func TestLiteralComments(t *testing.T) {
	src := "package com.x;\npublic class L {\n  String u = \"tcp://host\";\n" +
		"  int a(int target, int onCount, boolean c) {\n    if (c) target = Math.min(target, onCount - 1);\n    return target;\n  }\n" +
		"  String g = \"/*\";\n" +
		"  int a(int target, int onCount, boolean c) {\n    if (c) target = Math.min(target, onCount - 1);\n    return target;\n  }\n" +
		"  String h = \"*/\";\n" +
		"  int a(int target, int onCount, boolean c) {\n    if (c) target = Math.min(target, onCount - 1);\n    return target;\n  }\n}\n"
	dir := write(t, map[string]string{"L.java": src})
	code, out, stderr := run(t, dir, dir)
	want := warnRow("@DIR@/L.java", 5) + warnRow("@DIR@/L.java", 10) + warnRow("@DIR@/L.java", 15)
	if code != 0 || out != want || stderr != "" {
		t.Errorf("exit %d err %q out\n%s\nwant\n%s", code, stderr, out, want)
	}
}

// TestLiteralAlarmClass: the Pattern-B pre-pass reads the same stripped lines, so a "//" inside a string
// no longer hides the newOffnormalAlarm call after it on the line.
func TestLiteralAlarmClass(t *testing.T) {
	dir := write(t, map[string]string{
		"S.java":  shed,
		"BS.java": "public class BS extends BComponent implements BIAlarmSource {\n  void r() { log(\"http://x\"); alarmSupport.newOffnormalAlarm(d); }\n}\n",
	})
	code, out, _ := run(t, dir, dir)
	want := "ADVISORY  " + ID + "  @DIR@/S.java:4  console-only: a forces stage/sheds stage" + consoleMsg + "\n"
	if code != 0 || out != want {
		t.Errorf("exit %d out\n%s\nwant\n%s", code, out, want)
	}
}

// TestSpacedPathIndexed: a surface slot declared in a file whose path holds a blank still counts. The
// bash lint fed passes 0 and 1 through xargs, which split such a path, so awk never read the slot and
// the surfaced trip was a false WARN.
func TestSpacedPathIndexed(t *testing.T) {
	dir := write(t, map[string]string{
		"a b/BS.java": "public class BS extends BComponent {\n" +
			"  @NiagaraProperty(name=\"tripReason\", type=\"BString\", flags=Flags.SUMMARY)\n" +
			"  void a(boolean c) {\n    if (c) this.tripped = true;\n    setTripReason(\"x\");\n  }\n" +
			"  private boolean tripped;\n}\n",
		"it's/Core.java": shed,
		"it's/BCore.java": "public class BCore extends BComponent {\n" +
			"  @NiagaraProperty(name=\"lowAlarm\", type=\"BStatusBoolean\", flags=Flags.SUMMARY)\n" +
			"  void s(boolean c) { getLowAlarm().setValue(core.c); }\n}\n",
	})
	code, out, stderr := run(t, dir, dir)
	if code != 0 || out != "" || stderr != "" {
		t.Errorf("exit %d err %q out\n%s\nwant no row", code, stderr, out)
	}
}

// TestDedupeOnPathLine: the header promises one row per trip site, deduped on <file>:<line>. The bash
// dedupe keyed on the third field of the row split on runs of two or more blanks, so under a path with
// such a run every row of every file shared one key and only the first was printed.
func TestDedupeOnPathLine(t *testing.T) {
	two := "package com.x;\npublic class P {\n  int a(int target, int onCount, boolean c) {\n" +
		"    if (c) target = Math.min(target, onCount - 1);\n    if (c) target = Math.min(target, onCount - 2);\n" +
		"    return target;\n  }\n}\n"
	dir := write(t, map[string]string{"c  d/P.java": two, "c  d/Q.java": two})
	code, out, _ := run(t, dir, dir)
	want := warnRow("@DIR@/c  d/P.java", 4) + warnRow("@DIR@/c  d/P.java", 5) +
		warnRow("@DIR@/c  d/Q.java", 4) + warnRow("@DIR@/c  d/Q.java", 5)
	if code != 0 || out != want {
		t.Errorf("exit %d out\n%s\nwant\n%s", code, out, want)
	}
}
