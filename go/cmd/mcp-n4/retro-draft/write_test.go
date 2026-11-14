package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// The write mode's rows, ported from mcp-n4-kit/tests/test_new_retro.py (new_retro.py).

const (
	b1       = "00000000000000000000000000000001"
	retroMD  = "2026-10-01-LLM-session.md"
	friction = `{"batch_id": "` + b1 + `", "ts": "2026-10-01T11:00:00+00:00", "tool": "n4_set_slot", "ops": [{"nm": "s"}], "phase": "intent", "station_name": "LLM", "inverse_plan": [], "targets": {}}
{"batch_id": "` + b1 + `", "ts": "2026-10-01T11:00:01+00:00", "phase": "result", "accepted": null, "inverse": [], "verdict": "mismatch"}
`
)

type kit struct {
	t           *testing.T
	state, out  string
	index, file string
}

func newKit(t *testing.T) *kit {
	t.Helper()
	root := t.TempDir()
	k := &kit{t: t, state: filepath.Join(root, "state"), out: filepath.Join(root, "retros")}
	k.index, k.file = filepath.Join(k.out, "INDEX.md"), filepath.Join(k.out, retroMD)
	for _, d := range []string{k.state, k.out} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	index, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "retros", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	k.writeFile(k.index, string(index))
	return k
}

func (k *kit) writeFile(p, s string) {
	k.t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		k.t.Fatal(err)
	}
}

func (k *kit) friction() { k.writeFile(filepath.Join(k.state, "journal.jsonl"), friction) }

func (k *kit) read(p string) string {
	k.t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		k.t.Fatal(err)
	}
	return string(b)
}

func (k *kit) ls() []string {
	es, _ := os.ReadDir(k.out)
	var n []string
	for _, e := range es {
		n = append(n, e.Name())
	}
	sort.Strings(n)
	return n
}

// run is write mode for station LLM on 2026-10-01 with extra flags; it returns exit, out, err.
func (k *kit) run(extra ...string) (int, string, string) {
	args := append([]string{"--write", "--station", "LLM", "--state-dir", k.state, "--out-dir", k.out,
		"--date", "2026-10-01"}, extra...)
	return runArgs(args...)
}

func runArgs(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := run(args, cli.Stdio{Out: &out, Err: &errOut})
	return code, out.String(), errOut.String()
}

func TestWritesTheRetroAndAPendingRowWithItsDeltaCount(t *testing.T) {
	k := newKit(t)
	k.friction()
	code, out, _ := k.run()
	if code != 0 || out != "retro  "+k.file+"  written (1 delta(s), pending)\n" {
		t.Fatalf("exit %d, out %q", code, out)
	}
	if text := k.read(k.file); !strings.Contains(text, "<!-- review-status: pending -->") || !strings.Contains(text, b1) {
		t.Errorf("retro lacks the pending marker or the batch: %q", text)
	}
	if !strings.Contains(k.read(k.index), "| "+retroMD+" | LLM | 2026-10-01 | pending | 1 |") {
		t.Errorf("INDEX row missing")
	}
}

func TestKeepsTheIndexModeAndGivesANewRetro0644UnderTheUmask(t *testing.T) {
	old := syscall.Umask(0o022)
	defer syscall.Umask(old)
	k := newKit(t)
	if err := os.Chmod(k.index, 0o664); err != nil {
		t.Fatal(err)
	}
	if code, _, e := k.run(); code != 0 {
		t.Fatalf("exit %d: %s", code, e)
	}
	for p, want := range map[string]os.FileMode{k.index: 0o664, k.file: 0o644} {
		if st, _ := os.Stat(p); st.Mode().Perm() != want {
			t.Errorf("%s mode %o, want %o", filepath.Base(p), st.Mode().Perm(), want)
		}
	}
}

func TestNoFrictionWritesTheHonestyLineAndZeroDeltas(t *testing.T) {
	k := newKit(t)
	if code, _, _ := k.run(); code != 0 {
		t.Fatal(code)
	}
	if !strings.Contains(k.read(k.file), "no new deltas;") ||
		!strings.Contains(k.read(k.index), "| "+retroMD+" | LLM | 2026-10-01 | pending | 0 |") {
		t.Errorf("no honesty line or zero-delta row")
	}
}

func TestRefusesToOverwriteWithoutForceAndLeavesEverythingUntouched(t *testing.T) {
	k := newKit(t)
	k.run()
	before := k.read(k.index) + k.read(k.file)
	k.friction()
	code, out, errOut := k.run()
	if code != 3 || out != "" || errOut != "retro-draft: "+k.file+" already exists: pass --force to rewrite it\n" {
		t.Fatalf("exit %d, out %q, err %q", code, out, errOut)
	}
	if k.read(k.index)+k.read(k.file) != before {
		t.Errorf("files changed")
	}
}

func TestForceRewritesTheFileAndUpdatesTheRowWithoutDuplicatingIt(t *testing.T) {
	k := newKit(t)
	k.run()
	k.friction()
	if code, _, _ := k.run("--force"); code != 0 {
		t.Fatal(code)
	}
	index := k.read(k.index)
	if strings.Count(index, retroMD) != 1 || !strings.Contains(index, "| pending | 1 |") {
		t.Errorf("index: %q", index)
	}
}

