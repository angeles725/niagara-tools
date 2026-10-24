package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScriptsExist pins the member table to the toolbelt: every script the report runs is a toolbelt file.
func TestScriptsExist(t *testing.T) {
	t.Parallel()
	got := Scripts()
	if len(got) != 54 || got[0] != "verify-module.sh" || got[len(got)-1] != "triage-console.sh" {
		t.Errorf("Scripts() = %d entries %q...: want the 54 members, verify-module.sh first, triage-console.sh last", len(got), got[:1])
	}
	for _, s := range got {
		if _, err := os.Stat(filepath.Join("..", "..", "..", "build-n4-module-kit", "toolbelt", s)); err != nil {
			t.Errorf("member %s: %v", s, err)
		}
	}
}

func TestLintRowsVerdicts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		code int
		out  string
		v    verdict
		want string
	}{
		{"exit 3 is an env fault", 3, "FAIL  c  /a:1  r", rows, "A  ERROR  l  env fault (exit 3)\n"},
		{"none adds no row", 0, "", none, ""},
		{"rows: clean", 0, "", rows, "A  PASS  l  clean\n"},
		{"rows: a WARN row blocks the PASS", 0, "WARN  c  /a:1  r", rows, "A  WARN  c  a:1  r\n"},
		{"fail: a WARN row does not", 0, "WARN  c  /a:1  r", failRows, "A  WARN  c  a:1  r\nA  PASS  l  clean\n"},
		{"fail: a FAIL row does", 1, "FAIL  c  /a:1  r", failRows, "A  FAIL  c  a:1  r\n"},
	}
	for _, tt := range tests {
		var b bytes.Buffer
		r := &report{out: &b, n: map[string]int{}}
		if r.lintRows("A", "l", tt.code, tt.out, failWarn, tt.v); b.String() != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, b.String(), tt.want)
		}
	}
}

func TestAdvisoryMemberReason(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code int
		out  string
		want string
	}{
		{0, "WARN  s  /x/A.java:1  long", "A  WARN  s  A.java:1  long\nA  PASS  l  no failing row (1 WARN row above)\n"},
		{3, "WARN  s  /x/A.java:1  long\n \t\nl: cannot scan /x/U.java", "A  ERROR  l  env fault (exit 3): l: cannot scan /x/U.java\n"},
		{2, "FAIL  s  /x:1  r", "A  ERROR  l  env fault (exit 2)\n"},
		{1, "  indented", "A  ERROR  l  env fault (exit 1):   indented\n"},
	}
	for _, tt := range tests {
		var b bytes.Buffer
		r := &report{out: &b, n: map[string]int{}}
		r.advisoryMember("A", "l", tt.code, tt.out)
		if b.String() != tt.want || r.hadEnv != (tt.code != 0) {
			t.Errorf("advisoryMember(%d) = %q (env %v), want %q", tt.code, b.String(), r.hadEnv, tt.want)
		}
	}
}

func TestStepGates(t *testing.T) {
	t.Parallel()
	root := tree(t, "A-ux/")
	var b bytes.Buffer
	r := &report{out: &b, n: map[string]int{}, x: runner{tb: "/nonexistent"}}
	a := artifact{"A-ux", root + "/A-ux"}
	r.step(a, member{script: "lint-x", suffix: "-rt"})
	r.step(a, member{script: "lint-y"})
	r.step(a, member{script: "lint-z", in: inRc})
	if want := "A-ux  SKIP  lint-y  no src/\nA-ux  SKIP  lint-z  no src/rc/\n"; b.String() != want {
		t.Errorf("gates printed %q, want %q", b.String(), want)
	}
	if strings.Contains(b.String(), "lint-x") {
		t.Error("a member whose suffix the artifact lacks must print nothing")
	}
}
