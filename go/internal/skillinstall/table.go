package skillinstall

import (
	"os"
	"regexp"
	"strings"
)

// skill is one entry of the SKILLS table: a name and its tracked launcher, relative to the repository.
type skill struct{ name, path string }

// tableLine is the installer's `SKILLS="<name>:<path> <name>:<path>"` line: the one place that names
// the skills, read from the script by install-skill and by check-skill-drift.
var tableLine = regexp.MustCompile(`(?m)^SKILLS="(.*)"$`)

// readTable returns the entries of the first SKILLS line of the installer script; none is an empty
// table. The error is a script that cannot be read.
func readTable(installer string) ([]skill, error) {
	data, err := os.ReadFile(installer)
	if err != nil {
		return nil, err
	}
	m := tableLine.FindSubmatch(data)
	if m == nil {
		return nil, nil
	}
	var table []skill
	for _, entry := range strings.Fields(string(m[1])) {
		name, path, found := strings.Cut(entry, ":")
		if !found {
			path = entry // the shell's ${entry%%:*} and ${entry#*:} both give the whole entry
		}
		table = append(table, skill{name, path})
	}
	return table, nil
}

// lookup is the tracked path of a skill; the last entry of that name wins, as in the shell loop.
func lookup(table []skill, name string) (path string, ok bool) {
	for _, entry := range table {
		if entry.name == name {
			path, ok = entry.path, true
		}
	}
	return path, ok
}

// names joins the skill names in table order.
func names(table []skill, sep string) string {
	all := make([]string, len(table))
	for i, entry := range table {
		all[i] = entry.name
	}
	return strings.Join(all, sep)
}
