package buildmodule

import (
	"os"
)

// gate runs verify-module.sh on every jar the build produced, with the module sources and the signer the
// build expects (a WARN when it knows none: the gate then accepts any signature), then the report-module
// punch-list unless --no-report. A failing gate restores the pre-build modules/ jars the drift gate snapshotted. A jar that has not passed the gate does not go to a station.
func (b *build) gate() int {
	b.say(msgGate)
	source := b.root + "/" + b.opt.mod
	args := []string{"--src", source}
	if b.opt.target != "" {
		args = append(args, "--target-version", b.opt.target)
	}
	signer, warning := b.expectedSigner()
	if signer != "" {
		args = append(args, "--expected-signer", signer)
	}
	for _, p := range b.profiles {
		args = append(args, b.builtJar(p))
	}
	code := b.runScript(gateScript, args...)
	if warning != "" {
		b.warn("%s", warning) // after the gate's rows, where the operator reads the verdict
	}
	if code != 0 {
		b.rollBack()
		b.warn(msgGateFailed)
		return ExitGate
	}
	// The gate passed: modules/ now holds a verified jar, which an interrupt during report-module must not
	// undo (the gate returns an exit code, not proceed, so Main's own reset never runs).
	b.unverified.Store(false)
	b.recordMirrorBuild()
	if b.opt.skipReport {
		return ExitOK
	}
	return b.report(source)
}

// report runs report-module.sh over the module sources and maps its verdict to the chain's exit code.
func (b *build) report(source string) int {
	b.say(msgReport)
	args := []string{source}
	if b.opt.target != "" {
		args = append(args, "--target-version", b.opt.target)
	}
	if b.opt.uiProfile != "" {
		args = append(args, "--profile", b.opt.uiProfile)
	}
	if b.opt.legacy {
		args = append(args, "--legacy")
	}
	if b.opt.firstDeploy {
		args = append(args, "--first-deploy")
	}
	if b.home != "" {
		// the installation the template lint reads its stock modules from (retro K8)
		args = append(args, "--niagara-home", b.home)
	}
	switch b.runScript(reportScript, args...) {
	case 0:
		return ExitOK
	case reportEnvFault:
		b.warn(msgReportEnv)
		return ExitEnv
	case reportIncomplete:
		if !b.opt.firstDeploy && baselineMissing(source) {
			b.warn(msgReportIncomplete + msgSchemaRiskCrossRef)
		} else {
			b.warn(msgReportIncomplete)
		}
		return ExitReportPartial
	}
	// A FAILed punch-list is a jar that does not go to a station either: put the pre-build jar back as a failed
	// gate does, or a fix at the same version would read as drift (exit 51). [ev: retro client-persist-writes P4]
	b.rollBack()
	b.warn(msgReportFails)
	return ExitGate
}

// rollBack undoes what gradle's :jar step put into modules/ for a jar that failed the gate or the punch-list:
// the pre-build jar goes back, as a drift FAIL does, or a bare re-run would take the failed jar as its baseline
// and its signer as expected; on a first deploy there is none to put back, so the new jar is taken out again.
// keepReplaced then finds modules/ unchanged and keeps nothing under .mirror-replaced/.
func (b *build) rollBack() {
	for _, p := range b.profiles {
		b.restoreBaseline(p, restoredPlain)
		b.removeFirstDeploy(p)
	}
}

// baselineMissing reports whether an artifact report-module walks (walkedArtifacts: a docs/ folder is none) has
// no .deploy-baseline directory: then its schema-risk row is a floor SKIP naming the command that creates it.
func baselineMissing(source string) bool {
	artifacts, _ := walkedArtifacts(source)
	for _, a := range artifacts {
		if st, err := os.Stat(a + "/.deploy-baseline"); err != nil || !st.IsDir() {
			return true
		}
	}
	return false
}
