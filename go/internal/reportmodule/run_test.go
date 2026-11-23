package reportmodule

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func script(t *testing.T, dir, name, body string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(dir, name)
	must(t, os.WriteFile(p, []byte(body), mode))
	return p
}

func TestRunExitCodesAndOutput(t *testing.T) {
	// Not parallel: it writes an executable and runs it; a parallel test forking meanwhile holds the write fd
	// and the exec fails ETXTBSY (126). Sequential tests run before the parallel ones resume.
	dir := t.TempDir()
	x := runner{env: []string{"PATH=/usr/bin:/bin"}}
	tests := []struct {
		name, body string
		mode       os.FileMode
		out        string
		code       int
	}{
		{"merged and trimmed", "#!/bin/sh\necho out; echo err >&2; printf 'a\\0b\\n\\n'\nexit 1\n", 0o755, "out\nerr\nab", 1},
		{"signal", "#!/bin/sh\nkill -TERM $$\n", 0o755, "", 143},
		{"not executable", "#!/bin/sh\n", 0o644, "", 126},
		{"no shebang", "echo hi\n", 0o755, "", 126},
	}
	for _, tt := range tests {
		p := script(t, dir, tt.name, tt.body, tt.mode)
		if out, code, _ := x.run("", p); out != tt.out || code != tt.code {
			t.Errorf("%s: run = %q, %d; want %q, %d", tt.name, out, code, tt.out, tt.code)
		}
	}
	if _, code, _ := x.run("", filepath.Join(dir, "missing")); code != 127 {
		t.Errorf("a missing member is exit %d, want 127", code)
	}
	if out, _, _ := x.run(dir, script(t, dir, "pwd", "#!/bin/sh\npwd\n", 0o755)); out != dir {
		t.Errorf("run in %s printed %q", dir, out)
	}
}

func TestRunBoundsOutput(t *testing.T) {
	t.Parallel()
	c := &capWriter{max: 4}
	c.Write([]byte("abc"))
	c.Write([]byte("defg"))
	if c.b.String() != "abcd" || c.dropped != 3 {
		t.Errorf("capWriter kept %q, dropped %d", c.b.String(), c.dropped)
	}
}

func TestFirstLine(t *testing.T) {
	// Not parallel: it writes an executable and runs it; a parallel test forking meanwhile holds the write fd
	// and the exec fails ETXTBSY (126). Sequential tests run before the parallel ones resume.
	dir := t.TempDir()
	x := runner{env: []string{"PATH=/usr/bin:/bin"}}
	p := script(t, dir, "v", "#!/bin/sh\necho err >&2\nprintf 'v9.1.0\\nmore\\n'\nwhile :; do echo y || exit 0; done\n", 0o755)
	if got := x.firstLine(p, "--version"); got != "v9.1.0" {
		t.Errorf("firstLine = %q (stderr dropped, the writer stopped after the first line)", got)
	}
	if got := x.firstLine(filepath.Join(dir, "missing")); got != "" {
		t.Errorf("firstLine of a missing binary = %q", got)
	}
}

func TestCommandV(t *testing.T) {
	t.Parallel()
	a, b := t.TempDir(), t.TempDir()
	must(t, os.Mkdir(filepath.Join(a, "tool"), 0o755)) // a directory is never a command
	script(t, a, "plain", "x", 0o644)
	script(t, b, "plain", "x", 0o644)
	exe := script(t, b, "tool", "#!/bin/sh\n", 0o755)
	if got := commandV(a+":"+b, "tool"); got != exe {
		t.Errorf("commandV(tool) = %q, want %q", got, exe)
	}
	if got := commandV(a+":"+b, "plain"); got != filepath.Join(a, "plain") {
		t.Errorf("commandV(plain) = %q, want the first non-executable file (bash's last resort)", got)
	}
	if got := commandV(a, "nothing"); got != "" {
		t.Errorf("commandV(nothing) = %q", got)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// TestToJudge: what makes <rc>/vendor something lint-vendor-floor must judge. A directory and a link to one
// are; a link that leads nowhere and an rc dir nobody may search are too (the lint's exit 3, never a
// missing row). No entry, a regular file, a link to a file, and an rc that is itself a file (ENOTDIR:
// nothing can be under it) are nothing to run the lint on.
func TestToJudge(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root may search every directory")
	}
	t.Parallel()
	d := t.TempDir()
	mk := func(rel string) string {
		p := filepath.Join(d, rel)
		must(t, os.MkdirAll(p, 0o755))
		return p
	}
	must(t, os.WriteFile(filepath.Join(d, "file"), []byte("x\n"), 0o644))
	must(t, os.Symlink(mk("real"), filepath.Join(mk("linked"), "vendor")))
	must(t, os.Symlink(filepath.Join(d, "gone"), filepath.Join(mk("dangling"), "vendor")))
	must(t, os.Symlink(filepath.Join(d, "file"), filepath.Join(mk("tofile"), "vendor")))
	must(t, os.WriteFile(filepath.Join(mk("regular"), "vendor"), []byte("x\n"), 0o644))
	mk("dir/vendor")
	mk("none")
	locked := mk("locked")
	mk("locked/vendor")
	must(t, os.Chmod(locked, 0o600))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	cases := []struct {
		name, vendor string
		want         bool
	}{
		{"a directory", "dir/vendor", true},
		{"a link to a directory", "linked/vendor", true},
		{"a link that leads nowhere", "dangling/vendor", true},
		{"an rc dir nobody may search", "locked/vendor", true},
		{"no entry", "none/vendor", false},
		{"a regular file", "regular/vendor", false},
		{"a link to a file", "tofile/vendor", false},
		{"an rc that is a file (ENOTDIR)", "file/vendor", false},
	}
	for _, c := range cases {
		if got := toJudge(filepath.Join(d, c.vendor)); got != c.want {
			t.Errorf("%s: toJudge = %v, want %v", c.name, got, c.want)
		}
	}
}

// A parallel test that forks while another one still holds a stub open for writing makes exec fail with ETXTBSY
// ("text file busy"): the child holds the write descriptor until its own exec. That transient refusal is retried, so
// a member is never reported as "unexpected exit 126" for it (TestReadyVerifyModule failed so in CI).
func TestRunRetriesATextFileBusyExec(t *testing.T) {
	p := filepath.Join(t.TempDir(), "member")
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho ran\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := os.OpenFile(p, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	go func() { time.Sleep(40 * time.Millisecond); w.Close() }()
	x := runner{env: []string{"PATH=/usr/bin:/bin"}}
	if out, code, _ := x.run("", p); code != 0 || out != "ran" {
		t.Fatalf("exit %d, out %q: a busy executable must be retried until it can run", code, out)
	}
}
