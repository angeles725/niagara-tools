package main

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

// wantsVersion is true when a server command line asks for --version (argparse's version action:
// it wins wherever it appears among the options).
func wantsVersion(args []string) bool {
	for _, a := range args {
		if a == "--version" {
			return true
		}
	}
	return false
}

// printVersion prints "mcp-n4 <version> (git <build>)": the serverInfo version and the build stamp
// n4_describe_session reports as server_version and server.build.
func printVersion(s cli.Stdio) int {
	fmt.Fprintf(s.Out, "mcp-n4 %s (git %s)\n", registry.Default().Version, readtools.Build())
	return 0
}
