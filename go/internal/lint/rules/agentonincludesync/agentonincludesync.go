// Package agentonincludesync is lint-agent-on-include-sync (G99, kit issue #473): for every *.java under a part's
// src/ whose @NiagaraType declares agent = @AgentOn(types = ...) (one string or an array, one @AgentOn or several),
// the <type class="<fqcn>"> line of that part's module-include.xml must carry an <agent><on type="..."/> for every
// declared type. javac's annotation processor never rewrites a line whose existing <agent> elements all match, and a
// line with no <agent> matches any annotation (ModuleInclude.entryMatches, nre 4.14.0.162 vineflower
// javax/baja/nre/annotations/processors/ModuleInclude.java:376-415), so an @AgentOn added later is silently dropped
// and the agent never appears in Workbench. Doctrine: types/module-wiring.md § 2; the toolbelt wrapper
// toolbelt/lint-agent-on-include-sync.sh holds the user-facing contract and the mutation pins.
//
// Java is read as javasrc.Tokens (comments stripped, a literal one token), so an annotation in a comment or a string
// is not one; a types value that is not a string literal (a constant) cannot be resolved and is not checked. The
// part of a source file is the deepest directory holding a module-include.xml whose src/ contains it; srcTest/ and
// build/ are never read, and a *.java outside every src/ is not a part source.
package agentonincludesync

import (
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// ID is the lint name printed in every row.
const ID = "lint-agent-on-include-sync"

const include = "module-include.xml"

// Spec registers the rule: [--] <module-root>... (each root checked on its own), every *.java and module-include.xml
// outside srcTest/ and build/; a root with no module-include.xml is the ERROR row (exit 3). Its roots are module or
// part roots, never source roots, so n4lint --all leaves it out (Spec.ModuleRoot).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-agent-on-include-sync.sh [--] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java", include}, Prune: []string{"srcTest", "build"}},
		EmptyError: lint.EmptyRow{Msg: "no " + include + " under the root (wrong path: pass a module or part root)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// CheckTree reads every module-include.xml, then checks each part source's @AgentOn types against its part's file. A
// malformed module-include.xml is an error (exit 3) after the rows of the other files.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	parts := map[string]map[string][]string{} // part dir -> class -> on types
	var bad error
	for _, f := range files {
		if dir, ok := strings.CutSuffix(f.Path, "/"+include); ok {
			parts[dir] = nil // malformed: its sources are not checked (the error is the result)
			if doc, err := xmlsrc.Open(f.Path); err != nil {
				bad = err
			} else {
				parts[dir] = onTypes(doc)
			}
		}
	}
	if len(parts) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	var out []lint.Finding
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".java") {
			continue
		}
		dir, ok := partOf(f.Path, parts)
		if (!ok && !strings.Contains(f.Path, "/src/")) || (ok && parts[dir] == nil) {
			continue
		}
		for _, t := range agentTypes(javasrc.Tokens(f.Stripped())) {
			if msg := check(t, dir, parts[dir], ok); msg != "" {
				out = append(out, lint.Finding{Path: f.Path, Line: t.line, Msg: t.class + ": " + msg})
			}
		}
	}
	return out, bad
}

// check is the row message for one annotated type, or "" when its line carries every declared on type.
func check(t typeDecl, dir string, classes map[string][]string, inPart bool) string {
	want := strings.Join(t.types, ", ")
	if !inPart {
		return "no " + include + " beside its src/ (needs <on type> for: " + want + ")"
	}
	have, ok := classes[t.class]
	if !ok {
		return `no <type class="` + t.class + `"> line in ` + dir + "/" + include + " (needs <on type> for: " + want + ")"
	}
	var missing []string
	for _, w := range t.types {
		if !slices.Contains(have, w) {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return include + " <type> line lacks <on type> for: " + strings.Join(missing, ", ")
}

// partOf is the deepest part directory (one holding a module-include.xml) whose src/ holds path.
func partOf(path string, parts map[string]map[string][]string) (string, bool) {
	for i := strings.LastIndex(path, "/src/"); i >= 0; i = strings.LastIndex(path[:i], "/src/") {
		if _, ok := parts[path[:i]]; ok {
			return path[:i], true
		}
	}
	return "", false
}

// onTypes maps each <type class> to the type attribute of every <on> in its <agent> children.
func onTypes(doc *xmlsrc.Doc) map[string][]string {
	out := map[string][]string{}
	elems := doc.Elements()
	for _, ti := range doc.Find("type") {
		class, ok := elems[ti].Attr("class")
		if !ok {
			continue
		}
		on := out[class] // a <type> line with no <agent> is still a line: the key is set
		for _, ai := range doc.Children(ti) {
			if elems[ai].Name != "agent" {
				continue
			}
			for _, oi := range doc.Children(ai) {
				if v, ok := elems[oi].Attr("type"); ok && elems[oi].Name == "on" {
					on = append(on, v)
				}
			}
		}
		out[class] = on
	}
	return out
}
