package verifymodule

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// declaredDeps are the module names gradle.kts declares, read through ktssrc (comments and string
// contents are no code): every api/nre/implementation call whose first argument is a ":X" string literal
// or project(":X"). The bash grepped the text a line at a time, so a call in a comment or a string, or
// one named xapi(), declared X too (declared difference, truth-kts-phantom).
func declaredDeps(toks []ktssrc.Token) []string {
	var out []string
	at := func(i int, k ktssrc.Kind, text string) bool {
		return i < len(toks) && toks[i].Kind == k && (text == "" || toks[i].Text == text)
	}
	dep := func(i int) (string, bool) {
		if at(i, ktssrc.String, "") && strings.HasPrefix(toks[i].Value, ":") && !toks[i].Template {
			return toks[i].Value[1:], true
		}
		return "", false
	}
	for i, t := range toks {
		if t.Kind != ktssrc.Ident || (t.Text != "api" && t.Text != "nre" && t.Text != "implementation") ||
			!at(i+1, ktssrc.Punct, "(") || (i > 0 && at(i-1, ktssrc.Punct, ".")) {
			continue
		}
		if name, ok := dep(i + 2); ok {
			out = append(out, name)
		} else if at(i+2, ktssrc.Ident, "project") && at(i+3, ktssrc.Punct, "(") {
			if name, ok := dep(i + 4); ok {
				out = append(out, name)
			}
		}
	}
	return out
}

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
	declared := sortU(declaredDeps(toks))
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

// moduleTest looks for moduleTest-include.xml through the shared walk: a copy under a dot directory
// (.deploy-baseline/) or a sibling *-worktrees checkout is not this module's. The bash's unpruned find took
// one as present (declared difference, truth-moduletest-pruned).
func (r *run) moduleTest() bool {
	if !strings.HasSuffix(basename(r.jar.path, ".jar"), "-rt") {
		r.row("SKIP", "moduletest", "not an -rt jar")
		return true
	}
	pd, ok := r.srcOrSkip("moduletest")
	if !ok {
		return true
	}
	if !lint.IsDir(pd) {
		r.row("SKIP", "moduletest", "no source dir "+pd)
		return true
	}
	if !lint.IsDir(pd+"/src") || len(r.javaFiles(pd+"/src")) == 0 {
		r.row("SKIP", "moduletest", "no production .java under src/")
		return true
	}
	msg := "production .java in src/ but no moduleTest-include.xml — add BTestNg station-integration tests (see types/moduleTest.md)"
	switch {
	case len(r.walk(pd, "moduleTest-include.xml")) > 0:
		r.row("PASS", "moduletest", "moduleTest-include.xml present")
	case r.strict:
		r.row("FAIL", "moduletest", msg)
		return false
	default:
		r.row("WARN", "moduletest", msg)
	}
	return true
}

var typeValueRe = rx(`type[[:space:]]*=[[:space:]]*"[^"]*"`)

func (r *run) crossModuleType() bool {
	pd, ok := r.srcDirOrSkip("cross-module-type")
	if !ok {
		return true
	}
	var types []string
	for _, f := range r.javaFiles(pd + "/src") {
		t, ok := r.text(f)
		if !ok {
			return true
		}
		for _, m := range t.only(typeValueRe) {
			types = append(types, typeAttrRe.FindStringSubmatch(m)[1])
		}
	}
	warned := false
	for _, t := range sortU(types) {
		dot := strings.LastIndexByte(t, '.')
		if dot < 0 || dot+1 == len(t) || t[dot+1] < 'A' || t[dot+1] > 'Z' {
			continue
		}
		if hasAnyPrefix(t, "javax.baja.", "com.tridium.", "com.tridiumx.", "com.honeywell.") ||
			lint.IsDir(pd+"/src/"+strings.ReplaceAll(t[:dot], ".", "/")) {
			continue
		}
		r.row("WARN", "cross-module-type", "foreign type in @NiagaraProperty: "+t)
		warned = true
	}
	if !warned {
		r.row("PASS", "cross-module-type", "no cross-module type references in "+pd+"/src")
	}
	return true
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

var (
	subscriberRe = regexp.MustCompile(`(extends[[:space:]]+TypeSubscriber|TypeSubscriber[[:space:]]+)`)
	stoppedRe    = regexp.MustCompile(`void[[:space:]]+stopped[[:space:]]*\(`)
)

func (r *run) subscriptionLeak() bool {
	pd, ok := r.srcDirOrSkip("subscription-leak")
	if !ok {
		return true
	}
	warned := false
	for _, f := range r.javaFiles(pd + "/src") {
		t, ok := r.text(f)
		if !ok {
			return true
		}
		if t.count(subscriberRe) > 0 && t.count(stoppedRe) == 0 {
			r.row("WARN", "subscription-leak", f+": TypeSubscriber without stopped() — unsubscribeAll() never called, space-map leaks until restart")
			warned = true
		}
	}
	if !warned {
		r.row("PASS", "subscription-leak", "no TypeSubscriber without stopped() under "+pd+"/src")
	}
	return true
}

var (
	niagaraPropertyRe = regexp.MustCompile(`@NiagaraProperty`)
	classDeclRe       = rx(`(public[[:space:]]+)?(class|interface)[[:space:]]+[A-Za-z_][A-Za-z0-9_]*`)
)

func (r *run) slotWall() bool {
	pd, ok := r.srcDirOrSkip("slot-wall")
	if !ok {
		return true
	}
	warned := false
	for _, f := range r.javaFiles(pd + "/src") {
		t, ok := r.text(f)
		if !ok {
			return true
		}
		count := t.count(niagaraPropertyRe)
		if count <= 15 {
			continue
		}
		class := basename(f, ".java")
		if ms := t.only(classDeclRe); len(ms) > 0 {
			fields := awkFields(ms[0])
			class = fields[len(fields)-1]
		}
		r.row("WARN", "slot-wall", fmt.Sprintf("%d properties in class %s (%s) — compose into child BComponents (logic.md L21)", count, class, f))
		warned = true
	}
	if !warned {
		r.row("PASS", "slot-wall", "no flat slot wall (>15 @NiagaraProperty per class) under "+pd+"/src")
	}
	return true
}
