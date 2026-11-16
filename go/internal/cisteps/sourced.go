package cisteps

import (
	"fmt"
	"sort"
	"strings"
)

// The shellcheck closure of `ci-steps changed` (retro 2026-11-15 coordination-stretch T14). shellcheck reads
// a sourced file with its sourcer only when both are on its command line (a lone ci-local.sh read CACHE as
// unused without its libs), and it pairs `. "$ROOT/scripts/lib/x.sh"` with the command-line file whose path
// ends in the literal tail scripts/lib/x.sh. So the change's shell files are checked with the files they
// source and the files that source them, transitively, instead of CI's whole set (~43 s for 285 files); the
// whole set stays the fallback when a source line of the set names a path that cannot be resolved (T22).

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
// go/go-tool.sh, not the 110 other wrappers that source it.
//
// Any source line of the set that names no file of the set returns its <file>:<line> as the reason to check
// the whole set: it might be the only sourcer of a changed file. A changed path the set does not hold
// (deleted, or not a regular file) is left out. (retro 2026-11-15 coordination-stretch T22)
func shellcheckClosure(changed, set []string, read func(string) ([]byte, error)) ([]string, string, error) {
	refs := map[string][]sourceRef{}
	for _, p := range set {
		data, err := read(p)
		if err != nil {
			return nil, "", err
		}
		refs[p] = sourceRefs(p, data)
		for _, r := range refs[p] {
			// a line that names no file of the set might be the only sourcer of a changed file
			if len(r.resolve(set)) == 0 {
				return nil, fmt.Sprintf("%s:%d", p, r.line), nil
			}
		}
	}
	in := map[string]bool{}
	var queue []string
	for _, c := range changed {
		if contains(set, c) { // a changed path the set does not hold is not checked
			in[c] = true
			queue = append(queue, c)
		}
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
			for _, h := range r.resolve(set) {
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

// contains reports whether list holds s.
func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
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
