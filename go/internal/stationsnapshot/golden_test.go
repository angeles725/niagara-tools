package stationsnapshot

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// recordTools are the system tools testdata/setup.sh links into each case's .tools (its TOOLS list).
var recordTools = []string{"bash", "sh", "dirname", "basename", "mkdir", "mktemp", "rm", "cp", "chmod", "sha256sum",
	"cut", "wc", "tr", "find", "sort", "awk"}

// testStamp is the UTC time the fake `date` of testdata/setup.sh prints (20260102T030405Z).
var testStamp = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// TestGolden replays every testdata case (recorded from the bash station-snapshot.sh by golden/record/record.sh,
// or a truth-* case: a declared difference from the bash) through golden.Runner. The recorded script ran as
// <case>/toolbelt/station-snapshot.sh, the prefix of the shell's own error messages; a case's PATH is
// <case>/.tools, or <case>/.tools-nosha for the sha256sum guard.
func TestGolden(t *testing.T) {
	golden.Runner{
		Cases: 58,
		Setup: func(t *testing.T, dir string, env map[string]string) []string {
			env[ScriptEnv] = dir + "/toolbelt/station-snapshot.sh"
			return linkTools(t, dir)
		},
		Main: func(_, args []string, s cli.Stdio) int {
			return Command(Host{Now: func() time.Time { return testStamp.In(time.FixedZone("X", 3600)) }})(args, s)
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

// linkTools gives the case the <dir>/.tools and <dir>/.tools-nosha testdata/setup.sh gives it (the fake date
// included, though the Go port takes its time from Host.Now).
func linkTools(t *testing.T, dir string) []string {
	t.Helper()
	for _, set := range []string{".tools", ".tools-nosha"} {
		tools := filepath.Join(dir, set)
		if err := os.Mkdir(tools, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, tool := range recordTools {
			if set == ".tools-nosha" && tool == "sha256sum" {
				continue
			}
			for _, d := range []string{"/usr/bin/", "/bin/"} {
				if _, err := os.Stat(d + tool); err == nil {
					if err := os.Symlink(d+tool, filepath.Join(tools, tool)); err != nil {
						t.Fatal(err)
					}
					break
				}
			}
		}
		if err := os.WriteFile(filepath.Join(tools, "date"), []byte("#!/bin/sh\nprintf \"20260102T030405Z\\n\"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return []string{".tools/", ".tools-nosha/"}
}
