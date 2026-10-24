package buildstate

import (
	"fmt"
	"io"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Exit codes of gen-build-state (the gen-retro-index contract).
const (
	ExitOK    = gen.ExitOK    // BUILD-STATE.md written, or the inputs are valid under --check
	ExitStale = gen.ExitStale // a fragment or BUILD-STATE.src.md violation; nothing written
	ExitEnv   = gen.ExitEnv   // usage or environment error; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-build-state.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_BUILD_STATE_DEFAULT_ROOT"

// SourceName is the committed, hand-kept source of BUILD-STATE.md: every module envelope and every kit
// field, with the kit envelope's generated parts left empty. OutputName is the generated file; it is
// not committed (.gitignore), so two PRs that each add a fragment never touch the same line.
const (
	SourceName = "BUILD-STATE.src.md"
	OutputName = "BUILD-STATE.md"
)

const tool = "gen-build-state"

// Main implements `gen-build-state [--check] [<kit-root>]`: it writes <kit-root>/BUILD-STATE.md from
// <kit-root>/BUILD-STATE.src.md and <kit-root>/ledger.d/, or with --check only validates those inputs
// (the generated file is not committed, so there is nothing to compare). Any violation means nothing
// is written.
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
		fmt.Fprintf(s.Out, "gen-build-state: %s/%s and ledger.d/ are valid\n", root, SourceName)
		return ExitOK
	}
	return Output(tool, root, content).Write(s)
}

// Output is the generated BUILD-STATE.md of root as a file to write; tool prefixes its messages.
func Output(tool, root, content string) gen.File {
	return gen.File{Tool: tool, Path: root + "/" + OutputName, Content: content}
}

// Generate returns the BUILD-STATE.md that <root>/BUILD-STATE.src.md and <root>/ledger.d/ generate:
// gen.GeneratedLine, then the source with the kit envelope's generated parts filled in. A violation
// is reported to diag and returned as ExitStale (ExitEnv for a missing or unreadable input).
func Generate(root string, diag io.Writer) (string, int) {
	s := cli.Stdio{Out: io.Discard, Err: diag}
	dir, ok := gen.SubDir(tool, root, "ledger.d", s)
	if !ok {
		return "", ExitEnv
	}
	src, err := os.ReadFile(root + "/" + SourceName)
	if err != nil {
		fmt.Fprintf(diag, "gen-build-state: no %s under %s\n", SourceName, root)
		return "", ExitEnv
	}

	frags, problems, err := Scan(dir)
	if err != nil {
		fmt.Fprintf(diag, "gen-build-state: cannot list %s: %v; nothing generated\n", dir, err)
		return "", ExitEnv
	}
	unreadable := false
	for _, p := range problems {
		fmt.Fprintf(diag, "gen-build-state: %s: %s\n", p.Name, p.Reason)
		unreadable = unreadable || p.IO
	}
	switch {
	case unreadable:
		fmt.Fprint(diag, "gen-build-state: a fragment could not be read; nothing generated\n")
		return "", ExitEnv
	case len(problems) > 0:
		fmt.Fprint(diag, "gen-build-state: ledger fragment violation; nothing generated\n")
		return "", ExitStale
	case len(frags) == 0:
		fmt.Fprint(diag, "gen-build-state: ledger.d/ holds no fragment; nothing generated\n")
		return "", ExitStale
	}
	content, err := Render(string(src), frags)
	if err != nil {
		fmt.Fprintf(diag, "gen-build-state: %s: %v; nothing generated\n", SourceName, err)
		return "", ExitStale
	}
	// Mutation: GBS-source -- without this check a generated line pasted into the source is dropped silently
	if empty, _ := Render(string(src), nil); empty != string(src) {
		fmt.Fprintf(diag, "gen-build-state: %s holds generated content (a last_change value, open_issues items or "+
			"session-log lines in the kit envelope); move it into a ledger.d/ fragment; nothing generated\n", SourceName)
		gen.ShowDiff(empty, root+"/"+SourceName, diag)
		return "", ExitStale
	}
	return gen.GeneratedLine + "\n" + content, ExitOK
}
