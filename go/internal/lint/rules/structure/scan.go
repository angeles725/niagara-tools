package structure

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// scan is one run over a module root: the rows in report order and the first thing it could not read.
type scan struct {
	root     string
	mode     lint.Modularity // the module map's mode: strict turns on strict L3/L3b
	packages map[string]bool // strict only: every package the module's src/ declares (L3b)
	types    map[string]bool // every type name the module's module-include.xml files declare (L17)
	out      []lint.Finding
	err      error
}

// row adds a finding; sev Unset takes the rule's FAIL.
func (s *scan) row(sev lint.Severity, path, msg string) {
	s.out = append(s.out, lint.Finding{Severity: sev, Path: path, Msg: msg})
}

// lineRows adds a finding at every line of p's lines that re matches (path:line rows, one per offending line).
func (s *scan) lineRows(p string, lines []string, re *regexp.Regexp, msg string) {
	for i, ln := range lines {
		if re.MatchString(ln) {
			s.out = append(s.out, lint.Finding{Path: p, Line: i + 1, Msg: msg})
		}
	}
}

// fail keeps the first error: the run still reports every row it can, then exits 3.
func (s *scan) fail(err error) {
	if s.err == nil {
		s.err = err
	}
}

// rel is p relative to the module root, as ${p#"$MODULE_ROOT/"} spells it.
func (s *scan) rel(p string) string { return strings.TrimPrefix(p, s.root+"/") }

// lines reads p as grep reads it (lines split on "\n", a "\r" kept). An unreadable file is recorded
// and reads as empty: the bash grep 2>/dev/null took it for "no match", which is never clean here.
func (s *scan) lines(p string) []string {
	data, err := os.ReadFile(p)
	if err != nil {
		s.fail(errors.New("cannot scan " + p + ": " + strerror.Text(err)))
		return nil
	}
	lines, _ := javasrc.Lines(bytes.NewReader(data)) // a bytes.Reader never fails
	return lines
}

// kotlinLines reads p as a Kotlin script and returns its code lines (ktssrc.Code: comments dropped, literal
// contents removed, line numbers kept). A file it cannot read or scan is recorded and reads as empty.
func (s *scan) kotlinLines(p string) []string {
	data, err := os.ReadFile(p)
	if err != nil {
		s.fail(errors.New("cannot scan " + p + ": " + strerror.Text(err)))
		return nil
	}
	toks, err := ktssrc.Tokens(string(data))
	if err != nil {
		s.fail(errors.New("cannot scan " + p + ": " + err.Error()))
		return nil
	}
	return ktssrc.Code(toks)
}

// files lists the files named like any of names under dir, sorted, dot sub-directories pruned, as
// `find -H <dir> \( -type d -name '.*' -prune \) -o \( -type f -name <name> -print \) | sort` lists them.
// The start point itself is pruned when its name starts with "." (find applies -name to it, so a root
// given as "." lists nothing), and with follow unset a symlinked start point lists nothing (find without -H).
// A directory the walk cannot read is recorded.
func (s *scan) files(dir string, follow bool, names ...string) []string {
	if dotName(dir) {
		return nil
	}
	if st, err := os.Lstat(dir); !follow && err == nil && st.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	files, err := lint.Walk(dir, lint.Selection{Names: names})
	if err != nil {
		s.fail(errors.New("cannot list every file under " + dir + ": " + err.Error()))
	}
	return files
}

// dotName reports whether find's -name sees a name starting with "." for start point p: its last
// component after trailing slashes are dropped ("." for "./", ".." for "a/..").
func dotName(p string) bool {
	p = strings.TrimRight(p, "/")
	return strings.HasPrefix(p[strings.LastIndexByte(p, '/')+1:], ".")
}

// grep reports whether any line matches re (grep -qE).
func grep(lines []string, re *regexp.Regexp) bool { return count(lines, re) > 0 }

// count is the number of matching lines (grep -cE).
func count(lines []string, re *regexp.Regexp) int {
	n := 0
	for _, ln := range lines {
		if re.MatchString(ln) {
			n++
		}
	}
	return n
}
