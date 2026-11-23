package batshome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// suite builds a bats file text; its cases are spelled %test so this file's own text is no suite.
func suite(lines ...string) string {
	return strings.ReplaceAll(strings.Join(lines, "\n")+"\n", "%test ", "@test ")
}

var launchers = []string{"build.sh", "orient-guard.sh"}

// TestASuiteThatChangesHomeNeedsThePrebuild pins the rule: a suite that runs a Go launcher under another
// HOME (or none) builds the binary once, in setup_file, with the real environment; the finding names the
// suite, the first line that changes HOME and a launcher the suite runs.
func TestASuiteThatChangesHomeNeedsThePrebuild(t *testing.T) {
	for _, c := range []struct {
		name, text string
		want       string // "" = clean, else the finding row
	}{
		{"env HOME= without a prebuild", suite(`B="$KIT/toolbelt/build.sh"`, `%test "a" {`, `  run env HOME="$H" "$B" x`, `}`),
			"x.bats:3: runs build.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"a HOME= prefix", suite(`SCRIPT=orient-guard.sh`, `HOME=/x run "$SCRIPT"`),
			"x.bats:2: runs orient-guard.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"env -u HOME", suite(`run env -u HOME orient-guard.sh /tmp`),
			"x.bats:1: runs orient-guard.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"unset HOME", suite(`%test "a" {`, `  unset HOME`, `  run build.sh`, `}`),
			"x.bats:2: runs build.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"export HOME=", suite(`setup() { export HOME="$T"; }`, `run build.sh`),
			"x.bats:1: runs build.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"setup_file prebuilds", suite("setup_file() {", `  . "$ROOT/go/go-tool.sh"`, `  go_tool_bin "$ROOT/go" build-module >/dev/null`, "}", `run env HOME=/x build.sh`), ""},
		{"setup_file that builds nothing", suite("setup_file() {", "  :", "}", "setup() { :; }", `run env HOME=/x build.sh`),
			"x.bats:5: runs build.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"go_tool_bin only in a case", suite("setup_file() { :; }", `%test "a" {`, `  go_tool_bin "$ROOT/go" build-module`, `  run env HOME=/x build.sh`, `}`),
			"x.bats:4: runs build.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"no launcher named", suite(`run env HOME=/x some-other.sh`), ""},
		{"HOME unchanged", suite(`run build.sh --home "$H"`, `run env FAKE_HOME=/x SKILL_HOME=/y build.sh`), ""},
		{"a here-document is data", suite(`run build.sh`, `cat > f <<'SH'`, `HOME=/x build.sh`, `SH`), ""},
		{"a command after the here-document is read", suite(`run build.sh`, `cat > f <<EOF`, `x`, `EOF`, `HOME=/y run build.sh`),
			"x.bats:5: runs build.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"HOME= in a quoted assertion text", suite(`run build.sh`, `[[ "$output" == *"runs without HOME="* ]]`), ""},
		{"HOME= with a quoted value is read", suite(`run build.sh`, `HOME="$T/h ome" run x`),
			"x.bats:2: runs build.sh under a changed HOME without a setup_file that calls go_tool_bin"},
		{"a comment is not a command", suite(`# run env HOME=/x build.sh`, `run build.sh`), ""},
		{"a launcher only in a comment", suite(`# build.sh`, `run env HOME=/x other.sh`), ""},
	} {
		got := ""
		if f := Check("x.bats", c.text, launchers); f != nil {
			got = f.String()
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// TestAHelperTheSuiteLoadsMayPrebuild: bats runs the setup_file of a loaded helper as the suite's own.
func TestAHelperTheSuiteLoadsMayPrebuild(t *testing.T) {
	text := suite("load lib/setup", `run env HOME=/x build.sh`)
	lib := "setup_file() {\n  go_tool_bin \"$r/go\" build-module\n}\n"
	if f := Check("x.bats", text, launchers, lib); f != nil {
		t.Errorf("a loaded setup_file that builds: %s", f)
	}
	if f := Check("x.bats", text, launchers, "setup() { :; }\n"); f == nil {
		t.Error("a loaded helper with no prebuild is clean")
	}
}

// TestLaunchersAreTheScriptsThatSourceGoTool: a script is a launcher when it calls go_tool_bin; the
// names come from the kit toolbelt and scripts/ (*.sh, not their lib/).
func TestLaunchersAreTheScriptsThatSourceGoTool(t *testing.T) {
	root := t.TempDir()
	for rel, text := range map[string]string{
		"build-n4-module-kit/toolbelt/a.sh": "BIN=\"$(go_tool_bin \"$GO\" a)\"\n",
		"build-n4-module-kit/toolbelt/b.sh": "echo no go here\n",
		"scripts/c.sh":                      "go_tool_bin x c\n",
		".githooks/pre-push":                "go_tool_bin x pre-push\n", // a hook is no *.sh script
		"scripts/lib/d.sh":                  "go_tool_bin x d\n",        // a lib is sourced, not run
	} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Launchers(root)
	if err != nil || strings.Join(got, " ") != "a.sh c.sh" {
		t.Errorf("launchers %v, error %v", got, err)
	}
}

// TestEverySuiteOfTheRepositoryPrebuilds is the lint on the committed suites (retro
// go-toolbelt-small-port Δ4): none runs a Go launcher under a changed HOME without the setup_file prebuild.
func TestEverySuiteOfTheRepositoryPrebuilds(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	findings, err := Lint(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Error(f)
	}
}
