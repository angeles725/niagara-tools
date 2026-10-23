// Package sessionstore is lint-session-store-lazy-evict: a static session/token Map whose `.put(` has no
// sweep in the inserting method and no scheduled purge. Port of the awk in
// toolbelt/lint-session-store-lazy-evict.sh (the wrapper keeps the user-facing contract and its header
// documents the rule and its mutation pins). The truth-audit T5 fixes are listed in the wrapper header.
package sessionstore

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-session-store-lazy-evict"

// Spec registers the rule with the argument contract of toolbelt/lint-session-store-lazy-evict.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:  rule{},
		Mode:  lint.First,
		Usage: "usage: lint-session-store-lazy-evict.sh [--strict] <src-root>",
		// srcTest is pruned: the rule guards production session stores, and test sources never ship in
		// the module jar (truth audit T5: InventoryPlanTest, FordDispatchTest).
		Select: lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest"}},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
		EmptyError:          lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	// fieldDecl is a `static` Map field declaration: modifiers, a (java.util.) Map type whose generics
	// nest up to three levels, the name, then `=` or `;` (never `(` or `,`: a method or a parameter).
	fieldDecl = regexp.MustCompile(`(?:^|[^A-Za-z0-9_$.])static(?:\s+(?:final|volatile|transient|private|public|protected))*` +
		`\s+(?:java\.util\.(?:concurrent\.)?)?(?:Concurrent|ConcurrentSkipList|Linked|Weak|Identity|Tree|Sorted|Navigable)?(?:Hash)?Map` +
		`\s*<(?:[^<>]|<(?:[^<>]|<[^<>]*>)*>)*>\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*[=;]`)
	ident     = rx(`[A-Za-z_][A-Za-z0-9_]*`)
	schedHead = rx(`^.*Clock\.schedule[A-Za-z]*[[:space:]]*\(`)
)

const msg = "' inserted here with no sweep in the SAME method (no scheduled purge found either) -- add a" +
	" scheduled/size-bounded eviction, or bind auth to user+issue-time rather than the bare container id"

// Check reports, per class-level static Map field (in declaration order, each name once) that no
// scheduled action purges, every method that puts into it (a row at its first put) and does not sweep it.
// A put outside a method (a static initializer: a constant table) is never checked.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	// Every match reads comment- and literal-blanked code: a "//" in a string cuts nothing, and a
	// put, sweep or schedule spelled in a comment or a string is not one.
	code, _ := javasrc.BlankLiterals(f.Stripped(), nil)
	methods := f.Methods()
	fields := staticFields(code, methods, staticInits(code))
	if len(fields) == 0 {
		return nil, nil
	}
	var out []lint.Finding
	for _, name := range fields {
		put := use(name, `\.\s*put\s*\(`)
		if scheduledPurge(code, methods, name) {
			continue
		}
		for _, m := range methods {
			body := code[m.Start-1 : m.End]
			for i, ln := range body {
				if put.MatchString(ln) {
					if !sweeps(body, name) {
						out = append(out, lint.Finding{Line: m.Start + i, Msg: "static session/token map '" + name + msg})
					}
					break
				}
			}
		}
	}
	return out, nil
}

// use matches name as a whole identifier followed by the regexp tail.
func use(name, tail string) *regexp.Regexp {
	return regexp.MustCompile(`(?:^|[^A-Za-z0-9_$])` + regexp.QuoteMeta(name) + `\s*` + tail)
}

// sweeps reports whether a method body evicts from the named map: name.remove(, a removeIf or remove(
// on one of its views, or an iterator over it (or a view) together with a no-argument .remove().
func sweeps(body []string, name string) bool {
	view := `(?:\.\s*(?:keySet|values|entrySet)\s*\(\s*\)\s*)?`
	remove, iter := use(name, view+`\.\s*(?:remove|removeIf)\s*\(`), use(name, view+`\.\s*iterator\s*\(`)
	var iterated, removed bool
	for _, ln := range body {
		if remove.MatchString(ln) {
			return true
		}
		iterated = iterated || iter.MatchString(ln)
		removed = removed || itRemove.MatchString(ln)
	}
	return iterated && removed
}

var itRemove = regexp.MustCompile(`\.\s*remove\s*\(\s*\)`)

// scheduledPurge reports whether a Clock.schedule* call names an action (one hop: the method declared as
// do<Action>() or <action>()) whose body sweeps the named map. The arguments are read up to the call's
// ';', at most three lines.
func scheduledPurge(lines []string, methods []javasrc.Method, name string) bool {
	for i, ln := range lines {
		if !strings.Contains(ln, "Clock.schedule") {
			continue
		}
		call := ""
		for i2 := i; i2 <= i+2 && i2 < len(lines); i2++ {
			call += " " + lines[i2]
			if strings.Contains(lines[i2], ";") {
				break
			}
		}
		if loc := schedHead.FindStringIndex(call); loc != nil {
			call = call[loc[1]:]
		}
		if k := strings.IndexByte(call, ';'); k >= 0 {
			call = call[:k]
		}
		for _, act := range ident.FindAllString(call, -1) {
			do := "do" + strings.ToUpper(act[:1]) + act[1:]
			for _, m := range methods {
				if (m.Name == do || m.Name == act) && sweeps(lines[m.Start-1:m.End], name) {
					return true
				}
			}
		}
	}
	return false
}

// staticFields is every class-level static Map field name (outside each method and static initializer),
// first declaration first, each name once. It reads comment- and literal-blanked code.
func staticFields(code []string, methods []javasrc.Method, init []bool) []string {
	skip := append([]bool(nil), init...)
	for _, m := range methods {
		for i := m.Start - 1; i < m.End && i < len(skip); i++ {
			skip[i] = true
		}
	}
	var names []string
	seen := map[string]bool{}
	for i, ln := range code {
		if skip[i] || !strings.Contains(ln, "static") {
			continue
		}
		for _, m := range fieldDecl.FindAllStringSubmatch(ln, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				names = append(names, m[1])
			}
		}
	}
	return names
}

// staticInits marks the lines of every `static { ... }` initializer (to the file end if it never closes)
// in blanked code.
func staticInits(code []string) []bool {
	in := make([]bool, len(code))
	depth, open, start, word := 0, -1, 0, ""
	for i, ln := range code {
		for k := 0; k < len(ln); k++ {
			switch c := ln[k]; {
			case identByte(c):
				j := k
				for j < len(ln) && identByte(ln[j]) {
					j++
				}
				word, k = ln[k:j], j-1
			case c == '{':
				depth++
				if open < 0 && word == "static" {
					open, start = depth, i
				}
				word = ""
			case c == '}':
				if open == depth {
					for l := start; l <= i; l++ {
						in[l] = true
					}
					open = -1
				}
				depth--
				word = ""
			case !strings.ContainsRune(" \t\r\v\f", rune(c)):
				word = ""
			}
		}
	}
	for l := start; open >= 0 && l < len(in); l++ {
		in[l] = true
	}
	return in
}

func identByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
