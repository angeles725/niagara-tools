package retroindex

import (
	"fmt"
	"io"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Exit codes of gen-retro-index (unchanged from the bash tool).
const (
	ExitOK    = gen.ExitOK    // index written, or fresh under --check
	ExitStale = gen.ExitStale // stale or missing index under --check, or a retro-header violation
	ExitEnv   = gen.ExitEnv   // usage or environment error; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-retro-index.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_RETRO_INDEX_DEFAULT_ROOT"

const tool = "gen-retro-index"

// Main implements `gen-retro-index [--check] [<kit-root>]`: it writes <kit-root>/retros/INDEX.md,
// or with --check compares it with what the retro headers generate. Any violation means nothing
// is written.
func Main(args []string, s cli.Stdio) int {
	check, root, ok := gen.ParseArgs(tool, DefaultRootEnv, args, s)
	if !ok {
		return ExitEnv
	}
	return generate(root, check, s)
}

// Check reports whether <root>/retros/INDEX.md is fresh against the retro headers, as
// `--check <root>` decides, without the "is fresh" line; its diagnostics (stale, missing,
// violations) go to diag. It is the typed API sweep-build-state calls in process.
func Check(root string, diag io.Writer) bool {
	return generate(root, true, cli.Stdio{Out: io.Discard, Err: diag}) == ExitOK
}

// generate renders <root>/retros/INDEX.md and writes it, or compares it under check.
func generate(root string, check bool, s cli.Stdio) int {
	dir, ok := gen.SubDir(tool, root, "retros", s)
	if !ok {
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
	// Mutation: GRI-stale -- the shared compare in gen.File; dropping it makes --check vacuously clean
	return gen.File{Tool: tool, Path: out, Content: Render(entries) + "\n", TrimNewlines: true}.Run(check, s)
}
