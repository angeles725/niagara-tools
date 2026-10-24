package wiringmap

import (
	"os"
	"strings"
)

// awkSpace is awk's [[:space:]] in the C locale.
const awkSpace = " \t\n\v\f\r"

// readLexicon reads the display names of a module lexicon (none for ""): each line that is not a comment
// ("#" after optional blanks) and holds a "=" is a key (the text before the first "=") and a value (the
// rest), both trimmed of blanks; the first line of a key wins and a "|" in a value is escaped as `\|` for the
// table cell. An unreadable lexicon reads as empty.
func readLexicon(path string) map[string]string {
	names := map[string]string{}
	if path == "" {
		return names
	}
	data, _ := os.ReadFile(path)
	for _, ln := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		k, v, ok := strings.Cut(ln, "=")
		if !ok || strings.HasPrefix(strings.TrimLeft(ln, awkSpace), "#") {
			continue
		}
		k, v = strings.Trim(k, awkSpace), strings.Trim(v, awkSpace)
		if _, seen := names[k]; k != "" && !seen {
			names[k] = strings.ReplaceAll(v, "|", `\|`)
		}
	}
	return names
}

// row is one table row: the slot and its Workbench display name.
type row struct{ slot, display string }

// rows pairs each slot with its display name ("_(no lexicon key)_" when the lexicon has no bare slot key), as
// the bash `awk -F'\t' | read -r slot disp` pipeline did: the slot is cut at its first tab, and the read then
// trims tabs from both ends of the line and from the start of the display name.
func rows(slots []string, names map[string]string) []row {
	var out []row
	for _, s := range slots {
		key, _, _ := strings.Cut(s, "\t")
		d, ok := names[key]
		if !ok {
			d = "_(no lexicon key)_"
		}
		slot, disp, _ := strings.Cut(strings.Trim(key+"\t"+d, "\t"), "\t")
		out = append(out, row{slot, strings.TrimLeft(disp, "\t")})
	}
	return out
}
