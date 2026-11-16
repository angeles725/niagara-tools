package ntplmanifest

import (
	"os"
	"strings"
	"testing"
)

const tpl = "ModA-rt/src/templates/Unit.ntpl"

// TestCleanModule: a template with both entries, a well-formed manifest and only declared dependencies (a
// declared module, the nre, the part itself) prints nothing; a module that ships no template is one PASS
// row. Both exit 0.
func TestCleanModule(t *testing.T) {
	t.Parallel()
	expect(t, "every dependency declared", moduleRoot(t, map[string]string{tpl: template(t, "", "baja", "control-rt", "ModA-rt", "nre")}), 0, "")
	expect(t, "no dependency at all", moduleRoot(t, map[string]string{tpl: template(t, "")}), 0, "")
	expect(t, "no template", moduleRoot(t, nil), 0, "PASS  lint-ntpl-manifest-deps  @ROOT@: no *.ntpl under the root (nothing to check)\n")
	expect(t, "a sibling part declares it", moduleRoot(t, map[string]string{
		tpl: template(t, "", "web-rt", "ModA-ux"), "ModA-ux/ModA-ux.gradle.kts": "dependencies { api(\":web-rt\") }\n"}), 0, "")
}

// TestMissingEntry (NTPL1): an archive without template.bog or without template-manifest.xml fails, each
// absent entry named.
func TestMissingEntry(t *testing.T) {
	t.Parallel()
	expect(t, "no template.bog", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{manifestEntry, manifestXML("")})}), 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": lacks template.bog\n")
	expect(t, "no manifest", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog})}), 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": lacks template-manifest.xml\n")
	expect(t, "neither, and the entries are matched by their whole name", moduleRoot(t, map[string]string{
		tpl: ntpl(t, entry{"px/template.bog", bog}, entry{"template.manifest.xml", manifestXML("")})}), 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": lacks template.bog\nFAIL  ntpl-entries  @ROOT@/"+tpl+": lacks template-manifest.xml\n")
	expect(t, "two manifests", moduleRoot(t, map[string]string{
		tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, manifestXML("")}, entry{manifestEntry, manifestXML("", "ghost-rt")})}), 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": holds 2 entries named template-manifest.xml (which one a reader takes is not defined)\n")
}

// TestMalformedManifest (NTPL2): a manifest that is not well-formed XML fails with its position, and is
// never read as a manifest with no dependencies.
func TestMalformedManifest(t *testing.T) {
	t.Parallel()
	bad := "<templateManifest>\n  <dependencies>\n    <dependency name=\"ghost-rt\">\n  </dependencies>\n</templateManifest>\n"
	expect(t, "an unclosed element", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, bad})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml is not well-formed XML: 4:3: element <dependency> opened at 3:5 closed by </dependencies>\n")
	expect(t, "an empty manifest", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, ""})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml is not well-formed XML: 1:1: no root element\n")
	nameless := strings.Replace(manifestXML("", "baja"), `name="baja" `, "", 1)
	expect(t, "a dependency with no name", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, nameless})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml:4: a <dependency> with no name\n")
}

// TestUndeclaredDependency (NTPL3): a <dependency> naming a module no build file of the module declares
// fails; the names are listed once each, sorted. A call in a comment, in settings.gradle.kts or in a build
// output declares nothing, and a module with no build file at all cannot be compared.
func TestUndeclaredDependency(t *testing.T) {
	t.Parallel()
	expect(t, "two undeclared among declared ones", moduleRoot(t, map[string]string{tpl: template(t, "", "baja", "zeta-rt", "alpha-rt", "zeta-rt", "control-rt")}), 1,
		"FAIL  ntpl-deps  @ROOT@/"+tpl+": manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under @ROOT@: alpha-rt, zeta-rt\n")
	expect(t, "a comment, the settings file and a build output declare nothing", moduleRoot(t, map[string]string{
		tpl: template(t, "", "commented-rt", "settings-only-rt", "output-rt"), "ModA-rt/build/out.gradle.kts": "dependencies { api(\":output-rt\") }\n"}), 1,
		"FAIL  ntpl-deps  @ROOT@/"+tpl+": manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under @ROOT@: commented-rt, output-rt, settings-only-rt\n")
	bare := t.TempDir()
	write(t, bare, "templates/Unit.ntpl", template(t, "", "baja"))
	expect(t, "no build file", bare, 1, "FAIL  ntpl-deps  @ROOT@/templates/Unit.ntpl: no *.gradle.kts under @ROOT@: its 1 manifest dependencies cannot be compared\n")
}

