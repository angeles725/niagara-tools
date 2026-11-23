package foldaudit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFile is read from the kit root when it exists: local-only, never configured in CI. One entry per line:
//
//	root <dir>             a directory of retros outside the kit (a leading ~/ is the home directory)
//	alias <stem>=<ALIAS>   the name the retro with that stem is reported and cited by
//
// Blank lines and lines starting with # are ignored. It is absent by default.
const ConfigFile = "fold-roots.conf"

// addConfig appends the roots and aliases of <kit-root>/fold-roots.conf to the options (a --alias flag wins over a
// config alias). It returns the stderr text of a malformed file, "" otherwise.
func (o *options) addConfig(home string) string {
	f, err := os.Open(filepath.Join(o.root, ConfigFile))
	if err != nil {
		return "" // absent by default
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kind, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)
		switch kind {
		case "root":
			if rest != "" {
				if strings.HasPrefix(rest, "~/") {
					rest = filepath.Join(home, rest[2:])
				}
				o.externals = append(o.externals, rest)
				continue
			}
		case "alias":
			if stem, alias, ok := strings.Cut(rest, "="); ok && stem != "" && aliasName.MatchString(alias) {
				o.setAlias(stem, alias)
				continue
			}
		}
		return fmt.Sprintf("fold-audit: %s line %d: expected `root <dir>` or `alias <stem>=<ALIAS>`\n", ConfigFile, n)
	}
	return ""
}
