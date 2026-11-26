package reportmodule

import "testing"

// TestIncompleteNamesItsCause (#1116): an INCOMPLETE verdict says whether an environment fault (a member that
// could not run: an ERROR row) or a module check that was skipped (a floor SKIP: no built jar, no
// .deploy-baseline) left the report incomplete, so a skipped module check never hides among environment faults.
// Both causes are listed when both hold, and a FAIL still reads ISSUES with the cause in its note. The exit
// codes are the ones the verdict always had (the shell wrapper and build.sh key on them): the cause is text only.
func TestIncompleteNamesItsCause(t *testing.T) {
	envFault := map[string]string{"lint-delays": "echo 'lint-delays: java is not installed' >&2; exit 3\n"}
	tests := []struct {
		name     string
		module   func(*testing.T) string // a module whose floor checks run, or one whose floor SKIPs
		stubs    map[string]string
		flags    []string
		wantCode int
		verdict  string
	}{
		{"environment fault alone", baselineModule, envFault, nil, ExitEnv, "INCOMPLETE (environment)"},
		{"floor SKIP alone", floorSkipModule, nil, nil, ExitIncomplete, "INCOMPLETE (module)"},
		{"both", floorSkipModule, envFault, nil, ExitEnv, "INCOMPLETE (environment, module)"},
		{"FAIL and floor SKIP", floorSkipModule, map[string]string{"lint-delays": stubDelays}, nil, ExitFail,
			"ISSUES (+INCOMPLETE (module))"},
		{"FAIL, floor SKIP and environment fault", floorSkipModule,
			map[string]string{"lint-delays": stubDelays, "lint-no-system-out": "exit 3\n"}, nil, ExitEnv,
			"ISSUES (+INCOMPLETE (environment, module))"},
		{"--lint-only leaves a floor SKIP out of the cause", floorSkipModule, envFault, []string{"--lint-only"}, ExitEnv,
			"INCOMPLETE (environment)"},
		{"complete run stays CLEAN", baselineModule, nil, nil, ExitClean, "CLEAN"},
		{"FAIL alone stays ISSUES", baselineModule, map[string]string{"lint-delays": stubDelays}, nil, ExitFail, "ISSUES"},
	}
	for _, tt := range tests {
		root := tt.module(t)
		code, out, _ := baselineRun(t, tt.stubs, append(tt.flags, root)...)
		if got := summaryLine(out); code != tt.wantCode || !endsWith(got, "->  "+tt.verdict) {
			t.Errorf("%s: exit %d, summary %q; want exit %d and a summary ending in %q", tt.name, code, got, tt.wantCode, tt.verdict)
		}
	}
}

// floorSkipModule is a one-artifact module with no built jar and no .deploy-baseline: both floor checks SKIP.
func floorSkipModule(t *testing.T) string {
	t.Helper()
	return tree(t, "A-rt/src/")
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
