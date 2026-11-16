package checkclientsource

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
)

// BUILD-STATE envelope keys this check reads. The envelope markers and the line split are the ledger
// reader's (go/internal/buildstatefields).
const (
	moduleKey = "module:"
	sourceKey = "source_of_truth:"
	blanks    = " \t\n\v\f\r" // awk's [[:space:]] in the C locale
)

// SourceOfTruth is the source_of_truth value of module in a BUILD-STATE text, "" when it declares none:
// the first `source_of_truth:` line that follows the `module:` line naming the module inside a
// build-state.v1 envelope. Keys start at column 0. A line may end in CRLF: its CR is not part of a value.
func SourceOfTruth(text, module string) string {
	inside, current := false, ""
	for _, line := range buildstatefields.Lines(text) {
		// Mutation: CS-crlf -- keeping the CR of a CRLF line leaves it on every value, so no module is ever named (WARN undeclared, exit 0)
		line = strings.TrimSuffix(line, "\r")
		switch {
		case strings.HasPrefix(line, buildstatefields.OpenMarker):
			inside, current = true, ""
		case strings.HasPrefix(line, buildstatefields.CloseMarker):
			inside = false
		case inside && strings.HasPrefix(line, moduleKey):
			current = fieldValue(line, moduleKey)
		case inside && current == module && strings.HasPrefix(line, sourceKey):
			return fieldValue(line, sourceKey)
		}
	}
	return ""
}

// fieldValue is the value of a `key: value  # comment` line: the blanks after the key, and a `#` comment
// with the blanks before it, are removed. Trailing blanks with no comment are kept.
func fieldValue(line, key string) string {
	v := strings.TrimLeft(strings.TrimPrefix(line, key), blanks)
	if i := strings.IndexByte(v, '#'); i >= 0 {
		v = strings.TrimRight(v[:i], blanks)
	}
	return v
}
