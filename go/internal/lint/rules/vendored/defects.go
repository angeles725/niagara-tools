package vendored

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"
)

// defectHeader is the header of the kit's known-defects table (build-n4-module-kit/data/vendored-defects.md).
var defectHeader = []string{"Library", "File", "Affected", "Defect", "Fix"}

// defect is one row of that table: a library's versions that carry a known defect, for the files whose base
// name matches File (a path.Match pattern; * for every file of the library).
type defect struct {
	lib, file, text, fix string
	cons                 []constraint
}

// loadDefects reads the known-defects table at file. Any fault in it is an error (exit 3, no row): a table
// that cannot be read whole must not read as "no known defects".
func loadDefects(file string) ([]defect, error) {
	if file == "" {
		return nil, errors.New("no known-defects table: pass --defects <file> (toolbelt/lint-vendored.sh does)")
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.New("known-defects table unreadable: " + err.Error())
	}
	rows, found := readTable(strings.Split(string(b), "\n"), defectHeader)
	if !found {
		return nil, errors.New("known-defects table " + file + " has no table with the header | " +
			strings.Join(defectHeader, " | ") + " |")
	}
	var out []defect
	for _, r := range rows {
		at := "known-defects table " + file + ":" + strconv.Itoa(r.line) + ": "
		c := r.cells
		if c[0] == "" || c[1] == "" || c[2] == "" || c[3] == "" {
			return nil, errors.New(at + "Library, File, Affected and Defect must not be empty")
		}
		if _, err := path.Match(c[1], ""); err != nil {
			return nil, errors.New(at + "File " + strconv.Quote(c[1]) + " is not a pattern: " + err.Error())
		}
		cons, err := parseAffected(c[2])
		if err != nil {
			return nil, errors.New(at + err.Error())
		}
		out = append(out, defect{lib: c[0], file: c[1], cons: cons, text: c[3], fix: c[4]})
	}
	return out, nil
}

// applies: the defect is about this library and this file.
func (d defect) applies(lib, file string) bool {
	ok, _ := path.Match(d.file, path.Base(file))
	return ok && strings.EqualFold(d.lib, lib)
}
