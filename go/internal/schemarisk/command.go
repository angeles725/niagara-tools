package schemarisk

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// Exit codes of schema-risk: the verdict scale, with usage and env ABOVE it so a bad invocation never reads
// as OUTAGE in a deploy script (D4a; unchanged from the bash tool).
const (
	ExitSafe   = 0
	ExitLossy  = 1
	ExitOutage = 2
	ExitUsage  = 3
	ExitEnv    = 4
)

// Main implements `schema-risk <before-dir> <after-dir>` (further arguments are ignored, as in bash).
func Main(args []string, s cli.Stdio) int {
	if len(args) < 2 {
		fmt.Fprint(s.Err, "usage: schema-risk.sh <before-dir> <after-dir>\n")
		return ExitUsage
	}
	before, after := args[0], args[1]
	typeName := ""
	for i, dir := range []string{before, after} {
		path := dir + "/module-include.xml"
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(s.Err, "error: cannot read %s\n", path)
			return ExitEnv
		}
		doc, err := xmlsrc.Parse(path, b)
		if err != nil {
			fmt.Fprintf(s.Err, "error: %v\n", err)
			return ExitEnv
		}
		if i == 0 {
			typeName = firstTypeName(doc)
		}
	}
	var snaps [2][]Slot
	var unknown []Change
	for i, dir := range []string{before, after} {
		files, err := lint.Walk(dir, lint.Selection{Names: []string{"*.java"}})
		if err != nil {
			fmt.Fprintf(s.Err, "error: cannot list every file under %s: %v\n", dir, err)
			return ExitEnv
		}
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				// a partial read is never classified: the bash awk stopped here and judged the slots read before it
				fmt.Fprintf(s.Err, "error: cannot read %s: %s\n", f, strerror.Text(err))
				return ExitEnv
			}
			slots, bad := Parse(lines(string(b)))
			snaps[i] = append(snaps[i], slots...)
			for _, ln := range bad {
				unknown = append(unknown, Change{"UNKNOWN", fmt.Sprintf("<unparsed>@%s:%d", f, ln)})
			}
		}
	}
	worst, verdict := 0, "SAFE"
	for _, c := range append(Diff(snaps[0], snaps[1]), unknown...) {
		r := lookup(c.Kind)
		fmt.Fprintf(s.Out, "%s  %s  %s.%s: %s (%s)\n", r.Verdict, c.Kind, typeName, c.Label, r.Evidence, r.Note)
		if sc := score(r.Verdict); sc > worst {
			worst, verdict = sc, r.Verdict
		}
	}
	fmt.Fprintf(s.Out, "verdict=%s\n", verdict)
	return worst
}

// firstTypeName is the first non-empty name of a <type> element (a <dependency> or a comment never names the
// module's type), or "Module" when there is none.
func firstTypeName(doc *xmlsrc.Doc) string {
	for _, i := range doc.Find("type") {
		if n, _ := doc.Elements()[i].Attr("name"); n != "" {
			return n
		}
	}
	return "Module"
}

// lines splits text into awk records: newline-terminated, a last unterminated line kept.
func lines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}
