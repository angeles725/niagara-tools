// Command gen-changelog validates the changelog.d/ fragments (--check) or folds them into CHANGELOG.md's
// ## [Unreleased] section at release time (--write). Run it through scripts/gen-changelog.sh, which builds
// this binary on demand and passes the default repository root.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/changelog"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(changelog.FoldMain) }
