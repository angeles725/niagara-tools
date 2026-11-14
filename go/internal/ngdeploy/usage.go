package ngdeploy

import (
	_ "embed" // the usage text
	"os"
	"strings"
)

// usage is what --help prints and what follows an unknown flag: the text the bash tool carried.
//
//go:embed usage.txt
var usage string

// versionFile is the VERSION file, from the scripts directory.
const versionFile = "/../VERSION"

// versionText is what --version prints: the VERSION file beside the scripts directory, without its final
// newlines, or "unknown" when it cannot be read.
func (d *deploy) versionText() string {
	text, err := os.ReadFile(d.scriptDir + versionFile)
	if err != nil {
		return versionUnknown
	}
	return strings.TrimRight(string(text), "\n")
}
