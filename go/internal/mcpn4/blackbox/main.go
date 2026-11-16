// Package blackbox drives the real mcp-n4 server as a separate process over stdio, against a
// boxstation fake: the kit's black-box harness (it replaced the Python one, retired in T9).
// The process is the test binary itself: a package that uses the harness calls
// Main from its TestMain, and the harness re-executes the test binary with a marker that makes
// it run the server's main (mcpcli.Main) instead of the tests. No prebuilt go/bin is needed.
//
// Typical use:
//
//	func TestMain(m *testing.M) { blackbox.Main(m) }
//
//	func TestX(t *testing.T) {
//		r := blackbox.NewRig(t, nil)          // a fake station plus a read-only server
//		out := r.Srv.OK("n4_connect", blackbox.Args{"station": blackbox.StationName})
//		...
//	}
package blackbox

import (
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
)

// marker, set to "1" in a child's environment, makes the test binary act as mcp-n4.
const marker = "MCPN4_BLACKBOX_AS_SERVER"

// buildEnv, when set in a child's environment, is the build the server reports as its own: a
// test binary carries no VCS stamp, and the stale-binary tests need one.
const buildEnv = "MCPN4_BLACKBOX_BUILD"

// Main runs the tests, or, in a child the harness started, the mcp-n4 server; it never returns.
func Main(m *testing.M) {
	if os.Getenv(marker) == "1" {
		if build := os.Getenv(buildEnv); build != "" {
			readtools.StampBuild(build)
		}
		os.Exit(mcpcli.Main(os.Args[1:], cli.Stdio{Out: os.Stdout, Err: os.Stderr, Getenv: os.Getenv}))
	}
	os.Exit(m.Run())
}
