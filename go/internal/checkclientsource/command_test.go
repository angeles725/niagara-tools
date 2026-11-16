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
		!strings.HasSuffix(out, "      (module root missing, BUILD-STATE file missing or unreadable, git absent)\n") {
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

// TestUnreadableBuildStateIsAnEnvFault: a BUILD-STATE that exists and cannot be read is exit 3 naming
// the file, with no row; reading it as "undeclared" (WARN, exit 0) would pass a checkout nobody compared
// (retro go-check-scripts-ports Δ2).
func TestUnreadableBuildStateIsAnEnvFault(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	f := newFixture(t)
	state := filepath.Join(f.base, "BUILD-STATE.md")
	write(t, state, "<!-- build-state.v1 -->\nmodule: Mod\nsource_of_truth: r@main@"+f.second+"\n<!-- /build-state.v1 -->\n")
	if err := os.Chmod(state, 0); err != nil {
		t.Fatal(err)
	}
	expect(t, "unreadable BUILD-STATE", []string{"--build-state", state, "--module", "Mod", f.root}, ExitEnv, "",
		"check-client-source: cannot read "+state+": open "+state+": permission denied\n")
}

// TestBuildStateWithCRLFLineEndings: a BUILD-STATE file saved with CRLF line endings declares what it
// declares; read with the CR left on each value it named no module and the check WARNed "undeclared"
// (exit 0) over a checkout nobody compared.
func TestBuildStateWithCRLFLineEndings(t *testing.T) {
	f := newFixture(t)
	state := filepath.Join(f.base, "BUILD-STATE.md")
	write(t, state, "<!-- build-state.v1 -->\r\nmodule: Mod\r\nsource_of_truth: r@main@"+f.second+"\r\n<!-- /build-state.v1 -->\r\n"+
		"<!-- build-state.v1 -->\r\nmodule: Old  # stale\r\nsource_of_truth: r@main@"+f.first+"   # DECLARED\r\n<!-- /build-state.v1 -->\r\n")
	expect(t, "the declared commit", []string{"--build-state", state, "--module", "Mod", f.root}, ExitOK,
		"PASS  source-of-truth  "+f.top+" at the declared commit "+f.second+" on main\n", "")
	expect(t, "a value before a comment", []string{"--build-state", state, "--module", "Old", f.root}, ExitOK,
		"PASS  source-of-truth  "+f.top+" is ahead of the declared commit "+f.first+" (new local work) — update source_of_truth at close\n", "")
}

// TestEmptyBuildStatePathIsAUsageError: `--build-state ""` (an unset shell variable) names no file; it is
// the usage error, never "undeclared" and exit 0. With --declared, which wins, it is still not read.
func TestEmptyBuildStatePathIsAUsageError(t *testing.T) {
	f := newFixture(t)
	expect(t, "an empty path", []string{"--build-state", "", "--module", "Mod", f.root}, ExitUsage, "", wantUsage)
	expect(t, "an empty path and no module", []string{"--build-state", "", f.root}, ExitUsage, "", wantUsage)
	expect(t, "--declared still wins", []string{"--build-state", "", "--declared", "r@main@" + f.second, f.root}, ExitOK,
		"PASS  source-of-truth  "+f.top+" at the declared commit "+f.second+" on main\n", "")
}

// TestBuildStateSpellingsOfOneDeclaration: the same envelope saved by another editor is the same
// declaration: a UTF-8 byte-order mark (alone or with CRLF), CR-only line endings, a doubled CR before the
// LF, and blanks after the value of the module or the source_of_truth line.
func TestBuildStateSpellingsOfOneDeclaration(t *testing.T) {
	f := newFixture(t)
	lines := []string{"<!-- build-state.v1 -->", "module: Mod", "source_of_truth: r@main@" + f.second, "<!-- /build-state.v1 -->", ""}
	trailing := []string{"<!-- build-state.v1 -->", "module: Mod \t", "source_of_truth: r@main@" + f.second + "  \t", "<!-- /build-state.v1 -->", ""}
	cases := []struct{ name, text string }{
		{"a byte-order mark", bom + strings.Join(lines, "\n")},
		{"a byte-order mark and CRLF", bom + strings.Join(lines, "\r\n")},
		{"CR-only line endings", strings.Join(lines, "\r")},
		{"a doubled CR before the LF", strings.Join(lines, "\r\r\n")},
		{"blanks after the values", strings.Join(trailing, "\n")},
	}
	for _, c := range cases {
		state := filepath.Join(f.base, "BUILD-STATE.md")
		write(t, state, c.text)
		expect(t, c.name, []string{"--build-state", state, "--module", "Mod", f.root}, ExitOK,
			"PASS  source-of-truth  "+f.top+" at the declared commit "+f.second+" on main\n", "")
	}
}

// TestBuildStateKeepsAByteOrderMarkInsideTheText: only a mark that opens the file is dropped; one in the
// middle of a line still belongs to that line, so the envelope it would open is not one.
func TestBuildStateKeepsAByteOrderMarkInsideTheText(t *testing.T) {
	f := newFixture(t)
	state := filepath.Join(f.base, "BUILD-STATE.md")
	write(t, state, "notes\n"+bom+"<!-- build-state.v1 -->\nmodule: Mod\nsource_of_truth: r@main@"+f.second+"\n<!-- /build-state.v1 -->\n")
	expect(t, "a mark in the text", []string{"--build-state", state, "--module", "Mod", f.root}, ExitOK,
		"WARN  source-of-truth  source_of_truth undeclared for this module — record <path>@<branch>@<commit> in its BUILD-STATE.src.md envelope\n", "")
}
