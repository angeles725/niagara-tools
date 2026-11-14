package prepush

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var sweepCall = regexp.MustCompile(`(?m)^sweep (\S+)/BUILD-STATE\.md (\S+) (\S+)/INDEX\.md$`)

func TestSweepReadsFreshGeneratedStateFromATemporaryDirectory(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: retro", retro, pending)
	res := r.push(r.update(sha))
	res.want(t, 0)
	kit := r.dir + "/build-n4-module-kit"
	calls := r.stubCalls()
	m := sweepCall.FindStringSubmatch(calls)
	if m == nil || m[1] != m[3] || m[2] != kit+"/retros" {
		t.Fatalf("sweep call not <tmp>/BUILD-STATE.md %s/retros <tmp>/INDEX.md:\n%s", kit, calls)
	}
	want := "kit-state build-state " + kit + "\nkit-state retro-index " + kit + "\nkit-state lint-index " + kit + "\n" + m[0] + "\n"
	if calls != want {
		t.Errorf("stub calls\n%s\nwant\n%s", calls, want)
	}
	if _, err := os.Stat(m[1]); !os.IsNotExist(err) {
		t.Errorf("the temporary directory %s is left behind (%v)", m[1], err)
	}
	if strings.HasPrefix(m[1], r.dir) {
		t.Errorf("generated state written inside the work tree: %s", m[1])
	}
	// sweep's stdout is the hook's; the generators' stdout goes to the files and to nowhere (lint-index)
	if res.out != "changelog-gate: not applicable (no go/, build-n4-module-kit/, tools/ or scripts/ behaviour change in range)\nsweep stdout line\n" {
		t.Errorf("stdout %q", res.out)
	}
}

func TestGeneratorViolationFailsTheSweepBeforeTheNextStep(t *testing.T) {
	for step, later := range map[string]string{"build-state": "retro-index", "retro-index": "lint-index", "lint-index": "sweep "} {
		r := newRepo(t)
		t.Setenv("KIT_STATE_FAIL_ON", step)
		sha := r.change("docs: retro", retro, pending)
		r.push(r.update(sha)).want(t, 1, "kit-state: "+step+" violation", "ledger data in range is incoherent")
		if strings.Contains(r.stubCalls(), later) {
			t.Errorf("%s failed but %q still ran:\n%s", step, later, r.stubCalls())
		}
	}
}

func TestMissingKitWrapperFailsTheSweep(t *testing.T) {
	r := newRepo(t)
	r.run("rm", "-q", "build-n4-module-kit/toolbelt/kit-state.sh")
	sha := r.change("chore: drop the wrapper"+trivial+noChangelog, retro, pending)
	res := r.push(r.update(sha))
	if res.code == 0 {
		t.Errorf("a missing kit-state.sh passed the push\nstderr: %s", res.err)
	}
	res.want(t, res.code, "ledger data in range is incoherent")
}

func TestSweepThatCannotMakeItsTemporaryDirectoryRefuses(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: retro", retro, pending)
	t.Setenv("TMPDIR", r.dir+"/no-such-directory")
	res := r.push(r.update(sha))
	res.want(t, 1, "pre-push: mktemp failed; refusing to push unchecked")
	if strings.Contains(res.err, "incoherent") || r.stubCalls() != "" {
		t.Errorf("a sweep that never ran is reported as incoherent ledger data: %s", res.err)
	}
}
