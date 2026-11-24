// Package filesize keeps Go source files small with a debt list (the model research-sdd uses). A production Go file
// is one that is not a test, not under testdata, not under a path the go tool ignores (a name starting with "." or
// "_"), not under bin/ and not generated (a "// Code generated ... DO NOT EDIT." line before the package clause).
// A production file over SoftLimit (200) lines needs a line in modularity-debt.txt; a count of HardLimit (400) or
// less is a ceiling (the file may shrink freely and may grow up to 400 only with its line raised in the same change),
// a count over 400 is exact, and nothing new is admitted above 400. The list lives next to this package; its format,
// the kinds and the update mode are described in its header and in go/README.md. The messages are in judge.go, the
// list's parser in debt.go and the update mode in update.go. Split by responsibility, never only to get under the
// number.
package filesize

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	// SoftLimit is the number of lines above which a production file needs a debt entry.
	SoftLimit = 200
	// HardLimit is the number of lines above which a count is exact and no entry is admitted.
	HardLimit = 400
	// DebtFileName is the debt list, kept in this package's directory.
	DebtFileName = "modularity-debt.txt"
	// DebtFile is the list's path from the repository root, as the messages print it.
	DebtFile = "go/internal/filesize/" + DebtFileName
	// UpdateEnv turns the real-tree test into an updater when it is exactly "1".
	UpdateEnv = "NT_FILESIZE_UPDATE"
)

// Result reports what Check examined as well as what it found, so a zero cannot be mistaken for "did not look".
type Result struct {
	Scanned    int      // production files measured (generated files excluded)
	Generated  int      // generated files exempted
	Over       int      // production files over SoftLimit
	Entries    int      // well-formed debt entries read
	Violations []string // exact, actionable messages, sorted
}

var generatedLine = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// Check measures every production Go file under root against the limits and the debt list at debtPath. A missing
// debt file, an unreadable source file or a tree with no production file is an error, never a pass.
func Check(root, debtPath string) (Result, error) {
	var r Result
	files, generated, err := scanProduction(root)
	if err != nil {
		return r, err
	}
	data, err := os.ReadFile(debtPath)
	if err != nil {
		return r, fmt.Errorf("filesize: cannot read debt file: %w", err)
	}
	entries, bad, malformed := parseDebt(string(data))
	r.Scanned, r.Generated, r.Entries = len(files), generated, len(entries)
	r.Violations = append(bad, judge(files, entries, malformed, &r)...)
	sort.Strings(r.Violations)
	return r, nil
}

// scanProduction returns root-relative slash path -> line count for every production Go file, and how many
// generated files it exempted.
func scanProduction(root string) (map[string]int, int, error) {
	files := map[string]int{}
	generated := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("filesize: cannot read %s: %w", path, err)
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if goIgnores(d.Name()) || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("filesize: cannot read %s: %w", path, err)
		}
		if isGenerated(b) {
			generated++
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = countLines(b)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	if len(files) == 0 {
		return nil, 0, errors.New("filesize: no production Go files under " + root + " (an empty scan proves nothing)")
	}
	return files, generated, nil
}

// skipDir reports a directory the scan does not enter: testdata, bin (built tools) and what the go tool ignores.
// The root itself is always entered, whatever it is called.
func skipDir(name string) bool { return name == "testdata" || name == "bin" || goIgnores(name) }

// goIgnores reports a file or directory name the go tool ignores, one starting with "." or "_".
func goIgnores(name string) bool { return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") }

// countLines is the number of \n bytes, plus one for a non-empty file that does not end in \n.
func countLines(b []byte) int {
	n := bytes.Count(b, []byte{'\n'})
	if len(b) > 0 && b[len(b)-1] != '\n' {
		n++
	}
	return n
}

// isGenerated reports a "// Code generated ... DO NOT EDIT." line before the package clause.
func isGenerated(b []byte) bool {
	for _, line := range bytes.Split(b, []byte{'\n'}) {
		line = bytes.TrimRight(line, "\r")
		if generatedLine.Match(line) {
			return true
		}
		if bytes.HasPrefix(line, []byte("package ")) {
			return false
		}
	}
	return false
}
