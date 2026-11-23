package niagaraversion

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of check-target-version.
const (
	ExitOK    = 0 // the installation is the version the module was compiled against
	ExitFail  = 1 // a FAIL row: it is another one
	ExitUsage = 2 // bad arguments, or a jar or installation whose version cannot be read: nothing is compared
)

const usage = "usage: check-target-version <module.jar> <niagara_home | its modules directory>\n"

// Main implements `check-target-version <module.jar> <installation>`: one PASS or FAIL row that says both
// versions. A version that cannot be read on either side is exit 2 and no row, never a pass.
func Main(args []string, s cli.Stdio) int {
	if len(args) != 2 || len(args[0]) > 0 && args[0][0] == '-' || len(args[1]) > 0 && args[1][0] == '-' {
		fmt.Fprint(s.Err, usage)
		return ExitUsage
	}
	jar, target := args[0], args[1]
	compiled, err := CompiledFor(jar)
	if err != nil {
		fmt.Fprintf(s.Err, "check-target-version: %v\n", err)
		return ExitUsage
	}
	installed, root, err := InstallVersion(target)
	if err != nil {
		fmt.Fprintf(s.Err, "check-target-version: %v\n", err)
		return ExitUsage
	}
	same, err := Match(compiled, installed)
	if err != nil {
		fmt.Fprintf(s.Err, "check-target-version: %v\n", err)
		return ExitUsage
	}
	if !same {
		fmt.Fprintf(s.Out, "FAIL  target-version  %s was compiled against Niagara %s but %s is Niagara %s: "+
			"build it against an installation of %s (mirror-niagara-home.sh), or install it into one of %s\n",
			jar, compiled, root, installed, MajorMinor(installed), compiled)
		return ExitFail
	}
	fmt.Fprintf(s.Out, "PASS  target-version  %s was compiled against Niagara %s; %s is Niagara %s\n", jar, compiled, root, installed)
	return ExitOK
}
