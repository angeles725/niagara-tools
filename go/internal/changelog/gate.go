package changelog

import (
	"fmt"
	"os/exec"

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

// run checks the range and reports one verdict; each step lives in its own file: rangefacts.go reads the
// range, carriers.go checks fragments and trailers, unreleased.go judges [Unreleased], report.go prints.
func run(rng string, git gitRunner, s cli.Stdio) int {
	f, err := readRange(rng, git)
	if err != nil {
		return refuseEnv(s, err)
	}
	fragments, problems, err := fragmentsIn(f.added, f.head, git)
	if err != nil {
		return refuseEnv(s, err)
	}
	reasons, bad := trailersIn(f.log)
	if problems = append(problems, bad...); len(problems) > 0 {
		return refuseMalformed(s, problems)
	}
	u, err := unreleasedVerdict(f, git)
	if err != nil {
		return refuseEnv(s, err)
	}
	if u == unreleasedOutsideRelease {
		return refuseUnreleased(s)
	}
	return verdict(s, f.changed, fragments, u == unreleasedInRelease, reasons)
}
