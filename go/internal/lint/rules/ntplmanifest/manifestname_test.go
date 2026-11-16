package ntplmanifest

import (
	"strings"
	"testing"
)

// K4b: the Templates guide spells the manifest template.manifest.xml (niagara-help 4.13.2.18 Templates,
// "Example template.manifest.xml file"); shipped archives use template-manifest.xml. An archive with only the
// dotted name is read through it, and one with both is read through the hyphenated name alone.
func TestDottedManifestNameIsAFallback(t *testing.T) {
	dotted := moduleRoot(t, map[string]string{
		"ModA-rt/templates/Dotted.ntpl": ntpl(t, entry{bogEntry, bog}, entry{manifestAlt, manifestXML("", "baja", "local-rt", "far-rt")}),
		"ModA-rt/templates/Both.ntpl": ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, manifestXML("", "baja")},
			entry{manifestAlt, manifestXML("", "not-read-rt")}),
	})
	expect(t, "dotted name", dotted, 1, undeclaredRow("ModA-rt/templates/Dotted.ntpl", "far-rt"))
}

// A row about the manifest names the entry that was read: template.manifest.xml when the fallback was used
// (a malformed document, a wrong root, a repeated entry, a module the bog uses that no <dependency> names).
func TestRowsNameTheManifestRead(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{
		"ModA-rt/templates/Bad.ntpl":   ntpl(t, entry{bogEntry, bog}, entry{manifestAlt, "<template>"}),
		"ModA-rt/templates/Root.ntpl":  ntpl(t, entry{bogEntry, bog}, entry{manifestAlt, "<other/>\n"}),
		"ModA-rt/templates/Twice.ntpl": ntpl(t, entry{bogEntry, bog}, entry{manifestAlt, manifestXML("")}, entry{manifestAlt, manifestXML("")}),
		"ModA-rt/templates/Uses.ntpl":  ntpl(t, entry{bogEntry, usesBog}, entry{manifestAlt, manifestXML("", "control-rt")}),
	})
	code, out, _ := run(homeOption, installation(t, stock...), root)
	for _, want := range []string{
		root + "/ModA-rt/templates/Bad.ntpl: template.manifest.xml is not well-formed XML: ",
		root + "/ModA-rt/templates/Root.ntpl: template.manifest.xml: the root element is <other>",
		root + "/ModA-rt/templates/Twice.ntpl: holds 2 entries named template.manifest.xml",
		root + "/ModA-rt/templates/Uses.ntpl: template.bog uses kitIo, vendorw, vendorx: no <dependency> of template.manifest.xml names them",
	} {
		if code != 1 || !strings.Contains(out, want) {
			t.Errorf("exit %d stdout %q\nwant exit 1 and a row holding %q", code, out, want)
		}
	}
}
