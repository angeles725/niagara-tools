package foldaudit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/retrorows"
)

// ResearchRetros is the retro directory of the research repository, looked for when no external root is configured.
const ResearchRetros = "niagara-research/retros"

// degraded reports the typed state external-roots-unconfigured: no --external-root and no fold-roots.conf root, yet a
// niagara-research/retros directory (in $HOME, or beside the kit's repository) holds retros with a pending kit delta.
// The external audit did not run, so a clean-looking report would be a silent zero; the line says so and fails
// --strict. --no-research-probe skips it (CI's step and its local twin, whose verdict must not depend on the
// machine's research checkout). It names the directory by its label, never by its path.
func (t *audit) degraded() {
	if len(t.o.externals) > 0 || t.o.noResearchProbe {
		return
	}
	for _, dir := range researchCandidates(t.o.root, t.s.Env("HOME")) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		retros, deltas := 0, 0
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "INDEX.md" {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			n := 0
			for _, r := range retrorows.Parse(string(b)) {
				if r.Done == "" && retrorows.Area(r, "", "") != "" {
					n++
				}
			}
			if n > 0 {
				retros++
				deltas += n
			}
		}
		if retros > 0 {
			fmt.Fprintf(t.s.Out, "fold-audit: DEGRADED external-roots-unconfigured: %s has %d retro(s) with %d pending kit delta(s); "+
				"not audited — pass --external-root <dir> or add `root <dir>` to %s\n", ResearchRetros, retros, deltas, ConfigFile)
			t.warned = true
			return
		}
	}
}

// researchCandidates are the places the research repository's retros may be: $HOME, a sibling of the kit's
// repository, a sibling of the kit root.
func researchCandidates(kitRoot, home string) []string {
	var dirs []string
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ResearchRetros))
	}
	abs, err := filepath.Abs(kitRoot)
	if err != nil {
		return dirs
	}
	return append(dirs, filepath.Join(filepath.Dir(filepath.Dir(abs)), ResearchRetros), filepath.Join(filepath.Dir(abs), ResearchRetros))
}
