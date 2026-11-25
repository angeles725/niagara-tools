package ledgercheck

import (
	"strings"
	"time"
)

// The Markdown table readers of ledger-check: the template and the client ledger are read with the same ones.

// tableAfter is the index of the first table line after the heading line h (-1 when there is none before the next
// heading).
func tableAfter(lines []string, h string) int {
	in := false
	for i, l := range lines {
		switch {
		case strings.TrimSpace(l) == h:
			in = true
		case in && strings.HasPrefix(l, "#"):
			return -1
		case in && strings.HasPrefix(strings.TrimSpace(l), "|"):
			return i
		}
	}
	return -1
}

// cells splits a Markdown table line into trimmed cells.
func cells(l string) []string {
	l = strings.TrimSpace(l)
	l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
	out := strings.Split(l, "|")
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

// separator reports a table's |---|---| line.
func separator(c []string) bool {
	for _, x := range c {
		if strings.Trim(x, "-: ") != "" {
			return false
		}
	}
	return true
}

// versionRows is the index of every table line of the Versions section, header included: a blank, comment or prose
// line between rows does not end the table, only the next heading does.
func versionRows(lines []string) []int {
	var out []int
	in := false
	for i, l := range lines {
		switch t := strings.TrimSpace(l); {
		case t == "## Versions":
			in = true
		case in && strings.HasPrefix(l, "#"):
			return out
		// Mutation: LC-blank -- stopping at the first non-row line leaves every row after a blank line unchecked
		case in && strings.HasPrefix(t, "|"):
			out = append(out, i)
		}
	}
	return out
}

func sameCells(a, b []string) bool {
	return len(b) > 0 && strings.Join(a, "|") == strings.Join(b, "|")
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// realDate: d (already YYYY-MM-DD shaped) names a calendar day.
func realDate(d string) bool {
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}
