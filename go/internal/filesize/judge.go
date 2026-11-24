package filesize

import "fmt"

// Message formats. Each carries the exact line to add, change or delete. A file that moves keeps its line: the path
// changes, the count and the kind stay, and that is not a new entry. Every advice to split says "split it by
// responsibility": a file is split along what it does, never only to get under the number.
const (
	msgUnlisted     = "%s: %d lines, over %d and not listed: split it by responsibility, or add this line to %s:\n%s %d accepted <reason>"
	msgUnlistedHard = "%s: %d lines, over %d: a file over %d lines cannot be listed, split it by responsibility. If it is a listed file that moved, change the path on its line instead (same count and kind): that is not a new entry."
	msgGrew         = "%s: grew from %d to %d lines: split it by responsibility, or raise its line in %s to:\n%s %d %s %s"
	msgGrewHard     = "%s: grew from %d to %d lines and is over %d: a file over %d lines may not grow, split it by responsibility first."
	msgNotOver      = "%s: %d lines, no longer over %d: delete this line from %s:\n%s"
	msgNotProd      = "%s: listed but not a production Go file in this tree. Delete this line from %s, or, if the file moved, change the path on it (same count and kind, list kept sorted):\n%s"
	msgShrunkHard   = "%s: %d lines, below its recorded %d: a count over %d is exact, so lower its line in %s to:\n%s %d %s %s"
	msgAcceptedHigh = "%s:%d: an accepted entry may not be above %d lines (%s %d): split it by responsibility."
	msgMalformed    = "%s:%d: %s; expected \"<path> <lines> <legacy|accepted> <reason of at least %d characters>\", entries sorted by path"
)

// judgeEntry returns the violation for one listed entry whose file measures n lines (isProd false: not a production
// file of the tree), or "" when it passes. The order matters: a line to delete or to lower is said before "split
// it", and a line is printed only when the next run admits it.
func judgeEntry(e debtEntry, n int, isProd bool) string {
	if !isProd {
		return fmt.Sprintf(msgNotProd, e.Path, DebtFile, e.Raw)
	}
	if n <= SoftLimit {
		return fmt.Sprintf(msgNotOver, e.Path, n, SoftLimit, DebtFile, e.Raw)
	}
	// An accepted entry above the hard limit whose file is still above it has no admissible line, lowered or not.
	if e.Kind == "accepted" && e.Lines > HardLimit && n > HardLimit {
		return fmt.Sprintf(msgAcceptedHigh, DebtFile, e.LineNo, HardLimit, e.Path, e.Lines)
	}
	// A count over the hard limit is exact: with slack, a file that shrank could grow back to its old count unseen.
	if e.Lines > HardLimit && n < e.Lines {
		return fmt.Sprintf(msgShrunkHard, e.Path, n, e.Lines, HardLimit, DebtFile, e.Path, n, e.Kind, e.Reason)
	}
	if n <= e.Lines {
		return ""
	}
	if n > HardLimit {
		return fmt.Sprintf(msgGrewHard, e.Path, e.Lines, n, HardLimit, HardLimit)
	}
	return fmt.Sprintf(msgGrew, e.Path, e.Lines, n, DebtFile, e.Path, n, e.Kind, e.Reason)
}

// judge compares measured sizes with the debt entries and returns the violation messages. A path in malformed has a
// malformed line, already reported: its file is listed, badly, and is not reported again as unlisted.
func judge(files map[string]int, entries []debtEntry, malformed map[string]bool) []string {
	var out []string
	listed := map[string]bool{}
	for _, e := range entries {
		listed[e.Path] = true
		n, isProd := files[e.Path]
		if msg := judgeEntry(e, n, isProd); msg != "" {
			out = append(out, msg)
		}
	}
	for path, n := range files {
		if n <= SoftLimit {
			continue
		}
		if listed[path] || malformed[path] {
			continue
		}
		if n > HardLimit {
			out = append(out, fmt.Sprintf(msgUnlistedHard, path, n, HardLimit, HardLimit))
		} else {
			out = append(out, fmt.Sprintf(msgUnlisted, path, n, SoftLimit, DebtFile, path, n))
		}
	}
	return out
}
