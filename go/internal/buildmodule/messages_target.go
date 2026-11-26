package buildmodule

// --target: the deploy record (deploytarget.go).
const (
	msgTargetNoLine     = "build.sh: --target %s: no line for %s in %s (one per target and module: <target> <MOD>@<vendorVersion>=<commit>)"
	msgTargetUnreadable = "build.sh: --target %s: cannot read the line in %s: %s (want <target> <MOD>@<vendorVersion>=<commit>)"
	msgTargetExclusive  = "build.sh: --target and --schema-baseline-ref/--first-deploy exclude each other (the target records the deployed ref)"
	msgTargetRecord     = "==> deploy target %s: %s at %s (%s)"
	msgTargetVersion    = "build.sh: WARN — %[1]s records %[2]s@%[3]s for %[4]s but its commit declares defaultModuleVersion %[5]s"
	// msgRecordedElsewhere takes the file, the module, its recorded version, the target, the shipped version and the jar.
	msgRecordedElsewhere = "build.sh: INFO — %[1]s records %[2]s@%[3]s for %[4]s, not %[5]s: %[6]s.jar is not what that station runs, drift check skipped"
)
