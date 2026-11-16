package main

import (
	"bytes"
	"strings"
	"testing"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
)

func TestTheFirstArgumentSelectsASubcommand(t *testing.T) {
	for _, c := range []struct{ arg, want string }{
		{"install", "usage: mcp-n4 install"}, {"register", "usage: mcp-n4 register"},
	} {
		var out, errb bytes.Buffer
		code := run([]string{c.arg, "--help"}, kitcli.Stdio{Out: &out, Err: &errb})
		if code != 0 || !strings.HasPrefix(out.String(), c.want) {
			t.Errorf("%s --help: exit %d, stdout %q", c.arg, code, out.String())
		}
	}
}

func TestFlagsStillReachTheServer(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"--bogus"}, kitcli.Stdio{Out: &out, Err: &errb}); code != 2 ||
		!strings.Contains(errb.String(), "unrecognized arguments: --bogus") {
		t.Fatalf("exit %d, stderr %q, want the server's usage error", code, errb.String())
	}
}
