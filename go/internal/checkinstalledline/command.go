// Package checkinstalledline refuses a client branch that would silently downgrade a module group that is
// already installed (scripts/check-installed-line.sh): every GROUP build.gradle.kts carrying
// defaultModuleVersion("X") at the installed ref must exist on the branch with a version >= X.
// git.go reads the repository (local refs only, never a fetch), internal/vendorversion reads and compares a version;
// this file only composes them. [ev: retro r10-installed-line-guard Δ1]
package checkinstalledline

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// Exit codes of check-installed-line (unchanged from the bash tool).
const (
	ExitOK      = 0 // every group is at or above its installed version
	ExitRefused = 1 // any DOWNGRADE, MISSING or INCOMPARABLE row
	ExitUsage   = 2 // usage, an unknown ref, or no defaultModuleVersion at the installed ref
)

const tool = "check-installed-line"

// Main implements `check-installed-line <repo> <branch> <installed-ref>`. Rows, one per group in path
// order: `ok|DOWNGRADE|INCOMPARABLE <file> <installed> -> <branch>` or `MISSING <file> <installed>`; a
// <file> holding a control character, a double quote or a backslash is printed quoted (quote.go).
// An installed ref with nothing to guard is exit 2, never a pass.
func Main(args []string, s cli.Stdio) int {
	if len(args) != 3 {
		fmt.Fprintf(s.Err, "usage: %s.sh <repo> <branch> <installed-ref>\n", tool)
		return ExitUsage
	}
	repo := repository{dir: args[0], stderr: s.Err}
	branch, installed := args[1], args[2]
	for _, ref := range []string{branch, installed} {
		if !repo.isCommit(ref) {
			fmt.Fprintf(s.Out, "%s: unknown ref %s\n", tool, ref)
			return ExitUsage
		}
	}
	code, groups := ExitOK, 0
	for _, file := range repo.filesNaming(installed, "defaultModuleVersion(", "*build.gradle.kts") {
		want := vendorversion.SourceVersion(repo.show(installed, file))
		if want == "" {
			continue // the file names the call (a comment) without making it: not a group
		}
		groups++
		text, refused := row(file, want, vendorversion.SourceVersion(repo.show(branch, file)))
		fmt.Fprintln(s.Out, text)
		if refused {
			code = ExitRefused
		}
	}
	if groups == 0 {
		fmt.Fprintf(s.Out, "%s: no defaultModuleVersion in any build.gradle.kts at %s\n", tool, installed)
		return ExitUsage
	}
	return code
}

// row is the line of one group and whether it refuses the branch; got is "" when the branch has no
// version for the file.
func row(file, want, got string) (text string, refused bool) {
	file = display(file)
	if got == "" {
		// Mutation: ILG-missing -- skipping a group the branch lost passes a branch that dropped it
		return "MISSING " + file + " " + want, true
	}
	status := "ok"
	switch c, comparable := vendorversion.Cmp(got, want); {
	case !comparable:
		// Mutation: ILG-incomparable -- reading an incomparable pair as ok passes 2.10.0 -> 2.10.0-rc
		status = "INCOMPARABLE"
	case c < 0:
		// Mutation: ILG-downgrade -- dropping this case passes the branch that downgrades a group
		status = "DOWNGRADE"
	}
	return status + " " + file + " " + want + " -> " + got, status != "ok"
}
