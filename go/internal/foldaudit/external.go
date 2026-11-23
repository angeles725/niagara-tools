package foldaudit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/retrorows"
)

// external audits the retros of the external roots (--external-root, fold-roots.conf): the integration check of the
// kit also looks at retros other projects wrote. A retro counts when some delta row names mcp-n4 or build-n4-module
// (an n4_* tool, build-n4-module, kit, toolbelt, BUILD-LOOP, go/internal: retrorows.Area); each such row that is not
// applied must be cited in the kit corpus as `[ev: retro <name> Δ<id>]` or `[deferred: retro <name> Δ<id>]`, or be
// tracked by a closed issue (a `#N` on the row, or a closed retro-delta issue carrying the marker `<name> Δ<id>`).
// <name> is the alias when one is given, so a client-named stem never prints into a CI log or reaches GitHub.
// It returns the stderr text of an unusable root, "" otherwise.
func (t *audit) external() string {
	if len(t.o.externals) == 0 {
		return ""
	}
	retros, checked, missing, unaliased := 0, 0, 0, 0
	for i, dir := range t.o.externals {
		k := 0
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Sprintf("fold-audit: external root #%d not found or unreadable\n", i+1)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "INDEX.md" {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			k++
			name := t.externalName(e.Name())
			aliased := name != ""
			if !aliased {
				name = fmt.Sprintf("root%d-retro%d", i+1, k)
			}
			counted := false
			for _, r := range retrorows.Parse(string(b)) {
				if r.Done != "" || retrorows.Area(r, "", "") == "" {
					continue
				}
				if !counted {
					counted = true
					retros++
					if !aliased {
						unaliased++
					}
				}
				checked++
				if !t.externalCited(name, r) {
					fmt.Fprintf(t.s.Out, "fold-audit: WARN external %s %s not cited\n", name, r.ID)
					missing++
					t.warned = true
				}
			}
		}
	}
	if unaliased > 0 {
		fmt.Fprintf(t.s.Out, "fold-audit: WARN external: no --alias for %d retro(s): named root<N>-retro<K>, never by stem; "+
			"give each an alias so its citations can match\n", unaliased)
	}
	fmt.Fprintf(t.s.Out, "fold-audit: external: %d retros, %d deltas checked, %d not cited\n", retros, checked, missing)
	return ""
}

// externalName is the alias of a retro file (given by its stem or its dated name), else "": a stem may name a client,
// so it is never printed.
func (t *audit) externalName(file string) string {
	base := strings.TrimSuffix(file, ".md")
	if a := t.o.aliases[base]; a != "" {
		return a
	}
	if a := t.o.aliases[retrorows.Stem(file)]; a != "" {
		return a
	}
	return ""
}

// externalCited reports whether the kit accounts for the delta; a lookup gh cannot answer is reported UNVERIFIED and
// counts as not cited.
func (t *audit) externalCited(name string, r retrorows.Row) bool {
	if t.h.deltas[name+" "+r.ID] {
		return true
	}
	if r.Issue != 0 {
		closed, err := t.gh.issueClosed(r.Issue)
		if err != nil {
			t.unverifiedLine("external "+name, r.ID, fmt.Sprintf("issue #%d state unavailable", r.Issue), err)
		}
		return closed
	}
	closed, err := t.gh.markerClosed(name + " " + r.ID)
	if err != nil {
		t.unverifiedLine("external "+name, r.ID, "closed-issue lookup unavailable", err)
	}
	return closed
}
