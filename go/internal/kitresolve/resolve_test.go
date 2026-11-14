package kitresolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const (
	offlineWarn = "kit-resolve: WARN offline: could not fetch origin; freshness unchecked\n"
	usageLine   = "usage: kit-resolve.sh [--refresh] [--offline-ok]\n"
)

// TestEnvKitWinsOverKitMain: a current, clean $BUILD_N4_KIT is the kit; stdout is the one KIT line.
func TestEnvKitWinsOverKitMain(t *testing.T) {
	f := newFixture(t)
	f.clone(f.dev)
	f.clone(f.kitMain)
	f.env["BUILD_N4_KIT"] = filepath.Join(f.dev, kitDirName)
	code, out, errText := f.resolve()
	expect(t, "env", code, out, errText, 0, f.okLine(f.dev, "origin/main"), "")
}

// TestCandidateOrder: kit-main before the default checkout, $BUILD_N4_KIT_MAIN moves kit-main, and the
// default checkout serves when it is the only one.
func TestCandidateOrder(t *testing.T) {
	f := newFixture(t)
	f.clone(f.defaultTop)
	code, out, errText := f.resolve()
	expect(t, "default alone", code, out, errText, 0, f.okLine(f.defaultTop, "origin/main"), "")
	f.clone(f.kitMain)
	code, out, errText = f.resolve()
	expect(t, "kit-main first", code, out, errText, 0, f.okLine(f.kitMain, "origin/main"), "")
	elsewhere := filepath.Join(f.base, "elsewhere")
	f.clone(elsewhere)
	f.env["BUILD_N4_KIT_MAIN"] = elsewhere
	code, out, errText = f.resolve()
	expect(t, "BUILD_N4_KIT_MAIN", code, out, errText, 0, f.okLine(elsewhere, "origin/main"), "")
}

// TestNoCandidate: exit 3 naming every path tried; a $BUILD_N4_KIT without METHODOLOGY.md is exit 3 and
// never falls through to kit-main.
func TestNoCandidate(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.resolve()
	expect(t, "none", code, out, errText, 3, "", "kit-resolve: no kit found: tried $BUILD_N4_KIT (unset), "+
		filepath.Join(f.kitMain, kitDirName)+", "+filepath.Join(f.defaultTop, kitDirName)+"\n")
	f.clone(f.kitMain)
	f.env["BUILD_N4_KIT"] = f.base
	code, out, errText = f.resolve()
	expect(t, "env without the marker", code, out, errText, 3, "",
		"kit-resolve: BUILD_N4_KIT ("+f.base+") has no METHODOLOGY.md; fix or unset it\n")
}

// TestCleanKitMainBehindIsFastForwarded: a clean kit-main strictly behind a fetched origin/main is
// checked out at it, with or without --refresh.
func TestCleanKitMainBehindIsFastForwarded(t *testing.T) {
	for _, args := range [][]string{nil, {"--refresh"}} {
		f := newFixture(t)
		f.clone(f.kitMain)
		before := f.short7(f.kitMain, "HEAD")
		f.advance()
		code, out, errText := f.resolve(args...)
		after := f.short7(f.kitMain, "HEAD")
		expect(t, "refresh", code, out, errText, 0, f.okLine(f.kitMain, "origin/main"), "kit-resolve: refreshed "+before+"→"+after+"\n")
		if f.head(f.kitMain) != git(t, f.origin, "rev-parse", "main") {
			t.Errorf("%v: kit-main HEAD is not origin/main after the refresh", args)
		}
	}
}

// TestDirtyKitIsRefused: uncommitted changes are exit 2 with the path count, and a dirty kit-main
// behind origin/main is never checked out.
func TestDirtyKitIsRefused(t *testing.T) {
	f := newFixture(t)
	f.clone(f.kitMain)
	f.dirty(f.kitMain)
	want := "kit-resolve: REFUSED dirty: " + f.kitMain + " has uncommitted changes (1 path(s)); clean it first — " + remedyKitMain(f.kitMain) + "\n"
	code, out, errText := f.resolve()
	expect(t, "dirty, current", code, out, errText, 2, "", want)
	f.advance()
	before := f.head(f.kitMain)
	for _, args := range [][]string{nil, {"--refresh"}} {
		code, out, errText = f.resolve(args...)
		expect(t, "dirty, behind", code, out, errText, 2, "", want)
	}
	if f.head(f.kitMain) != before {
		t.Error("a dirty kit-main was checked out")
	}
}

