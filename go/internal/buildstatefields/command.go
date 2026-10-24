package buildstatefields

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/buildstate"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
	"github.com/angeles725/niagara-tools/go/internal/retroindex"
)

// Exit codes of sweep-build-state (unchanged from the bash tool).
const (
	ExitOK   = 0 // clean (the --age report always exits 0)
	ExitFail = 1 // a named integrity violation
	ExitEnv  = 3 // usage or environment error
)

// DefaultMaxAge is the --age escalation threshold in days.
const DefaultMaxAge = 30

const (
	usageText    = "usage: sweep-build-state.sh <BUILD-STATE.md> <retros-dir> <INDEX.md>\n"
	ageUsageText = "usage: sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age N] <retros-dir> <INDEX.md>\n"
)

// Main implements `sweep-build-state <BUILD-STATE.md> <retros-dir> <INDEX.md>` (the content check)
// and `sweep-build-state --age --today <YYYY-MM-DD> [--max-age N] <retros-dir> <INDEX.md>` (the
// retro-debt aging report).
func Main(args []string, s cli.Stdio) int {
	if len(args) > 0 && args[0] == "--age" {
		return age(args[1:], s)
	}
	if len(args) != 3 {
		fmt.Fprint(s.Err, usageText)
		return ExitEnv
	}
	state, retrodir, index := args[0], args[1], args[2]
	switch {
	case !gen.IsRegular(state):
		fmt.Fprintf(s.Err, "sweep: no BUILD-STATE file: %s\n", state)
		return ExitEnv
	case !gen.IsDir(retrodir):
		fmt.Fprintf(s.Err, "sweep: no retros dir: %s\n", retrodir)
		return ExitEnv
	case !gen.IsRegular(index):
		fmt.Fprintf(s.Err, "sweep: no INDEX file: %s\n", index)
		return ExitEnv
	}
	if msg, code := content(state, retrodir, index, s); code != ExitOK {
		if msg != "" {
			fmt.Fprintf(s.Err, "sweep: %s\n", msg)
		}
		return code
	}
	return ExitOK
}

// content runs the four checks in order and returns the first failure.
func content(state, retrodir, index string, s cli.Stdio) (string, int) {
	// 1. envelope well-formedness
	text, err := os.ReadFile(state)
	if err != nil {
		return fmt.Sprintf("cannot read BUILD-STATE file: %s: %v", state, err), ExitEnv
	}
	if msg := CheckEnvelopes(string(text)); msg != "" {
		return msg, ExitFail
	}

	// 2. INDEX integrity
	indexText, err := os.ReadFile(index)
	if err != nil {
		return fmt.Sprintf("cannot read INDEX file: %s: %v", index, err), ExitEnv
	}
	rows, msg, code := indexRows(string(indexText), retrodir, s)
	if code != ExitOK {
		return msg, code
	}
	if msg, code := unindexedRetro(retrodir, rows); code != ExitOK {
		return msg, code
	}
	return freshness(state, retrodir, index, string(indexText), s)
}

// indexRows checks every INDEX row against its retro (present, marker in domain and equal to the
// row's status) and returns the set of indexed retro files.
func indexRows(indexText, retrodir string, s cli.Stdio) (map[string]bool, string, int) {
	rows := map[string]bool{}
	for _, line := range Lines(indexText) {
		r, ok := ParseIndexRow(line)
		if !ok {
			continue
		}
		if r.Status == "" {
			return nil, fmt.Sprintf("INDEX row for %s has no valid review-status (pending|folded)", r.File), ExitFail
		}
		path := retrodir + "/" + r.File
		if !gen.IsRegular(path) {
			return nil, "INDEX row points at a missing retro: " + r.File, ExitFail
		}
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(s.Err, "sweep: cannot read %s: %v\n", path, err) // no marker read: tolerated, as before
		}
		if m := Marker(string(b)); m != "" {
			if m != "pending" && m != "folded" {
				return nil, fmt.Sprintf("retro marker out of domain: %s has review-status '%s' (pending|folded)", r.File, m), ExitFail
			}
			if m != r.Status {
				return nil, fmt.Sprintf("retro marker disagrees with INDEX row: %s marker='%s' INDEX='%s'", r.File, m, r.Status), ExitFail
			}
		}
		rows[r.File] = true
	}
	return rows, "", ExitOK
}

// unindexedRetro names the first retro (sorted, INDEX.md and dot-files excluded) with no INDEX row.
func unindexedRetro(retrodir string, rows map[string]bool) (string, int) {
	entries, err := os.ReadDir(retrodir)
	if err != nil {
		return fmt.Sprintf("cannot list retros dir: %s: %v", retrodir, err), ExitEnv
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, ".md") && !strings.HasPrefix(n, ".") && n != "INDEX.md" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		if !rows[n] {
			return "retro has no INDEX row: " + n, ExitFail
		}
	}
	return "", ExitOK
}

