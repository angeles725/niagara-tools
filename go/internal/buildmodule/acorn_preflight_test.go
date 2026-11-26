package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// acornFixture is a build whose Foo-ux holds a vendored library (the input of report-module's vendor-floor check)
// and whose KIT_NODE is a stub: it logs its arguments to node.log and resolves acorn only when acorn-ok exists.
func acornFixture(t *testing.T, resolvable bool) *fixture {
	t.Helper()
	f := newFixture(t)
	f.write(f.root+"/Foo/Foo-ux/src/rc/vendor/lib/chart.min.js", "")
	node := f.dir + "/bin/node"
	f.write(node, "#!/usr/bin/env bash\n{ printf '%s' node; printf ' <%s>' \"$@\"; echo; } >> \""+f.dir+"/node.log\"\n"+
		"[ -f \""+f.dir+"/acorn-ok\" ]\n")
	if err := os.Chmod(node, 0o755); err != nil {
		t.Fatal(err)
	}
	if resolvable {
		f.write(f.dir+"/acorn-ok", "")
	}
	f.env["KIT_NODE"], f.env["KIT_ACORN"] = node, ""
	return f
}

// TestAcornPreflightNoNode: vendored JS and no node is exit 10 before gradle, naming what is missing and the
// one way to fix it, not a SKIP found by report-module at the end of the build (#1116).
func TestAcornPreflightNoNode(t *testing.T) {
	f := acornFixture(t, true)
	f.env["KIT_NODE"] = f.dir + "/bin/no-such-node"
	f.wants("no node", f.build(), ExitEnv, "build.sh: FAIL — node not found", "set KIT_NODE=<node binary>", "Foo-ux/src/rc/vendor")
	if f.call("gradlew") != "" {
		t.Errorf("gradle ran after the preflight failed: %s", f.calls())
	}
}

// TestAcornPreflightNoAcorn: node there, acorn not resolvable: exit 10 before gradle with the install command
// (toolbelt/eslint of the invoked toolbelt) and the KIT_ACORN alternative.
func TestAcornPreflightNoAcorn(t *testing.T) {
	f := acornFixture(t, false)
	f.wants("no acorn", f.build(), ExitEnv, "build.sh: FAIL — acorn is not resolvable from 'acorn'",
		"npm install --prefix "+f.kit+"/eslint", "or set KIT_ACORN=<acorn package dir>")
	if f.call("gradlew") != "" {
		t.Errorf("gradle ran after the preflight failed: %s", f.calls())
	}
}

// TestAcornPreflightResolvable: node and acorn resolve, the chain goes on and prints nothing about acorn.
func TestAcornPreflightResolvable(t *testing.T) {
	f := acornFixture(t, true)
	f.wants("resolvable", f.build(), ExitOK)
	if strings.Contains(f.out+f.err, "acorn") || f.call("gradlew") == "" {
		t.Errorf("a resolvable acorn must be silent and let gradle run:\n%s\n%s\n%s", f.out, f.err, f.calls())
	}
}

// TestAcornPreflightSpec: the require() spec node is asked for is KIT_ACORN, else the toolbelt's own npm install
// when it exists, else the bare name (the resolution lint-vendor-floor itself uses).
func TestAcornPreflightSpec(t *testing.T) {
	for _, c := range []struct{ name, kitAcorn, installed, want string }{
		{"KIT_ACORN", "/opt/acorn", "", "/opt/acorn"},
		{"toolbelt install", "", "yes", "<kit>/eslint/node_modules/acorn"},
		{"bare name", "", "", "acorn"},
	} {
		f := acornFixture(t, true)
		f.env["KIT_ACORN"] = c.kitAcorn
		if c.installed != "" {
			f.mkdir(f.kit + "/eslint/node_modules/acorn")
		}
		f.wants(c.name, f.build(), ExitOK)
		want := "node <-e> <require(process.argv[1])> <" + strings.ReplaceAll(c.want, "<kit>", f.kit) + ">"
		if got := strings.TrimSpace(f.read(f.dir + "/node.log")); got != want {
			t.Errorf("%s: node was run as %q, want %q", c.name, got, want)
		}
	}
}

// TestAcornPreflightOnlyForVendoredJS: a module with no vendored library needs no node and no acorn, so a build
// without them is untouched; --no-preflight skips the check for a module that has one.
func TestAcornPreflightOnlyForVendoredJS(t *testing.T) {
	f := newFixture(t)
	f.env["KIT_NODE"] = f.dir + "/bin/no-such-node"
	f.wants("no vendor/", f.build(), ExitOK)
	f.mkdir(f.root + "/Foo/Foo-ux/src/rc/vendor/empty")
	f.wants("vendor/ without a script", f.build(), ExitOK)
	f.write(f.root+"/Foo/Foo-ux/src/rc/vendor/a.js", "")
	f.wants("--no-preflight", f.build("--no-preflight"), ExitOK)
	f.wants("vendored JS", f.build(), ExitEnv, "node not found")
}
