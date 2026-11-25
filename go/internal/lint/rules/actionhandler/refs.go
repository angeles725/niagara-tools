package actionhandler

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const skipsPipeline = " runs the handler outside the action pipeline (no links, action event, audit or Context): " +
	"post new Invocation(this, <action>, arg, cx) from an ASYNC post() override and invoke it instead (types/actions.md § 2)"

// refRows reports `this::doX` and `new Thread(() -> doX(` for each action X declared on c (course F3), outside the
// body of a post(Action...) override (lint-async-blocking judges that) and outside a nested class.
func refRows(toks []javasrc.Token, classes []javasrc.Class, c javasrc.Class, decls []javasrc.Decl) []lint.Finding {
	names := map[string]bool{}
	for _, a := range Actions(toks, c) {
		if a.Name != "" {
			names[doName(a.Name)] = true
		}
	}
	if len(names) == 0 {
		return nil
	}
	var skip [][2]int
	for _, k := range classes {
		if k.At > c.At && k.Body[0] >= c.Body[0] && k.Body[1] <= c.Body[1] {
			skip = append(skip, k.Body)
		}
	}
	for _, d := range decls {
		if d.Class == c.Name && !d.Anonymous && d.Name == "post" && len(d.Params) > 0 && simpleName(d.Params[0]) == "Action" {
			skip = append(skip, d.Body)
		}
	}
	var out []lint.Finding
	seen := map[int]bool{}
	add := func(i int, what string) {
		if !seen[toks[i].Line] {
			seen[toks[i].Line] = true
			out = append(out, lint.Finding{Severity: lint.Warn, Line: toks[i].Line + 1, Msg: what + skipsPipeline})
		}
	}
	for i := c.Body[0]; i < c.Body[1]; i++ {
		if inAny(skip, i) {
			continue
		}
		if i+3 < c.Body[1] && toks[i].Text == "this" && toks[i+1].Text == ":" && toks[i+2].Text == ":" && names[toks[i+3].Text] {
			add(i+3, "this::"+toks[i+3].Text)
		}
		if i+2 < c.Body[1] && toks[i].Text == "new" && toks[i+1].Text == "Thread" && toks[i+2].Text == "(" {
			if k := lambdaTo(toks, i+2, c.Body[1], names); k > 0 {
				add(k, "new Thread(() -> "+toks[k].Text+"(...))")
			}
		}
	}
	return out
}

func inAny(ranges [][2]int, i int) bool {
	for _, r := range ranges {
		if r[0] <= i && i < r[1] {
			return true
		}
	}
	return false
}

// lambdaTo is the index of the action handler a lambda inside the call whose "(" is at open calls first
// (`-> doX(` or `-> { doX(`), or 0.
func lambdaTo(toks []javasrc.Token, open, end int, names map[string]bool) int {
	for depth, j := 0, open; j < end; j++ {
		switch toks[j].Text {
		case "(":
			depth++
		case ")":
			if depth--; depth == 0 {
				return 0
			}
		case "-":
			k := j + 2
			if j+1 < end && toks[j+1].Text == ">" && k < end && toks[k].Text == "{" {
				k++
			}
			if j+1 < end && toks[j+1].Text == ">" && k+1 < end && names[toks[k].Text] && toks[k+1].Text == "(" {
				return k
			}
		}
	}
	return 0
}
