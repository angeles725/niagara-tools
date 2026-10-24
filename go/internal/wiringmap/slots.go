// Package wiringmap is generate-wiring-map (toolbelt/generate-wiring-map.sh): it scaffolds the wiring-map.md
// of a facade source tree, its OPERATOR slots in Table 1 and its SUMMARY/READONLY slots in Table 2, each row
// with the Workbench display name from the module lexicon, then the commissioning checklist.
package wiringmap

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// slotName is the first `name = "<slot>"` of an annotation, blanks as awk's [[:space:]] in the C locale. Any
// identifier ending in "name" matches, as in the bash scan.
var slotName = regexp.MustCompile(`name[ \t\n\v\f\r]*=[ \t\n\v\f\r]*"([^"]+)"`)

// Slots returns the slot names of the @NiagaraProperty annotations of one Java source, in source order:
// op those whose annotation text holds "OPERATOR", sum the others that hold "SUMMARY" or "READONLY". An
// annotation is joined from its "@NiagaraProperty" line until a "(" was seen and the parentheses balance,
// counted as text (javasrc.NiagaraProperties); one with no name is skipped.
func Slots(src string) (op, sum []string) {
	lines, _ := javasrc.Lines(strings.NewReader(src))
	javasrc.NiagaraProperties(lines, func(text string, _ int) {
		m := slotName.FindStringSubmatch(text)
		switch {
		case m == nil:
		case strings.Contains(text, "OPERATOR"):
			op = append(op, m[1])
		case strings.Contains(text, "SUMMARY") || strings.Contains(text, "READONLY"):
			sum = append(sum, m[1])
		}
	})
	return op, sum
}
