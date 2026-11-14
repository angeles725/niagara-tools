package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// fixtureScript is the build.sh the fixture puts in its toolbelt directory; fixtureUsage is its header.
const (
	fixtureScript = "#!/usr/bin/env bash\n# build.sh usage\n#   two spaces kept\n#\n#no space\nset -u\n# not usage\n"
	fixtureUsage  = "build.sh usage\n  two spaces kept\n\nno space\n"
)

func TestUsageIsTheHeaderOfTheInvokedScript(t *testing.T) {
	if got := scriptHeader(fixtureScript); got != fixtureUsage {
		t.Errorf("header %q, want %q", got, fixtureUsage)
	}
	if got := scriptHeader("#!/bin/sh"); got != "" {
		t.Errorf("a one-line script has header %q", got)
	}
}

func TestUsageOfAnUnreadableScriptSaysSo(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(f.kit + "/" + scriptName); err != nil {
		t.Fatal(err)
	}
	want := "build.sh: usage text unavailable: cannot read " + f.kit + "/" + scriptName + "\n"
	if code := f.run("--help"); code != ExitOK || f.out != want {
		t.Errorf("exit %d stdout %q, want %q", code, f.out, want)
	}
}

// TestTheRealScriptHasAUsage: the shipped launcher keeps a header the tool can print.
func TestTheRealScriptHasAUsage(t *testing.T) {
	script, err := os.ReadFile("../../../build-n4-module-kit/toolbelt/" + scriptName)
	if err != nil {
		t.Fatal(err)
	}
	if u := scriptHeader(string(script)); !strings.HasPrefix(u, "build.sh") || !strings.Contains(u, "Exit: 0 chain passed") {
		t.Errorf("the header of toolbelt/build.sh is not the usage text: %.120q", u)
	}
}
