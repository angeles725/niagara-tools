package verifymodule

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// srcOrSkip is the opening of every --src check: the jar's profile dir <src>/<jar basename>, or the
// "no --src" SKIP row (ok false).
func (r *run) srcOrSkip(check string) (pd string, ok bool) {
	if r.src == "" {
		r.row("SKIP", check, "no --src")
		return "", false
	}
	return r.src + "/" + basename(r.jar.path, ".jar"), true
}

// srcDirOrSkip is srcOrSkip, then the "no <pd>/src" SKIP row when that is not a directory.
func (r *run) srcDirOrSkip(check string) (string, bool) {
	pd, ok := r.srcOrSkip(check)
	if ok && !lint.IsDir(pd+"/src") {
		r.row("SKIP", check, "no "+pd+"/src")
		return "", false
	}
	return pd, ok
}

// javaFiles is the bash java_files: every *.java under root through the shared walk (dot and *-worktrees
// dirs pruned, D9b). A sub-directory the walk cannot enter is named on stderr once per run and makes the
// verdict INCOMPLETE (exit 3); the files it did list are still checked.
func (r *run) javaFiles(root string) []string { return r.walk(root, "*.java") }

// walk is every regular file named name under root through the shared walk, a walk error handled as
// javaFiles says.
func (r *run) walk(root, name string) []string {
	files, err := lint.Walk(root, lint.Selection{Names: []string{name}})
	if err != nil {
		// One line per unreadable directory, however many walks (src/ and the profile dir) reach it.
		if !contains(r.walkErrs, err.Error()) {
			r.walkErrs = append(r.walkErrs, err.Error())
			fmt.Fprintln(r.s.Err, "verify-module: cannot list every file under "+root+": "+err.Error())
		}
	}
	return files
}

// text is a source file as grep reads it, read once per run; false (and the run ends, exit 3) when it
// cannot be read. The bash read it with 2>/dev/null, as empty.
func (r *run) text(path string) (grepText, bool) {
	if t, ok := r.texts[path]; ok {
		return t, true
	}
	data, ok := r.readFile(path)
	if !ok {
		return grepText{}, false
	}
	t := newGrepText(string(data))
	r.texts[path] = t
	return t, true
}

// javaView is one .java as the javasrc-based checks read it: raw (as grep reads it), code (javasrc.Strip:
// comments blanked, literal text kept, line numbers kept) and bare (code with literal contents masked, so
// only code is left).
type javaView struct {
	raw        grepText
	code, bare []string
}

// java is the view of path, built once per run; false (and the run ends, exit 3) when it cannot be read.
func (r *run) java(path string) (*javaView, bool) {
	if v, ok := r.views[path]; ok {
		return v, true
	}
	t, ok := r.text(path)
	if !ok {
		return nil, false
	}
	code, _ := javasrc.Strip(t.lines)
	v := &javaView{raw: t, code: code, bare: javasrc.MaskLiterals(code)}
	r.views[path] = v
	return v, true
}

// readFile is path's bytes; false (and the run ends, exit 3) when it cannot be read.
func (r *run) readFile(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		r.cannotRead(path, strerror.Text(err))
		return nil, false
	}
	return data, true
}

func (r *run) typeCount() bool {
	pd, ok := r.srcOrSkip("typecount")
	if !ok {
		return true
	}
	inc := pd + "/module-include.xml"
	if !lint.IsFile(inc) {
		r.row("SKIP", "typecount", "no "+inc)
		return true
	}
	data, ok := r.readFile(inc)
	if !ok {
		return true
	}
	doc, err := xmlsrc.Parse(inc, data)
	if err != nil {
		r.envError(err.Error())
		return true
	}
	tn, sn := r.jar.typeCount(), itoa(len(doc.Find("type")))
	if itoa(tn) == sn {
		r.row("PASS", "typecount", fmt.Sprintf("jar declares %d types == module-include.xml", tn))
		return true
	}
	r.row("FAIL", "typecount", fmt.Sprintf("jar declares %d types, %s declares %s (stale build or missing slotomatic)", tn, inc, sn))
	return false
}

var rawFacetRe = rx(`BFacets\.make\(BFacets\.(MIN|MAX), *-?[0-9]`)

func (r *run) facets() bool {
	pd, ok := r.srcDirOrSkip("facets")
	if !ok {
		return true
	}
	for _, f := range r.javaFiles(pd + "/src") {
		t, ok := r.text(f)
		if !ok {
			return true
		}
		if h := t.numbered(rawFacetRe); len(h) > 0 {
			r.row("FAIL", "facets", "raw-number MIN/MAX facet (wrap in BDouble.make): "+f+":"+h[0])
			return false
		}
	}
	r.row("PASS", "facets", "no raw-number MIN/MAX facet under "+pd+"/src")
	return true
}

var (
	ordLiteralRe = rx(`"(station:|local:|slot:/)`)
	commentRe    = regexp.MustCompile(`(/\*|//)`)
)

func (r *run) ordLiteral() bool {
	pd, ok := r.srcDirOrSkip("ord-literal")
	if !ok {
		return true
	}
	warned := false
	for _, f := range r.javaFiles(pd + "/src") {
		t, ok := r.text(f)
		if !ok {
			return true
		}
		if strings.Contains(f, "/srcTest/") || (strings.Contains(basename(f, ".java"), "OrdConstants") && t.count(commentRe) > 0) {
			continue
		}
		for _, hit := range t.numbered(ordLiteralRe) {
			if !strings.Contains(hit, "defaultValue") {
				r.row("WARN", "ord-literal", f+":"+hit)
				warned = true
			}
		}
	}
	if !warned {
		r.row("PASS", "ord-literal", "no hardcoded ORD literals under "+pd+"/src")
	}
	return true
}

// lineWarn is the shared body of the grep -n WARN checks: one row per printed hit, else the PASS row. With
// bare, re is matched on the javasrc bare view (comments and literal contents blanked) and the row quotes
// the raw line.
func (r *run) lineWarn(check string, re *regexp.Regexp, bare bool, pass string) bool {
	pd, ok := r.srcDirOrSkip(check)
	if !ok {
		return true
	}
	warned := false
	for _, f := range r.javaFiles(pd + "/src") {
		v, ok := r.java(f)
		if !ok {
			return true
		}
		for i, ln := range v.raw.lines {
			match := ln
			if bare {
				match = v.bare[i]
			}
			if re.MatchString(match) && v.raw.shown(ln) {
				r.row("WARN", check, f+":"+itoa(i+1)+":"+ln)
				warned = true
			}
		}
	}
	if !warned {
		r.row("PASS", check, fmt.Sprintf(pass, pd))
	}
	return true
}

var (
	transientOpRe = rx(`TRANSIENT[^;]*OPERATOR|OPERATOR[^;]*TRANSIENT`)
	compact3Re    = rx(`import (java\.awt|javax\.swing|java\.sql)\.`)
)

// transientOperator matches the bare view: a TRANSIENT and OPERATOR in a comment or a string are no
// flags (the bash grepped the raw text; declared difference, truth-java-view).
func (r *run) transientOperator() bool {
	return r.lineWarn("transient-operator", transientOpRe, true, "no TRANSIENT+OPERATOR slot combination under %s/src")
}

func (r *run) compact3Imports() bool {
	if b := basename(r.jar.path, ".jar"); !strings.HasSuffix(b, "-rt") && !strings.HasSuffix(b, "-ux") {
		r.row("SKIP", "compact3-import", "not an -rt or -ux jar")
		return true
	}
	return r.lineWarn("compact3-import", compact3Re, false, "no Compact3 non-safe imports under %s/src")
}
