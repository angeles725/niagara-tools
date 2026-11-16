package ntplmanifest

import (
	"os"
	"strings"
	"testing"
)

const station = "overlay/defaults/workbench/newStations/Plant.ntpl"

// stationDeps is what a stock station template lists: many modules every installation ships, none of them
// declared by the module that ships the template.
var stationDeps = []string{"fox-rt", "jetty-rt", "hx-wb", "web-rt", "control-rt", "bajaui-wb", "baja"}

// TestStockModulesAreNoFinding: with the installation's module list (the option, or NIAGARA_HOME in the
// environment), a dependency the installation ships is not a finding, declared or not: a template built
// like a stock station template passes. One that is neither shipped nor declared still fails.
func TestStockModulesAreNoFinding(t *testing.T) {
	t.Parallel()
	nh := installation(t, stock...)
	root := moduleRoot(t, map[string]string{station: template(t, ` isStation="true"`, stationDeps...)})
	if code, out, errText := run(homeOption, nh, root); code != 0 || out != "" || errText != "" {
		t.Errorf("the option: exit %d stdout %q stderr %q, want a clean pass", code, out, errText)
	}
	if code, out, errText := runEnv(map[string]string{homeEnv: nh}, root); code != 0 || out != "" || errText != "" {
		t.Errorf("the environment: exit %d stdout %q stderr %q, want a clean pass", code, out, errText)
	}
	root = moduleRoot(t, map[string]string{station: template(t, ` isStation="true"`, append([]string{"vendorx-rt", "fox"}, stationDeps...)...)})
	want := "FAIL  ntpl-deps  " + root + "/" + station + ": manifest dependencies neither declared (api/nre/implementation) in any *.gradle.kts under " +
		root + " nor shipped by the installation " + nh + ": fox, vendorx-rt\n"
	if code, out, errText := run(homeOption, nh, root); code != 1 || out != want || errText != "" {
		t.Errorf("a third-party module: exit %d\nstdout %q\nstderr %q\nwant exit 1\nstdout %q", code, out, errText, want)
	}
}

// TestOptionWinsOverTheEnvironment: --niagara-home is the installation of the run even when NIAGARA_HOME
// names another one; a module only the other one ships is still missing.
func TestOptionWinsOverTheEnvironment(t *testing.T) {
	t.Parallel()
	given, ambient := installation(t, "fox-rt"), installation(t, "fox-rt", "jetty-rt")
	root := moduleRoot(t, map[string]string{tpl: template(t, "", "fox-rt", "jetty-rt")})
	code, out, errText := runEnv(map[string]string{homeEnv: ambient}, homeOption, given, root)
	if code != 1 || !strings.HasSuffix(out, " nor shipped by the installation "+given+": jetty-rt\n") || errText != "" {
		t.Errorf("exit %d stdout %q stderr %q, want exit 1 naming jetty-rt against %s", code, out, errText, given)
	}
}

