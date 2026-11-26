// Package modulemap is the lint-module-map rule: a module root's MODULE-MAP.md (types/structure.md § Module map)
// checked against the module tree. A lint.RootRule: it reads the map and walks the module's -rt/-ux/-wb/-se parts
// itself. The wrapper toolbelt/lint-module-map.sh keeps the user-facing contract and its header documents the
// checks and their mutation pins.
//
//	MM1  no MODULE-MAP.md at the root                                          WARN (nothing else runs)
//	MM2  no valid modularity: header, or no five-column concern table          FAIL (nothing else runs)
//	MM3  a listed path does not exist, is a directory, is absolute or leaves the root,
//	     or a glob matches no file                                             FAIL
//	MM4  a part's src/**/*.java or src/rc/**/*.{js,html,css} (minus vendor/,
//	     ext/, *.min.*) is in no row's Files (FAIL strict, WARN legacy)
//	     or in two rows' Files (FAIL)
//	MM5  a non-plumbing row names no existing srcTest/ path in Tests (a row of
//	     src/rc/ files also counts a webtest/**/*.test.mjs path)               FAIL strict, WARN legacy
//	MM6  a part's srcTest/**/*.java is in no row                               WARN
//	MM7  a concern folder (the directory right under a role folder: rt model/ or
//	     adapter/; ux route/, service/ or model/; wb and se model/) has files in
//	     two rows, or a non-plumbing row lists files in concern folders of two
//	     names (model/x and adapter/x are one concern), or two non-plumbing rows
//	     share a concern and a part (case-insensitive; one row per part is fine)  FAIL strict, WARN legacy
//	MM8  a file the candidate added since --base <ref> that no row lists, among the files MM4 does not
//	     require: srcTest/**/*.java, webtest/**/*.test.{mjs,js}, vendored src/rc files (vendor/, ext/, *.min.*);
//	     a new source file in no row stays MM4's. Needs --base (no --base: it does not run; a ref that does
//	     not resolve: one INFO row)                                            FAIL strict, WARN legacy
package modulemap

import (
	"errors"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-module-map"

// Spec registers the rule with the contract of toolbelt/lint-module-map.sh: [--base <ref>] <module-root> (--base is the
// only option; a second root or another flag is the usage line, exit 3); rows "<sev>  lint-module-map  <path>[:<line>]  MM<n>: <reason>"; exit 0 clean or
// WARN-only, 1 on any FAIL, 3 on a usage error, a root with no part, an unreadable directory under a part or an
// unreadable map file.
func Spec() lint.Spec {
	return lint.Spec{Root: rule{}, Mode: lint.Valued, Option: baseOption,
		Usage: "usage: lint-module-map.sh [--base <ref>] <module-root>", BarePathRow: true,
		EmptyError: lint.EmptyRow{Msg: "no module parts found (no *-rt, *-ux, *-wb or *-se sub-directory: wrong path or not a module root)"}}
}

type rule struct{ base string } // base: the --base ref of this run, "" when not given

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// row is one concern row of the map.
type row struct {
	idx, line           int // idx: the row's position in the table (the key owners records)
	concern, part, when string
	files, tests        []string
}

// scan is one run over a module root: the findings in report order and the module's files.
type scan struct {
	root, mapPath string
	base          string // the ref MM8 compares against; "" skips MM8
	mode          lint.Modularity
	files         []string // every file under the parts, relative to root, in walk order
	out           []lint.Finding
}

// CheckRoot runs MM1-MM8 on root: the per-row checks (MM3, then MM5) in map order, then MM7, MM4, MM6 and MM8.
func (r rule) CheckRoot(root string) ([]lint.Finding, error) {
	s := &scan{root: strings.TrimRight(root, "/"), base: r.base}
	if s.root == "" {
		s.root = "/"
	}
	parts, err := s.parts()
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	s.mapPath = s.abs(lint.MapFile)
	if !lint.IsFile(s.mapPath) {
		return []lint.Finding{{Severity: lint.Warn, Path: root,
			Msg: "MM1: no MODULE-MAP.md (a legacy module: add one with modularity: legacy and legacy-reason:, types/structure.md § Module map)"}}, nil
	}
	b, err := os.ReadFile(s.mapPath)
	if err != nil {
		return nil, errors.New("cannot read " + s.mapPath + ": " + err.Error())
	}
	lines := strings.Split(string(b), "\n")
	mode, at, problem := lint.MapHeader(lines)
	if problem != "" {
		return s.mm2(at, problem), nil
	}
	s.mode = mode
	rows, at, problem := table(lines)
	if problem != "" {
		return s.mm2(at, problem), nil
	}
	for _, p := range parts {
		if err := s.walk(p); err != nil {
			return nil, err
		}
	}
	owners := map[string][]int{} // source file -> the rows (idx) whose Files list it
	listed := map[string]bool{}  // every exact entry of any cell
	for _, r := range rows {
		s.entries(r, r.files, owners, listed)
		tests, spa := s.entries(r, r.tests, nil, listed)
		if underRC(r.files) { // an SPA row's test lives in webtest/, not srcTest/
			tests += spa
		}
		if !strings.EqualFold(r.concern, "plumbing") && tests == 0 {
			s.add(s.strictSev(), s.mapPath, r.line, "MM5: concern "+r.concern+" names no existing srcTest path in Tests")
		}
	}
	s.mm7(rows, owners)
	s.mm4(rows, owners)
	s.mm6(listed)
	s.mm8(rows)
	return s.out, nil
}

// mm2 is the lone MM2 row of a map whose header or table is not valid.
func (s *scan) mm2(line int, problem string) []lint.Finding {
	return []lint.Finding{{Path: s.mapPath, Line: line, Msg: "MM2: " + problem}}
}

// strictSev is FAIL in a strict module and WARN in a legacy one.
func (s *scan) strictSev() lint.Severity {
	if s.mode == lint.ModularityStrict {
		return lint.Fail
	}
	return lint.Warn
}

func (s *scan) add(sev lint.Severity, p string, line int, msg string) {
	s.out = append(s.out, lint.Finding{Severity: sev, Path: p, Line: line, Msg: msg})
}
