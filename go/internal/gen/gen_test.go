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

// index is a File in the index style (retros/INDEX.md, toolbelt/INDEX.md).
func index(path string) File {
	return File{Tool: "gen-x", Path: path, Content: "a\nb\n", TrimNewlines: true}
}

// ledger is a File in the ledger style (BUILD-STATE.md).
func ledger(path string) File {
	return File{Tool: "gen-x", Path: path, Content: "a\nb\n", StaleNote: " vs ledger.d/", Atomic: true}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFileCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		file      func(string) File
		committed string // "" leaves the target missing
		code      int
		out, errs string // errs is a prefix: the diff follows a stale line
	}{
		{"index fresh", index, "a\nb\n", ExitOK, "gen-x: %s is fresh\n", ""},
		{"index ignores trailing newlines", index, "a\nb\n\n\n", ExitOK, "gen-x: %s is fresh\n", ""},
		{"index ignores a missing final newline", index, "a\nb", ExitOK, "gen-x: %s is fresh\n", ""},
		{"index stale", index, "a\nc\n", ExitStale, "", "gen-x: %s is stale; run gen-x.sh and commit the result\n"},
		{"index missing", index, "", ExitStale, "", "gen-x: %s is missing; run gen-x.sh\n"},
		{"ledger fresh", ledger, "a\nb\n", ExitOK, "gen-x: %s is fresh\n", ""},
		{"ledger counts trailing newlines", ledger, "a\nb\n\n", ExitStale, "", "gen-x: %s is stale vs ledger.d/; run gen-x.sh and commit the result\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "OUT.md")
			if tt.committed != "" {
				write(t, path, tt.committed)
			}
			o, e, s := stdio("")
			code := tt.file(path).Run(true, s)
			wantOut, wantErr := strings.ReplaceAll(tt.out, "%s", path), strings.ReplaceAll(tt.errs, "%s", path)
			if code != tt.code || o.String() != wantOut || !strings.HasPrefix(e.String(), wantErr) || (wantErr == "") != (e.String() == "") {
				t.Errorf("got (%d, %q, %q), want (%d, %q, prefix %q)", code, o.String(), e.String(), tt.code, wantOut, wantErr)
			}
			if b, _ := os.ReadFile(path); string(b) != tt.committed {
				t.Errorf("--check changed the target to %q", b)
			}
		})
	}
}

func TestFileCheckShowsTheDiff(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "OUT.md")
	write(t, path, "a\nc\n")
	_, e, s := stdio("")
	index(path).Run(true, s)
	if !strings.Contains(e.String(), "< b\n---\n> c\n") {
		t.Errorf("stale --check printed no generated-vs-committed diff: %q", e.String())
	}
}

func TestFileCheckNonRegularIndexIsMissing(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "OUT.md")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	_, e, s := stdio("")
	if code := index(path).Run(true, s); code != ExitStale || e.String() != "gen-x: "+path+" is missing; run gen-x.sh\n" {
		t.Errorf("got (%d, %q)", code, e.String())
	}
}

func TestFileWrite(t *testing.T) {
	t.Parallel()
	for _, file := range []func(string) File{index, ledger} {
		path := filepath.Join(t.TempDir(), "OUT.md")
		write(t, path, "old\n")
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
		o, e, s := stdio("")
		f := file(path)
		if code := f.Run(false, s); code != ExitOK || o.String() != "gen-x: wrote "+path+"\n" || e.String() != "" {
			t.Errorf("atomic=%v: got (%d, %q, %q)", f.Atomic, code, o.String(), e.String())
		}
		st, err := os.Stat(path)
		if b, _ := os.ReadFile(path); err != nil || string(b) != "a\nb\n" || st.Mode().Perm() != 0o600 {
			t.Errorf("atomic=%v: target is %q with mode %v, want the content and the old mode 0600", f.Atomic, b, st.Mode().Perm())
		}
		if matches, _ := filepath.Glob(path + ".tmp.*"); len(matches) != 0 {
			t.Errorf("atomic=%v: temporary files left behind: %v", f.Atomic, matches)
		}
	}
}

func TestFileWriteFailure(t *testing.T) {
	t.Parallel()
	for _, file := range []func(string) File{index, ledger} {
		path := filepath.Join(t.TempDir(), "no-such-dir", "OUT.md")
		o, e, s := stdio("")
		f := file(path)
		if code := f.Run(false, s); code != ExitEnv || o.String() != "" || !strings.HasPrefix(e.String(), "gen-x: cannot write "+path+": ") {
			t.Errorf("atomic=%v: got (%d, %q, %q)", f.Atomic, code, o.String(), e.String())
		}
	}
}
