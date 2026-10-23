package structure

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// scan is one run over a module root: the rows in report order and the first thing it could not read.
type scan struct {
	root string
	out  []lint.Finding
	err  error
}

// row adds a finding; sev Unset takes the rule's FAIL.
func (s *scan) row(sev lint.Severity, path, msg string) {
	s.out = append(s.out, lint.Finding{Severity: sev, Path: path, Msg: msg})
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

// dirname is POSIX dirname(1): "." for a name without a slash, "/" for the root.
func dirname(p string) string {
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return "."
	}
	if p = strings.TrimRight(p[:i], "/"); p == "" {
		return "/"
	}
	return p
}

// isFile is test -f: a regular file, symlinks followed.
func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

// isDir is test -d: a directory, symlinks followed.
func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
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
