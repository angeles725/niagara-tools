package servlet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// tree makes the files (path -> text; a path ending in "/" is a directory) under a fresh temp directory.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, text := range files {
		full := filepath.Join(root, p)
		if strings.HasSuffix(p, "/") {
			must(t, os.MkdirAll(full, 0o755))
			continue
		}
		must(t, os.MkdirAll(filepath.Dir(full), 0o755))
		must(t, os.WriteFile(full, []byte(text), 0o644))
	}
	return root
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// TestRCDirAbove (issue #1035 review R3): the climb finds the rc/ of the servlet's own module and no other. It
// ends at a gradle project root (Kotlin or Groovy script) and at a module part (module-include.xml, a part
// build script, a -rt/-ux directory), after looking at that directory's own rc/ and src/rc/, and at maxClimb
// levels. want is the rc directory relative to the temp root, "" for none; start is where the servlet is.
func TestRCDirAbove(t *testing.T) {
	t.Parallel()
	deep := func(n int) string { return strings.Repeat("d/", n) }
	for _, c := range []struct {
		name, start, want string
		files             map[string]string
	}{
		{"src/rc beside com", "a/src/com/x", "a/src/rc", map[string]string{"a/src/rc/": ""}},
		{"rc at the source root", "a/com", "a/rc", map[string]string{"a/rc/": ""}},
		{"no marker: a parent's rc is the nearest", "p/mod/src/com", "p/rc", map[string]string{"p/rc/": ""}},
		{"build.gradle.kts stops the climb", "p/mod/src/com", "", map[string]string{"p/rc/": "", "p/mod/build.gradle.kts": ""}},
		{"build.gradle stops the climb", "p/mod/src/com", "", map[string]string{"p/rc/": "", "p/mod/build.gradle": ""}},
		{"settings.gradle.kts stops the climb", "p/mod/src/com", "", map[string]string{"p/rc/": "", "p/mod/settings.gradle.kts": ""}},
		{"settings.gradle stops the climb", "p/mod/src/com", "", map[string]string{"p/rc/": "", "p/mod/settings.gradle": ""}},
		{"the root's own rc/ is its module's", "m/src/com", "m/rc", map[string]string{"m/rc/": "", "m/build.gradle.kts": ""}},
		{"the root's own src/rc is its module's", "m/src/com", "m/src/rc", map[string]string{"m/src/rc/": "", "m/build.gradle": ""}},
		{"module-include.xml ends a part", "p/X-rt/src/com", "", map[string]string{"p/src/rc/": "", "p/X-rt/module-include.xml": ""}},
		{"a part build script ends a part", "p/X-rt/src/com", "", map[string]string{"p/rc/": "", "p/X-rt/X-rt.gradle.kts": ""}},
		{"a Groovy part script ends a part", "p/X-rt/src/com", "", map[string]string{"p/rc/": "", "p/X-rt/X-rt.gradle": ""}},
		{"a -rt directory ends a part", "p/X-rt/src/com", "", map[string]string{"p/rc/": "", "p/X-rt/src/": ""}},
		{"a -ux directory ends a part", "p/X-ux/src/com", "", map[string]string{"p/rc/": "", "p/X-ux/src/": ""}},
		{"a part keeps its own src/rc", "p/X-ux/src/com", "p/X-ux/src/rc", map[string]string{"p/rc/": "", "p/X-ux/module-include.xml": "", "p/X-ux/src/rc/": ""}},
		{"maxClimb levels up is found", deep(maxClimb), "rc", map[string]string{"rc/": ""}},
		{"one level past maxClimb is not", deep(maxClimb + 1), "", map[string]string{"rc/": ""}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root := tree(t, c.files)
			must(t, os.MkdirAll(filepath.Join(root, c.start), 0o755))
			want := ""
			if c.want != "" {
				want = filepath.Join(root, c.want)
			}
			if got := rcDirAbove(filepath.Join(root, c.start)); got != want {
				t.Errorf("rcDirAbove(%s) = %q, want %q", c.start, got, want)
			}
		})
	}
}

// TestRCReadError (R3-rcrefs-error-swallowed): an rc/ file the lint cannot read is an error of the servlet
// file, which the engine prints as `cannot scan <file>: lint-servlet: ...` and exit 3 (the way an unreadable
// source file ends), and the file's rows are still reported, without the evidence it could not collect.
func TestRCReadError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := tree(t, map[string]string{"src/rc/index.html": "<img src=\"img/a.png\">\n", "src/com/B.java": ""})
	must(t, os.Chmod(filepath.Join(root, "src/rc/index.html"), 0))
	src := "class B extends BWebServlet {\n  void doGet(WebOp op) {\n    r.setHeader(\"Cache-Control\", \"max-age=60\");\n  }\n}\n"
	fds, err := Spec().Rule.Check(javasrc.NewFile(filepath.Join(root, "src/com/B.java"), strings.Split(strings.TrimSuffix(src, "\n"), "\n")))
	if err == nil || !strings.Contains(err.Error(), "rc/ pages could not be read: ") || !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("error = %v, want rc/ pages could not be read: ... Permission denied", err)
	}
	if len(fds) != 1 || fds[0].Check != "cache-nofinger" || fds[0].Msg != msgCache {
		t.Errorf("rows = %+v, want the one old cache-nofinger row", fds)
	}
}
