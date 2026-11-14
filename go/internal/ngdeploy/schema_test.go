package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

func TestSchemaPrecheck(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture)
		args    []string
		code    int
		want    string
		ran     bool // schema-risk was run
	}{
		{"no baseline, no first deploy", func(f *fixture) { f.set("FIRST_DEPLOY", "-") }, nil, ExitNoBaseline,
			"err:[ng-deploy] ERROR schema-risk precheck needs SCHEMA_BASELINE_DIR (the deployed -rt source baseline), or --first-deploy for a module never deployed\n", false},
		{"mode C too", func(f *fixture) { f.set("FIRST_DEPLOY", "-") }, []string{"--mode", "C"}, ExitNoBaseline, "err:schema-risk precheck needs SCHEMA_BASELINE_DIR", false},
		{"mode B has no schema", func(f *fixture) { f.set("FIRST_DEPLOY", "-") }, []string{"--mode", "B"}, ExitOK, "[ng-deploy] copy ok\n", false},
		{"FIRST_DEPLOY=1 in the env file", func(*fixture) {}, nil, ExitOK, "[ng-deploy] schema-risk: N/A (--first-deploy: no deployed baseline)\n", false},
		{"--first-deploy", func(f *fixture) { f.set("FIRST_DEPLOY", "-") }, []string{"--first-deploy"}, ExitOK, "[ng-deploy] schema-risk: N/A (--first-deploy: no deployed baseline)\n", false},
		{"an exported FIRST_DEPLOY is ignored", func(f *fixture) { f.set("FIRST_DEPLOY", "-"); f.t.Setenv("FIRST_DEPLOY", "1") }, nil, ExitNoBaseline, "err:schema-risk precheck needs", false},
		{"the baseline is not a directory", func(f *fixture) { f.set("SCHEMA_BASELINE_DIR", f.env) }, nil, ExitNoBaseline, "err:[ng-deploy] ERROR SCHEMA_BASELINE_DIR is not a directory: ", false},
		{"schema-risk is not executable", func(f *fixture) { f.baseline(); f.chmod(f.dir+"/tools/sr.sh", 0o644) }, nil, ExitSchema,
			"err:[ng-deploy] ERROR schema-risk precheck unavailable (not executable at ", false},
		{"SAFE", func(f *fixture) { f.baseline() }, nil, ExitOK, "schema-risk says\n[ng-deploy] schema-risk ok (SAFE)\n", true},
		{"a baseline wins over first deploy", func(f *fixture) { f.set("SCHEMA_BASELINE_DIR", f.dir+"/baseline") }, []string{"--mode", "C"}, ExitOK, "[ng-deploy] schema-risk ok (SAFE)\n", true},
		{"a baseline from the environment", func(f *fixture) { f.set("FIRST_DEPLOY", "-"); f.t.Setenv("SCHEMA_BASELINE_DIR", f.dir+"/baseline") }, nil, ExitOK, "[ng-deploy] schema-risk ok (SAFE)\n", true},
		{"LOSSY", func(f *fixture) { f.baseline(); f.t.Setenv("FAKE_SR_EXIT", "1") }, nil, ExitSchema,
			"err:[ng-deploy] ERROR schema-risk verdict LOSSY: a saved value would be lost (use --accept-lossy after checking the live bog)\n", true},
		{"LOSSY accepted", func(f *fixture) { f.baseline(); f.t.Setenv("FAKE_SR_EXIT", "1") }, []string{"--accept-lossy"}, ExitOK, "err:[ng-deploy] WARN schema-risk LOSSY accepted (--accept-lossy)\n", true},
		{"OUTAGE is never accepted", func(f *fixture) { f.baseline(); f.t.Setenv("FAKE_SR_EXIT", "2") }, []string{"--accept-lossy"}, ExitSchema,
			"err:[ng-deploy] ERROR schema-risk verdict OUTAGE: the slot change would break saved data; deploy blocked\n", true},
		{"a schema-risk fault", func(f *fixture) { f.baseline(); f.t.Setenv("FAKE_SR_EXIT", "3") }, []string{"--accept-lossy"}, ExitSchema,
			"err:[ng-deploy] ERROR schema-risk precheck could not run (exit 3); deploy blocked\n", true},
	} {
		f := newFixture(t)
		tc.arrange(f)
		code := f.run(append([]string{"--no-backup"}, tc.args...)...)
		f.want(code, tc.code, tc.want)
		call := "sr.sh <" + f.dir + "/baseline> <" + f.src + "/demo/demo-rt>\n"
		if strings.Contains(f.calls(), call) != tc.ran {
			t.Errorf("%s: calls:\n%s", tc.name, f.calls())
		}
		if (code == ExitOK) != f.exists(f.deployedJar("rt")) && tc.name != "mode B has no schema" {
			t.Errorf("%s: exit %d but the rt jar deployed is %v", tc.name, code, f.exists(f.deployedJar("rt")))
		}
		os.Unsetenv("FIRST_DEPLOY")
		os.Unsetenv("SCHEMA_BASELINE_DIR")
	}
}
