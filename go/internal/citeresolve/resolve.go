// Package citeresolve checks each citeparse.Pointer against the source trees it names (kit, research
// corpus, niagara-help, decompiles, memories) and reports whether the cited file, section, delta or
// line exists, with an excerpt. It never judges whether the source says what the card claims.
package citeresolve

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

// Status is the outcome of one pointer.
type Status string

// The statuses. ToolError is a failure of the tool (an unreadable file), never a negative finding.
const (
	Resolved       Status = "RESOLVED"
	MissingFile    Status = "MISSING_FILE"
	MissingSection Status = "MISSING_SECTION"
	MissingDelta   Status = "MISSING_DELTA"
	LineOutOfRange Status = "LINE_OUT_OF_RANGE"
	BareID         Status = "BARE_ID"
	Unparsed       Status = "UNPARSED"
	ToolError      Status = "TOOL_ERROR"
)

// excerptLen is how many characters of each cited line the excerpt keeps.
const excerptLen = 80

// Roots are the trees pointers resolve against.
type Roots struct {
	Kit      string // build-n4-module-kit/ (retros/, types/, METHODOLOGY.md)
	Repo     string // the repository holding the kit (CONTRIBUTING.md, tests/, go/)
	Research string // niagara-research (blocks, niagara-help/, organized/)
	Modules  string // the client module trees (Cliente/…)
	Memory   string // the memory directory (mem pointers)
	Version  string // the niagara-help version an unversioned devguide/guide pointer means
}

// Result is the outcome of one pointer: its status, a detail (excerpt and notes) and the file it hit.
type Result struct {
	Status Status
	Detail string
	File   string
}

// Resolver checks one kind of pointer.
type Resolver interface {
	Resolve(p citeparse.Pointer, c *Card) Result
}

// Card is the per-card state: its directory and the files its earlier pointers resolved to, which an
// ellipsis (…/) pointer resolves against.
type Card struct {
	Dir   string
	files []string
}

// Engine resolves pointers with one resolver per kind and one shared file cache.
type Engine struct {
	roots Roots
	fs    *files
	by    map[citeparse.Kind]Resolver
}

// New returns an engine over roots.
func New(r Roots) *Engine {
	e := &Engine{roots: r, fs: newFiles()}
	e.by = map[citeparse.Kind]Resolver{
		citeparse.Retro:    retroResolver{e},
		citeparse.Corpus:   corpusResolver{e},
		citeparse.Bare:     fixed{BareID, "an ID with no source file"},
		citeparse.Unparsed: fixed{Unparsed, ""},
	}
	e.registerFiles()
	return e
}

// NewCard starts the state of the card at path.
func (e *Engine) NewCard(path string) *Card { return &Card{Dir: filepath.Dir(path)} }

// Resolve checks p; notes the parser kept and a deferred tag are appended to the detail.
func (e *Engine) Resolve(p citeparse.Pointer, c *Card) Result {
	r, ok := e.by[p.Kind]
	if !ok {
		return Result{Status: Unparsed, Detail: "no resolver for kind " + string(p.Kind)}
	}
	res := r.Resolve(p, c)
	if res.Status == Resolved && res.File != "" {
		c.files = append(c.files, res.File)
	}
	var extra []string
	if len(p.Notes) > 0 {
		extra = append(extra, "unchecked: "+strings.Join(p.Notes, " "))
	}
	if p.Deferred {
		extra = append(extra, "deferred")
	}
	if len(extra) > 0 {
		res.Detail = strings.TrimSpace(res.Detail + " [" + strings.Join(extra, "; ") + "]")
	}
	return res
}

type fixed struct {
	s   Status
	why string
}

func (f fixed) Resolve(citeparse.Pointer, *Card) Result { return Result{Status: f.s, Detail: f.why} }

// files caches file contents and directory listings for the run.
type files struct {
	text map[string][]string
	errs map[string]error
	dirs map[string][]string
	idx  map[string]map[string][]string
}

func newFiles() *files {
	return &files{text: map[string][]string{}, errs: map[string]error{}, dirs: map[string][]string{}}
}

// lines returns the lines of the file at path (cached).
func (f *files) lines(path string) ([]string, error) {
	if l, ok := f.text[path]; ok {
		return l, f.errs[path]
	}
	b, err := os.ReadFile(path)
	var l []string
	if err == nil {
		l = strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"), "\n")
	}
	f.text[path], f.errs[path] = l, err
	return l, err
}

// kind stats path: "" when absent, "dir" or "file"; err only when it cannot be told (not ENOENT).
func kind(path string) (string, error) {
	st, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrInvalid):
		return "", nil
	case errors.Is(err, syscall.ENOTDIR):
		return "", nil
	case err != nil:
		return "", err
	case st.IsDir():
		return "dir", nil
	}
	return "file", nil
}

// names lists dir (cached); a missing dir is empty.
func (f *files) names(dir string) ([]string, error) {
	if n, ok := f.dirs[dir]; ok {
		return n, nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
		return nil, err
	}
	var n []string
	for _, e := range ents {
		n = append(n, e.Name())
	}
	f.dirs[dir] = n
	return n, nil
}

func toolError(err error) Result { return Result{Status: ToolError, Detail: err.Error()} }

func excerpt(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > excerptLen {
		return string(r[:excerptLen])
	}
	return s
}

// checkLines verifies every range of p lies inside lines and returns the excerpt of each range's first line.
func checkLines(p citeparse.Pointer, lines []string) (Result, bool) {
	var ex []string
	for _, r := range p.Lines {
		if r.From < 1 || r.To < r.From || r.To > len(lines) {
			return Result{Status: LineOutOfRange, Detail: fmt.Sprintf("lines %d-%d, file has %d", r.From, r.To, len(lines))}, false
		}
		ex = append(ex, excerpt(lines[r.From-1]))
	}
	return Result{Status: Resolved, Detail: strings.Join(ex, " | ")}, true
}