// TestStationTemplate (NTPL4, K2): a template under a newStations directory is a station template; its
// manifest root must carry isStation="true". Elsewhere the attribute is not required.
func TestStationTemplate(t *testing.T) {
	t.Parallel()
	const station = "overlay/defaults/workbench/newStations/Plant.ntpl"
	expect(t, "no isStation", moduleRoot(t, map[string]string{station: template(t, "", "baja")}), 1,
		"FAIL  ntpl-station  @ROOT@/"+station+": a template under newStations/ is a station template, but its manifest root <templateManifest> lacks isStation=\"true\"\n")
	expect(t, "isStation false", moduleRoot(t, map[string]string{station: template(t, ` isStation="false"`, "baja")}), 1,
		"FAIL  ntpl-station  @ROOT@/"+station+": a template under newStations/ is a station template, but its manifest root <templateManifest> lacks isStation=\"true\"\n")
	expect(t, "isStation true", moduleRoot(t, map[string]string{station: template(t, ` isStation="true"`, "baja")}), 0, "")
	expect(t, "a file named like the directory is no station directory", moduleRoot(t, map[string]string{"ModA-rt/src/newStations.ntpl": template(t, "")}), 0, "")
}

// TestRowsInPathOrder: every archive is judged, in path order, over several roots; a template in a part's
// build output or in a dot directory is not shipped source and is not read.
func TestRowsInPathOrder(t *testing.T) {
	t.Parallel()
	a := moduleRoot(t, map[string]string{
		"ModA-rt/src/b.ntpl":     ntpl(t, entry{bogEntry, bog}),
		"ModA-rt/src/a.ntpl":     template(t, "", "ghost-rt"),
		"ModA-rt/build/out.ntpl": "not a zip",
		".cache/c.ntpl":          "not a zip",
	})
	b := moduleRoot(t, nil)
	want := "FAIL  ntpl-deps  " + a + "/ModA-rt/src/a.ntpl: manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under " + a + ": ghost-rt\n" +
		"FAIL  ntpl-entries  " + a + "/ModA-rt/src/b.ntpl: lacks template-manifest.xml\n" +
		"PASS  lint-ntpl-manifest-deps  " + b + ": no *.ntpl under the root (nothing to check)\n"
	if code, out, errText := run("--", a, b); code != 1 || out != want || errText != "" {
		t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit 1\nstdout %q", code, out, errText, want)
	}
}

// TestUsageAndEmptyRoot: no root is the usage line; a root with neither a build file nor a template is the
// ERROR row; both exit 3, never a pass.
func TestUsageAndEmptyRoot(t *testing.T) {
	t.Parallel()
	if code, out, errText := run(); code != 3 || out != "" || errText != "usage: lint-ntpl-manifest-deps.sh [--] <module-root>...\n" {
		t.Errorf("no root: exit %d stdout %q stderr %q", code, out, errText)
	}
	empty := t.TempDir()
	write(t, empty, "README.txt", "notes\n")
	expect(t, "an empty root", empty, 3, "ERROR  lint-ntpl-manifest-deps  @ROOT@  no *.gradle.kts and no *.ntpl under the root (wrong path: pass a module root)\n")
}

// TestUnreadableInputIsExit3: an archive or a build file that cannot be opened fails the scan closed after
// the rows of what was read.
func TestUnreadableInputIsExit3(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := moduleRoot(t, map[string]string{"ModA-rt/src/a.ntpl": ntpl(t, entry{bogEntry, bog}), "ModA-rt/src/locked.ntpl": template(t, "")})
	locked := root + "/ModA-rt/src/locked.ntpl"
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	wantOut := "FAIL  ntpl-entries  " + root + "/ModA-rt/src/a.ntpl: lacks template-manifest.xml\n"
	wantErr := "lint-ntpl-manifest-deps: cannot scan " + locked + ": open " + locked + ": permission denied\n"
	if code, out, errText := run(root); code != 3 || out != wantOut || errText != wantErr {
		t.Errorf("a locked archive: exit %d\nstdout %q\nstderr %q\nwant exit 3\nstdout %q\nstderr %q", code, out, errText, wantOut, wantErr)
	}
	root = moduleRoot(t, map[string]string{tpl: template(t, "", "ghost-rt")})
	build := root + "/ModA-rt/ModA-rt.gradle.kts"
	if err := os.Chmod(build, 0); err != nil {
		t.Fatal(err)
	}
	wantErr = "lint-ntpl-manifest-deps: cannot scan " + build + ": open " + build + ": permission denied\n"
	if code, out, errText := run(root); code != 3 || out != "" || errText != wantErr {
		t.Errorf("a locked build file: exit %d stdout %q stderr %q, want exit 3 and %q", code, out, errText, wantErr)
	}
}
