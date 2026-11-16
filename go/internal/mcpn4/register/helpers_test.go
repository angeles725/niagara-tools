package register

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	secretUser = "opuser-7f3a"
	secretPass = "pw-9c1e-not-in-launcher"
)

// fakeHome is a HOME (a t.TempDir) with the station's env file holding fake credentials.
type fakeHome struct {
	dir      string
	out, err strings.Builder
}

func newHome(t *testing.T) *fakeHome {
	t.Helper()
	h := &fakeHome{dir: t.TempDir()}
	t.Setenv("HOME", h.dir) // belt and braces: Main reads HOME through Stdio, never the process
	h.write(t, ".config/mcp-n4/lab.env", "LAB_N4_USER="+secretUser+"\nLAB_N4_PASSWORD="+secretPass+"\n")
	return h
}

func (h *fakeHome) path(rel string) string { return filepath.Join(h.dir, filepath.FromSlash(rel)) }

func (h *fakeHome) write(t *testing.T, rel, text string) {
	t.Helper()
	p := h.path(rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (h *fakeHome) read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(h.path(rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
