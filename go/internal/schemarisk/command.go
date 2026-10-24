package schemarisk

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
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

var typeNameRe = regexp.MustCompile(`name="([^"]*)"`)

// Main implements `schema-risk <before-dir> <after-dir>` (further arguments are ignored, as in bash).
func Main(args []string, s cli.Stdio) int {
	if len(args) < 2 {
		fmt.Fprint(s.Err, "usage: schema-risk.sh <before-dir> <after-dir>\n")
		return ExitUsage
	}
	before, after := args[0], args[1]
	var xml string
	for i, dir := range []string{before, after} {
		b, err := os.ReadFile(dir + "/module-include.xml")
		if err != nil {
			fmt.Fprintf(s.Err, "error: cannot read %s/module-include.xml\n", dir)
			return ExitEnv
		}
		if i == 0 {
			xml = string(b)
		}
	}
	typeName := "Module"
	for _, ln := range lines(xml) {
		if m := typeNameRe.FindStringSubmatch(ln); m != nil && m[1] != "" {
			typeName = m[1]
			break
		}
	}
	var snaps [2][]Slot
	for i, dir := range []string{before, after} {
		files, err := lint.Walk(dir, lint.Selection{Names: []string{"*.java"}})
		if err != nil {
			fmt.Fprintf(s.Err, "error: cannot list every file under %s: %v\n", dir, err)
			return ExitEnv
		}
		lns, err := javaLines(files)
		if err != nil {
			fmt.Fprintf(s.Err, "error: %v\n", err)
			return ExitEnv
		}
		snaps[i] = Parse(lns)
	}
	worst, verdict := 0, "SAFE"
	for _, c := range Diff(snaps[0], snaps[1]) {
		r := lookup(c.Kind)
		fmt.Fprintf(s.Out, "%s  %s  %s.%s: %s (%s)\n", r.Verdict, c.Kind, typeName, c.Label, r.Evidence, r.Note)
		if sc := score(r.Verdict); sc > worst {
			worst, verdict = sc, r.Verdict
		}
	}
	fmt.Fprintf(s.Out, "verdict=%s\n", verdict)
	return worst
}

// javaLines is the lines of files in order; a file it cannot read fails the snapshot closed (env), never a
// verdict from the slots read before it (the bash awk stopped there and classified the partial read).
func javaLines(files []string) ([]string, error) {
	var out []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %s", f, strerror.Text(err))
		}
		out = append(out, lines(string(b))...)
	}
	return out, nil
}

// lines splits text into awk records: newline-terminated, a last unterminated line kept.
func lines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}