// TestWithoutAnInstallationNothingBlocks: with no installation path the lint cannot tell a stock module
// from a missing one. It says so in one WARN row, reports the undeclared dependencies of each template as
// WARN, and exits 0. A module with nothing undeclared prints nothing; a finding the lint can judge on its
// own (a missing entry) still fails.
func TestWithoutAnInstallationNothingBlocks(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{station: template(t, ` isStation="true"`, stationDeps...), tpl: template(t, "", "ghost-rt", "baja")})
	note := "WARN  lint-ntpl-manifest-deps  " + root + ": the installation's module list is not known, so a stock module cannot be told from a missing one: " +
		"pass --niagara-home <dir> or set NIAGARA_HOME (the ntpl-deps rows are WARN until then)\n"
	want := note +
		"WARN  ntpl-deps  " + root + "/" + tpl + ": manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under " + root + ": ghost-rt\n" +
		"WARN  ntpl-deps  " + root + "/" + station + ": manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under " + root +
		": bajaui-wb, control-rt, fox-rt, hx-wb, jetty-rt, web-rt\n"
	if code, out, errText := run(root); code != 0 || out != want || errText != "" {
		t.Errorf("no installation: exit %d\nstdout %q\nstderr %q\nwant exit 0\nstdout %q", code, out, errText, want)
	}
	if code, out, errText := runEnv(map[string]string{homeEnv: ""}, root); code != 0 || out != want || errText != "" {
		t.Errorf("an empty NIAGARA_HOME is no installation: exit %d stdout %q stderr %q", code, out, errText)
	}
	clean := moduleRoot(t, map[string]string{tpl: template(t, "", "baja", "local-rt")})
	if code, out, errText := run(clean); code != 0 || out != "" || errText != "" {
		t.Errorf("nothing undeclared: exit %d stdout %q stderr %q, want no row", code, out, errText)
	}
	broken := moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog})})
	if code, out, _ := run(broken); code != 1 || out != "FAIL  ntpl-entries  "+broken+"/"+tpl+": lacks template-manifest.xml\n" {
		t.Errorf("a missing entry without an installation: exit %d stdout %q, want exit 1 and its row", code, out)
	}
}

// TestInstallationPathThatGivesNoList: a path that does not exist, one with no modules/ directory, one
// whose modules is a file and one whose modules/ cannot be listed are exit 3 naming what gave the path;
// never a run that treats the unknown list as empty or as absent. It holds for a module with no template
// too: the path is wrong whatever is checked.
func TestInstallationPathThatGivesNoList(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{tpl: template(t, "", "fox-rt")})
	missing := t.TempDir() + "/gone"
	bare := t.TempDir()
	file := t.TempDir()
	write(t, file, "modules", "x\n")
	cases := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{"a path that does not exist", nil, []string{homeOption, missing, root}, "--niagara-home " + missing + ": no modules/ directory (not a Niagara installation)\n"},
		{"no modules/ under it", nil, []string{homeOption, bare, root}, "--niagara-home " + bare + ": no modules/ directory (not a Niagara installation)\n"},
		{"modules is a file", nil, []string{homeOption, file, root}, "--niagara-home " + file + ": no modules/ directory (not a Niagara installation)\n"},
		{"from the environment", map[string]string{homeEnv: bare}, []string{root}, "NIAGARA_HOME " + bare + ": no modules/ directory (not a Niagara installation)\n"},
		{"a module with no template", nil, []string{homeOption, bare, moduleRoot(t, nil)}, "--niagara-home " + bare + ": no modules/ directory (not a Niagara installation)\n"},
	}
	for _, c := range cases {
		if code, out, errText := runEnv(c.env, c.args...); code != 3 || out != "" || errText != "lint-ntpl-manifest-deps: "+c.want {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 3 and %q", c.name, code, out, errText, c.want)
		}
	}
	if os.Geteuid() == 0 {
		return // root lists every directory
	}
	locked := installation(t, stock...)
	if err := os.Chmod(locked+"/modules", 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked+"/modules", 0o755) })
	code, out, errText := run(homeOption, locked, root)
	if code != 3 || out != "" || !strings.HasPrefix(errText, "lint-ntpl-manifest-deps: --niagara-home "+locked+": cannot list modules/: ") {
		t.Errorf("an unreadable modules/: exit %d stdout %q stderr %q, want exit 3 and the cannot-list line", code, out, errText)
	}
}

