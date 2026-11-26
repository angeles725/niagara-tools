package reportmodule

import (
	"strings"
	"testing"
)

// vendoredLedger is rc/vendor/VENDORED.md listing OrbitControls.js at the given three.js version.
func vendoredLedger(version string) string {
	return "| File | Library | Version | Source URL |\n|---|---|---|---|\n" +
		"| OrbitControls.js | three.js | " + version + " | https://example.org/three/OrbitControls.js |\n"
}

// vendoredRun runs the report with the real lint-vendored member over a -ux part whose rc dir holds files.
func vendoredRun(t *testing.T, files map[string]string) (int, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("runs toolbelt members as subprocesses")
	}
	files["A-ux/module-include.xml"] = "<x/>"
	return realMembers(t, moduleTree(t, files), "lint-vendored")
}

// TestVendoredDefectIsAWarnRow (#1145): a -ux part whose src/rc/vendor/ holds a file at a version the kit's
// known-defects table lists gets the lint's WARN row, relayed under the part, and the exit stays the one a
// fixed version gets (a WARN row changes only the WARN count); a fixed version gets no row.
func TestVendoredDefectIsAWarnRow(t *testing.T) {
	files := func(version string) map[string]string {
		return map[string]string{"A-ux/src/rc/vendor/OrbitControls.js": "x", "A-ux/src/rc/vendor/VENDORED.md": vendoredLedger(version)}
	}
	code, out := vendoredRun(t, files("r160"))
	fixedCode, fixed := vendoredRun(t, files("r161"))
	if !strings.Contains(out, "A-ux  WARN  lint-vendored  OrbitControls.js  three.js r160 (VENDORED.md:3) has a known defect") {
		t.Errorf("no WARN row for the defective vendored file:\n%s", out)
	}
	if strings.Contains(fixed, "lint-vendored") {
		t.Errorf("a fixed version gets no row:\n%s", fixed)
	}
	if code != fixedCode || !strings.Contains(out, " · 1 WARN · ") || !strings.Contains(fixed, " · 0 WARN · ") {
		t.Errorf("exit %d with the WARN row, %d without; WARN rows never change the exit:\n%s", code, fixedCode, out)
	}
}

// TestVendoredWithoutVendorDirIsSilent (#1145): an rc dir with no vendor/ (or no src/rc at all) gets no
// lint-vendored row of any status.
func TestVendoredWithoutVendorDirIsSilent(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"rc without vendor/": {"A-ux/src/rc/index.html": "x"},
		"no rc":              {"A-ux/src/a/A.java": "class A {}"},
		"a vendor file":      {"A-ux/src/rc/vendor": "x"},
		"an -rt part":        {"A-rt/src/rc/vendor/OrbitControls.js": "x", "A-rt/src/rc/vendor/VENDORED.md": vendoredLedger("r160")},
	} {
		if _, out := vendoredRun(t, files); strings.Contains(out, "lint-vendored") {
			t.Errorf("%s: a lint-vendored row:\n%s", name, out)
		}
	}
}
