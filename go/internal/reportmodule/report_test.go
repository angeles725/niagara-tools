package reportmodule

import (
	"bytes"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func TestSummaryVerdictAndExit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                              string
		fail, env, floor, lintOnly        bool
		artifacts                         int
		wantCode                          int
		wantVerdict, wantPlural, wantNote string
	}{
		{"clean", false, false, false, false, 1, ExitClean, "CLEAN", "1 artifact ·", ""},
		{"floor skip", false, false, true, false, 2, ExitIncomplete, "INCOMPLETE", "2 artifacts ·", ""},
		{"lint-only floor", false, false, true, true, 1, ExitClean, "CLEAN", "1 artifact ·", "--lint-only"},
		{"fail and floor skip show both", true, false, true, false, 1, ExitFail, "ISSUES (+INCOMPLETE)", "", ""},
		{"fail with a --lint-only floor skip", true, false, true, true, 1, ExitFail, "ISSUES", "", "--lint-only"},
		{"env beats fail, verdict text stays", true, true, false, false, 1, ExitEnv, "ISSUES", "", ""},
		{"env alone reads INCOMPLETE, never CLEAN", false, true, false, false, 1, ExitEnv, "INCOMPLETE", "", ""},
		{"env under --lint-only reads INCOMPLETE", false, true, false, true, 1, ExitEnv, "INCOMPLETE", "", ""},
	}
	for _, tt := range tests {
		var out, errw bytes.Buffer
		r := &report{out: &out, errw: &errw, n: map[string]int{}, hadFail: tt.fail, hadEnv: tt.env, floorSkip: tt.floor,
			o: options{lintOnly: tt.lintOnly}}
		code := r.summary(tt.artifacts)
		s := out.String()
		if code != tt.wantCode || !bytes.HasSuffix(out.Bytes(), []byte("->  "+tt.wantVerdict+"\n")) ||
			!bytes.Contains(out.Bytes(), []byte(tt.wantPlural)) || !bytes.Contains(errw.Bytes(), []byte(tt.wantNote)) ||
			(tt.wantNote == "" && errw.Len() > 0) {
			t.Errorf("%s: exit %d, out %q, err %q", tt.name, code, s, errw.String())
		}
	}
}

func TestEmitCounts(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	r := &report{out: &out, n: map[string]int{}}
	for _, st := range []string{pass, failed, warn, skip, advisory, errored, "INFO", "FAILED"} {
		r.emit("A", st, "c", "d")
	}
	if r.n[pass] != 1 || r.n[failed] != 1 || r.n[warn] != 1 || r.n[skip] != 1 || r.n[advisory] != 1 || len(r.n) != 5 || !r.hadFail {
		t.Errorf("counts %v, hadFail %v: only the five summary statuses count, and FAIL alone sets hadFail", r.n, r.hadFail)
	}
	if want := "A  PASS  c  d\n"; out.String()[:len(want)] != want {
		t.Errorf("row = %q", out.String())
	}
}

// TestMissingToolbeltIsRefused (W2): a toolbelt directory that does not exist would make every member exit 127
// (rows members then read PASS clean); the report refuses before any member runs.
func TestMissingToolbeltIsRefused(t *testing.T) {
	t.Parallel()
	root := tree(t, "A-rt/src/")
	for _, tb := range []string{root + "/no-such-toolbelt", root + "/A-rt/src/x"} {
		var out, errw bytes.Buffer
		env := map[string]string{ToolbeltEnv: tb}
		s := cli.Stdio{Out: &out, Err: &errw, Getenv: func(k string) string { return env[k] }}
		code := mainEnv([]string{"--lint-only", root}, s, []string{"PATH=/usr/bin:/bin"})
		if want := "report-module: toolbelt not found: " + tb + "\n"; code != ExitEnv || out.Len() != 0 || errw.String() != want {
			t.Errorf("W2 %s: exit %d, out %q, err %q; want exit 3, no row, %q", tb, code, out.String(), errw.String(), want)
		}
	}
}
