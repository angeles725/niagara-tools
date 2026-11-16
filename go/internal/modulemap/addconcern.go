package modulemap

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

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
	test := testRoot + "/" + rd + "/" + concern + "/" + cls + "Test.java"
	files := map[string]string{
		srcDir + "/" + cls + ".java": pureClass(pkg, cls, concern),
		test:                         pureTest(pkg, cls),
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
	// undo removes what this run wrote: its files, then the directories it created, deepest first (os.Remove
	// leaves a directory that is not empty), so a failed run can be retried as is. Both lists hold the same
	// convention: OS paths joined under o.root (mkdirs records created that way), never map-relative names.
	var written, created []string
	undo := func() {
		for _, p := range written {
			_ = os.Remove(p)
		}
		// Mutation: MMA-undo-dirs -- removing only the files leaves the concern folders, and the retry is refused
		for i := len(created) - 1; i >= 0; i-- {
			_ = os.Remove(created[i])
		}
	}
	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	for _, f := range names {
		p := filepath.Join(o.root, f)
		if err := mkdirs(filepath.Dir(p), &created); err != nil {
			undo()
			return fail(3, "module-map: %v", err)
		}
		if err := os.WriteFile(p, []byte(files[f]), 0o644); err != nil {
			undo()
			return fail(3, "module-map: %v", err)
		}
		written = append(written, p)
	}
	mp := filepath.Join(o.root, lint.MapFile)
	tmp := mp + ".tmp"
	if err := writeTemp(tmp, []byte(strings.Join(lines, "\n"))); err != nil {
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
