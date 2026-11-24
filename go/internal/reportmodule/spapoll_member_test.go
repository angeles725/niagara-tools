package reportmodule

import (
	"bytes"
	"testing"
)

// TestSpaPollFailRowShowsAndIsCounted (issue #966 item 2): lint-spa-poll-no-recovery exits 1 with a FAIL row for a
// swallowed failure (#895) and 0 with WARN rows for the missing recovery; the report shows the FAIL row, counts it
// and fails, and a WARN row is shown and counted as a warning only.
func TestSpaPollFailRowShowsAndIsCounted(t *testing.T) {
	// Not parallel: it writes an executable and runs it (ETXTBSY with a parallel fork, see TestTruncatedMemberIsAnEnvFault).
	var m member
	for _, c := range perArtifact {
		if c.script == "lint-spa-poll-no-recovery" {
			m = c
		}
	}
	if m.script == "" {
		t.Fatal("the member table has no lint-spa-poll-no-recovery")
	}
	root := tree(t, "A-ux/src/rc/")
	tests := []struct {
		name, body, want string
		fail             bool
		fails, warns     int
	}{
		{"a FAIL row (exit 1)", "printf 'FAIL  lint-spa-poll-no-recovery  /x/a.js:7  swallows a failure\\n'; exit 1\n",
			"A-ux  FAIL  lint-spa-poll-no-recovery  a.js:7  swallows a failure\n", true, 1, 0},
		{"a WARN row (exit 0)", "printf 'WARN  lint-spa-poll-no-recovery  /x/a.js:2  no recovery\\n'\n",
			"A-ux  WARN  lint-spa-poll-no-recovery  a.js:2  no recovery\n", false, 0, 1},
	}
	for _, tt := range tests {
		tb := t.TempDir()
		script(t, tb, "lint-spa-poll-no-recovery.sh", "#!/bin/sh\n"+tt.body, 0o755)
		var out, errw bytes.Buffer
		r := &report{out: &out, errw: &errw, n: map[string]int{}, x: runner{tb: tb, env: []string{"PATH=/usr/bin:/bin"}}}
		r.step(artifact{"A-ux", root + "/A-ux"}, m)
		if out.String() != tt.want || r.hadFail != tt.fail || r.hadEnv || r.n[failed] != tt.fails || r.n[warn] != tt.warns {
			t.Errorf("%s: got %q (fail %v, env %v, %d FAIL, %d WARN), want %q (fail %v, %d FAIL, %d WARN)", tt.name,
				out.String(), r.hadFail, r.hadEnv, r.n[failed], r.n[warn], tt.want, tt.fail, tt.fails, tt.warns)
		}
	}
}
