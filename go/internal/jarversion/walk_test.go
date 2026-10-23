package jarversion

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// lock makes dir unreadable for the rest of the test; it skips when that does not block reads (root).
func lock(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if f, err := os.Open(dir); err == nil {
		f.Close()
		t.Skip("running as root: mode 000 does not block reads")
	}
}

func TestListJars(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	m := filepath.Join(d, "Mod")
	for _, p := range []string{"libs/b.jar", "libs/a.jar", "libs/B.jar", "x.jar/inner.jar", ".hidden.jar", "deep/er/c.jar",
		"build/libs/out.jar", "sub/build/x.jar", ".git/x.jar", "sub/.cache/y.jar", "libs/readme.txt", "libs/a.jar.bak"} {
		touch(t, filepath.Join(m, p))
	}
	if err := os.Symlink(filepath.Join(m, "libs", "a.jar"), filepath.Join(m, "link.jar")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(m, "deep"), filepath.Join(m, "linkdir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(m, filepath.Join(d, "modlink")); err != nil {
		t.Fatal(err)
	}
	want := []string{m + "/.hidden.jar", m + "/deep/er/c.jar", m + "/libs/B.jar", m + "/libs/a.jar", m + "/libs/b.jar", m + "/x.jar/inner.jar"}

	cases := []struct {
		name, root string
		want       []string
	}{
		{"regular *.jar files, build/ and dot dirs pruned, symlinks skipped, C byte order", m, want},
		{"a trailing slash is not doubled", m + "/", want},
		{"a doubled slash is kept as given", m + "//", []string{m + "//.hidden.jar", m + "//deep/er/c.jar", m + "//libs/B.jar", m + "//libs/a.jar", m + "//libs/b.jar", m + "//x.jar/inner.jar"}},
		{"a root named build or .dot is walked, not pruned", filepath.Join(m, "build"), []string{m + "/build/libs/out.jar"}},
		{"a symlinked root is not followed (find -P)", filepath.Join(d, "modlink"), nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, werr := ListJars(tt.root)
			if werr != "" || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListJars(%s) = %q, %q; want %q", tt.root, got, werr, tt.want)
			}
		})
	}
}

func TestListJarsUnreadable(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	m := filepath.Join(d, "Mod")
	touch(t, filepath.Join(m, "libs", "ok.jar"))
	touch(t, filepath.Join(m, "libs", "locked", "a.jar"))
	lock(t, filepath.Join(m, "libs", "locked"))
	// Mutation: BJCV-finderr -- returning no walk error for an unreadable directory reports the module clean.
	got, werr := ListJars(m)
	if want := "find: '" + m + "/libs/locked': Permission denied"; werr != want {
		t.Errorf("walk error = %q, want %q", werr, want)
	}
	if want := []string{m + "/libs/ok.jar"}; !reflect.DeepEqual(got, want) {
		t.Errorf("listed = %q, want %q (the readable part is still scanned)", got, want)
	}
	if _, werr := ListJars(filepath.Join(d, "nope")); werr != "find: '"+d+"/nope': No such file or directory" {
		t.Errorf("missing root error = %q", werr)
	}
}

func TestFindQuote(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"plain/path":    "'plain/path'",
		"it's":          `'it\'s'`,
		`back\slash`:    `'back\\slash'`,
		"tab\there":     `'tab\there'`,
		"nl\nx":         `'nl\nx'`,
		"bell\a":        `'bell\a'`,
		"del\x7f":       `'del\177'`,
		"utf8-\xc3\xa9": `'utf8-\303\251'`,
		`dq"x`:          `'dq"x'`,
	}
	for in, want := range cases {
		if got := findQuote(in); got != want {
			t.Errorf("findQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestStrerror(t *testing.T) {
	t.Parallel()
	cases := map[syscall.Errno]string{
		syscall.EACCES: "Permission denied",
		syscall.ENOENT: "No such file or directory",
		syscall.ELOOP:  "Too many levels of symbolic links",
		syscall.EIO:    "Input/output error",
	}
	for errno, want := range cases {
		if got := strerror.Text(errno); got != want {
			t.Errorf("strerror(%d) = %q, want %q", errno, got, want)
		}
	}
}
