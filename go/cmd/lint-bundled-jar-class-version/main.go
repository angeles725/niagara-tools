// Command lint-bundled-jar-class-version flags bundled ext-jars under a module source tree that carry
// Java 9+ bytecode (class-file major version > 52). Run it through
// toolbelt/lint-bundled-jar-class-version.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/jarversion"
)

func main() { cli.Run(jarversion.Main) }