// TestInstallationModuleNames: a module is the name of a jar under modules/, part suffix kept: fox-rt.jar
// ships fox-rt, not fox and not fox-wb. A file that is no jar, a dot file and a jar in a sub-directory
// ship nothing. An installation that ships no module is still an installation (every dependency fails).
func TestInstallationModuleNames(t *testing.T) {
	t.Parallel()
	nh := installation(t, "fox-rt")
	write(t, nh, "modules/readme-rt.txt", "")
	write(t, nh, "modules/.hidden-rt.jar", "")
	write(t, nh, "modules/sub/nested-rt.jar", "")
	root := moduleRoot(t, map[string]string{tpl: template(t, "", "fox-rt", "fox", "fox-wb", "readme-rt", "readme-rt.txt", ".hidden-rt", "nested-rt", "sub")})
	code, out, _ := run(homeOption, nh, root)
	if want := ": .hidden-rt, fox, fox-wb, nested-rt, readme-rt, readme-rt.txt, sub\n"; code != 1 || !strings.HasSuffix(out, want) {
		t.Errorf("exit %d stdout %q, want exit 1 ending %q", code, out, want)
	}
	empty := t.TempDir()
	if err := os.Mkdir(empty+"/modules", 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run(homeOption, empty, root); code != 1 || !strings.Contains(out, "FAIL  ntpl-deps  ") || !strings.Contains(out, "fox-rt") {
		t.Errorf("an installation with no module: exit %d stdout %q, want FAIL rows naming fox-rt", code, out)
	}
}

// TestEmptyInstallationOptionIsAUsageError (K7): --niagara-home "" names no installation; it is a usage
// error (the lint's usage exit, 3, nothing checked), never a run that drops to "no installation" and WARN.
// The last value still wins, and an empty NIAGARA_HOME stays "no installation" (TestWithoutAnInstallation).
func TestEmptyInstallationOptionIsAUsageError(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{tpl: template(t, "", "ghost-rt")})
	want := "lint-ntpl-manifest-deps: --niagara-home: an empty value names no installation\nusage: lint-ntpl-manifest-deps.sh [--niagara-home <dir>] <module-root | template-module.jar>\n"
	nh := installation(t, stock...)
	for name, args := range map[string][]string{"alone": {homeOption, "", root}, "after a path": {homeOption, nh, homeOption, "", root}} {
		if code, out, errText := run(args...); code != 3 || out != "" || errText != want {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 3 and %q", name, code, out, errText, want)
		}
	}
	if code, out, _ := run(homeOption, "", homeOption, nh, root); code != 1 || !strings.HasSuffix(out, ": ghost-rt\n") {
		t.Errorf("an empty value then a path: exit %d stdout %q, want the path's run (exit 1 naming ghost-rt)", code, out)
	}
}

// TestShippedModuleIsARegularReadableJar (K7): a jar name counts as shipped only when it is a regular file
// the lint can open: a dangling link, a directory and an unreadable file named <module>.jar ship nothing,
// while a link to a readable jar ships its name.
func TestShippedModuleIsARegularReadableJar(t *testing.T) {
	t.Parallel()
	nh := installation(t, "baja", "real-rt")
	symlink(t, nh+"/modules/gone.jar", nh+"/modules/dangling-rt.jar")
	if err := os.Mkdir(nh+"/modules/dir-rt.jar", 0o755); err != nil {
		t.Fatal(err)
	}
	symlink(t, nh+"/modules/real-rt.jar", nh+"/modules/linked-rt.jar")
	deps := []string{"dangling-rt", "dir-rt", "linked-rt", "real-rt"}
	want := ": dangling-rt, dir-rt\n"
	if os.Geteuid() != 0 {
		write(t, nh, "modules/locked-rt.jar", "")
		if err := os.Chmod(nh+"/modules/locked-rt.jar", 0); err != nil {
			t.Fatal(err)
		}
		deps, want = append(deps, "locked-rt"), ": dangling-rt, dir-rt, locked-rt\n"
	}
	root := moduleRoot(t, map[string]string{tpl: template(t, "", deps...)})
	if code, out, errText := run(homeOption, nh, root); code != 1 || !strings.HasSuffix(out, want) || errText != "" {
		t.Errorf("exit %d stdout %q stderr %q, want exit 1 ending %q", code, out, errText, want)
	}
}
