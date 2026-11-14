package ngdeploy

import (
	"strings"
	"testing"
)

func TestTheVerifyModuleGate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture)
		args    []string
		code    int
		want    string
		call    string // the gate call, "" when it must not run
	}{
		{"passes", func(*fixture) {}, nil, ExitOK, "[ng-deploy] gate: verify-module on demo-rt.jar\ndemo-ux.jar\n[ng-deploy] gate ok\n[ng-deploy] copy ok\n", "vm.sh <--src> <SRC/demo> <RT> <UX>\n"},
		{"mode C gates the rt jar only", func(*fixture) {}, []string{"--mode", "C"}, ExitOK, "[ng-deploy] gate: verify-module on demo-rt.jar\n[ng-deploy] gate ok\n", "vm.sh <--src> <SRC/demo> <RT>\n"},
		{"fails", func(f *fixture) { f.t.Setenv("FAKE_VM_EXIT", "1") }, nil, ExitVerify,
			"err:[ng-deploy] ERROR verify-module gate failed; jars not fit to deploy (use --no-gate to override)\n", "vm.sh <--src> <SRC/demo> <RT> <UX>\n"},
		{"fails in mode C", func(f *fixture) { f.t.Setenv("FAKE_VM_EXIT", "1") }, []string{"--mode", "C"}, ExitVerify, "err:verify-module gate failed", "vm.sh <--src> <SRC/demo> <RT>\n"},
		{"mode B has no gate", func(f *fixture) { f.t.Setenv("FAKE_VM_EXIT", "1") }, []string{"--mode", "B"}, ExitOK, "[ng-deploy] copy ok\n", ""},
		{"--no-gate", func(f *fixture) { f.t.Setenv("FAKE_VM_EXIT", "1") }, []string{"--no-gate"}, ExitOK, "[ng-deploy] copy ok\n", ""},
		{"a missing gate blocks", func(f *fixture) { f.t.Setenv("VERIFY_MODULE_BIN", f.dir+"/absent") }, nil, ExitVerify,
			"err:[ng-deploy] ERROR verify-module gate unavailable (not executable at DIR/absent); jars not verified (use --no-gate to override)\n", ""},
		{"a gate that is not executable blocks", func(f *fixture) { f.chmod(f.dir+"/tools/vm.sh", 0o644) }, []string{"--mode", "C"}, ExitVerify, "err:verify-module gate unavailable (not executable at DIR/tools/vm.sh)", ""},
		{"--no-gate needs no gate", func(f *fixture) { f.t.Setenv("VERIFY_MODULE_BIN", f.dir+"/absent") }, []string{"--no-gate"}, ExitOK, "[ng-deploy] copy ok\n", ""},
	} {
		f := newFixture(t)
		tc.arrange(f)
		expand := strings.NewReplacer("SRC", f.src, "RT", f.builtJar("rt"), "UX", f.builtJar("ux"), "DIR", f.dir)
		code := f.run(append([]string{"--no-backup"}, tc.args...)...)
		f.want(code, tc.code, expand.Replace(tc.want))
		if tc.call == "" && strings.Contains(f.calls(), "vm.sh") || !strings.Contains(f.calls(), expand.Replace(tc.call)) {
			t.Errorf("%s: calls:\n%s", tc.name, f.calls())
		}
		if code != ExitOK && (f.exists(f.deployedJar("rt")) || f.exists(f.deployedJar("ux")) || strings.Contains(f.out, "copy ok")) {
			t.Errorf("%s: a jar reached the station after exit %d", tc.name, code)
		}
	}
}
