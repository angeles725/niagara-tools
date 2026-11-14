package buildmodule

// gate runs verify-module.sh on every jar the build produced, with the module sources, then the
// report-module punch-list unless --no-report. A jar that has not passed the gate does not go to a station.
func (b *build) gate() int {
	b.say(msgGate)
	source := b.root + "/" + b.opt.mod
	args := []string{"--src", source}
	if b.opt.target != "" {
		args = append(args, "--target-version", b.opt.target)
	}
	for _, p := range b.profiles {
		args = append(args, b.builtJar(p))
	}
	if b.runScript(gateScript, args...) != 0 {
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
	switch b.runScript(reportScript, args...) {
	case 0:
		return ExitOK
	case reportEnvFault:
		b.warn(msgReportEnv)
		return ExitEnv
	case reportIncomplete:
		b.warn(msgReportIncomplete)
		return ExitReportPartial
	}
	b.warn(msgReportFails)
	return ExitGate
}
