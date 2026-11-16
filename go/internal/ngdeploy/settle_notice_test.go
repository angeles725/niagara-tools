package ngdeploy

import (
	"strings"
	"testing"
)

const homeForeignLeft = "[ng-deploy] NOTICE: the jars the build installed in {modules} were taken out again and the module jars moved aside for it were put back; " +
	"the jar named above was not there before the build and is not the jar this run built: it was left in place\n"

// The NOTICE after a deploy that stopped is built from what settleHome did: it says the directory holds the
// jars it held before the build only when every jar was put back or removed; a jar of another origin left in
// place gets a NOTICE that does not say so; a jar that could not be removed is the ERROR alone
// (retro 2026-11-15-coordination-stretch-deltas D7).
func TestTheHomeNoticeIsBuiltFromWhatSettleDid(t *testing.T) {
	gateFailed := "[ng-deploy] ERROR verify-module gate failed; jars not fit to deploy (use --no-gate to override)\n"
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture, modules string)
		tail    string
	}{
		{"all put back", func(f *fixture, modules string) {}, gateFailed + homeRestored},
		{"a jar of another origin left", func(f *fixture, modules string) {
			f.write(f.dir+"/do/gradlew", installing(f.dir+"/home", `printf 'not of this run' > "`+modules+`/demo-ux.jar"`, "rt"))
		}, gateFailed + "[ng-deploy] WARN demo-ux.jar in {modules} was not there before the build and is not the jar this run built (build/libs): it is left where it is\n" + homeForeignLeft},
		{"a removal that failed", func(f *fixture, modules string) { failRemove(f.t, "/home/modules/demo-ux.jar") },
			gateFailed + "[ng-deploy] ERROR {modules} does NOT hold the jars it held before the build: demo-rt.jar is the previous jar, " +
				"demo-ux.jar, which {modules} did not have, could not be removed (Device or resource busy): remove it before the next build or deploy, demo-wb.jar is the previous jar\n"},
	} {
		f := newFixture(t)
		modules := withHome(t, f, true)
		t.Setenv("FAKE_VM_EXIT", "1")
		tc.arrange(f, modules)
		code := f.run("--no-backup")
		if want := strings.ReplaceAll(tc.tail, "{modules}", modules); code != ExitVerify || !strings.HasSuffix(f.err, want) || strings.Count(f.err, "NOTICE") > 1 {
			t.Errorf("%s: exit %d, stderr:\n%swant the tail:\n%s", tc.name, code, f.err, want)
		}
	}
}
