package platread

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// noise is what plat prints before its answer: the INFO lines of the NRE boot log (INFO, localized as
// INFORMACIÓN, INFORMATION...), the registry's missing-class lines and the module permission blocks
// (Type:, Purpose:, Parameters:, Risk Level:). Any other log line, an error included, is kept.
var noise = regexp.MustCompile(`^(INFO\S* \[|INFO \[nre\] |\S+ \[[^]]*\]\[sys\.registry\] Missing class |` +
	"Type: [A-Z_]+$| (Purpose|Parameters|Risk Level): )")

func (c call) exe() string {
	for _, name := range []string{"plat.exe", "plat"} {
		if p := filepath.Join(c.home, "bin", name); fileExists(p) {
			return p
		}
	}
	return filepath.Join(c.home, "bin", "plat")
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func (c call) run(s cli.Stdio, cr cred) int {
	if c.dry {
		fmt.Fprintln(s.Out, "plat "+strings.Join(c.argv(cr, true), " "))
		return 0
	}
	out, err := exec.Command(c.exe(), c.argv(cr, false)...).CombinedOutput()
	text := strings.ReplaceAll(strings.ReplaceAll(string(out), "\r", ""), cr.password, masked)
	if answer := strings.Trim(Clean(text), "\n"); answer != "" { // no blank line for an empty answer
		fmt.Fprintln(s.Out, answer)
	}
	if why := failure(text); why != "" && err == nil {
		fmt.Fprintln(s.Err, "plat-read: "+why)
		return 1
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	if err != nil {
		fmt.Fprintln(s.Err, "plat-read: "+strings.ReplaceAll(err.Error(), cr.password, masked))
		return 1
	}
	return 0
}
