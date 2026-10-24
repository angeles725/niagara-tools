package gen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const rootEnv = "GEN_TEST_DEFAULT_ROOT"

// stdio returns buffers and a Stdio whose rootEnv is defaultRoot.
func stdio(defaultRoot string) (*bytes.Buffer, *bytes.Buffer, cli.Stdio) {
	var o, e bytes.Buffer
	return &o, &e, cli.Stdio{Out: &o, Err: &e, Getenv: func(name string) string {
		if name == rootEnv {
			return defaultRoot
		}
		return ""
	}}
}

func TestParseArgs(t *testing.T) {
	t.Parallel()
	const usage = "usage: gen-x.sh [--check] [<kit-root>]\n"
	const unset = "gen-x: no <kit-root> given and " + rootEnv + " is unset (run toolbelt/gen-x.sh)\n"
	tests := []struct {
		name        string
		args        []string
		defaultRoot string
		check, ok   bool
		root, errs  string
	}{
		{name: "root argument", args: []string{"kit"}, ok: true, root: "kit"},
		{name: "check then root", args: []string{"--check", "kit"}, check: true, ok: true, root: "kit"},
		{name: "repeated check", args: []string{"--check", "--check", "kit"}, check: true, ok: true, root: "kit"},
		{name: "default root", args: nil, defaultRoot: "def", ok: true, root: "def"},
		{name: "empty argument keeps the default", args: []string{""}, defaultRoot: "def", ok: true, root: "def"},
		{name: "unknown flag", args: []string{"-x"}, errs: usage},
		{name: "double dash", args: []string{"--"}, errs: usage},
		{name: "two roots", args: []string{"a", "b"}, errs: usage},
		{name: "no root and no default", args: []string{"--check"}, errs: unset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, e, s := stdio(tt.defaultRoot)
			check, root, ok := ParseArgs("gen-x", rootEnv, tt.args, s)
			if check != tt.check || root != tt.root || ok != tt.ok || o.String() != "" || e.String() != tt.errs {
				t.Errorf("got (%v, %q, %v, %q, %q), want (%v, %q, %v, \"\", %q)",
					check, root, ok, o.String(), e.String(), tt.check, tt.root, tt.ok, tt.errs)
			}
		})
	}
}

func TestSubDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "retros"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, e, s := stdio("")
	if dir, ok := SubDir("gen-x", root, "retros", s); !ok || dir != root+"/retros" || e.String() != "" {
		t.Errorf("existing dir: got (%q, %v, %q)", dir, ok, e.String())
	}
	for _, sub := range []string{"missing", "file"} {
		_, e, s := stdio("")
		if _, ok := SubDir("gen-x", root, sub, s); ok || e.String() != "gen-x: no "+sub+"/ under "+root+"\n" {
			t.Errorf("%s: got (%v, %q)", sub, ok, e.String())
		}
	}
}

func TestIsRegularAndIsDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "nope")
	for _, tt := range []struct {
		path         string
		regular, dir bool
	}{{file, true, false}, {link, true, false}, {dir, false, true}, {missing, false, false}} {
		if IsRegular(tt.path) != tt.regular || IsDir(tt.path) != tt.dir {
			t.Errorf("%s: IsRegular=%v IsDir=%v, want %v %v", tt.path, IsRegular(tt.path), IsDir(tt.path), tt.regular, tt.dir)
		}
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func file(path string) File { return File{Tool: "gen-x", Path: path, Content: "a\nb\n"} }

func TestFileWrite(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "OUT.md")
	write(t, path, "old\n")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	o, e, s := stdio("")
	if code := file(path).Write(s); code != ExitOK || o.String() != "gen-x: wrote "+path+"\n" || e.String() != "" {
		t.Errorf("got (%d, %q, %q)", code, o.String(), e.String())
	}
	st, err := os.Stat(path)
	if b, _ := os.ReadFile(path); err != nil || string(b) != "a\nb\n" || st.Mode().Perm() != 0o600 {
		t.Errorf("target is %q with mode %v, want the content and the old mode 0600", b, st.Mode().Perm())
	}
	if matches, _ := filepath.Glob(path + ".tmp.*"); len(matches) != 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}
}

func TestFileWriteFailure(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "no-such-dir", "OUT.md")
	o, e, s := stdio("")
	if code := file(path).Write(s); code != ExitEnv || o.String() != "" || !strings.HasPrefix(e.String(), "gen-x: cannot write "+path+": ") {
		t.Errorf("got (%d, %q, %q)", code, o.String(), e.String())
	}
}

// TestFileWriteReadOnlyTarget pins how a read-only target is written (R4-atomic-mode-umask, #344): the
// write replaces it through a temporary file in the writable directory and keeps the read-only mode.
func TestFileWriteReadOnlyTarget(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root writes into a mode-444 file")
	}
	path := filepath.Join(t.TempDir(), "OUT.md")
	write(t, path, "old\n")
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	o, e, s := stdio("")
	code := file(path).Write(s)
	b, _ := os.ReadFile(path)
	st, err := os.Stat(path)
	if err != nil || code != ExitOK || string(b) != "a\nb\n" || st.Mode().Perm() != 0o444 || e.String() != "" {
		t.Errorf("got (%d, %q, %q), target %q", code, o.String(), e.String(), b)
	}
}

func TestShowDiff(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "OUT.md")
	write(t, path, "a\nc\n")
	var e strings.Builder
	ShowDiff("a\nb\n", path, &e)
	if !strings.Contains(e.String(), "< b\n---\n> c\n") {
		t.Errorf("ShowDiff printed no generated-vs-file diff: %q", e.String())
	}
}
