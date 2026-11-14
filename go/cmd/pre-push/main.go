// Command pre-push is the kit's git pre-push gate. Run it through .githooks/pre-push, which builds this
// binary on demand and passes on the hook arguments and stdin.
package main

import (
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/prepush"
)

func main() {
	os.Exit(prepush.Run(os.Args[1:], os.Stdin, cli.Stdio{Out: os.Stdout, Err: os.Stderr, Getenv: os.Getenv}))
}
