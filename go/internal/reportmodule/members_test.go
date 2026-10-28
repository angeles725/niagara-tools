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
	if len(got) != 60 || got[0] != "verify-module.sh" || got[len(got)-1] != "triage-console.sh" {
		t.Errorf("Scripts() = %d entries %q...: want the 60 members, verify-module.sh first, triage-console.sh last", len(got), got[:1])
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
		{"a non-zero exit with no FAIL row fails closed", 127, "WARN  c  /a:1  r", none,
			"A  WARN  c  a:1  r\nA  FAIL  l  exited 127 with no FAIL row (crash or unexpected output) -- run l.sh manually\n"},
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
		r.advisoryMember("A", "l", tt.code, tt.out, false)
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

// TestTruncatedMemberIsAnEnvFault (W1): a member whose output passes the bound lost its rows past it (a FAIL row
// among them), so it is an ERROR row and an env fault, never PASS clean.
func TestTruncatedMemberIsAnEnvFault(t *testing.T) {
	// Not parallel: it writes an executable and runs it; a parallel test forking meanwhile holds the write fd
	// and the exec fails ETXTBSY (126). Sequential tests run before the parallel ones resume.
	root := tree(t, "A-rt/src/")
	tb := t.TempDir()
	script(t, tb, "lint-no-system-out.sh", "#!/bin/sh\nprintf 'scanning many files, a long line\\nFAIL  sysout  /x/A.java:3  System.out\\n'\n", 0o755)
	var out, errw bytes.Buffer
	r := &report{out: &out, errw: &errw, n: map[string]int{}, x: runner{tb: tb, env: []string{"PATH=/usr/bin:/bin"}, max: 16}}
	r.step(artifact{"A-rt", root + "/A-rt"}, member{script: "lint-no-system-out", f: failOnly, v: rows})
	want := "A-rt  ERROR  lint-no-system-out  env fault (output over 16 bytes, truncated) -- run lint-no-system-out.sh manually\n"
	if out.String() != want || !r.hadEnv {
		t.Errorf("W1: got %q (env %v), want %q", out.String(), r.hadEnv, want)
	}
}

// TestAdvisoryMemberStrictFail (strict-new Δ4): lint-size exits 1 with FAIL rows in a strict module; the report
// relays them as FAIL (no PASS verdict, no env fault), so a strict module's report never reads PASS or ERROR for
// a size breach. An exit 1 with no FAIL row stays an env fault.
func TestAdvisoryMemberStrictFail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		out       string
		want      string
		fail, env bool
	}{
		{"FAIL rows relay as FAIL", "WARN  lint-size  /x/A.java:2  adapter-branches\nFAIL  lint-size  /x/A.java:6  class A too long",
			"A  WARN  lint-size  A.java:2  adapter-branches\nA  FAIL  lint-size  A.java:6  class A too long\n", true, false},
		{"WARN rows only stay an env fault", "WARN  lint-size  /x/A.java:2  long",
			"A  ERROR  l  env fault (exit 1)\n", false, true},
	}
	for _, tt := range tests {
		var b bytes.Buffer
		r := &report{out: &b, n: map[string]int{}}
		r.advisoryMember("A", "l", 1, tt.out, true)
		if b.String() != tt.want || r.hadFail != tt.fail || r.hadEnv != tt.env {
			t.Errorf("%s: %q (fail %v, env %v), want %q (fail %v, env %v)", tt.name, b.String(), r.hadFail, r.hadEnv, tt.want, tt.fail, tt.env)
		}
	}
}

// TestAdvisoryStrictRelayIsModeAwareOnly: the exit-1-with-FAIL relay is limited to the mode-aware member
// (lint-size, whose rows FAIL in a strict module); another advisory member exiting 1 with a FAIL row is broken,
// not strict, and stays an env fault.
func TestAdvisoryStrictRelayIsModeAwareOnly(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	r := &report{out: &b, n: map[string]int{}}
	r.advisoryMember("A", "l", 1, "FAIL  l  /x/A.java:6  broken", false)
	if want := "A  ERROR  l  env fault (exit 1)\n"; b.String() != want || r.hadFail || !r.hadEnv {
		t.Errorf("not mode-aware: %q (fail %v, env %v), want %q as an env fault", b.String(), r.hadFail, r.hadEnv, want)
	}
	for _, list := range [][]member{perArtifact, perModule} {
		for _, m := range list {
			if m.modeAware != (m.script == "lint-size") {
				t.Errorf("member %s: modeAware %v, want true only for lint-size", m.script, m.modeAware)
			}
		}
	}
}
