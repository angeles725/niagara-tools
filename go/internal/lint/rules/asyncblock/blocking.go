package asyncblock

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// call are the methods whose bare call blocks: a sleep, a network or file stream, a synchronous send.
var call = map[string]string{"openConnection": "openConnection()", "getOutputStream": "getOutputStream()", "sendSync": "sendSync()"}

// blocking lists the blocking constructs in the body toks[body[0]:body[1]] as "label line N", one per label,
// in source order.
func blocking(toks []javasrc.Token, body [2]int) []string {
	var hits []string
	seen := map[string]bool{}
	add := func(label string, tok int) {
		if !seen[label] {
			seen[label] = true
			hits = append(hits, fmt.Sprintf("%s line %d", label, toks[tok].Line+1))
		}
	}
	bql := false
	for i := body[0]; i < body[1]; i++ {
		if toks[i].Kind == javasrc.TokLiteral && strings.Contains(strings.ToLower(toks[i].Text), "bql:") {
			bql = true
		}
	}
	for i := body[0]; i < body[1]; i++ {
		t := toks[i]
		next := func(k int, s string) bool { return i+k < body[1] && toks[i+k].Text == s }
		switch {
		case t.Text == "Thread" && next(1, ".") && next(2, "sleep") && next(3, "("):
			add("Thread.sleep()", i+2)
		case t.Text == "HttpClient":
			add("HttpClient", i)
		case call[t.Text] != "" && next(1, "(") && i > body[0]:
			add(call[t.Text], i)
		case bql && t.Text == "get" && next(1, "(") && i > body[0] && toks[i-1].Text == ".":
			add("BQL .get()", i)
		}
	}
	return hits
}

// directRows reports each `this::doX` and `-> doX(` (or `-> { doX(`) inside a post(Action...) override.
func directRows(toks []javasrc.Token, p javasrc.Decl) []lint.Finding {
	var out []lint.Finding
	isDo := func(s string) bool { return len(s) > 2 && s[:2] == "do" && unicode.IsUpper(rune(s[2])) }
	row := func(label string, line int) {
		out = append(out, lint.Finding{Severity: lint.Warn, Line: line + 1, Msg: fmt.Sprintf(direct, label)})
	}
	for i := p.Body[0]; i < p.Body[1]; i++ {
		switch {
		case i+3 < p.Body[1] && toks[i].Text == "this" && toks[i+1].Text == ":" && toks[i+2].Text == ":" && isDo(toks[i+3].Text):
			row("this::"+toks[i+3].Text, toks[i+3].Line)
		case i+3 < p.Body[1] && toks[i].Text == "-" && toks[i+1].Text == ">":
			k := i + 2
			if toks[k].Text == "{" {
				k++
			}
			if k+1 < p.Body[1] && isDo(toks[k].Text) && toks[k+1].Text == "(" {
				row("() -> "+toks[k].Text+"(...)", toks[k].Line)
			}
		}
	}
	return out
}
