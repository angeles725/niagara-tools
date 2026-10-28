package structure

import (
	"path"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Strict L3 and L3b (types/structure.md § L1–L11): a module whose MODULE-MAP.md says modularity: strict keeps
// every Baja-free class in a role package, every model/ class free of framework imports and tested, and
// model/ independent of the module's other packages. A legacy module (or one with no map) keeps the
// advisory L3 directory WARN of pureModel, byte for byte.

var (
	pkgDecl    = regexp.MustCompile(`^[[:space:]]*package[[:space:]]+([A-Za-z_][A-Za-z0-9_.]*)[[:space:]]*;`)
	importDecl = regexp.MustCompile(`^[[:space:]]*import[[:space:]]+(static[[:space:]]+)?([A-Za-z_][A-Za-z0-9_.]*(\.\*)?)[[:space:]]*;`)
	// a framework type: what a pure (model) class may not import
	framework = regexp.MustCompile(`^(javax\.baja|com\.tridium)\.`)
)

// strictParts are the part suffixes strict L3 applies to, with the role packages a Baja-free class may sit in.
var strictParts = map[string][]string{
	"-rt": {"model"},
	"-ux": {"route", "service", "model"},
	"-wb": {"model"},
	"-se": {"model"},
}

// javaFile is one src/ Java file of a part, comments blanked.
type javaFile struct {
	path, pkg string
	code      []string
}

// imports are the file's import targets ("static" dropped) with their 1-based lines.
func (f javaFile) imports(fn func(target string, line int)) {
	for i, ln := range f.code {
		if m := importDecl.FindStringSubmatch(ln); m != nil {
			fn(m[2], i+1)
		}
	}
}

// pure: no framework import and no @NiagaraType, so the class runs in plain JUnit.
func (f javaFile) pure() bool {
	pure := true
	f.imports(func(t string, _ int) { pure = pure && !framework.MatchString(t) })
	return pure && !grep(f.code, niagaraAnn)
}

// readJava reads one Java file with its comments blanked and its package.
func (s *scan) readJava(p string) javaFile {
	code, _ := javasrc.Strip(s.lines(p))
	f := javaFile{path: p, code: code}
	for _, ln := range code {
		if m := pkgDecl.FindStringSubmatch(ln); m != nil {
			f.pkg = m[1]
			break
		}
	}
	return f
}

// modulePackages is every package a src/ Java file of the root's profiles declares (L3b reads it).
func (s *scan) modulePackages(profiles []string) map[string]bool {
	pkgs := map[string]bool{}
	for _, dir := range profiles {
		if src := dir + "/src"; lint.IsDir(src) {
			for _, p := range s.files(src, false, "*.java") {
				if f := s.readJava(p); f.pkg != "" {
					pkgs[f.pkg] = true
				}
			}
		}
	}
	return pkgs
}

// strictModel is strict L3 and L3b for one part, rows in file path order.
func (s *scan) strictModel(dir, name, src string, java []string) {
	roles := partRoles(name)
	if roles == nil {
		return
	}
	tests := map[string]bool{} // srcTest file base names
	if lint.IsDir(dir + "/srcTest") {
		for _, p := range s.files(dir+"/srcTest", false, "*.java") {
			tests[path.Base(p)] = true
		}
	}
	for _, p := range java {
		base := path.Base(p)
		if base == "package-info.java" || base == "module-info.java" {
			continue
		}
		f := s.readJava(p)
		dirs := strings.Split(strings.TrimPrefix(path.Dir(p), src), "/")
		if !hasSegment(dirs, "model") {
			if f.pure() && !hasSegment(dirs, roles...) {
				s.row(lint.Unset, s.rel(p), "L3: Baja-free class outside a role package (strict: move it under "+roleList(roles)+")")
			}
			continue
		}
		f.imports(func(t string, line int) {
			if framework.MatchString(t) {
				s.out = append(s.out, lint.Finding{Path: s.rel(p), Line: line,
					Msg: "L3: model/ class imports " + t + " (strict: a model class imports no javax.baja or com.tridium type)"})
			}
		})
		class := strings.TrimSuffix(base, ".java")
		if !tests[class+"Test.java"] {
			s.row(lint.Unset, s.rel(p), "L3: model/ class "+class+" has no srcTest "+class+"Test.java (strict: every model class has its pure test)")
		}
		f.imports(func(t string, line int) {
			if pkg := s.ownPackage(t); pkg != "" {
				s.out = append(s.out, lint.Finding{Path: s.rel(p), Line: line,
					Msg: "L3b: model/ class imports " + t + " from the module's non-model package " + pkg +
						" (strict: model/ depends only on model/ and libraries)"})
			}
		})
	}
}

// ownPackage is the module package (outside any model package) an import target names a member of, or "".
// The longest declared package wins; the segment after it must name a type (upper-case first) or be "*", so
// com.x.model.Core is never read as a member of com.x.
func (s *scan) ownPackage(target string) string {
	best := ""
	for pkg := range s.packages {
		rest, ok := strings.CutPrefix(target, pkg+".")
		if !ok || len(pkg) <= len(best) || hasSegment(strings.Split(pkg, "."), "model") {
			continue
		}
		if next := strings.SplitN(rest, ".", 2)[0]; next == "*" || (next != "" && next[0] >= 'A' && next[0] <= 'Z') {
			best = pkg
		}
	}
	return best
}

// partRoles are the role packages of a strict part (nil: strict L3 does not apply to it).
func partRoles(name string) []string {
	for suf, roles := range strictParts {
		if strings.HasSuffix(name, suf) {
			return roles
		}
	}
	return nil
}

func roleList(roles []string) string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = r + "/"
	}
	if len(out) == 1 {
		return out[0]
	}
	return strings.Join(out[:len(out)-1], ", ") + " or " + out[len(out)-1]
}

// hasSegment reports whether any of segs is one of names.
func hasSegment(segs []string, names ...string) bool {
	for _, seg := range segs {
		for _, n := range names {
			if seg == n {
				return true
			}
		}
	}
	return false
}
