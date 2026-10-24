package retroindex

import (
	"fmt"
	"io"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Exit codes of gen-retro-index (unchanged from the bash tool).
const (
	ExitOK    = gen.ExitOK    // index written, or the retro headers are valid under --check
	ExitStale = gen.ExitStale // a retro-header violation; nothing written
	ExitEnv   = gen.ExitEnv   // usage or environment error; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-retro-index.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_RETRO_INDEX_DEFAULT_ROOT"

const tool = "gen-retro-index"

// Main implements `gen-retro-index [--check] [<kit-root>]`: it writes <kit-root>/retros/INDEX.md, or
// with --check only validates the retro headers (the index is not committed, so there is nothing to
// compare). Any violation means nothing is written.
func Main(args []string, s cli.Stdio) int {
	check, root, ok := gen.ParseArgs(tool, DefaultRootEnv, args, s)
	if !ok {
		return ExitEnv
	}
	content, code := Generate(root, s.Err)
	if code != ExitOK {
		return code
	}
	if check {
		fmt.Fprintf(s.Out, "gen-retro-index: the retro headers under %s/retros are valid\n", root)
		return ExitOK
	}
	return Output(tool, root, content).Write(s)
}

// Output is the generated retros/INDEX.md of root as a file to write; tool prefixes its messages.
func Output(tool, root, content string) gen.File {
	return gen.File{Tool: tool, Path: root + "/retros/INDEX.md", Content: content, Atomic: true}
}

// Generate returns the retros/INDEX.md that the headers of <root>/retros/*.md generate. A violation
// is reported to diag and returned as ExitStale (ExitEnv for a missing dir or an unreadable retro).
func Generate(root string, diag io.Writer) (string, int) {
	dir, ok := gen.SubDir(tool, root, "retros", cli.Stdio{Out: io.Discard, Err: diag})
	if !ok {
		return "", ExitEnv
	}
	entries, problems, err := Scan(dir)
	if err != nil {
		fmt.Fprintf(diag, "gen-retro-index: cannot list %s: %v; index not generated\n", dir, err)
		return "", ExitEnv
	}
	unreadable := false
	for _, p := range problems {
		fmt.Fprintf(diag, "gen-retro-index: %s: %s\n", p.Name, p.Reason)
		unreadable = unreadable || p.IO
	}
	if unreadable {
		fmt.Fprint(diag, "gen-retro-index: a retro could not be read; index not generated\n")
		return "", ExitEnv
	}
	if len(problems) > 0 {
		fmt.Fprint(diag, "gen-retro-index: retro-header violation; index not generated\n")
		return "", ExitStale
	}
	return Render(entries) + "\n", ExitOK
}
