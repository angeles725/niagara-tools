package commissioningverify

import (
	"bytes"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		args []string
		want options
		ok   bool
	}{
		{[]string{"m", "--bog", "b", "--module", "M", "--values-owed", "v", "--wiring-map", "w", "--strict"}, options{"m", "b", "M", "v", "w", true}, true},
		{[]string{"m", "--", "-x", "n"}, options{root: "m"}, true},
		{[]string{"--", "m"}, options{}, true},
		{[]string{"m", "--bog"}, options{root: "m"}, false},
		{[]string{"m", "-"}, options{root: "m"}, false},
		{[]string{"m", "n"}, options{root: "m"}, false},
	} {
		if got, ok := parseArgs(c.args); got != c.want || ok != c.ok {
			t.Errorf("parseArgs(%q) = %+v, %v; want %+v, %v", c.args, got, ok, c.want, c.ok)
		}
	}
}

func TestMainEnvErrors(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	for _, c := range []struct {
		args   []string
		stderr string
	}{
		{nil, usage},
		{[]string{d + "/nope"}, "commissioning-verify: not a directory: " + d + "/nope\n"},
		{[]string{d, "--values-owed", d}, "commissioning-verify: values-owed table not found: " + d + "\n"},
		{[]string{d, "--wiring-map", d + "/w"}, "commissioning-verify: wiring map not found: " + d + "/w\n"},
		{[]string{d, "--bog", "b"}, "commissioning-verify: --bog requires --module\n" + usage},
		{[]string{d}, "commissioning-verify: no -rt/src or src/ found under " + d + "\n"},
	} {
		var o, e bytes.Buffer
		if code := Main(c.args, cli.Stdio{Out: &o, Err: &e}); code != ExitEnv || o.Len() != 0 || e.String() != c.stderr {
			t.Errorf("Main(%q) = %d, out %q, err %q; want 3, no rows, err %q", c.args, code, o.String(), e.String(), c.stderr)
		}
	}
}