// TestStaleEnvAndDefaultAreRefusedNeverRefreshed: only kit-main is fast-forwarded; --refresh on another
// candidate says so and changes nothing.
func TestStaleEnvAndDefaultAreRefusedNeverRefreshed(t *testing.T) {
	f := newFixture(t)
	f.clone(f.defaultTop)
	f.clone(f.dev)
	f.advance()
	defaultHead, devHead := f.head(f.defaultTop), f.head(f.dev)
	code, out, errText := f.resolve()
	expect(t, "default", code, out, errText, 2, "",
		"kit-resolve: REFUSED stale: "+f.defaultTop+" is 1 commit(s) behind origin/main — "+f.remedyDefault(f.defaultTop)+"\n")
	code, out, errText = f.resolve("--refresh")
	expect(t, "default --refresh", code, out, errText, 2, "",
		"kit-resolve: --refresh applies only to the kit-main worktree; not refreshing "+f.defaultTop+"\n"+
			"kit-resolve: REFUSED stale: "+f.defaultTop+" is 1 commit(s) behind origin/main — "+f.remedyDefault(f.defaultTop)+"\n")
	f.env["BUILD_N4_KIT"] = filepath.Join(f.dev, kitDirName)
	code, out, errText = f.resolve()
	expect(t, "env", code, out, errText, 2, "",
		"kit-resolve: REFUSED stale: "+f.dev+" is 1 commit(s) behind origin/main — "+remedyEnv(f.dev)+"\n")
	if f.head(f.defaultTop) != defaultHead || f.head(f.dev) != devHead {
		t.Error("a candidate other than kit-main was checked out")
	}
}

// TestAheadAndDivergedAreRefused: a HEAD that is not an ancestor of origin/main is never the kit.
func TestAheadAndDivergedAreRefused(t *testing.T) {
	f := newFixture(t)
	f.clone(f.kitMain)
	f.localCommit(f.kitMain)
	head, main := f.short7(f.kitMain, "HEAD"), f.short7(f.kitMain, "origin/main")
	code, out, errText := f.resolve()
	expect(t, "ahead", code, out, errText, 2, "", "kit-resolve: REFUSED ahead: "+f.kitMain+" HEAD "+head+
		" has commits not on origin/main ("+main+") — "+remedyKitMain(f.kitMain)+"\n")
	f.advance()
	before := f.head(f.kitMain)
	code, out, errText = f.resolve()
	main = f.short7(f.kitMain, "origin/main")
	expect(t, "diverged", code, out, errText, 2, "", "kit-resolve: REFUSED diverged: "+f.kitMain+" HEAD "+head+
		" has commits not on origin/main ("+main+") and is 1 commit(s) behind — "+remedyKitMain(f.kitMain)+"\n")
	if f.head(f.kitMain) != before {
		t.Error("a diverged kit-main was checked out")
	}
}

// TestUsageAndHelp: an unknown argument is the usage line and exit 3; --help prints the launcher header.
func TestUsageAndHelp(t *testing.T) {
	f := newFixture(t)
	for _, args := range [][]string{{"--bogus"}, {"--refresh", "extra"}} {
		code, out, errText := f.resolve(args...)
		expect(t, args[0], code, out, errText, 3, "", usageLine)
	}
	script := filepath.Join(f.base, "scripts", "kit-resolve.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\n# kit-resolve.sh — fixture header.\nset -u\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.env[installtools.LauncherEnv] = script
	for _, flag := range []string{"-h", "--help"} {
		code, out, errText := f.resolve("--offline-ok", flag, "--bogus")
		expect(t, flag, code, out, errText, 0, "kit-resolve.sh — fixture header.\n", "")
	}
}
