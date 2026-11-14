package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func names(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func withUmask(t *testing.T, m int) {
	t.Helper()
	old := syscall.Umask(m)
	t.Cleanup(func() { syscall.Umask(old) })
}

func TestWriteModes(t *testing.T) {
	withUmask(t, 0o022)
	for _, tc := range []struct {
		name     string
		existing os.FileMode // 0: no existing file
		want     os.FileMode
	}{
		{"a new file is 0644 under the umask", 0, 0o644},
		{"an existing target keeps its exact mode", 0o664, 0o664},
		{"an existing private target stays private", 0o600, 0o600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "f.md")
			if tc.existing != 0 {
				if err := os.WriteFile(p, []byte("old"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(p, tc.existing); err != nil {
					t.Fatal(err)
				}
			}
			if err := Write(p, []byte("new\n")); err != nil {
				t.Fatal(err)
			}
			st, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			if got := st.Mode().Perm(); got != tc.want {
				t.Errorf("mode = %o, want %o", got, tc.want)
			}
			if b, _ := os.ReadFile(p); string(b) != "new\n" {
				t.Errorf("content = %q", b)
			}
			if n := names(t, dir); len(n) != 1 {
				t.Errorf("leftover files: %v", n)
			}
		})
	}
}

func TestNewFileFollowsAStricterUmask(t *testing.T) {
	withUmask(t, 0o077)
	p := filepath.Join(t.TempDir(), "f")
	if err := Write(p, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", st.Mode().Perm())
	}
}

func TestFailuresLeaveNoTempAndTheOldContent(t *testing.T) {
	boom := errors.New("boom")
	for _, tc := range []struct {
		name  string
		setup func()
	}{
		{"write fails", func() { writeData = func(*os.File, []byte) error { return boom } }},
		{"rename fails", func() { rename = func(string, string) error { return boom } }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { writeData, rename = writeAll, os.Rename })
			tc.setup()
			dir := t.TempDir()
			p := filepath.Join(dir, "INDEX.md")
			if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := Write(p, []byte("new")); !errors.Is(err, boom) {
				t.Fatalf("err = %v, want boom", err)
			}
			if b, _ := os.ReadFile(p); string(b) != "old" {
				t.Errorf("content = %q, want old", b)
			}
			if n := names(t, dir); len(n) != 1 {
				t.Errorf("leftover files: %v", n)
			}
		})
	}
}

// A temp name that already exists is retried with a fresh name; a name source that keeps
// colliding is refused after maxAttempts, and nothing is left behind.
func TestTempNameCollision(t *testing.T) {
	t.Cleanup(func() { tempSuffix = randomSuffix })
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	taken := filepath.Join(dir, ".f-taken.tmp")
	if err := os.WriteFile(taken, []byte("someone else's"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	tempSuffix = func() string {
		calls++
		if calls == 1 {
			return "taken"
		}
		return "fresh"
	}
	if err := Write(p, []byte("x")); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if b, _ := os.ReadFile(taken); string(b) != "someone else's" {
		t.Errorf("a colliding temp file was overwritten: %q", b)
	}
	tempSuffix = func() string { return "taken" }
	err := Write(p, []byte("y"))
	if err == nil || !strings.Contains(err.Error(), "temporary file") {
		t.Fatalf("err = %v, want a refusal naming the temporary file", err)
	}
	if b, _ := os.ReadFile(p); string(b) != "x" {
		t.Errorf("content = %q, want x", b)
	}
}

func TestMissingDirectoryIsAnError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "no", "such", "f")
	if err := Write(p, []byte("x")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want ErrNotExist", err)
	}
}
