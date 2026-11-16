package verifymodule

import (
	"os"
	"path"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// The --src phantom-dep check: the module names a part's gradle.kts declares (lint.DeclaredDeps) against the
// modules its sources use.

// firstGradleKts is `find <pd> -maxdepth 1 -name '*.gradle.kts' 2>/dev/null | sort | head -1`: any type of
// entry, pd itself included, a symlinked pd not entered (find -P).
func firstGradleKts(pd string) string {
	var hits []string
	if ok, _ := path.Match("*.gradle.kts", lint.Basename(pd)); ok {
		hits = append(hits, pd)
	}
	if st, err := os.Lstat(pd); err == nil && st.IsDir() {
		entries, _ := os.ReadDir(pd)
		for _, e := range entries {
			if ok, _ := path.Match("*.gradle.kts", e.Name()); ok {
				hits = append(hits, pd+"/"+e.Name())
			}
		}
	}
	if len(hits) == 0 {
		return ""
	}
	sort.Strings(hits)
	return hits[0]
}

func (r *run) phantomDep() bool {
	pd, ok := r.srcOrSkip("phantom-dep")
	if !ok {
		return true
	}
	gkt := firstGradleKts(pd)
	if gkt == "" {
		r.row("SKIP", "phantom-dep", "no *.gradle.kts in "+pd)
		return true
	}
	if r.jar.mx == "" {
		r.row("SKIP", "phantom-dep", "no META-INF/module.xml")
		return true
	}
	data, ok := r.readFile(gkt)
	if !ok {
		return true
	}
	toks, err := ktssrc.Tokens(string(data))
	if err != nil { // an unreadable script declares nothing we can trust
		r.envError(gkt + ": " + err.Error())
		return true
	}
	declared := sortU(lint.DeclaredDeps(toks))
	var deps, phantom []string
	for _, i := range r.jar.doc.Find("dependency") {
		if name, ok := r.jar.doc.Elements()[i].Attr("name"); ok {
			deps = append(deps, name)
		}
	}
	for _, dep := range sortU(deps) {
		if dep != "" && !contains(declared, dep) {
			phantom = append(phantom, dep)
		}
	}
	if len(phantom) > 0 {
		r.row("WARN", "phantom-dep", "module.xml dependency not declared in gradle.kts api()/nre(): "+strings.Join(phantom, ", "))
		return true
	}
	r.row("PASS", "phantom-dep", "all module.xml dependencies declared in gradle.kts")
	return true
}
