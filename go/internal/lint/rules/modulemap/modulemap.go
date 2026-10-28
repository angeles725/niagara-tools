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
//	MM5  a non-plumbing row names no existing srcTest/ path in Tests           FAIL strict, WARN legacy
//	MM6  a part's srcTest/**/*.java is in no row                               WARN
//	MM7  a concern folder (the directory right under a role folder: rt model/ or
//	     adapter/; ux route/, service/ or model/; wb and se model/) has files in
//	     two rows, or a non-plumbing row lists files in concern folders of two
//	     names (model/x and adapter/x are one concern)                          FAIL strict, WARN legacy
package modulemap

import (
	"errors"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-module-map"

// Spec registers the rule with the contract of toolbelt/lint-module-map.sh: one <module-root> (later arguments
// ignored, no flag parsed); rows "<sev>  lint-module-map  <path>[:<line>]  MM<n>: <reason>"; exit 0 clean or
// WARN-only, 1 on any FAIL, 3 on a usage error, a root with no part, an unreadable directory under a part or an
// unreadable map file.
func Spec() lint.Spec {
	return lint.Spec{Root: rule{}, Mode: lint.Raw, Usage: "usage: lint-module-map.sh <module-root>", BarePathRow: true,
		EmptyError: lint.EmptyRow{Msg: "no module parts found (no *-rt, *-ux, *-wb or *-se sub-directory: wrong path or not a module root)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// header is the table header the map must carry, cell by cell.
var header = []string{"Concern", "Part", "Files", "Tests", "Change here when"}

// row is one concern row of the map.
type row struct {
	line                int
	concern, part, when string
	files, tests        []string
}

// scan is one run over a module root: the findings in report order and the module's files.
type scan struct {
	root, mapPath string
	mode          lint.Modularity
	files         []string // every file under the parts, relative to root, in walk order
	out           []lint.Finding
}

// CheckRoot runs MM1-MM6 on root: the per-row checks (MM3, then MM5) in map order, then MM4 and MM6 by path.
func (rule) CheckRoot(root string) ([]lint.Finding, error) {
	s := &scan{root: strings.TrimRight(root, "/")}
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
	owners := map[string][]string{} // source file -> the concerns whose Files list it
	listed := map[string]bool{}     // every exact entry of any cell
	for _, r := range rows {
		s.entries(r, r.files, owners, listed)
		tests := s.entries(r, r.tests, nil, listed)
		if !strings.EqualFold(r.concern, "plumbing") && tests == 0 {
			s.add(s.strictSev(), s.mapPath, r.line, "MM5: concern "+r.concern+" names no existing srcTest path in Tests")
		}
	}
	s.mm7(rows, owners)
	s.mm4(owners)
	s.mm6(listed)
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

// abs is rel under the root as given.
func (s *scan) abs(rel string) string {
	if s.root == "/" {
		return "/" + rel
	}
	return s.root + "/" + rel
}

// parts lists the root's -rt/-ux/-wb/-se sub-directories, in name order.
func (s *scan) parts() ([]string, error) {
	if !lint.IsDir(s.root) { // a wrong path: the empty-root ERROR row, never a clean pass
		return nil, nil
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, errors.New("cannot list " + s.root + ": " + err.Error())
	}
	var parts []string
	for _, e := range entries {
		n := e.Name()
		if !lint.IsDir(s.abs(n)) {
			continue
		}
		for _, suf := range []string{"-rt", "-ux", "-wb", "-se"} {
			if strings.HasSuffix(n, suf) && len(n) > len(suf) {
				parts = append(parts, n)
				break
			}
		}
	}
	return parts, nil
}

// walk adds every file under the part to s.files (relative to the root), skipping dot directories and the
// part's build/ output. A symlinked directory is not descended (its files are listed under their real path, and
// a link back up the tree cannot loop); a symlinked file is listed. An unreadable directory fails the scan closed.
func (s *scan) walk(part string) error {
	var visit func(rel string) error
	visit = func(rel string) error {
		entries, err := os.ReadDir(s.abs(rel))
		if err != nil {
			return errors.New("cannot list every file under " + s.abs(part) + ": " + err.Error())
		}
		for _, e := range entries {
			n, child := e.Name(), rel+"/"+e.Name()
			if strings.HasPrefix(n, ".") || child == part+"/build" {
				continue
			}
			if e.Type()&os.ModeSymlink != 0 && lint.IsDir(s.abs(child)) {
				continue
			}
			if lint.IsDir(s.abs(child)) {
				if err := visit(child); err != nil {
					return err
				}
			} else if lint.IsFile(s.abs(child)) {
				s.files = append(s.files, child)
			}
		}
		return nil
	}
	return visit(part)
}

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
		rows = append(rows, row{line: i + 1, concern: c[0], part: c[1], when: c[4], files: split(c[2]), tests: split(c[3])})
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

// entries checks one cell of r (MM3) and returns how many of its entries name an existing file under a srcTest/
// directory (MM5 counts them for the Tests cell). Exact entries go into listed; with owners set (the Files
// cell), every source file an entry lists records r's concern, once per row.
func (s *scan) entries(r row, cell []string, owners map[string][]string, listed map[string]bool) int {
	tests := 0
	mine := map[string]bool{}
	own := func(f string) {
		if owners != nil && !mine[f] {
			mine[f] = true
			owners[f] = append(owners[f], r.concern)
		}
	}
	for _, e := range cell {
		where := e + " (concern " + r.concern + ")"
		clean := path.Clean(e)
		if path.IsAbs(e) || clean == ".." || strings.HasPrefix(clean, "../") {
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" is not a relative path inside the module root")
			continue
		}
		if strings.ContainsAny(e, "*?[") {
			n := 0
			for _, f := range s.files {
				if !strings.HasSuffix(f, ".java") && matchGlob(clean, f) {
					n++
					own(f)
					if inSrcTest(f) {
						tests++
					}
				}
			}
			if n == 0 {
				s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" matches no file (a glob never lists a .java file)")
			}
			continue
		}
		switch {
		case lint.IsFile(s.abs(clean)):
			listed[clean] = true
			own(clean)
			if inSrcTest(clean) {
				tests++
			}
		case lint.IsDir(s.abs(clean)):
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" is a directory: list its files, or use a src/rc glob")
		default:
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" does not exist")
		}
	}
	return tests
}

// inSrcTest: rel lies under a srcTest/ directory of a part.
func inSrcTest(rel string) bool { return strings.Contains("/"+rel, "/srcTest/") }

// mm4 checks every source file of the parts against the concerns that list it, in path order.
func (s *scan) mm4(owners map[string][]string) {
	var srcs []string
	for _, f := range s.files {
		if isSource(f) {
			srcs = append(srcs, f)
		}
	}
	sort.Strings(srcs)
	for _, f := range srcs {
		switch o := owners[f]; {
		case len(o) == 0:
			s.add(s.strictSev(), s.abs(f), 0, "MM4: in no concern (list it in the Files of the concern that owns it)")
		case len(o) > 1:
			s.add(lint.Fail, s.abs(f), 0, "MM4: in "+strconv.Itoa(len(o))+" concerns ("+strings.Join(o, ", ")+"); list it in one")
		}
	}
}

// isSource: a part's src/**/*.java, or a src/rc/**/*.{js,html,css} outside vendor/ and ext/ and not *.min.*.
func isSource(rel string) bool {
	seg := strings.Split(rel, "/")
	if len(seg) < 3 || seg[1] != "src" {
		return false
	}
	name := seg[len(seg)-1]
	if strings.HasSuffix(name, ".java") {
		return true
	}
	if seg[2] != "rc" || strings.Contains(name, ".min.") {
		return false
	}
	for _, d := range seg[3 : len(seg)-1] {
		if d == "vendor" || d == "ext" {
			return false
		}
	}
	switch path.Ext(name) {
	case ".js", ".html", ".css":
		return true
	}
	return false
}

// roleFolders are the fixed role folders of each part suffix (types/structure.md § Module map); the concern
// folders under them carry any per-module name (modularity-structure Δ2).
var roleFolders = map[string][]string{
	"-rt": {"model", "adapter"},
	"-ux": {"route", "service", "model"},
	"-wb": {"model"},
	"-se": {"model"},
}

// ConcernFolder is the concern folder of a part's src/**/*.java file rel (relative to the module root): the
// path up to the directory right under the first role-folder segment after src/, or "" for a file that sits
// in no concern folder (directly in a role folder, or outside every role folder).
func ConcernFolder(rel string) string {
	seg := strings.Split(rel, "/")
	if len(seg) < 4 || seg[1] != "src" || !strings.HasSuffix(rel, ".java") {
		return ""
	}
	var roles []string
	for suf, r := range roleFolders {
		if strings.HasSuffix(seg[0], suf) {
			roles = r
		}
	}
	for i := 2; i < len(seg)-2; i++ {
		for _, r := range roles {
			if seg[i] == r {
				return strings.Join(seg[:i+2], "/")
			}
		}
	}
	return ""
}

// mm7 relates concern folders and rows: per non-plumbing row in map order, a row whose Files lie in concern
// folders of two or more names (one concern may span role folders: rt model/battery and adapter/battery); then per concern folder in path order, a folder whose files two rows list.
func (s *scan) mm7(rows []row, owners map[string][]string) {
	byRow := map[string]map[string]bool{}    // concern -> its concern folders
	byFolder := map[string]map[string]bool{} // concern folder -> the concerns listing its files
	for f, concerns := range owners {
		cf := ConcernFolder(f)
		if cf == "" || !isSource(f) {
			continue
		}
		for _, c := range concerns {
			if byRow[c] == nil {
				byRow[c] = map[string]bool{}
			}
			byRow[c][cf] = true
			if byFolder[cf] == nil {
				byFolder[cf] = map[string]bool{}
			}
			byFolder[cf][c] = true
		}
	}
	for _, r := range rows {
		names := map[string]bool{} // the folders' own names: model/battery and adapter/battery are one concern
		for cf := range byRow[r.concern] {
			names[path.Base(cf)] = true
		}
		if strings.EqualFold(r.concern, "plumbing") || len(names) < 2 {
			continue
		}
		fs := keys(byRow[r.concern])
		s.add(s.strictSev(), s.mapPath, r.line, "MM7: concern "+r.concern+" lists files in "+strconv.Itoa(len(fs))+
			" concern folders ("+strings.Join(fs, ", ")+"); keep one concern folder per row")
	}
	for _, cf := range keys(byFolder) {
		if cs := keys(byFolder[cf]); len(cs) > 1 {
			s.add(s.strictSev(), s.abs(cf), 0, "MM7: concern folder in "+strconv.Itoa(len(cs))+" rows ("+
				strings.Join(cs, ", ")+"); one row owns a concern folder")
		}
	}
}

// keys is m's keys in sorted order.
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// mm6 WARNs every srcTest Java file no row lists by its path, in path order.
func (s *scan) mm6(listed map[string]bool) {
	var tests []string
	for _, f := range s.files {
		if inSrcTest(f) && strings.HasSuffix(f, ".java") && !listed[f] {
			tests = append(tests, f)
		}
	}
	sort.Strings(tests)
	for _, f := range tests {
		s.add(lint.Warn, s.abs(f), 0, "MM6: test in no concern (list it in the Tests of the concern it covers)")
	}
}

// matchGlob matches rel against pat segment by segment: "**" spans any number of segments (none included),
// every other segment is a path.Match pattern (so * and ? stay inside one segment). A malformed pattern
// matches nothing.
func matchGlob(pat, rel string) bool {
	return matchSegs(strings.Split(pat, "/"), strings.Split(rel, "/"))
}

func matchSegs(p, r []string) bool {
	if len(p) == 0 {
		return len(r) == 0
	}
	if p[0] == "**" {
		for i := 0; i <= len(r); i++ {
			if matchSegs(p[1:], r[i:]) {
				return true
			}
		}
		return false
	}
	if len(r) == 0 {
		return false
	}
	ok, err := path.Match(p[0], r[0])
	return err == nil && ok && matchSegs(p[1:], r[1:])
}

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
	owners := map[string][]string{}
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

// walked is a scan of root with every part walked: lint.ErrEmptyRoot with no part, an error for an unreadable
// part directory.
func walked(root string) (*scan, error) {
	s := &scan{root: strings.TrimRight(root, "/")}
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
	for _, p := range parts {
		if err := s.walk(p); err != nil {
			return nil, err
		}
	}
	return s, nil
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

// Row is one concern row of a map as the module-map tool reads it (modularity-structure Δ1, Δ4).
type Row struct {
	Line                int // 1-based line of the row in MODULE-MAP.md
	Concern, Part, When string
	Files, Tests        []string // the cell entries as written (globs unexpanded)
}

// Map is a module root's parsed MODULE-MAP.md: its mode, its rows and its lines (for a writer that inserts a
// row after Rows' last line).
type Map struct {
	Mode  lint.Modularity
	Rows  []Row
	Lines []string
}

// ReadMap parses root's MODULE-MAP.md with the lint's own header and table reader. A missing or unreadable map is
// an error; an invalid header or table is an error naming its MM2 problem and line.
func ReadMap(root string) (Map, error) {
	p := strings.TrimRight(root, "/") + "/" + lint.MapFile
	b, err := os.ReadFile(p)
	if err != nil {
		return Map{}, errors.New("cannot read " + p + ": " + err.Error())
	}
	lines := strings.Split(string(b), "\n")
	mode, at, problem := lint.MapHeader(lines)
	if problem == "" {
		var rows []row
		rows, at, problem = table(lines)
		if problem == "" {
			m := Map{Mode: mode, Lines: lines}
			for _, r := range rows {
				m.Rows = append(m.Rows, Row{Line: r.line, Concern: r.concern, Part: r.part, When: r.when, Files: r.files, Tests: r.tests})
			}
			return m, nil
		}
	}
	return Map{}, errors.New(p + ":" + strconv.Itoa(at) + ": MM2: " + problem)
}

// MatchGlob reports whether the map entry pat (a "**"/"*" glob of the Files or Tests cell) matches rel.
func MatchGlob(pat, rel string) bool { return matchGlob(path.Clean(pat), rel) }
