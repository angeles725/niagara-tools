package mirrorhome

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fixture is a temporary directory holding a niagara_home at <dir>/nh.
type fixture struct {
	t        *testing.T
	dir      string // canonical
	out, err string
}

var madeAt = time.Date(2026, 3, 4, 5, 6, 7, 0, time.Local)

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, dir: dir}
	f.mkdir("nh/etc/m2/repository", "nh/bin", "nh/modules", "nh/.hidden")
	f.touch("nh/modules/baja.jar", "nh/modules/Demo-rt.jar", "nh/modules/control-rt.jar", "nh/modules/notes.txt", "nh/modules/.dot.jar", "nh/top.txt")
	return f
}

func (f *fixture) path(rel string) string { return f.dir + "/" + rel }

func (f *fixture) mkdir(rel ...string) {
	f.t.Helper()
	for _, r := range rel {
		if err := os.MkdirAll(f.path(r), 0o755); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fixture) touch(rel ...string) {
	f.t.Helper()
	for _, r := range rel {
		f.mkdir(filepath.Dir(r))
		if err := os.WriteFile(f.path(r), nil, 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fixture) symlink(target, rel string) {
	f.t.Helper()
	if err := os.Symlink(target, f.path(rel)); err != nil {
		f.t.Fatal(err)
	}
}

// mirror runs the tool at madeAt and returns its exit code; f.out and f.err hold what it printed.
func (f *fixture) mirror(args ...string) int {
	var out, errb bytes.Buffer
	code := run(args, cli.Stdio{Out: &out, Err: &errb}, func() time.Time { return madeAt })
	f.out, f.err = out.String(), errb.String()
	return code
}

// wants compares the last run with an exit code, stdout and stderr.
func (f *fixture) wants(name string, code, want int, stdout, stderr string) {
	f.t.Helper()
	if code != want || f.out != stdout || f.err != stderr {
		f.t.Errorf("%s: exit %d stdout %q stderr %q, want %d %q %q", name, code, f.out, f.err, want, stdout, stderr)
	}
}

// tree lists everything under rel: one "path kind [-> target]" line per entry, sorted.
func (f *fixture) tree(rel string) string {
	f.t.Helper()
	var lines []string
	root := f.path(rel)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		line := strings.TrimPrefix(p, f.dir+"/") + " " + d.Type().String()
		if d.Type() == fs.ModeSymlink {
			target, _ := os.Readlink(p)
			line += " -> " + target
		}
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// unchanged fails when the tree under rel differs from before.
func (f *fixture) unchanged(name, rel, before string) {
	f.t.Helper()
	if after := f.tree(rel); after != before {
		f.t.Errorf("%s: %s changed:\n%s\nwas:\n%s", name, rel, after, before)
	}
}
