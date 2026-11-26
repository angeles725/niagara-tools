package vendored

import (
	"os"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ledgerHeader is the header of rc/vendor/VENDORED.md: one row per vendored file.
var ledgerHeader = []string{"File", "Library", "Version", "Source URL"}

// entry is one ledger row: the file (relative to rc/vendor/, forward slashes), its library, version and source.
type entry struct {
	line         int
	file, lib    string
	version, url string
	ver          version
	verOK        bool // version reads as an r-number or a semver
}

// readLedger reads the ledger at file: its entries by file (the first row of a file wins) and a WARN for
// every incomplete row. found is false when the file holds no ledger table.
func readLedger(file string) (entries map[string]entry, problems []lint.Finding, found bool, err error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, false, err
	}
	rows, found := readTable(strings.Split(string(b), "\n"), ledgerHeader)
	entries = map[string]entry{}
	for _, r := range rows {
		c := r.cells
		e := entry{line: r.line, file: strings.TrimPrefix(c[0], "./"), lib: c[1], version: c[2], url: c[3]}
		if empty := firstEmpty(c); empty >= 0 {
			problems = append(problems, lint.Finding{Path: file, Line: r.line, Msg: "row has an empty " + ledgerHeader[empty] + " cell"})
			if empty == 0 {
				continue // no file to list
			}
		}
		if e.ver, e.verOK = parseVersion(e.version); !e.verOK && e.version != "" {
			problems = append(problems, lint.Finding{Path: file, Line: r.line, Msg: "version " + strconv.Quote(e.version) +
				" is neither an r-number (r161) nor a semver (1.4.2)"})
		}
		if _, dup := entries[e.file]; !dup {
			entries[e.file] = e
		}
	}
	return entries, problems, found, nil
}

// firstEmpty is the index of the first empty cell, -1 when none.
func firstEmpty(cells []string) int {
	for i, c := range cells {
		if c == "" {
			return i
		}
	}
	return -1
}
