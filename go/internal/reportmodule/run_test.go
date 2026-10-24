package reportmodule

import (
	"os"
	"path/filepath"
	"testing"
)

func script(t *testing.T, dir, name, body string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(dir, name)
	must(t, os.WriteFile(p, []byte(body), mode))
	return p
}

func TestRunExitCodesAndOutput(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
