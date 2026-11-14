package ngdeploy

import "testing"

func TestEachRequiredVariableIsNamedWhenMissing(t *testing.T) {
	for _, key := range []string{"MODULE_NAME", "GRADLEW_PATH", "NIAGARA_HOME", "NIAGARA_USER_HOME", "JAVA_HOME", "STATION_MODULES_DIR"} {
		for _, value := range []string{"-", ""} { // absent, and present but empty
			f := newFixture(t)
			f.set(key, value)
			code := f.run()
			if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR "+key+" is required but not set\n" || f.calls() != "" {
				t.Errorf("%s (%q): exit %d, stdout %q, stderr %q", key, value, code, f.out, f.err)
			}
		}
	}
}

func TestTheExpectedTypeCountsAreRequiredByMode(t *testing.T) {
	for _, tc := range []struct {
		key, mode string
		required  bool
	}{
		{"EXPECTED_RT_TYPES", "A", true}, {"EXPECTED_RT_TYPES", "C", true}, {"EXPECTED_RT_TYPES", "B", false},
		{"EXPECTED_UX_TYPES", "A", true}, {"EXPECTED_UX_TYPES", "B", true}, {"EXPECTED_UX_TYPES", "C", false},
	} {
		f := newFixture(t)
		f.set(tc.key, "-")
		code := f.run("--no-backup", "--no-deploy", "--mode", tc.mode)
		if tc.required && (code != ExitEnv || f.err != "[ng-deploy] ERROR "+tc.key+" is required for mode "+tc.mode+"\n") {
			t.Errorf("%s mode %s: exit %d, stderr %q", tc.key, tc.mode, code, f.err)
		}
		if !tc.required && code != ExitOK {
			t.Errorf("%s mode %s: exit %d, stderr %q", tc.key, tc.mode, code, f.err)
		}
	}
}

// Declared difference: under bash "09" made the count test itself fail, which passed any jar (exit 0), and
// a word stopped the deploy after the build with exit 1.
func TestAnExpectedTypeCountMustBeAWholeNumber(t *testing.T) {
	for _, value := range []string{"09", "nine", "+9", " 9", "9.0", "-1"} {
		f := newFixture(t)
		f.set("EXPECTED_RT_TYPES", "'"+value+"'")
		code := f.run("--no-backup", "--mode", "C")
		if code != ExitEnv || f.err != "[ng-deploy] ERROR EXPECTED_RT_TYPES must be a whole number\n" || f.calls() != "" {
			t.Errorf("%q: exit %d, stderr %q, calls %q", value, code, f.err, f.calls())
		}
	}
	f := newFixture(t)
	f.set("EXPECTED_UX_TYPES", "nine") // not read in mode C
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C"), ExitOK)
}

func TestStationModulesDirMustBeAnExistingWSLPath(t *testing.T) {
	f := newFixture(t)
	for value, want := range map[string]string{
		`'C:\Niagara\modules'`: "STATION_MODULES_DIR must use WSL /mnt/c/... paths, not Windows-style backslashes",
		f.dir + "/absent":      "STATION_MODULES_DIR path not found: " + f.dir + "/absent",
		f.env:                  "STATION_MODULES_DIR path not found: " + f.env,
	} {
		f.set("STATION_MODULES_DIR", value)
		if code := f.run(); code != ExitEnv || f.err != "[ng-deploy] ERROR "+want+"\n" || f.calls() != "" {
			t.Errorf("%s: exit %d, stderr %q", value, code, f.err)
		}
	}
}
