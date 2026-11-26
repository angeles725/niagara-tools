package vendored

import (
	"strconv"
	"strings"
)

// tableRow is one body row of a markdown table: its 1-based line and its cells, trimmed, with a pair of
// surrounding backticks removed. A row shorter than the header is padded with empty cells.
type tableRow struct {
	line  int
	cells []string
}

// readTable finds the first table whose header cells equal header (case-insensitive) and returns its body
// rows; found is false when no line is such a header. The separator row is skipped; the table ends at the
// first line that does not start with "|".
func readTable(lines []string, header []string) (rows []tableRow, found bool) {
	for i := 0; i < len(lines); i++ {
		if !equalCells(splitRow(lines[i]), header) {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			cells := splitRow(lines[j])
			switch {
			case cells == nil:
				return rows, true
			case j == i+1 && isSeparator(cells):
			default:
				for len(cells) < len(header) {
					cells = append(cells, "")
				}
				rows = append(rows, tableRow{line: j + 1, cells: cells})
			}
		}
		return rows, true
	}
	return nil, false
}

// splitRow returns the cells of a "| a | b |" line, nil for a line that is not a table row.
func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	cells := strings.Split(line, "|")
	for i, c := range cells {
		c = strings.TrimSpace(c)
		if len(c) >= 2 && strings.HasPrefix(c, "`") && strings.HasSuffix(c, "`") {
			c = strings.TrimSpace(c[1 : len(c)-1])
		}
		cells[i] = c
	}
	return cells
}

func equalCells(cells, want []string) bool {
	if len(cells) != len(want) {
		return false
	}
	for i := range cells {
		if !strings.EqualFold(cells[i], want[i]) {
			return false
		}
	}
	return true
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "-: ") != "" || c == "" {
			return false
		}
	}
	return true
}

func itoa(n int) string { return strconv.Itoa(n) }
