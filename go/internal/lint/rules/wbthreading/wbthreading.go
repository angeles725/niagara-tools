// Package wbthreading is lint-wb-threading: two Workbench checks over a -wb src tree (B809 §809.7).
// ui-thread-traversal: a doInvoke body (with the same-class private/protected methods it reaches within 3 call
// levels) calls getNavChildren|getNavNodes|BqlQuery with no invokeLater|BJobService|JobThread anywhere on that
// chain; agent-breadth: an @AgentOn(baja:Component) with no justif|why|broad text within 3 lines above. Port of
// the two awk programs of toolbelt/lint-wb-threading.sh (the wrapper keeps the user-facing contract and its
// header documents the rule and its mutation pins).
//
// It is a per-file lint.Rule: the traversal rows of a file, then its agent rows, each row check first
// (Spec.CheckFirst). The awk read raw lines and visited a callee set with `for (k in a)`, so the call chain a
// row names follows gawk's hash order (rules/awkorder).
package wbthreading

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/awkorder"
)

// ID is the lint name n4lint accepts.
const ID = "lint-wb-threading"

// Spec registers the rule with the argument contract of toolbelt/lint-wb-threading.sh: --strict before or after
// the root (Mode Anywhere), check-first rows, a fail-closed walk and the K20 ERROR row.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Anywhere,
		Usage:      "usage: lint-wb-threading.sh [--strict] <wb-src-dir>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		CheckFirst: true,
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	helperDecl = regexp.MustCompile(`^[[:space:]]+(private|protected)[[:space:]]`)
	helperName = regexp.MustCompile(`^[a-z][a-zA-Z0-9_]*$`)
	doInvoke   = regexp.MustCompile(`[[:space:]]doInvoke[[:space:]]*\(`)
	nav        = regexp.MustCompile(`getNavChildren|getNavNodes|BqlQuery`)
	guard      = regexp.MustCompile(`invokeLater|BJobService|JobThread`)
	nonIdent   = regexp.MustCompile(`[^a-zA-Z_0-9]`)
	idWord     = regexp.MustCompile(`^[a-z][a-zA-Z0-9_]+$`)
	justified  = regexp.MustCompile(`justif|why|broad`)
)

const (
	traversal = "doInvoke -> %s without invokeLater/BJobService (B809: flag for human review)"
	breadth   = "@AgentOn(baja:Component) without justification comment within 3 lines above (B809)"
	depth     = 3 // call levels expanded below doInvoke
)

// Check runs the traversal awk, then the agent awk, over the file's raw lines.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	return append(traversals(f.Raw), agents(f.Raw)...), nil
}

// traversals is the thread awk: phase 1 collects the private/protected method bodies, phase 2 checks each
// doInvoke body expanded through them.
func traversals(lines []string) []lint.Finding {
	mb := map[string]string{}
	for i, ln := range lines { // phase 1
		if !helperDecl.MatchString(ln) {
			continue
		}
		before, _, found := strings.Cut(ln, "(")
		if !found {
			continue
		}
		words := strings.FieldsFunc(before, awkBlank)
		if len(words) == 0 || !helperName.MatchString(words[len(words)-1]) {
			continue
		}
		mb[words[len(words)-1]], _ = braceBody(lines, i)
	}
	var out []lint.Finding
	for i := 0; i < len(lines); i++ { // phase 2
		if !doInvoke.MatchString(lines[i]) {
			continue
		}
		body, end := braceBody(lines, i)
		expanded := expand(mb, body, depth, map[string]bool{"doInvoke": true})
		if nav.MatchString(expanded) && !guard.MatchString(expanded) {
			ch := chain(mb, body, depth, map[string]bool{})
			if ch == "?" {
				ch = "..."
			}
			out = append(out, lint.Finding{Check: "ui-thread-traversal", Line: i + 1, Msg: strings.Replace(traversal, "%s", ch, 1)})
		}
		i = end
	}
	return out
}

// braceBody is the awk brace_body(): the lines from start ("\n" before each) until the brace count, once a
// "{" was seen, drops to zero or below; end is the 0-based index of the last line read (len(lines) when the
// body never closes, as the awk's g_end = NR+1).
func braceBody(lines []string, start int) (body string, end int) {
	var b strings.Builder
	depth, seen := 0, false
	j := start
	for ; j < len(lines); j++ {
		b.WriteString("\n")
		b.WriteString(lines[j])
		depth += strings.Count(lines[j], "{") - strings.Count(lines[j], "}")
		if strings.Contains(lines[j], "{") {
			seen = true
		}
		if seen && depth <= 0 {
			break
		}
	}
	return b.String(), j
}

// ids is the awk get_ids(): the lowercase-first identifiers of src (two characters or more), in the order a
// gawk `for (k in a)` visits them after inserting them in first-occurrence order.
func ids(src string) []string {
	var keys []string
	seen := map[string]bool{}
	for _, w := range strings.FieldsFunc(nonIdent.ReplaceAllString(src, " "), awkBlank) {
		if idWord.MatchString(w) && !seen[w] {
			seen[w] = true
			keys = append(keys, w)
		}
	}
	return awkorder.Order(keys)
}

// expand is the awk expand_body(): body plus the bodies of the same-class callees reachable within d levels;
// vis is shared by the whole expansion (cycle-safe).
func expand(mb map[string]string, body string, d int, vis map[string]bool) string {
	if d <= 0 {
		return body
	}
	for _, callee := range ids(body) {
		if m, ok := mb[callee]; ok && !vis[callee] {
			vis[callee] = true
			body += "\n" + expand(mb, m, d-1, vis)
		}
	}
	return body
}

// chain is the awk chain_from(): "getNavChildren" when body holds a nav call (whichever one), the first
// callee path to one ("helper() -> getNavChildren"), or "?".
func chain(mb map[string]string, body string, d int, vis map[string]bool) string {
	if nav.MatchString(body) {
		return "getNavChildren"
	}
	if d <= 0 {
		return "?"
	}
	for _, callee := range ids(body) {
		if m, ok := mb[callee]; ok && !vis[callee] {
			vis[callee] = true
			if c := chain(mb, m, d-1, vis); c != "?" {
				return callee + "() -> " + c
			}
		}
	}
	return "?"
}

// agents is the agent awk: an @AgentOn line naming baja:Component with no justif|why|broad on the 3 lines above.
func agents(lines []string) []lint.Finding {
	var out []lint.Finding
	for i, ln := range lines {
		if !strings.Contains(ln, "@AgentOn") || !strings.Contains(ln, "baja:Component") {
			continue
		}
		found := false
		for k := i - 1; k >= i-3 && k >= 0; k-- {
			if justified.MatchString(lines[k]) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, lint.Finding{Check: "agent-breadth", Line: i + 1, Msg: breadth})
		}
	}
	return out
}

// awkBlank is awk's default field separator: space, tab and newline.
func awkBlank(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }
