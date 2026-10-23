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
		Rule:   rule{},
		Mode:   lint.First,
		Usage:  "usage: lint-session-store-lazy-evict.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
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
	signature = rx(`[A-Za-z_][A-Za-z0-9_<>\[\]]*[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([^;{}]*\)`)
)

const msg = "' inserted here with no sweep in the SAME method (no scheduled purge found either) -- add a" +
	" scheduled/size-bounded eviction, or bind auth to user+issue-time rather than the bare container id"

// Check reports, per class-level static Map field (in declaration order, each name once), its first
// `.put(` line outside a static initializer when neither the enclosing method nor a scheduled action's body removes from it.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := make([]string, len(f.Raw))
	for i, ln := range f.Raw {
		if k := strings.Index(ln, "//"); k >= 0 {
			ln = ln[:k]
		}
		lines[i] = ln
	}
	code, _ := javasrc.BlankLiterals(f.Stripped(), nil)
	init := staticInits(code)
	fields := staticFields(code, f.Methods(), init)
	if len(fields) == 0 {
		return nil, nil
	}
	purged := scheduledPurges(lines, fields)
	var out []lint.Finding
	for _, name := range fields {
		if purged[name] {
			continue
		}
		put := -1 // the first put outside a static initializer: a table filled only there is a constant
		for i := range lines {
			if !init[i] && strings.Contains(lines[i], name+".put(") {
				put = i
				break
			}
		}
		if put < 0 {
			continue
		}
		from, to := enclosing(lines, put)
		if firstLine(lines, from, to, name+".remove(") < 0 {
			out = append(out, lint.Finding{Line: put + 1, Msg: "static session/token map '" + name + msg})
		}
	}
	return out, nil
}

// firstLine is the first index in lines[from..to] containing s, or -1.
func firstLine(lines []string, from, to int, s string) int {
	for i := from; i <= to && i < len(lines); i++ {
		if strings.Contains(lines[i], s) {
			return i
		}
	}
	return -1
}

// blockEnd is the line where the braces counted from line from first balance after one opened, or -1.
func blockEnd(lines []string, from int) int {
	depth, started := 0, false
	for i := from; i < len(lines); i++ {
		for k := 0; k < len(lines[i]); k++ {
			switch lines[i][k] {
			case '{':
				depth, started = depth+1, true
			case '}':
				depth--
			}
		}
		if started && depth == 0 {
			return i
		}
	}
	return -1
}

// scheduledPurges is bash step 2: the fields a Clock.schedule* action's body (do<Action>() or <action>(),
// the first declaration found) removes from, one hop.
func scheduledPurges(lines, fields []string) map[string]bool {
	purged := map[string]bool{}
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
			decl := regexp.MustCompile(`[A-Za-z0-9_>\]][[:space:]]+(` + do + `|` + act + `)[[:space:]]*\([^;]*\)[[:space:]]*(\{|$|throws)`)
			for j := range lines {
				if !decl.MatchString(" " + lines[j]) {
					continue
				}
				for j2, end := j, blockEnd(lines, j); j2 <= end; j2++ { // never closed: no body (bash send = 0)
					for _, fl := range fields {
						if strings.Contains(lines[j2], fl+".remove(") {
							purged[fl] = true
						}
					}
				}
				break
			}
		}
	}
	return purged
}

// enclosing is bash step 3's scope of the put on line put: from the nearest method-signature-shaped line
// at or above it (not a call ending in ';') to where its braces balance (the file end if they never do),
// or the put line alone when no signature is found (a static initializer: fail closed).
func enclosing(lines []string, put int) (int, int) {
	for i := put; i >= 0; i-- {
		loc := signature.FindStringIndex(lines[i])
		if loc == nil || strings.HasPrefix(strings.TrimLeft(lines[i][loc[1]:], " \t\n\v\f\r"), ";") {
			continue
		}
		if end := blockEnd(lines, i); end >= 0 {
			return i, end
		}
		return i, len(lines) - 1
	}
	return put, put
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
