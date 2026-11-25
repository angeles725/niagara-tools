package skillinstall

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// runCheck is `install-skill --check`: the rows of check-skill-drift for every skill of the SKILLS table in
// every harness, table order then harness order. It never writes: 0 when every checked copy is current,
// 1 when any row is a FAIL, 3 when a tracked launcher is missing or a digest cannot be computed.
// Every tracked launcher is looked up before the first row, so an environment error prints no rows.
func runCheck(o installOptions, table []skill, launcher installtools.Launcher, s cli.Stdio) int {
	tracked := make([]string, len(table))
	for i, entry := range table {
		tracked[i] = launcher.RepoRoot() + "/" + entry.path
		if isFile, _ := installtools.IsFile(tracked[i]); !isFile {
			fmt.Fprintf(s.Err, installTool+": tracked source not found: %s\n", tracked[i])
			return exitEnv
		}
	}
	code := exitOK
	for i, entry := range table {
		d := driftOptions{skill: entry.name, home: o.home, harness: allHarnesses, explicit: true, homeGiven: o.homeGiven}
		for _, h := range harnesses {
			switch c := checkHarness(driftCheck{o: d, h: h, rel: entry.path, tracked: tracked[i]}, s); c {
			case exitOK:
			case exitEnv:
				return exitEnv
			default:
				code = c
			}
		}
	}
	return code
}
