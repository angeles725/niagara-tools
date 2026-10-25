package lint

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// XMLWarnings are doc's warnings (xmlsrc.Doc.Warnings: what Niagara reads although XML 1.0 forbids it) as Warn
// findings at their line, the column in the message, so every rule that reads a .bog reports them alike.
func XMLWarnings(doc *xmlsrc.Doc) []Finding {
	var out []Finding
	for _, w := range doc.Warnings() {
		out = append(out, Finding{Severity: Warn, Line: w.Line, Msg: fmt.Sprintf("col %d: %s", w.Col, w.Msg)})
	}
	return out
}
