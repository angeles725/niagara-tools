// Command run-pure-test runs one pure-logic JUnit test of a module part outside gradle. Run it through
// build-n4-module-kit/toolbelt/run-pure-test.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/puretest"
)

func main() { cli.Run(puretest.Main) }
