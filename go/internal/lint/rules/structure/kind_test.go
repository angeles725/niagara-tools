package structure

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
)

const mapTable = "\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n"

// themeTree is the smallest theme module of types/theme.md § 1, § 3 and § 7 under M, its map header being header.
func themeTree(header string) map[string]string {
	return map[string]string{".git/HEAD": "x\n",
		"M/MODULE-MAP.md": "# MODULE-MAP — themeAcme\n\n" + header + mapTable +
			"| plumbing | ux | themeAcme-ux/themeAcme-ux.gradle.kts; themeAcme-ux/module-include.xml | — | build only |\n" +
			"| plumbing | ux | themeAcme-ux/nss/theme.nss; themeAcme-ux/ux/theme.css | — | a colour or font changes |\n",
		"M/themeAcme-ux/themeAcme-ux.gradle.kts": "moduleManifest { defines { define(\"themeName\", \"themeAcme\") } }\n",
		"M/themeAcme-ux/module-include.xml":      "<types/>\n",
		"M/themeAcme-ux/nss/theme.nss":           ".button { background: #336699; }\n",
		"M/themeAcme-ux/ux/theme.css":            "body { color: #336699; }\n"}
}

// TestDeclaredKindHeader: the kind is the first `type:` line of the map header, read as the modularity line is
// (before the table, outside a ``` fence, a trailing # comment dropped) and matched exactly. Anything else
// declares nothing, so the zero-Java -ux part keeps its L9 FAIL.
func TestDeclaredKindHeader(t *testing.T) {
	for header, exempt := range map[string]bool{
		"modularity: strict\ntype: theme\n":                   true,
		"type: theme # zero-Java\nmodularity: strict\n":       true,
		"modularity: strict\ntype:theme\n":                    true,
		"modularity: strict\n":                                false,
		"modularity: strict\ntype: Theme\n":                   false,
		"modularity: strict\ntype: themes\n":                  false,
		"modularity: strict\ntype: library\ntype: theme\n":    false,
		"modularity: strict\n```\ntype: theme\n```\n":         false,
		"modularity: strict\n" + mapTable + "\ntype: theme\n": false,
	} {
		dir := t.TempDir()
		write(t, dir, themeTree(header))
		code, out, errs := run(filepath.Join(dir, "M"))
		if got := code == 0 && out == ""; got != exempt || errs != "" || (!exempt && !strings.Contains(out, l9Empty+l9ThemeHint)) {
			t.Errorf("header %q: exit %d out %q err %q, want exempt=%v", header, code, out, errs, exempt)
		}
	}
}

// TestDeclaredKindMapPassesModuleMap: the map a theme or a library declares its kind in is a map lint-module-map
// accepts (the `type:` line is a header line it ignores, and a module of resources has plumbing rows only).
func TestDeclaredKindMapPassesModuleMap(t *testing.T) {
	library := map[string]string{".git/HEAD": "x\n",
		"M/MODULE-MAP.md": "# MODULE-MAP — acmeJdbc\n\nmodularity: strict\ntype: library\n" + mapTable +
			"| plumbing | rt | acmeJdbc-rt/acmeJdbc-rt.gradle.kts; acmeJdbc-rt/module-include.xml | — | build only |\n",
		"M/acmeJdbc-rt/acmeJdbc-rt.gradle.kts": "dependencies { api(\":baja\") }\n",
		"M/acmeJdbc-rt/module-include.xml":     "<types/>\n"}
	for name, tree := range map[string]map[string]string{"theme": themeTree("modularity: strict\ntype: theme\n"), "library": library} {
		dir := t.TempDir()
		write(t, dir, tree)
		root := filepath.Join(dir, "M")
		if code, out, errs := run(root); code != 0 || out != "" || errs != "" {
			t.Errorf("%s: lint-structure exit %d out %q err %q, want a clean pass", name, code, out, errs)
		}
		var o, e bytes.Buffer
		if code := lint.Command(modulemap.Spec())([]string{modulemap.ID, root}, cli.Stdio{Out: &o, Err: &e}); code != 0 || o.Len() != 0 || e.Len() != 0 {
			t.Errorf("%s: lint-module-map exit %d out %q err %q, want a clean pass", name, code, o.String(), e.String())
		}
	}
}

// TestUndeclaredLibraryRowSaysHow: an -rt part with no palette and no declaration FAILs L5, and the row says
// how to declare a library, as the -ux L9 row does for a theme (types/utility-lib.md § 6). The row text before
// the hint is unchanged.
func TestUndeclaredLibraryRowSaysHow(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "M/A-rt/module-include.xml": "<types/>\n"})
	want := "FAIL  lint-structure  A-rt  L5: module.palette missing for -rt profile" +
		" (a library module declares `type: library` in its MODULE-MAP.md header, types/utility-lib.md § 6)\n"
	if code, out, errs := run(filepath.Join(dir, "M")); code != 1 || out != want || errs != "" {
		t.Errorf("exit %d err %q out\n%s\nwant\n%s", code, errs, out, want)
	}
}
