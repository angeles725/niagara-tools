package commissioningverify

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSourceDirs(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	for _, p := range []string{"r/B/B-rt/src", "r/A/A-rt/src", "r/A/x/src", "r/A/A-rt/src/d/Z-rt/src", "r/src", "real/Y/Y-rt/src"} {
		must(t, os.MkdirAll(filepath.Join(d, p), 0o755))
	}
	must(t, os.Symlink(filepath.Join(d, "real"), filepath.Join(d, "r/L")))
	must(t, os.Symlink(filepath.Join(d, "r"), filepath.Join(d, "link")))
	r := d + "/r/"
	want := []srcDir{{r + "A/A-rt/src", "A-rt"}, {r + "B/B-rt/src", "B-rt"}}
	if got := sourceDirs(r); !reflect.DeepEqual(got, want) {
		t.Errorf("sourceDirs(%q) = %q, want %q (sorted, the symlink r/L not followed)", r, got, want)
	}
	if got := sourceDirs(d + "/link"); len(got) != 2 || got[0].path != d+"/link/A/A-rt/src" {
		t.Errorf("a symlinked root is followed like find -H: got %q", got)
	}
	must(t, os.MkdirAll(filepath.Join(d, "plain/q/q-rt"), 0o755))
	if got := sourceDirs(d + "/plain/q/.."); len(got) != 0 {
		t.Errorf("no -rt/src and no src/: got %q, want none", got)
	}
	must(t, os.MkdirAll(filepath.Join(d, "plain/src"), 0o755))
	if got, want := sourceDirs(d+"/plain/q/.."), []srcDir{{d + "/plain/q/../src", ".."}}; !reflect.DeepEqual(got, want) {
		t.Errorf("fallback src/ keeps the root spelling: got %q, want %q", got, want)
	}
}

func TestRun(t *testing.T) {
	t.Setenv("NIAGARA_TOOLS_CV_LEAK", " leaked") // a host variable outside env must not reach the script
	d := t.TempDir()
	script := func(name, body string, mode os.FileMode) string {
		p := filepath.Join(d, name)
		must(t, os.WriteFile(p, []byte("#!/usr/bin/env bash\n"+body+"\n"), mode))
		return p
	}
	for _, c := range []struct {
		path, out string
		code      int
	}{
		{script("ok", `echo "a $LC_ALL $*${NIAGARA_TOOLS_CV_LEAK-}"; echo b >&2; printf 'c\0\n\n\n'`, 0o755), "a C x y\nb\nc", 0},
		{script("two", "exit 2", 0o755), "", 2},
		{script("noexec", "exit 0", 0o644), "", 126},
		{filepath.Join(d, "missing"), "", 127},
		{script("killed", "kill -TERM $$", 0o755), "", 143},
	} {
		if out, code := run(c.path, minimalEnv(), "x", "y"); out != c.out || code != c.code {
			t.Errorf("run(%s) = %q, %d; want %q, %d", filepath.Base(c.path), out, code, c.out, c.code)
		}
	}
}

// minimalEnv is the only environment the tests give the lints: PATH (bash, go) and HOME (the go build cache).
func minimalEnv() []string { return []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")} }

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
