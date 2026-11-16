package ntplmanifest

import (
	"strconv"
	"strings"
	"testing"
)

// jarAt writes a template module jar (entries in the order given) under a fresh directory and returns its path.
func jarAt(t *testing.T, entries ...entry) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "ModT.jar", ntpl(t, entries...))
	return dir + "/ModT.jar"
}

// K4b: a template module jar as input. Each *.ntpl or *.napl entry is a template archive, read from the jar
// within the archive bounds and checked as a shipped one is; its rows name <jar>!/<entry>. A jar declares no
// build dependencies, so a dependency is judged against the installation alone.
func TestTemplateModuleJarChecksEveryTemplateInside(t *testing.T) {
	jar := jarAt(t,
		entry{"META-INF/module.xml", "<module/>"},
		entry{"templates/Good.ntpl", template(t, "", "baja", "fox-rt")},
		entry{"templates/Needs.napl", template(t, "", "baja", "local-rt")},
		entry{"templates/newStations/Site.ntpl", template(t, "")},
		entry{"templates/Broken.ntpl", ntpl(t, entry{manifestEntry, manifestXML("")})},
	)
	expect(t, "jar", jar, 1,
		"FAIL  ntpl-entries  @ROOT@!/templates/Broken.ntpl: lacks template.bog\n"+
			"FAIL  ntpl-deps  @ROOT@!/templates/Needs.napl: manifest dependencies not shipped by the installation @HOME@ (a template module jar declares no build dependencies): local-rt\n"+
			"FAIL  ntpl-station  @ROOT@!/templates/newStations/Site.ntpl: a template under newStations/ is a station template, but its manifest root <template> lacks isStation=\"true\"\n")
}

// A jar with no template entry is one PASS row; a jar that is not a zip is one ntpl-entries row on the jar.
func TestTemplateModuleJarWithoutTemplatesOrNotAZip(t *testing.T) {
	expect(t, "no template", jarAt(t, entry{"META-INF/module.xml", "<module/>"}), 0,
		"PASS  lint-ntpl-manifest-deps  @ROOT@: no *.ntpl or *.napl entry in the jar (nothing to check)\n")
	dir := t.TempDir()
	write(t, dir, "Bad.jar", "not a zip")
	c, out, _ := run(homeOption, installation(t, stock...), dir+"/Bad.jar")
	if c != 1 || !strings.HasPrefix(out, "FAIL  ntpl-entries  "+dir+"/Bad.jar: not a zip archive (") {
		t.Errorf("not a zip: exit %d, stdout %q", c, out)
	}
}

// The bounds: a template entry that inflates past the per-template bound is one row and is not read; past
// the run's budget, which each template and its entries draw on, the first template not read is named and
// nothing more of the jar is read.
func TestTemplateModuleJarBounds(t *testing.T) {
	defer func(one, all int64) { maxJarTemplate, maxInflate = one, all }(maxJarTemplate, maxInflate)
	small := template(t, "", "baja")
	inside := len(bog) + len(manifestXML("", "baja")) // what reading one small template inflates past itself
	maxJarTemplate, maxInflate = int64(len(small)), int64(2*(len(small)+inside))
	big := template(t, "", stock...) // a longer manifest: a larger archive than small
	if len(big) <= len(small) {
		t.Fatalf("the big template (%d bytes) is not larger than the small one (%d)", len(big), len(small))
	}
	jar := jarAt(t, entry{"a/Big.ntpl", big}, entry{"b/One.ntpl", small}, entry{"c/Two.ntpl", small}, entry{"d/Three.ntpl", small})
	expect(t, "bounds", jar, 1,
		"FAIL  ntpl-entries  @ROOT@!/a/Big.ntpl: inflates past "+strconv.FormatInt(maxJarTemplate, 10)+" bytes -- not read\n"+
			"FAIL  ntpl-entries  @ROOT@!/d/Three.ntpl: the templates and their entries inflate past "+strconv.FormatInt(maxInflate, 10)+" bytes in all, the bound of one run -- the template and everything after it not read\n")
}
