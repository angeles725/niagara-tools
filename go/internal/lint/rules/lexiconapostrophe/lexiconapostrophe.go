// Package lexiconapostrophe is lint-lexicon-apostrophe, the lint-lexicon-ascii companion (corpus B1242 §1242.7 K11,
// from B1236 §1236.4): Lexicon.getText(key, args...) runs MessageFormat.format(value, args), and MessageFormat reads a
// single apostrophe as a quote, so "doesn't {0}" renders "doesnt ..." and a second lone apostrophe swallows the text
// between. A literal apostrophe is written as a doubled one. toolbelt/lint-lexicon-apostrophe.sh is the wrapper and its header is
// the user-facing contract.
//
// A per-file rule over *.lexicon (Java .properties): a "#" or "!" line is a comment, a value continues on the next
// line after an odd run of trailing backslashes, and the key ends at the first unescaped "=", ":" or blank. Row LX1:
// a value with a MessageFormat argument ("{" then a digit) and an apostrophe that is not part of a doubled pair and does not
// quote a blank-free text holding a brace (the deliberate '{' form). A value without arguments is not judged: read
// with getText(key) it is never formatted (the course's Lab 7 "doesn't" key is safe that way).
package lexiconapostrophe

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-lexicon-apostrophe"

// Spec registers the rule: [--strict] [--] <module-root>, WARN rows; a root with no *.lexicon is clean.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:    rule{},
		Mode:    lint.One,
		Usage:   "usage: lint-lexicon-apostrophe.sh [--strict] <module-root>",
		Select:  lint.Selection{Names: []string{"*.lexicon"}},
		EmptyOK: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const lx1 = " has a MessageFormat argument and a single ' that MessageFormat drops (doesn't renders doesnt): write '' (types/lexicon.md § 3)"

// Check reports LX1 at the line of the entry's key.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for i := 0; i < len(f.Raw); i++ {
		line := strings.TrimLeft(strings.TrimSuffix(f.Raw[i], "\r"), " \t\f")
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		start := i
		for continued(line) && i+1 < len(f.Raw) {
			i++
			line = line[:len(line)-1] + strings.TrimLeft(strings.TrimSuffix(f.Raw[i], "\r"), " \t\f")
		}
		key, value := split(line)
		if hasArgument(value) && loneApostrophe(value) {
			out = append(out, lint.Finding{Line: start + 1, Msg: "LX1 " + key + lx1})
		}
	}
	return out, nil
}

// continued: the line ends with an odd run of backslashes.
func continued(s string) bool {
	n := 0
	for j := len(s) - 1; j >= 0 && s[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

// split cuts a logical line into its key and its value: the key ends at the first unescaped "=", ":" or blank;
// blanks and one "=" or ":" between them are the separator.
func split(s string) (string, string) {
	k := 0
	for ; k < len(s); k++ {
		if s[k] == '\\' {
			k++
			continue
		}
		if s[k] == '=' || s[k] == ':' || s[k] == ' ' || s[k] == '\t' || s[k] == '\f' {
			break
		}
	}
	if k > len(s) {
		k = len(s)
	}
	key, rest := s[:k], strings.TrimLeft(s[k:], " \t\f")
	if rest != "" && (rest[0] == '=' || rest[0] == ':') {
		rest = strings.TrimLeft(rest[1:], " \t\f")
	}
	return key, rest
}

// hasArgument: the value holds a MessageFormat argument, "{" then optional blanks then a digit.
func hasArgument(v string) bool {
	for j := 0; j < len(v); j++ {
		if v[j] != '{' {
			continue
		}
		k := j + 1
		for k < len(v) && v[k] == ' ' {
			k++
		}
		if k < len(v) && v[k] >= '0' && v[k] <= '9' {
			return true
		}
	}
	return false
}

// loneApostrophe: an apostrophe that is not a doubled pair and does not open a deliberate quote: a blank-free text holding a
// brace up to the next apostrophe ('{' or '{0}').
func loneApostrophe(v string) bool {
	for j := 0; j < len(v); j++ {
		if v[j] != '\'' {
			continue
		}
		if j+1 < len(v) && v[j+1] == '\'' {
			j++
			continue
		}
		end := strings.IndexByte(v[j+1:], '\'')
		if end < 0 {
			return true
		}
		quoted := v[j+1 : j+1+end]
		// Mutation: LX-closed-quote -- treating every closed quote as deliberate passes 'OK', '{ 0}' and '{0} x'
		if !strings.ContainsAny(quoted, "{}") || strings.ContainsAny(quoted, " \t") {
			return true
		}
		j += end + 1
	}
	return false
}
