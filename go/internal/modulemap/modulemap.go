// Package modulemap is the module-map tool behind toolbelt/module-map.sh (modularity-structure Δ1 and Δ4): it
// reads a module root's MODULE-MAP.md with the lint-module-map parser and
//
//	add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]
//	    creates the concern folder under the part's role folder, a pure <Name>Logic class and its
//	    <Name>LogicTest (an rt <Name>Adapter stub under adapter/<concern>/ with --adapter) and inserts the
//	    concern's MODULE-MAP row after the table's last row; refuses an existing concern or folder
//	where <file|concern> --root <module-root>
//	    prints the concern that lists the file (or the named concern): its part, files, tests and "change here when"
//	tests <concern> --root <module-root>
//	    runs the concern's pure JUnit tests through run-pure-test.sh (found in $MODULE_MAP_TOOLBELT) and names
//	    each station test (a javax.baja import) and JS spec it skips
//
// Exit: 0 ok · 1 refused (add-concern), not found (where, tests) or a test failed (tests) · 2 usage · 3 env
// (no or invalid map, no such part, a write or the runner failing to start or reporting an env error).
package modulemap

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	mmrule "github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
)

const usage = "usage: module-map.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]\n" +
	"       module-map.sh where <file|concern> --root <module-root>\n" +
	"       module-map.sh tests <concern> --root <module-root>"

// defaultRole is the role folder a new concern goes under, per part (types/structure.md § Module map).
var defaultRole = map[string]string{"rt": "model", "ux": "route", "wb": "model", "se": "model"}

var javaName = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// exitErr is an error carrying the exit code it maps to.
type exitErr struct {
	code int
	msg  string
}

func (e exitErr) Error() string { return e.msg }

func fail(code int, format string, a ...any) error { return exitErr{code, fmt.Sprintf(format, a...)} }

// opts are the parsed arguments of one subcommand.
type opts struct {
	cmd, target, root, part string
	adapter                 bool
}

func parse(args []string) (opts, error) {
	var o opts
	if len(args) == 0 {
		return o, fail(2, "%s", usage)
	}
	o.cmd, args = args[0], args[1:]
	switch o.cmd {
	case "add-concern", "where", "tests":
	default:
		return o, fail(2, "module-map: unknown subcommand %q\n%s", o.cmd, usage)
	}
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--root", "--part":
			if i+1 >= len(args) {
				return o, fail(2, "module-map: %s needs a value\n%s", a, usage)
			}
			i++
			if a == "--root" {
				o.root = args[i]
			} else {
				o.part = args[i]
			}
		case "--adapter":
			o.adapter = true
		default:
			if strings.HasPrefix(a, "--") || o.target != "" {
				return o, fail(2, "module-map: unexpected argument %q\n%s", a, usage)
			}
			o.target = a
		}
	}
	if o.target == "" || o.root == "" {
		return o, fail(2, "%s", usage)
	}
	if o.cmd != "add-concern" && (o.part != "" || o.adapter) {
		return o, fail(2, "module-map: --part and --adapter belong to add-concern\n%s", usage)
	}
	return o, nil
}

// Main is the module-map command.
func Main(args []string, s cli.Stdio) int {
	o, err := parse(args)
	if err == nil {
		switch o.cmd {
		case "add-concern":
			err = addConcern(o, s)
		case "where":
			err = where(o, s)
		default:
			err = tests(o, s)
		}
	}
	if err == nil {
		return 0
	}
	fmt.Fprintln(s.Err, err.Error())
	var e exitErr
	if errors.As(err, &e) {
		return e.code
	}
	return 3
}

func readMap(root string) (mmrule.Map, error) {
	m, err := mmrule.ReadMap(root)
	if err != nil {
		return m, fail(3, "module-map: %v", err)
	}
	return m, nil
}

func findRow(m mmrule.Map, concern string) (mmrule.Row, bool) {
	for _, r := range m.Rows {
		if strings.EqualFold(r.Concern, concern) {
			return r, true
		}
	}
	return mmrule.Row{}, false
}

// partDir is the root's single sub-directory named *-<part>.
func partDir(root, part string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fail(3, "module-map: cannot list %s: %v", root, err)
	}
	var found []string
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, "-"+part) && len(n) > len(part)+1 && lint.IsDir(filepath.Join(root, n)) {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		return "", fail(3, "module-map: want one *-%s part under %s, found %d", part, root, len(found))
	}
	return found[0], nil
}

