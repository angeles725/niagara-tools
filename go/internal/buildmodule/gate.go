package buildmodule

import (
	"os"
	"strings"
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
		// gradle's :jar step already put the new jar into modules/: put the pre-build one back, as a drift
		// FAIL does, or a bare re-run would take the failed jar as its baseline and its signer as expected.
		// On a first deploy there is none to put back: the new jar is taken out again.
		for _, p := range b.profiles {
			b.restoreBaseline(p, restoredPlain)
			b.removeFirstDeploy(p)
		}
		b.warn(msgGateFailed)
		return ExitGate
	}
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
	b.warn(msgReportFails)
	return ExitGate
}

// baselineMissing reports whether an artifact report-module walks (a directory under source that is not a
// symlink and not a dot-directory) has no .deploy-baseline directory: then its schema-risk row is a floor SKIP
// naming the command that creates it.
func baselineMissing(source string) bool {
	entries, _ := os.ReadDir(source)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if st, err := os.Stat(source + "/" + e.Name() + "/.deploy-baseline"); err != nil || !st.IsDir() {
			return true
		}
	}
	return false
}
