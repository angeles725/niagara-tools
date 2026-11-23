package preflight

import (
	"strings"
	"testing"
)

// Retro delivery-guards-and-ui-contrast Δ2: in a kit checkout the preflight says up front what the pre-push
// gate will ask of a kit change, so it is written with the change and not found by the refusal.
func TestDeliveryRowNamesWhatAKitChangeNeeds(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/", "repo/changelog.d/", "repo/build-n4-module-kit/ledger.d/", "repo/g/build.gradle.kts=x")
	git(t, d+"/repo", "init", "-q", "-b", "main")
	_, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/repo/g")
	want := d + "/repo is a kit checkout: a change under go/, build-n4-module-kit/, tools/ or scripts/ " +
		"is pushed with a changelog.d/<date>-<slug>.md fragment (or a Changelog: none (<reason>) trailer), " +
		"a build-n4-module-kit/ledger.d/<date>-<slug>.md fragment committed alone and a pending retro " +
		"(toolbelt/new-retro.sh; or a Retro: none (trivial: <reason>) trailer): CONTRIBUTING.md 6.1 and 6.2\n"
	if !strings.Contains(out, "PASS  delivery ") || !strings.Contains(out, want) {
		t.Errorf("rows\n%s\nwant a PASS delivery row carrying %q", out, want)
	}
}

// A module checkout, a directory outside git and a checkout with only one of the two directories get no row.
func TestDeliveryRowIsSilentOutsideAKitCheckout(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/", "repo/g/build.gradle.kts=x", "half/changelog.d/", "half/g/build.gradle.kts=x")
	if _, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/repo/g"); strings.Contains(out, "delivery") {
		t.Errorf("not a git work tree, rows\n%s", out)
	}
	git(t, d+"/repo", "init", "-q", "-b", "main")
	if _, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/repo/g"); strings.Contains(out, "delivery") {
		t.Errorf("a module checkout, rows\n%s", out)
	}
	git(t, d+"/half", "init", "-q", "-b", "main")
	if _, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/half/g"); strings.Contains(out, "delivery") {
		t.Errorf("a checkout with changelog.d/ only, rows\n%s", out)
	}
}
