package cisteps

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const goRoot = "../.."

// skipGlobs reads the step globs scripts/ci-local.sh skips on purpose (SKIP_RULES), so the guard shares them.
func skipGlobs(t *testing.T) []string {
	t.Helper()
	var globs []string
	for _, m := range regexp.MustCompile(`(?m)^  "([^|"]+)\|`).FindAllStringSubmatch(string(mustRead(t, "../../../scripts/ci-local.sh")), -1) {
		globs = append(globs, m[1])
	}
	if len(globs) == 0 {
		t.Fatal("no SKIP_RULES in scripts/ci-local.sh")
	}
	return globs
}

// TestEveryCIStepAndTreeGateHasALocalTwin is the guard: a new ci.yml run step or a Go test that reads the
// repository tree fails here until the --changed plan (gates.go) runs it or says why not. A tree reader is found only
// by a relative-path literal in a *_test.go ("../../x", filepath.Join("..", "..", "x")): a reader that locates the
// repository through runtime.Caller, os.Getwd or `git rev-parse` is not detected and needs its Gate row by hand.
func TestEveryCIStepAndTreeGateHasALocalTwin(t *testing.T) {
	steps, err := Parse("ci.yml", mustRead(t, "../../../.github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range CheckParity(steps, skipGlobs(t), goRoot) {
		t.Error(m)
	}
}

func TestParityNamesTheUnmappedStepAndWhereToAddIt(t *testing.T) {
	steps, err := Parse("w.yml", []byte(head+"      - name: brand-new-gate\n        run: echo x\n      - name: shellcheck all\n        run: shellcheck a\n"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	msgs := CheckParity(steps, nil, dir)
	got := strings.Join(msgs, "\n")
	if !strings.Contains(got, `ci.yml step "brand-new-gate" has no local twin: add a Gate`) || !strings.Contains(got, "go/internal/cisteps/gates.go") {
		t.Errorf("messages %q", msgs)
	}
	if err := os.MkdirAll(dir+"/internal/x", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/internal/x/x_test.go", []byte("package x\nvar k = \"../../../docs/a.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(CheckParity(nil, nil, dir), "\n"); !strings.Contains(got, "package internal/x reads the repository tree in its tests and has no gate") {
		t.Errorf("unmapped tree reader: %q", got)
	}
}

func TestSelectGatesMatchesTheChangedPathsToTheirTriggers(t *testing.T) {
	names := func(paths ...string) string {
		var n []string
		for _, g := range SelectGates(paths) {
			n = append(n, g.Name)
		}
		return strings.Join(n, "|")
	}
	if got := names("build-n4-module-kit/METHODOLOGY.md"); !strings.Contains(got, "go test ./internal/cardwidth") || strings.Contains(got, "filesize") {
		t.Errorf("a kit .md: %q, want the cardwidth gate and no filesize gate", got)
	}
	if got := names("go/internal/x/x.go"); !strings.Contains(got, "go test ./internal/filesize") || strings.Contains(got, "cardwidth") {
		t.Errorf("a Go file: %q, want the filesize gate and no cardwidth gate", got)
	}
	if got := names("README.md"); !strings.Contains(got, "go test ./internal/cardwidth") {
		t.Errorf("a live card: %q", got)
	}
	if got := names("build-n4-module-kit/retros/2026-11-15-x.md", "docs/a.md"); strings.Contains(got, "cardwidth") {
		t.Errorf("a .md cardwidth does not read: %q", got)
	}
	if got := names("go/internal/batshome/batshome.go"); !strings.Contains(got, "suites the bats files feed") {
		t.Errorf("a gate package's own source: %q", got)
	}
	if got := names("scripts/notes.txt"); got != "pr-size (WARN only: the size:exception label decides in CI)" {
		t.Errorf("an unrelated file: %q, want only pr-size", got)
	}
}

func TestChangedPlansGateRowsOnlyWithTheGatesFlag(t *testing.T) {
	dir, _ := gitRepo(t)
	write(t, dir, "README.md", "x\n")
	tree := filepath.Join(t.TempDir(), "tree")
	code, out, errs := runChanged(t, dir, "--tree", tree, "--gates")
	if code != 0 || errs != "" {
		t.Fatalf("exit %d\n%s", code, errs)
	}
	want := "gate\tgo test ./internal/cardwidth\thard\tcd go && go test -count=1 ./internal/cardwidth\n" +
		"gate\tpr-size (WARN only: the size:exception label decides in CI)\tsoft\t" + prSizeCmd + "\n"
	if out != want {
		t.Errorf("rows\n%q\nwant\n%q", out, want)
	}
	if code, out, _ := runChanged(t, dir, "--tree", tree); code != 0 || out != "" {
		t.Errorf("without --gates: exit %d, plan %q, want no rows", code, out)
	}
}

// TestGateCommandsNameScriptsTheRepositoryHas: a Cmd gate that names a scripts/*.sh the repository lacks would
// skip forever in the fixtures and fail nowhere.
func TestGateCommandsNameScriptsTheRepositoryHas(t *testing.T) {
	n := 0
	for _, g := range Gates {
		if strings.ContainsAny(g.Command(), "\n\t") {
			t.Errorf("gate %q: its command is a plan row, one line without a tab", g.Name)
		}
		for _, s := range regexp.MustCompile(`scripts/[\w-]+\.sh`).FindAllString(g.Cmd, -1) {
			n++
			if _, err := os.Stat("../../../" + s); err != nil {
				t.Errorf("gate %q runs %s: %v", g.Name, s, err)
			}
		}
	}
	if n == 0 {
		t.Error("no gate Cmd names a script: the check is vacuous")
	}
}

// TestGateTriggersAreValidPathPatterns: globMatch drops path.Match's error, so a malformed trigger would never match
// and its gate would silently stop running.
func TestGateTriggersAreValidPathPatterns(t *testing.T) {
	n := 0
	for _, g := range Gates {
		for _, trig := range g.Triggers {
			for _, seg := range strings.Split(trig, "/") {
				n++
				if _, err := path.Match(seg, "x"); err != nil {
					t.Errorf("gate %q: trigger %q: segment %q: %v", g.Name, trig, seg, err)
				}
			}
		}
	}
	if n == 0 {
		t.Error("no gate has a trigger: the check is vacuous")
	}
}
