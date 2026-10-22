package buildstate

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of gen-build-state (the gen-retro-index contract).
const (
	ExitOK    = 0 // BUILD-STATE.md written, or fresh under --check
	ExitStale = 1 // stale under --check, or a fragment / BUILD-STATE structure violation
	ExitEnv   = 3 // usage or environment error; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-build-state.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_BUILD_STATE_DEFAULT_ROOT"

// Main implements `gen-build-state [--check] [<kit-root>]`: it rewrites the generated parts of the kit
// envelope of <kit-root>/BUILD-STATE.md from <kit-root>/ledger.d/, or with --check compares them.
// Any violation means nothing is written.
func Main(args []string, s cli.Stdio) int {
	check, root := false, ""
	for _, a := range args {
		switch {
		case a == "--check":
			check = true
		case strings.HasPrefix(a, "-") || root != "":
			fmt.Fprint(s.Err, "usage: gen-build-state.sh [--check] [<kit-root>]\n")
			return ExitEnv
		default:
			root = a
		}
	}
	if root == "" {
		if root = s.Env(DefaultRootEnv); root == "" {
			fmt.Fprintf(s.Err, "gen-build-state: no <kit-root> given and %s is unset (run toolbelt/gen-build-state.sh)\n", DefaultRootEnv)
			return ExitEnv
		}
	}
	dir, path := root+"/ledger.d", root+"/BUILD-STATE.md"
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		fmt.Fprintf(s.Err, "gen-build-state: no ledger.d/ under %s\n", root)
		return ExitEnv
	}
	committed, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(s.Err, "gen-build-state: no BUILD-STATE.md under %s\n", root)
		return ExitEnv
	}

	frags, problems, err := Scan(dir)
	if err != nil {
		fmt.Fprintf(s.Err, "gen-build-state: cannot list %s: %v; BUILD-STATE.md not written\n", dir, err)
		return ExitEnv
	}
	unreadable := false
	for _, p := range problems {
		fmt.Fprintf(s.Err, "gen-build-state: %s: %s\n", p.Name, p.Reason)
		unreadable = unreadable || p.IO
	}
	switch {
	case unreadable:
		fmt.Fprint(s.Err, "gen-build-state: a fragment could not be read; BUILD-STATE.md not written\n")
		return ExitEnv
	case len(problems) > 0:
		fmt.Fprint(s.Err, "gen-build-state: ledger fragment violation; BUILD-STATE.md not written\n")
		return ExitStale
	case len(frags) == 0:
		fmt.Fprint(s.Err, "gen-build-state: ledger.d/ holds no fragment; BUILD-STATE.md not written\n")
		return ExitStale
	}
	content, err := Render(string(committed), frags)
	if err != nil {
		fmt.Fprintf(s.Err, "gen-build-state: %v; BUILD-STATE.md not written\n", err)
		return ExitStale
	}

	if check {
		// Mutation: GBS-stale -- dropping the generated-vs-committed compare makes --check vacuously clean
		if content != string(committed) {
			fmt.Fprintf(s.Err, "gen-build-state: %s is stale vs ledger.d/; run gen-build-state.sh and commit the result\n", path)
			showDiff(content, path, s)
			return ExitStale
		}
		fmt.Fprintf(s.Out, "gen-build-state: %s is fresh\n", path)
		return ExitOK
	}
	if err := writeAtomic(path, content); err != nil {
		fmt.Fprintf(s.Err, "gen-build-state: cannot write %s: %v\n", path, err)
		return ExitEnv
	}
	fmt.Fprintf(s.Out, "gen-build-state: wrote %s\n", path)
	return ExitOK
}

// writeAtomic replaces path through a temporary file in its directory, so a failed write never
// leaves a truncated ledger behind.
func writeAtomic(path, content string) error {
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmp, []byte(content), mode); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// showDiff writes `diff - <path>` (generated vs committed) to s.Err; diagnostic only.
func showDiff(generated, path string, s cli.Stdio) {
	cmd := exec.Command("diff", "-", path)
	cmd.Stdin = strings.NewReader(generated)
	cmd.Stdout, cmd.Stderr = s.Err, s.Err
	_ = cmd.Run()
}
