package jarversion

import (
	"fmt"
	"io"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of lint-bundled-jar-class-version (unchanged from the bash tool).
const (
	ExitOK   = 0 // no FAIL row
	ExitFail = 1 // any FAIL row
	ExitEnv  = 3 // usage, a root that is not a directory, or a sub-directory the walk cannot enter
)

const tool = "lint-bundled-jar-class-version"

// Main implements `lint-bundled-jar-class-version <module-root>`. Like the bash tool (awk_lint_parse
// arity raw) the first argument is the root whatever it looks like and the rest are ignored. An
// incomplete walk is named on stderr and still scans the jars it listed before exiting 3.
func Main(args []string, s cli.Stdio) int {
	if len(args) < 1 {
		fmt.Fprintf(s.Err, "usage: %s.sh <module-root>\n", tool)
		return ExitEnv
	}
	root := args[0]
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		fmt.Fprintf(s.Err, "%s: not a directory: %s\n", tool, root)
		return ExitEnv
	}
	jars, walkErr := ListJars(root)
	if walkErr != "" {
		fmt.Fprintf(s.Err, "%s: cannot list every file under %s: %s\n", tool, root, walkErr)
	}
	code := scan(jars, s.Out)
	if walkErr != "" {
		return ExitEnv
	}
	return code
}

// scan prints one FAIL row per jar with a Java 9+ sampled class and returns ExitFail when it printed any.
func scan(jars []string, out io.Writer) int {
	code := ExitOK
	for _, jar := range jars {
		if f, hit := CheckJar(jar); hit {
			fmt.Fprint(out, FailRow(jar, f.Entry, f.Major))
			code = ExitFail
		}
	}
	return code
}
