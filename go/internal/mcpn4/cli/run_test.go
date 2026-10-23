package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
)

func run(args []string, stdin string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = Run(args, strings.NewReader(stdin), kitcli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

// argvCorpus runs through both servers with an empty stdin. --allow-writes without a flag error
// is left out: this version refuses it on purpose (TestAllowWritesIsRefusedUntilS10).
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

// TestDifferentialPython is the command-line equivalence proof: for every argv above, exit code,
// stdout and stderr equal those of `python3 -m mcp_n4.server` with an empty stdin (argparse at
// 80 columns, no color). It skips when python3 is missing.
func TestDifferentialPython(t *testing.T) {
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not on PATH")
	}
	kit, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		code     int
		out, err string
	}
	want := make([]result, len(argvCorpus))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, argv := range argvCorpus {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cmd := exec.Command(py, append([]string{"-m", "mcp_n4.server"}, argv...)...)
			cmd.Dir = kit
			cmd.Env = append(os.Environ(), "COLUMNS=80", "NO_COLOR=1", "PYTHON_COLORS=0", "PYTHONDONTWRITEBYTECODE=1")
			var o, e bytes.Buffer
			cmd.Stdout, cmd.Stderr = &o, &e
			code := 0
			if err := cmd.Run(); err != nil {
				ee, ok := err.(*exec.ExitError)
				if !ok {
					t.Errorf("%q: %v", argv, err)
					return
				}
				code = ee.ExitCode()
			}
			want[i] = result{code, o.String(), e.String()}
		}()
	}
	wg.Wait()
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

func TestAllowWritesIsRefusedUntilS10(t *testing.T) {
	t.Parallel()
	code, out, errOut := run([]string{"--allow-writes", "--station", "s=https://h"}, `{"id":1,"method":"ping"}`)
	if code != 2 || out != "" || errOut != "mcp_n4.server: --allow-writes "+"is not implemented in this version of the Go server\n" {
		t.Errorf("got %d %q %q", code, out, errOut)
	}
}

func TestEofEndsTheServerWithExitZero(t *testing.T) {
	t.Parallel()
	code, out, errOut := run(nil, "{\"id\":1,\"method\":\"ping\"}\n\n{\"method\":\"ping\"}")
	if code != 0 || out != "{\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {}}\n" || errOut != "" {
		t.Errorf("got %d %q %q", code, out, errOut)
	}
}
