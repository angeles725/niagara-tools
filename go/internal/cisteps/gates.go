package cisteps

import (
	"path"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/prepush"
)

// Gate is one check `ci-local.sh --changed` runs when a changed file matches one of its Triggers. Pkgs (the
// `go test` of Go packages whose tests read the repository tree) or Step (the body of the ci.yml step whose name
// matches that glob, replayed as CI runs it) says what runs, so the check cannot differ from CI's.
type Gate struct {
	Name     string                 // what ci-local prints
	Triggers []string               // repo-relative globs; ** matches any directories; a Pkgs gate also fires on its own sources
	Match    func(path string) bool // or a predicate: the repository's own definition of a file set
	Pkgs     []string               // packages under go/ whose tests read the tree
	Step     string                 // ci.yml run-step name glob replayed
	Soft     bool                   // any non-zero exit prints WARN and does not fail (a label decides in CI)
	Twins    []string               // further ci.yml step globs this gate stands for
	Cmd      string                 // else this bash command, from the repo root ($CI_BASE, $CI_HEAD name the change)
}

const goStep = "Go gofmt + vet + test*" // the ci.yml step that runs every Go test; the `go test` gates run only the part of it a change needs

const (
	kitTree  = "build-n4-module-kit/**"
	mcpTree  = "mcp-n4-kit/**"
	fixtures = "tests/fixtures/**"
	toolbelt = "build-n4-module-kit/toolbelt/"
)

// prSizeCmd skips with a line in a tree that has no scripts/pr-size.sh (a throwaway fixture repository);
// TestGateCommandsNameScriptsTheRepositoryHas keeps the real one from going missing.
const prSizeCmd = `[ -f scripts/pr-size.sh ] || { echo "pr-size: no scripts/pr-size.sh in this tree"; exit 0; }; bash scripts/pr-size.sh "$CI_BASE" "$CI_HEAD"`

// shellHygieneCmd runs the kit's shell-hygiene suite (SH1: no unquoted `&` or `$` in a pattern-substitution replacement);
// like prSizeCmd it skips with a line in a tree that has no tests/shell-hygiene.bats (a throwaway fixture repository).
const shellHygieneCmd = `[ -f tests/shell-hygiene.bats ] || { echo "shell-hygiene: no tests/shell-hygiene.bats in this tree"; exit 0; }; bats tests/shell-hygiene.bats`

// Gates is the one table: change -> check. TestEveryCIStepAndTreeGateHasALocalTwin fails when a ci.yml run step
// or a Go package that reads the tree has no row here.
var Gates = []Gate{
	{Name: "go test ./internal/filesize", Pkgs: []string{"internal/filesize"}, Twins: []string{goStep},
		Triggers: []string{"go/**/*.go", "go/internal/filesize/modularity-debt.txt"}},
	{Name: "go test ./internal/cardwidth", Pkgs: []string{"internal/cardwidth"}, Twins: []string{goStep},
		Match: prepush.IsLiveDoctrine, Triggers: []string{"go/internal/cardwidth/baseline.txt", "go/internal/prepush/paths.go"}},
	{Name: "go test (the suites the bats files feed)", Pkgs: []string{"internal/batshome"}, Twins: []string{goStep},
		Triggers: []string{"tests/*.bats", "tests/helpers/*"}},
	{Name: "go test (readers of build-n4-module-kit)", Twins: []string{goStep},
		Pkgs: []string{"internal/buildstate", "internal/buildstatefields", "internal/facetscard", "internal/kitstate",
			"internal/ledgercheck", "internal/lint/rules/structure", "internal/lintindex", "internal/retroindex",
			"internal/scaffoldmodule", "internal/storedrepack", "internal/triageconsole", "internal/vendorfloor", "internal/verifymodule"},
		Triggers: []string{kitTree, "tests/fixtures/facets-card/**", "tests/fixtures/triage-console/**"}},
	{Name: "go test (readers of mcp-n4-kit)", Twins: []string{goStep},
		Pkgs: []string{"internal/mcpn4/bql", "internal/mcpn4/journal", "internal/mcpn4/px", "internal/mcpn4/pyxml",
			"internal/mcpn4/readtools", "internal/mcpn4/retro"},
		Triggers: []string{mcpTree}},
	{Name: "go test (readers of scripts, tests and hooks)", Twins: []string{goStep},
		Pkgs: []string{"internal/checkbatsskips", "internal/checkclientsource", "internal/cisteps", "internal/foldaudit",
			"internal/guardpins", "internal/obixlinkaudit", "internal/prepush", "internal/schemarisk", "internal/skillinstall"},
		Triggers: []string{"scripts/*.sh", "scripts/lib/*.sh", ".githooks/*", ".github/workflows/*.yml", "tests/bats-skip-allowlist.txt",
			fixtures, "*-kit/skill/**", kitTree + "/toolbelt/lint-*.sh"}},
	// A changed bats file or bats helper can break shell-hygiene.bats, which the other bats lints do not run (issue #1052).
	{Name: "bats tests/shell-hygiene.bats", Triggers: []string{"tests/*.bats", "tests/lib/**"}, Cmd: shellHygieneCmd},
	{Name: "bats-shard --check", Step: "bats-shard --check*", Triggers: []string{"tests/*.bats", "scripts/bats-shard.sh"}},
	{Name: "sweep-build-state + sweep-fold-audit", Step: "sweep-build-state + sweep-fold-audit*",
		Triggers: []string{toolbelt + "*.sh", "build-n4-module-kit/retros/**", "build-n4-module-kit/ledger.d/**", "build-n4-module-kit/BUILD-STATE.src.md"}},
	{Name: "lint-guard-pins --strict", Step: "lint-guard-pins*", Triggers: []string{toolbelt + "*.sh", "tests/*.bats"}},
	{Name: "gen-lint-index --check", Step: "gen-lint-index*", Triggers: []string{toolbelt + "**"}},
	{Name: "gen-retro-index --check", Step: "gen-retro-index*", Triggers: []string{"build-n4-module-kit/retros/**", toolbelt + "gen-retro-index.sh"}},
	{Name: "gen-build-state --check", Step: "gen-build-state*",
		Triggers: []string{"build-n4-module-kit/ledger.d/**", "build-n4-module-kit/BUILD-STATE.src.md", toolbelt + "gen-build-state.sh"}},
	{Name: "gen-changelog --check", Step: "gen-changelog*", Triggers: []string{"changelog.d/**", "scripts/gen-changelog.sh"}},
	{Name: "lint-timers", Step: "lint-timers*", Triggers: []string{"tests/fixtures/lint-timers/**", toolbelt + "lint-timers.sh"}},
	{Name: "scaffold-diff", Step: "scaffold-diff*",
		Triggers: []string{"build-n4-module-kit/templates/**", "build-n4-module-kit/fixtures/**", toolbelt + "scaffold-module.sh"}},
	{Name: "schema-risk fixtures", Step: "schema-risk fixtures*", Triggers: []string{"tests/fixtures/schema-risk/**", toolbelt + "schema-risk.sh"}},
	{Name: "plano-ok fixture", Step: "plano-ok fixture*", Triggers: []string{"tests/fixtures/plano/**", toolbelt + "verify-module.sh"}},
	{Name: "triage-console smoke", Step: "triage-console basic smoke*", Triggers: []string{"tests/fixtures/triage-console/**", toolbelt + "triage-console.sh"}},
	// pr-size.yml, not ci.yml: the 400-line rule; locally a WARN, as the size:exception label decides in CI.
	{Name: "pr-size (WARN only: the size:exception label decides in CI)", Soft: true, Triggers: []string{"**"},
		Cmd: prSizeCmd},
}