// freshness checks the generated INDEX and the generated kit ledger, when present, in process. A
// GENERATED INDEX (line 1 = gen.GeneratedLine) is compared with what the retro headers of the kit at
// <retros-dir>/.. generate now, wherever the file lives, so a caller may pass kit-state output from a
// temp dir; a hand-written index (test fixtures) skips the check.
func freshness(state, retrodir, index, indexText string, s cli.Stdio) (string, int) {
	// 3. generated INDEX freshness. A generator's diagnostics are sweep output; its result is compared here.
	// Mutation: GRI-sweep -- skipping the freshness call lets a deltas/module header drift pass the gate
	if strings.HasPrefix(indexText, gen.GeneratedLine) {
		want, code := retroindex.Generate(abs(retrodir+"/.."), s.Err)
		if code != ExitOK {
			return "the retro headers under " + retrodir + " do not generate an INDEX", ExitFail
		}
		if strings.TrimRight(indexText, "\n") != strings.TrimRight(want, "\n") {
			return index + " is stale vs the retro headers; run toolbelt/gen-retro-index.sh", ExitFail
		}
	}

	// 4. generated kit ledger freshness, checked in-process by buildstate.Check.
	// Mutation: GBS-sweep -- skipping this call lets a fragment land without its regenerated BUILD-STATE.md
	if stateroot := abs(filepath.Dir(state)); gen.IsDir(stateroot + "/ledger.d") {
		if filepath.Base(state) != "BUILD-STATE.md" {
			return "a ledger.d/ kit needs the state file named BUILD-STATE.md: " + state, ExitFail
		}
		if !buildstate.Check(stateroot, s.Err) {
			return "BUILD-STATE.md is stale vs ledger.d/ (or a fragment is malformed); run toolbelt/gen-build-state.sh", ExitFail
		}
	}
	return "", ExitOK
}

// abs is the logical absolute path, as `cd <dir> && pwd` printed it (no symlink resolution).
func abs(path string) string {
	a, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return a
}

// age implements the --age report: total pending rows (dispositioned 0-delta rows excluded), how
// many are older than --max-age days, and the oldest age, from each row's filename date.
func age(args []string, s cli.Stdio) int {
	today, maxAge, args, ok := parseAgeFlags(args)
	if !ok {
		fmt.Fprint(s.Err, ageUsageText)
		return ExitEnv
	}
	if today == "" {
		fmt.Fprint(s.Err, "sweep-build-state: --today <YYYY-MM-DD> is required\n")
		return ExitEnv
	}
	if len(args) != 2 {
		fmt.Fprint(s.Err, ageUsageText)
		return ExitEnv
	}
	retrodir, index := args[0], args[1]
	if !gen.IsDir(retrodir) {
		fmt.Fprintf(s.Err, "sweep-build-state: no retros dir: %s\n", retrodir)
		return ExitEnv
	}
	if !gen.IsRegular(index) {
		fmt.Fprintf(s.Err, "sweep-build-state: no INDEX file: %s\n", index)
		return ExitEnv
	}
	day, ok := parseDay(today)
	if !ok {
		fmt.Fprintf(s.Err, "sweep-build-state: --today is not a YYYY-MM-DD date: %s\n", today)
		return ExitEnv
	}
	limit, err := strconv.Atoi(maxAge)
	if err != nil {
		fmt.Fprintf(s.Err, "sweep-build-state: --max-age is not a whole number: %s\n", maxAge)
		return ExitEnv
	}
	text, err := os.ReadFile(index)
	if err != nil {
		fmt.Fprintf(s.Err, "sweep-build-state: cannot read INDEX file: %s: %v\n", index, err)
		return ExitEnv
	}

	total, escalated, oldest := pendingAges(string(text), day, limit)
	fmt.Fprintf(s.Out, "total_pending=%d\nescalated_count=%d\n", total, escalated)
	if total == 0 {
		fmt.Fprint(s.Out, "oldest_age=N/A\n")
	} else {
		fmt.Fprintf(s.Out, "oldest_age=%d\n", oldest)
	}
	return ExitOK
}

// parseAgeFlags reads the --age flags (--today, --max-age, a closing --) and returns the operands
// after them; ok is false on an unknown flag or a flag missing its value.
func parseAgeFlags(args []string) (today, maxAge string, rest []string, ok bool) {
	maxAge = strconv.Itoa(DefaultMaxAge)
	for len(args) > 0 {
		switch a := args[0]; {
		case a == "--today" || a == "--max-age":
			if len(args) < 2 {
				return "", "", nil, false
			}
			if a == "--today" {
				today = args[1]
			} else {
				maxAge = args[1]
			}
			args = args[2:]
		case a == "--":
			return today, maxAge, args[1:], true
		case strings.HasPrefix(a, "-"):
			return "", "", nil, false
		default:
			return today, maxAge, args, true
		}
	}
	return today, maxAge, args, true
}

// pendingAges counts the INDEX's pending rows (dispositioned 0-delta rows excluded), how many were
// filed more than limit days before day, and the oldest age in days (-1 when no row has a date).
func pendingAges(indexText string, day time.Time, limit int) (total, escalated, oldest int) {
	oldest = -1
	for _, line := range Lines(indexText) {
		r, ok := ParseIndexRow(line)
		if !ok || r.Status != "pending" || DispositionSkips(line) {
			continue
		}
		total++
		filed, ok := parseDay(prefix(r.File, 10))
		if !ok {
			continue // counted as pending, but a name without a valid date prefix has no age
		}
		days := int(day.Sub(filed).Hours() / 24)
		if days > oldest {
			oldest = days
		}
		if days > limit {
			escalated++
		}
	}
	return total, escalated, oldest
}

// parseDay reads a YYYY-MM-DD calendar date (UTC midnight).
func parseDay(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

func prefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
