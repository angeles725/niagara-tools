package modulemap

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	mmrule "github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
)

func where(o opts, s cli.Stdio) error {
	m, err := readMap(o.root)
	if err != nil {
		return err
	}
	var hits []mmrule.Row
	if r, ok := findRow(m, o.target); ok {
		hits = append(hits, r)
	} else {
		rel := o.target
		if filepath.IsAbs(rel) {
			absRoot, _ := filepath.Abs(o.root)
			if r, err := filepath.Rel(absRoot, rel); err == nil {
				rel = r
			}
		}
		rel = path.Clean(filepath.ToSlash(rel))
		for _, r := range m.Rows {
			if lists(r.Files, rel) || lists(r.Tests, rel) {
				hits = append(hits, r)
			}
		}
	}
	if len(hits) == 0 {
		return fail(1, "module-map: no concern of %s lists %s", lint.MapFile, o.target)
	}
	for i, r := range hits {
		if i > 0 {
			fmt.Fprintln(s.Out)
		}
		fmt.Fprintf(s.Out, "concern  %s\npart     %s\n", r.Concern, r.Part)
		for _, f := range r.Files {
			fmt.Fprintf(s.Out, "files    %s\n", f)
		}
		for _, f := range r.Tests {
			fmt.Fprintf(s.Out, "tests    %s\n", f)
		}
		fmt.Fprintf(s.Out, "change   %s\n", r.When)
	}
	return nil
}

// lists: an entry of cell names rel exactly or is a glob matching it.
func lists(cell []string, rel string) bool {
	for _, e := range cell {
		if path.Clean(e) == rel || (strings.ContainsAny(e, "*?[") && mmrule.MatchGlob(e, rel)) {
			return true
		}
	}
	return false
}
