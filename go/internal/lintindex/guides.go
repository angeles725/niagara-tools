package lintindex

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lintheader"
)

// The Guide and Doc columns and the --check-guides gate: a lint that checks module code is named in the
// types/*.md guide that holds its rule; a lint that checks the kit itself declares `# Scope: kit` in its header.

// docWindow is how many lines either side of a guide line that names a lint may hold its citation.
const docWindow = 6

// Guide is one types/*.md guide: Name is its kit-relative path (types/x.md), Text its content.
type Guide struct{ Name, Text string }

var (
	// A citation of the official help: the evidence tag or a path into the niagara-help tree.
	docCite = regexp.MustCompile(`\[CERT-doc|niagara-help/`)
	// `# Scope: kit` (alone or followed by a dash and a sentence): the lint checks the kit, not module code.
	kitScope = regexp.MustCompile(`^#[[:space:]]*Scope:[[:space:]]*kit($|[^A-Za-z0-9_-])`)
)

// ReadGuides reads every types/*.md of the kit root, in name order. A root with no types/ has none.
func ReadGuides(root string) ([]Guide, error) {
	paths, err := filepath.Glob(filepath.Join(root, "types", "*.md"))
	if err != nil {
		return nil, err
	}
	var out []Guide
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, Guide{"types/" + filepath.Base(p), string(b)})
	}
	return out, nil
}

// GuideRefs returns the `types/<guide>.md:<line>` of every line of guides that names script (its base
// name, with or without .sh, not as part of a longer lint name or a longer word), and the line of the
// nearest citation of the official help (`[CERT-doc` or a `niagara-help/` path) within docWindow lines of each,
// both in guide order and without repeats.
func GuideRefs(guides []Guide, script string) (guide, doc []string) {
	name := regexp.QuoteMeta(strings.TrimSuffix(script, ".sh"))
	named := regexp.MustCompile(`(^|[^A-Za-z0-9_-])` + name + `(\.sh)?($|[^A-Za-z0-9_-])`)
	for _, g := range guides {
		lines := strings.Split(g.Text, "\n")
		seen := map[int]bool{}
		for i, line := range lines {
			if !named.MatchString(line) {
				continue
			}
			guide = append(guide, fmt.Sprintf("%s:%d", g.Name, i+1))
			if c := nearestCite(lines, i); c >= 0 && !seen[c] {
				seen[c] = true
				doc = append(doc, fmt.Sprintf("%s:%d", g.Name, c+1))
			}
		}
	}
	return guide, doc
}

// nearestCite is the index of the line within docWindow of lines[at] that cites the official help (the
// earlier one on a tie), or -1.
func nearestCite(lines []string, at int) int {
	for d := 0; d <= docWindow; d++ {
		for _, i := range []int{at - d, at + d} {
			if i >= 0 && i < len(lines) && docCite.MatchString(lines[i]) {
				return i
			}
		}
	}
	return -1
}

// KitScope reports whether the header of a lint script (its text) declares `# Scope: kit`.
func KitScope(text string) bool {
	for _, line := range lintheader.Lines(text) {
		if kitScope.MatchString(line) {
			return true
		}
	}
	return false
}

// CheckGuides writes to diag one line per lint that no guide names and whose header does not declare
// `# Scope: kit`, and returns how many lints are named, how many are kit-scope without a guide, and how many
// are neither.
func CheckGuides(rows []Row, diag io.Writer) (named, kit, bad int) {
	sorted := append([]Row(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, r := range sorted {
		switch {
		case r.Guide != "":
			named++
		case r.Kit:
			kit++
		default:
			bad++
			fmt.Fprintf(diag, "gen-lint-index: %s: named by no types/*.md guide (name it in the guide section of its rule, "+
				"or declare \"# Scope: kit\" in its header)\n", r.Name)
		}
	}
	return named, kit, bad
}
