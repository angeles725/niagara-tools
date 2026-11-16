package cisteps

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// The shellcheck closure of `ci-steps changed` (retro 2026-11-15 coordination-stretch T14). shellcheck reads
// a sourced file with its sourcer only when both are on its command line (a lone ci-local.sh read CACHE as
// unused without its libs), and it pairs `. "$ROOT/scripts/lib/x.sh"` with the command-line file whose path
// ends in the literal tail scripts/lib/x.sh. So the change's shell files are checked with the files they
// source and the files that source them, transitively, instead of CI's whole set (~43 s for 285 files); the
// whole set stays the fallback when a source line names a path that cannot be resolved.

var (
	sourceCmd       = regexp.MustCompile(`^[ \t]*(?:\.|source)[ \t]+(\S.*)$`)
	sourceDirective = regexp.MustCompile(`^[ \t]*#[ \t]*shellcheck[ \t].*\bsource=(\S+)`)
	// expansion is one parameter or command expansion: $NAME, $1, ${...}, $(...).
	expansion = regexp.MustCompile(`\$(?:[A-Za-z_][A-Za-z0-9_]*|[0-9@*#?$!-]|\{[^}]*\}|\([^)]*\))`)
)

// sourceRef is one `.`/`source` line: the literal path tail it names ("" when the path cannot be read), or
// with exact, a repository path from a `# shellcheck source=SCRIPTDIR/...` directive.
type sourceRef struct {
	line  int
	tail  string
	exact bool
}

// sourceRefs lists the source lines of a script, here-document text skipped. A `# shellcheck source=` line
// before the command names its path (/dev/null: none to follow).
func sourceRefs(script string, data []byte) []sourceRef {
	lines := strings.Split(string(data), "\n")
	body := heredocBody(lines)
	var refs []sourceRef
	directive := ""
	for i, line := range lines {
		if body[i] {
			continue
		}
		if m := sourceDirective.FindStringSubmatch(line); m != nil {
			directive = m[1]
			continue
		}
		m := sourceCmd.FindStringSubmatch(line[:commentAt(line)])
		if m == nil {
			directive = ""
			continue
		}
		switch {
		case directive == "/dev/null":
		case strings.HasPrefix(directive, "SCRIPTDIR/"):
			refs = append(refs, sourceRef{i + 1, path.Join(path.Dir(script), strings.TrimPrefix(directive, "SCRIPTDIR/")), true})
		case directive != "":
			refs = append(refs, sourceRef{i + 1, literalTail(directive), false})
		default:
			refs = append(refs, sourceRef{i + 1, literalTail(firstCommand(m[1])), false})
		}
		directive = ""
	}
	return refs
}

// firstCommand cuts the argument text at the first unquoted `;`, `&&` or `||`.
func firstCommand(arg string) string {
	for i := 0; i < len(arg); i++ {
		if (arg[i] == ';' || strings.HasPrefix(arg[i:], "&&") || strings.HasPrefix(arg[i:], "||")) && !quotedAt(arg, i) {
			return strings.TrimSpace(arg[:i])
		}
	}
	return strings.TrimSpace(arg)
}

// literalTail is the path after the argument's last expansion, quotes and leading ./ and ../ dropped; ""
// when no literal file name follows the expansions or one could not be parsed.
func literalTail(arg string) string {
	if locs := expansion.FindAllStringIndex(arg, -1); len(locs) > 0 {
		arg = arg[locs[len(locs)-1][1]:]
	}
	arg = strings.NewReplacer(`"`, "", `'`, "").Replace(arg)
	if strings.ContainsAny(arg, "$`(){} \t") {
		return ""
	}
	parts := strings.Split(arg, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == ".." || parts[i] == "." || parts[i] == "" {
			parts = parts[i+1:]
			break
		}
	}
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return ""
	}
	return strings.Join(parts, "/")
}

// resolve lists the set's files a reference names: the exact path, or every path that is the tail or ends
// in /tail.
func (r sourceRef) resolve(set []string) []string {
	var hits []string
	for _, p := range set {
		if p == r.tail || (!r.exact && strings.HasSuffix(p, "/"+r.tail)) {
			hits = append(hits, p)
		}
	}
	if r.tail == "" {
		return nil
	}
	return hits
}

// shellcheckClosure returns the changed files of the set with the files that source them (upward,
// transitively) and every file those source (downward, transitively), sorted. A lib that a changed file
// sources is checked with that sourcer, not with all of its others: a change to one toolbelt wrapper takes
// go/go-tool.sh, not the 110 other wrappers that source it. A source line of the closure that names no file of the set
// returns its <file>:<line> as the reason to check the whole set instead.
func shellcheckClosure(changed, set []string, read func(string) ([]byte, error)) ([]string, string, error) {
	refs := map[string][]sourceRef{}
	for _, p := range set {
		data, err := read(p)
		if err != nil {
			return nil, "", err
		}
		refs[p] = sourceRefs(p, data)
	}
	in := map[string]bool{}
	queue := append([]string{}, changed...)
	for _, c := range changed {
		in[c] = true
	}
	for grew := true; grew; { // upward: the sourcers of a changed file, of those, and so on
		grew = false
		for _, p := range set {
			for _, r := range refs[p] {
				if !in[p] && anyIn(r.resolve(set), in) {
					in[p], grew = true, true
					queue = append(queue, p)
				}
			}
		}
	}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, r := range refs[p] {
			hits := r.resolve(set)
			if len(hits) == 0 {
				return nil, fmt.Sprintf("%s:%d", p, r.line), nil
			}
			for _, h := range hits {
				if !in[h] {
					in[h] = true
					queue = append(queue, h)
				}
			}
		}
	}
	out := make([]string, 0, len(in))
	for p := range in {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, "", nil
}

// anyIn reports whether one of paths is in the set.
func anyIn(paths []string, set map[string]bool) bool {
	for _, p := range paths {
		if set[p] {
			return true
		}
	}
	return false
}
