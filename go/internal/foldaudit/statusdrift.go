package foldaudit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// The line-1 retro marker (BUILD-LOOP.md: the header is the retro's INDEX row, the single source of its status).
	foldedHeaderRe = regexp.MustCompile(`^<!--\s*review-status:\s*folded\s*-->`)
	// The body stamp new-retro writes as `**Status**: PENDING — …`; a fold must flip it with the header.
	pendingBodyRe = regexp.MustCompile(`^\*\*Status\*\*: *PENDING`)
)

// statusDrift reports a folded retro whose body `**Status**` line still says PENDING: the header and the body
// disagree about the retro's lifecycle. The retro is <kit-root>/retros/<file>; one that cannot be read, or whose
// own line-1 marker is not folded (a stale INDEX row), is not audited here. It fails --strict like every WARN.
func (t *audit) statusDrift(file string) {
	b, err := os.ReadFile(filepath.Join(t.o.root, "retros", file))
	if err != nil {
		return
	}
	lines := strings.Split(string(b), "\n")
	if !foldedHeaderRe.MatchString(lines[0]) {
		return
	}
	for i, l := range lines {
		if pendingBodyRe.MatchString(l) {
			fmt.Fprintf(t.s.Out, "fold-audit: WARN %s:%d header says folded but body Status says PENDING: make it `**Status**: FOLDED …`\n", file, i+1)
			t.nDrift++
			t.warned = true
			return
		}
	}
}
