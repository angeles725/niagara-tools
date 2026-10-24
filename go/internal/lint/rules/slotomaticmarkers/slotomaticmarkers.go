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
package slotomaticmarkers

import (
	"fmt"
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

// Check reports every closed region whose stamp count is not 1, on its BEGIN line (its END line when no
// BEGIN came first), and a BEGIN left open at the end of the file.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
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
			open = false
		}
	}
	if open {
		out = append(out, lint.Finding{Check: "unclosed-region", Severity: lint.Warn, Line: start + 1,
			Msg: javasrc.RegionBegin + " with no END marker after it -- Slot-o-Matic reads no region here"})
	}
	return out, nil
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
