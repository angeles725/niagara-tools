package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

// fakeT records how a replay ends: Fatalf and Skipf stop the goroutine as testing does.
type fakeT struct {
	testing.TB
	msgs     []string
	failed   bool
	skipped  bool
	cleanups []func()
}

func (f *fakeT) Helper()           {}
func (f *fakeT) Failed() bool      { return f.failed }
func (f *fakeT) Skipped() bool     { return f.skipped }
func (f *fakeT) Cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }
func (f *fakeT) Errorf(format string, a ...any) {
	f.failed = true
	f.msgs = append(f.msgs, fmt.Sprintf(format, a...))
}
func (f *fakeT) Fatalf(format string, a ...any) { f.Errorf(format, a...); runtime.Goexit() }
func (f *fakeT) Skipf(format string, a ...any) {
	f.skipped = true
	f.msgs = append(f.msgs, fmt.Sprintf(format, a...))
	runtime.Goexit()
}
func (f *fakeT) Fatal(a ...any) { f.Fatalf("%s", fmt.Sprint(a...)) }

// replay runs calls against a golden holding recorded in the scratch package dir and returns
// the outcome: "pass", "fail: ..." or "skip: ...".
func replay(t *testing.T, recorded string, calls func(g *Golden)) string {
	t.Setenv(pytarget.RecordEnv, "")
	dir := t.TempDir()
	t.Chdir(dir)
	if recorded != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("testdata", "x.golden.jsonl"), []byte(recorded), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f := &fakeT{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		calls(Open(f, "x"))
	}()
	<-done
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
	switch {
	case f.skipped:
		return "skip: " + strings.Join(f.msgs, "; ")
	case f.failed:
		return "fail: " + strings.Join(f.msgs, "; ")
	}
	return "pass"
}

func TestReplay(t *testing.T) {
	// The digest of script "s" (first 8 bytes of its SHA-256).
	const rec = `{"script":"043a718774c572bd","stdin":"in1","stdout":"out1"}` + "\n" +
		`{"script":"043a718774c572bd","stdin":"in2","stdout":"out2"}` + "\n"
	both := func(g *Golden) {
		if got := string(g.Run("s", []byte("in1"))) + string(g.Run("s", []byte("in2"))); got != "out1out2" {
			g.t.Errorf("replayed %q", got)
		}
	}
	bigIn := []byte(strings.Repeat("a", maxInline+1))
	big := `{"script":"043a718774c572bd","stdin_sha256":"` + sha256Hex(bigIn) + `","stdout":"o"}` + "\n"
	for _, tc := range []struct {
		name, recorded, want string
		calls                func(g *Golden)
	}{
		{"recorded calls replay in order", rec, "pass", both},
		{"missing golden skips with the re-record hint", "", "skip: golden testdata/x.golden.jsonl missing: re-record with MCP_N4_RECORD=1", both},
		{"changed cases are stale", rec, "fail: golden testdata/x.golden.jsonl is stale: call 1 sends other cases (first difference at byte 2)",
			func(g *Golden) { g.Run("s", []byte("inX")) }},
		{"changed script is stale", rec, "fail: golden testdata/x.golden.jsonl is stale: call 1 runs a changed script",
			func(g *Golden) { g.Run("t", []byte("in1")) }},
		{"an extra call is stale", rec, "fail: golden testdata/x.golden.jsonl is stale: call 3 beyond the 2 recorded",
			func(g *Golden) { both(g); g.Run("s", []byte("in3")) }},
		{"a left-over record is stale", rec, "fail: golden testdata/x.golden.jsonl is stale: 2 recorded calls, the test made 1",
			func(g *Golden) { g.Run("s", []byte("in1")) }},
		{"large cases replay by digest", big, "pass", func(g *Golden) { g.Run("s", bigIn) }},
		{"changed large cases are stale", big, "fail: golden testdata/x.golden.jsonl is stale: call 1 sends other cases (SHA-256 differs)",
			func(g *Golden) { g.Run("s", append([]byte("x"), bigIn[1:]...)) }},
		{"an alias replays as the run's real text", `{"script":"043a718774c572bd","stdin":"in $R","stdout":"out $R"}` + "\n", "pass",
			func(g *Golden) {
				g.Alias("/tmp/run2", "$R")
				if got := string(g.Run("s", []byte("in /tmp/run2"))); got != "out /tmp/run2" {
					g.t.Errorf("replayed %q", got)
				}
			}},
		{"a corrupt golden fails", "{", "fail: golden testdata/x.golden.jsonl: record 1: unexpected EOF", both},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := replay(t, tc.recorded, tc.calls); !strings.HasPrefix(got, tc.want) {
				t.Errorf("got  %s\nwant %s...", got, tc.want)
			}
		})
	}
}
