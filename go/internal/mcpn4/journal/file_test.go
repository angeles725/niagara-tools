package journal

// Go twins of test_safety.TestJournal (append side), TestExistingStateFiles, TestStateDirCheck
// and TestAuditLog.test_append_and_modes (go_server.go_twin("journal", "unit")).

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

func stateDir(t *testing.T) string { return filepath.Join(t.TempDir(), "state", "mcp-n4") }

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func entry(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Loads([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

func appendAll(t *testing.T, f File, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if err := f.Append(entry(t, text)); err != nil {
			t.Fatal(err)
		}
	}
}

func fileText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAppendWritesOneJsonLinePerEntry(t *testing.T) {
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": "b1", "inverse": [{"nm": "v", "h": "3", "n": "x"}]}`, `{"batch_id":"b2","inverse":[],"f":1.0,"u":"é\ud800"}`)
	want := `{"batch_id": "b1", "inverse": [{"nm": "v", "h": "3", "n": "x"}]}` + "\n" +
		`{"batch_id": "b2", "inverse": [], "f": 1.0, "u": "\u00e9\ud800"}` + "\n"
	if got := fileText(t, j.Path); got != want {
		t.Fatalf("got %q", got)
	}
}

func TestDirIs0700AndFileIs0600(t *testing.T) {
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": "b1"}`)
	if mode(t, j.StateDir) != 0o700 || mode(t, j.Path) != 0o600 {
		t.Fatalf("modes %o %o", mode(t, j.StateDir), mode(t, j.Path))
	}
	if mode(t, filepath.Dir(j.StateDir)) != 0o777&^umask() {
		t.Fatalf("a missing parent gets 0777 less the umask, got %o", mode(t, filepath.Dir(j.StateDir)))
	}
}

func umask() os.FileMode {
	old := syscall.Umask(0)
	syscall.Umask(old)
	return os.FileMode(old)
}

func TestExistingDirIsNeverChmoddedByAppend(t *testing.T) {
	dir := stateDir(t)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	os.Chmod(dir, 0o750)
	j := New(dir)
	appendAll(t, j.File, `{"batch_id": "b1"}`)
	if mode(t, dir) != 0o750 || mode(t, j.Path) != 0o600 {
		t.Fatalf("modes %o %o", mode(t, dir), mode(t, j.Path))
	}
}

func TestCreatedDirIs0700WhateverTheUmask(t *testing.T) {
	old := syscall.Umask(0)
	defer syscall.Umask(old)
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": "b1"}`)
	if mode(t, j.StateDir) != 0o700 || mode(t, j.Path) != 0o600 {
		t.Fatalf("modes %o %o", mode(t, j.StateDir), mode(t, j.Path))
	}
	parent := t.TempDir()
	syscall.Umask(0o277) // mkdir alone would leave 0500: the dir it created is chmodded
	j = New(filepath.Join(parent, "s"))
	appendAll(t, j.File, `{"batch_id": "b1"}`)
	if mode(t, j.StateDir) != 0o700 {
		t.Fatalf("dir mode %o", mode(t, j.StateDir))
	}
}

// seed creates the state dir 0700 and an empty journal.jsonl with perm.
func seed(t *testing.T, perm os.FileMode) (string, string) {
	t.Helper()
	dir := stateDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "journal.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	os.Chmod(path, perm)
	return dir, path
}

func TestAppendRefusesALooseExistingFileAndLeavesItAlone(t *testing.T) {
	dir, path := seed(t, 0o644)
	err := New(dir).Append(entry(t, `{"batch_id": "b1"}`))
	pe, ok := err.(*PermissionError)
	want := path + " is readable or writable by group/others (mode 644): run `chmod 600 " + path +
		"` yourself (the server never changes the permissions of an existing file)"
	if !ok || pe.Msg != want || pe.PyType() != "PermissionError" {
		t.Fatalf("got %#v", err)
	}
	if mode(t, path) != 0o644 || fileText(t, path) != "" {
		t.Fatal("the loose file was changed")
	}
	if err := New(dir).CheckPrivate(); err == nil || err.(*safety.Error).Msg != want {
		t.Fatalf("CheckPrivate: %v", err)
	}
}

func TestAppendStillAcceptsAPrivateExistingFile(t *testing.T) {
	dir, path := seed(t, 0o600)
	os.WriteFile(path, []byte("{\"a\": 1}\n"), 0o600)
	appendAll(t, New(dir).File, `{"batch_id": "b1"}`)
	if got := fileText(t, path); got != "{\"a\": 1}\n{\"batch_id\": \"b1\"}\n" {
		t.Fatalf("got %q", got)
	}
	if err := New(dir).CheckPrivate(); err != nil {
		t.Fatal(err)
	}
}

func TestAppendFollowsNoDanglingLinkAsANewFile(t *testing.T) {
	dir, path := seed(t, 0o600)
	os.Remove(path)
	target := filepath.Join(t.TempDir(), "elsewhere")
	os.Symlink(target, path)
	// O_EXCL refuses the link, the second open follows it and finds nothing to append to.
	if err := New(dir).Append(entry(t, `{"batch_id": "b1"}`)); err == nil {
		t.Fatal("appended through a dangling link")
	}
	if _, err := os.Lstat(target); err == nil {
		t.Fatal("created the link target")
	}
}

func TestCheckStateDirNamesALooseStateFileAtStartup(t *testing.T) {
	dir, path := seed(t, 0o664)
	err := safety.CheckStateDir(dir)
	if err == nil || !strings.Contains(err.Error(), path+" is readable or writable by group/others (mode 664)") {
		t.Fatalf("got %v", err)
	}
}

func TestCheckStateDirAcceptsPrivateStateFiles(t *testing.T) {
	dir, _ := seed(t, 0o600)
	if err := safety.CheckStateDir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestMissingDirIsFine(t *testing.T) {
	if err := safety.CheckStateDir(stateDir(t)); err != nil {
		t.Fatal(err)
	}
}

func TestOwnedPrivateDirIsFine(t *testing.T) {
	dir := stateDir(t)
	os.MkdirAll(dir, 0o700)
	if err := safety.CheckStateDir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestGroupOrWorldWritableDirIsRefusedAndNotChmodded(t *testing.T) {
	dir := stateDir(t)
	os.MkdirAll(dir, 0o700)
	for _, m := range []os.FileMode{0o775, 0o777, 0o722} {
		os.Chmod(dir, m)
		err := safety.CheckStateDir(dir)
		if err == nil || !strings.Contains(err.Error(), "--state-dir") || mode(t, dir) != m {
			t.Fatalf("%o: %v, mode now %o", m, err, mode(t, dir))
		}
	}
}

func TestDirOwnedByAnotherUserIsRefused(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: every directory is the current user's")
	}
	// Python fakes os.getuid; Go cannot, so it uses a directory root owns.
	err := safety.CheckStateDir("/usr")
	if err == nil || !strings.Contains(err.Error(), "is owned by another user") {
		t.Fatalf("got %v", err)
	}
}

func TestAFileInPlaceOfTheDirIsRefused(t *testing.T) {
	dir := stateDir(t)
	os.MkdirAll(filepath.Dir(dir), 0o700)
	os.WriteFile(dir, nil, 0o600)
	if err := safety.CheckStateDir(dir); err == nil || !strings.Contains(err.Error(), "exists and is not a directory") {
		t.Fatalf("got %v", err)
	}
	if err := New(dir).Append(entry(t, `{"batch_id": "b1"}`)); err == nil {
		t.Fatal("appended under a file")
	}
}

func TestConstructionHasNoFilesystemSideEffect(t *testing.T) {
	dir := stateDir(t)
	New(dir)
	NewAuditLog(dir)
	if _, err := os.Lstat(dir); err == nil {
		t.Fatal("constructing created the state dir")
	}
}

func TestAppendAndModes(t *testing.T) {
	a := NewAuditLog(stateDir(t))
	appendAll(t, a.File, `{"tool": "n4_set_slot", "outcome": "planned"}`)
	if fileText(t, filepath.Join(a.StateDir, "audit.jsonl")) != "{\"tool\": \"n4_set_slot\", \"outcome\": \"planned\"}\n" {
		t.Fatal("audit line")
	}
	if mode(t, a.StateDir) != 0o700 || mode(t, a.Path) != 0o600 {
		t.Fatalf("modes %o %o", mode(t, a.StateDir), mode(t, a.Path))
	}
}
