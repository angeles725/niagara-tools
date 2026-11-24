// Package filesize keeps Go source files small with a band: a non-test file over WarnLimit lines is a warning (reported,
// never failing), and one over FailLimit lines fails unless it is on the recorded list of files that were already
// larger when the rule arrived. A listed file may not grow and gets no warning.
// The list only shrinks: split a listed file, then lower or delete its line. Split by responsibility, never only to
// get under the number.
package filesize

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	// WarnLimit is the number of lines above which a non-test Go file outside the baseline is reported as a warning.
	WarnLimit = 200
	// FailLimit is the number of lines above which a non-test Go file outside the baseline fails the gate.
	FailLimit = 300
)

// Finding is one file that breaks the rule, with the reason.
type Finding struct {
	Path   string
	Lines  int
	Reason string
	// Warn marks a finding that is reported but never fails the gate.
	Warn bool
}

// Check walks root for non-test Go files and compares each against the limits and the baseline (path -> lines).
func Check(root string, baseline map[string]int) ([]Finding, error) {
	sizes, err := Sizes(root)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for path, n := range sizes {
		allowed, listed := baseline[path]
		switch {
		case !listed && n > FailLimit:
			out = append(out, Finding{path, n, fmt.Sprintf("holds %d lines, over the limit of %d: split it by responsibility", n, FailLimit), false})
		case !listed && n > WarnLimit:
			out = append(out, Finding{path, n, fmt.Sprintf("holds %d lines, over %d (it fails over %d): split by responsibility, never only to get under the number", n, WarnLimit, FailLimit), true})
		case listed && n > allowed:
			out = append(out, Finding{path, n, fmt.Sprintf("grew from its recorded %d lines to %d: split it before adding to it", allowed, n), false})
		case listed && n <= WarnLimit:
			out = append(out, Finding{path, n, fmt.Sprintf("is now within the limit (%d lines): delete its baseline line", n), false})
		}
	}
	for path := range baseline {
		if _, ok := sizes[path]; !ok {
			out = append(out, Finding{path, 0, "is on the baseline but no longer exists: delete its baseline line", false})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Sizes returns the line count of every non-test Go file under root, keyed by its slash path relative to root.
func Sizes(root string) (map[string]int, error) {
	sizes := map[string]int{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "bin" || name == "testdata" || strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		n, err := countLines(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sizes[filepath.ToSlash(rel)] = n
		return nil
	})
	return sizes, err
}

// ReadBaseline parses "<lines> <path>" rows; blank lines and lines starting with # are ignored.
func ReadBaseline(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	baseline := map[string]int{}
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		n, convErr := strconv.Atoi(fields[0])
		if len(fields) != 2 || convErr != nil {
			return nil, fmt.Errorf("%s:%d: want \"<lines> <path>\", got %q", path, line, text)
		}
		baseline[fields[1]] = n
	}
	return baseline, sc.Err()
}

func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	n := strings.Count(string(data), "\n")
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n, nil
}
