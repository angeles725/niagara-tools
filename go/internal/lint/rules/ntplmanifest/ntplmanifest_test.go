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
	expect(t, "every dependency declared", moduleRoot(t, map[string]string{tpl: template(t, "", "baja", "local-rt", "ModA-rt", "nre")}), 0, "")
	expect(t, "no dependency at all", moduleRoot(t, map[string]string{tpl: template(t, "")}), 0, "")
	expect(t, "no template", moduleRoot(t, nil), 0, "PASS  lint-ntpl-manifest-deps  @ROOT@: no *.ntpl under the root (nothing to check)\n")
	expect(t, "a sibling part declares it", moduleRoot(t, map[string]string{
		tpl: template(t, "", "other-rt", "ModA-ux"), "ModA-ux/ModA-ux.gradle.kts": "dependencies { api(\":other-rt\") }\n"}), 0, "")
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
		tpl: ntpl(t, entry{"px/template.bog", bog}, entry{"px/" + manifestEntry, manifestXML("")})}), 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": lacks template.bog\nFAIL  ntpl-entries  @ROOT@/"+tpl+": lacks template-manifest.xml\n")
	expect(t, "two manifests", moduleRoot(t, map[string]string{
		tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, manifestXML("")}, entry{manifestEntry, manifestXML("", "ghost-rt")})}), 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": holds 2 entries named template-manifest.xml (which one a reader takes is not defined)\n")
}

// TestMalformedManifest (NTPL2): a manifest that is not well-formed XML fails with its position, and is
// never read as a manifest with no dependencies.
func TestMalformedManifest(t *testing.T) {
	t.Parallel()
	bad := "<template>\n  <dependencies>\n    <dependency name=\"ghost-rt\">\n  </dependencies>\n</template>\n"
	expect(t, "an unclosed element", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, bad})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml is not well-formed XML: 4:3: element <dependency> opened at 3:5 closed by </dependencies>\n")
	expect(t, "an empty manifest", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, ""})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml is not well-formed XML: 1:1: no root element\n")
	nameless := strings.Replace(manifestXML("", "baja"), `name="baja" `, "", 1)
	expect(t, "a dependency with no name", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, nameless})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml:4: a <dependency> with no name\n")
}

// TestManifestThatNoReaderTakes: a well-formed document that is not a template manifest, and a
// <dependency> outside <dependencies>, fail ntpl-manifest. Neither is a silent pass: the undeclared module
// such a manifest names would otherwise go unreported.
func TestManifestThatNoReaderTakes(t *testing.T) {
	t.Parallel()
	other := "<project>\n  <dependencies>\n    <dependency name=\"ghost-rt\"/>\n  </dependencies>\n</project>\n"
	expect(t, "another root element", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, other})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml: the root element is <project>, not <template> -- not a template manifest\n")
	stray := "<template>\n  <dependency name=\"ghost-rt\"/>\n  <resources>\n    <dependency name=\"other-rt\"/>\n  </resources>\n  <dependencies>\n    <dependency name=\"baja\"/>\n  </dependencies>\n</template>\n"
	expect(t, "a dependency under the root and one under another element", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, stray})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml:2: a <dependency> outside <dependencies> (no reader takes it for a dependency)\n"+
			"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml:4: a <dependency> outside <dependencies> (no reader takes it for a dependency)\n")
}

// TestUndeclaredDependency (NTPL3): with the installation's module list, a <dependency> naming a module
// that no build file of the module declares and the installation does not ship fails; the names are listed
// once each, sorted. A call in a comment, in settings.gradle.kts or in a build output declares nothing, and
// a module with no build file declares nothing either.
func TestUndeclaredDependency(t *testing.T) {
	t.Parallel()
	expect(t, "two undeclared among declared and stock ones", moduleRoot(t, map[string]string{tpl: template(t, "", "baja", "zeta-rt", "fox-rt", "alpha-rt", "zeta-rt", "local-rt")}), 1,
		undeclaredRow(tpl, "alpha-rt, zeta-rt"))
	expect(t, "a comment, the settings file and a build output declare nothing", moduleRoot(t, map[string]string{
		tpl: template(t, "", "commented-rt", "settings-only-rt", "output-rt"), "ModA-rt/build/out.gradle.kts": "dependencies { api(\":output-rt\") }\n"}), 1,
		undeclaredRow(tpl, "commented-rt, output-rt, settings-only-rt"))
	bare := t.TempDir()
	write(t, bare, "templates/Unit.ntpl", template(t, "", "baja", "ghost-rt"))
	expect(t, "no build file", bare, 1, undeclaredRow("templates/Unit.ntpl", "ghost-rt"))
}

// TestSelfDeclaredPart: a part is a dependency of its own templates without declaring itself, by the name
// of the build file's directory (ModB-rt/build.gradle.kts) and by the build file's stem
// (parts/ModC-rt.gradle.kts), each on its own. The stem "build" names no module, and neither does a
// directory that holds no build file.
func TestSelfDeclaredPart(t *testing.T) {
	t.Parallel()
	expect(t, "the directory name alone", moduleRoot(t, map[string]string{
		tpl: template(t, "", "ModB-rt"), "ModB-rt/build.gradle.kts": "dependencies {}\n"}), 0, "")
	expect(t, "the file stem alone", moduleRoot(t, map[string]string{
		tpl: template(t, "", "ModC-rt"), "parts/ModC-rt.gradle.kts": "dependencies {}\n"}), 0, "")
	expect(t, "neither the stem build nor a directory without a build file", moduleRoot(t, map[string]string{
		tpl: template(t, "", "build", "templates", "src"), "ModB-rt/build.gradle.kts": "dependencies {}\n"}), 1,
		undeclaredRow(tpl, "build, src, templates"))
}

