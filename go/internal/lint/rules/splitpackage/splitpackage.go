// Package splitpackage is split-package-check: one Java package declared in two N4 modules of a project. N4 is not
// OSGi: ModuleClassLoader asks the parent, the module's own jar, then its dependencies, and the first hit wins, so a
// same-named class in a second module is silently shadowed. Port of toolbelt/split-package-check.sh (find, a read
// loop with an awk per source file, sort and an awk aggregation); the wrapper keeps the user-facing contract and its
// header documents the rule and its mutation pins.
//
// It is a lint.RootRule: it searches the project root for modules itself (a build marker next to src/), lists each
// module's src/ with the shared fail-closed walk, and reads each file's package declaration. The build markers
// (module-include.xml, build.gradle.kts, build.gradle) are only looked for by name, never read: no XML is parsed.
package splitpackage

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// ID is the lint name n4lint accepts and every row prints (the bash name; it is not a lint-* script).
const ID = "split-package-check"

// Spec registers the rule with the argument contract of toolbelt/split-package-check.sh: [--strict] [--] and
// exactly one project root, rows `WARN  split-package-check  <package>  <detail>` (BarePathRow).
func Spec() lint.Spec {
	return lint.Spec{Root: rule{}, Mode: lint.One, Usage: "usage: split-package-check.sh [--strict] <project-root>", BarePathRow: true}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	pkgDecl = regexp.MustCompile(`^[[:space:]]*package[[:space:]]+([A-Za-z_][A-Za-z0-9_.]*)[[:space:]]*;`)
	markers = map[string]bool{"module-include.xml": true, "build.gradle.kts": true, "build.gradle": true}
	pruned  = map[string]bool{"build": true, "node_modules": true, "src": true, "srcTest": true}
)

const detail = "declared in %d modules: %s -- first-dependency-wins shadows one copy; one package belongs to one module (types/issues-and-gotchas.md §D4c)"

// decl is one source file's package: <package> <module> <file>, as the bash wrote one line of its list.
type decl struct{ pkg, mod, file string }

// CheckRoot is the bash script: the module search, the package list, then one row per split package. Every
// listing or read failure is a stderr line and makes the run exit 3 after the rows, as had_err did.
func (rule) CheckRoot(given string) ([]lint.Finding, error) {
	root := strings.TrimSuffix(given, "/") // ROOT="${1%/}"
	var errs []string
	fail := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	modules, err := findModules(root)
	if err != nil {
		fail("cannot list every file under %s: %s", root, err)
	}
	if len(modules) == 0 {
		fail("no module (a module-include.xml / build.gradle[.kts] next to src/) under %s", root)
		return nil, joined(errs)
	}
	var decls []decl
	for _, d := range modules {
		files, err := lint.Walk(d+"/src", lint.Selection{Names: []string{"*.java"}})
		if err != nil {
			fail("cannot list every file under %s/src: %s", d, err)
		}
		mod := strings.TrimPrefix(d, root+"/")
		for _, f := range files {
			pkg, err := packageOf(f)
			if err != nil {
				fail("cannot scan %s: awk: fatal: cannot open file `%s' for reading: %s", f, f, strerror.Text(err))
				continue
			}
			decls = append(decls, decl{pkg, mod, strings.TrimPrefix(f, root+"/")})
		}
	}
	sort.Slice(decls, func(i, j int) bool { // sort -t TAB -k1,1 -k2,2 -k3,3 under LC_ALL=C
		a, b := decls[i], decls[j]
		if a.pkg != b.pkg {
			return a.pkg < b.pkg
		}
		if a.mod != b.mod {
			return a.mod < b.mod
		}
		return a.file < b.file
	})
	out := aggregate(decls)
	if errs != nil {
		return out, joined(errs)
	}
	return out, nil
}

// joined is the stderr lines as one error: the runner prints "<lint>: " before it, so every later line carries
// that prefix itself, one "split-package-check: ..." line per failure as the bash printed them.
func joined(errs []string) error { return errors.New(strings.Join(errs, "\n"+ID+": ")) }

// aggregate is the awk flush loop: per package the distinct modules in order, each with its first file.
func aggregate(decls []decl) []lint.Finding {
	var out []lint.Finding
	for i := 0; i < len(decls); {
		j, last := i, ""
		var list []string
		for ; j < len(decls) && decls[j].pkg == decls[i].pkg; j++ {
			if decls[j].mod != last {
				list = append(list, decls[j].mod+" ("+decls[j].file+")")
				last = decls[j].mod
			}
		}
		if len(list) > 1 {
			out = append(out, lint.Finding{Path: decls[i].pkg, Msg: fmt.Sprintf(detail, len(list), strings.Join(list, ", "))})
		}
		i = j
	}
	return out
}

// findModules is the module search: `find -H root -mindepth 1` pruning dot directories, build, node_modules, src
// and srcTest, a module being the directory of a marker file that has a src/ directory beside it (sorted). The
// root is followed when it is a symlink; links below it are neither followed nor listed. The first directory it
// cannot read is the error, after the rest is searched.
func findModules(root string) ([]string, error) {
	seen := map[string]bool{}
	var mods []string
	var first error
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if first == nil {
				first = &lint.WalkError{Path: dir, Err: errors.Unwrap(err)}
			}
			return
		}
		for _, e := range entries {
			p := dir + "/" + e.Name()
			switch {
			case e.IsDir():
				if !strings.HasPrefix(e.Name(), ".") && !pruned[e.Name()] {
					walk(p)
				}
			case e.Type().IsRegular() && markers[e.Name()]:
				if st, err := os.Stat(dir + "/src"); err == nil && st.IsDir() && !seen[dir] {
					seen[dir] = true
					mods = append(mods, dir)
				}
			}
		}
	}
	walk(root)
	sort.Strings(mods)
	return mods, first
}

// packageOf is the awk program: the first line that, cut at its first "//", declares `package <name>;`;
// "(default)" when none does.
func packageOf(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		var pe *os.PathError
		if errors.As(err, &pe) {
			return "", pe.Err
		}
		return "", err
	}
	lines, _ := javasrc.Lines(strings.NewReader(string(data)))
	for _, ln := range lines {
		if k := strings.Index(ln, "//"); k >= 0 {
			ln = ln[:k]
		}
		if m := pkgDecl.FindStringSubmatch(ln); m != nil {
			return m[1], nil
		}
	}
	return "(default)", nil
}
