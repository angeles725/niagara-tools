package prepush

import (
	"io"
	"os"
	"os/exec"
)

// runSweep is the content half: ledger + index coherence, one call shape for every exit.
// BUILD-STATE.md and the two INDEX.md files are generated and not committed, so they are generated
// from the committed sources (the lint index only validated) into a temporary directory and swept
// there; any generator violation fails it. It runs the pushed tree's own kit-state.sh and
// sweep-build-state.sh and returns the status of the first step that fails, 0 when all pass. ran is
// false when no step could start (the refusal is already printed): that is no verdict on the ledger data.
// [ev: retro ledger-generated-not-committed]
func (h hook) runSweep() (rc int, ran bool) {
	tmp, err := os.MkdirTemp("", "tmp.")
	if err != nil {
		printf(h.s.Err, msgCannotRun, "mktemp", err) // the cause, as mktemp(1) named it for the bash hook
		return refuse(h.s.Err, msgMktemp), false
	}
	defer os.RemoveAll(tmp)
	return h.sweepIn(tmp), true
}

// sweepIn generates the state into tmp and sweeps it.
func (h hook) sweepIn(tmp string) int {
	state, index := tmp+"/BUILD-STATE.md", tmp+"/INDEX.md"
	kitState := h.kit + kitStateTool

	if rc := h.generate(state, kitState, "build-state", h.kit); rc != 0 {
		return rc
	}
	if rc := h.generate(index, kitState, "retro-index", h.kit); rc != 0 {
		return rc
	}
	if rc := h.script(nil, kitState, "lint-index", h.kit); rc != 0 {
		return rc
	}
	return h.script(h.s.Out, h.kit+sweepTool, state, h.kit+"/retros", index)
}

// generate runs a kit script with its stdout written to the file at path.
func (h hook) generate(path, script string, args ...string) int {
	f, err := os.Create(path)
	if err != nil {
		printf(h.s.Err, msgCannotRun, script, err)
		return 1
	}
	rc := h.script(f, script, args...)
	if err := f.Close(); err != nil && rc == 0 {
		printf(h.s.Err, msgCannotRun, script, err)
		return 1
	}
	return rc
}

// script runs `bash <script> <args>` with its stdout on out (nil drops it) and its stderr on the
// hook's, and returns its status.
func (h hook) script(out io.Writer, script string, args ...string) int {
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Stdout, cmd.Stderr = out, h.s.Err
	return exitStatus(script, cmd.Run(), h.s.Err)
}
