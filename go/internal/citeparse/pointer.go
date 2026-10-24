// Package citeparse reads the evidence pointers out of a /build-n4-module kit card: the `[ev: …]` and
// `[deferred: …]` tags and the inline citations of prose. It is pure (no I/O); go/internal/citeresolve
// checks each pointer against the source trees.
package citeparse

import (
	"fmt"
	"strconv"
	"strings"
)

// Kind names what a pointer points at, and so which resolver checks it.
type Kind string

// The pointer kinds. Bare and Unparsed are findings of the parser itself; Template and Unverifiable are
// refinements of a parsed clause (refine.go): no tree can hold what they name.
const (
	Retro        Kind = "retro"        // Target = retro slug
	Corpus       Kind = "corpus"       // Target = B<n> research block
	Code         Kind = "code"         // Target = <mod>-<profile>, File = file in that module part
	Path         Kind = "path"         // Target = a path relative to one of the roots
	Devguide     Kind = "devguide"     // Target = file under devguide-clean/
	Guide        Kind = "guide"        // Target = file under guides-clean/
	Help         Kind = "niagara-help" // Target = path under niagara-help/versions/<Version>/
	Mem          Kind = "mem"          // Target = memory file name prefix
	Vineflower   Kind = "vineflower"   // Target = <mod>-<profile>, File = path in its vineflower/ copy
	Javap        Kind = "javap"        // Target = jar file name
	Jar          Kind = "jar"          // Target = jar file name, File = the entry in it (`x.jar!entry`)
	Archive      Kind = "archive"      // Target = path of a zip, File = the entry in it (`x.zip!entry`)
	Course       Kind = "course"       // Target = path under the dev-course tree (or its course-text)
	Bitacora     Kind = "bitacora"     // Target = slug of a client bitacora/<date>-<slug>.md
	Bare         Kind = "bare"         // an ID with no source (OBS-2, WIR-G3, Δn TOKEN)
	Unparsed     Kind = "unparsed"     // text the grammar does not cover
	Template     Kind = "template"     // a placeholder (`<stem>`, `B<n>`, `…`): not a pointer
	Unverifiable Kind = "unverifiable" // evidence of a kind no tree holds; Notes[0] names the class
)

// Range is an inclusive 1-based line or row range.
type Range struct{ From, To int }

// Pointer is one evidence pointer found on a card line.
type Pointer struct {
	Line     int // 1-based card line where the pointer starts
	Kind     Kind
	Target   string
	File     string // Code and Vineflower only
	Version  string // Help, Vineflower and Javap only
	Lines    []Range
	Anchor   string   // a quoted phrase after the lines that must be on or near them
	Sections []string // corpus § tokens, or the heading of a kit cross-reference
	Rows     []Range
	Pages    []Range  // course PDF pages
	Claims   []string // corpus C<k>
	Rules    []string // corpus rule k
	Delta    int      // retro Δn (0 = none)
	Tokens   []string // words that must appear in the cited file
	Notes    []string // locator text the tool does not check
	Deferred bool     // read from a [deferred: …] tag
	Raw      string   // the clause as written
}

// Locator renders the parsed locator: line ranges, sections, rows, claims, rules, delta and tokens.
func (p Pointer) Locator() string {
	var parts []string
	if len(p.Lines) > 0 {
		parts = append(parts, ":"+FormatRanges(p.Lines))
	}
	if p.Anchor != "" {
		parts = append(parts, `"`+p.Anchor+`"`)
	}
	if len(p.Pages) > 0 {
		parts = append(parts, "p"+FormatRanges(p.Pages))
	}
	for _, s := range p.Sections {
		parts = append(parts, "§"+s)
	}
	if len(p.Rows) == 1 && p.Rows[0].From == p.Rows[0].To {
		parts = append(parts, "row "+FormatRanges(p.Rows))
	} else if len(p.Rows) > 0 {
		parts = append(parts, "rows "+FormatRanges(p.Rows))
	}
	parts = append(parts, p.Claims...)
	for _, r := range p.Rules {
		parts = append(parts, "rule "+r)
	}
	if p.Delta > 0 {
		parts = append(parts, fmt.Sprintf("Δ%d", p.Delta))
	}
	parts = append(parts, p.Tokens...)
	return strings.Join(parts, " ")
}

// FormatRanges writes ranges as `a-b,c`.
func FormatRanges(rs []Range) string {
	var b strings.Builder
	for i, r := range rs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(r.From))
		if r.To != r.From {
			b.WriteString("-" + strconv.Itoa(r.To))
		}
	}
	return b.String()
}

// ParseRanges reads `a[-b](,c[-d])*` (spaces after a comma allowed); ok is false on anything else.
func ParseRanges(s string) ([]Range, bool) {
	var out []Range
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		from, to, hasTo := strings.Cut(part, "-")
		a, err := strconv.Atoi(from)
		if err != nil || a < 0 {
			return nil, false
		}
		b := a
		if hasTo {
			if b, err = strconv.Atoi(to); err != nil {
				return nil, false
			}
		}
		out = append(out, Range{a, b})
	}
	return out, len(out) > 0
}
