// Package buildstate generates the per-change parts of the kit block of
// build-n4-module-kit/BUILD-STATE.md — `last_change:`, the `open_issues:` items and the session log
// after `last_session:` — from one fragment file per change under build-n4-module-kit/ledger.d/.
// Parallel kit PRs used to edit those same lines and conflicted on every merge; a fragment is a new
// file, so two PRs never touch the same line. [ev: retro ledger-fragments]
//
// Fragment contract (build-n4-module-kit/ledger.d/<YYYY-MM-DD>-<slug>.md, line based):
//
//	<!-- ledger: kit -->                    required, once
//	<!-- date: YYYY-MM-DD -->               required, once; equals the filename date
//	<!-- last-change-note: <text> -->       optional, once; the `# comment` of last_change
//	log: YYYY-MM-DD · <text>                exactly one (its date = the header date); one session-log line
//	open_issue: <text>                      zero or more; each becomes an `  - <text>` item
//
// Blank lines are ignored; any other line is a violation. The one archive fragment,
// 0000-archive-<label>.md, holds the ledger as it stood before fragments: any number of log lines of
// any date, kept in file order.
package buildstate

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Fragment is one parsed ledger.d file.
type Fragment struct {
	Name, Date, Note string
	Archive          bool
	Logs, Issues     []string
}

// Problem is a ledger.d entry that yields no fragment: a contract violation, or (IO) an unreadable file.
type Problem struct {
	Name, Reason string
	IO           bool
}

const (
	errLedger    = "needs exactly one <!-- ledger: kit --> header"
	errDate      = "needs exactly one <!-- date: YYYY-MM-DD --> header"
	errName      = "the filename must be <date header>-<slug>.md (slug: a-z, 0-9, '.', '-') or 0000-archive-<label>.md"
	errLogCount  = "needs exactly one log: line (the archive fragment: at least one)"
	errLogDate   = "the log: line date must equal the date header"
	errLogShape  = "a log: line must read `log: YYYY-MM-DD · <text>`"
	errIssue     = "an open_issue: line needs text"
	errLine      = "not a header, log: or open_issue: line"
	errDuplicate = "header given twice"
	errStray     = "not a ledger fragment (only *.md files belong in ledger.d/)"
)

var (
	datePattern    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	logPattern     = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2}) · \S`)
	namePattern    = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})-[a-z0-9][a-z0-9.-]*\.md$`)
	archivePattern = regexp.MustCompile(`^0000-archive-[A-Za-z0-9][A-Za-z0-9.-]*\.md$`)
)

// header returns the value of a `<!-- key: value -->` line and whether the line is one.
func header(line, key string) (string, bool) {
	rest, ok := strings.CutPrefix(line, "<!-- "+key+":")
	if !ok || !strings.HasSuffix(rest, "-->") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimSuffix(rest, "-->")), true
}

// ParseFragment parses the fragment called name. It returns the fragment, or the violation that keeps
// it out ("" = ok), or the read error that stopped it (never reported as a violation).
func ParseFragment(name string, r io.Reader) (Fragment, string, error) {
	f := Fragment{Name: name, Archive: archivePattern.MatchString(name)}
	var ledger string
	seen := map[string]bool{}
	br := bufio.NewReader(r)
	for n := 1; ; n++ {
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return Fragment{}, "", err
		}
		if line == "" && err == io.EOF {
			break
		}
		line = strings.TrimSuffix(line, "\n")
		at := "line " + strconv.Itoa(n) + ": "
		if strings.TrimSpace(line) == "" && !strings.Contains(line, "\r") {
			continue
		}
		hdr := false
		for _, key := range []string{"ledger", "date", "last-change-note"} {
			v, ok := header(line, key)
			if !ok {
				continue
			}
			if seen[key] {
				return Fragment{}, at + errDuplicate, nil
			}
			seen[key], hdr = true, true
			switch key {
			case "ledger":
				ledger = v
			case "date":
				f.Date = v
			default:
				f.Note = v
			}
		}
		if hdr {
			continue
		}
		if v, ok := strings.CutPrefix(line, "log: "); ok {
			m := logPattern.FindStringSubmatch(v)
			if m == nil || strings.Contains(v, "\r") {
				return Fragment{}, at + errLogShape, nil
			}
			if !f.Archive && datePattern.MatchString(f.Date) && m[1] != f.Date {
				return Fragment{}, at + errLogDate, nil
			}
			f.Logs = append(f.Logs, v)
			continue
		}
		if v, ok := strings.CutPrefix(line, "open_issue:"); ok && !strings.Contains(v, "\r") {
			if v = strings.TrimPrefix(v, " "); strings.TrimSpace(v) == "" {
				return Fragment{}, at + errIssue, nil
			}
			f.Issues = append(f.Issues, v)
			continue
		}
		return Fragment{}, at + errLine, nil
	}
	switch {
	case ledger != "kit":
		return Fragment{}, errLedger, nil
	case !datePattern.MatchString(f.Date):
		return Fragment{}, errDate, nil
	case !f.Archive && !nameMatches(name, f.Date):
		return Fragment{}, errName, nil
	case len(f.Logs) == 0 || (!f.Archive && len(f.Logs) != 1):
		return Fragment{}, errLogCount, nil
	}
	if !f.Archive {
		for _, l := range f.Logs { // a log line read before the date header
			if l[:10] != f.Date {
				return Fragment{}, errLogDate, nil
			}
		}
	}
	return f, "", nil
}

func nameMatches(name, date string) bool {
	m := namePattern.FindStringSubmatch(name)
	return m != nil && m[1] == date
}

// Scan parses every entry of dir in bytewise name order (dotfiles skipped). It fails closed: an
// unlistable dir is an error, an unreadable fragment an IO problem, any non-*.md entry a problem.
func Scan(dir string) ([]Fragment, []Problem, error) {
	entries, err := os.ReadDir(dir) // sorted by filename
	if err != nil {
		return nil, nil, err
	}
	var frags []Fragment
	var problems []Problem
	for _, de := range entries {
		name := de.Name()
		if name[0] == '.' {
			continue
		}
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err == nil && (!st.Mode().IsRegular() || !strings.HasSuffix(name, ".md")) {
			problems = append(problems, Problem{Name: name, Reason: errStray})
			continue
		}
		var f Fragment
		var reason string
		if err == nil {
			f, reason, err = parseFile(name, path)
		}
		switch {
		case err != nil:
			problems = append(problems, Problem{Name: name, Reason: "cannot read: " + err.Error(), IO: true})
		case reason != "":
			problems = append(problems, Problem{Name: name, Reason: reason})
		default:
			frags = append(frags, f)
		}
	}
	return frags, problems, nil
}

func parseFile(name, path string) (Fragment, string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return Fragment{}, "", err
	}
	defer fh.Close()
	return ParseFragment(name, fh)
}
