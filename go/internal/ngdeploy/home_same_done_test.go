package ngdeploy

import (
	"strings"
	"testing"
)

// In the same-directory mode nothing is copied, so the closing line does not say "copied to": it says the
// build installed the jars there and the copy step verified them (#993 D11, seen live on a lab install).
func TestTheClosingLineOfTheSameDirectoryModeSaysNothingWasCopied(t *testing.T) {
	f := newFixture(t)
	modules := sameDir(t, f)
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	if code := f.run("--no-backup"); code != ExitOK {
		t.Fatalf("exit %d, stderr %q", code, f.err)
	}
	// the line names the station directory as the env spelled it (sameDir spells it through a link)
	line := lineWith(f.out, "[ng-deploy] done: ")
	if !strings.HasPrefix(line, "[ng-deploy] done: demo-rt.jar demo-ux.jar installed by the build in ") ||
		!strings.HasSuffix(line, " and verified there (nothing to copy)") || modules == "" {
		t.Errorf("done line %q", line)
	}
	if strings.Contains(f.out, "copied to") {
		t.Errorf("the same-directory run still says copied to:\n%s", f.out)
	}
}

// lineWith is the first line of out that starts with prefix, "" when none does.
func lineWith(out, prefix string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	return ""
}
