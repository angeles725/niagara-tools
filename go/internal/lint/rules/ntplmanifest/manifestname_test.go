package ntplmanifest

import "testing"

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
