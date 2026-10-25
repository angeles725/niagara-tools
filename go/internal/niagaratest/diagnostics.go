package niagaratest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// The run log's bounds: every read is a tail of lines, each line cut at diagLineCap bytes, and a file is read
// from at most its last diagWindow bytes; no buffer is ever sized from a file's size.
const (
	diagRowLines    = 500
	diagStderrLines = 200
	diagStdoutLines = 50
	diagLineCap     = 4 << 10
	diagWindow      = 8 << 20
	diagMaxEntries  = 200
)

// evidenceCap bounds each file a failed run keeps beside its log: the testng-results.xml copy, the traces and the
// stderr copy.
const evidenceCap = 4 << 20

// lineRing is an io.Writer that keeps the last max lines written to it, each cut at lineCap bytes; CR is dropped.
type lineRing struct {
	mu           sync.Mutex
	max, lineCap int
	lines        []string // a ring: next is the oldest once full
	next, total  int
	cur          []byte
	cut          bool
}

func newLineRing(max, lineCap int) *lineRing { return &lineRing{max: max, lineCap: lineCap} }

func (r *lineRing) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(p)
	for {
		i := bytes.IndexByte(p, '\n')
		chunk := p
		if i >= 0 {
			chunk = p[:i]
		}
		if room := r.lineCap - len(r.cur); len(chunk) > room {
			r.cur, r.cut = append(r.cur, chunk[:max(room, 0)]...), true
		} else {
			r.cur = append(r.cur, chunk...)
		}
		if i < 0 {
			return n, nil
		}
		r.push()
		p = p[i+1:]
	}
}

func (r *lineRing) push() {
	line := strings.ReplaceAll(string(r.cur), "\r", "")
	if r.cut {
		line += fmt.Sprintf(" [line cut at %d bytes]", r.lineCap)
	}
	if len(r.lines) < r.max {
		r.lines = append(r.lines, line)
	} else {
		r.lines[r.next] = line
		r.next = (r.next + 1) % r.max
	}
	r.total++
	r.cur, r.cut = r.cur[:0], false
}

// snapshot is the kept lines, oldest first (a pending unterminated line included), and how many were seen.
func (r *lineRing) snapshot() ([]string, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.cur) > 0 || r.cut {
		r.push()
	}
	out := append(append([]string{}, r.lines[r.next:]...), r.lines[:r.next]...)
	return out, r.total
}

// tee sends a copy of everything the run prints on s to r: the log keeps the runner's own rows.
func tee(s cli.Stdio, r *lineRing) cli.Stdio {
	return cli.Stdio{Out: io.MultiWriter(s.Out, r), Err: io.MultiWriter(s.Err, r), Getenv: s.Getenv}
}

// tailFile is the last n lines of a file read through a lineRing from at most its last diagWindow bytes (a
// line the window starts inside is dropped); ok is false when the file cannot be opened.
func tailFile(path string, n int) (lines []string, total int, windowed, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, false, false
	}
	defer f.Close()
	r := newLineRing(n, diagLineCap)
	var src io.Reader = f
	if info, err := f.Stat(); err == nil && info.Size() > diagWindow {
		if _, err := f.Seek(-diagWindow, io.SeekEnd); err == nil {
			windowed = true
			src = &skipFirstLine{r: f}
		}
	}
	_, _ = io.Copy(r, io.LimitReader(src, diagWindow))
	lines, total = r.snapshot()
	return lines, total, windowed, true
}

// skipFirstLine drops everything up to and including the first newline: the partial line a seek lands in.
type skipFirstLine struct {
	r    io.Reader
	done bool
}

func (s *skipFirstLine) Read(p []byte) (int, error) {
	for !s.done {
		n, err := s.r.Read(p)
		if i := bytes.IndexByte(p[:n], '\n'); i >= 0 {
			s.done = true
			return copy(p, p[i+1:n]), err
		}
		if err != nil {
			return 0, err
		}
	}
	return s.r.Read(p)
}

// evidence is what the run log takes from the scratch dir, read before cleanup deletes it.
type evidence struct {
	stderr, stdout, listing []string
	testng                  testngEvidence // kept beside the log on a failed run only
	fullStderr              stderrEvidence // kept beside the log on a failed run only
}

// gatherEvidence reads the tails of test.exe's streams and lists its -output: dir, every read bounded.
func gatherEvidence(p *plan) evidence {
	var ev evidence
	for _, f := range []struct {
		name string
		n    int
		dst  *[]string
	}{{"stderr", diagStderrLines, &ev.stderr}, {"stdout", diagStdoutLines, &ev.stdout}} {
		lines, total, windowed, ok := tailFile(p.sub("out/"+f.name+".txt"), f.n)
		switch {
		case !ok:
			*f.dst = []string{"(no " + f.name + ".txt: test.exe did not run)"}
		case windowed:
			*f.dst = append([]string{fmt.Sprintf("(last %d lines of the file's last %d bytes)", len(lines), diagWindow)}, lines...)
		default:
			*f.dst = append([]string{fmt.Sprintf("(last %d of %d lines)", len(lines), total)}, lines...)
		}
	}
	ev.testng = gatherTestNG(p.sub("out/testng/testng-results.xml"))
	ev.fullStderr = gatherStderr(p.sub("out/stderr.txt"))
	root := p.sub("out/testng")
	if !isDir(root) {
		ev.listing = []string{"(no -output: dir)"}
		return ev
	}
	n := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if n++; n > diagMaxEntries {
			return fs.SkipAll
		}
		rel, _ := filepath.Rel(root, path)
		size := int64(-1)
		if info, err := d.Info(); err == nil {
			size = info.Size()
		}
		ev.listing = append(ev.listing, fmt.Sprintf("%s  %d bytes", filepath.ToSlash(rel), size))
		return nil
	})
	switch {
	case n > diagMaxEntries:
		ev.listing = append(ev.listing, fmt.Sprintf("(listing stopped at %d files)", diagMaxEntries))
	case n == 0:
		ev.listing = []string{"(empty)"}
	}
	return ev
}

