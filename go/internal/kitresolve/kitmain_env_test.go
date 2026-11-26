package kitresolve

import "testing"

// TestEnvNamingKitMainIsFastForwarded (#1122): build.sh names its kit with BUILD_N4_KIT; when that is the
// kit of the kit-main worktree, a clean kit-main strictly behind origin/main is refreshed as when no
// BUILD_N4_KIT is set, instead of refused as stale.
func TestEnvNamingKitMainIsFastForwarded(t *testing.T) {
	f := newFixture(t)
	f.clone(f.kitMain)
	before := f.short7(f.kitMain, "HEAD")
	f.advance()
	f.env["BUILD_N4_KIT"] = f.kitMain + "/" + kitDirName
	code, out, errText := f.resolve("--offline-ok")
	after := f.short7(f.kitMain, "HEAD")
	expect(t, "env kit-main", code, out, errText, 0, f.okLine(f.kitMain, "origin/main"),
		"kit-resolve: refreshed "+before+"→"+after+"\n")
}

// TestEnvNamingAnotherCloneBehindStaysRefused: the refresh is for the kit-main worktree only; any other
// clone behind origin/main named by BUILD_N4_KIT is still refused as stale and left where it is.
func TestEnvNamingAnotherCloneBehindStaysRefused(t *testing.T) {
	f := newFixture(t)
	other := f.base + "/other"
	f.clone(other)
	before := f.head(other)
	f.advance()
	f.env["BUILD_N4_KIT"] = other + "/" + kitDirName
	code, _, errText := f.resolve("--offline-ok")
	if code == 0 || f.head(other) != before {
		t.Fatalf("exit %d, head moved %v: %s", code, f.head(other) != before, errText)
	}
}
