package kitresolve

import (
	"fmt"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// candidate is the kit that won and where it came from.
type candidate struct {
	kit     string // absolute, symlinks resolved (`cd <kit> && pwd -P`)
	source  string
	kitMain string // the kit-main worktree location, named by the default checkout's remedy
}

// stop ends the command with a message and an exit code.
type stop struct {
	code    int
	message string
}

func refused(format string, a ...any) *stop {
	return &stop{exitRefused, "REFUSED " + fmt.Sprintf(format, a...)}
}

// choose returns the first candidate holding METHODOLOGY.md: $BUILD_N4_KIT (which must hold it when
// set), the kit-main worktree, the default checkout. A candidate that cannot be examined is a refusal,
// never skipped: the next one would be served in its place.
func choose(s cli.Stdio) (candidate, *stop) {
	home := s.Env("HOME")
	kitMain := s.Env(envKitMain)
	if kitMain == "" {
		kitMain = home + homeKitMain
	}
	defaultKit := home + homeDefaultKit
	c := candidate{kitMain: kitMain}
	holds := func(kit string) (bool, *stop) {
		marker := kit + "/" + kitMarker
		is, err := installtools.IsFile(marker)
		if err != nil {
			return false, refused("unreadable: cannot check %s: %s", marker, installtools.Reason(err))
		}
		return is, nil
	}
	if env := s.Env(envKit); env != "" {
		is, st := holds(env)
		switch {
		case st != nil:
			return c, st
		case !is:
			return c, &stop{exitUsage, fmt.Sprintf("%s (%s) has no %s; fix or unset it", envKit, env, kitMarker)}
		}
		c.kit, c.source = env, sourceEnv
	} else {
		for _, try := range []struct{ kit, source string }{{kitMain + "/" + kitDirName, sourceKitMain}, {defaultKit, sourceDefault}} {
			is, st := holds(try.kit)
			if st != nil {
				return c, st
			}
			if is {
				c.kit, c.source = try.kit, try.source
				break
			}
		}
		if c.kit == "" {
			return c, &stop{exitUsage, fmt.Sprintf("no kit found: tried $%s (unset), %s/%s, %s", envKit, kitMain, kitDirName, defaultKit)}
		}
	}
	physical, err := filepath.Abs(c.kit)
	if err == nil {
		physical, err = filepath.EvalSymlinks(physical)
	}
	if err != nil {
		return c, refused("unreadable: cannot resolve %s: %s", c.kit, installtools.Reason(err))
	}
	c.kit = physical
	return c, nil
}

// remedy is the way out printed with every refusal, by candidate.
func (c candidate) remedy(top string) string {
	switch c.source {
	case sourceKitMain:
		return "refresh: git -C " + top + " checkout --detach " + freshRef
	case sourceEnv:
		return "update " + top + " to " + freshRef + ", or unset " + envKit + " to use the kit-main worktree"
	}
	return "use the kit-main worktree: git -C " + top + " worktree add --detach " + c.kitMain + " " + freshRef
}
