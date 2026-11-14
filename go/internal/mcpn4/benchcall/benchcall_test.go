package benchcall

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// serverMarker makes the test binary act as the mcp-n4 server, the way n4call starts go/bin/mcp-n4.
const serverMarker = "BENCHCALL_TEST_AS_MCP_N4"

func TestMain(m *testing.M) {
	if os.Getenv(serverMarker) == "1" {
		os.Exit(mcpcli.Main(os.Args[1:], cli.Stdio{Out: os.Stdout, Err: os.Stderr, Getenv: os.Getenv}))
	}
	os.Exit(m.Run())
}

const (
	user     = "user-Bc11-unique-canary"
	password = "pw-Bc22-unique-canary"
)

// bench is a fake station, the calls file and the command line that reaches it.
type bench struct {
	t    *testing.T
	fake *boxstation.Station
	dir  string
	env  []string
}

func newBench(t *testing.T) *bench {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b := &bench{t: t, fake: boxstation.New(user, password, "Lab"), dir: t.TempDir()}
	b.fake.Start(t)
	b.env = append(os.Environ(), "MCP_N4_USER="+user, "MCP_N4_PASSWORD="+password, serverMarker+"=1", "BENCHCALL_SERVER="+exe)
	return b
}

// run writes calls to a file and runs n4call on it; it returns the exit code, the decoded
// output lines and stderr.
func (b *bench) run(calls string, flags ...string) (int, []map[string]any, string) {
	b.t.Helper()
	file := filepath.Join(b.dir, "calls.jsonl")
	if err := os.WriteFile(file, []byte(calls), 0o600); err != nil {
		b.t.Fatal(err)
	}
	exe, _ := os.Executable()
	argv := append([]string{"--calls", file}, flags...)
	argv = append(argv, "--", exe, "--station", "Lab="+b.fake.URL, "--allow-http-for-tests", "--credential-env", "MCP_N4",
		"--allow-writes", "--write-scope", "station:|slot:/", "--state-dir", filepath.Join(b.dir, "state"))
	var out, errOut bytes.Buffer
	code := Main(argv, cli.Stdio{Out: &out, Err: &errOut}, b.env)
	var lines []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			b.t.Fatalf("output line is not JSON: %v\n%s", err, l)
		}
		lines = append(lines, m)
	}
	return code, lines, errOut.String()
}

const connect = `{"tool":"n4_connect","args":{"station":"Lab","expected_station":"Lab"}}` + "\n"

func TestAReadCallPrintsItsResultAsOneJSONLine(t *testing.T) {
	b := newBench(t)
	code, lines, errOut := b.run(connect)
	if code != 0 || len(lines) != 1 {
		t.Fatalf("exit %d, %d lines\n%s", code, len(lines), errOut)
	}
	res, _ := lines[0]["result"].(map[string]any)
	if lines[0]["tool"] != "n4_connect" || lines[0]["phase"] != "read" || lines[0]["error"] != false || res["station_name"] != "Lab" {
		t.Errorf("line: %v", lines[0])
	}
}

func TestAWriteCallDoesDryRunThenExecutesWithTheToken(t *testing.T) {
	b := newBench(t)
	create := `{"tool":"n4_create_component","write":true,"args":{"parent_ord":"station:|slot:/","name":"BenchFolder","type":"baja:Folder"}}` + "\n"
	code, lines, errOut := b.run(connect + create)
	if code != 0 || len(lines) != 3 {
		t.Fatalf("exit %d, %d lines\n%s", code, len(lines), errOut)
	}
	if lines[1]["phase"] != "dry_run" || lines[2]["phase"] != "execute" || lines[2]["error"] != false {
		t.Errorf("phases: %v / %v", lines[1], lines[2])
	}
	if !b.fake.Has("BenchFolder") {
		t.Error("the component was not created on the station")
	}
}

func TestDryRunOnlyNeverExecutes(t *testing.T) {
	b := newBench(t)
	create := `{"tool":"n4_create_component","write":true,"args":{"parent_ord":"station:|slot:/","name":"BenchFolder","type":"baja:Folder"}}` + "\n"
	code, lines, _ := b.run(connect+create, "--dry-run-only")
	if code != 0 || len(lines) != 2 || lines[1]["phase"] != "dry_run" || b.fake.Has("BenchFolder") {
		t.Errorf("exit %d lines %d present=%v", code, len(lines), b.fake.Has("BenchFolder"))
	}
}

func TestAToolErrorIsExit1AndStopsTheRun(t *testing.T) {
	b := newBench(t)
	bad := `{"tool":"n4_read_slots","args":{"ord":"station:|slot:/NoSuchThing"}}` + "\n"
	code, lines, _ := b.run(connect + bad + connect)
	if code != 1 || len(lines) != 2 || lines[1]["error"] != true {
		t.Errorf("exit %d, lines %v", code, lines)
	}
}

func TestUsageErrorsAreExit2AndStartNothing(t *testing.T) {
	for _, argv := range [][]string{{}, {"--calls", "x"}, {"--calls"}, {"--nope", "--", "srv"}} {
		var out, errOut bytes.Buffer
		if code := Main(argv, cli.Stdio{Out: &out, Err: &errOut}, nil); code != 2 || !strings.Contains(errOut.String(), "usage: n4call") {
			t.Errorf("%v: exit %d\n%s", argv, code, errOut.String())
		}
	}
}

func TestAMalformedCallLineIsRefusedWithItsLineNumberBeforeTheServerStarts(t *testing.T) {
	b := newBench(t)
	code, lines, errOut := b.run(connect + "{not json}\n")
	if code != 2 || len(lines) != 0 || !strings.Contains(errOut, "calls.jsonl:2:") || b.fake.Requests() != 0 {
		t.Errorf("exit %d lines %d requests %d\n%s", code, len(lines), b.fake.Requests(), errOut)
	}
}

func TestNoCredentialValueReachesTheOutput(t *testing.T) {
	b := newBench(t)
	echo := `{"tool":"n4_read_slots","args":{"ord":"station:|slot:/` + password + user + `"}}` + "\n"
	code, lines, errOut := b.run(connect + echo)
	if code != 1 || len(lines) != 2 {
		t.Fatalf("exit %d, %d lines: the echoing call did not run\n%s", code, len(lines), errOut)
	}
	text, _ := json.Marshal(lines)
	if !strings.Contains(string(text), "***") {
		t.Errorf("the refusal that echoes the ord shows no mask: %s", text)
	}
	if all := string(text) + errOut; strings.Contains(all, password) || strings.Contains(all, user) {
		t.Errorf("a credential value leaked:\n%s", all)
	}
}

func TestTheCredentialPrefixIsReadInBothFlagForms(t *testing.T) {
	env := []string{"LAB_USER=u1", "LAB_PASSWORD=p1", "MCP_N4_PASSWORD=other", "LAB_EMPTY="}
	for _, server := range [][]string{{"srv", "--credential-env", "LAB"}, {"srv", "--credential-env=LAB"}} {
		if got := strings.Join(secretsOf(server, env), ","); got != "u1,p1" {
			t.Errorf("%v: secrets %q", server, got)
		}
	}
	if got := strings.Join(secretsOf([]string{"srv"}, env), ","); got != "other" {
		t.Errorf("default prefix: secrets %q", got)
	}
}
