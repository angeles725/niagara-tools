package changelog

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of changelog-gate (the kit's 0/1/3 contract).
const (
	ExitPass = 0 // an entry, a valid trailer, or a range that changes no kit behaviour
	ExitFail = 1 // kit behaviour changed with no entry, or a malformed fragment or trailer
	ExitEnv  = 3 // usage error, a malformed range, or git missing or failing
)

const usage = "usage: changelog-gate --range <base>..<head>"

// Main implements `changelog-gate --range <base>..<head>`, run in the repository.
func Main(args []string, s cli.Stdio) int {
	if len(args) != 2 || args[0] != "--range" {
		fmt.Fprintln(s.Err, usage)
		return ExitEnv
	}
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(s.Err, "changelog-gate: git is not on PATH; refusing to pass the range unchecked")
		return ExitEnv
	}
	return run(args[1], realGit, s)
}

func run(rng string, git gitRunner, s cli.Stdio) int {
	env := func(err error) int {
		fmt.Fprintf(s.Err, "changelog-gate: %v; refusing to pass the range unchecked\n", err)
		return ExitEnv
	}
	ends := strings.Split(rng, "..")
	if len(ends) != 2 || ends[0] == "" || ends[1] == "" || strings.HasPrefix(ends[1], ".") {
		fmt.Fprintf(s.Err, "changelog-gate: malformed range %q\n%s\n", rng, usage)
		return ExitEnv
	}
	for _, e := range ends {
		if _, err := git("rev-parse", "--verify", "--quiet", e+"^{commit}"); err != nil {
			return env(fmt.Errorf("range end %q is not a commit", e))
		}
	}
	head := ends[1]
	changedOut, err := git("diff", "--name-only", "--no-renames", rng)
	if err != nil {
		return env(err)
	}
	addedOut, err := git("diff", "--name-only", "--no-renames", "--diff-filter=AM", rng, "--", FragmentDir+"/")
	if err != nil {
		return env(err)
	}
	logOut, err := git("log", "--format=%B", rng)
	if err != nil {
		return env(err)
	}

	fragments, problems, err := fragmentsIn(nonEmpty(addedOut), head, git)
	if err != nil {
		return env(err)
	}
	reasons, bad := trailersIn(logOut)
	problems = append(problems, bad...)
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(s.Err, "changelog-gate: %s\n", p)
		}
		fmt.Fprintln(s.Err, "changelog-gate: malformed changelog fragment or trailer in range; refusing (see CONTRIBUTING.md § 6.2)")
		return ExitFail
	}

	var behaviour []string
	changed := nonEmpty(changedOut)
	for _, p := range changed {
		if IsBehaviour(p) {
			behaviour = append(behaviour, p)
		}
	}
	if len(behaviour) == 0 {
		fmt.Fprintln(s.Out, "changelog-gate: not applicable (no go/, build-n4-module-kit/, tools/ or scripts/ behaviour change in range)")
		return ExitPass
	}
	if len(fragments) > 0 {
		fmt.Fprintf(s.Out, "changelog-gate: PASS — fragment %s\n", strings.Join(fragments, ", "))
		return ExitPass
	}
	if contains(changed, "CHANGELOG.md") {
		ok, err := addsUnreleased(rng, head, git)
		if err != nil {
			return env(err)
		}
		if ok {
			fmt.Fprintf(s.Out, "changelog-gate: PASS — CHANGELOG.md %s gains an entry\n", UnreleasedHeader)
			return ExitPass
		}
	}
	if len(reasons) > 0 {
		fmt.Fprintf(s.Out, "changelog-gate: PASS — Changelog: none (%s)\n", strings.Join(reasons, "; "))
		return ExitPass
	}
	fmt.Fprintln(s.Err, "changelog-gate: kit behaviour changed with no CHANGELOG entry:")
	for i, p := range behaviour {
		if i == 20 {
			fmt.Fprintf(s.Err, "  … and %d more\n", len(behaviour)-20)
			break
		}
		fmt.Fprintf(s.Err, "  %s\n", p)
	}
	fmt.Fprintf(s.Err, "  fix: add %s/<YYYY-MM-DD>-<slug>.md (### Added|Changed|Fixed|Removed — <title>, then '- ' bullets),\n", FragmentDir)
	fmt.Fprintf(s.Err, "       or add a line under CHANGELOG.md %s, or a 'Changelog: none (<reason>)' commit trailer.\n", UnreleasedHeader)
	return ExitFail
}
