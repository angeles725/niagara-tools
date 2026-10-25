package niagaratest

import (
	"bytes"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// TestStdin covers what a golden case cannot feed: the log on standard input, named "-" or not named.
func TestStdin(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"verdict"}, {"verdict", "-"}} {
		var o, e bytes.Buffer
		code := run(args, cli.Stdio{Out: &o, Err: &e}, strings.NewReader("Total tests run: 1, Failures: 0, Skips: 0\n"))
		want := "PASS  run-niagara-test  line 1: run 1, failures 0, skips 0\n" +
			"PASS  run-niagara-test  verdict: 1 test summary, 1 test run, 0 failures\n"
		if code != 0 || o.String() != want || e.Len() != 0 {
			t.Errorf("%q: exit %d, stdout %q, stderr %q", args, code, o.String(), e.String())
		}
	}
}
