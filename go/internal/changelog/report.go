package changelog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// refuseEnv reports a range the gate could not check: a malformed range (with the usage) or git failing.
func refuseEnv(s cli.Stdio, err error) int {
	var m malformedRange
	if errors.As(err, &m) {
		fmt.Fprintf(s.Err, "changelog-gate: malformed range %q\n%s\n", m.rng, usage)
		return ExitEnv
	}
	fmt.Fprintf(s.Err, "changelog-gate: %v; refusing to pass the range unchecked\n", err)
	return ExitEnv
}

func refuseMalformed(s cli.Stdio, problems []string) int {
	for _, p := range problems {
		fmt.Fprintf(s.Err, "changelog-gate: %s\n", p)
	}
	fmt.Fprintln(s.Err, "changelog-gate: malformed changelog fragment or trailer in range; refusing (see CONTRIBUTING.md § 6.2)")
	return ExitFail
}

func refuseUnreleased(s cli.Stdio) int {
	fmt.Fprintf(s.Err, "changelog-gate: CHANGELOG.md %s gains a line outside a release (the range deletes no %s/ fragment and leaves VERSION unchanged); two PRs that both add lines there conflict\n", UnreleasedHeader, FragmentDir)
	fmt.Fprintf(s.Err, "  fix: move the entry to %s/<YYYY-MM-DD>-<slug>.md; only scripts/gen-changelog.sh --write adds lines to %s, at release (CONTRIBUTING.md § 6.2)\n", FragmentDir, UnreleasedHeader)
	return ExitFail
}

// verdict passes a range with no behaviour change or with a carrier (a fragment, a release [Unreleased]
// entry, a trailer, in that order) and refuses the rest, listing at most 20 behaviour paths.
func verdict(s cli.Stdio, changed, fragments []string, unreleased bool, reasons []string) int {
	var behaviour []string
	for _, p := range changed {
		if IsBehaviour(p) {
			behaviour = append(behaviour, p)
		}
	}
	if len(behaviour) == 0 {
		fmt.Fprintln(s.Out, "changelog-gate: not applicable (no go/, build-n4-module-kit/, tools/ or scripts/ behaviour change in range)")
		return ExitPass
	}
	if len(fragments) > 0 {
		fmt.Fprintf(s.Out, "changelog-gate: PASS — fragment %s\n", strings.Join(fragments, ", "))
		return ExitPass
	}
	if unreleased {
		fmt.Fprintf(s.Out, "changelog-gate: PASS — CHANGELOG.md %s gains an entry from the release fold\n", UnreleasedHeader)
		return ExitPass
	}
	if len(reasons) > 0 {
		fmt.Fprintf(s.Out, "changelog-gate: PASS — Changelog: none (%s)\n", strings.Join(reasons, "; "))
		return ExitPass
	}
	fmt.Fprintln(s.Err, "changelog-gate: kit behaviour changed with no CHANGELOG entry:")
	for i, p := range behaviour {
		if i == 20 {
			fmt.Fprintf(s.Err, "  … and %d more\n", len(behaviour)-20)
			break
		}
		fmt.Fprintf(s.Err, "  %s\n", p)
	}
	fmt.Fprintf(s.Err, "  fix: add %s/<YYYY-MM-DD>-<slug>.md (### Added|Changed|Fixed|Removed — <title>, then '- ' bullets),\n", FragmentDir)
	fmt.Fprintln(s.Err, "       or a 'Changelog: none (<reason>)' commit trailer.")
	return ExitFail
}
