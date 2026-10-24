package kitticket

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
var recordTools = []string{"bash", "sh", "awk", "sed", "grep", "head", "cat", "mkdir", "mv", "rm", "printf", "dirname"}

// testPid is the pid the truth-* cases name the temp files by (the bash used its $$).
const testPid = 4242

// TestGolden replays every testdata case (recorded from the bash kit-ticket.sh by golden/record/record.sh, or a
// truth-* case that only renames its temp files to testPid) through golden.Runner. The recorded script ran as
// <case>/toolbelt/kit-ticket.sh, so the default root is the case dir and the script path its toolbelt copy; a
// gh-present case has its own fake gh in the tree, which finds cat through <case>/.tools.
func TestGolden(t *testing.T) {
	golden.Runner{
		Cases: 55,
		Setup: func(t *testing.T, dir string, env map[string]string) []string {
			env[DefaultRootEnv] = dir
			env[ScriptEnv] = dir + "/toolbelt/kit-ticket.sh"
			return linkTools(t, dir)
		},
		Main: func(env, args []string, s cli.Stdio) int {
			return Command(Host{Today: func() string { return "2026-01-02" }, Pid: testPid, Env: env})(args, s)
		},
	}.Run(t)
}

// TestRecordToolsMatchSetup pins recordTools to the TOOLS list of testdata/setup.sh.
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

// linkTools gives the case the <dir>/.tools testdata/setup.sh gives it (the fake date included, though the Go
// port takes its date from Host.Today).
func linkTools(t *testing.T, dir string) []string {
	t.Helper()
	tools := filepath.Join(dir, ".tools")
	if err := os.Mkdir(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tool := range recordTools {
		for _, d := range []string{"/usr/bin/", "/bin/"} {
			if _, err := os.Stat(d + tool); err == nil {
				if err := os.Symlink(d+tool, filepath.Join(tools, tool)); err != nil {
					t.Fatal(err)
				}
				break
			}
		}
	}
	if err := os.WriteFile(filepath.Join(tools, "date"), []byte("#!/bin/sh\nprintf \"2026-01-02\\n\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return []string{".tools/"}
}
