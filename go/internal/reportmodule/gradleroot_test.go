package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// TestAGradleRootIsRefusedWithTheDirectoryToPass (retro build-kit-bench-session Δ10): report-module takes the module
// directory (<gradle root>/<MOD>, the one holding <MOD>-rt), build.sh the gradle root. Given the gradle root it exits 3
// naming the directory to pass and writes no row; the header says which directory each tool means.
func TestAGradleRootIsRefusedWithTheDirectoryToPass(t *testing.T) {
	root := moduleTree(t, map[string]string{
		"settings.gradle.kts":           "",
		"Foo/Foo-rt/Foo-rt.gradle.kts":  "",
		"Foo/Foo-rt/module-include.xml": "<module/>",
		"gradle/wrapper/x":              "",
	})
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: os.Getenv}
	code := mainEnv([]string{"--lint-only", root}, s, os.Environ())
	want := "report-module: " + root + " is a gradle root, not a module root: pass " + filepath.Join(root, "Foo") + "\n"
	if code != ExitEnv || e.String() != want || o.Len() != 0 {
		t.Errorf("exit %d stdout %q stderr %q, want exit %d, no stdout, stderr %q", code, o.String(), e.String(), ExitEnv, want)
	}
	script, err := os.ReadFile("../../../build-n4-module-kit/toolbelt/report-module.sh")
	if err != nil || !strings.Contains(string(script), "<module-root>: the MODULE directory") {
		t.Error("the report-module.sh header does not say that <module-root> is the module directory")
	}
}
