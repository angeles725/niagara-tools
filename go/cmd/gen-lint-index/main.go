// Command gen-lint-index generates build-n4-module-kit/toolbelt/INDEX.md from each
// toolbelt/lint-*.sh header. Run it through toolbelt/gen-lint-index.sh, which builds
// this binary on demand and passes the default kit root.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lintindex"
)

func main() { cli.Run(lintindex.Main) }
