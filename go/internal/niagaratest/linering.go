package niagaratest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

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
