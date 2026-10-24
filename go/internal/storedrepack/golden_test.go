package storedrepack

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
	"github.com/angeles725/niagara-tools/go/internal/storedrepack/jarlist"
)

// goldenCases is the exact number of testdata cases: a lost or an extra case fails the run.
const goldenCases = 22

// TestGolden replays every testdata case (recorded from the bash stored-repack.sh by golden/record/record.sh
// with the testdata/setup.sh hooks, or a truth-* case) through golden.Runner. buildJars and listOut are the
// Go side of those hooks: the input jars come from their .spec files, and the output jar is compared through
// its jarlist listing (names, order, method, size, content hash), never by its bytes.
func TestGolden(t *testing.T) {
	golden.Runner{Cases: goldenCases, Setup: buildJars, Main: func(_, args []string, s cli.Stdio) int {
		code := Main(args, s)
		listOut(t, args)
		return code
	}}.Run(t)
}

// buildJars is setup_case of testdata/setup.sh: every <path>.spec of the case tree becomes the jar <path>.
// It returns the entries the tree comparison skips: the built jars and the output jars the cases name.
func buildJars(t *testing.T, dir string, _ map[string]string) []string {
	t.Helper()
	skip := []string{"out.jar", "sub/out.jar"}
	var specs []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() && strings.HasSuffix(p, ".spec") {
			specs = append(specs, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(specs)
	for _, p := range specs {
		spec, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		jar, err := jarlist.Build(spec)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if err := os.WriteFile(strings.TrimSuffix(p, ".spec"), jar, 0o644); err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(dir, strings.TrimSuffix(p, ".spec"))
		skip = append(skip, filepath.ToSlash(rel))
	}
	return skip
}

// listOut is the tail of setup.sh's rec-run.sh: when the second argument names a zip after the run, its
// listing goes to <out>.listing.
func listOut(t *testing.T, args []string) {
	t.Helper()
	if len(args) < 2 {
		return
	}
	if st, err := os.Stat(args[1]); err != nil || !st.Mode().IsRegular() {
		return
	}
	if l, err := jarlist.List(args[1]); err == nil {
		if err := os.WriteFile(args[1]+".listing", []byte(l), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
