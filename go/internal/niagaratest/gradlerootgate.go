package niagaratest

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// checkGradleRoot is the gate at the end of cleanup: the root's status against the one taken before the copy,
// then the opt-in restore. It reports whether the root is unchanged.
func (h host) checkGradleRoot(ctx context.Context, p *plan, s cli.Stdio) bool {
	now, err := h.rootStatus(ctx, p.gradleRoot, p.rootTop)
	if err != nil {
		row(s, "FAIL", "gradle root unchanged: cannot re-read it: "+err.Error())
		return false
	}
	ch := p.root.changed(now)
	if len(ch) == 0 {
		row(s, "PASS", fmt.Sprintf("gradle root unchanged (%d tracked changes before, same after)", len(p.root)))
		return true
	}
	// The restore's rows follow the FAIL row, which first needs to know what the restore checked out.
	var after bytes.Buffer
	var restored []string
	if p.restoreRoot {
		restored = h.restoreGradleRoot(ctx, p, now, ch, cli.Stdio{Out: &after, Err: s.Err, Getenv: s.Getenv})
	}
	row(s, "FAIL", "gradle root changed by the build: "+listed(ch, maxRootListed)+trapHint(ch, restored, p.restoreRoot))
	_, _ = s.Out.Write(after.Bytes())
	return false
}

// trapHint names the known cause when the build changed a */module-include.xml: niagara gradle plugin 7.3.40
// and 7.6.22 (observed) append the test types to the production file in the moduleTest compile, then says
// whether --restore-gradle-root checked it out. Any other change keeps the generic row ("").
func trapHint(ch, restored []string, restoreRoot bool) string {
	isInclude := func(p string) bool { return strings.HasSuffix(p, "/module-include.xml") }
	if !slices.ContainsFunc(ch, isInclude) {
		return ""
	}
	restore := "rerun with --restore-gradle-root to check it out"
	switch {
	case slices.ContainsFunc(restored, isInclude):
		restore = "--restore-gradle-root restored the ones that were clean before (next row)"
	case restoreRoot:
		restore = "--restore-gradle-root did not restore it (next row)"
	}
	return " (module-include.xml: niagara gradle plugin 7.3.40 and 7.6.22 (observed) append the test types to " +
		"the PRODUCTION module-include.xml in :<part>:moduleTestJar or the moduleTest compile whenever the test " +
		"sources changed; " + restore + "; see build-n4-module-kit/types/moduleTest.md, plugin-version trap)"
}

// restoreGradleRoot checks out the changed paths that were clean before and are now modified or deleted in
// the worktree, nothing else, then re-reads the status to confirm them, and returns the confirmed paths.
// Untracked files the build created are listed, never deleted.
func (h host) restoreGradleRoot(ctx context.Context, p *plan, now rootSnapshot, ch []string, s cli.Stdio) []string {
	var restore, untracked []string
	for _, path := range ch {
		if _, dirty := p.root[path]; dirty {
			continue
		}
		switch st := now[path].status; {
		case st == "??":
			untracked = append(untracked, path)
		case len(st) == 2 && st[0] == ' ' && strings.ContainsRune("MDT", rune(st[1])):
			restore = append(restore, path)
		}
	}
	left := ""
	if len(untracked) > 0 {
		left = " (untracked files the build created, left in place: " + listed(untracked, maxRootListed) + ")"
	}
	if len(restore) == 0 {
		row(s, "PASS", "gradle root restored: nothing to check out"+left)
		return nil
	}
	// --literal-pathspecs: a path is a file name, never pathspec magic such as :(glob) or a wildcard.
	args := append([]string{"--literal-pathspecs", "-C", p.gradleRoot, "checkout", "--"}, restore...)
	if code, err := h.run.Run(ctx, Cmd{Name: "git", Args: args}); err != nil || code != 0 {
		row(s, "FAIL", fmt.Sprintf("gradle root not restored: git checkout exit %d (%v)", code, err))
		return nil
	}
	after, err := h.rootStatus(ctx, p.gradleRoot, p.rootTop)
	if err != nil {
		row(s, "FAIL", "gradle root not restored: cannot re-read it: "+err.Error())
		return nil
	}
	var still []string
	for _, path := range restore {
		if _, ok := after[path]; ok {
			still = append(still, path)
		}
	}
	if len(still) > 0 {
		row(s, "FAIL", "gradle root not restored: "+listed(still, maxRootListed)+" still differs after git checkout")
		return nil
	}
	row(s, "PASS", "gradle root restored: "+listed(restore, maxRootListed)+left)
	return restore
}
