package duplicatehelper

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

// TestGolden: the wrapper contract, written by hand from the deferred G41 spec (no bash lint to record).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 4) }

// rows runs the rule on a module root holding files (path -> source) and returns its stdout, root cut.
func rows(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, src := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var o, e bytes.Buffer
	if code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e}); code != 0 || e.Len() != 0 {
		t.Fatalf("exit %d stderr %q out %q, want 0 and no stderr", code, e.String(), o.String())
	}
	return strings.ReplaceAll(o.String(), dir+"/", "")
}

// body is n distinct statements.
func body(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("    out[" + string(rune('a'+i)) + "] = (char) (in[" + string(rune('a'+i)) + "] & 0xff);\n")
	}
	return b.String()
}

const (
	rtA = "Mod/Mod-rt/src/com/x/BA.java"
	rtB = "Mod/Mod-rt/src/com/x/BB.java"
	uxC = "Mod/Mod-ux/src/com/x/ux/BC.java"
)

func class(name, members string) string {
	return "package com.x;\npublic class " + name + " {\n" + members + "}\n"
}

func helper(name, b string) string {
	return "  private static String " + name + "(byte[] in) {\n    char[] out = new char[in.length];\n" + b + "    return new String(out);\n  }\n"
}

func warn(path string, line int, first string, firstLine int, detail string) string {
	return "WARN  " + ID + "  " + path + ":" + strconv.Itoa(line) + "  same shape as " + first + ":" + strconv.Itoa(firstLine) + " (" + detail + ")\n"
}

