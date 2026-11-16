package ngdeploy

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// gate runs the verify-module gate on the built jars of a mode with Java, unless --no-gate, with the module
// sources and the signer the deploy expects (expectedSigner). A gate that is missing or not executable
// blocks the deploy like one that fails (exit 50): no gate is not a pass.
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
	args := []string{"--src", d.moduleDir}
	signer, warning := d.expectedSigner()
	if signer != "" {
		args = append(args, "--expected-signer", signer)
	}
	d.say(msgGate, strings.Join(names, " "))
	status := d.show(tool, append(args, jars...)...)
	if warning != "" {
		d.warn("%s", warning) // after the gate's rows, where the operator reads the verdict
	}
	if status != 0 {
		return d.fail(ExitVerify, msgGateFailed)
	}
	d.say(msgGateOK)
	return proceed
}

// expectedSigner is the signer the gate must find on every jar (verify-module.sh --expected-signer), as
// build.sh resolves it (modjar.ExpectedSigner): the niagaraSigningAlias gradle property of the gradle root
// that holds gradlew, else the one signer of the module's jars that were in <NIAGARA_HOME>/modules before
// the build (home.go moved them aside), never the development signer. When the deploy knows none, signer is
// "" and warning is the line that says why: the gate then accepts any signature.
func (d *deploy) expectedSigner() (signer, warning string) {
	alias := modjar.ConfiguredAlias(d.gradleProps, d.env, dirname(d.env("GRADLEW_PATH")))
	var deployed [][]string
	for _, jar := range d.homeJars {
		if jar.moved && d.inMode(jar.name) {
			deployed = append(deployed, modjar.Signers(jar.aside))
		}
	}
	switch signer, why := modjar.ExpectedSigner(alias, deployed); why {
	case modjar.NoSigner:
		return "", warnNoSigner
	case modjar.DevDeployed:
		return "", fmt.Sprintf(warnDevSigner, modjar.DevSigner)
	default:
		return signer, ""
	}
}

// inMode reports whether a jar name is the jar of a profile of the mode.
func (d *deploy) inMode(name string) bool {
	for _, profile := range profilesOf[d.raw[optMode]] {
		if d.jarName(profile) == name {
			return true
		}
	}
	return false
}
