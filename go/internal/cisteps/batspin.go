package cisteps

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

var (
	batsVerLine = regexp.MustCompile(`^\s*BATS_VER=([0-9.]+)\s*$`)
	batsShaLine = regexp.MustCompile(`^\s*BATS_SHA256=([0-9a-f]{64})\s*$`)
)

// batsPin reads the bats-core pin of the workflow's first "Install bats*" step: its BATS_VER= and
// BATS_SHA256= lines. found is false when the workflow has no such step. Each variable must be set
// on exactly one line of the step: a missing, malformed or repeated one is an error, so a second
// assignment can never silently pick the pin the replay downloads. (scripts/lib/ci-local-tools.sh
// provision_bats used sed | head -n 1, which took the first of several.)
func batsPin(steps []Step) (ver, sha string, found bool, err error) {
	for _, st := range steps {
		if !strings.HasPrefix(st.Name, "Install bats") {
			continue
		}
		lines := strings.Split(st.Body, "\n")
		if ver, err = pinLine(st.Name, lines, batsVerLine, "BATS_VER=<version>", "BATS_VER="); err != nil {
			return "", "", true, err
		}
		if sha, err = pinLine(st.Name, lines, batsShaLine, "BATS_SHA256=<64 hex>", "BATS_SHA256="); err != nil {
			return "", "", true, err
		}
		return ver, sha, true, nil
	}
	return "", "", false, nil
}

// pinLine is the capture of the one line of the step's script that matches re.
func pinLine(step string, lines []string, re *regexp.Regexp, shape, name string) (string, error) {
	var val string
	var at []int
	for i, l := range lines {
		if m := re.FindStringSubmatch(l); m != nil {
			val, at = m[1], append(at, i+1)
		}
	}
	switch len(at) {
	case 1:
		return val, nil
	case 0:
		return "", fmt.Errorf("ci-steps bats-pin: step %q has no %s line", step, shape)
	}
	return "", fmt.Errorf("ci-steps bats-pin: step %q sets %s more than once (lines %d and %d of the step)", step, name, at[0], at[1])
}

// batsPinMain implements `ci-steps bats-pin <workflow.yml>`: one `<version> <sha256>` line for the
// workflow's "Install bats*" step, nothing (exit 0) when it has none. Exit 2 for usage, an
// unreadable or unparseable workflow, 3 for a missing, malformed or repeated pin.
func batsPinMain(args []string, s cli.Stdio) int {
	if len(args) != 1 {
		fmt.Fprint(s.Err, "usage: ci-steps bats-pin <workflow.yml>\n")
		return ExitBad
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(s.Err, osError(args[0], err))
		return ExitBad
	}
	steps, err := Parse(args[0], data)
	if err != nil {
		fmt.Fprintln(s.Err, err)
		return ExitBad
	}
	ver, sha, found, err := batsPin(steps)
	if err != nil {
		fmt.Fprintln(s.Err, err)
		return ExitEnv
	}
	if found {
		_, _ = io.WriteString(s.Out, ver+" "+sha+"\n")
	}
	return 0
}
