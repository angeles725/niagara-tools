package lint

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// A RootRule takes a regular file as its root only when its Spec opts in with FileRoot and FileRoot accepts
// that file; every other spec refuses a file root as before: exit 3, "<name>: not a directory: <root>".
func TestFileRootIsOptIn(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "ModT.jar")
	other := filepath.Join(dir, "notes.txt")
	for _, p := range []string{jar, other} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	acceptJar := func(p string) bool { return strings.HasSuffix(p, ".jar") }
	for name, tc := range map[string]struct {
		fileRoot func(string) bool
		root     string
		code     int
		stderr   string
	}{
		"no FileRoot, a file":             {nil, jar, 3, "lint-r: not a directory: " + jar + "\n"},
		"FileRoot, a file it refuses":     {acceptJar, other, 3, "lint-r: not a directory: " + other + "\n"},
		"FileRoot, a file it accepts":     {acceptJar, jar, 1, ""},
		"FileRoot, a directory as before": {acceptJar, dir, 1, ""},
	} {
		var seen []string
		sp := rootSpec(rootFake{seen: &seen})
		sp.FileRoot = tc.fileRoot
		var out, errb bytes.Buffer
		code := Command(sp)([]string{"lint-r", tc.root}, cli.Stdio{Out: &out, Err: &errb})
		ran := len(seen) == 1 && seen[0] == tc.root
		if code != tc.code || errb.String() != tc.stderr || ran != (tc.code != 3) {
			t.Errorf("%s: exit %d, stderr %q, roots seen %v", name, code, errb.String(), seen)
		}
	}
}
