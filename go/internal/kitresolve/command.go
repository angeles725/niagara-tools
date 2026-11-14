package kitresolve

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// Main implements `kit-resolve [--refresh] [--offline-ok]`: stdout is the one KIT line of a resolved
// kit, every other message goes to stderr.
func Main(args []string, s cli.Stdio) int {
	say := func(format string, a ...any) { fmt.Fprintf(s.Err, tool+": "+format+"\n", a...) }
	o, ok := parseArgs(args)
	switch {
	case !ok:
		fmt.Fprint(s.Err, "usage: kit-resolve.sh [--refresh] [--offline-ok]\n")
		return exitUsage
	case o.help:
		return help(s)
	}
	line, st := resolve(o, s, say)
	if st != nil {
		say("%s", st.message)
		return st.code
	}
	fmt.Fprint(s.Out, line)
	return exitOK
}

// help prints the header of the invoked launcher, the one home of the usage text.
func help(s cli.Stdio) int {
	launcher, err := installtools.LauncherFrom(s)
	if err == nil {
		var header string
		if header, err = launcher.Header(); err == nil {
			fmt.Fprint(s.Out, header)
			return exitOK
		}
	}
	fmt.Fprintf(s.Err, "%s: usage text unavailable: %v\n", tool, err)
	return exitUsage
}

// resolve applies the rules in the order of the script header and returns the KIT line, or why not.
func resolve(o options, s cli.Stdio, say func(string, ...any)) (string, *stop) {
	c, st := choose(s)
	if st != nil {
		return "", st
	}
	top, ok := topLevel(c.kit)
	if !ok {
		return "", refused("untracked: %s is not in a git work tree; its freshness cannot be checked", c.kit)
	}
	r := repo{top: top, errTo: s.Err}
	remedy := c.remedy(top)

	fresh := freshRef
	if !r.fetch() {
		say("WARN offline: could not fetch origin; freshness unchecked")
		if !o.offlineOK {
			return "", refused("offline: rerun with --offline-ok to accept the local %s", freshRef)
		}
		fresh = freshRef + " unfetched"
	}
	if !r.hasFreshRef() {
		return "", refused("no %s ref in %s", freshRef, top)
	}
	if o.refresh && c.source != sourceKitMain {
		say("--refresh applies only to the kit-main worktree; not refreshing %s", top)
	}
	state := func() (dirty, ahead, behind int, st *stop) {
		var err error
		if dirty, err = r.dirtyPaths(); err == nil {
			ahead, behind, err = r.aheadBehind()
		}
		if err != nil {
			st = refused("unreadable: %v; its freshness cannot be checked", err)
		}
		return dirty, ahead, behind, st
	}
	dirty, ahead, behind, st := state()
	if st != nil {
		return "", st
	}
	// auto-refresh: kit-main only, only after a successful fetch, only clean, only a strict fast-forward
	if c.source == sourceKitMain && fresh == freshRef && dirty == 0 && behind > 0 && ahead == 0 {
		old := r.short("HEAD", 7)
		if !r.checkoutFresh() {
			return "", refused("refresh failed in %s", top)
		}
		say("refreshed %s→%s", old, r.short("HEAD", 7))
		if dirty, ahead, behind, st = state(); st != nil {
			return "", st
		}
	}
	switch {
	case dirty > 0:
		return "", refused("dirty: %s has uncommitted changes (%d path(s)); clean it first — %s", top, dirty, remedy)
	case ahead > 0 && behind == 0:
		// any candidate: HEAD must be origin/main or behind it — unmerged commits are never served as the kit
		return "", refused("ahead: %s HEAD %s has commits not on %s (%s) — %s", top, r.short("HEAD", 7), freshRef, r.short(freshRef, 7), remedy)
	case ahead > 0:
		return "", refused("diverged: %s HEAD %s has commits not on %s (%s) and is %d commit(s) behind — %s",
			top, r.short("HEAD", 7), freshRef, r.short(freshRef, 7), behind, remedy)
	case behind > 0:
		return "", refused("stale: %s is %d commit(s) behind %s — %s", top, behind, freshRef, remedy)
	}
	sha := r.short("HEAD", 0)
	if sha == "" {
		return "", refused("unreadable: git rev-parse HEAD failed in %s; its freshness cannot be checked", top)
	}
	return fmt.Sprintf("KIT=%s SHA=%s (%s, clean)\n", c.kit, sha, fresh), nil
}
