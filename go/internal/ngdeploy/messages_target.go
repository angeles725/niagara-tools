package ngdeploy

// --target: the record of what a station runs (deploytargets.go).
const (
	msgTargetName = "--target %q is not a target name (letters, digits, '.', '_' and '-', starting with a letter or a digit)"
	// msgTargetNotRecorded takes the target, the file and one of the reasons below.
	msgTargetNotRecorded = "jars copied, but the deploy was not recorded for target %s in %s: %s (build.sh --target cannot use it)"
	whyTargetNoCommit    = "the commit is unknown: %s"
	whyTargetNoVersion   = "the vendorVersion of %s cannot be read, or holds a blank or a #"
	// The closing line, "record: ..." (nextsteps.go): what was written (the line, the file, the target), that
	// nothing was because no target was named, or that the target's line stays (the target, the mode).
	noteTargetWritten = "%s written to %s (build.sh --target %s reads it)"
	noteNoTarget      = "not written to " + deployTargetsFile + "; to record this deploy for build.sh --target, deploy with --target <name> (a name of your own, one per station)"
	noteTargetKept    = "%s not updated in " + deployTargetsFile + ": mode %s deploys no -rt jar, so the -rt commit it names stays what the station runs"
)
