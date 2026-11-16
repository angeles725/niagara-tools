package modulemap

import (
	"errors"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Summary is a module's modularity at a glance: report-module's modularity row (modularity-gaps Δ8).
type Summary struct {
	Mode     lint.Modularity
	Concerns int // rows of the map's table; -1 with no valid map
	Unlisted int // part source files (MM4's set) in no concern's Files; -1 with no valid map
}

// Summarize reads root's map and parts as CheckRoot does, without judging them. The mode is lint.ModuleMode's (the
// source the per-file rules pick their severity from: the nearest map at or above root, its header's mode even
// when its table is not valid). The counts are root's own map's (lint-module-map reads only that one): -1 with
// no map at root (nothing more is read), an invalid header or no valid table. With a map at root, a root with no
// part (lint-module-map's empty-root exit 3), an unreadable map or an unreadable part directory is an error.
func Summarize(root string) (Summary, error) {
	sum := Summary{Mode: lint.ModuleMode(root), Concerns: -1, Unlisted: -1}
	mapPath := strings.TrimRight(root, "/") + "/" + lint.MapFile
	if !lint.IsFile(mapPath) {
		return sum, nil
	}
	s, err := walked(root)
	if err != nil {
		return sum, err
	}
	s.mapPath = mapPath
	b, err := os.ReadFile(s.mapPath)
	if err != nil {
		return sum, errors.New("cannot read " + s.mapPath + ": " + err.Error())
	}
	lines := strings.Split(string(b), "\n")
	if _, _, problem := lint.MapHeader(lines); problem != "" {
		return sum, nil
	}
	rows, _, problem := table(lines)
	if problem != "" {
		return sum, nil
	}
	owners := map[string][]int{}
	for _, r := range rows {
		s.entries(r, r.files, owners, map[string]bool{})
	}
	unlisted := 0
	for _, f := range s.files {
		if isSource(f) && len(owners[f]) == 0 {
			unlisted++
		}
	}
	sum.Concerns, sum.Unlisted = len(rows), unlisted
	return sum, nil
}

// PartJava is every Java source of root's parts (<part>/src/**/*.java) as the lint walks them (a symlinked part or
// file in, a symlinked directory inside a part out), relative to root and sorted: report-module's largest class
// reads this set, so its largest and unlisted describe the same files.
func PartJava(root string) ([]string, error) {
	s, err := walked(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range s.files {
		if isSource(f) && strings.HasSuffix(f, ".java") {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out, nil
}
