// Package linktargetflags is lint-link-target-flags: flags on linked slots, READONLY on a link-in target
// (LTF1) and TRANSIENT on an operator mode/HOA slot (LTF2). Port of toolbelt/lint-link-target-flags.sh (one
// awk run per file plus an awk harvest of the --wiring-map Table 2); the wrapper keeps the user-facing
// contract and its header documents the rule and its mutation pins.
//
// It is a per-file lint.Rule (comment blocks on raw lines, annotations on code: see Check) and a
// lint.Configurer that reads the --wiring-map file before the walk. The map contract is the layout of
// toolbelt/generate-wiring-map.sh: a "## " section whose heading holds "Table 2", the section's first table
// row as its header (the Source column is the header cell starting with "Source", else the third), and
// backticked first-column slots whose Source cell names a source. Matching is ASCII under LC_ALL=C.
package linktargetflags

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-link-target-flags"

// Spec registers the rule with the argument contract of toolbelt/lint-link-target-flags.sh:
// [--wiring-map <file>] <java-src-dir> in any order, a fail-closed walk, and the K20 ERROR row.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       &rule{},
		Mode:       lint.Valued,
		Option:     "--wiring-map",
		Usage:      "usage: lint-link-target-flags.sh [--wiring-map <docs/wiring-map.md>] <java-src-dir>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no Java sources found (K20: wrong path or empty scaffold)"},
	}
}

// rule holds the Table 2 link-in targets of the --wiring-map file (empty without one).
type rule struct{ targets map[string]bool }

func (*rule) ID() string              { return ID }
func (*rule) Severity() lint.Severity { return lint.Fail }

var (
	comment  = regexp.MustCompile(`^[[:space:]]*(//|/\*|\*)`)
	nameAttr = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([^"]+)"`)
	linkText = regexp.MustCompile(`link-in|linked from|commissioning link|written by (a )?blink`)
	modeName = regexp.MustCompile(`(Mode|Hoa|HOA)[0-9]*$`)
	section  = regexp.MustCompile(`^##[[:space:]]`)
	slotCell = regexp.MustCompile("^`[A-Za-z_][A-Za-z0-9_]*`$")
	selfCell = regexp.MustCompile(`^\(?(self|self-set|n/a|none)\)?$`)
	letter   = regexp.MustCompile(`[a-z]`)
)

const (
	ltf1 = `LTF1: link-in target "%s" (%s) carries READONLY -- LinkCheck refuses a READONLY target; declare it SUMMARY only and guard writes server-side`
	ltf2 = `LTF2: operator mode/HOA slot "%s" is TRANSIENT -- it reverts to its default on restart and a link re-pushes it; persist it`
)

// Configure reads the --wiring-map file. A path that is not a regular file is the bash lint's error; a
// file that cannot be read is an env error too (the bash awk printed its error and went on with no target).
func (*rule) Configure(a lint.Args) (lint.Configured, error) {
	if a.Option == "" {
		return &rule{targets: map[string]bool{}}, nil
	}
	if st, err := os.Stat(a.Option); err != nil || !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: --wiring-map file not found: %s\n", ID, a.Option)
	}
	data, err := os.ReadFile(a.Option)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot read --wiring-map %s: %v\n", ID, a.Option, err)
	}
	return &rule{targets: tableTwo(strings.TrimSuffix(string(data), "\n"))}, nil
}

// tableTwo is the awk harvest: the backticked first-column slots of each Table 2 section whose Source cell,
// lowered, trimmed and with backticks removed, names a source (a letter, and not _(fill)_, self, self-set,
// n/a or none, optionally parenthesised).
func tableTwo(text string) map[string]bool {
	targets := map[string]bool{}
	in2, header, scol := false, false, 3
	for _, ln := range strings.Split(text, "\n") {
		if section.MatchString(ln) {
			in2, header, scol = strings.Contains(ln, "Table 2"), false, 3
			continue
		}
		if !in2 || !strings.HasPrefix(ln, "|") {
			continue
		}
		c := strings.Split(ln, "|") // c[0] is the text before the leading pipe, as awk's c[1]
		if !header {
			header = true
			for k := 1; k < len(c); k++ {
				if strings.HasPrefix(trim(c[k]), "Source") {
					scol = k
				}
			}
			continue
		}
		slot, src := cell(c, 1), lower(cell(c, scol))
		src = trim(strings.ReplaceAll(src, "`", ""))
		if !slotCell.MatchString(slot) || !letter.MatchString(src) || src == "_(fill)_" || selfCell.MatchString(src) {
			continue
		}
		targets[strings.ReplaceAll(slot, "`", "")] = true
	}
	return targets
}

// cell is the trimmed k-th cell (0-based after the text before the leading pipe), "" past the last one.
func cell(c []string, k int) string {
	if k < len(c) {
		return trim(c[k])
	}
	return ""
}

// Check is the awk's line loop: a run of comment lines heads the @NiagaraProperty annotations after it
// (a blank line, code or another annotation ends it); an annotation is read until its parentheses close.
// The comment block is read on the raw lines (its text is the point); the annotation is read on code
// (linescan.Code: comments blanked, literal contents removed) for its marker, its parentheses and its flag
// tokens, and on the stripped lines (literal text kept) for its name. Truth fix 2026-10-24: the awk read the
// raw text, so a commented-out annotation, a flag word in a defaultValue string or in a comment inside the
// annotation, and a parenthesis in a string all counted as code.
func (r *rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	code, named := linescan.Code(f), f.Stripped()
	inProp, prevCmt := false, false
	buf, nbuf, cmt, pline := "", "", "", 0
	for i, ln := range f.Raw {
		switch {
		case inProp:
			buf, nbuf = buf+" "+code[i], nbuf+" "+named[i]
		case strings.Contains(code[i], "@NiagaraProperty"):
			inProp, buf, nbuf, pline = true, code[i], named[i], i+1
		case comment.MatchString(ln):
			if prevCmt {
				cmt += " " + ln
			} else {
				cmt = ln // a comment after a property starts a new block
			}
			prevCmt = true
			continue
		default:
			cmt, prevCmt = "", false
			continue
		}
		if strings.Count(buf, "(")-strings.Count(buf, ")") > 0 || !strings.Contains(buf, "(") {
			continue
		}
		if m := nameAttr.FindStringSubmatch(nbuf); m != nil {
			name := m[1]
			linked, mapped := linkText.MatchString(lower(cmt)), r.targets[name]
			if strings.Contains(buf, "READONLY") && (linked || mapped) {
				why := "wiring-map Table 2" // "(wiring-map Table 2)" is matched by report-module.sh and commissioning-verify.sh
				if linked {
					why = "link comment"
				}
				out = append(out, lint.Finding{Line: pline, Msg: fmt.Sprintf(ltf1, name, why)})
			}
			if strings.Contains(buf, "TRANSIENT") && strings.Contains(buf, "OPERATOR") &&
				(modeName.MatchString(name) || strings.HasPrefix(name, "hoa")) {
				out = append(out, lint.Finding{Line: pline, Msg: fmt.Sprintf(ltf2, name)})
			}
		}
		inProp, buf, nbuf, pline, prevCmt = false, "", "", 0, false // cmt kept: a block heads its run of properties
	}
	return out, nil
}

// trim is the awk trim(): leading and trailing [[:space:]] removed.
func trim(s string) string { return strings.Trim(s, " \t\n\v\f\r") }

// lower is tolower under LC_ALL=C: ASCII letters only.
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
