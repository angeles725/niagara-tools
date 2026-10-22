package facetscard

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of check-facets-card (unchanged from the bash tool).
const (
	ExitOK   = 0 // the card matches the source
	ExitDiff = 1 // any difference
	ExitEnv  = 3 // usage or environment error: an unreadable file, or no keys parsed on either side
)

// Main implements `check-facets-card <card.md> <BFacets.java | keys-file>`.
func Main(args []string, s cli.Stdio) int {
	if len(args) != 2 {
		fmt.Fprint(s.Err, "usage: check-facets-card.sh <card.md> <BFacets.java | keys-file>\n")
		return ExitEnv
	}
	card, src := args[0], args[1]
	texts := make([]string, 2)
	for i, f := range args {
		text, ok := readRegular(f)
		if !ok {
			fmt.Fprintf(s.Err, "check-facets-card: cannot read %s\n", f)
			return ExitEnv
		}
		texts[i] = text
	}
	cardKeys, srcKeys := CardKeys(texts[0]), SourceKeys(src, texts[1])
	// Mutation: FC-empty -- an empty side must be exit 3, never a vacuous match
	if len(cardKeys) == 0 {
		fmt.Fprintf(s.Err, "check-facets-card: no card rows parsed from %s\n", card)
		return ExitEnv
	}
	if len(srcKeys) == 0 {
		fmt.Fprintf(s.Err, "check-facets-card: no key constants parsed from %s\n", src)
		return ExitEnv
	}

	diffs := Compare(cardKeys, srcKeys)
	for _, d := range diffs {
		fmt.Fprintln(s.Out, d)
	}
	if len(diffs) == 0 {
		fmt.Fprintf(s.Out, "check-facets-card: %d keys match %s\n", len(cardKeys), src)
		return ExitOK
	}
	fmt.Fprintf(s.Out, "check-facets-card: card %s differs from %s (%d card rows, %d source keys)\n",
		card, src, len(cardKeys), len(srcKeys))
	return ExitDiff
}

// readRegular returns the contents of path when it is a readable regular file (symlinks followed,
// as `[ -f ] && [ -r ]` did); /dev/null and directories are not.
func readRegular(path string) (string, bool) {
	if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
		return "", false
	}
	b, err := os.ReadFile(path)
	return string(b), err == nil
}
