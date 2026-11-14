package checkclientsource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildState: CS10, CS11. --build-state reads the module's source_of_truth from its envelope.
func TestBuildState(t *testing.T) {
	f := newFixture(t)
	state := filepath.Join(f.base, "BUILD-STATE.md")
	write(t, state, "<!-- build-state.v1 -->\nmodule: Other\nsource_of_truth: elsewhere@main@"+f.first+"\n<!-- /build-state.v1 -->\n"+
		"<!-- build-state.v1 -->\nmodule: Mod\nsource_of_truth: r@main@"+f.second+"   # DECLARED\n<!-- /build-state.v1 -->\n"+
		"<!-- build-state.v1 -->\nmodule: Bare\n<!-- /build-state.v1 -->\n")
	pass := "PASS  source-of-truth  " + f.top + " at the declared commit " + f.second + " on main\n"
	undeclared := "WARN  source-of-truth  source_of_truth undeclared for this module — record <path>@<branch>@<commit> in its BUILD-STATE.src.md envelope\n"
	expect(t, "CS10", []string{"--build-state", state, "--module", "Mod", f.root}, ExitOK, pass, "")
	expect(t, "CS11 no field", []string{"--build-state", state, "--module", "Bare", f.root}, ExitOK, undeclared, "")
	expect(t, "no envelope for the module", []string{"--build-state", state, "--module", "Nope", f.root}, ExitOK, undeclared, "")
	expect(t, "--declared wins over --build-state", []string{"--build-state", state, "--module", "Other", "--declared", "r@main@" + f.second, f.root}, ExitOK, pass, "")
	expect(t, "--build-state needs --module", []string{"--build-state", state, f.root}, ExitUsage, "", wantUsage)
	missing := filepath.Join(f.base, "missing.md")
	expect(t, "missing BUILD-STATE", []string{"--build-state", missing, "--module", "Mod", f.root}, ExitEnv, "",
		"check-client-source: BUILD-STATE not found: "+missing+"\n")
}

// TestUsage: CS-usage, CS-malformed. Usage and a malformed declared value are exit 2, a missing module
// root exit 3, and -h prints the help on stdout.
func TestUsage(t *testing.T) {
	f := newFixture(t)
	nope := filepath.Join(f.base, "nope")
	cases := []struct {
		name string
		args []string
		code int
		err  string
	}{
		{"no module root", nil, ExitUsage, wantUsage},
		{"two module roots", []string{f.root, f.root}, ExitUsage, wantUsage},
		{"unknown option", []string{"--nope", f.root}, ExitUsage, wantUsage},
		{"an option with no value", []string{"--declared"}, ExitUsage, wantUsage},
		{"missing module root", []string{"--declared", "unknown", nope}, ExitEnv, "check-client-source: module root not found: " + nope + "\n"},
		{"CS-malformed", []string{"--declared", "r-main", f.root}, ExitUsage,
			"check-client-source: malformed source_of_truth (want <path>@<branch>@<commit> or unknown): r-main\n"},
		{"a fourth field", []string{"--declared", "r@main@abc@x", f.root}, ExitUsage, wantUsage},
		{"an empty field", []string{"--declared", "r@@abc", f.root}, ExitUsage, wantUsage},
	}
	for _, c := range cases {
		expect(t, c.name, c.args, c.code, "", c.err)
	}
	code, out, errText := run("-h")
	if code != ExitOK || errText != "" || !strings.HasPrefix(out, "check-client-source.sh — is this module root the declared SOURCE OF TRUTH checkout?\n") ||
		!strings.HasSuffix(out, "      (module root missing, BUILD-STATE file missing, git absent)\n") {
		t.Errorf("-h: exit %d stderr %q stdout %q", code, errText, out)
	}
	expect(t, "-- ends the options", []string{"--", f.root}, ExitOK,
		"WARN  source-of-truth  source_of_truth undeclared for this module — record <path>@<branch>@<commit> in its BUILD-STATE.src.md envelope\n", "")
}

// TestHelpMatchesTheWrapperHeader: usage.txt is the header of scripts/check-client-source.sh (its lines
// 2-28 without the leading "# "), which the bash tool printed for -h; the two never drift.
func TestHelpMatchesTheWrapperHeader(t *testing.T) {
	b, err := os.ReadFile("../../../scripts/check-client-source.sh")
	if err != nil {
		t.Fatal(err)
	}
	var want strings.Builder
	for _, line := range strings.Split(string(b), "\n")[1:28] {
		want.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "#"), " ") + "\n")
	}
	if helpText != want.String() {
		t.Errorf("usage.txt differs from the wrapper header:\n%s\nwant:\n%s", helpText, want.String())
	}
}
