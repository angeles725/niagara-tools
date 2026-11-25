package buildmodule

import (
	"path/filepath"
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// The mirror hint of a build that stops on its niagara_home (issue #1088): the agent then had to guess
// where a mirror is and often made a new one beside a matching one.
const (
	mirrorGlob       = "niagara-mirror-*"
	msgMirrorsFrom   = "  mirrors under %s (made from this installation first):"
	msgMirrors       = "  mirrors under %s:"
	msgMirrorRow     = "    %s  (from %s)"
	msgMirrorTry     = "  try: build.sh %s %s %s"
	msgMirrorNone    = "  no mirror under %s: make one with toolbelt/mirror-niagara-home.sh <installation> %s/niagara-mirror-<version>"
	mirrorUnknownSrc = "unknown source"
)

// mirrorHint lists the mirrors in $HOME/niagara-mirror-* (directories holding .niagara-mirror) after a
// stop on niagara_home; real is the real installation given, "" when none was. The mirrors made from
// real come first, then the rest by name, and a try line names the first. It never picks one: the
// build still stops. With no mirror, a missing niagara_home says how to make one (a real one's message
// already does).
func (b *build) mirrorHint(real string) {
	home := b.s.Env("HOME")
	if home == "" {
		return
	}
	if real != "" {
		if abs, err := filepath.Abs(real); err == nil {
			real = filepath.Clean(abs)
		}
	}
	type mirror struct{ dir, source string }
	var found []mirror
	dirs, _ := filepath.Glob(home + "/" + mirrorGlob)
	for _, d := range dirs {
		if modjar.IsMirror(d) {
			found = append(found, mirror{d, modjar.MirrorSource(d)})
		}
	}
	if len(found) == 0 {
		if real == "" {
			b.warn(msgMirrorNone, home, home)
		}
		return
	}
	same := func(m mirror) bool { return real != "" && m.source != "" && filepath.Clean(m.source) == real }
	sort.SliceStable(found, func(i, j int) bool {
		if same(found[i]) != same(found[j]) {
			return same(found[i])
		}
		return found[i].dir < found[j].dir
	})
	if real != "" {
		b.warn(msgMirrorsFrom, home)
	} else {
		b.warn(msgMirrors, home)
	}
	for _, m := range found {
		source := m.source
		if source == "" {
			source = mirrorUnknownSrc
		}
		b.warn(msgMirrorRow, m.dir, source)
	}
	b.warn(msgMirrorTry, b.opt.root, b.opt.mod, found[0].dir)
}
