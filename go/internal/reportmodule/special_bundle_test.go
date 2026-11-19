package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// sized writes a file of n bytes under root (its parent directories made).
func sized(t *testing.T, root, rel string, n int) {
	t.Helper()
	p := filepath.Join(root, rel)
	must(t, os.MkdirAll(filepath.Dir(p), 0o755))
	must(t, os.WriteFile(p, bytes.Repeat([]byte("a"), n), 0o644))
}

// TestBundleGzip (retro kit-gzip-bundle-writer-briefs Δ1): a -ux part that ships a dist/*.js over 300 KB with no .gz twin WARNs
// (the servlet then sends the bundle uncompressed through a slow tunnel); a twin, a small bundle or a .js.map
// alone is PASS; no src/rc/dist/ prints no row.
// Mutation: RM-bundle-gzip-twin -- ignoring the twin turns the "with a .gz twin" case into a WARN
// Mutation: RM-bundle-gzip-limit -- a limit of 300 000 bytes (not 300 KB) WARNs the 300 KB edge case
func TestBundleGzip(t *testing.T) {
	t.Parallel()
	const dist = "A-ux/src/rc/dist/"
	tests := []struct {
		name  string
		files map[string]int
		want  string
	}{
		{"over 300 KB with no twin WARNs", map[string]int{"app.js": 400 * 1024},
			"A-ux  WARN  bundle-gzip  dist/app.js  400 KB over 300 KB with no app.js.gz twin: pre-compress it in the bundle step and serve it with Content-Encoding: gzip (types/dashboard.md § Static serving)\n"},
		{"over 300 KB with a twin is PASS", map[string]int{"app.js": 400 * 1024, "app.js.gz": 120 * 1024},
			"A-ux  PASS  bundle-gzip  no dist/*.js over 300 KB without a .gz twin\n"},
		{"200 KB with no twin is PASS", map[string]int{"app.js": 200 * 1024},
			"A-ux  PASS  bundle-gzip  no dist/*.js over 300 KB without a .gz twin\n"},
		{"exactly 300 KB is PASS", map[string]int{"app.js": 300 * 1024},
			"A-ux  PASS  bundle-gzip  no dist/*.js over 300 KB without a .gz twin\n"},
		{"a big source map is not a bundle", map[string]int{"app.js": 10, "app.js.map": 2048 * 1024},
			"A-ux  PASS  bundle-gzip  no dist/*.js over 300 KB without a .gz twin\n"},
		{"a nested chunk counts", map[string]int{"chunks/three.js": 512 * 1024, "app.js": 10},
			"A-ux  WARN  bundle-gzip  dist/chunks/three.js  512 KB over 300 KB with no three.js.gz twin: pre-compress it in the bundle step and serve it with Content-Encoding: gzip (types/dashboard.md § Static serving)\n"},
	}
	for _, tt := range tests {
		root := t.TempDir()
		for rel, n := range tt.files {
			sized(t, root, dist+rel, n)
		}
		var b bytes.Buffer
		r := &report{out: &b, n: map[string]int{}}
		if r.bundleGzip(artifact{"A-ux", root + "/A-ux"}); b.String() != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, b.String(), tt.want)
		}
		if r.hadFail || r.hadEnv {
			t.Errorf("%s: a bundle row must never fail the report (fail %v, env %v)", tt.name, r.hadFail, r.hadEnv)
		}
	}
}

// TestBundleGzipNoDist: a -ux part with no src/rc/dist/ (a module with no bundle step) prints no row, so the
// reports of every existing module stay as they were.
func TestBundleGzipNoDist(t *testing.T) {
	t.Parallel()
	root := tree(t, "A-ux/src/rc/index.html")
	var b bytes.Buffer
	r := &report{out: &b, n: map[string]int{}}
	if r.bundleGzip(artifact{"A-ux", root + "/A-ux"}); b.String() != "" {
		t.Errorf("no dist/: %q, want no row", b.String())
	}
}

// TestBundleGzipUnreadable: a dist/ nobody may read is an ERROR row (fail-closed), never a PASS.
func TestBundleGzipUnreadable(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads every directory")
	}
	root := t.TempDir()
	sized(t, root, "A-ux/src/rc/dist/app.js", 10)
	d := filepath.Join(root, "A-ux/src/rc/dist")
	must(t, os.Chmod(d, 0o300))
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	var b bytes.Buffer
	r := &report{out: &b, n: map[string]int{}}
	r.bundleGzip(artifact{"A-ux", root + "/A-ux"})
	if !r.hadEnv || !bytes.HasPrefix(b.Bytes(), []byte("A-ux  ERROR  bundle-gzip  cannot read src/rc/dist/: ")) {
		t.Errorf("unreadable dist/: %q (env %v), want an ERROR row", b.String(), r.hadEnv)
	}
}