// TestStationTemplate (NTPL4, K2): a template under a newStations directory is a station template; its
// manifest root must carry isStation="true". Elsewhere the attribute is not required.
func TestStationTemplate(t *testing.T) {
	t.Parallel()
	const station = "overlay/defaults/workbench/newStations/Plant.ntpl"
	expect(t, "no isStation", moduleRoot(t, map[string]string{station: template(t, "", "baja")}), 1,
		"FAIL  ntpl-station  @ROOT@/"+station+": a template under newStations/ is a station template, but its manifest root <template> lacks isStation=\"true\"\n")
	expect(t, "isStation false", moduleRoot(t, map[string]string{station: template(t, ` isStation="false"`, "baja")}), 1,
		"FAIL  ntpl-station  @ROOT@/"+station+": a template under newStations/ is a station template, but its manifest root <template> lacks isStation=\"true\"\n")
	expect(t, "isStation true", moduleRoot(t, map[string]string{station: template(t, ` isStation="true"`, "baja")}), 0, "")
	expect(t, "a file named like the directory is no station directory", moduleRoot(t, map[string]string{"ModA-rt/src/newStations.ntpl": template(t, "")}), 0, "")
}

// TestRowsInPathOrder: every archive is judged, in path order; the extension counts in any letter case; a
// template in a part's build output or in a dot directory is not shipped source and is not read.
func TestRowsInPathOrder(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{
		"ModA-rt/src/b.ntpl":     ntpl(t, entry{bogEntry, bog}),
		"ModA-rt/src/a.ntpl":     template(t, "", "ghost-rt"),
		"ModA-rt/src/c.NTPL":     ntpl(t, entry{manifestEntry, manifestXML("")}),
		"ModA-rt/src/d.Ntpl":     "not a zip",
		"ModA-rt/src/e.ntpl.bak": "not a zip",
		"ModA-rt/build/out.ntpl": "not a zip",
		".cache/c.ntpl":          "not a zip",
	})
	expect(t, "four templates", root, 1, undeclaredRow("ModA-rt/src/a.ntpl", "ghost-rt")+
		"FAIL  ntpl-entries  @ROOT@/ModA-rt/src/b.ntpl: lacks template-manifest.xml\n"+
		"FAIL  ntpl-entries  @ROOT@/ModA-rt/src/c.NTPL: lacks template.bog\n"+
		"FAIL  ntpl-entries  @ROOT@/ModA-rt/src/d.Ntpl: not a zip archive (zip: not a valid zip file) -- archive refused\n")
}

// TestUsageAndEmptyRoot: no root, a second root and an unknown option are the usage line; a root with
// neither a build file nor a template is the ERROR row; all exit 3, never a pass.
func TestUsageAndEmptyRoot(t *testing.T) {
	t.Parallel()
	const wantUsage = "usage: lint-ntpl-manifest-deps.sh [--niagara-home <dir>] <module-root | template-module.jar>\n"
	root := moduleRoot(t, nil)
	for name, args := range map[string][]string{"no root": nil, "two roots": {root, root}, "an unknown option": {"--strict", root}, "an option with no value": {root, homeOption}} {
		if code, out, errText := run(args...); code != 3 || out != "" || errText != wantUsage {
			t.Errorf("%s: exit %d stdout %q stderr %q", name, code, out, errText)
		}
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

// TestUnreadableDirectoryWithTemplatesIsExit3: a directory the walk cannot enter fails the scan closed even
// when templates were found elsewhere: their rows are printed, the exit is 3, never 0 or 1.
func TestUnreadableDirectoryWithTemplatesIsExit3(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := moduleRoot(t, map[string]string{"ModA-rt/src/a.ntpl": template(t, ""), "ModA-rt/src/b.ntpl": ntpl(t, entry{bogEntry, bog}), "ModA-rt/locked/c.ntpl": "not a zip"})
	locked := root + "/ModA-rt/locked"
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	wantOut := "FAIL  ntpl-entries  " + root + "/ModA-rt/src/b.ntpl: lacks template-manifest.xml\n"
	wantErr := "lint-ntpl-manifest-deps: cannot list every file under " + root + ": find: '" + locked + "': Permission denied\n"
	if code, out, errText := run(root); code != 3 || out != wantOut || errText != wantErr {
		t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit 3\nstdout %q\nstderr %q", code, out, errText, wantOut, wantErr)
	}
	clean := moduleRoot(t, map[string]string{"ModA-rt/src/a.ntpl": template(t, ""), "ModA-rt/locked/c.ntpl": "not a zip"})
	lockedClean := clean + "/ModA-rt/locked"
	if err := os.Chmod(lockedClean, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(lockedClean, 0o755) })
	wantErr = "lint-ntpl-manifest-deps: cannot list every file under " + clean + ": find: '" + lockedClean + "': Permission denied\n"
	if code, out, errText := run(clean); code != 3 || out != "" || errText != wantErr {
		t.Errorf("only clean templates were read: exit %d stdout %q stderr %q, want exit 3 and %q", code, out, errText, wantErr)
	}
}
