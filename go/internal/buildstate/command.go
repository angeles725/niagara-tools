package buildstate

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Exit codes of gen-build-state (the gen-retro-index contract).
const (
	ExitOK    = gen.ExitOK    // BUILD-STATE.md written, or fresh under --check
	ExitStale = gen.ExitStale // stale under --check, or a fragment / BUILD-STATE structure violation
	ExitEnv   = gen.ExitEnv   // usage or environment error; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-build-state.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_BUILD_STATE_DEFAULT_ROOT"

const tool = "gen-build-state"

// Main implements `gen-build-state [--check] [<kit-root>]`: it rewrites the generated parts of the kit
// envelope of <kit-root>/BUILD-STATE.md from <kit-root>/ledger.d/, or with --check compares them.
// Any violation means nothing is written.
func Main(args []string, s cli.Stdio) int {
	check, root, ok := gen.ParseArgs(tool, DefaultRootEnv, args, s)
	if !ok {
		return ExitEnv
	}
	return generate(root, check, s)
}

// generate regenerates <root>/BUILD-STATE.md from <root>/ledger.d/ and writes it, or compares it under check.
func generate(root string, check bool, s cli.Stdio) int {
	dir, ok := gen.SubDir(tool, root, "ledger.d", s)
	if !ok {
		return ExitEnv
	}
	path := root + "/BUILD-STATE.md"
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
	// Mutation: GBS-stale -- the shared compare in gen.File; dropping it makes --check vacuously clean
	return gen.File{Tool: tool, Path: path, Content: content, StaleNote: " vs ledger.d/", Atomic: true}.Run(check, s)
}
