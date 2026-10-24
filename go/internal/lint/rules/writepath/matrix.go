package writepath

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

var (
	slotName    = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
	cellCut     = regexp.MustCompile("[ \t\n\v\f\r`/(].*$")
	backticked  = regexp.MustCompile("^`([^`]+)`")
	htmlComment = regexp.MustCompile(`<!--[^>]*-->`)
)

// space is awk's [[:space:]] in the C locale.
const space = " \t\n\v\f\r"

// matrix is the write-path matrix: its path as found (rows print it) and its lines.
type matrix struct {
	path  string
	lines []string
}

// matrix resolves the matrix: the --matrix file as given, or the first <dir>/docs/write-path-matrix.md
// that holds a table row (a line starting with "|"), walking up from the root to the first directory
// holding a .git (or the filesystem root). A file without one (a heading, a stub) is skipped: the bash
// `grep -q '^\|'` matched any line, so it stopped the walk there (truth fix T1).
func (s *scan) matrix(override string) (*matrix, error) {
	if override != "" {
		if !lint.IsFile(override) {
			return nil, s.errorRow("--matrix file not found: " + override)
		}
		return readMatrix(override)
	}
	var looked []string
	for dir := s.root; ; {
		p := dir + "/docs/write-path-matrix.md"
		looked = append(looked, p)
		if data, err := os.ReadFile(p); err == nil && lint.IsFile(p) {
			if m := newMatrix(p, data); m.hasRow() {
				return m, nil
			}
		}
		up := lint.Dirname(dir)
		if lint.IsDir(dir+"/.git") || up == dir {
			return nil, s.errorRow("no write-path-matrix.md found (looked in " + strings.Join(looked, ", ") + ")")
		}
		dir = up
	}
}

func readMatrix(p string) (*matrix, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %s", p, strerror.Text(err))
	}
	return newMatrix(p, data), nil
}

func newMatrix(p string, data []byte) *matrix {
	ls, _ := javasrc.Lines(strings.NewReader(string(data))) // a strings.Reader never fails
	return &matrix{path: p, lines: ls}
}

func (m *matrix) hasRow() bool {
	for _, ln := range m.lines {
		if strings.HasPrefix(ln, "|") {
			return true
		}
	}
	return false
}

// cell is awk -F'|' $2 trimmed: the text between a row's first and second "|".
func cell(row string) string {
	c := row[1:]
	if i := strings.IndexByte(c, '|'); i >= 0 {
		c = c[:i]
	}
	return strings.Trim(c, space)
}

// slots are the slot names the matrix has a row for: a "|" line whose first cell starts with a slot name,
// backticked (`name`...) or bare (cut at a blank, backtick, "/" or "(").
func (m *matrix) slots() map[string]bool {
	set := map[string]bool{}
	for _, ln := range m.lines {
		if !strings.HasPrefix(ln, "|") {
			continue
		}
		c := cell(ln)
		if strings.HasPrefix(c, "`") {
			c = c[1:]
			if i := strings.IndexByte(c, '`'); i >= 0 {
				c = c[:i]
			}
		} else {
			c = cellCut.ReplaceAllString(c, "")
		}
		if slotName.MatchString(c) {
			set[c] = true
		}
	}
	return set
}

// stale is the per-row pass: a row whose backticked first-cell slot is in no source (covered) is STALE
// unless that row, comments stripped, carries [concept]; a [concept] row whose slot is covered is DRIFT.
func (m *matrix) stale(covered map[string]bool) []lint.Finding {
	var out []lint.Finding
	for i, ln := range m.lines {
		if !strings.HasPrefix(ln, "|") { // every line, a last one without "\n" too (truth fix T2)
			continue
		}
		row := htmlComment.ReplaceAllString(ln, "")
		concept := strings.Contains(row, "[concept]")
		g := backticked.FindStringSubmatch(cell(row)) // only a backticked first cell names a slot here
		if g == nil || !slotName.MatchString(g[1]) {
			continue
		}
		name := g[1]
		at := lint.Finding{Severity: lint.Warn, Path: m.path, Line: i + 1}
		switch {
		case covered[name] && concept:
			at.Status, at.Msg = "DRIFT", "slot "+name+": concept marker but a source slot exists"
		case !covered[name] && !concept:
			at.Status, at.Msg = "STALE", "slot "+name+": no source slot with that name"
		default:
			continue
		}
		out = append(out, at)
	}
	return out
}
