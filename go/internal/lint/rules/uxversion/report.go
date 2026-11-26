package uxversion

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// The check ids added for sub-tree versions and a ?v= built in JavaScript.
const (
	checkSubtree = "subtree-version"
	checkDynamic = "dynamic-version"
)

// hashed is a content-addressed file name (main-18a30d6b12.js): the hash is the cache key, so no ?v= is needed.
var hashed = regexp.MustCompile(`-[0-9a-f]{8,}\.(js|css|mjs)$`)

// isHashed reports whether the path of ref (query and fragment off) names a content-addressed file.
func isHashed(ref string) bool {
	p, _, _ := strings.Cut(ref, "?")
	p, _, _ = strings.Cut(p, "#")
	return hashed.MatchString(p)
}

// scan is the findings of one file's lines.
func scan(path string, lines []string, want string, subs subtrees) []lint.Finding {
	var refs []ref
	switch strings.ToLower(filepath.Ext(path)) {
	case ".css":
		refs = cssRefs(lines)
	case ".js":
		refs = jsRefs(jssrc.Tokens(lines))
	default:
		refs = htmlRefs(lines)
	}
	refs = append(refs, jsVersionRefs(scriptTokens(path, lines))...)
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].line < refs[j].line })
	var out []lint.Finding
	for _, r := range refs {
		id, msg, sev := judge(r, want, subs)
		if id == "" {
			continue
		}
		// Mutation: UXV-waiver -- ignoring the allow marker reports the waived reference again
		if why := waiverAt(lines, r, id); why != "" {
			msg, sev = fmt.Sprintf("%s: %s allowed: %s", id, refText(r), why), lint.Pass
		}
		out = append(out, lint.Finding{Path: path, Line: r.line, Severity: sev, Msg: msg})
	}
	return out
}

func refText(r ref) string {
	if r.dynamic != "" {
		return r.dynamic
	}
	return r.text
}

// judge is the check id, message and severity of reference r; id is "" when r is fine or not read.
func judge(r ref, want string, subs subtrees) (id, msg string, sev lint.Severity) {
	if r.dynamic != "" {
		return checkDynamic, fmt.Sprintf("%s: %s: dynamic version, check by hand", checkDynamic, r.dynamic), lint.Warn
	}
	chk := r.chk
	if chk == "" {
		chk = r.text
	}
	if isHashed(chk) {
		return "", "", lint.Pass
	}
	sub, in := subs.find(chk)
	if in {
		want = sub.ver
	}
	id, got := verdict(chk, want)
	switch {
	case id == checkMissing:
		return id, fmt.Sprintf("%s: %s has no ?v=%s suffix (types/dashboard.md: cache-bust every static asset with the module version)", id, r.text, want), lint.Warn
	case id == checkStale && in:
		return checkSubtree, fmt.Sprintf("%s: %s has ?v=%s but %s is declared ?v=%s in %s:%d", checkSubtree, r.text, got, sub.dir, sub.ver, sub.file, sub.line), lint.Fail
	case id == checkStale:
		return id, fmt.Sprintf("%s: %s has ?v=%s but defaultModuleVersion is %s (types/frontend-standard.md §11)", id, r.text, got, want), lint.Fail
	}
	return "", "", lint.Pass
}
