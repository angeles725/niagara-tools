package ntplmanifest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// checker gathers the rows of every archive of one root. root and each path are already spelled for a row
// (rowpath.Display): a name with a control character never splits a row.
type checker struct {
	root    string
	mod     module
	home    home
	stock   map[string]bool // the modules the installation ships; nil when no installation was given
	out     []lint.Finding
	unknown bool // an ntpl-deps row was printed without the installation's module list
	jar     bool // the archives are the templates of a template module jar, which declares no build dependencies
}

// rows is every row, the note about the missing module list first when a row depends on it.
func (c *checker) rows() []lint.Finding {
	if !c.unknown {
		return c.out
	}
	return append([]lint.Finding{{Severity: lint.Warn, Path: c.root, Msg: noHome}}, c.out...)
}

// check adds the rows of one archive: its entries, then its manifest, then what the manifest declares.
func (c *checker) check(path string, station bool, a archive) {
	row := func(check, format string, args ...any) {
		c.out = append(c.out, lint.Finding{Check: check, Path: path, Msg: fmt.Sprintf(format, args...)})
	}
	for _, msg := range a.entries {
		row(checkEntries, "%s", msg)
	}
	if a.unread != "" {
		row(checkManifest, "%s", a.unread)
	}
	if a.manifest == nil {
		return
	}
	m, why := parseManifest(a.manifestName, a.manifest)
	if why != "" {
		// Mutation: NTPL2 -- taking a manifest that does not parse for one with no dependencies passes it
		row(checkManifest, "%s", why)
		return
	}
	if m.root != rootElement {
		// Mutation: NTPL-root -- reading any document for a manifest passes a file that declares nothing a reader takes
		row(checkManifest, wrongRoot, a.manifestName, m.root)
		return
	}
	for _, line := range m.nameless {
		row(checkManifest, namelessDep, a.manifestName, line)
	}
	for _, line := range m.stray {
		row(checkManifest, strayDep, a.manifestName, line)
	}
	if station && !m.station {
		// Mutation: NTPL4 -- dropping this row ships a newStations/ template whose manifest is not a station manifest
		row(checkStation, notAStation, m.root)
	}
	c.deps(path, m.deps)
	c.uses(path, a.manifestName, m.deps, a.uses)
}

// deps adds the ntpl-deps row of one manifest: the dependencies the module does not declare and, with an
// installation, does not find there either. Without one a stock module cannot be told from a missing one,
// so the row is WARN and never blocks. A module whose build files were not all read is not compared (the
// scan fails closed on the build file; a row here would compare against a partial list).
func (c *checker) deps(path string, deps []string) {
	missing := c.missing(deps)
	switch {
	case !c.mod.known || len(missing) == 0:
	case c.stock == nil:
		// Mutation: NTPL-nohome -- failing here blocks a module on stock modules the lint cannot know about
		c.unknown = true
		msg := fmt.Sprintf(undeclaredAny, c.root, strings.Join(missing, ", "))
		if c.jar {
			msg = fmt.Sprintf(jarUnshippedAny, strings.Join(missing, ", "))
		}
		c.out = append(c.out, lint.Finding{Check: checkDeps, Severity: lint.Warn, Path: path, Msg: msg})
	default:
		// Mutation: NTPL3 -- dropping this row passes a template that needs a module nobody declares or ships
		msg := fmt.Sprintf(undeclared, c.root, c.home.dir, strings.Join(missing, ", "))
		if c.jar {
			msg = fmt.Sprintf(jarUnshipped, c.home.dir, strings.Join(missing, ", "))
		}
		c.out = append(c.out, lint.Finding{Check: checkDeps, Path: path, Msg: msg})
	}
}

// missing are the names of deps that the module does not declare and the installation does not ship,
// sorted, each once.
func (c *checker) missing(deps []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range deps {
		// Mutation: NTPL-stock -- comparing against the build files alone fails every station template on the stock modules it lists
		if !c.mod.names[d] && !c.stock[d] && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	sort.Strings(out)
	return out
}

// underStationDir reports a template with a newStations directory between the root and itself.
func underStationDir(root, path string) bool {
	rel := strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
	parts := strings.Split(rel, "/")
	for _, dir := range parts[:len(parts)-1] {
		if dir == stationDir {
			return true
		}
	}
	return false
}
