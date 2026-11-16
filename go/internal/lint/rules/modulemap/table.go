package modulemap

import (
	"strconv"
	"strings"
)

// table finds the first table whose header is the five map columns and reads its rows. It returns the rows,
// or the line and the MM2 problem of a map with no such table, a row of the wrong width or no row at all.
func table(lines []string) ([]row, int, string) {
	start, fenced := -1, false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			fenced = !fenced
			continue
		}
		if !fenced && strings.HasPrefix(t, "|") && equalCells(cells(t), header) {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, 0, "no | " + strings.Join(header, " | ") + " | table"
	}
	var rows []row
	for i := start + 1; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(t, "|") {
			break
		}
		c := cells(t)
		if i == start+1 && isSeparator(c) {
			continue
		}
		if len(c) != len(header) {
			return nil, i + 1, "row has " + strconv.Itoa(len(c)) + " cells, want " + strconv.Itoa(len(header))
		}
		if c[0] == "" {
			return nil, i + 1, "row has an empty Concern cell"
		}
		rows = append(rows, row{idx: len(rows), line: i + 1, concern: c[0], part: c[1], when: c[4], files: split(c[2]), tests: split(c[3])})
	}
	if len(rows) == 0 {
		return nil, start + 1, "the table has no concern row"
	}
	return rows, 0, ""
}

// cells splits a table line on "|" (the outer pipes dropped) and trims each cell.
func cells(t string) []string {
	t = strings.TrimSuffix(strings.TrimPrefix(t, "|"), "|")
	c := strings.Split(t, "|")
	for i := range c {
		c[i] = strings.TrimSpace(c[i])
	}
	return c
}

func equalCells(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

func isSeparator(c []string) bool {
	for _, x := range c {
		if strings.Trim(x, ":-") != "" || x == "" {
			return false
		}
	}
	return true
}

// split reads a Files or Tests cell: ";"-separated entries, backticks and spaces trimmed; "—", "–", "-" and an
// empty entry mean none.
func split(cell string) []string {
	var out []string
	for _, e := range strings.Split(cell, ";") {
		e = strings.TrimSpace(strings.Trim(strings.TrimSpace(e), "`"))
		switch e {
		case "", "—", "–", "-":
			continue
		}
		out = append(out, e)
	}
	return out
}
