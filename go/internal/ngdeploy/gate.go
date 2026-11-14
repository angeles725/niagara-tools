package ngdeploy

import "strings"

// gate runs the verify-module gate on the built jars of a mode with Java, unless --no-gate. A gate that
// is missing or not executable blocks the deploy like one that fails (exit 50): no gate is not a pass.
func (d *deploy) gate() int {
	if d.isOn(optNoGate) || !d.hasProfile("rt") {
		return proceed
	}
	tool, jars := d.raw[optVerifyBin], d.builtJars()
	if !isExecutable(tool) {
		return d.fail(ExitVerify, msgGateMissing, tool)
	}
	names := make([]string, len(jars))
	for i, jar := range jars {
		names[i] = jar[strings.LastIndex(jar, "/")+1:]
	}
	// The bash tool joined the names with its IFS, a newline: the line is kept as it printed it.
	d.say(msgGate, strings.Join(names, "\n"))
	if d.show(tool, append([]string{"--src", d.moduleDir}, jars...)...) != 0 {
		return d.fail(ExitVerify, msgGateFailed)
	}
	d.say(msgGateOK)
	return proceed
}
