package livesmoke

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func resolve(t *testing.T, env map[string]string, extra ...string) (string, *Args) {
	t.Helper()
	a, err := ParseArgs(append([]string{"--apply"}, extra...))
	if err != nil {
		t.Fatal(err)
	}
	return ResolveServer(a, func(k string) string { return env[k] }), a
}

func executableFile(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "mcp-n4")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheGoBinaryIsTheDefaultLauncher(t *testing.T) {
	binary := executableFile(t)
	problem, a := resolve(t, map[string]string{"MCP_N4_GO_BIN": binary})
	if problem != "" {
		t.Fatal(problem)
	}
	if got := ServerArgv(a)[:2]; !reflect.DeepEqual(got, []string{binary, "--allow-writes"}) {
		t.Errorf("argv starts %v", got)
	}
}

func TestTheDefaultBinaryIsMcpN4NextToThisBinary(t *testing.T) {
	_, a := resolve(t, map[string]string{})
	if a.GoBin != DefaultGoBin() {
		t.Errorf("go bin = %s, want %s", a.GoBin, DefaultGoBin())
	}
	exe, _ := os.Executable()
	exe, _ = filepath.EvalSymlinks(exe)
	if a.GoBin != filepath.Join(filepath.Dir(exe), "mcp-n4") {
		t.Errorf("go bin = %s, not mcp-n4 next to %s", a.GoBin, exe)
	}
}

func TestTheFlagBeatsTheEnvironment(t *testing.T) {
	binary := executableFile(t)
	problem, a := resolve(t, map[string]string{"MCP_N4_GO_BIN": "/no/such/mcp-n4"}, "--go-bin", binary)
	if problem != "" {
		t.Fatal(problem)
	}
	if ServerArgv(a)[0] != binary {
		t.Errorf("argv[0] = %s", ServerArgv(a)[0])
	}
}

func TestAMissingGoBinaryNamesTheBuildAndStartsNothing(t *testing.T) {
	problem, _ := resolve(t, map[string]string{"MCP_N4_GO_BIN": "/no/such/mcp-n4"})
	for _, want := range []string{"/no/such/mcp-n4", "go build -o bin/ ./cmd/mcp-n4"} {
		if !strings.Contains(problem, want) {
			t.Errorf("problem %q misses %q", problem, want)
		}
	}
}

func TestThereIsNoPythonFallback(t *testing.T) {
	_, err := ParseArgs([]string{"--apply", "--impl", "python"})
	var ue *UsageError
	if !errors.As(err, &ue) || ue.Msg != "unrecognized arguments: --impl python" {
		t.Errorf("err = %v, want the argparse usage error", err)
	}
}

func TestTheChildOrdJoinNeverDoublesASlashAndRefusesPaths(t *testing.T) {
	for parent, want := range map[string]string{"station:|slot:/": "station:|slot:/X", "station:|slot:/A/": "station:|slot:/A/X"} {
		if got, err := ChildOrd(parent, "X"); err != nil || got != want {
			t.Errorf("ChildOrd(%s) = %s, %v", parent, got, err)
		}
	}
	for _, bad := range []string{"", "a/b", "a|b", ".."} {
		if _, err := ChildOrd("station:|slot:/A", bad); err == nil {
			t.Errorf("ChildOrd accepted %q", bad)
		}
	}
}

func TestAMissingGoBinaryIsExit2BeforeAnyStationContact(t *testing.T) {
	r := newRunner(t)
	r.setEnv("MCP_N4_GO_BIN=/no/such/mcp-n4")
	code, out, errOut := r.run(r.argv(true))
	if code != 2 || !strings.Contains(errOut, "go build") {
		t.Errorf("exit %d\n%s%s", code, out, errOut)
	}
	if n := r.fake.Requests(); n != 0 {
		t.Errorf("%d requests reached the station", n)
	}
}

func TestMainRunsWithoutSideEffectsInPlanMode(t *testing.T) {
	r := newRunner(t)
	if code, _, _ := r.run(nil); code != 0 {
		t.Errorf("plan-only exit %d", code)
	}
}

func TestScopeIsTheScratchFolderPlusTheRoot(t *testing.T) {
	if got := WriteScopes(); !reflect.DeepEqual(got, []string{"station:|slot:/McpSmoke", "station:|slot:/"}) {
		t.Errorf("WriteScopes = %v", got)
	}
}