// TestSpecCases: the six cases of the deferred G41 spec, plus the edges each rule needs.
func TestSpecCases(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"spec 1: two private readCp437(byte[]) with identical 10-statement bodies in two classes",
			map[string]string{rtA: class("BA", helper("readCp437", body(8))), uxC: class("BC", helper("readCp437", body(8)))},
			warn(uxC, 3, rtA, 3, "readCp437: same body, 10 statements")},
		{"spec 2: same name and signature, different bodies",
			map[string]string{rtA: class("BA", "  static int scale(int v, double f) { return (int) (v * f); }\n"),
				rtB: class("BB", "  static int scale(int x, double k) { return (int) Math.round(x * k); }\n")},
			warn(rtB, 3, rtA, 3, "scale(int,double): same name and parameter types")},
		{"spec 3: two @Override implementations of one interface method",
			map[string]string{rtA: class("BA", "  @Override\n  public void run() { a(); }\n"), rtB: class("BB", "  @Override public void run() { b(); }\n")}, ""},
		{"spec 4: matching bodies under 8 statements",
			map[string]string{rtA: class("BA", helper("one", body(5))), rtB: class("BB", helper("two", body(5)))}, ""},
		{"8 statements is the threshold",
			map[string]string{rtA: class("BA", helper("one", body(6))), rtB: class("BB", helper("two", body(6)))},
			warn(rtB, 3, rtA, 3, "two: same body, 8 statements")},
		{"spec 5: a copy inside a comment or a text block",
			map[string]string{rtA: class("BA", helper("one", body(8))),
				rtB: class("BB", "  /*\n"+helper("two", body(8))+"  */\n  // "+strings.ReplaceAll(helper("three", body(8)), "\n", " ")+"\n  static final String S = \"\"\"\n"+helper("four", body(8))+"  \"\"\";\n")}, ""},
		{"spec 6: generated slotomatic regions",
			map[string]string{rtA: class("BA", "//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/\n  public Type getType() { return TYPE; }\n  public double getSetpoint() { return getDouble(setpoint); }\n//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/\n"),
				rtB: class("BB", "//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/\n  public Type getType() { return TYPE; }\n  public double getSetpoint() { return getDouble(setpoint); }\n//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/\n")}, ""},
		{"after the generated region the rule applies again",
			map[string]string{rtA: class("BA", "//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/\n  public Type getType() { return TYPE; }\n//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/\n  static int clamp(int v) { return v; }\n"),
				rtB: class("BB", "  static int clamp(int v) { return v < 0 ? 0 : v; }\n")},
			warn(rtB, 3, rtA, 6, "clamp(int): same name and parameter types")},
		{"a region marker inside a string is code",
			map[string]string{rtA: class("BA", "  String m = \"BEGIN BAJA AUTO GENERATED CODE\";\n  static int clamp(int v) { return v; }\n"),
				rtB: class("BB", "  static int clamp(int v) { return -v; }\n")},
			warn(rtB, 3, rtA, 4, "clamp(int): same name and parameter types")},
		{"same body twice in one file",
			map[string]string{rtA: class("BA", helper("one", body(8))+helper("two", body(8)))},
			warn(rtA, 15, rtA, 3, "two: same body, 10 statements")},
		{"a renamed identifier is another shape",
			map[string]string{rtA: class("BA", helper("one", body(8))), rtB: class("BB", strings.ReplaceAll(helper("two", body(8)), "out", "res"))}, ""},
		// Tuning (2026-10-24 real trees): the signature rule reads helpers only, private or static methods,
		// which cannot override; a public or package instance method may implement a supertype's method
		// without @Override (TableModel getters, a manager command's doInvoke, a value type's toJson).
		{"instance methods that may override are not helpers",
			map[string]string{rtA: class("BA", "  public int getRowCount() { return rows.length; }\n  String toJson() { return a; }\n"),
				rtB: class("BB", "  public int getRowCount() { return 0; }\n  String toJson() { return b; }\n")}, ""},
		{"a private instance helper is one",
			map[string]string{rtA: class("BA", "  private void logError(String w, Throwable t) { log.severe(w); }\n"),
				rtB: class("BB", "  private void logError(String where, Throwable e) { LOG.log(where, e); }\n")},
			warn(rtB, 3, rtA, 3, "logError(String,Throwable): same name and parameter types")},
		// Tuning (2026-10-24, DashboardFord AckOutcome/WriteOutcome.refused): a static factory returning its
		// own class constructs that class, as a constructor does.
		{"static factories of two types",
			map[string]string{rtA: class("BA", "  static BA refused(String id, String why) { return new BA(false, id, why); }\n"),
				rtB: class("BB", "  static BB refused(String slot, String why) { return new BB(false, slot, why, 0); }\n")}, ""},
		// Tuning (2026-10-24, DashboardFord AlarmJson.unavailable -> FordJson.unavailable): a one-statement body
		// that calls the same-named method on another type is the shared helper's caller, the fix itself.
		{"a delegate to the shared helper",
			map[string]string{rtA: class("BA", "  static String unavailable(String r) { return \"{\\\"r\\\":\" + r + \"}\"; }\n"),
				rtB: class("BB", "  static String unavailable(String reason) {\n    return BA.unavailable(reason);\n  }\n")}, ""},
		{"a longer body that also calls it is still a copy",
			map[string]string{rtA: class("BA", "  static String unavailable(String r) { return r; }\n"),
				rtB: class("BB", "  static String unavailable(String r) { log(r); return BA.unavailable(r); }\n")},
			warn(rtB, 3, rtA, 3, "unavailable(String): same name and parameter types")},
		{"overloads are not the same signature",
			map[string]string{rtA: class("BA", "  int f(int a) { return a; }\n"), rtB: class("BB", "  int f(long a) { return 1; }\n")}, ""},
		{"constructors are not helpers",
			map[string]string{"Mod/Mod-rt/src/a/Util.java": "package a;\nclass Util {\n  Util() { x(); }\n}\n", "Mod/Mod-wb/src/b/Util.java": "package b;\nclass Util {\n  Util() { y(); }\n}\n"}, ""},
		{"anonymous-class methods implement their type",
			map[string]string{rtA: class("BA", "  void a() { go(new Runnable() { public void run() { x(); } }); }\n"), rtB: class("BB", "  void b() { go(new Runnable() { public void run() { y(); } }); }\n")}, ""},
		{"srcTest and a directory that is not a part are not read",
			map[string]string{rtA: class("BA", helper("one", body(8))), "Mod/Mod-rt/srcTest/com/x/T.java": class("T", helper("one", body(8))),
				"Mod/tools/src/G.java": class("G", helper("one", body(8)))}, ""},
	} {
		if got := rows(t, tc.files); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}
