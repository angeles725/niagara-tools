package retroindex

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of gen-retro-index (unchanged from the bash tool).
const (
	ExitOK    = 0 // index written, or fresh under --check
	ExitStale = 1 // stale or missing index under --check, or a retro-header violation
	ExitEnv   = 3 // usage or environment error; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-retro-index.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_RETRO_INDEX_DEFAULT_ROOT"

// Main implements `gen-retro-index [--check] [<kit-root>]`: it writes <kit-root>/retros/INDEX.md,
// or with --check compares it with what the retro headers generate. Any violation means nothing
// is written.
func Main(args []string, s cli.Stdio) int {
	check, root := false, ""
	for _, a := range args {
		switch {
		case a == "--check":
			check = true
		case strings.HasPrefix(a, "-") || root != "":
			fmt.Fprint(s.Err, "usage: gen-retro-index.sh [--check] [<kit-root>]\n")
			return ExitEnv
		default:
			root = a // an empty argument leaves the default in place, as the bash tool did
		}
	}
	if root == "" {
		if root = s.Env(DefaultRootEnv); root == "" {
			fmt.Fprintf(s.Err, "gen-retro-index: no <kit-root> given and %s is unset (run toolbelt/gen-retro-index.sh)\n", DefaultRootEnv)
			return ExitEnv
		}
	}
	dir := root + "/retros"
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		fmt.Fprintf(s.Err, "gen-retro-index: no retros/ under %s\n", root)
		return ExitEnv
	}
	out := dir + "/INDEX.md"

	entries, problems, err := Scan(dir)
	if err != nil {
		fmt.Fprintf(s.Err, "gen-retro-index: cannot list %s: %v; index not written\n", dir, err)
		return ExitEnv
	}
	unreadable := false
	for _, p := range problems {
		fmt.Fprintf(s.Err, "gen-retro-index: %s: %s\n", p.Name, p.Reason)
		unreadable = unreadable || p.IO
	}
	if unreadable {
		fmt.Fprint(s.Err, "gen-retro-index: a retro could not be read; index not written\n")
		return ExitEnv
	}
	if len(problems) > 0 {
		fmt.Fprint(s.Err, "gen-retro-index: retro-header violation; index not written\n")
		return ExitStale
	}
	content := Render(entries)

	if check {
		if st, err := os.Stat(out); err != nil || !st.Mode().IsRegular() {
			fmt.Fprintf(s.Err, "gen-retro-index: %s is missing; run gen-retro-index.sh\n", out)
			return ExitStale
		}
		committed, err := os.ReadFile(out)
		// Mutation: GRI-stale -- dropping the generated-vs-committed compare makes --check vacuously clean
		// Trailing newlines are ignored on both sides, as the bash $(...) comparison did.
		if err != nil || strings.TrimRight(string(committed), "\n") != content {
			fmt.Fprintf(s.Err, "gen-retro-index: %s is stale; run gen-retro-index.sh and commit the result\n", out)
			showDiff(content+"\n", out, s)
			return ExitStale
		}
		fmt.Fprintf(s.Out, "gen-retro-index: %s is fresh\n", out)
		return ExitOK
	}

	if err := os.WriteFile(out, []byte(content+"\n"), 0o666); err != nil {
		fmt.Fprintf(s.Err, "gen-retro-index: cannot write %s: %v\n", out, err)
		return ExitEnv
	}
	fmt.Fprintf(s.Out, "gen-retro-index: wrote %s\n", out)
	return ExitOK
}

// showDiff writes `diff - <path>` (generated vs committed, POSIX normal format) to s.Err; it is
// diagnostic only, so a missing diff binary prints nothing.
func showDiff(generated, path string, s cli.Stdio) {
	cmd := exec.Command("diff", "-", path)
	cmd.Stdin = strings.NewReader(generated)
	cmd.Stdout, cmd.Stderr = s.Err, s.Err
	_ = cmd.Run()
}
