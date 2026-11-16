// Package checkbatsskips is the global bats SKIP allowlist assertion (audit A6; the CI bats job runs it
// through scripts/check-bats-skips.sh): it reads the TAP of a bats run and fails when a `# skip <reason>`
// matches no line of the allowlist, so a suite cannot start skipping for a new reason and still read green.
// tap.go reads the TAP, allowlist.go the patterns; this file only composes them.
package checkbatsskips

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of check-bats-skips (unchanged from the bash tool).
const (
	ExitOK       = 0 // every skip is allowlisted
	ExitUnlisted = 1 // a skip reason matches no allowlist line
	ExitEnv      = 3 // usage, an unreadable TAP or allowlist, a TAP with no plan, an allowlist line that is not an ERE of the dialect
)

const tool = "check-bats-skips"

// Main implements `check-bats-skips <bats.tap> <allowlist>`: one `UNLISTED skip: <test> — <reason>` row
// on stdout per skip no allowlist line matches, then the count (stderr and exit 1 when any is unlisted).
func Main(args []string, s cli.Stdio) int {
	if len(args) != 2 {
		return env(s, "usage: "+tool+".sh <bats.tap> <allowlist>")
	}
	texts := make([]string, len(args))
	for i, path := range args {
		text, ok := readRegular(path)
		if !ok {
			return env(s, "cannot read "+path)
		}
		texts[i] = text
	}
	tapPath, allowPath := args[0], args[1]
	// Mutation: SK5-no-plan -- dropping this check reads an empty or truncated TAP (bats never ran) as "no skips"
	if !HasPlan(texts[0]) {
		return env(s, "no 1..N plan in "+tapPath+" (bats did not run)")
	}
	allow, fault := ParseAllowlist(texts[1])
	if fault != "" {
		// Mutation: SK6-bad-ere -- skipping a line that does not compile makes it a silent non-match
		return env(s, fault)
	}
	skips := Skips(texts[0])
	unlisted := 0
	for _, skip := range skips {
		// Mutation: SK2-any-skip-ok -- accepting every reason lets a suite skip for a new reason and stay green
		if allow.Allows(skip.Reason) {
			continue
		}
		unlisted++
		reason := skip.Reason
		if reason == "" {
			reason = "(no reason)"
		}
		fmt.Fprintf(s.Out, "UNLISTED skip: %s — %s\n", skip.Test, reason)
	}
	if unlisted > 0 {
		fmt.Fprintf(s.Err, "%s: %d of %d skip(s) match no line of %s\n", tool, unlisted, len(skips), allowPath)
		return ExitUnlisted
	}
	fmt.Fprintf(s.Out, "%s: %d skip(s), all allowlisted\n", tool, len(skips))
	return ExitOK
}

// env prints one `check-bats-skips: <message>` line on stderr and returns ExitEnv.
func env(s cli.Stdio, message string) int {
	fmt.Fprintf(s.Err, "%s: %s\n", tool, message)
	return ExitEnv
}

// readRegular reads a regular file (a symlink to one counts); anything else is not readable input.
func readRegular(path string) (string, bool) {
	if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
		return "", false
	}
	b, err := os.ReadFile(path)
	return string(b), err == nil
}
