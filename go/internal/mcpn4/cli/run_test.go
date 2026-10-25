package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
)

func run(args []string, stdin string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = Run(args, strings.NewReader(stdin), kitcli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

// argvCorpus runs through both servers with an empty stdin. A start with --allow-writes reads
// the state dir: TestWriteStartDifferentialPython runs those on private temporary dirs.
var argvCorpus = [][]string{
	{}, {"--help"}, {"-h"}, {"--he"}, {"--h"}, {"-hx"}, {"-h-x"}, {"-h=x"}, {"--help=x"}, {"--station", "--", "x"}, {"--", "--", "-h"}, {"--allow"}, {"--allow=1"}, {"--bogus"}, {"x"}, {"-x"}, {"-5"},
	{"--", "--allow-http-for-tests"}, {"--allow-http"}, {"--allow-http-for-tests=1"}, {"--allow-writes="},
	{"--token-ttl"}, {"--token-ttl", "0"}, {"--token-ttl", "-5"}, {"--token-ttl", "abc"}, {"--token-ttl", " 7 "},
	{"--token-ttl", "1_0"}, {"--token-ttl", "1__0"}, {"--token-ttl", "0x1"}, {"--token-ttl", "+3"}, {"--token-ttl="},
	{"--token-ttl", "99999999999999999999999999"}, {"--max-writes", "-0"}, {"--max-writes", "1.0"},
	{"--auth-cooldown", "5"}, {"--auth-cooldown", "0"}, {"--auth-cooldown", "30"}, {"--auth-c", "45"},
	{"--load-wait", "0.5"}, {"--load-wait", "x"}, {"--load-wait", "nan"}, {"--load-wait", "-NaN"}, {"--load-wait", "inf"},
	{"--load-wait", "1e400"}, {"--load-wait", "60"}, {"--load-wait", "1_5"}, {"--load-wait", ".5e1"}, {"--load-wait", "0x10"},
	{"--load-wait", "5."}, {"--load-wait", "1e"}, {"--http-timeout", "4"}, {"--http-timeout", "301"}, {"--http-timeout", "x"},
	{"--http-timeout", "5.0"}, {"--http-timeout", "300"}, {"--station", "-x"}, {"--station", "-a b"}, {"--station"},
	{"--station", "a"}, {"--station", "=https://h"}, {"--station", "a="}, {"--station", "a=b"}, {"--station", "a=http://h"},
	{"--allow-http-for-tests", "--station", "a=http://h"}, {"--station", "a=https://h", "--station", "a=http://h"},
	{"--station=bad", "--token-ttl=0"}, {"--station-home", "x"}, {"--station-home", "a=/p"},
	{"--insecure-tls", "z", "--allow-tier-b", "y", "--station", "a=https://q"},
	{"--allow-tier-c", "b", "--allow-tier-c", "a", "--station", "c=https://q"},
	{"--station", "s=https://q", "--insecure-tls", "s", "--allow-tier-b", "s", "--allow-tier-c", "s"},
	{"--station", "a\xffb"}, {"--credential-env", "LAB", "--state-dir", "/x", "--progress-file", "/p", "--write-scope", "s"},
	{"--token-ttl", "0", "--bogus"}, {"--bogus", "--allow"}, {"--allow-writes", "--token-ttl", "0"},
	{"--allow-writes", "--station", "a=b"},
}

// serverScript starts `python3 -m mcp_n4.server` from the kit once per argv of the case, four
// at a time, with the case's environment added and an empty stdin, and prints each exit code,
// stdout and stderr. Arguments and output travel as Latin-1 text, so any byte round-trips.
const serverScript = `import json, os, subprocess, sys
from concurrent.futures import ThreadPoolExecutor
case = json.load(sys.stdin)
env = dict(os.environ, **case["env"])
def run(argv):
    p = subprocess.run([os.fsencode(sys.executable), b"-m", b"mcp_n4.server"] + [a.encode("latin-1") for a in argv],
                       cwd=sys.path[0], env=env, stdin=subprocess.DEVNULL, capture_output=True)
    return [p.returncode, p.stdout.decode("latin-1"), p.stderr.decode("latin-1")]
with ThreadPoolExecutor(4) as pool:
    print(json.dumps(list(pool.map(run, case["argvs"]))))`

type result struct {
	code     int
	out, err string
}

// pythonServer is what serverScript, recorded in g, answers for each argv.
func pythonServer(t *testing.T, g *golden.Golden, env map[string]string, argvs [][]string) []result {
	t.Helper()
	latin1 := func(s string) string { // each byte as the code point of the same value
		r := make([]rune, len(s))
		for i := range len(s) {
			r[i] = rune(s[i])
		}
		return string(r)
	}
	bytesOf := func(s string) string {
		var b []byte
		for _, r := range s {
			b = append(b, byte(r))
		}
		return string(b)
	}
	var enc [][]string
	for _, argv := range argvs {
		e := []string{}
		for _, a := range argv {
			e = append(e, latin1(a))
		}
		enc = append(enc, e)
	}
	in, err := json.Marshal(map[string]any{"env": env, "argvs": enc})
	if err != nil {
		t.Fatal(err)
	}
	var rows [][3]any
	if err := json.Unmarshal(g.Run(serverScript, in), &rows); err != nil || len(rows) != len(argvs) {
		t.Fatalf("python answered %d of %d argvs (%v)", len(rows), len(argvs), err)
	}
	out := make([]result, len(rows))
	for i, r := range rows {
		out[i] = result{int(r[0].(float64)), bytesOf(r[1].(string)), bytesOf(r[2].(string))}
	}
	return out
}

// TestDifferentialPython is the command-line equivalence proof: for every argv above, exit code,
// stdout and stderr equal those of `python3 -m mcp_n4.server` with an empty stdin (argparse at
// 80 columns, no color), recorded in testdata/argv.golden.jsonl.
func TestDifferentialPython(t *testing.T) {
	want := pythonServer(t, golden.Open(t, "argv"),
		map[string]string{"COLUMNS": "80", "NO_COLOR": "1", "PYTHON_COLORS": "0", "PYTHONDONTWRITEBYTECODE": "1"}, argvCorpus)
	for i, argv := range argvCorpus {
		code, out, errOut := run(argv, "")
		if got := (result{code, out, errOut}); got != want[i] {
			t.Errorf("%q:\n go     %d %q %q\n python %d %q %q", argv, got.code, got.out, lastLine(got.err),
				want[i].code, want[i].out, lastLine(want[i].err))
		}
	}
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1]
}

func TestEofEndsTheServerWithExitZero(t *testing.T) {
	t.Parallel()
	code, out, errOut := run(nil, "{\"id\":1,\"method\":\"ping\"}\n\n{\"method\":\"ping\"}")
	if code != 0 || out != "{\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {}}\n" || errOut != "" {
		t.Errorf("got %d %q %q", code, out, errOut)
	}
}

// The help is where the operator reads the protocol contract (test_server's doc-contract test):
// the versions spoken, and that a newer one is added only once its semantics are implemented.
func TestHelpStatesTheSupportedProtocolSetContract(t *testing.T) {
	t.Parallel()
	code, out, _ := run([]string{"--help"}, "")
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{"this server speaks 2025-06-18, 2025-03-26 and 2024-11-05",
		"A client asking for a newer revision is answered with 2025-06-18. Only add a newer revision to " +
			"that tuple once its semantics (new message shapes, capabilities, required fields) are implemented"} {
		if code != 0 || !strings.Contains(flat, want) {
			t.Errorf("--help (exit %d) lacks %q", code, want)
		}
	}
}