// FullOnly lists the ci.yml run steps `--changed` does not run, each with why (the full replay runs them).
var FullOnly = []struct{ Step, Why string }{
	{"Install the pinned npm tools*", "installs from the network; the Go and bats steps that need node check for it"},
	{"Select this job's bats shard*", "matrix plumbing: CI's shard variables reach later steps through $GITHUB_ENV"},
	{"Go build*", "a compile check; the gates and gofmt above compile what they run"},
	{"Pre-build go/bin*", "a cache warm-up for the parallel bats suites"},
	{"bats (all unit tests)*", "the whole suite takes minutes; --changed runs the bats lints and kit-links over the change"},
	{"run-pure-test.bats*", "needs the pre-fetched JUnit jars; part of the full replay"},
	{"sweep-retro-debt*", "an aging report, non-strict: it cannot fail"},
	{"changelog-gate over*", ".githooks/pre-push runs it on the pushed range, which --changed does not know"},
}

// NotTreeGates lists packages the scan reads as tree readers that need no --changed gate, each with why.
var NotTreeGates = map[string]string{
	"internal/buildmodule":         "golden suite over the fixtures, about 20 s: `go test ./...` and the full replay run it",
	"internal/commissioningverify": "golden suite over the toolbelt, about 10 s: `go test ./...` and the full replay run it",
	"internal/reportmodule":        "end-to-end suite over the toolbelt, about 45 s: `go test ./...` and the full replay run it",
	"internal/mcpn4/blackbox":      "end-to-end suite of the server, about 45 s: `go test ./...` and the full replay run it",
	"internal/javasrc":             "its tree read is a benchmark, which `go test` does not run",
	"internal/orientguard":         "its `../..` literal is a path-normalisation case, not a read",
}

// planSteps are the ci.yml steps `--changed` runs through its own plan rows (shellcheck, bats lints, gofmt).
var planSteps = []string{"shellcheck (scripts, toolbelt, tests)*", "lint-bats-heredoc*"}

// SelectGates returns the gates a change needs, in table order: one whose Triggers (or, for a Pkgs gate, whose
// packages' own sources) match a changed path.
func SelectGates(paths []string) []Gate {
	var out []Gate
	for _, g := range Gates {
		triggers := g.Triggers
		for _, p := range g.Pkgs {
			triggers = append(triggers[:len(triggers):len(triggers)], "go/"+p+"/**")
		}
	match:
		for _, f := range paths {
			if g.Match != nil && g.Match(f) {
				out = append(out, g)
				break
			}
			for _, t := range triggers {
				if globMatch(strings.Split(t, "/"), strings.Split(f, "/")) {
					out = append(out, g)
					break match
				}
			}
		}
	}
	return out
}

// Command is what runs: the gate's Cmd, the `go test` of its packages, or "step:<glob>" for a ci.yml step replay.
func (g Gate) Command() string {
	switch {
	case g.Step != "":
		return "step:" + g.Step
	case g.Cmd != "":
		return g.Cmd
	}
	return "cd go && go test -count=1 ./" + strings.Join(g.Pkgs, " ./")
}

// globMatch matches path segments against pattern segments: ** is any number of directories, the rest as path.Match.
func globMatch(pat, p []string) bool {
	if len(pat) == 0 {
		return len(p) == 0
	}
	if pat[0] == "**" {
		for i := 0; i <= len(p); i++ {
			if globMatch(pat[1:], p[i:]) {
				return true
			}
		}
		return false
	}
	// A malformed pattern is no match here; TestGateTriggersAreValidPathPatterns fails one at test time.
	ok, _ := path.Match(pat[0], firstOr(p))
	return len(p) > 0 && ok && globMatch(pat[1:], p[1:])
}

func firstOr(p []string) string {
	if len(p) == 0 {
		return ""
	}
	return p[0]
}