// redaction is one path the log hides behind label, in its Windows form (also spelt with forward slashes)
// and its WSL form; an empty form is skipped.
type redaction struct{ win, wsl, label string }

type redactor []struct {
	re    *regexp.Regexp
	label string
}

// newRedactor matches every form case-insensitively and only as a whole path component (C:\Users\you is not
// a prefix of C:\Users\young), the longest path first so the home copy wins over the scratch dir around it.
func newRedactor(rs ...redaction) redactor {
	type form struct{ path, label string }
	var forms []form
	for _, r := range rs {
		for _, p := range []string{r.win, strings.ReplaceAll(r.win, `\`, "/"), r.wsl} {
			if p = strings.TrimRight(p, `\/`); len(p) > 1 {
				forms = append(forms, form{p, r.label})
			}
		}
	}
	sort.SliceStable(forms, func(i, j int) bool { return len(forms[i].path) > len(forms[j].path) })
	var out redactor
	for _, f := range forms {
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(f.path) + `([^A-Za-z0-9_.\-]|$)`)
		out = append(out, struct {
			re    *regexp.Regexp
			label string
		}{re, f.label})
	}
	return out
}

func (r redactor) apply(s string) string {
	for _, x := range r {
		s = x.re.ReplaceAllString(s, x.label+"${1}")
	}
	return s
}

// redactorFor hides the scratch dir, its home copy, %USERPROFILE% and the WSL home; install paths stay, the
// user's own and needed to diagnose.
func (h host) redactorFor(p *plan) redactor {
	return newRedactor(redaction{p.home().win, p.home().wsl, "<copy>"}, redaction{p.dir.win, p.dir.wsl, "<scratch>"},
		redaction{p.profile.win, p.profile.wsl, "<you>"}, redaction{"", h.home, "<you>"})
}

// exitNames spell the run's exit codes in the log.
var exitNames = map[int]string{exitPass: "PASS", exitFail: "FAIL", exitUsage: "usage or refused", exitEnv: "environment", exitTimeout: "timeout"}

// writeRunLog writes the redacted run log, mode 0600 in a 0700 dir, and prints the row naming it. A failed run
// always keeps one; a green run only for an explicit --log-dir or --keep-home. It returns false when a log was
// due and could not be written.
func (h host) writeRunLog(p *plan, ev evidence, code int, s cli.Stdio) bool {
	if code == exitPass && !p.logDirSet && !p.keepHome {
		return true
	}
	status, label := "FAIL", "diagnostics kept"
	if code == exitPass {
		status, label = "PASS", "run log"
	}
	if p.logDir == "" {
		row(s, "FAIL", "diagnostics: no log dir (no $XDG_STATE_HOME or home); pass --log-dir")
		return false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "run-niagara-test run log\nmodule: %s\npart: %s\ninstall: %s\nrun id: %s\nwritten: %s\n",
		p.module, p.part, p.install.win, p.runID, time.Now().UTC().Format(time.RFC3339))
	section := func(title string, lines []string) {
		fmt.Fprintf(&b, "\n== %s ==\n", title)
		for _, l := range lines {
			b.WriteString(l + "\n")
		}
	}
	rows := []string{"(none)"}
	if p.rows != nil {
		rows, _ = p.rows.snapshot()
	}
	section("runner rows", rows)
	section("test.exe stderr", ev.stderr)
	section("test.exe stdout", ev.stdout)
	section("test.exe -output: dir", ev.listing)
	path, err := createLog(p.logDir, p.module)
	if err != nil {
		row(s, "FAIL", "diagnostics: cannot write the run log in "+p.logDir+": "+err.Error())
		return false
	}
	red, kept := h.redactorFor(p), true
	if code != exitPass {
		stem := strings.TrimSuffix(path.name, ".log")
		rows, kerr := keepTestNG(ev.testng, stem, red)
		if kerr == nil {
			var more []string
			more, kerr = keepStderr(ev.fullStderr, stem, red)
			rows = append(rows, more...)
		}
		if kerr != nil {
			rows, kept = append(rows, "cannot keep the fail evidence: "+kerr.Error()), false
		}
		section("kept evidence", rows)
	}
	name := exitNames[code]
	if name == "" {
		name = "unknown"
	}
	fmt.Fprintf(&b, "\nexit: %d (%s)\n", code, name)
	_, err = path.f.WriteString(red.apply(b.String()))
	if cerr := path.f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		row(s, "FAIL", "diagnostics: cannot write the run log in "+p.logDir+": "+err.Error())
		return false
	}
	p.logPath = path.name
	row(s, status, label+": "+path.name)
	if !kept {
		row(s, "FAIL", "diagnostics: cannot keep the fail evidence beside "+path.name)
	}
	return kept
}

type logFile struct {
	f    *os.File
	name string
}

// createLog makes <dir>/<UTC timestamp>-<module>.log exclusively (a -<n> suffix when that second is taken).
func createLog(dir, module string) (logFile, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return logFile{}, err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for i := 1; i <= 100; i++ {
		n := filepath.Join(dir, fmt.Sprintf("%s-%s.log", stamp, module))
		if i > 1 {
			n = filepath.Join(dir, fmt.Sprintf("%s-%s-%d.log", stamp, module, i))
		}
		f, err := os.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return logFile{f, n}, err
	}
	return logFile{}, fmt.Errorf("100 logs named %s-%s already", stamp, module)
}