// roleDir is the part's role folder for a new concern, relative to <part>/src: the shallowest Java package
// directory itself when it is the role folder (a ux module's route/), else its default role child.
func roleDir(src, role string) (string, error) {
	best, depth := "", -1
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() && (rel == "rc" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".java") {
			dir := filepath.ToSlash(filepath.Dir(rel))
			if n := strings.Count(dir, "/"); depth < 0 || n < depth || (n == depth && dir < best) {
				best, depth = dir, n
			}
		}
		return nil
	})
	if err != nil {
		return "", fail(3, "module-map: cannot walk %s: %v", src, err)
	}
	if depth < 0 {
		return "", fail(3, "module-map: no Java source under %s to place the concern beside", src)
	}
	if path.Base(best) == role {
		return best, nil
	}
	return best + "/" + role, nil
}

func addConcern(o opts, s cli.Stdio) error {
	if o.part == "" {
		o.part = "rt"
	}
	role, ok := defaultRole[o.part]
	switch {
	case !javaName.MatchString(o.target):
		return fail(2, "module-map: concern name %q: a Java identifier starting with an uppercase letter (Battery, CoolingLoop)", o.target)
	case !ok:
		return fail(2, "module-map: --part %q: one of rt, ux, wb, se", o.part)
	case o.adapter && o.part != "rt":
		return fail(2, "module-map: --adapter adds an rt adapter class; use it with --part rt")
	}
	m, err := readMap(o.root)
	if err != nil {
		return err
	}
	concern := strings.ToLower(o.target)
	if r, dup := findRow(m, concern); dup {
		return fail(1, "module-map: concern %s already has a row (MODULE-MAP.md:%d); nothing written", r.Concern, r.Line)
	}
	part, err := partDir(o.root, o.part)
	if err != nil {
		return err
	}
	rd, err := roleDir(filepath.Join(o.root, part, "src"), role)
	if err != nil {
		return err
	}
	pkg := strings.ReplaceAll(rd, "/", ".") + "." + concern
	testRoot := part + "/srcTest"
	if lint.IsDir(filepath.Join(o.root, part, "srcTest", "test")) {
		testRoot += "/test"
	}
	cls := o.target + "Logic"
	srcDir := part + "/src/" + rd + "/" + concern
	files := map[string]string{
		srcDir + "/" + cls + ".java":                                  pureClass(pkg, cls, concern),
		testRoot + "/" + rd + "/" + concern + "/" + cls + "Test.java": pureTest(pkg, cls),
	}
	order := []string{srcDir + "/" + cls + ".java"}
	dirs := []string{srcDir}
	if o.adapter {
		base := path.Dir(rd)
		adir := part + "/src/" + base + "/adapter/" + concern
		apkg := strings.ReplaceAll(base, "/", ".") + ".adapter." + concern
		files[adir+"/"+o.target+"Adapter.java"] = adapterClass(apkg, pkg, o.target, cls, concern)
		order = append(order, adir+"/"+o.target+"Adapter.java")
		dirs = append(dirs, adir)
	}
	test := testRoot + "/" + rd + "/" + concern + "/" + cls + "Test.java"
	for _, d := range dirs {
		if _, err := os.Lstat(filepath.Join(o.root, d)); err == nil {
			return fail(1, "module-map: concern folder %s already exists; nothing written", d)
		}
	}
	if _, err := os.Lstat(filepath.Join(o.root, test)); err == nil {
		return fail(1, "module-map: %s already exists; nothing written", test)
	}
	row := fmt.Sprintf("| %s | %s | %s | %s | <the %s decision changes: replace this placeholder> |",
		concern, o.part, strings.Join(order, "; "), test, concern)
	last := m.Rows[len(m.Rows)-1].Line
	lines := append(append(append([]string{}, m.Lines[:last]...), row), m.Lines[last:]...)
	var written []string
	undo := func() {
		for _, f := range written {
			_ = os.Remove(filepath.Join(o.root, f))
		}
	}
	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	for _, f := range names {
		p := filepath.Join(o.root, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			undo()
			return fail(3, "module-map: %v", err)
		}
		if err := os.WriteFile(p, []byte(files[f]), 0o644); err != nil {
			undo()
			return fail(3, "module-map: %v", err)
		}
		written = append(written, f)
	}
	mp := filepath.Join(o.root, lint.MapFile)
	tmp := mp + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		undo()
		return fail(3, "module-map: %v", err)
	}
	if err := os.Rename(tmp, mp); err != nil {
		_ = os.Remove(tmp)
		undo()
		return fail(3, "module-map: %v", err)
	}
	for _, f := range names {
		fmt.Fprintf(s.Out, "created  %s\n", f)
	}
	fmt.Fprintf(s.Out, "mapped   %s:%d  %s (fill its \"Change here when\" cell)\n", lint.MapFile, last+1, concern)
	return nil
}

