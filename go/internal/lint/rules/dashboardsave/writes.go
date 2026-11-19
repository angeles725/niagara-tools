package dashboardsave

import (
	"errors"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

var (
	// part is a file under <part>-rt|-ux|-wb|-se/src/; what precedes it is the module root.
	part = regexp.MustCompile(`(^|/)[^/]+-(rt|ux|wb|se)/src/`)
	// ctxDecl declares a Context or WebOp (a parameter or a local): the request's Context.
	ctxDecl = regexp.MustCompile(`(^|[^A-Za-z0-9_.])(Context|WebOp)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	// slotCall is a call that writes when its last argument is the request Context.
	slotCall = regexp.MustCompile(`\.\s*(set|add|remove|invoke)\s*\(`)
	// specialEvent always writes a schedule.
	specialEvent = regexp.MustCompile(`(^|[^A-Za-z0-9_])addSpecialEvent\s*\(`)
	// saveAction is the station's save action slot; saveCall a station save call (Sys.getStation().save(),
	// station.saveAsync(, Station.save().
	saveAction = regexp.MustCompile(`BStation\s*\.\s*save\b`)
	saveCall   = regexp.MustCompile(`[sS]tation(\s*\(\s*\))?\s*\.\s*save[A-Za-z]*\s*\(`)
)

// partOf is the module root and the part kind (rt, ux, wb, se) of a file under a part's src/.
func partOf(path string) (root, kind string, ok bool) {
	m := part.FindStringSubmatchIndex(path)
	if m == nil {
		return "", "", false
	}
	root = path[:m[0]]
	if m[2] == m[3] { // the part is the path's first segment: the root is the working directory
		root = "."
	} else if root == "" {
		root = "/"
	}
	return root, path[m[4]:m[5]], true
}

// codeView is the file's code joined into one text: comments and literal contents blanked, lines kept.
func codeView(f *javasrc.File) string {
	lines, _ := javasrc.BlankLiterals(f.Stripped(), nil)
	return strings.Join(lines, "\n")
}

// savesStation reports a station save requested in code.
func savesStation(code string) bool {
	return saveAction.MatchString(code) || saveCall.MatchString(code)
}

// writeLines is the 1-based line of every write call in code (once per line, in order): an addSpecialEvent(,
// or a slot call whose last top-level argument is a Context or WebOp this file declares.
func writeLines(code string) []int {
	ctx := map[string]bool{}
	for _, m := range ctxDecl.FindAllStringSubmatch(code, -1) {
		ctx[m[3]] = true
	}
	var lines []int
	add := func(at int) {
		n := strings.Count(code[:at], "\n") + 1
		if len(lines) == 0 || lines[len(lines)-1] != n {
			lines = append(lines, n)
		}
	}
	for _, loc := range sortedCalls(code) {
		if loc.special || ctx[lastArg(code, loc.open)] {
			add(loc.at)
		}
	}
	return lines
}

// call is one candidate write: where it starts, its opening parenthesis and whether it is addSpecialEvent.
type call struct {
	at, open int
	special  bool
}

// sortedCalls lists the slot calls and addSpecialEvent calls of code in text order.
func sortedCalls(code string) []call {
	var out []call
	for _, l := range slotCall.FindAllStringIndex(code, -1) {
		out = append(out, call{at: l[0], open: l[1] - 1})
	}
	for _, l := range specialEvent.FindAllStringIndex(code, -1) {
		out = append(out, call{at: l[0], open: l[1] - 1, special: true})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// lastArg is the last top-level argument of the call whose "(" is at open, trimmed; "" for a call with fewer
// than two arguments or no closing parenthesis.
func lastArg(code string, open int) string {
	depth, comma := 0, -1
	for i := open + 1; i < len(code); i++ {
		switch code[i] {
		case '(', '[', '{':
			depth++
		case ']', '}':
			depth--
		case ')':
			if depth == 0 {
				if comma < 0 {
					return ""
				}
				return strings.TrimSpace(code[comma+1 : i])
			}
			depth--
		case ',':
			if depth == 0 {
				comma = i
			}
		}
	}
	return ""
}

// declaresPersistence reports a non-empty "persistence:" line outside ``` fences in the module root's
// MODULE-MAP.md. No map declares nothing; a map that exists but cannot be read fails the scan closed.
func declaresPersistence(root string) (bool, error) {
	p := strings.TrimSuffix(root, "/") + "/" + lint.MapFile
	if !lint.IsFile(p) {
		return false, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return false, errors.New("cannot read " + p + ": " + err.Error())
	}
	fenced := false
	for _, raw := range strings.Split(string(b), "\n") {
		l := strings.TrimSpace(raw)
		if strings.HasPrefix(l, "```") {
			fenced = !fenced
			continue
		}
		if v, ok := strings.CutPrefix(l, "persistence:"); ok && !fenced && strings.TrimSpace(v) != "" {
			return true, nil
		}
	}
	return false, nil
}
