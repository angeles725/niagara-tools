// Package pytarget names the CPython the mcp-n4 server targets and gives the differential tests
// that interpreter. The Go port reproduces that Python's exact output, and some of it changes
// between minors (JSON decode errors gained "Illegal trailing comma" in 3.13), so a differential
// against any other version would compare against the wrong text: it is skipped, never passed.
package pytarget

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Minor is the CPython the server targets: the python3 the MCP config launches (README:
// "command": "python3"), 3.15 on the reference host. CI installs it with actions/setup-python.
const Minor = "3.15"

// Env names an interpreter to use instead of python3 on PATH (CI sets it to the setup-python
// path, so the other steps keep the runner's own python3). An interpreter named here must be
// the target: a mismatch fails the test instead of skipping it.
const Env = "MCP_N4_PYTHON"

// RecordEnv set to 1 makes the differentials run the target and record its output as goldens
// (package golden); recording without the target fails instead of skipping.
const RecordEnv = "MCP_N4_RECORD"

// Python returns the target interpreter. Otherwise it skips t (fails it when Env or RecordEnv
// is set), printing the version found and the target.
func Python(t testing.TB) string {
	t.Helper()
	py := os.Getenv(Env)
	stop := t.Skipf
	if py != "" || os.Getenv(RecordEnv) == "1" {
		stop = t.Fatalf
	}
	if py == "" {
		var err error
		if py, err = exec.LookPath("python3"); err != nil {
			stop("Python differential not run: no python3 on PATH and %s unset; it targets CPython %s", Env, Minor)
		}
	}
	out, err := exec.Command(py, "-I", "-c", "import sys; print('%d.%d' % sys.version_info[:2])").Output()
	if err != nil {
		stop("Python differential not run: %s did not report its version (%v); it targets CPython %s", py, err, Minor)
	}
	if got := strings.TrimSpace(string(out)); got != Minor {
		stop("Python differential not run: %s is CPython %s, the target is %s (set %s to a %s interpreter)",
			py, got, Minor, Env, Minor)
	}
	return py
}