func where(o opts, s cli.Stdio) error {
	m, err := readMap(o.root)
	if err != nil {
		return err
	}
	var hits []mmrule.Row
	if r, ok := findRow(m, o.target); ok {
		hits = append(hits, r)
	} else {
		rel := o.target
		if filepath.IsAbs(rel) {
			absRoot, _ := filepath.Abs(o.root)
			if r, err := filepath.Rel(absRoot, rel); err == nil {
				rel = r
			}
		}
		rel = path.Clean(filepath.ToSlash(rel))
		for _, r := range m.Rows {
			if lists(r.Files, rel) || lists(r.Tests, rel) {
				hits = append(hits, r)
			}
		}
	}
	if len(hits) == 0 {
		return fail(1, "module-map: no concern of %s lists %s", lint.MapFile, o.target)
	}
	for i, r := range hits {
		if i > 0 {
			fmt.Fprintln(s.Out)
		}
		fmt.Fprintf(s.Out, "concern  %s\npart     %s\n", r.Concern, r.Part)
		for _, f := range r.Files {
			fmt.Fprintf(s.Out, "files    %s\n", f)
		}
		for _, f := range r.Tests {
			fmt.Fprintf(s.Out, "tests    %s\n", f)
		}
		fmt.Fprintf(s.Out, "change   %s\n", r.When)
	}
	return nil
}

// lists: an entry of cell names rel exactly or is a glob matching it.
func lists(cell []string, rel string) bool {
	for _, e := range cell {
		if path.Clean(e) == rel || (strings.ContainsAny(e, "*?[") && mmrule.MatchGlob(e, rel)) {
			return true
		}
	}
	return false
}

var pkgLine = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)

func tests(o opts, s cli.Stdio) error {
	m, err := readMap(o.root)
	if err != nil {
		return err
	}
	r, ok := findRow(m, o.target)
	if !ok {
		return fail(1, "module-map: no concern %s in %s", o.target, lint.MapFile)
	}
	files, err := expand(o.root, r.Tests)
	if err != nil {
		return err
	}
	failed, envErr := false, false
	for _, f := range files {
		switch {
		case strings.HasSuffix(f, ".js"):
			fmt.Fprintf(s.Out, "skip  %s  Jasmine browser spec (run it with the module's Karma harness, types/dashboard.md DUX-TEST1)\n", f)
			continue
		case !strings.HasSuffix(f, ".java") || !strings.Contains("/"+f, "/srcTest/"):
			fmt.Fprintf(s.Out, "skip  %s  not a srcTest Java test\n", f)
			continue
		}
		b, err := os.ReadFile(filepath.Join(o.root, f))
		if err != nil {
			return fail(3, "module-map: %v", err)
		}
		if strings.Contains(string(b), "javax.baja") {
			fmt.Fprintf(s.Out, "skip  %s  imports javax.baja (a station test: run it under moduleTest on a station)\n", f)
			continue
		}
		pm := pkgLine.FindSubmatch(b)
		if pm == nil {
			return fail(3, "module-map: %s has no package line", f)
		}
		fqcn := string(pm[1]) + "." + strings.TrimSuffix(path.Base(f), ".java")
		tb := s.Env("MODULE_MAP_TOOLBELT")
		if tb == "" {
			return fail(3, "module-map: MODULE_MAP_TOOLBELT is unset (run through toolbelt/module-map.sh)")
		}
		part := strings.SplitN(f, "/", 2)[0]
		fmt.Fprintf(s.Out, "run   %s  %s\n", f, fqcn)
		cmd := exec.Command(filepath.Join(tb, "run-pure-test.sh"), filepath.Join(o.root, part), fqcn)
		cmd.Stdout, cmd.Stderr = s.Out, s.Err
		if err := cmd.Run(); err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) && ee.ExitCode() == 1 {
				failed = true
			} else {
				envErr = true
				fmt.Fprintf(s.Err, "module-map: %s: %v\n", fqcn, err)
			}
		}
	}
	switch {
	case failed:
		return fail(1, "module-map: a test of concern %s failed", r.Concern)
	case envErr:
		return fail(3, "module-map: a test of concern %s could not run", r.Concern)
	}
	return nil
}

// expand resolves a Tests cell to the files it names, relative to root: an exact entry as written, a glob by
// matching every regular file under root (dot directories skipped), in path order.
func expand(root string, cell []string) ([]string, error) {
	var all []string
	walked := false
	var out []string
	for _, e := range cell {
		if !strings.ContainsAny(e, "*?[") {
			out = append(out, path.Clean(e))
			continue
		}
		if !walked {
			walked = true
			err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() && p != root && strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				if d.Type().IsRegular() {
					rel, _ := filepath.Rel(root, p)
					all = append(all, filepath.ToSlash(rel))
				}
				return nil
			})
			if err != nil {
				return nil, fail(3, "module-map: cannot walk %s: %v", root, err)
			}
			sort.Strings(all)
		}
		for _, f := range all {
			if mmrule.MatchGlob(e, f) {
				out = append(out, f)
			}
		}
	}
	return out, nil
}
