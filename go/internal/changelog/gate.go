package changelog

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of changelog-gate (the kit's 0/1/3 contract).
const (
	ExitPass = 0 // an entry, a valid trailer, or a range that changes no kit behaviour
	ExitFail = 1 // kit behaviour changed with no entry, or a malformed fragment or trailer
	ExitEnv  = 3 // usage error, a malformed range, or git missing or failing
)

// MaxReasonLen caps the reason of a `Changelog: none (<reason>)` trailer, in characters.
const MaxReasonLen = 120

// behaviourRoots are the trees whose changes change kit behaviour. notBehaviour are the retro gate's own
// exclusions inside them (.githooks/pre-push `relevant`): built binaries and the ledger/retro bookkeeping.
var (
	behaviourRoots = []string{"go/", "build-n4-module-kit/", "tools/", "scripts/"}
	notBehaviour   = regexp.MustCompile(`^(go/bin/|build-n4-module-kit/(BUILD-STATE(\.src)?\.md$|ledger\.d/|retros/))`)
	trailer        = regexp.MustCompile(`^Changelog: none \((.*)\)$`)
	hunk           = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)
)

const usage = "usage: changelog-gate --range <base>..<head>"

// IsBehaviour reports whether a changed path changes kit behaviour (the paths the gate applies to).
func IsBehaviour(path string) bool {
	for _, r := range behaviourRoots {
		if strings.HasPrefix(path, r) {
			return !notBehaviour.MatchString(path)
		}
	}
	return false
}

// CheckReason returns why a trailer reason is not acceptable, or "": it must be non-empty, one printable
// line and at most MaxReasonLen characters.
func CheckReason(reason string) string {
	r := strings.TrimSpace(reason)
	switch {
	case r == "":
		return "the reason is empty"
	case utf8.RuneCountInString(r) > MaxReasonLen:
		return fmt.Sprintf("the reason is longer than %d characters", MaxReasonLen)
	case strings.IndexFunc(r, func(c rune) bool { return !unicode.IsPrint(c) }) >= 0:
		return "the reason holds a non-printable character"
	}
	return ""
}

type gitRunner func(args ...string) (string, error)

func realGit(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if ee, ok := err.(*exec.ExitError); ok {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(ee.Stderr)))
	}
	return string(out), err
}

// Main implements `changelog-gate --range <base>..<head>`, run in the repository.
func Main(args []string, s cli.Stdio) int {
	if len(args) != 2 || args[0] != "--range" {
		fmt.Fprintln(s.Err, usage)
		return ExitEnv
	}
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(s.Err, "changelog-gate: git is not on PATH; refusing to pass the range unchecked")
		return ExitEnv
	}
	return run(args[1], realGit, s)
}

func run(rng string, git gitRunner, s cli.Stdio) int {
	env := func(err error) int {
		fmt.Fprintf(s.Err, "changelog-gate: %v; refusing to pass the range unchecked\n", err)
		return ExitEnv
	}
	ends := strings.Split(rng, "..")
	if len(ends) != 2 || ends[0] == "" || ends[1] == "" || strings.HasPrefix(ends[1], ".") {
		fmt.Fprintf(s.Err, "changelog-gate: malformed range %q\n%s\n", rng, usage)
		return ExitEnv
	}
	for _, e := range ends {
		if _, err := git("rev-parse", "--verify", "--quiet", e+"^{commit}"); err != nil {
			return env(fmt.Errorf("range end %q is not a commit", e))
		}
	}
	head := ends[1]
	changedOut, err := git("diff", "--name-only", "--no-renames", rng)
	if err != nil {
		return env(err)
	}
	addedOut, err := git("diff", "--name-only", "--no-renames", "--diff-filter=AM", rng, "--", FragmentDir+"/")
	if err != nil {
		return env(err)
	}
	logOut, err := git("log", "--format=%B", rng)
	if err != nil {
		return env(err)
	}

	var problems, fragments, reasons []string
	for _, p := range nonEmpty(addedOut) {
		if !strings.HasPrefix(p, FragmentDir+"/") {
			continue
		}
		why := CheckName(strings.TrimPrefix(p, FragmentDir+"/"))
		if why == "" {
			body, err := git("show", head+":"+p)
			if err != nil {
				return env(err)
			}
			why = CheckFragment(body)
		}
		if why != "" {
			problems = append(problems, fmt.Sprintf("%s: %s", p, why))
			continue
		}
		fragments = append(fragments, p)
	}
	for _, l := range strings.Split(logOut, "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "Changelog:") {
			continue
		}
		m := trailer.FindStringSubmatch(l)
		why := "not the form 'Changelog: none (<reason>)'"
		if m != nil {
			why = CheckReason(m[1])
		}
		if why != "" {
			problems = append(problems, fmt.Sprintf("trailer %q: %s", l, why))
			continue
		}
		reasons = append(reasons, strings.TrimSpace(m[1]))
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(s.Err, "changelog-gate: %s\n", p)
		}
		fmt.Fprintln(s.Err, "changelog-gate: malformed changelog fragment or trailer in range; refusing (see CONTRIBUTING.md § 6.2)")
		return ExitFail
	}

	var behaviour []string
	changed := nonEmpty(changedOut)
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
	if contains(changed, "CHANGELOG.md") {
		ok, err := addsUnreleased(rng, head, git)
		if err != nil {
			return env(err)
		}
		if ok {
			fmt.Fprintf(s.Out, "changelog-gate: PASS — CHANGELOG.md %s gains an entry\n", UnreleasedHeader)
			return ExitPass
		}
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
	fmt.Fprintf(s.Err, "       or add a line under CHANGELOG.md %s, or a 'Changelog: none (<reason>)' commit trailer.\n", UnreleasedHeader)
	return ExitFail
}

// addsUnreleased reports whether the range's CHANGELOG.md diff adds a non-blank line strictly inside the
// `## [Unreleased]` section of the head version.
func addsUnreleased(rng, head string, git gitRunner) (bool, error) {
	cur, err := git("show", head+":CHANGELOG.md")
	if err != nil {
		return false, err
	}
	start, end, ok := UnreleasedBounds(strings.Split(cur, "\n"))
	if !ok {
		return false, nil
	}
	patch, err := git("diff", "-U0", "--no-renames", rng, "--", "CHANGELOG.md")
	if err != nil {
		return false, err
	}
	line := 0 // 1-based line number in the head version of the next '+' or ' ' line
	for _, l := range strings.Split(patch, "\n") {
		if m := hunk.FindStringSubmatch(l); m != nil {
			line, _ = strconv.Atoi(m[1])
			continue
		}
		if line == 0 || strings.HasPrefix(l, "+++") || !strings.HasPrefix(l, "+") {
			continue
		}
		if idx := line - 1; idx > start && idx < end && strings.TrimSpace(l[1:]) != "" {
			return true, nil
		}
		line++
	}
	return false, nil
}

func nonEmpty(out string) []string {
	var r []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			r = append(r, l)
		}
	}
	return r
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
