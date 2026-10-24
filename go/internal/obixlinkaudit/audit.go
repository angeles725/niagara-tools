// Package obixlinkaudit diffs the source column of a wiring map (generate-wiring-map.sh) against the
// links into one component, read from an oBIX response: obix-link-audit, run through
// toolbelt/obix-link-audit.sh. Rows, summary and exit codes are the bash tool's.
package obixlinkaudit

import (
	"fmt"
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// Declared is one wiring-map row of the selected table.
type Declared struct {
	Slot  string // the audited slot on the component ("" when the map leaves it undeclared)
	Want  string // the declared source slot ("" when undeclared)
	Label string // the facade slot, named on a SKIP row
	Comp  string // the declared source component (Table 2 only; "" compares the slot alone)
}

// Link is one link into the component: target slot <- source slot of source component.
type Link struct{ Target, Source, Comp string }

// trim drops every backtick, then leading and trailing blanks and tabs.
func trim(s string) string { return strings.Trim(strings.ReplaceAll(s, "`", ""), " \t") }

// lastSegment is the slot of a declared source: the text after its last '/' or '.'.
func lastSegment(s string) string { return s[strings.LastIndexAny(s, "/.")+1:] }

// component is the path before the slot (Plant/Rack.comp1Running -> Plant/Rack); "" for a bare slot.
func component(s string) string {
	s = strings.ReplaceAll(strings.TrimPrefix(s, "slot:"), ".", "/")
	i := strings.LastIndex(s, "/")
	if i < 0 {
		return ""
	}
	return strings.TrimLeft(s[:i], "/")
}

// DeclaredRows reads the rows of "## Table <table> " sections. Table 2: the audited slot is the facade
// slot (column 1), fed by the RT slot (column 2); Table 1: the audited slot is the RT slot, fed by the
// facade slot, compared by slot alone.
func DeclaredRows(mapText, table string) []Declared {
	var rows []Declared
	in := false
	for _, line := range strings.Split(mapText, "\n") {
		if strings.HasPrefix(line, "## ") {
			in = strings.HasPrefix(line, "## Table "+table+" ")
			continue
		}
		cells := strings.Split(line, "|")
		if !in || len(cells) < 4 {
			continue
		}
		f, r := trim(cells[1]), trim(cells[2])
		if f == "" || strings.Trim(f, "-") == "" || f == "Facade slot" || strings.HasPrefix(f, "_(") {
			continue
		}
		d := Declared{Label: f}
		declared := r != "" && !strings.HasPrefix(r, "_(")
		switch {
		case table == "2" && declared:
			d.Slot, d.Want, d.Comp = f, lastSegment(r), component(r)
		case table == "2":
			d.Slot = f
		case declared:
			d.Slot, d.Want = lastSegment(r), f
		default:
			d.Want = f
		}
		rows = append(rows, d)
	}
	return rows
}

// Links reads every element whose display is "Indirect: <src> -> <tgt>" or "Direct: ..." (the arrow
// as →, &#x2192; or ->), in document order; a display without an arrow is no link.
func Links(doc *xmlsrc.Doc) []Link {
	var links []Link
	for _, e := range doc.Elements() {
		v, _ := e.Attr("display")
		rest, ok := strings.CutPrefix(v, "Indirect: ")
		if !ok {
			if rest, ok = strings.CutPrefix(v, "Direct: "); !ok {
				continue
			}
		}
		src, tgt, ok := strings.Cut(strings.ReplaceAll(rest, "→", "->"), "->")
		if !ok {
			continue
		}
		src, tgt = strings.Trim(src, " \t"), strings.Trim(tgt, " \t")
		c := src
		if i := strings.LastIndex(src, "."); i >= 0 {
			c = src[:i]
		}
		links = append(links, Link{Target: tgt[strings.LastIndex(tgt, ".")+1:], Source: src[strings.LastIndex(src, ".")+1:], Comp: c})
	}
	return links
}

// inComponent: the actual source component path ends in the declared one on a whole path segment.
func inComponent(actual, want string) bool {
	actual = strings.ReplaceAll(strings.TrimLeft(strings.TrimPrefix(actual, "slot:"), "/"), ".", "/")
	return actual == want || strings.HasSuffix(actual, "/"+want)
}

// Report prints one row per declared row and the summary line; it returns 1 on any MISMATCH or
// MISSING row, else 0.
func Report(w io.Writer, rows []Declared, links []Link) int {
	into := map[string][]Link{}
	for _, l := range links {
		into[l.Target] = append(into[l.Target], l)
	}
	var match, mismatch, missing, skip int
	for _, d := range rows {
		desc := d.Want
		if d.Comp != "" {
			desc += " in " + d.Comp
		}
		got, linked := into[d.Slot]
		switch {
		case d.Want == "" || d.Slot == "":
			fmt.Fprintf(w, "SKIP  obix-link-audit  %s  source not declared in the wiring map (fill it before commissioning)\n", d.Label)
			skip++
		case !linked:
			fmt.Fprintf(w, "MISSING  obix-link-audit  %s  declared %s, no link into %s\n", d.Slot, desc, d.Slot)
			missing++
		case matches(d, got):
			fmt.Fprintf(w, "MATCH  obix-link-audit  %s  linked from declared %s\n", d.Slot, desc)
			match++
		default:
			from := make([]string, len(got))
			for i, l := range got {
				from[i] = l.Source + " (" + l.Comp + ")"
			}
			fmt.Fprintf(w, "MISMATCH  obix-link-audit  %s  declared %s, linked from %s\n", d.Slot, desc, strings.Join(from, ", "))
			mismatch++
		}
	}
	verdict, code := "CLEAN", 0
	if mismatch+missing > 0 {
		verdict, code = "ISSUES", 1
	}
	fmt.Fprintf(w, "obix-link-audit: %d MATCH · %d MISMATCH · %d MISSING · %d SKIP  ->  %s\n", match, mismatch, missing, skip, verdict)
	return code
}

// matches: a link from the declared slot and, when the map names one, the declared component.
// Mutation: OLA-comp -- comparing the slot alone lets the right slot of the wrong instance MATCH.
func matches(d Declared, got []Link) bool {
	for _, l := range got {
		if l.Source == d.Want && (d.Comp == "" || inComponent(l.Comp, d.Comp)) {
			return true
		}
	}
	return false
}
