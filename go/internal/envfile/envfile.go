// Package envfile reads an operator env file without a shell: the plain assignments a bash `source` would
// run, and nothing else. A tool that used to `source` such a file reads it here; a line bash would have
// executed (a command, a substitution) is refused with its file and line number, never skipped, so the
// file means the same thing to every reader.
//
// Supported, as bash reads them: blank lines, `# comments` (whole line or after a blank), `KEY=value`,
// `export KEY=value`, several assignments on one line, values made of bare text, '...' (literal) and "..."
// (with \" \\ \$ \` escapes), and a backslash that escapes one bare byte. $NAME and ${NAME} expand in bare
// text and inside double quotes, from an earlier assignment of the file first, then from the caller's
// Lookup; an unset variable is refused by name, as the read under `set -u` it replaces. A bare ~ at the
// start of a value or after a ":" in it, alone or before "/" or ":", is the home directory (expand.go).
// Only a space and a tab separate words: the carriage return of a CRLF file stays in the value, unless the
// caller passes TrimCR (options.go).
//
// Refused: commands, functions, $(...), backquotes, arithmetic, ${NAME<op>...} forms, $1 $@ $? and the
// other shell parameters, $'...', ~user, shell operators, a value that spans lines.
//
// An error never quotes the text of a line or a value: an env file holds credentials.
package envfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

// Lookup is a variable the reader knows outside the file (its environment): its value and whether it is
// set. A set but empty variable expands to nothing; an unset one is refused.
type Lookup func(name string) (value string, set bool)

// Assignment is one KEY=value of a file, with the line it stands on.
type Assignment struct {
	Key, Value string
	Line       int
}

// SyntaxError is a line that is not a plain assignment.
type SyntaxError struct {
	Path   string
	Line   int
	Reason string
}

func (e *SyntaxError) Error() string { return fmt.Sprintf("%s: line %d: %s", e.Path, e.Line, e.Reason) }

// Load reads the assignments of the file path, in file order. found is false, without an error, when there
// is no regular file there (absent, or a directory), which the caller tolerates as bash's `[[ -f ]]` did.
// A path that cannot be checked for another reason, or a file that cannot be read, is an error.
func Load(path string, look Lookup, opts ...Option) (assignments []Assignment, found bool, err error) {
	st, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	case !st.Mode().IsRegular():
		return nil, false, nil
	}
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	assignments, err = Parse(path, string(text), look, opts...)
	return assignments, err == nil, err
}

// Parse reads the assignments of an env file text; path only names the file in an error. The first line
// that is not a plain assignment stops the read: nothing is returned with a SyntaxError.
func Parse(path, text string, look Lookup, opts ...Option) ([]Assignment, error) {
	var out []Assignment
	sc := &scope{file: map[string]string{}, outer: look}
	var r reader
	for _, opt := range opts {
		opt(&r)
	}
	for i, line := range strings.Split(text, "\n") {
		if r.crLines != nil && strings.HasSuffix(line, "\r") {
			line = line[:len(line)-1]
			*r.crLines = append(*r.crLines, i+1)
		}
		words, reason := splitWords(line, sc)
		if reason == "" {
			var keep []Assignment
			if keep, reason = assignments(words); reason == "" {
				for _, a := range keep {
					a.Line = i + 1
					out = append(out, a)
					sc.file[a.Key] = a.Value
				}
				continue
			}
		}
		return nil, &SyntaxError{Path: path, Line: i + 1, Reason: reason}
	}
	return out, nil
}

// assignments turns the words of one line into assignments: every word must be one, after an optional
// leading `export`. reason says why the line is refused; "" when it is not.
func assignments(words []word) (out []Assignment, reason string) {
	if len(words) > 0 && words[0].isBare(exportWord) {
		if words = words[1:]; len(words) == 0 {
			return nil, reasonExport
		}
	}
	for _, w := range words {
		if !isName(w.key) {
			return nil, reasonNotAssignment
		}
		out = append(out, Assignment{Key: w.key, Value: w.value})
	}
	return out, ""
}

// isName reports a shell variable name: a letter or underscore, then letters, digits and underscores.
func isName(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		letter := c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		if !letter && (i == 0 || c < '0' || c > '9') {
			return false
		}
	}
	return s != ""
}
