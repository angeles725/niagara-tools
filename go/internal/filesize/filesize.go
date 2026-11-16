// Package filesize keeps Go source files small: a non-test file may hold at most Limit lines unless it is on the
// recorded list of files that were already larger when the rule arrived, and a listed file may not grow.
// The list only shrinks: split a listed file, then lower or delete its line.
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

// Limit is the number of lines a non-test Go file may hold without being on the baseline.
const Limit = 200

// Finding is one file that breaks the rule, with the reason.
type Finding struct {
	Path   string
	Lines  int
	Reason string
}

// Check walks root for non-test Go files and compares each against the limit and the baseline (path -> lines).
func Check(root string, baseline map[string]int) ([]Finding, error) {
	sizes, err := Sizes(root)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for path, n := range sizes {
		allowed, listed := baseline[path]
		switch {
		case !listed && n > Limit:
			out = append(out, Finding{path, n, fmt.Sprintf("holds %d lines, over the limit of %d: split it by responsibility", n, Limit)})
		case listed && n > allowed:
			out = append(out, Finding{path, n, fmt.Sprintf("grew from its recorded %d lines to %d: split it before adding to it", allowed, n)})
		case listed && n <= Limit:
			out = append(out, Finding{path, n, fmt.Sprintf("is now within the limit (%d lines): delete its baseline line", n)})
		}
	}
	for path := range baseline {
		if _, ok := sizes[path]; !ok {
			out = append(out, Finding{path, 0, "is on the baseline but no longer exists: delete its baseline line"})
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
