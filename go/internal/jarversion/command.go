package jarversion

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of lint-bundled-jar-class-version (unchanged from the bash tool).
const (
	ExitOK   = 0 // no FAIL row
	ExitFail = 1 // any FAIL row
	ExitEnv  = 3 // usage, a root that is not a directory, a sub-directory the walk cannot enter, or a jar that cannot be read
)

const tool = "lint-bundled-jar-class-version"

// Main implements `lint-bundled-jar-class-version <module-root>`. Like the bash tool (awk_lint_parse
// arity raw) the first argument is the root whatever it looks like and the rest are ignored. An
// incomplete walk or an unreadable jar is named on stderr; the other jars are still scanned before exit 3.
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
	code, scanErr := scan(jars, s)
	if walkErr != "" || scanErr {
		return ExitEnv
	}
	return code
}

// scan prints one FAIL row per jar with a Java 9+ sampled class and returns ExitFail when it printed any.
// A jar that cannot be checked in full is named on stderr ("cannot scan", as lib/awk-lint.sh words it)
// and sets scanErr; the scan goes on.
func scan(jars []string, s cli.Stdio) (code int, scanErr bool) {
	code = ExitOK
	for _, jar := range jars {
		f, hit, err := CheckJar(jar)
		if err != nil {
			// Mutation: BJCV-jarerr -- dropping this error reports an unreadable or corrupt jar clean.
			fmt.Fprintf(s.Err, "%s: cannot scan %s: %s\n", tool, jar, Reason(err))
			scanErr = true
		}
		if hit {
			fmt.Fprint(s.Out, FailRow(jar, f.Entry, f.Major))
			code = ExitFail
		}
	}
	return code, scanErr
}
