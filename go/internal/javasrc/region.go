package javasrc

import "strings"

// The text of the two markers Slot-o-Matic writes around the code it generates, on a line comment
// (`//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/` …
// `//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/`) or in a block
// comment (devkit-wb com/tridium/slottool/JavaUnit.java, vineflower 4.14.0.162, :479-497).
const (
	RegionBegin = "BEGIN BAJA AUTO GENERATED CODE"
	RegionEnd   = "END BAJA AUTO GENERATED CODE"
)

// Marker reports whether line i (0-based) of f holds the marker text m inside a comment: m is in the
// raw line and not in the stripped one, so m quoted in a string literal or a text block is code, not a
// marker. Neither marker text contains the other, so a line is a BEGIN or an END marker, not both.
func (f *File) Marker(i int, m string) bool {
	return strings.Contains(f.Raw[i], m) && !strings.Contains(f.Stripped()[i], m)
}
