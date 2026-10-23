// Package sessionstore is lint-session-store-lazy-evict: a static session/token Map whose `.put(` has no
// sweep in the inserting method and no scheduled purge. Port of the awk in
// toolbelt/lint-session-store-lazy-evict.sh (the wrapper keeps the user-facing contract and its header
// documents the rule and its mutation pins). Matching keeps gawk's POSIX leftmost-longest semantics
// (regexp Longest) and C-locale bytes.
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
	mapDecl   = rx(`(Map|HashMap|ConcurrentHashMap)[[:space:]]*<[^>]*>[[:space:]]+[A-Za-z_][A-Za-z0-9_]*`)
	tailIdent = rx(`[A-Za-z_][A-Za-z0-9_]*[[:space:]]*$`)
	ident     = rx(`[A-Za-z_][A-Za-z0-9_]*`)
	schedHead = rx(`^.*Clock\.schedule[A-Za-z]*[[:space:]]*\(`)
	signature = rx(`[A-Za-z_][A-Za-z0-9_<>\[\]]*[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([^;{}]*\)`)
)

const msg = "' inserted here with no sweep in the SAME method (no scheduled purge found either) -- add a" +
	" scheduled/size-bounded eviction, or bind auth to user+issue-time rather than the bare container id"

// Check reports, per static Map field (in declaration order, a repeated name once per declaration), its
// first `.put(` line when neither the enclosing method nor a scheduled action's body removes from it.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := make([]string, len(f.Raw))
	for i, ln := range f.Raw {
		if k := strings.Index(ln, "//"); k >= 0 {
			ln = ln[:k]
		}
		lines[i] = ln
	}
	var fields []string
	for _, ln := range lines {
		if !strings.Contains(ln, "static") {
			continue
		}
		if seg := mapDecl.FindString(ln); seg != "" {
			fields = append(fields, strings.Trim(tailIdent.FindString(seg), " \t\n\v\f\r"))
		}
	}
	if len(fields) == 0 {
		return nil, nil
	}
	purged := scheduledPurges(lines, fields)
	var out []lint.Finding
	for _, name := range fields {
		if purged[name] {
			continue
		}
		put := firstLine(lines, 0, len(lines)-1, name+".put(")
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
