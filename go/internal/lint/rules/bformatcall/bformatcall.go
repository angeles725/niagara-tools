// Package bformatcall is lint-bformat-call: BFormat special calls written in a form BFormat never parses.
// BFormat matches time() and user() only as the exact calls "time()" and "user()" (S/util/BFormat.java:443-444), so
// %time(<arg>)% and %user% fall through to reflection and print %err:...%. Port of toolbelt/lint-bformat-call.sh (the
// wrapper keeps the user-facing contract and its header documents the rule and its mutation pins).
//
// It is a per-file lint.Rule over *.java, *.lexicon, *.bog and *.palette; Check says how each is read.
package bformatcall

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/props"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// ID is the lint name printed in every row.
const ID = "lint-bformat-call"

// Spec registers the rule with the argument contract of toolbelt/lint-bformat-call.sh: [--strict] [--] <src-root>,
// the awk_lint_run walk (a listing error still scans the files listed, then exit 3) and the K20 row.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.One,
		Usage:  "usage: lint-bformat-call.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java", "*.lexicon", "*.bog", "*.palette"}},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
		EmptyError:          lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	timeArg  = regexp.MustCompile(`%time\([^)]`)
	bareUser = regexp.MustCompile(`%user%`)
)

const (
	bfc1 = "BFC1 time() takes no argument -- %time(<fmt>)% goes to reflection and prints %err:...%; chain instead: %time().toDateString% (types/logic.md BFormat)"
	bfc2 = "BFC2 %user% is a reflective call that fails -- write %user()% (types/logic.md BFormat)"
)

// Check scans each file where its patterns live: *.java on javasrc.Stripped (comments blanked, string literals kept,
// since BFormat patterns live in strings), *.bog and *.palette through the shared XML scanner (xml), *.lexicon as
// a properties file (lexicon).
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	switch {
	case strings.HasSuffix(f.Path, ".java"):
		return scan(f.Stripped(), func(i int) int { return i + 1 }), nil
	case strings.HasSuffix(f.Path, ".bog") || strings.HasSuffix(f.Path, ".palette"):
		return xml(f.Path)
	}
	return lexicon(f.Raw), nil
}

// lexicon reads a .lexicon as the java.util.Properties file it is (internal/props): every value, its escapes read, at
// the line its entry starts on. Truth fix 2026-10-24 over the awk's line read: a # or ! comment is not read, a value
// continued onto the next line is reported at its entry, and a \u0025 escape is the percent it stands for.
func lexicon(raw []string) []lint.Finding {
	entries := props.Read(raw)
	values := make([]string, len(entries))
	for i, e := range entries {
		values[i] = e.Value
	}
	return scan(values, func(i int) int { return entries[i].Line + 1 })
}

// xml reads a .bog or .palette through xmlsrc (R8-D12; a zipped .bog through its file.xml): every element's attribute
// values and text, entity-decoded, one BFC1 and one BFC2 row at most per element, at the element's line. A malformed
// file is the positioned *xmlsrc.Error (exit 3). Truth fix 2026-10-24 over the awk, which read the text line by line:
// a pattern in an XML comment counted, an entity-encoded one (&#37;user&#37;) was missed, a value wrapped onto the
// next line was reported there, and a zipped bog was read as its compressed bytes.
func xml(path string) ([]lint.Finding, error) {
	doc, err := xmlsrc.Open(path)
	if err != nil {
		return nil, err
	}
	elems := doc.Elements()
	texts := make([]string, len(elems))
	for i, e := range elems {
		var b strings.Builder
		for _, a := range e.Attrs {
			b.WriteString(a.Value)
			b.WriteByte(0) // a pattern never spans two values
		}
		b.WriteString(doc.Text(i))
		texts[i] = b.String()
	}
	return scan(texts, func(i int) int { return elems[i].Line }), nil
}

// scan reports each text's BFC1 and BFC2 rows at line(i).
func scan(texts []string, line func(i int) int) []lint.Finding {
	var out []lint.Finding
	for i, t := range texts {
		if timeArg.MatchString(t) {
			out = append(out, lint.Finding{Line: line(i), Msg: bfc1})
		}
		if bareUser.MatchString(t) {
			out = append(out, lint.Finding{Line: line(i), Msg: bfc2})
		}
	}
	return out
}
