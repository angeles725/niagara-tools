package reportmodule

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

// codeTimeout is the exit code runLimited reports for a run it killed at its limit (timeout(1)'s own).
const codeTimeout = 124

// defaultJSTestsLimit is the seconds a `node --test` run may take; KIT_JS_TESTS_TIMEOUT overrides it.
const defaultJSTestsLimit = 600

// jsTestsLimit is the run limit in seconds named by v (a positive integer), else the default.
func jsTestsLimit(v string) int {
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return defaultJSTestsLimit
}

// runLimited is run with a time limit: a run still going after limit seconds is killed (the node test runner's
// own children are given a moment to release the output pipe) and reports codeTimeout. Like exec, a program
// still open for writing (ETXTBSY) is retried before it counts as 126.
func (x runner) runLimited(dir string, limit int, path string, args ...string) (out string, code int, ok bool) {
	w := &capWriter{max: x.bound()}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(limit)*time.Second)
	defer cancel()
	var err error
	for try := 0; ; try++ {
		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Dir, cmd.Stdout, cmd.Stderr, cmd.Env = dir, w, w, x.env
		cmd.WaitDelay = 2 * time.Second
		if err = cmd.Run(); !textBusy(err) || try == busyTries {
			break
		}
		time.Sleep(busyWait)
	}
	code = exitCode(err)
	if ctx.Err() != nil {
		code = codeTimeout
	}
	return trimOutput(w.b.Bytes()), code, w.dropped == 0
}
