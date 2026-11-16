package prepush

import (
	"errors"
	"io/fs"
	"os"
)

// ciLints runs the pushed work tree's `scripts/ci-local.sh --changed --range <rng>`: the pinned
// shellcheck, the kit's bats lints and gofmt over the files the range changes, read from its end commit.
// #684 failed two CI jobs that this would have refused locally. (retro 2026-11-15 coordination-stretch C3)
// Its output goes to the hook's stderr; any non-zero status (a finding, a tool that cannot be
// provisioned, the host-budget gate) refuses the push with that status. A tree without the script
// (not a niagara-tools checkout) skips the step in one named line.
func (h hook) ciLints(rng string) int {
	script := h.toplevel + "/" + ciLocalScript
	if _, err := os.Stat(script); errors.Is(err, fs.ErrNotExist) {
		printf(h.s.Err, msgCILintsSkipped, ciLocalScript)
		return 0
	}
	rc := h.script(h.s.Err, script, "--changed", "--range", rng)
	if rc != 0 {
		printf(h.s.Err, msgCILintsFailed, ciLocalScript, rng, rc)
	}
	return rc
}
