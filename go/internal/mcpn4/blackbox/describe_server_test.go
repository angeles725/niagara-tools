package blackbox

// n4_describe_session's server block: which binary serves, when it was built and started, and a
// warning when the binary on disk was replaced after the server started.

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// copiedServer starts a server from a copy of the test binary, so the test owns its mtime.
func copiedServer(t *testing.T) (*Server, string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "mcp-n4")
	src, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(bin, os.O_CREATE|os.O_WRONLY, 0o755)
	if err == nil {
		_, err = io.Copy(dst, src)
	}
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(bin, past, past); err != nil {
		t.Fatal(err)
	}
	return Start(t, Options{Exe: bin}), bin
}

func TestDescribeReportsTheServerBuildWithoutASession(t *testing.T) {
	s, bin := copiedServer(t)
	out := s.OK("n4_describe_session", nil)
	server, _ := out["server"].(map[string]any)
	// a test binary carries no VCS stamp: the build is unknown and built_at is the binary's mtime
	if server["binary"] != bin || server["build"] != "unknown" || out["warning"] != nil {
		t.Fatalf("server = %v, warning = %v", server, out["warning"])
	}
	built, err1 := time.Parse(time.RFC3339, str(server["built_at"]))
	started, err2 := time.Parse(time.RFC3339, str(server["started_at"]))
	if err1 != nil || err2 != nil || !built.Before(started) || time.Since(started) > time.Minute {
		t.Errorf("built_at %v, started_at %v", server["built_at"], server["started_at"])
	}
}

func TestDescribeWarnsWhenTheBinaryOnDiskIsNewerThanTheServer(t *testing.T) {
	s, bin := copiedServer(t)
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(bin, later, later); err != nil {
		t.Fatal(err)
	}
	out := s.OK("n4_describe_session", nil)
	warning := str(out["warning"])
	if !strings.Contains(warning, bin) || !strings.Contains(warning, "restart") {
		t.Errorf("warning = %q", warning)
	}
	// built_at is the binary that serves, read at start, not the one now on disk
	server, _ := out["server"].(map[string]any)
	if built, err := time.Parse(time.RFC3339, str(server["built_at"])); err != nil || !built.Before(time.Now()) {
		t.Errorf("built_at followed the new binary: %v", server)
	}
}

func str(v any) string { s, _ := v.(string); return s }