func TestDryRunPrintsAndWritesNothing(t *testing.T) {
	k := newKit(t)
	k.friction()
	before := k.read(k.index)
	code, out, _ := k.run("--dry-run")
	if code != 0 || !strings.Contains(out, "## Proposed kit deltas") ||
		!strings.HasSuffix(out, "\nINDEX row: | "+retroMD+" | LLM | 2026-10-01 | pending | 1 |\n") {
		t.Fatalf("exit %d, out %q", code, out)
	}
	if n := k.ls(); len(n) != 1 || k.read(k.index) != before {
		t.Errorf("wrote something: %v", n)
	}
}

func TestSinceFiltersTheWrittenDraftAndBadInputIsExit3(t *testing.T) {
	k := newKit(t)
	k.friction()
	_, early, _ := k.run("--since", "2026-10-01T00:00:00Z", "--dry-run")
	_, late, _ := k.run("--since", "2026-10-02T00:00:00Z", "--dry-run")
	if !strings.Contains(early, b1) || strings.Contains(late, b1) || !strings.Contains(late, "no new deltas;") {
		t.Errorf("--since did not filter")
	}
	if code, _, e := k.run("--since", "nope"); code != 3 || !strings.HasPrefix(e, "retro-draft: ") {
		t.Errorf("bad --since: %d %q", code, e)
	}
	code, _, e := runArgs("--write", "--station", "../evil", "--state-dir", k.state, "--out-dir", k.out)
	if code != 3 || e != "retro-draft: --station must match [A-Za-z0-9][A-Za-z0-9._-]{0,63}\n" {
		t.Errorf("bad station: %d %q", code, e)
	}
}

func TestAnImpossibleCalendarDateIsRefused(t *testing.T) {
	k := newKit(t)
	for _, bad := range []string{"2026-13-45", "2026-02-30", "0000-00-00"} {
		code, _, e := runArgs("--write", "--station", "LLM", "--state-dir", k.state, "--out-dir", k.out, "--date", bad)
		if code != 3 || e != "retro-draft: --date must be a real calendar date, YYYY-MM-DD\n" {
			t.Errorf("%s: %d %q", bad, code, e)
		}
	}
	if n := k.ls(); len(n) != 1 {
		t.Errorf("wrote something: %v", n)
	}
}

func TestMissingIndexIsExit3(t *testing.T) {
	k := newKit(t)
	os.Remove(k.index)
	code, _, e := k.run()
	if code != 3 || e != "retro-draft: "+k.index+" not found: create the retros directory with its INDEX.md first\n" {
		t.Errorf("%d %q", code, e)
	}
	if n := k.ls(); len(n) != 0 {
		t.Errorf("wrote something: %v", n)
	}
}

func TestBothFilesAreReplacedLeavingNoTemporaryFile(t *testing.T) {
	k := newKit(t)
	k.friction()
	if code, _, _ := k.run(); code != 0 {
		t.Fatal(code)
	}
	if n := k.ls(); len(n) != 2 || n[0] != retroMD || n[1] != "INDEX.md" {
		t.Errorf("files: %v", n)
	}
}

// Draft mode keeps its contract: --date is still required there, and the write-mode
// flags need --write.
func TestDraftModeStillRequiresDateAndRefusesWriteFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--state-dir", "d", "--station", "S"},
		{"--state-dir", "d", "--station", "S", "--date", "2026-10-01", "--dry-run"},
		{"--state-dir", "d", "--station", "S", "--date", "2026-10-01", "--force"},
		{"--state-dir", "d", "--station", "S", "--date", "2026-10-01", "--out-dir", "x"},
		{"--write", "--station", "S"},
	} {
		if code, out, e := runArgs(args...); code != 2 || out != "" || !strings.HasPrefix(e, "usage: retro-draft ") {
			t.Errorf("%q: %d %q %q", args, code, out, e)
		}
	}
}

func TestDefaultOutDirIsTheKitRetrosNextToTheBinary(t *testing.T) {
	exe, _ := os.Executable()
	exe, _ = filepath.EvalSymlinks(exe)
	want := filepath.Join(filepath.Dir(exe), "..", "..", "mcp-n4-kit", "retros")
	if got := defaultOutDir(); got != want {
		t.Errorf("defaultOutDir = %q, want %q", got, want)
	}
}

// Without --date the write mode dates the retro today in UTC, not in local time.
func TestTheDefaultDateIsTodayInUTC(t *testing.T) {
	k := newKit(t)
	now := time.Date(2026, 11, 14, 23, 30, 0, 0, time.FixedZone("UTC-6", -6*3600)) // 2026-11-15 UTC
	var out, errOut bytes.Buffer
	o := writeOpts{stateDir: k.state, station: "LLM", outDir: k.out}
	if code := writeRetro(o, now, cli.Stdio{Out: &out, Err: &errOut}); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(k.out, "2026-11-15-LLM-session.md")); err != nil {
		t.Errorf("not dated today in UTC: %v (files %v)", err, k.ls())
	}
}
