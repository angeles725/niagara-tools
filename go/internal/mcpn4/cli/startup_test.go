package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// The Go twins of test_tools_write's start tests (go_twin("cli", ...)) and the write start
// differential.

func looseDir(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	os.Chmod(dir, 0o777)
	return dir
}

func TestDefaultStateDirIsUnderTheHomeStateDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := stateDir(""); got != filepath.Join(home, ".local", "state", "mcp-n4") {
		t.Errorf("stateDir() = %q", got)
	}
}

func TestWriteModeRefusesALooseExistingStateDirAndLeavesItAlone(t *testing.T) {
	dir := looseDir(t)
	_, err := NewServer(Config{AllowWrites: true, StateDir: dir, TokenTTL: 300, MaxWrites: 200})
	var se *safety.Error
	if !errors.As(err, &se) {
		t.Fatalf("err = %v", err)
	}
	if info, _ := os.Stat(dir); info.Mode().Perm() != 0o777 {
		t.Errorf("mode = %o", info.Mode().Perm())
	}
}

func TestReadOnlyModeDoesNotCareAboutTheStateDir(t *testing.T) {
	if _, err := NewServer(Config{StateDir: looseDir(t)}); err != nil {
		t.Fatal(err)
	}
}

func TestMainExits2WithAClearMessage(t *testing.T) {
	code, out, errOut := run([]string{"--allow-writes", "--state-dir", looseDir(t)}, "")
	if code != 2 || out != "" || !strings.Contains(errOut, "--state-dir") {
		t.Errorf("got %d %q %q", code, out, errOut)
	}
}

func TestMainExits2ForAChainedOrNonSlotWriteScope(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	for _, bad := range []string{"station:|slot:/A|h:1f", "station:|h:1f", "station:|slot:/A/../B"} {
		code, _, errOut := run([]string{"--allow-writes", "--state-dir", dir, "--write-scope", bad}, "")
		if code != 2 || !strings.Contains(errOut, "--write-scope") {
			t.Errorf("%s: got %d %q", bad, code, errOut)
		}
	}
}

func TestMainRejectsMalformedStationValues(t *testing.T) {
	for _, bad := range []string{"nourl", "=https://h", "name=", ""} {
		code, _, errOut := run([]string{"--station", bad}, "")
		if code != 2 || !strings.Contains(errOut, "--station expects NAME=URL") {
			t.Errorf("%q: got %d %q", bad, code, errOut)
		}
	}
}

func TestTheWriteStateRefusesThemToo(t *testing.T) {
	dir := t.TempDir()
	for _, lim := range [][2]int{{0, 200}, {300, 0}, {-1, 200}} {
		_, err := plan.NewState([]string{"station:|slot:/A"}, dir, lim[0], lim[1])
		var ve *pyjson.ValueError
		if !errors.As(err, &ve) {
			t.Errorf("%v: err = %v", lim, err)
		}
	}
}

// TestWriteStartDifferentialPython: a start with --allow-writes on each kind of state dir and
// scope exits, prints and refuses as `python3 -m mcp_n4.server` does (same dirs, empty stdin),
// recorded in testdata/write-start.golden.jsonl.
func TestWriteStartDifferentialPython(t *testing.T) {
	g := golden.Open(t, "write-start")
	tmp := t.TempDir()
	g.Alias(filepath.Dir(tmp), "$TMP") // every t.TempDir of the test shares it
	loose := looseDir(t)
	file := filepath.Join(tmp, "file")
	os.WriteFile(file, nil, 0o600)
	withLoose := filepath.Join(tmp, "with-loose")
	os.Mkdir(withLoose, 0o700)
	os.WriteFile(filepath.Join(withLoose, "audit.jsonl"), nil, 0o640)
	missing := filepath.Join(tmp, "missing", "state")
	argvs := [][]string{
		{"--state-dir", loose}, {"--state-dir", file}, {"--state-dir", withLoose}, {"--state-dir", missing},
		{"--state-dir", missing, "--write-scope", "station:|slot:/A", "--write-scope", "station:|slot:/A|h:1"},
		{"--state-dir", missing, "--write-scope", "station:"}, {"--state-dir", missing, "--write-scope", "station:|slot:/A/"},
		{"--state-dir", loose, "--write-scope", "x"}, {"--state-dir", loose, "--station", "a=http://h"},
		{"--state-dir", missing, "--token-ttl", "1", "--max-writes", "1"},
	}
	for i := range argvs {
		argvs[i] = append([]string{"--allow-writes"}, argvs[i]...)
	}
	want := pythonServer(t, g, map[string]string{"PYTHONDONTWRITEBYTECODE": "1"}, argvs)
	for i, argv := range argvs {
		code, out, errOut := run(argv, "")
		if w := want[i]; code != w.code || out != w.out || errOut != w.err {
			t.Errorf("%q:\n go     %d %q %q\n python %d %q %q", argv, code, out, errOut, w.code, w.out, w.err)
		}
	}
	if _, err := os.Stat(missing); err == nil {
		t.Error("a start created the state dir")
	}
}
