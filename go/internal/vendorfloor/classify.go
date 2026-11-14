package vendorfloor

import (
	"fmt"
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// position is the " (line:col)" acorn appends to a parse error; the row already carries the line.
var position = regexp.MustCompile(` \(\d+:\d+\)$`)

// classify turns what the driver found in one file into rows, in report order: the syntax row first,
// then one row per use of an API above the floor, in token order. path is the row path.
// --strict promotes every WARN row to FAIL.
func classify(rec record, path string, apis APIs, strict bool) []lint.Finding {
	var out []lint.Finding
	add := func(sev lint.Severity, line int, msg string) {
		if strict && sev == lint.Warn {
			sev = lint.Fail
		}
		out = append(out, lint.Finding{Rule: rowID, Severity: sev, Path: path, Line: line, Msg: msg})
	}
	if s := rec.Syntax; s != nil {
		sev, msg := syntaxVerdict(*s)
		add(sev, max(s.Line, 1), msg)
	}
	for _, u := range rec.Names {
		for _, call := range apiCalls(u, apis) {
			add(lint.Warn, u.Line, call)
		}
	}
	return out
}

// syntaxVerdict rates a file that is not a classic script at the floor: it breaks the whole page
// (FAIL) unless it is an ES module at the floor, which only needs another way of loading (WARN).
func syntaxVerdict(s syntaxError) (lint.Severity, string) {
	if s.Module {
		return lint.Warn, fmt.Sprintf(moduleOnly, s.Message)
	}
	return lint.Fail, fmt.Sprintf(syntaxFail, FloorECMAVersion, FloorChrome, position.ReplaceAllString(s.Message, ""))
}

// apiCalls words each API above the floor that the name token u is a use of. An API throws only
// where it runs and may be feature-guarded, so every one is a WARN. Member and Qualified share no
// method name (APIs.Validate), so a member call never doubles a qualified row.
func apiCalls(u nameUse, apis APIs) []string {
	var msgs []string
	called, member := u.Next == "(", u.Prev == "."
	if chrome, ok := apis.Member[u.Name]; ok && member && called {
		msgs = append(msgs, apiMsg("."+u.Name+"(", chrome))
	}
	if chrome, ok := apis.Qualified[u.Prev2+"."+u.Name]; ok && member && called {
		msgs = append(msgs, apiMsg(u.Prev2+"."+u.Name+"(", chrome))
	}
	if chrome, ok := apis.GlobalCall[u.Name]; ok && !member && called {
		msgs = append(msgs, apiMsg(u.Name+"(", chrome))
	}
	if chrome, ok := apis.GlobalNew[u.Name]; ok && u.Prev == "new" {
		msgs = append(msgs, apiMsg("new "+u.Name, chrome))
	}
	return msgs
}
