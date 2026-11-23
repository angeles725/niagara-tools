package buildmodule

import (
	"github.com/angeles725/niagara-tools/go/internal/fstype"
	"os"
	"strings"
	"testing"
)

// forbidHostTools puts a shim for each name at the front of PATH that records its call in the returned log
// and fails: the chain does its file work itself (retro go-build-module-port Δ3), so no shim may run.
func (f *fixture) forbidHostTools(names ...string) (log string) {
	f.t.Helper()
	bin, log := f.dir+"/forbidden-bin", f.dir+"/forbidden.log"
	for _, name := range names {
		f.write(bin+"/"+name, "#!/bin/sh\necho \""+name+" $*\" >> '"+log+"'\nexit 99\n")
		if err := os.Chmod(bin+"/"+name, 0o755); err != nil {
			f.t.Fatal(err)
		}
	}
	f.t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	return log
}

// TestTheChainDoesItsFileWorkItself: the drift snapshot (temporary directory, copy with mode and times, the
// module version read from the jar), the mirror's kept jar, the restore after a failed report, the
// extraction of a --schema-baseline-ref and the filesystem-type probe run without cp, mkdir, mktemp, unzip
// or df: none of the shims is called and every run still ends as it did with the real tools.
func TestTheChainDoesItsFileWorkItself(t *testing.T) {
	for _, c := range []struct {
		name   string
		report string
		code   int
	}{{"a passing build", "exit 0", ExitOK}, {"a report FAIL restores the jar", "exit 1", ExitGate}} {
		f, _ := mirrorBuild(t)
		f.does("gradlew", `cat "`+f.built("rt")+`" > "`+f.deployed("rt")+`"`) // the stock stub copies with cp
		f.env[fstype.MountsEnv] = ""                                          // the type probe reads the real mount table
		f.deployedRepo()
		f.seesBaselines(strings.TrimPrefix(c.report, "exit "))
		before := f.read(f.deployed("rt"))
		log := f.forbidHostTools("cp", "mkdir", "mktemp", "unzip", "df")
		f.wants(c.name, f.run("--schema-baseline-ref", "v1", f.root, "Foo", f.home), c.code)
		if got := f.read(log); got != "" {
			t.Errorf("%s: host tools ran:\n%s", c.name, got)
		}
		if c.code == ExitGate && f.read(f.deployed("rt")) != before {
			t.Errorf("%s: the pre-build jar is not back in modules/", c.name)
		}
	}
}
