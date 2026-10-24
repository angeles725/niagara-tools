package preflight

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// recordTools are the system tools testdata/setup.sh links into each case's .tools (its TOOLS list).
var recordTools = []string{"bash", "sh", "env", "grep", "sed", "awk", "cut", "find", "paste", "sort", "uniq", "wc", "tr", "comm", "basename", "dirname", "head", "tail", "xargs", "df", "sleep", "cat", "printf", "mkdir", "ls", "readlink", "timeout", "rm", "seq"}

// TestGolden replays every testdata case (recorded from the bash preflight.sh by golden/record/record.sh, or
// a truth-* case hand-edited from one) through golden.Runner: the case's whole environment, its tree, and the
// exact output and tree the run leaves. Fake commands are scripts in the tree; the case PATH finds the system
// tools through <case>/.tools.
func TestGolden(t *testing.T) {
	golden.Runner{Cases: 125, Setup: linkTools, Main: func(env, args []string, s cli.Stdio) int {
		return Command(Host{Output: System.Output, Env: env})(args, s)
	}}.Run(t)
}

// TestRecordToolsMatchSetup pins recordTools to the TOOLS list of testdata/setup.sh: the replay and the recording
// link the same tools.
func TestRecordToolsMatchSetup(t *testing.T) {
	b, err := os.ReadFile("testdata/setup.sh")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^TOOLS=\(([^)]*)\)$`).FindSubmatch(b)
	if m == nil {
		t.Fatal("testdata/setup.sh has no TOOLS=(...) line")
	}
	if got := strings.Fields(string(m[1])); !slices.Equal(got, recordTools) {
		t.Errorf("setup.sh TOOLS %q, recordTools %q", got, recordTools)
	}
}

// linkTools gives the case the <dir>/.tools testdata/setup.sh gives it.
func linkTools(t *testing.T, dir string, _ map[string]string) []string {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, ".tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tool := range recordTools {
		for _, d := range []string{"/usr/bin/", "/bin/"} {
			if _, err := os.Stat(d + tool); err == nil {
				if err := os.Symlink(d+tool, filepath.Join(dir, ".tools", tool)); err != nil {
					t.Fatal(err)
				}
				break
			}
		}
	}
	return []string{".tools/"}
}
