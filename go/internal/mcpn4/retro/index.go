package retro

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// The rules the written retro follows (formerly mcp-n4-kit/tools/new_retro.py): which station
// and date are accepted, the file name, and its pending row in retros/INDEX.md. Pure: no I/O.

var (
	stationRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	dateRE    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

	errStation = errors.New("--station must match [A-Za-z0-9][A-Za-z0-9._-]{0,63}")
	errDate    = errors.New("--date must be a real calendar date, YYYY-MM-DD")
)

// CheckStation refuses a station name that is unsafe in a file name.
func CheckStation(station string) error {
	if !stationRE.MatchString(station) {
		return errStation
	}
	return nil
}

// CheckDate refuses anything but a real calendar date YYYY-MM-DD from year 1 on.
func CheckDate(date string) error {
	if !dateRE.MatchString(date) || strings.HasPrefix(date, "0000") {
		return errDate
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return errDate
	}
	return nil
}

// FileName is the retro's file name in the retros directory.
func FileName(date, station string) string {
	return date + "-" + station + "-session.md"
}

// IndexRow is the retro's pending row in INDEX.md.
func IndexRow(name, station, date string, deltas int) string {
	return fmt.Sprintf("| %s | %s | %s | pending | %d |", name, station, date, deltas)
}

// IndexRows is the INDEX text with row appended, or replacing the first row that names name
// (later rows naming it are dropped). Lines split as Python's str.splitlines does, and the
// result ends with one newline.
func IndexRows(index, name, row string) string {
	lines := splitLines(index)
	prefix := "| " + name + " |"
	out, replaced := make([]string, 0, len(lines)+1), false
	for _, ln := range lines {
		if !strings.HasPrefix(ln, prefix) {
			out = append(out, ln)
		} else if !replaced {
			out, replaced = append(out, row), true
		}
	}
	if !replaced {
		out = append(out, row)
	}
	return strings.Join(out, "\n") + "\n"
}

// splitLines is Python's str.splitlines(): every line boundary it knows, a trailing one
// not starting an empty last line, and "" giving no lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		switch rs[i] {
		case '\r':
			lines = append(lines, string(rs[start:i]))
			if i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
			start = i + 1
		case '\n', '\v', '\f', '\x1c', '\x1d', '\x1e', '\u0085', ' ', ' ':
			lines = append(lines, string(rs[start:i]))
			start = i + 1
		}
	}
	if start < len(rs) {
		lines = append(lines, string(rs[start:]))
	}
	return lines
}
