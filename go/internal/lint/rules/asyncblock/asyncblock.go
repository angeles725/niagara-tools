// Package asyncblock is lint-async-blocking (ledger G28, course F3/B2: corpus B1238 §1238.3-§1238.5, B1254 :128).
// An action declared with Flags.ASYNC runs on the ONE engine thread every timer callback and every other ASYNC
// action in the station shares; a do<Name> that blocks there stalls them all. The sanctioned offload is an ASYNC
// action whose post(Action, ...) override hands `new Invocation(this, action, arg, cx)` to the module's own
// executor, which calls invoke() on it. toolbelt/lint-async-blocking.sh is the wrapper and its header is the
// user-facing contract.
//
// A per-file rule over *.java read as javasrc tokens (comments blanked, a literal one token). WARN only (never
// FAIL: a plain ASYNC handler is fine for short bounded work, and the course exam accepted an ASYNC history
// export). Rows: an ASYNC action of a class with NO post(Action...) override whose do<Name> body calls
// Thread.sleep(, openConnection(, getOutputStream(, sendSync(, names HttpClient, or calls .get( with a "bql:"
// string literal in the body (a BQL ord resolved on the engine thread); and a post(Action...) override that runs
// the handler directly (`this::doX` or `-> doX(`), which skips the action pipeline. Only `flags = ... Flags.ASYNC`
// is read (a numeric flags value is not), and a do method inherited from another file is not seen.
package asyncblock

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/actionhandler"
)

// ID is the lint name printed in every row.
const ID = "lint-async-blocking"

// Spec registers the rule with the argument contract of toolbelt/lint-async-blocking.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Each,
		Usage:  "usage: lint-async-blocking.sh [--strict] <src-root>...",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	fix   = "types/actions.md § 2"
	block = "blocks the shared engine thread (%s) and the class has no post(Action...) override: keep a plain ASYNC " +
		"handler for short bounded work only; otherwise override post() and hand new Invocation(this, %s, arg, cx) to " +
		"your own executor (" + fix + ")"
	direct = "post() runs %s on another thread: the handler skips the action pipeline (no links, action event, audit " +
		"or Context); pass new Invocation(this, <action>, arg, cx) to the executor and invoke it there (" + fix + ")"
)

// Check reports each finding on the do method's line (blocking) or on the reference's line (post), in line order.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	decls := javasrc.Decls(toks)
	var out []lint.Finding
	for _, c := range javasrc.Classes(toks) {
		var posts []javasrc.Decl
		dos := map[string]javasrc.Decl{}
		for _, d := range decls {
			switch {
			case d.Class != c.Name || d.Anonymous:
			case d.Name == "post" && len(d.Params) > 0 && d.Params[0][strings.LastIndex(d.Params[0], ".")+1:] == "Action":
				posts = append(posts, d)
			case len(d.Name) > 2 && d.Name[:2] == "do" && unicode.IsUpper(rune(d.Name[2])):
				dos[d.Name] = d
			}
		}
		for _, p := range posts {
			out = append(out, directRows(toks, p)...)
		}
		if len(posts) > 0 {
			continue
		}
		for _, a := range actionhandler.Actions(toks, c) {
			if a.Name == "" || !a.HasFlag("ASYNC") {
				continue
			}
			d, ok := dos["do"+string(unicode.ToUpper(rune(a.Name[0])))+a.Name[1:]]
			if !ok {
				continue
			}
			if hits := blocking(toks, d.Body); len(hits) > 0 {
				out = append(out, lint.Finding{Severity: lint.Warn, Line: d.Line + 1,
					Msg: fmt.Sprintf("ASYNC action \"%s\": %s "+block, a.Name, d.Name, strings.Join(hits, ", "), a.Name)})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out, nil
}
