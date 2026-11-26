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

// niagara_target of the module's record (niagara_target.go).
const (
	// msgRecordedTarget takes the module, the recorded version, the file, the installation, its version and the
	// major.minor the record names.
	msgRecordedTarget = "niagara_target precheck: %[1]s is recorded for Niagara %[2]s (niagara_target in %[3]s) but the target " +
		"installation %[4]s is Niagara %[5]s; nothing was built or copied: deploy into an installation of %[6]s, or, " +
		"if the station moved, update niagara_target there"
	// msgRecordedTargetOK takes the recorded version, the module, the file, the installation and its version.
	msgRecordedTargetOK = "niagara_target ok (%[1]s of %[2]s in %[3]s; %[4]s is Niagara %[5]s)"
	// warnRecordedTargetUnread takes the recorded version, the file, the station directory and the reason.
	warnRecordedTargetUnread = "WARN niagara_target %[1]s (%[2]s) is not compared with %[3]s: %[4]v"
)
