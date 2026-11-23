package foldaudit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/retrogh"
	"github.com/angeles725/niagara-tools/go/internal/retrorows"
)

// lookup asks GitHub, lazily and at most once per question, whether the issue that tracks a delta is closed: a delta
// whose issue is closed counts as cited (the issue is the durable record; the fix PR said `Fixes #N`). gh is never
// called for a run that has no row with an issue number and no external root.
type lookup struct {
	run        retrogh.Runner
	loaded     bool
	markers    map[string]retrogh.Issue
	markersErr error
	states     map[int]string
	stateErrs  map[int]error
	unverified int
}

// markerClosed reports whether a closed retro-delta issue carries the marker "<src> <id>" in its body or a comment.
func (l *lookup) markerClosed(key string) (bool, error) {
	if !l.loaded {
		l.loaded = true
		var issues []retrogh.Issue
		issues, l.markersErr = retrogh.ListLabeled(l.run, retrogh.Label)
		l.markers = retrogh.Markers(issues)
	}
	if l.markersErr != nil {
		return false, l.markersErr
	}
	is, ok := l.markers[key]
	return ok && is.Closed(), nil
}

// issueClosed reports whether issue n is closed.
func (l *lookup) issueClosed(n int) (bool, error) {
	if l.states == nil {
		l.states, l.stateErrs = map[int]string{}, map[int]error{}
	}
	if err, ok := l.stateErrs[n]; ok {
		return false, err
	}
	st, ok := l.states[n]
	if !ok {
		var err error
		if st, err = retrogh.State(l.run, n); err != nil {
			l.stateErrs[n] = err
			return false, err
		}
		l.states[n] = st
	}
	return strings.EqualFold(st, "CLOSED"), nil
}

// unverifiedLine is the report of a lookup gh could not answer: the delta stays uncited, and it is said, not passed.
func (t *audit) unverifiedLine(who, id, what string, err error) {
	msg := err.Error()
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = msg[:i]
	}
	fmt.Fprintf(t.s.Out, "fold-audit: UNVERIFIED %s %s (%s: %s)\n", who, id, what, msg)
	t.gh.unverified++
}

// trackedClosed reports whether the row id of the kit retro file carries a "#N" whose issue is closed.
func (t *audit) trackedClosed(file, id string) bool {
	rows, ok := t.retroRows[file]
	if !ok {
		if t.retroRows == nil {
			t.retroRows = map[string][]retrorows.Row{}
		}
		if b, err := os.ReadFile(filepath.Join(t.o.root, "retros", file)); err == nil {
			rows = retrorows.Parse(string(b))
		}
		t.retroRows[file] = rows
	}
	for _, r := range rows {
		if r.ID != id || r.Issue == 0 {
			continue
		}
		closed, err := t.gh.issueClosed(r.Issue)
		if err != nil {
			t.unverifiedLine(file, id, fmt.Sprintf("issue #%d state unavailable", r.Issue), err)
		}
		return closed
	}
	return false
}

// issueSummary is the closing line for lookups gh could not answer.
func (t *audit) issueSummary() {
	if t.gh.unverified > 0 {
		fmt.Fprintf(t.s.Out, "fold-audit: issues: %d unverified\n", t.gh.unverified)
	}
}
