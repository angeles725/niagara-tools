// Package slotomaticmarkers is the lint-slotomatic-markers rule (kit lint G3, METHODOLOGY.md K15): a
// Slot-o-Matic generated region (javasrc.RegionBegin … javasrc.RegionEnd, markers read by File.Marker,
// so a marker quoted in a literal is code) must hold exactly one `/*@ $<fqcn>(<crc>)1.0$ @*/` stamp line.
// Slot-o-Matic counts the region's lines whose Java trim() starts with "/*@" and throws "Found multiple
// metadata blocks" on any count other than one (devkit-wb vineflower 4.14.0.162
// com/tridium/slottool/JavaUnit.java:270-272); Compiler logs "Could not process <file>" and the task
// succeeds (Compiler.java:240-246), so the file is skipped while the build stays green until compileJava.
// The region is read as JavaUnit reads it (:479-497): a BEGIN marker sets the start, an END marker closes
// the region from that start (line 1 when no BEGIN came first). Declared stricter than Slot-o-Matic, which
// reads only the last region and only a marker comment of the exact text: every closed region is checked
// and a comment holding the marker text is a marker. A BEGIN with no later END is a WARN (unclosed-region).
//
// Sub-check type-outside-region (FAIL, retro r10-client-gate-retro Δ1): a @NiagaraType class that holds a
// BEGIN … END region and also declares a static TYPE field or a no-argument getType() method directly in its
// body outside that region. Slot-o-Matic writes both members into the region, so the hand-written one is a
// duplicate and compileJava fails ("variable TYPE is already defined"). A class with no region is left to
// the build-verify.md "Reference sources" rule (run slotomatic first); a nested type, a local variable, an
// anonymous class and a mention in a comment or a literal are not members of the annotated class.
package slotomaticmarkers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-slotomatic-markers"

// Spec registers the rule with the argument contract of toolbelt/lint-slotomatic-markers.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Each,
		Usage:  "usage: lint-slotomatic-markers.sh [--strict] <src-root>...",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

const msg = "generated region holds %d /*@ $...$ @*/ stamp lines, Slot-o-Matic needs exactly 1: it logs" +
	" \"Could not process <file>: Found multiple metadata blocks\", skips the file and the build stays green" +
	" -- delete the BEGIN/END markers and rerun :<part>:slotomatic"

const dupMsg = "hand-written %s outside the generated region of %s: Slot-o-Matic writes its own there, so" +
	" compileJava fails (%s is already defined) -- delete the hand-written one and keep the region"

// Check reports every closed region whose stamp count is not 1, on its BEGIN line (its END line when no
// BEGIN came first), a BEGIN left open at the end of the file, and every type-outside-region member.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	var regions [][2]int // closed BEGIN … END regions, 0-based lines
	start, open, begun := 0, false, false
	for i := range f.Raw {
		switch {
		case f.Marker(i, javasrc.RegionBegin):
			start, open, begun = i, true, true
		case f.Marker(i, javasrc.RegionEnd):
			if n := stamps(f.Raw[start : i+1]); n != 1 {
				row := start
				if !begun {
					row = i
				}
				out = append(out, lint.Finding{Line: row + 1, Msg: fmt.Sprintf(msg, n)})
			}
			if begun {
				regions = append(regions, [2]int{start, i})
			}
			open = false
		}
	}
	if open {
		out = append(out, lint.Finding{Check: "unclosed-region", Severity: lint.Warn, Line: start + 1,
			Msg: javasrc.RegionBegin + " with no END marker after it -- Slot-o-Matic reads no region here"})
	}
	if len(regions) > 0 {
		out = append(out, outsideRegion(javasrc.Tokens(f.Stripped()), regions)...)
		sort.SliceStable(out, func(a, b int) bool { return out[a].Line < out[b].Line })
	}
	return out, nil
}

// outsideRegion is the type-outside-region rows: for each @NiagaraType class that owns a region (the
// innermost class whose body holds the region's BEGIN line), its static TYPE fields and no-argument
// getType() methods declared outside every region it owns.
func outsideRegion(toks []javasrc.Token, regions [][2]int) []lint.Finding {
	classes := javasrc.Classes(toks)
	owned := make([][][2]int, len(classes))
	for _, r := range regions {
		owner := -1
		for ci, c := range classes {
			if c.Body[1] < len(toks) && toks[c.Body[0]-1].Line < r[0] && r[0] < toks[c.Body[1]].Line {
				owner = ci // Classes is in source order, so a nested class comes after its enclosing one
			}
		}
		if owner >= 0 {
			owned[owner] = append(owned[owner], r)
		}
	}
	inside := func(rs [][2]int, line int) bool {
		for _, r := range rs {
			if r[0] < line && line < r[1] {
				return true
			}
		}
		return false
	}
	var out []lint.Finding
	decls := javasrc.Decls(toks)
	for ci, c := range classes {
		if len(owned[ci]) == 0 || !javasrc.Annotated(toks, c, "NiagaraType") {
			continue
		}
		for _, fd := range javasrc.Fields(toks, c.Body) {
			if fd.Static && fd.Name == "TYPE" && !inside(owned[ci], fd.Line) {
				out = append(out, lint.Finding{Check: "type-outside-region", Line: fd.Line + 1,
					Msg: fmt.Sprintf(dupMsg, "TYPE", c.Name, "variable TYPE")})
			}
		}
		for _, d := range decls {
			if d.Name == "getType" && len(d.Params) == 0 && !d.Anonymous && d.Class == c.Name &&
				c.Body[0] <= d.Body[0] && d.Body[0] < c.Body[1] && !inside(owned[ci], d.Line) {
				out = append(out, lint.Finding{Check: "type-outside-region", Line: d.Line + 1,
					Msg: fmt.Sprintf(dupMsg, "getType()", c.Name, "method getType()")})
			}
		}
	}
	return out
}

// stamps counts the lines that start with "/*@" after Java's String.trim() (bytes <= ' ' dropped).
func stamps(lines []string) int {
	n := 0
	for _, ln := range lines {
		if strings.HasPrefix(strings.TrimLeftFunc(ln, func(r rune) bool { return r <= ' ' }), "/*@") {
			n++
		}
	}
	return n
}
