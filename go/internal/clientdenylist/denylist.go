// Package clientdenylist reads the private client denylist the same way the client guard does (niagara-go
// internal/clientguard): one Go RE2 pattern per line, matched case-insensitively, blank lines and lines whose first
// non-blank character is '#' ignored. A hit is reported by rule number (the denylist line), never by the matched text
// or the pattern, so a report cannot re-leak an identifier. The list itself never enters this repository.
package clientdenylist

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"regexp/syntax"
	"strings"
)

type rule struct {
	line int
	re   *regexp.Regexp
}

// Rules is a parsed denylist.
type Rules []rule

// Parse reads a denylist. A pattern that does not compile fails the whole list (fail closed); the error names the
// line and the kind of syntax error, never the pattern.
func Parse(r io.Reader) (Rules, error) {
	var rules Rules
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		re, err := regexp.Compile("(?i)" + text)
		if err != nil {
			kind := "invalid regexp"
			var se *syntax.Error
			if errors.As(err, &se) {
				kind = string(se.Code)
			}
			return nil, fmt.Errorf("denylist line %d: %s", n, kind)
		}
		rules = append(rules, rule{n, re})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("denylist: %w", err)
	}
	return rules, nil
}

// Load parses the denylist at path. A missing file is not an error (found is false: CI and other machines hold no
// private list); any other read or parse failure is.
func Load(path string) (rules Rules, found bool, err error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("denylist: cannot open it (%v)", errors.Unwrap(err))
	}
	defer f.Close()
	rules, err = Parse(f)
	return rules, true, err
}

// Match returns the rule numbers (denylist lines) whose pattern matches text.
func (rs Rules) Match(text string) []int {
	var hits []int
	for _, r := range rs {
		if r.re.MatchString(text) {
			hits = append(hits, r.line)
		}
	}
	return hits
}
